package repository

import (
	"errors"

	"github.com/Baaaki/digital-square/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	// Note: GORM automatically excludes soft-deleted users (deleted_at IS NOT NULL)
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	
	return &user, nil
}

func (r *UserRepository) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	
	return &user, nil
}

func (r *UserRepository) GetUserByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.Where("id = ?", id).First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// GetAllUsers returns all users including soft-deleted ones
func (r *UserRepository) GetAllUsers() ([]*models.User, error) {
	var users []*models.User
	// Unscoped() includes soft-deleted records (deleted_at IS NOT NULL)
	err := r.db.Unscoped().Order("created_at DESC").Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

// SoftDeleteUser marks a user as deleted (sets DeletedAt)
func (r *UserRepository) SoftDeleteUser(id uuid.UUID) error {
	return r.db.Delete(&models.User{}, id).Error
}

// BulkSoftDelete marks multiple users as deleted
func (r *UserRepository) BulkSoftDelete(ids []uuid.UUID) error {
	return r.db.Delete(&models.User{}, ids).Error
}