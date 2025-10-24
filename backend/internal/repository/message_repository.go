package repository

import (
    "time"

    "github.com/Baaaki/digital-square/internal/models"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

type MessageRepository struct {
    db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
    return &MessageRepository{db: db}
}

func (r *MessageRepository) CreateMessage(message *models.Message) error {
    return r.db.Create(message).Error
}

// GetMessageByID retrieves a message by ID
func (r *MessageRepository) GetMessageByID(id uint64) (*models.Message, error) {
    var message models.Message
    err := r.db.Preload("User").First(&message, id).Error
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, nil
        }
        return nil, err
    }
    return &message, nil
}

// GetMessagesBefore retrieves messages before a given ID (for infinite scroll)
func (r *MessageRepository) GetMessagesBefore(beforeID uint64, limit int) ([]models.Message, error) {
    var messages []models.Message
    err := r.db.
        Preload("User").
        Where("id < ?", beforeID).
        Order("created_at DESC").
        Limit(limit).
        Find(&messages).Error
    
    return messages, err
}

// GetRecentMessages retrieves the most recent messages
func (r *MessageRepository) GetRecentMessages(limit int) ([]models.Message, error) {
    var messages []models.Message
    err := r.db.
        Preload("User").
        Order("created_at DESC").
        Limit(limit).
        Find(&messages).Error
    
    return messages, err
}

// SoftDeleteMessage soft deletes a message (user delete)
func (r *MessageRepository) SoftDeleteMessage(messageID uint64, deletedBy uuid.UUID, isDeletedByAdmin bool) error {
    now := time.Now()
    return r.db.Model(&models.Message{}).
        Where("id = ?", messageID).
        Updates(map[string]interface{}{
            "deleted_at":          gorm.DeletedAt{Time: now, Valid: true}, // ✅ Set actual timestamp
            "deleted_by":          deletedBy,
            "is_deleted_by_admin": isDeletedByAdmin,
        }).Error
}

// BatchInsert bulk inserts messages (for WAL → PostgreSQL)
func (r *MessageRepository) BatchInsert(messages []models.Message) error {
    if len(messages) == 0 {
        return nil
    }
    return r.db.CreateInBatches(messages, 500).Error
}

func (r*MessageRepository) GetByMessageID (messageID string) (*models.Message, error) {
    var message models.Message
    err:= r.db.Where("message_id = ?", messageID).First(&message).Error
    if err != nil {
        return nil, err
    }
    return &message, nil
}