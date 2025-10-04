package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Content   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"index"`

	//Soft Delete Fields
	IsDeleted        bool `gorm:"default:false;index"`
	IsDeletedByAdmin bool `gorm:"default:false"`
	DeletedAt        *time.Time
	DeletedBy        *uuid.UUID `gorm:"type:uuid"`

	//Foreign Key Relationship
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}
