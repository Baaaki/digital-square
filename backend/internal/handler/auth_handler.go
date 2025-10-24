package handler

import (
    "net/http"

    "github.com/Baaaki/digital-square/internal/service"
    "github.com/Baaaki/digital-square/pkg/logger"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
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
        logger.Log.Warn("Registration request parsing failed",
            zap.String("ip", c.ClientIP()),
            zap.Error(err),
        )
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid request body",
        })
        return
    }

    logger.Log.Info("User registration attempt",
        zap.String("username", req.Username),
        zap.String("email", req.Email),
        zap.String("ip", c.ClientIP()),
    )

    // 2. Call service
    user, token, err := h.authService.Register(req.Username, req.Email, req.Password)
    if err != nil {
        logger.Log.Error("Registration failed",
            zap.String("username", req.Username),
            zap.String("email", req.Email),
            zap.Error(err),
        )

        // Handle different error types
        statusCode := http.StatusBadRequest

        // You can add more specific error handling here
        // For now, all service errors return 400

        c.JSON(statusCode, gin.H{
            "error": err.Error(),
        })
        return
    }

    // 3. Set token in HTTP-only cookie with security flags
    isProduction := h.authService.IsProduction()

    c.SetSameSite(http.SameSiteLaxMode) // CSRF protection
    c.SetCookie(
        "token",           // name
        token,             // value
        7*24*60*60,       // maxAge (7 days in seconds)
        "/",              // path
        "",               // domain (empty = current domain)
        isProduction,     // secure (HTTPS-only in production)
        true,             // httpOnly (JavaScript cannot access)
    )

    logger.Log.Info("User registered successfully",
        zap.String("user_id", user.ID.String()),
        zap.String("username", user.Username),
        zap.String("role", string(user.Role)),
    )

    // 4. Return success response (without token in body)
    c.JSON(http.StatusCreated, gin.H{
        "message": "User registered successfully",
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
        logger.Log.Warn("Login request parsing failed",
            zap.String("ip", c.ClientIP()),
            zap.Error(err),
        )
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid request body",
        })
        return
    }

    logger.Log.Info("User login attempt",
        zap.String("email", req.Email),
        zap.String("ip", c.ClientIP()),
    )

    // 2. Call service
    user, token, err := h.authService.Login(req.Email, req.Password)
    if err != nil {
        logger.Log.Warn("Login failed",
            zap.String("email", req.Email),
            zap.Error(err),
        )

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

    // 3. Set token in HTTP-only cookie with security flags
    isProduction := h.authService.IsProduction()

    c.SetSameSite(http.SameSiteLaxMode) // CSRF protection
    c.SetCookie(
        "token",           // name
        token,             // value
        7*24*60*60,       // maxAge (7 days in seconds)
        "/",              // path
        "",               // domain (empty = current domain)
        isProduction,     // secure (HTTPS-only in production)
        true,             // httpOnly (JavaScript cannot access)
    )

    logger.Log.Info("User logged in successfully",
        zap.String("user_id", user.ID.String()),
        zap.String("username", user.Username),
        zap.String("role", string(user.Role)),
    )

    // 4. Return success response (without token in body)
    c.JSON(http.StatusOK, gin.H{
        "message": "Login successful",
        "user": gin.H{
            "id":       user.ID,
            "username": user.Username,
            "email":    user.Email,
            "role":     user.Role,
        },
    })
}