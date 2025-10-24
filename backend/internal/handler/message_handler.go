package handler

import (
	"net/http"
	"strconv"

	"github.com/Baaaki/digital-square/internal/models"
	"github.com/Baaaki/digital-square/internal/service"
	"github.com/Baaaki/digital-square/internal/utils"
	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	messageService *service.MessageService
}

func NewMessageHandler(messageService *service.MessageService) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
	}
}

// GET /api/messages/before/:id
func (h *MessageHandler) GetBefore(c *gin.Context) {
	// 1. Auth check
	claims, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	userClaims := claims.(*utils.Claims)
	isAdmin := userClaims.Role == models.RoleAdmin

	// 2.step: Parse message ID from URL
	messageIDStr := c.Param("id")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message ID"})
		return
	}

	//3.step: Fetch 50 older messages fron postgreSQL
	limit := 50
	messages, err := h.messageService.GetMessagesBefore(messageID, limit, isAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch messages"})
		return
	}

	// 4. Filter deleted messages based on role
	filteredMessages := h.filterMessages(messages, isAdmin)

	c.JSON(http.StatusOK, gin.H{
		"messages": filteredMessages,
		"count":    len(filteredMessages),
		"has_more": len(filteredMessages) == limit,
	})
}

// filterMessages masks deleted message content based on user role
func (h *MessageHandler) filterMessages(messages []models.Message, isAdmin bool) []gin.H {
	result := make([]gin.H, 0, len(messages))

	for _, msg := range messages {
		msgData := gin.H{
			"id":         msg.ID,
			"message_id": msg.MessageID,
			"user_id":    msg.UserID,
			"username":   msg.Username, // ✅ Use denormalized username field
			"content":    msg.Content,
			"created_at": msg.CreatedAt,
			"deleted":    msg.DeletedAt.Valid,
		}

		// Handle deleted messages
		if msg.DeletedAt.Valid {
			if isAdmin {
				// Admin sees content + metadata
				msgData["deleted_by_admin"] = msg.IsDeletedByAdmin
				msgData["deleted_by"] = msg.DeletedBy
			} else {
				// User sees placeholder
				if msg.IsDeletedByAdmin {
					msgData["content"] = "This message was deleted by admin"
				} else {
					msgData["content"] = "This message was deleted"
				}
			}
		}

		result = append(result, msgData)
	}

	return result
}
