package wal

import (
    "bufio"
    "encoding/json"
    "os"
    "path/filepath"
    "sync"
    "time"
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

    file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
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
    w.mu.Lock()
    defer w.mu.Unlock()

    data, err := json.Marshal(entry)
    if err != nil {
        return err
    }

    // Write to file
    _, err = w.file.WriteString(string(data) + "\n")
    if err != nil {
        return err
    }

    // Force sync to disk (durability)
    return w.file.Sync()
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
    w.mu.Lock()
    defer w.mu.Unlock()

    // Read all entries
    allEntries, err := w.readAllUnsafe()
    if err != nil {
        return err
    }

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

    // Rewrite WAL file with only remaining entries
    tempFile := w.filePath + ".tmp"
    f, err := os.Create(tempFile)
    if err != nil {
        return err
    }
    defer f.Close()

    for _, entry := range remainingEntries {
        data, _ := json.Marshal(entry)
        f.WriteString(string(data) + "\n")
    }

    f.Sync()

    // Replace old file with new one (atomic)
    return os.Rename(tempFile, w.filePath)
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