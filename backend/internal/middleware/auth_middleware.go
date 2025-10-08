package middleware

import (
    "net/http"
    "strings"

    "github.com/Baaaki/digital-square/internal/utils"
    "github.com/gin-gonic/gin"
)

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Get Authorization header
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "Authorization header required",
            })
            c.Abort()
            return
        }
        
        // 2. Extract token from "Bearer <token>"
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenString == authHeader {
            // "Bearer " prefix yoksa
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "Invalid authorization format. Use: Bearer <token>",
            })
            c.Abort()
            return
        }
        
        // 3. Validate token
        claims, err := utils.ValidateToken(tokenString, jwtSecret)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "Invalid or expired token",
            })
            c.Abort()
            return
        }
        
        // 4. Add claims to context (handlers can access)
        c.Set("user_id", claims.UserID)
        c.Set("user_email", claims.Email)
        c.Set("user_role", claims.Role)
        c.Set("claims", claims)
        
        // 5. Continue to handler
        c.Next()
    }
}

func AdminMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get role from context (set by AuthMiddleware)
        role, exists := c.Get("user_role")
        if !exists {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "Unauthorized",
            })
            c.Abort()
            return
        }
        
        // Check if admin
        if role != "admin" {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "Admin access required",
            })
            c.Abort()
            return
        }
        
        // Continue to handler
        c.Next()
    }
}