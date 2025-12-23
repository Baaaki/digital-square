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
	"github.com/Baaaki/digital-square/pkg/logger"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger FIRST (before anything else)
	if err := logger.Init(true); err != nil { // true = development mode
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Log.Info("Starting Digital Square Backend")

	cfg := config.Load()
	logger.Log.Info("Config loaded successfully")

	database.Connect(cfg)
	database.Migrate()

	// Initialize WAL
	logger.Log.Info("Initializing WAL (Write-Ahead Log)")
	walInstance, err := wal.NewWAL("./data/wal.log")
	if err != nil {
		logger.Log.Fatal("Failed to initialize WAL", zap.Error(err))
	}
	defer walInstance.Close()
	logger.Log.Info("WAL initialized successfully")

	// Initialize Redis Broker (cache only for Phase 1-2)
	logger.Log.Info("Connecting to Redis")
	redisBroker, err := broker.NewRedisMessageBroker(cfg.RedisURL)
	if err != nil {
		logger.Log.Fatal("Failed to initialize Redis broker", zap.Error(err))
	}
	defer redisBroker.Close()
	logger.Log.Info("Redis connected successfully")

	// Rate limiter setup
	rateLimiterConfig := middleware.RateLimiterConfig{
		MaxRequests: cfg.RateLimitMaxRequests,
		Window:      cfg.RateLimitWindow,
		BlockTime:   cfg.RateLimitBlockTime,
	}
	rateLimiter := middleware.NewRateLimiter(redisBroker.GetClient(), rateLimiterConfig)
	logger.Log.Info("Rate limiter initialized",
		zap.Int("max_requests", cfg.RateLimitMaxRequests),
		zap.Duration("window", cfg.RateLimitWindow))

	// Initialize repositories
	userRepo := repository.NewUserRepository(database.DB)
	messageRepo := repository.NewMessageRepository(database.DB)

	// Initialize services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, 24*time.Hour, cfg.Environment)
	messageService := service.NewMessageService(messageRepo, redisBroker, walInstance)

	// Start batch writer (WAL → PostgreSQL every 1 minute)
	ctx := context.Background()
	messageService.StartBatchWriter(ctx)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	adminHandler := handler.NewAdminHandler(authService)
	messageHandler := handler.NewMessageHandler(messageService)
	wsHandler := handler.NewWebSocketHandler(messageService, cfg.JWTSecret)

	// Setup Gin router
	router := gin.Default()

	// Security Headers Middleware (MUST be first for all responses)
	router.Use(middleware.SecurityHeadersMiddleware())

	// HSTS Middleware (HTTPS enforcement in production)
	router.Use(middleware.HSTSMiddleware(cfg.Environment == "production"))

	// CORS configuration (allow cookies from frontend)
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001", "http://localhost:10000"}, // Frontend URL (3000, 3001, or 10000 for Docker)
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Cookie"},
		ExposeHeaders:    []string{"Set-Cookie"},
		AllowCredentials: true, // ✅ Cookie'lerin gönderilmesine izin ver
		MaxAge:           12 * time.Hour,
	}))

	// Rate limiting middleware (after CORS, before routes)
	router.Use(rateLimiter.Middleware())

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

	// Admin routes (require JWT + Admin role)
	admin := router.Group("/api/admin")
	admin.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	admin.Use(middleware.AdminMiddleware())
	{
		admin.GET("/users", adminHandler.GetAllUsers)
		admin.POST("/ban", adminHandler.BanUser)
		admin.POST("/ban-bulk", adminHandler.BanBulk)
	}

	// Start server
	logger.Log.Info("Server starting", zap.String("port", cfg.ServerPort))
	logger.Log.Info("Direct broadcast mode (single node)")
	if err := router.Run(cfg.ServerPort); err != nil {
		logger.Log.Fatal("Failed to start server", zap.Error(err))
	}
}