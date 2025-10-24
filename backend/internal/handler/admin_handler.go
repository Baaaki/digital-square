package handler

import (
	"net/http"

	"github.com/Baaaki/digital-square/internal/service"
	"github.com/Baaaki/digital-square/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AdminHandler struct {
	authService *service.AuthService
}

func NewAdminHandler(authService *service.AuthService) *AdminHandler {
	return &AdminHandler{
		authService: authService,
	}
}

// Request types
type BanUserRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Reason string `json:"reason" binding:"required"`
}

type BanBulkRequest struct {
	UserIDs []string `json:"user_ids" binding:"required"`
	Reason  string   `json:"reason" binding:"required"`
}

// GetAllUsers returns all users (including banned ones)
// GET /admin/users
func (h *AdminHandler) GetAllUsers(c *gin.Context) {
	logger.Log.Info("Admin fetching all users",
		zap.String("admin_id", c.GetString("user_id")),
	)

	users, err := h.authService.GetAllUsers()
	if err != nil {
		logger.Log.Error("Failed to fetch users",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch users",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
	})
}

// BanUser bans a single user (soft delete + delete all messages)
// POST /admin/ban
func (h *AdminHandler) BanUser(c *gin.Context) {
	var req BanUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Log.Warn("Ban user request parsing failed",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	adminID := c.GetString("user_id")
	logger.Log.Info("Admin banning user",
		zap.String("admin_id", adminID),
		zap.String("target_user_id", req.UserID),
		zap.String("reason", req.Reason),
	)

	if err := h.authService.BanUser(req.UserID, adminID, req.Reason); err != nil {
		logger.Log.Error("Failed to ban user",
			zap.Error(err),
			zap.String("user_id", req.UserID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to ban user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User banned successfully",
	})
}

// BanBulk bans multiple users at once
// POST /admin/ban-bulk
func (h *AdminHandler) BanBulk(c *gin.Context) {
	var req BanBulkRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Log.Warn("Bulk ban request parsing failed",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	adminID := c.GetString("user_id")
	logger.Log.Info("Admin bulk banning users",
		zap.String("admin_id", adminID),
		zap.Int("count", len(req.UserIDs)),
		zap.String("reason", req.Reason),
	)

	if err := h.authService.BanBulk(req.UserIDs, adminID, req.Reason); err != nil {
		logger.Log.Error("Failed to bulk ban users",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to ban users",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Users banned successfully",
	})
}
