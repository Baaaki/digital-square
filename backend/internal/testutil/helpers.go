package testutil

import (
	"testing"

	"github.com/google/uuid"
)

// ParseUUID parses a UUID string and fails the test if invalid
// This is useful for converting TestUser.ID (string) to uuid.UUID
func ParseUUID(t *testing.T, uuidStr string) uuid.UUID {
	id, err := uuid.Parse(uuidStr)
	if err != nil {
		t.Fatalf("Invalid UUID string: %s, error: %v", uuidStr, err)
	}
	return id
}

// MustParseUUID parses a UUID string and panics if invalid
// Use this for test setup where you're confident the UUID is valid
func MustParseUUID(uuidStr string) uuid.UUID {
	id, err := uuid.Parse(uuidStr)
	if err != nil {
		panic("Invalid UUID: " + uuidStr)
	}
	return id
}
