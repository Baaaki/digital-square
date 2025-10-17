package main

import (
	"log"
	"os"

	"github.com/Baaaki/digital-square/internal/config"
	"github.com/Baaaki/digital-square/internal/database"
	"github.com/Baaaki/digital-square/internal/models"
	"github.com/Baaaki/digital-square/internal/utils"
	"github.com/google/uuid"
)

func main() {
	cfg := config.Load()
	database.Connect(cfg)

	// Get admin email from env
	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPassword := os.Getenv("ADMIN_PASSWORD")

	if adminUsername == "" || adminEmail == "" || adminPassword == "" {
		log.Fatal("Missing enviroment variables: ADMIN_USERNAME, ADMIN_EMAIL, ADMIN_PASSWORD")
	}

	// Check if admin with this email already exists
	var admin models.User
	result := database.DB.Where("email = ?", adminEmail).First(&admin)

	if result.Error == nil {
		log.Println("✅ Admin user already exists:", admin.Username)
		log.Println("   Email:", admin.Email)
		return
	}

	// Hash password using utils.HashPassword (Argon2id)
	passwordHash, err := utils.HashPassword(adminPassword)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	// Create new admin user
	admin = models.User{
		ID:           uuid.New(),
		Username:     adminUsername,
		Email:        adminEmail,
		PasswordHash: passwordHash,
		Role:         models.RoleAdmin,
	}

	if err := database.DB.Create(&admin).Error; err != nil {
		log.Fatal("Failed to create admin:", err)
	}

	log.Println("✅ Admin user created successfully!")
	log.Println("   Username:", admin.Username)
	log.Println("   Email:", admin.Email)
}
