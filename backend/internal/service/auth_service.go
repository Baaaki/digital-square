package service

import (
	"errors"
	"regexp"
	"time"

	"github.com/Baaaki/digital-square/internal/models"
	"github.com/Baaaki/digital-square/internal/repository"
	"github.com/Baaaki/digital-square/internal/utils"
	"github.com/google/uuid"
)

var (
	ErrEmailAlreadyExists    = errors.New("email already exists")
	ErrUsernameAlreadyExists = errors.New("username already exists")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

type AuthService struct {
	userRepo      *repository.UserRepository
	jwtSecret     string
	jwtExpiration time.Duration
}

func NewAuthService(userRepo *repository.UserRepository, jwtSecret string, jwtExpiration time.Duration) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		jwtSecret:     jwtSecret,
		jwtExpiration: jwtExpiration,
	}
}

func (s *AuthService) Register(username, email, password string) (*models.User, string, error) {
	// 1. Validate input
	if err := s.validateRegisterInput(username, email, password); err != nil {
		return nil, "", err
	}

	// 2. Check if email already exists
	existingUser, err := s.userRepo.GetUserByEmail(email)
	if err != nil {
		return nil, "", err
	}
	if existingUser != nil {
		return nil, "", ErrEmailAlreadyExists
	}

	// 3. Check if username already exists
	existingUser, err = s.userRepo.GetUserByUsername(username)
	if err != nil {
		return nil, "", err
	}
	if existingUser != nil {
		return nil, "", ErrUsernameAlreadyExists
	}

	// 4. Hash password (Argon2)
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, "", err
	}

	// 5. Create user
	user := &models.User{
		ID:           uuid.New(),
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         models.RoleUser, // Default role
	}

	if err := s.userRepo.CreateUser(user); err != nil {
		return nil, "", err
	}

	// 6. Generate JWT token
	token, err := utils.GenerateToken(user, s.jwtSecret, s.jwtExpiration)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *AuthService) Login(email, password string) (*models.User, string, error) {
	// 1. Get user by email
	user, err := s.userRepo.GetUserByEmail(email)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		return nil, "", ErrInvalidCredentials
	}

	// 2. Verify password
	valid, err := utils.VerifyPassword(password, user.PasswordHash)
	if err != nil {
		return nil, "", err
	}
	if !valid {
		return nil, "", ErrInvalidCredentials
	}

	// 3. Generate JWT token
	token, err := utils.GenerateToken(user, s.jwtSecret, s.jwtExpiration)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *AuthService) validateRegisterInput(username, email, password string) error {
    // Username validation
    if len(username) < 3 {
        return errors.New("username must be at least 3 characters")
    }
    if len(username) > 50 {
        return errors.New("username must be at most 50 characters")
    }
    
    // Email validation (regex)
    if !emailRegex.MatchString(email) {
        return errors.New("invalid email format")
    }
    if len(email) > 100 {
        return errors.New("email too long")
    }
    
    // Password validation
    if len(password) < 8 {
        return errors.New("password must be at least 8 characters")
    }
    if len(password) > 128 {
        return errors.New("password too long")
    }
    
    return nil
}
