package broker

import "github.com/Baaaki/digital-square/internal/models"

// MessageBroker provides caching for recent messages
// Phase 1-2: Cache only (single node architecture)
// Phase 3: Pub/Sub will be added for multi-node communication
type MessageBroker interface {
	// Cache operations (Phase 1-2)
	CacheMessage(msg models.Message) error
	GetRecentMessages(limit int) ([]models.Message, error)

	Close() error

	// Phase 3: Uncomment for multi-node deployment
	// Publish(msg Message) error
	// Subscribe() (<-chan Message, error)
}