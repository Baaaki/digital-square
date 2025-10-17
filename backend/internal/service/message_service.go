package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/Baaaki/digital-square/internal/broker"
	"github.com/Baaaki/digital-square/internal/models"
	"github.com/Baaaki/digital-square/internal/repository"
	"github.com/Baaaki/digital-square/internal/wal"
	"github.com/google/uuid"
)

var(
	ErrMessageNotFound = errors.New("message not found")
	ErrUnauthorized = errors.New("unauthorized to delete this message")
)

type MessageService struct {
	messageRepo *repository.MessageRepository //for database
	broker broker.MessageBroker // for pub/sub
	wal *wal.WAL // for wal, you know :D
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

func (s *MessageService) SendMessage(userID uuid.UUID, username, content string) (*models.Message, error) {
	messageID := uuid.New().String()
	now := time.Now()

	msg := &models.Message{
		MessageID: messageID,
		UserID: userID,
		Content: content,
		CreatedAt: now,
	}

	walEntry := wal.WALEntry{
		MessageID: messageID,
		UserID: userID.String(),
		Content: content,
		Timestamp: now,
	}
	if err:= s.wal.Write(walEntry); err != nil {
		return nil, err
	}

	brokerMsg := broker.Message{
		MessageID: messageID,
		UserID: userID.String(),
		Username: username,
		Content: content,
		Timestamp: now.Format(time.RFC3339),
	}
	if err := s.broker.Publish(brokerMsg) ; err != nil {
		return nil, err
	}

	// PostgreSQL write is now handled by batch writer (async)
	// Messages are persisted to PostgreSQL every 1 minute from WAL

	return msg, nil
}

func (s *MessageService) GetRecentMessages(limit int) ([]models.Message, error) {
	return s.messageRepo.GetRecentMessages(limit)
}

func (s *MessageService) GetMessagesBefore (beforeID uint64, limit int, isAdmin bool) ([]models.Message, error) {
	return s.messageRepo.GetMessagesBefore(beforeID, limit)
}

func (s *MessageService) DeleteMessage (messageID string, userID uuid.UUID, isAdmin bool) error {
	msg, err := s.messageRepo.GetByMessageID(messageID)
	if err != nil {
		return ErrMessageNotFound
	}
	if !isAdmin && msg.UserID != userID {
		return ErrUnauthorized
	}

	deletedBy := userID
	isDeletedByAdmin := isAdmin

	if err := s.messageRepo.SoftDeleteMessage(msg.ID, deletedBy, isDeletedByAdmin); err != nil {
		return err
	}

	return nil
}

// StartBatchWriter starts a background goroutine that writes messages from WAL to PostgreSQL
// Runs every 1 minute and writes ALL messages in WAL (no limit)
func (s *MessageService) StartBatchWriter(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		log.Println("🚀 Batch Writer started: Writing WAL → PostgreSQL every 1 minute")

		for {
			select {
			case <-ctx.Done():
				log.Println("⏹️  Batch Writer stopped")
				return

			case <-ticker.C:
				s.processBatch()
			}
		}
	}()
}

// processBatch reads ALL messages from WAL and writes to PostgreSQL
func (s *MessageService) processBatch() {
	// 1. Get ALL entries from WAL
	entries, err := s.wal.GetAllEntries()
	if err != nil {
		log.Printf("❌ Batch Writer: Failed to read WAL: %v", err)
		return
	}

	// 2. If WAL is empty, skip (no unnecessary PostgreSQL calls)
	if len(entries) == 0 {
		// WAL is empty, nothing to do
		return
	}

	log.Printf("📦 Batch Writer: Found %d messages in WAL", len(entries))

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
	if err := s.messageRepo.BatchInsert(messages); err != nil {
		log.Printf("❌ Batch Writer: Failed to insert messages: %v", err)
		return
	}

	log.Printf("✅ Batch Writer: %d messages written to PostgreSQL", len(messages))

	// 5. Cleanup WAL (only after successful PostgreSQL write)
	if err := s.wal.Cleanup(messageIDs); err != nil {
		log.Printf("❌ Batch Writer: Failed to cleanup WAL: %v", err)
		return
	}

	log.Printf("🧹 Batch Writer: WAL cleaned up successfully")
}
