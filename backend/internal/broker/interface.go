package broker

type Message struct {
	MessageID      string `json:"message_id"`
	UserID         string `json:"user_id"`
	Username       string `json:"username"`
	Content        string `json:"content"`
	Timestamp      string `json:"timestamp"`
	DeletedByAdmin bool   `json:"deleted_by_admin"`
}

type MessageBroker interface {
	// Publish sends a message to the global chat channel
	Publish(msg Message) error

	// Subscribe returns a receive-only channel for incoming messages
	Subscribe() (<-chan Message, error)

	Close() error
}