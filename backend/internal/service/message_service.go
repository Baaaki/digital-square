package service

import (
	"context"
	"errors"
	"html"
	"time"
	"unicode/utf8"

	"github.com/Baaaki/digital-square/internal/broker"
	"github.com/Baaaki/digital-square/internal/models"
	"github.com/Baaaki/digital-square/internal/repository"
	"github.com/Baaaki/digital-square/internal/wal"
	"github.com/Baaaki/digital-square/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrMessageNotFound = errors.New("message not found")
	ErrUnauthorized    = errors.New("unauthorized to delete this message")
	ErrMessageTooLong  = errors.New("message too long (max 5000 characters)")
	ErrMessageTooShort = errors.New("message cannot be empty")
)

type MessageService struct {
	messageRepo *repository.MessageRepository //for database
	broker      broker.MessageBroker          // for pub/sub
	wal         *wal.WAL                      // for wal, you know :D
}

func NewMessageService(
	messageRepo *repository.MessageRepository,
	broker broker.MessageBroker,
	wal *wal.WAL,
) *MessageService {
	return &MessageService{
		messageRepo: messageRepo,
		broker:      broker,
		wal:         wal,
	}
}

// validateMessageContent validates message content for security and length constraints
func (s *MessageService) validateMessageContent(content string) error {
	// 1. Empty message check
	if content == "" {
		return ErrMessageTooShort
	}

	// 2. Max length check (5000 characters, Unicode-aware)
	if utf8.RuneCountInString(content) > 5000 {
		return ErrMessageTooLong
	}

	// 3. Minimum length check (at least 1 character)
	if utf8.RuneCountInString(content) < 1 {
		return ErrMessageTooShort
	}

	return nil
}

func (s *MessageService) SendMessage(userID uuid.UUID, username, content string) (*models.Message, error) {
	start := time.Now()
	messageID := uuid.New().String()
	now := time.Now()

	// 1. VALIDATE INPUT (length, empty check)
	if err := s.validateMessageContent(content); err != nil {
		logger.Log.Warn("Message validation failed",
			zap.String("user_id", userID.String()),
			zap.Int("content_length", utf8.RuneCountInString(content)),
			zap.Error(err),
		)
		return nil, err
	}

	// 2. SANITIZE CONTENT (XSS Prevention)
	sanitizedContent := html.EscapeString(content)

	logger.Log.Debug("Processing message send",
		zap.String("user_id", userID.String()),
		zap.String("message_id", messageID),
		zap.Int("original_length", len(content)),
		zap.Int("sanitized_length", len(sanitizedContent)),
	)

	msg := &models.Message{
		MessageID: messageID,
		UserID:    userID,
		Username:  username, // ✅ Store username (denormalized for performance)
		Content:   sanitizedContent, // ✅ Sanitized content (not original)
		CreatedAt: now,
	}

	// 1. Write to WAL FIRST (sync - durability, crash recovery)
	walStart := time.Now()
	walEntry := wal.WALEntry{
		MessageID: msg.MessageID,
		UserID:    msg.UserID.String(),
		Content:   msg.Content,
		Timestamp: msg.CreatedAt,
	}
	if err := s.wal.Write(walEntry); err != nil {
		logger.Log.Error("Failed to write to WAL",
			zap.String("message_id", messageID),
			zap.Error(err),
		)
		return nil, err
	}
	walDuration := time.Since(walStart)

	logger.Log.Info("Message written to WAL",
		zap.String("message_id", messageID),
		zap.String("user_id", userID.String()),
		zap.Duration("wal_write_duration", walDuration),
		zap.Duration("total_duration", time.Since(start)),
	)

	// 2. Write to Redis cache asynchronously (for new connections)
	go func() {
		cacheStart := time.Now()
		if err := s.broker.CacheMessage(*msg); err != nil {
			logger.Log.Warn("Failed to cache message to Redis",
				zap.String("message_id", messageID),
				zap.Error(err),
			)
		} else {
			logger.Log.Debug("Message cached to Redis",
				zap.String("message_id", messageID),
				zap.Duration("cache_duration", time.Since(cacheStart)),
			)
		}
	}()

	// 3. WebSocket handler will broadcast to all connected clients (in-memory)
	//    PostgreSQL write will be handled by Batch Writer (every 1 minute)

	return msg, nil
}

func (s *MessageService) GetRecentMessages(limit int) ([]models.Message, error) {
	start := time.Now()

	// Try Redis cache first (updated in real-time)
	cachedMsgs, err := s.broker.GetRecentMessages(limit)
	if err == nil && len(cachedMsgs) > 0 {
		logger.Log.Debug("Cache HIT: Retrieved messages from Redis",
			zap.Int("message_count", len(cachedMsgs)),
			zap.Duration("duration", time.Since(start)),
		)
		return cachedMsgs, nil
	}

	// Cache miss - fallback to PostgreSQL
	logger.Log.Debug("Cache MISS: Fetching from PostgreSQL",
		zap.Int("limit", limit),
	)

	dbStart := time.Now()
	messages, err := s.messageRepo.GetRecentMessages(limit)
	if err != nil {
		logger.Log.Error("Failed to get recent messages from PostgreSQL",
			zap.Error(err),
		)
		return nil, err
	}

	logger.Log.Info("Retrieved messages from PostgreSQL",
		zap.Int("message_count", len(messages)),
		zap.Duration("db_duration", time.Since(dbStart)),
		zap.Duration("total_duration", time.Since(start)),
	)

	// Warm up Redis cache for next connection
	if len(messages) > 0 {
		go func() {
			warmupStart := time.Now()
			for _, msg := range messages {
				s.broker.CacheMessage(msg)
			}
			logger.Log.Info("Warmed up Redis cache",
				zap.Int("message_count", len(messages)),
				zap.Duration("warmup_duration", time.Since(warmupStart)),
			)
		}()
	}

	return messages, nil
}

