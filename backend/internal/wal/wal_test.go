package wal

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Baaaki/digital-square/pkg/logger"
)

func TestWAL_WriteAfterCleanup(t *testing.T) {
	// Initialize logger for WAL operations
	logger.Init(false)

	// Setup: Create temp WAL file
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	w, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer w.Close()

	// Step 1: Write 3 messages to WAL
	entries := []WALEntry{
		{MessageID: "msg1", UserID: "user1", Content: "Hello 1", Timestamp: time.Now()},
		{MessageID: "msg2", UserID: "user1", Content: "Hello 2", Timestamp: time.Now()},
		{MessageID: "msg3", UserID: "user1", Content: "Hello 3", Timestamp: time.Now()},
	}

	for _, entry := range entries {
		if err := w.Write(entry); err != nil {
			t.Fatalf("Failed to write entry: %v", err)
		}
	}

	// Verify: WAL should have 3 entries
	allEntries, err := w.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read WAL: %v", err)
	}
	if len(allEntries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(allEntries))
	}

	// Step 2: Simulate batch writer - cleanup first 2 messages
	persistedIDs := []string{"msg1", "msg2"}
	if err := w.Cleanup(persistedIDs); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Verify: WAL should have 1 entry remaining
	remainingEntries, err := w.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read WAL after cleanup: %v", err)
	}
	if len(remainingEntries) != 1 {
		t.Fatalf("Expected 1 entry after cleanup, got %d", len(remainingEntries))
	}
	if remainingEntries[0].MessageID != "msg3" {
		t.Fatalf("Expected msg3, got %s", remainingEntries[0].MessageID)
	}

	// Step 3: Write NEW messages after cleanup (THIS WAS THE BUG!)
	newEntries := []WALEntry{
		{MessageID: "msg4", UserID: "user1", Content: "Hello 4", Timestamp: time.Now()},
		{MessageID: "msg5", UserID: "user1", Content: "Hello 5", Timestamp: time.Now()},
	}

	for _, entry := range newEntries {
		if err := w.Write(entry); err != nil {
			t.Fatalf("Failed to write NEW entry after cleanup: %v", err)
		}
	}

	// Verify: WAL should have 3 entries now (msg3, msg4, msg5)
	finalEntries, err := w.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read WAL after new writes: %v", err)
	}
	if len(finalEntries) != 3 {
		t.Fatalf("Expected 3 entries after new writes, got %d", len(finalEntries))
	}

	// Verify message IDs
	expectedIDs := []string{"msg3", "msg4", "msg5"}
	for i, entry := range finalEntries {
		if entry.MessageID != expectedIDs[i] {
			t.Fatalf("Expected %s at index %d, got %s", expectedIDs[i], i, entry.MessageID)
		}
	}

	t.Log("✅ WAL works correctly after cleanup!")
}

func TestWAL_MultipleCleanups(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test_multi.wal")

	w, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer w.Close()

	// Write 5 messages
	for i := 1; i <= 5; i++ {
		entry := WALEntry{
			MessageID: "msg" + string(rune('0'+i)),
			UserID:    "user1",
			Content:   "Message " + string(rune('0'+i)),
			Timestamp: time.Now(),
		}
		if err := w.Write(entry); err != nil {
			t.Fatalf("Failed to write: %v", err)
		}
	}

	// Cleanup 1: Remove first 2
	w.Cleanup([]string{"msg1", "msg2"})

	// Write new message
	w.Write(WALEntry{MessageID: "msg6", UserID: "user1", Content: "Message 6", Timestamp: time.Now()})

	// Cleanup 2: Remove next 2
	w.Cleanup([]string{"msg3", "msg4"})

	// Write new message
	w.Write(WALEntry{MessageID: "msg7", UserID: "user1", Content: "Message 7", Timestamp: time.Now()})

	// Final check
	finalEntries, _ := w.ReadAll()
	if len(finalEntries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(finalEntries))
	}

	expectedIDs := []string{"msg5", "msg6", "msg7"}
	for i, entry := range finalEntries {
		if entry.MessageID != expectedIDs[i] {
			t.Fatalf("Expected %s, got %s", expectedIDs[i], entry.MessageID)
		}
	}

	t.Log("✅ Multiple cleanups work correctly!")
}
