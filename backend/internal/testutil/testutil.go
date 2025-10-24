package testutil

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestDatabase holds test database connection (in-memory SQLite)
type TestDatabase struct {
	DB  *gorm.DB
	DSN string
}

// TestRedis holds test Redis mock (miniredis)
type TestRedis struct {
	Server *miniredis.Miniredis
	URL    string
}

// TestUser is a SQLite-compatible version of models.User for testing
type TestUser struct {
	ID           string `gorm:"type:text;primaryKey"` // SQLite uses TEXT for UUID
	Username     string `gorm:"type:varchar(50);uniqueIndex;not null"`
	Email        string `gorm:"type:varchar(100);uniqueIndex;not null"`
	PasswordHash string `gorm:"type:varchar(255);not null"`
	Role         string `gorm:"type:varchar(20);not null;default:'user'"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// TableName overrides the table name for GORM
func (TestUser) TableName() string {
	return "users"
}

// TestMessage is a SQLite-compatible version of models.Message for testing
type TestMessage struct {
	ID               uint64         `gorm:"primaryKey;autoIncrement"`
	MessageID        string         `gorm:"type:varchar(50);uniqueIndex;not null"`
	UserID           string         `gorm:"type:text;not null;index"` // SQLite uses TEXT for UUID
	Content          string         `gorm:"type:text;not null"`
	CreatedAt        time.Time      `gorm:"index"`
	DeletedAt        sql.NullTime   `gorm:"index"`
	DeletedBy        sql.NullString `gorm:"type:text"` // UUID as text
	IsDeletedByAdmin bool           `gorm:"default:false"`
	User             TestUser       `gorm:"foreignKey:UserID;references:ID"`
}

// TableName overrides the table name for GORM
func (TestMessage) TableName() string {
	return "messages"
}

// SetupTestDatabase creates an in-memory SQLite database for integration tests
// No Docker required! Fast and isolated.
func SetupTestDatabase(t *testing.T) *TestDatabase {
	// Use in-memory SQLite database (":memory:" means RAM-only)
	dsn := "file::memory:?cache=shared"

	// Connect with GORM
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto-migrate SQLite-compatible test models
	err = db.AutoMigrate(&TestUser{}, &TestMessage{})
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return &TestDatabase{
		DB:  db,
		DSN: dsn,
	}
}

// Teardown cleans up the test database (closes connection)
func (td *TestDatabase) Teardown(t *testing.T) {
	sqlDB, err := td.DB.DB()
	if err != nil {
		t.Logf("Warning: Failed to get underlying DB: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		t.Logf("Warning: Failed to close database: %v", err)
	}
}

// SetupTestRedis creates an in-memory Redis mock (miniredis)
// No Docker required! Fast and isolated.
func SetupTestRedis(t *testing.T) *TestRedis {
	// Start miniredis (in-memory Redis mock)
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	// Get Redis URL (format: redis://localhost:PORT)
	redisURL := fmt.Sprintf("redis://%s", server.Addr())

	return &TestRedis{
		Server: server,
		URL:    redisURL,
	}
}

// Teardown cleans up the test Redis mock
func (tr *TestRedis) Teardown(t *testing.T) {
	tr.Server.Close()
}

// CleanDatabase deletes all records from tables (for test isolation)
func CleanDatabase(t *testing.T, db *gorm.DB) {
	// Delete all records from tables (SQLite doesn't support TRUNCATE)
	tables := []string{"messages", "users"}
	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("DELETE FROM %s", table)).Error; err != nil {
			t.Logf("Warning: Failed to clean table %s: %v", table, err)
		}
	}
}
