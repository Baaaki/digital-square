package main

import (
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

	// Initialize Redis Broker
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

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	wsHandler := handler.NewWebSocketHandler(messageService, redisBroker, cfg.JWTSecret)

	// Setup Gin router
	router := gin.Default()

	// Public routes
	router.POST("/api/auth/register", authHandler.Register)
	router.POST("/api/auth/login", authHandler.Login)

	// Protected routes (require JWT)
	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		// WebSocket connection
		protected.GET("/ws", wsHandler.HandleWebSocket)
	}

	// Start server
	log.Printf("Server starting on %s", cfg.ServerPort)
	log.Println("Broadcast listener started")
	if err := router.Run(cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
