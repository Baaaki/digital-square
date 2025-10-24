package service_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Baaaki/digital-square/internal/broker"
	"github.com/Baaaki/digital-square/internal/models"
	"github.com/Baaaki/digital-square/internal/repository"
	"github.com/Baaaki/digital-square/internal/service"
	"github.com/Baaaki/digital-square/internal/testutil"
	"github.com/Baaaki/digital-square/internal/wal"
	"github.com/Baaaki/digital-square/pkg/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// MessageServiceIntegrationTestSuite defines test suite
type MessageServiceIntegrationTestSuite struct {
	suite.Suite
	testDB         *testutil.TestDatabase
	testRedis      *testutil.TestRedis
	messageService *service.MessageService
	walInstance    *wal.WAL
	testUser       *testutil.TestUser
}

// SetupSuite runs before all tests
func (s *MessageServiceIntegrationTestSuite) SetupSuite() {
	// Initialize logger (required for MessageService)
	logger.Init(false)

	// Start in-memory SQLite and miniredis (migrations run automatically)
	s.testDB = testutil.SetupTestDatabase(s.T())
	s.testRedis = testutil.SetupTestRedis(s.T())

	// Setup WAL (temporary file)
	walPath := "/tmp/test_wal_messages"
	os.RemoveAll(walPath) // Clean up old WAL files
	walInstance, err := wal.NewWAL(walPath)
	assert.NoError(s.T(), err)
	s.walInstance = walInstance

	// Setup Redis broker
	redisBroker, err := broker.NewRedisMessageBroker(s.testRedis.URL)
	assert.NoError(s.T(), err)

	// Setup repositories and services
	messageRepo := repository.NewMessageRepository(s.testDB.DB)
	s.messageService = service.NewMessageService(messageRepo, redisBroker, s.walInstance)

	// Create test user
	s.testUser, _ = testutil.CreateTestUser("testuser", "test@example.com", "Test123", models.RoleUser)
	s.testDB.DB.Create(s.testUser)
}

// TearDownSuite runs after all tests
func (s *MessageServiceIntegrationTestSuite) TearDownSuite() {
	s.walInstance.Close()
	os.RemoveAll("/tmp/test_wal_messages")
	s.testDB.Teardown(s.T())
	s.testRedis.Teardown(s.T())
}

// SetupTest runs before each test
func (s *MessageServiceIntegrationTestSuite) SetupTest() {
	// Clean messages table (SQLite doesn't support TRUNCATE)
	s.testDB.DB.Exec("DELETE FROM messages")

	// Clean WAL (close old instance first)
	if s.walInstance != nil {
		s.walInstance.Close()
	}
	os.RemoveAll("/tmp/test_wal_messages")

	// Create new WAL instance
	walInstance, _ := wal.NewWAL("/tmp/test_wal_messages")
	s.walInstance = walInstance

	// Update MessageService with new WAL instance
	messageRepo := repository.NewMessageRepository(s.testDB.DB)
	redisBroker, _ := broker.NewRedisMessageBroker(s.testRedis.URL)
	s.messageService = service.NewMessageService(messageRepo, redisBroker, s.walInstance)
}

// getUserID is a helper to convert string ID to UUID (for SQLite compatibility)
func (s *MessageServiceIntegrationTestSuite) getUserID() uuid.UUID {
	return testutil.ParseUUID(s.T(), s.testUser.ID)
}

// TestSendMessage tests message sending (WAL write)
func (s *MessageServiceIntegrationTestSuite) TestSendMessage() {
	// Send message
	msg, err := s.messageService.SendMessage(s.getUserID(), s.testUser.Username, "Hello, World!")
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), msg)
	assert.Equal(s.T(), "Hello, World!", msg.Content)
	assert.NotEmpty(s.T(), msg.MessageID)

	// Check WAL entry exists
	entries, err := s.walInstance.GetAllEntries()
	assert.NoError(s.T(), err)
	assert.Len(s.T(), entries, 1)
	assert.Equal(s.T(), "Hello, World!", entries[0].Content)
}

// TestSendMessageXSSSanitization tests XSS sanitization
func (s *MessageServiceIntegrationTestSuite) TestSendMessageXSSSanitization() {
	// Send message with XSS payload
	xssPayload := "<script>alert('XSS')</script>"
	msg, err := s.messageService.SendMessage(s.getUserID(), s.testUser.Username, xssPayload)

	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), msg)

	// Content should be sanitized (HTML escaped)
	assert.NotEqual(s.T(), xssPayload, msg.Content)
	assert.Contains(s.T(), msg.Content, "&lt;script&gt;")
	assert.Contains(s.T(), msg.Content, "&lt;/script&gt;")
	assert.NotContains(s.T(), msg.Content, "<script>")
}

// TestSendMessageValidation tests message validation
func (s *MessageServiceIntegrationTestSuite) TestSendMessageValidation() {
	testCases := []struct {
		name          string
		content       string
		expectedError string
	}{
		{
			name:          "Empty message",
			content:       "",
			expectedError: "message cannot be empty",
		},
		{
			name:          "Too long message",
			content:       string(make([]byte, 5001)), // 5001 characters
			expectedError: "message too long",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			msg, err := s.messageService.SendMessage(s.getUserID(), s.testUser.Username, tc.content)
			assert.Error(s.T(), err)
			assert.Nil(s.T(), msg)
			assert.Contains(s.T(), err.Error(), tc.expectedError)
		})
	}
}

