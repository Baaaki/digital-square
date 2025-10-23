package main

import (
	"context"
	"log"
	"time"

	"github.com/Baaaki/digital-square/internal/broker"
	"github.com/Baaaki/digital-square/internal/config"
	"github.com/Baaaki/digital-square/internal/database"
	"github.com/Baaaki/digital-square/internal/handler"
	"github.com/Baaaki/digital-square/internal/middleware"
	"github.com/Baaaki/digital-square/internal/repository"
	"github.com/Baaaki/digital-square/internal/service"
	"github.com/Baaaki/digital-square/internal/wal"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	log.Println("Config loaded successfully")

	database.Connect(cfg)
	database.Migrate()

	// Initialize WAL
	walInstance, err := wal.NewWAL("./data/wal.log")
	if err != nil {
		log.Fatalf("Failed to initialize WAL: %v", err)
	}
	defer walInstance.Close()

	// Initialize Redis Broker (cache only for Phase 1-2)
	redisBroker, err := broker.NewRedisMessageBroker(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to initialize Redis broker: %v", err)
	}
	defer redisBroker.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepository(database.DB)
	messageRepo := repository.NewMessageRepository(database.DB)

	// Initialize services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, 24*time.Hour)
	messageService := service.NewMessageService(messageRepo, redisBroker, walInstance)

	// Start batch writer (WAL → PostgreSQL every 1 minute)
	ctx := context.Background()
	messageService.StartBatchWriter(ctx)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	messageHandler := handler.NewMessageHandler(messageService)
	wsHandler := handler.NewWebSocketHandler(messageService, cfg.JWTSecret)

	// Setup Gin router
	router := gin.Default()

	// CORS configuration (allow cookies from frontend)
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001"}, // Frontend URL (3000 or 3001)
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Cookie"},
		ExposeHeaders:    []string{"Set-Cookie"},
		AllowCredentials: true, // ✅ Cookie'lerin gönderilmesine izin ver
		MaxAge:           12 * time.Hour,
	}))

	// Public routes
	router.POST("/api/auth/register", authHandler.Register)
	router.POST("/api/auth/login", authHandler.Login)

	// Protected routes (require JWT)
	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		// WebSocket connection
		protected.GET("/ws", wsHandler.HandleWebSocket)
		
		// Message endpoints
		protected.GET("/messages/before/:id", messageHandler.GetBefore)
	}

	// Start server
	log.Printf("Server starting on %s", cfg.ServerPort)
	log.Println("Direct broadcast mode (single node)")
	if err := router.Run(cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}