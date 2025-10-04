package models

import (
	"time"
	"github.com/google/uuid"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type User struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Username string `gorm:"type:varchar(50);unique;not null"`
	Email string `gorm:"type:varchar(100);unique;not null"`
	Password string `gorm:"type:varchar(255);not null"`
	Role Role `gorm:"type:varchar(20);not null;default:'user'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}