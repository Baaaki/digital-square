package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
	ServerPort  string
	Environment string
	JWTExpiry   time.Duration
	WALPath     string

	// Rate limiting
	RateLimitMaxRequests int
	RateLimitWindow      time.Duration
	RateLimitBlockTime   time.Duration
}

func Load() *Config {
	// Try to load .env file, but don't fail if it doesn't exist
	// (Docker containers use environment variables directly)
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	expiryStr := os.Getenv("JWT_EXPIRY")
	expiry, err := time.ParseDuration(expiryStr)
	if err != nil {
		log.Fatal("Invalid JWT_EXPIRY format")
	}

	walPath := os.Getenv("WAL_PATH")
	if walPath == "" {
		walPath = "data/wal_messages"
	}

	// Rate limiting defaults
	rateLimitMax := getEnvAsInt("RATE_LIMIT_MAX_REQUESTS", 100)
	rateLimitWindow := getEnvAsDuration("RATE_LIMIT_WINDOW", "1m")
	rateLimitBlock := getEnvAsDuration("RATE_LIMIT_BLOCK_TIME", "5m")

	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		ServerPort:  os.Getenv("SERVER_PORT"),
		Environment: os.Getenv("ENVIRONMENT"),
		JWTExpiry:   expiry,
		WALPath:     walPath,

		RateLimitMaxRequests: rateLimitMax,
		RateLimitWindow:      rateLimitWindow,
		RateLimitBlockTime:   rateLimitBlock,
	}

	return cfg
}

// getEnvAsInt retrieves environment variable as int with default value
func getEnvAsInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		log.Printf("Invalid %s value, using default: %d", key, defaultVal)
		return defaultVal
	}
	return val
}

// getEnvAsDuration retrieves environment variable as duration with default value
func getEnvAsDuration(key string, defaultVal string) time.Duration {
	valStr := os.Getenv(key)
	if valStr == "" {
		valStr = defaultVal
	}
	duration, err := time.ParseDuration(valStr)
	if err != nil {
		log.Printf("Invalid %s value, using default: %s", key, defaultVal)
		duration, _ = time.ParseDuration(defaultVal)
	}
	return duration
}
