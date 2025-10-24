package handler

import (
	"net/http"
	"sync"
	"time"

	"github.com/Baaaki/digital-square/internal/models"
	"github.com/Baaaki/digital-square/internal/service"
	"github.com/Baaaki/digital-square/internal/utils"
	"github.com/Baaaki/digital-square/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
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

	// For delete events and initial messages
	Deleted        bool `json:"deleted,omitempty"`
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
		logger.Log.Error("Failed to upgrade WebSocket connection",
			zap.String("user_id", claims.UserID.String()),
			zap.String("username", claims.Username),
			zap.Error(err),
		)
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
	totalClients := len(h.clients)
	h.mu.Unlock()

	logger.Log.Info("WebSocket client connected",
		zap.String("user_id", client.userID.String()),
		zap.String("username", client.username),
		zap.String("role", string(client.role)),
		zap.Int("total_clients", totalClients),
	)
	
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
			logger.Log.Info("WebSocket session expired",
				zap.String("user_id", client.userID.String()),
				zap.String("username", client.username),
				zap.Duration("session_duration", time.Since(client.connectedAt)),
			)
			h.closeClientGracefully(client, "session expired after 15 minutes")
			return

		default:
			client.conn.SetReadDeadline(time.Now().Add(pongWait))

			var req WSRequest
			err := client.conn.ReadJSON(&req)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.Log.Warn("WebSocket unexpected close",
						zap.String("user_id", client.userID.String()),
						zap.String("username", client.username),
						zap.Error(err),
					)
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
	logger.Log.Debug("Message received",
		zap.String("user_id", client.userID.String()),
		zap.String("username", client.username),
		zap.Int("content_length", len(req.Content)),
	)

	if req.Content == "" {
		h.sendAck(client, req.TempID, "", "error", "content cannot be empty")
		return
	}

	msg, err := h.messageService.SendMessage(client.userID, client.username, req.Content)
	if err != nil {
		logger.Log.Error("Failed to send message (WAL Error)",
			zap.String("user_id", client.userID.String()),
			zap.String("username", client.username),
			zap.Error(err),
		)
		h.sendAck(client, req.TempID, "", "error", "failed to write to WAL")
		return
	}

	logger.Log.Info("Message written to WAL",
		zap.String("message_id", msg.MessageID),
		zap.String("user_id", client.userID.String()),
		zap.String("username", client.username),
	)

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

	logger.Log.Debug("Broadcasted message to all clients",
		zap.String("message_id", msg.MessageID),
		zap.Int("client_count", clientCount),
	)

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
		logger.Log.Error("Failed to delete message",
			zap.String("message_id", req.MessageID),
			zap.String("user_id", client.userID.String()),
			zap.Error(err),
		)
		h.sendError(client, err.Error())
		return
	}

	logger.Log.Info("Message deleted",
		zap.String("message_id", req.MessageID),
		zap.String("user_id", client.userID.String()),
		zap.Bool("is_admin", isAdmin),
	)

	// Direct broadcast delete event to all connected clients (in-memory)
	h.broadcastDeleteEvent(req.MessageID, isAdmin)

	// Send success response to deleter
	client.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := client.conn.WriteJSON(WSResponse{
		Type:      "delete_success",
		MessageID: req.MessageID,
	}); err != nil {
		logger.Log.Warn("Failed to send delete success response", zap.Error(err))
	}
}

func (h *WebSocketHandler) broadcastToAll(msg WSResponse) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range h.clients {

		conn.SetWriteDeadline(time.Now().Add(writeWait))

		err := conn.WriteJSON(msg)
		if err != nil {
			logger.Log.Debug("Failed to send message to client", zap.Error(err))
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
			logger.Log.Debug("Failed to broadcast delete event", zap.Error(err))
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
				logger.Log.Debug("Ping failed",
					zap.String("username", client.username),
					zap.Error(err),
				)
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
		logger.Log.Debug("Failed to send session_expired message", zap.Error(err))
	}

	// Send WebSocket close frame (Gorilla WebSocket protocol)
	client.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := client.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason),
	); err != nil {
		logger.Log.Debug("Failed to send close frame", zap.Error(err))
	}

	logger.Log.Info("Closed WebSocket connection gracefully",
		zap.String("username", client.username),
		zap.String("reason", reason),
	)
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
		logger.Log.Info("WebSocket client disconnected",
			zap.String("username", client.username),
			zap.Duration("session_duration", duration),
			zap.Int("remaining_clients", len(h.clients)),
		)
	}
}

func (h *WebSocketHandler) sendError(client *Client, errorMsg string) {
	client.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := client.conn.WriteJSON(WSResponse{
		Type:  "error",
		Error: errorMsg,
	}); err != nil {
		logger.Log.Debug("Failed to send error message", zap.Error(err))
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
		logger.Log.Debug("Failed to send ACK", zap.Error(err))
	}
}

// sendInitialMessages sends last 100 messages from Redis/PostgreSQL to newly connected client
func (h *WebSocketHandler) sendInitialMessages(client *Client) {
	// Get last 100 messages from database (Redis cache or PostgreSQL)
	messages, err := h.messageService.GetRecentMessages(100)
	if err != nil {
		logger.Log.Error("Failed to load initial messages",
			zap.String("username", client.username),
			zap.Error(err),
		)
		return
	}

	isAdmin := client.role == models.RoleAdmin

	// Reverse messages so newest is sent first (frontend expects newest at top)
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]

		// Prepare message content and deleted flag
		content := msg.Content
		deleted := msg.DeletedAt.Valid
		deletedByAdmin := msg.IsDeletedByAdmin

		// Mask deleted message content for non-admin users
		if deleted && !isAdmin {
			if msg.IsDeletedByAdmin {
				content = "This message was deleted by admin"
			} else {
				content = "This message was deleted"
			}
		}

		wsMsg := WSResponse{
			Type:           "message",
			ID:             msg.ID,        // PostgreSQL ID (for pagination)
			MessageID:      msg.MessageID, // UUID (global unique identifier)
			UserID:         msg.UserID.String(),
			Username:       msg.Username, // ✅ Use denormalized username field
			Content:        content,
			Timestamp:      msg.CreatedAt.Format(time.RFC3339),
			Deleted:        deleted,        // ✅ Send deleted flag
			DeletedByAdmin: deletedByAdmin, // ✅ Send deleted_by_admin flag
		}

		client.conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := client.conn.WriteJSON(wsMsg); err != nil {
			logger.Log.Warn("Failed to send initial message",
				zap.String("username", client.username),
				zap.Error(err),
			)
			return
		}
	}

	logger.Log.Info("Sent initial messages to new client",
		zap.String("username", client.username),
		zap.Int("message_count", len(messages)),
	)
}