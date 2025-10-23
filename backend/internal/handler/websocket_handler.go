package handler

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Baaaki/digital-square/internal/models"
	"github.com/Baaaki/digital-square/internal/service"
	"github.com/Baaaki/digital-square/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	maxSessionLifetime = 15 * time.Minute
	writeWait          = 10 * time.Second // Time allowed to write a message to the peer
	pongWait           = 60 * time.Second
	pingPeriod         = (pongWait * 9) / 10 // 54 seconds
	maxMessageSize     = 512 * 1024          // 512 KB
)

type WSMessageType string

const (
	WSMessageTypeSend   WSMessageType = "send_message"
	WSMessageTypeDelete WSMessageType = "delete_message"
)

type WSRequest struct {
	Type      WSMessageType `json:"type"`
	TempID    string        `json:"temp_id,omitempty"`
	Content   string        `json:"content,omitempty"`    // For send_message
	MessageID string        `json:"message_id,omitempty"` // For delete_message
}

type WSResponse struct {
	Type      string `json:"type"` // "message", "ack", "error", "message_deleted", "session_expired"
	ID        uint64 `json:"id,omitempty"`        // PostgreSQL auto-increment ID (for pagination)
	MessageID string `json:"message_id,omitempty"` // UUID (global unique identifier)
	UserID    string `json:"user_id,omitempty"`
	Username  string `json:"username,omitempty"`
	Content   string `json:"content,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Error     string `json:"error,omitempty"`

	// For delete events
	DeletedByAdmin bool `json:"deleted_by_admin,omitempty"`

	//For ACK
	TempID string `json:"temp_id,omitempty"`
	Status string `json:"status,omitempty"`
}

type WebSocketHandler struct {
	messageService *service.MessageService
	jwtSecret      string
	clients        map[*websocket.Conn]*Client
	mu             sync.RWMutex
}

type Client struct {
	conn        *websocket.Conn
	userID      uuid.UUID
	username    string
	role        models.Role
	connectedAt time.Time
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // add origin check in production
	},
}

func NewWebSocketHandler(
	messageService *service.MessageService,
	jwtSecret string,
) *WebSocketHandler {
	return &WebSocketHandler{
		messageService: messageService,
		jwtSecret:      jwtSecret,
		clients:        make(map[*websocket.Conn]*Client),
	}
}

func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// Get claims from context (set by AuthMiddleware)
	claimsInterface, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Type assertion to convert interface{} to *utils.Claims
	claims, ok := claimsInterface.(*utils.Claims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid claims format"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &Client{
		conn:        conn,
		userID:      claims.UserID,
		username:    claims.Username,
		role:        claims.Role,
		connectedAt: time.Now(),
	}

	h.mu.Lock()
	h.clients[conn] = client
	h.mu.Unlock()

	log.Printf("Client connected: %s (total: %d)", client.username, len(h.clients))
	
	// ✅ SEND INITIAL 100 MESSAGES FROM REDIS/POSTGRESQL
	go h.sendInitialMessages(client)

	defer h.removeClient(conn)

	h.handleClient(client)
}

// handleClient listens for messages from a specific client
func (h *WebSocketHandler) handleClient(client *Client) {
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetReadLimit(maxMessageSize)

	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	sessionTimer := time.NewTimer(maxSessionLifetime)
	defer sessionTimer.Stop()

	done := make(chan struct{})
	defer close(done)

	go h.pingClient(client, ticker, done)

	for {
		select {
		case <-sessionTimer.C:
			log.Printf("Session expired for %s (15 minutes)", client.username)
			h.closeClientGracefully(client, "session expired after 15 minutes")
			return

		default:
			client.conn.SetReadDeadline(time.Now().Add(pongWait))

			var req WSRequest
			err := client.conn.ReadJSON(&req)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}

			switch req.Type {
			case WSMessageTypeSend:
				h.handleSendMessage(client, req)

			case WSMessageTypeDelete:
				h.handleDeleteMessage(client, req)

			default:
				h.sendError(client, "unknown message type")
			}
		}
	}
}

func (h *WebSocketHandler) handleSendMessage(client *Client, req WSRequest) {
	log.Printf("🔵 Message received from %s (content: %.30s...)", client.username, req.Content)

	if req.Content == "" {
		h.sendAck(client, req.TempID, "", "error", "content cannot be empty")
		return
	}

	msg, err := h.messageService.SendMessage(client.userID, client.username, req.Content)
	if err != nil {
		log.Printf("❌ Failed to send message (WAL Error): %v", err)
		h.sendAck(client, req.TempID, "", "error", "failed to write to WAL")
		return
	}

	log.Printf("✅ Message %s written to WAL", msg.MessageID)

	// Direct broadcast to all connected clients (in-memory, same node)
	h.mu.RLock()
	clientCount := len(h.clients)
	h.mu.RUnlock()

	h.broadcastToAll(WSResponse{
		Type:      "message",
		ID:        msg.ID,        // PostgreSQL ID (for pagination)
		MessageID: msg.MessageID, // UUID (global unique identifier)
		UserID:    client.userID.String(),
		Username:  client.username,
		Content:   msg.Content,
		Timestamp: msg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})

	log.Printf("📤 Broadcasted message to %d clients", clientCount)

	h.sendAck(client, req.TempID, msg.MessageID, "success", "")
}

func (h *WebSocketHandler) handleDeleteMessage(client *Client, req WSRequest) {
	// Validate message ID
	if req.MessageID == "" {
		h.sendError(client, "message_id is required")
		return
	}

	isAdmin := client.role == models.RoleAdmin
	err := h.messageService.DeleteMessage(req.MessageID, client.userID, isAdmin)
	if err != nil {
		log.Printf("Failed to delete message: %v", err)
		h.sendError(client, err.Error())
		return
	}

	// Direct broadcast delete event to all connected clients (in-memory)
	h.broadcastDeleteEvent(req.MessageID, isAdmin)

	// Send success response to deleter
	client.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := client.conn.WriteJSON(WSResponse{
		Type:      "delete_success",
		MessageID: req.MessageID,
	}); err != nil {
		log.Printf("Failed to send delete success response: %v", err)
	}
}

func (h *WebSocketHandler) broadcastToAll(msg WSResponse) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range h.clients {

		conn.SetWriteDeadline(time.Now().Add(writeWait))

		err := conn.WriteJSON(msg)
		if err != nil {
			log.Printf("Failed to send to client: %v", err)
			// Don't remove client here, handleClient will do cleanup
		}
	}
}

func (h *WebSocketHandler) broadcastDeleteEvent(messageID string, deletedByAdmin bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	deleteMsg := WSResponse{
		Type:           "message_deleted",
		MessageID:      messageID,
		DeletedByAdmin: deletedByAdmin,
	}

	for conn := range h.clients {
		conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteJSON(deleteMsg); err != nil {
			log.Printf("Failed to broadcast delete event: %v", err)
		}
	}
}

func (h *WebSocketHandler) pingClient(client *Client, ticker *time.Ticker, done <-chan struct{}) {
	for {
		select {
		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))

			// Send ping message
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping failed for %s: %v", client.username, err)
				return
			}

		case <-done:
			// handleClient exited, stop pinging
			return
		}
	}
}

func (h *WebSocketHandler) closeClientGracefully(client *Client, reason string) {
	client.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := client.conn.WriteJSON(WSResponse{
		Type:  "session_expired",
		Error: reason,
	}); err != nil {
		log.Printf("Failed to send session_expired message: %v", err)
	}

	// Send WebSocket close frame (Gorilla WebSocket protocol)
	client.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := client.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason),
	); err != nil {
		log.Printf("Failed to send close frame: %v", err)
	}

	log.Printf("Closed connection for %s: %s", client.username, reason)
}

func (h *WebSocketHandler) removeClient(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, exists := h.clients[conn]
	if exists {
		delete(h.clients, conn)
		conn.Close()

		// Calculate session duration
		duration := time.Since(client.connectedAt)
		log.Printf("Client disconnected: %s (session duration: %v, remaining: %d)",
			client.username, duration.Round(time.Second), len(h.clients))
	}
}

func (h *WebSocketHandler) sendError(client *Client, errorMsg string) {
	client.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := client.conn.WriteJSON(WSResponse{
		Type:  "error",
		Error: errorMsg,
	}); err != nil {
		log.Printf("Failed to send error message: %v", err)
	}
}

func (h *WebSocketHandler) sendAck(client *Client, tempID, messageID, status, errorMsg string) {
	client.conn.SetWriteDeadline(time.Now().Add(writeWait))

	ackResponse := WSResponse{
		Type:      "ack",
		TempID:    tempID,
		MessageID: messageID, // If success is full, if error is empty
		Status:    status,
	}

	if status == "error" {
		ackResponse.Error = errorMsg
	}

	if err := client.conn.WriteJSON(ackResponse); err != nil {
		log.Printf("Failed to send ACK: %v", err)
	}
}

// sendInitialMessages sends last 100 messages from Redis/PostgreSQL to newly connected client
func (h *WebSocketHandler) sendInitialMessages(client *Client) {
	// Get last 100 messages from database (Redis cache or PostgreSQL)
	messages, err := h.messageService.GetRecentMessages(100)
	if err != nil {
		log.Printf("Failed to load initial messages for %s: %v", client.username, err)
		return
	}

	isAdmin := client.role == models.RoleAdmin

	// Send each message to client
	for _, msg := range messages {
		// Skip deleted messages for non-admin users
		if msg.DeletedAt.Valid && !isAdmin {
			continue
		}

		content := msg.Content
		// Mask deleted message content for non-admin users
		if msg.DeletedAt.Valid && !isAdmin {
			if msg.IsDeletedByAdmin {
				content = "This message was deleted by admin"
			} else {
				content = "This message was deleted"
			}
		}

		wsMsg := WSResponse{
			Type:      "message",
			ID:        msg.ID,        // PostgreSQL ID (for pagination)
			MessageID: msg.MessageID, // UUID (global unique identifier)
			UserID:    msg.UserID.String(),
			Username:  msg.User.Username,
			Content:   content,
			Timestamp: msg.CreatedAt.Format(time.RFC3339),
		}

		client.conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := client.conn.WriteJSON(wsMsg); err != nil {
			log.Printf("Failed to send initial message to %s: %v", client.username, err)
			return
		}
	}

	log.Printf("Sent %d initial messages to %s", len(messages), client.username)
}