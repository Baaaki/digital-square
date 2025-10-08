package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Username     string    `gorm:"type:varchar(50);uniqueIndex;not null"`
	Email        string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	PasswordHash string    `gorm:"type:varchar(255);not null"`
	Role         Role      `gorm:"type:varchar(20);not null;default:'user'"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}