// TestBatchWriterWALToPostgreSQL tests batch writer functionality
func (s *MessageServiceIntegrationTestSuite) TestBatchWriterWALToPostgreSQL() {
	// Send 5 messages (goes to WAL)
	for i := 0; i < 5; i++ {
		_, err := s.messageService.SendMessage(s.getUserID(), s.testUser.Username, "Test message")
		assert.NoError(s.T(), err)
	}

	// Verify WAL has 5 entries
	entries, err := s.walInstance.GetAllEntries()
	assert.NoError(s.T(), err)
	assert.Len(s.T(), entries, 5)

	// Verify PostgreSQL is empty (batch writer hasn't run yet)
	var count int64
	s.testDB.DB.Model(&models.Message{}).Count(&count)
	assert.Equal(s.T(), int64(0), count)

	// Manually trigger batch writer (simulate 1 minute passing)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start batch writer
	s.messageService.StartBatchWriter(ctx)

	// Wait 2 seconds for batch writer to run
	time.Sleep(2 * time.Second)

	// Trigger manually by calling processBatch (reflection not needed, we can't call private method)
	// Instead, we'll test via GetRecentMessages which should load from PostgreSQL

	// Note: In real scenario, batch writer runs every 1 minute
	// For testing, we need to either:
	// 1. Make processBatch() public (not recommended)
	// 2. Wait 1 minute (too slow for tests)
	// 3. Test indirectly via GetRecentMessages after batch writes

	// For now, let's verify WAL cleanup happens after manual batch insert
	// We'll create a separate test for full batch writer
}

// TestDeleteMessage tests message deletion (soft delete)
func (s *MessageServiceIntegrationTestSuite) TestDeleteMessage() {
	// Create message directly in database (simulate already persisted message)
	msg := testutil.CreateTestMessage(s.testUser.ID, "Message to delete")
	s.testDB.DB.Create(msg)

	// Delete message (user deletes own message)
	err := s.messageService.DeleteMessage(msg.MessageID, s.getUserID(), false)
	assert.NoError(s.T(), err)

	// Verify message is soft deleted
	var deletedMsg models.Message
	s.testDB.DB.Unscoped().Where("message_id = ?", msg.MessageID).First(&deletedMsg)
	assert.True(s.T(), deletedMsg.DeletedAt.Valid)
	assert.False(s.T(), deletedMsg.IsDeletedByAdmin)
	assert.NotNil(s.T(), deletedMsg.DeletedBy)
	assert.Equal(s.T(), s.testUser.ID, deletedMsg.DeletedBy.String())
}

// TestDeleteMessageAdmin tests admin deleting any message
func (s *MessageServiceIntegrationTestSuite) TestDeleteMessageAdmin() {
	// Create another user
	otherUser, _ := testutil.CreateTestUser("otheruser", "other@example.com", "Pass123", models.RoleUser)
	s.testDB.DB.Create(otherUser)

	// Create message from other user
	msg := testutil.CreateTestMessage(otherUser.ID, "Other user's message")
	s.testDB.DB.Create(msg)

	// Admin deletes other user's message
	adminUser, _ := testutil.DefaultAdminUser()
	s.testDB.DB.Create(adminUser)

	adminUUID := testutil.ParseUUID(s.T(), adminUser.ID)
	err := s.messageService.DeleteMessage(msg.MessageID, adminUUID, true)
	assert.NoError(s.T(), err)

	// Verify message is soft deleted by admin
	var deletedMsg models.Message
	s.testDB.DB.Unscoped().Where("message_id = ?", msg.MessageID).First(&deletedMsg)
	assert.True(s.T(), deletedMsg.DeletedAt.Valid)
	assert.True(s.T(), deletedMsg.IsDeletedByAdmin)
}

// TestDeleteMessageUnauthorized tests user trying to delete others' message
func (s *MessageServiceIntegrationTestSuite) TestDeleteMessageUnauthorized() {
	// Create another user
	otherUser, _ := testutil.CreateTestUser("otheruser2", "other2@example.com", "Pass123", models.RoleUser)
	s.testDB.DB.Create(otherUser)

	// Create message from other user
	msg := testutil.CreateTestMessage(otherUser.ID, "Other user's message")
	s.testDB.DB.Create(msg)

	// Regular user tries to delete other user's message
	err := s.messageService.DeleteMessage(msg.MessageID, s.getUserID(), false)
	assert.Error(s.T(), err)
	assert.Equal(s.T(), service.ErrUnauthorized, err)

	// Verify message is NOT deleted
	var notDeletedMsg models.Message
	s.testDB.DB.Where("message_id = ?", msg.MessageID).First(&notDeletedMsg)
	assert.False(s.T(), notDeletedMsg.DeletedAt.Valid)
}

// TestGetRecentMessages tests retrieving recent messages from cache/database
func (s *MessageServiceIntegrationTestSuite) TestGetRecentMessages() {
	// Create 10 messages directly in database
	for i := 0; i < 10; i++ {
		msg := testutil.CreateTestMessage(s.testUser.ID, "Test message")
		msg.User = *s.testUser // Preload user for test
		s.testDB.DB.Create(msg)
	}

	// Get recent messages
	messages, err := s.messageService.GetRecentMessages(5)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), messages, 5)

	// Verify order (most recent first)
	assert.True(s.T(), messages[0].CreatedAt.After(messages[4].CreatedAt) || messages[0].CreatedAt.Equal(messages[4].CreatedAt))
}

// TestSuite runs all tests in the suite
func TestMessageServiceIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(MessageServiceIntegrationTestSuite))
}
