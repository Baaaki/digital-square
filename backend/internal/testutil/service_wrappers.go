package testutil

import (
	"github.com/Baaaki/digital-square/internal/models"
	"github.com/Baaaki/digital-square/internal/service"
	"github.com/google/uuid"
)

// TestMessageService wraps MessageService to accept string UUIDs for SQLite compatibility
type TestMessageService struct {
	*service.MessageService
}

// WrapMessageService creates a test wrapper for MessageService
func WrapMessageService(svc *service.MessageService) *TestMessageService {
	return &TestMessageService{svc}
}

// SendMessageStr accepts string UUID (SQLite-compatible) and converts to uuid.UUID
func (t *TestMessageService) SendMessageStr(userID, username, content string) (*models.Message, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	return t.MessageService.SendMessage(uid, username, content)
}

// DeleteMessageStr accepts string UUID (SQLite-compatible) and converts to uuid.UUID
func (t *TestMessageService) DeleteMessageStr(messageID, userID string, isAdmin bool) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return t.MessageService.DeleteMessage(messageID, uid, isAdmin)
}