func (s *MessageService) GetMessagesBefore(beforeID uint64, limit int, isAdmin bool) ([]models.Message, error) {
	return s.messageRepo.GetMessagesBefore(beforeID, limit)
}

func (s *MessageService) DeleteMessage(messageID string, userID uuid.UUID, isAdmin bool) error {
	start := time.Now()

	logger.Log.Debug("Processing message delete",
		zap.String("message_id", messageID),
		zap.String("user_id", userID.String()),
		zap.Bool("is_admin", isAdmin),
	)

	msg, err := s.messageRepo.GetByMessageID(messageID)
	if err != nil {
		logger.Log.Warn("Message not found for deletion",
			zap.String("message_id", messageID),
			zap.Error(err),
		)
		return ErrMessageNotFound
	}

	if !isAdmin && msg.UserID != userID {
		logger.Log.Warn("Unauthorized delete attempt",
			zap.String("message_id", messageID),
			zap.String("requesting_user_id", userID.String()),
			zap.String("message_owner_id", msg.UserID.String()),
		)
		return ErrUnauthorized
	}

	deletedBy := userID
	isDeletedByAdmin := isAdmin

	if err := s.messageRepo.SoftDeleteMessage(msg.ID, deletedBy, isDeletedByAdmin); err != nil {
		logger.Log.Error("Failed to soft delete message",
			zap.String("message_id", messageID),
			zap.Error(err),
		)
		return err
	}

	// Update Redis cache - mark message as deleted (soft delete in cache)
	// This allows admins to see deleted messages from cache
	if err := s.broker.MarkMessageAsDeleted(messageID, isDeletedByAdmin); err != nil {
		logger.Log.Warn("Failed to update Redis cache for deleted message",
			zap.String("message_id", messageID),
			zap.Error(err),
		)
		// Don't fail the delete operation if Redis update fails
		// PostgreSQL is source of truth
	}

	logger.Log.Info("Message deleted successfully",
		zap.String("message_id", messageID),
		zap.String("deleted_by", userID.String()),
		zap.Bool("is_admin", isAdmin),
		zap.Duration("duration", time.Since(start)),
	)

	return nil
}

// StartBatchWriter starts a background goroutine that writes messages from WAL to PostgreSQL
// Runs every 1 minute and writes ALL messages in WAL (no limit)
func (s *MessageService) StartBatchWriter(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		logger.Log.Info("Batch Writer started: Writing WAL to PostgreSQL every 1 minute")

		for {
			select {
			case <-ctx.Done():
				logger.Log.Info("Batch Writer stopped")
				return

			case <-ticker.C:
				logger.Log.Debug("Batch Writer tick - checking WAL")
				s.processBatch()
			}
		}
	}()
}

// processBatch reads ALL messages from WAL and writes to PostgreSQL
func (s *MessageService) processBatch() {
	start := time.Now()

	// 1. Get ALL entries from WAL
	entries, err := s.wal.GetAllEntries()
	if err != nil {
		logger.Log.Error("Batch Writer: Failed to read WAL",
			zap.Error(err),
		)
		return
	}

	// 2. If WAL is empty, skip (no unnecessary PostgreSQL calls)
	if len(entries) == 0 {
		// WAL is empty, nothing to do (no log needed - too noisy)
		return
	}

	logger.Log.Info("Batch Writer: Found messages in WAL",
		zap.Int("message_count", len(entries)),
	)

	// 3. Convert WAL entries to models.Message
	messages := make([]models.Message, 0, len(entries))
	messageIDs := make([]string, 0, len(entries))

	for _, entry := range entries {
		userID, _ := uuid.Parse(entry.UserID)
		messages = append(messages, models.Message{
			MessageID: entry.MessageID,
			UserID:    userID,
			Content:   entry.Content,
			CreatedAt: entry.Timestamp,
		})
		messageIDs = append(messageIDs, entry.MessageID)
	}

	// 4. Batch insert to PostgreSQL
	insertStart := time.Now()
	if err := s.messageRepo.BatchInsert(messages); err != nil {
		logger.Log.Error("Batch Writer: Failed to insert messages to PostgreSQL",
			zap.Int("message_count", len(messages)),
			zap.Error(err),
		)
		return
	}
	insertDuration := time.Since(insertStart)

	logger.Log.Info("Batch Writer: Messages written to PostgreSQL",
		zap.Int("message_count", len(messages)),
		zap.Duration("insert_duration", insertDuration),
	)

	// 5. Cleanup WAL (only after successful PostgreSQL write)
	cleanupStart := time.Now()
	if err := s.wal.Cleanup(messageIDs); err != nil {
		logger.Log.Error("Batch Writer: Failed to cleanup WAL",
			zap.Int("message_count", len(messageIDs)),
			zap.Error(err),
		)
		return
	}
	cleanupDuration := time.Since(cleanupStart)

	logger.Log.Info("Batch Writer: Batch processing completed",
		zap.Int("message_count", len(messages)),
		zap.Duration("cleanup_duration", cleanupDuration),
		zap.Duration("total_duration", time.Since(start)),
	)
}
