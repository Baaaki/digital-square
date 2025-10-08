package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
    ID                uint64         `gorm:"primaryKey;autoIncrement"`
    MessageID         string         `gorm:"type:varchar(50);uniqueIndex;not null"`
    UserID            uuid.UUID      `gorm:"type:uuid;not null;index"`
	Content           string         `gorm:"type:text;not null"`
    CreatedAt         time.Time      `gorm:"index:idx_created_time"` 
    
	DeletedAt         gorm.DeletedAt `gorm:"index"`
    DeletedBy         *uuid.UUID     `gorm:"type:uuid;index"`
    IsDeletedByAdmin  bool           `gorm:"default:false"`

	User              User           `gorm:"foreignKey:UserID;references:ID"`
}

