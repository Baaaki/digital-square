package main

import (
	"log"

	"github.com/Baaaki/digital-square/internal/config"
	"github.com/Baaaki/digital-square/internal/database"
)

func main() {
	cfg := config.Load()
	log.Println("Config loaded successfully")

	database.Connect(cfg)

	database.Migrate()

	log.Println("Server started successfully!")
}