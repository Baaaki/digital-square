package service

import (
	"errors"
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

	if err := s.messageRepo.CreateMessage(msg) ; err != nil {
		return nil, err
	}

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
