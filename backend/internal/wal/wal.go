package wal

import (
    "bufio"
    "encoding/json"
    "os"
    "path/filepath"
    "sync"
    "time"

    "github.com/Baaaki/digital-square/pkg/logger"
    "go.uber.org/zap"
)

// WALEntry represents a message in the WAL
type WALEntry struct {
    MessageID string    `json:"message_id"`
    UserID    string    `json:"user_id"`
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
}

// WAL manages write-ahead log
type WAL struct {
    filePath string
    file     *os.File
    mu       sync.Mutex
}

// NewWAL creates a new WAL instance
func NewWAL(filePath string) (*WAL, error) {
    // Create directory if it doesn't exist
    dir := filepath.Dir(filePath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, err
    }

    // Open file with READ+WRITE+APPEND mode (for concurrent read/write)
    file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
    if err != nil {
        return nil, err
    }

    return &WAL{
        filePath: filePath,
        file:     file,
    }, nil
}

// Write appends a message to WAL
func (w *WAL) Write(entry WALEntry) error {
    start := time.Now()
    w.mu.Lock()
    defer w.mu.Unlock()

    data, err := json.Marshal(entry)
    if err != nil {
        logger.Log.Error("WAL: Failed to marshal entry",
            zap.String("message_id", entry.MessageID),
            zap.Error(err),
        )
        return err
    }

    // Write to file
    writeStart := time.Now()
    _, err = w.file.WriteString(string(data) + "\n")
    if err != nil {
        logger.Log.Error("WAL: Failed to write to file",
            zap.String("message_id", entry.MessageID),
            zap.Error(err),
        )
        return err
    }

    // Force sync to disk (durability)
    syncStart := time.Now()
    if err := w.file.Sync(); err != nil {
        logger.Log.Error("WAL: Failed to sync to disk",
            zap.String("message_id", entry.MessageID),
            zap.Error(err),
        )
        return err
    }
    syncDuration := time.Since(syncStart)

    logger.Log.Debug("WAL: Entry written and synced",
        zap.String("message_id", entry.MessageID),
        zap.Duration("write_duration", time.Since(writeStart)),
        zap.Duration("sync_duration", syncDuration),
        zap.Duration("total_duration", time.Since(start)),
    )

    return nil
}

// ReadAll reads all entries from WAL
func (w *WAL) ReadAll() ([]WALEntry, error) {
    w.mu.Lock()
    defer w.mu.Unlock()

    return w.readAllUnsafe()  // ← Ortak kodu çağır
}

// GetAllEntries returns all entries from WAL (for batch writer)
// Returns empty slice if WAL is empty (no unnecessary processing)
func (w *WAL) GetAllEntries() ([]WALEntry, error) {
    w.mu.Lock()
    defer w.mu.Unlock()

    entries, err := w.readAllUnsafe()
    if err != nil {
        return nil, err
    }

    return entries, nil
}

// Cleanup removes entries that have been persisted to PostgreSQL
func (w *WAL) Cleanup(persistedIDs []string) error {
    start := time.Now()
    w.mu.Lock()
    defer w.mu.Unlock()

    logger.Log.Debug("WAL: Starting cleanup",
        zap.Int("persisted_count", len(persistedIDs)),
    )

    // Read all entries
    allEntries, err := w.readAllUnsafe()
    if err != nil {
        logger.Log.Error("WAL: Failed to read entries for cleanup",
            zap.Error(err),
        )
        return err
    }

    beforeCount := len(allEntries)

    // Create map for fast lookup
    persistedMap := make(map[string]bool)
    for _, id := range persistedIDs {
        persistedMap[id] = true
    }

    // Filter out persisted entries
    var remainingEntries []WALEntry
    for _, entry := range allEntries {
        if !persistedMap[entry.MessageID] {
            remainingEntries = append(remainingEntries, entry)
        }
    }

    afterCount := len(remainingEntries)
    deletedCount := beforeCount - afterCount

    // Close the current file before replacing it
    if err := w.file.Close(); err != nil {
        logger.Log.Error("WAL: Failed to close file for cleanup",
            zap.Error(err),
        )
        return err
    }

    // Rewrite WAL file with only remaining entries
    tempFile := w.filePath + ".tmp"
    f, err := os.Create(tempFile)
    if err != nil {
        logger.Log.Error("WAL: Failed to create temp file",
            zap.String("temp_file", tempFile),
            zap.Error(err),
        )
        return err
    }

    for _, entry := range remainingEntries {
        data, _ := json.Marshal(entry)
        f.WriteString(string(data) + "\n")
    }

    f.Sync()
    f.Close()

    // Replace old file with new one (atomic)
    if err := os.Rename(tempFile, w.filePath); err != nil {
        logger.Log.Error("WAL: Failed to rename temp file",
            zap.String("temp_file", tempFile),
            zap.String("target_file", w.filePath),
            zap.Error(err),
        )
        return err
    }

    // Reopen the file with same flags (CRITICAL!)
    newFile, err := os.OpenFile(w.filePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
    if err != nil {
        logger.Log.Error("WAL: Failed to reopen file after cleanup",
            zap.String("file_path", w.filePath),
            zap.Error(err),
        )
        return err
    }

    // Update the file pointer to the new file
    w.file = newFile

    logger.Log.Info("WAL: Cleanup completed",
        zap.Int("before_count", beforeCount),
        zap.Int("deleted_count", deletedCount),
        zap.Int("remaining_count", afterCount),
        zap.Duration("duration", time.Since(start)),
    )

    return nil
}

// readAllUnsafe reads all entries without locking (internal use only)
func (w *WAL) readAllUnsafe() ([]WALEntry, error) {
    file, err := os.Open(w.filePath)
    if err != nil {
        if os.IsNotExist(err) {
            return []WALEntry{}, nil
        }
        return nil, err
    }
    defer file.Close()

    var entries []WALEntry
    scanner := bufio.NewScanner(file)

    for scanner.Scan() {
        var entry WALEntry
        if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
            continue
        }
        entries = append(entries, entry)
    }

    return entries, scanner.Err()
}

// Close closes the WAL file
func (w *WAL) Close() error {
    w.mu.Lock()
    defer w.mu.Unlock()
    return w.file.Close()
}