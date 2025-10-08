package config

import (
	"log"
	"os"
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
}

func Load() *Config {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
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

	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		ServerPort:  os.Getenv("SERVER_PORT"),
		Environment: os.Getenv("ENVIRONMENT"),
		JWTExpiry:   expiry,
		WALPath:     walPath,
	}

	return cfg
}
