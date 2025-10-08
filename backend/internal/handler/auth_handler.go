package handler

import (
    "net/http"

    "github.com/Baaaki/digital-square/internal/service"
    "github.com/gin-gonic/gin"
)

type AuthHandler struct {
    authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
    return &AuthHandler{
        authService: authService,
    }
}

type RegisterRequest struct {
    Username string `json:"username" binding:"required"`
    Email    string `json:"email" binding:"required"`
    Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
    Email    string `json:"email" binding:"required"`
    Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
    var req RegisterRequest
    
    // 1. Parse JSON request
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid request body",
        })
        return
    }
    
    // 2. Call service
    user, token, err := h.authService.Register(req.Username, req.Email, req.Password)
    if err != nil {
        // Handle different error types
        statusCode := http.StatusBadRequest
        
        // You can add more specific error handling here
        // For now, all service errors return 400
        
        c.JSON(statusCode, gin.H{
            "error": err.Error(),
        })
        return
    }
    
    // 3. Return success response
    c.JSON(http.StatusCreated, gin.H{
        "message": "User registered successfully",
        "token":   token,
        "user": gin.H{
            "id":       user.ID,
            "username": user.Username,
            "email":    user.Email,
            "role":     user.Role,
        },
    })
}


func (h *AuthHandler) Login(c *gin.Context) {
    var req LoginRequest
    
    // 1. Parse JSON request
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid request body",
        })
        return
    }
    
    // 2. Call service
    user, token, err := h.authService.Login(req.Email, req.Password)
    if err != nil {
        // Handle different error types
        statusCode := http.StatusUnauthorized
        
        // Invalid credentials should return 401
        if err == service.ErrInvalidCredentials {
            statusCode = http.StatusUnauthorized
        }
        
        c.JSON(statusCode, gin.H{
            "error": err.Error(),
        })
        return
    }
    
    // 3. Return success response
    c.JSON(http.StatusOK, gin.H{
        "message": "Login successful",
        "token":   token,
        "user": gin.H{
            "id":       user.ID,
            "username": user.Username,
            "email":    user.Email,
            "role":     user.Role,
        },
    })
}