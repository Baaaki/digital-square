package testutil

import (
	"database/sql"
	"time"

	"github.com/Baaaki/digital-square/internal/models"
	"github.com/Baaaki/digital-square/internal/utils"
	"github.com/google/uuid"
)

// CreateTestUser creates a SQLite-compatible test user with hashed password
func CreateTestUser(username, email, password string, role models.Role) (*TestUser, error) {
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	return &TestUser{
		ID:           uuid.New().String(), // SQLite stores UUID as string
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         string(role),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

// CreateTestMessage creates a SQLite-compatible test message
func CreateTestMessage(userID string, content string) *TestMessage {
	return &TestMessage{
		MessageID: uuid.New().String(),
		UserID:    userID, // SQLite stores UUID as string
		Content:   content,
		CreatedAt: time.Now(),
	}
}

// DefaultTestUser returns a default test user (regular user)
func DefaultTestUser() (*TestUser, error) {
	return CreateTestUser("testuser", "test@example.com", "Test123456", models.RoleUser)
}

// DefaultAdminUser returns a default admin user
func DefaultAdminUser() (*TestUser, error) {
	return CreateTestUser("admin", "admin@example.com", "Admin123456", models.RoleAdmin)
}

// CreateTestMessageWithDelete creates a test message with deletion info
func CreateTestMessageWithDelete(userID string, content string, deletedBy string, isDeletedByAdmin bool) *TestMessage {
	now := time.Now()
	return &TestMessage{
		MessageID: uuid.New().String(),
		UserID:    userID,
		Content:   content,
		CreatedAt: now,
		DeletedAt: sql.NullTime{Time: now, Valid: true},
		DeletedBy: sql.NullString{String: deletedBy, Valid: true},
		IsDeletedByAdmin: isDeletedByAdmin,
	}
}
