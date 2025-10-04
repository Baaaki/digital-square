package database

import (
	"github.com/Baaaki/digital-square/internal/config"
    "github.com/Baaaki/digital-square/internal/models" 
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect(cfg *config.Config){
	var err error

	DB, err = gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})

	if err!= nil {
		log.Fatal("Failed to connect database:", err)
	}
	
	log.Println("Database connect successfully")
}

func Migrate(){
	err := DB.AutoMigrate(&models.User{}, &models.Message{})

	if err != nil {
		log.Fatal("Migration failed:", err)
	}

	log.Println("Database migration completed")
}