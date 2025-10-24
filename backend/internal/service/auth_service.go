package service

import (
	"errors"
	"regexp"
	"time"

	"github.com/Baaaki/digital-square/internal/models"
	"github.com/Baaaki/digital-square/internal/repository"
	"github.com/Baaaki/digital-square/internal/utils"
	"github.com/Baaaki/digital-square/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
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
	environment   string
}

func NewAuthService(userRepo *repository.UserRepository, jwtSecret string, jwtExpiration time.Duration, environment string) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		jwtSecret:     jwtSecret,
		jwtExpiration: jwtExpiration,
		environment:   environment,
	}
}

// IsProduction returns true if running in production environment
func (s *AuthService) IsProduction() bool {
	return s.environment == "production"
}

func (s *AuthService) Register(username, email, password string) (*models.User, string, error) {
	start := time.Now()

	logger.Log.Debug("Processing user registration",
		zap.String("username", username),
		zap.String("email", email),
	)

	// 1. Validate input
	if err := s.validateRegisterInput(username, email, password); err != nil {
		logger.Log.Warn("Registration validation failed",
			zap.String("username", username),
			zap.String("email", email),
			zap.Error(err),
		)
		return nil, "", err
	}

	// 2. Check if email already exists
	existingUser, err := s.userRepo.GetUserByEmail(email)
	if err != nil {
		logger.Log.Error("Failed to check email existence",
			zap.String("email", email),
			zap.Error(err),
		)
		return nil, "", err
	}
	if existingUser != nil {
		logger.Log.Warn("Email already exists",
			zap.String("email", email),
		)
		return nil, "", ErrEmailAlreadyExists
	}

	// 3. Check if username already exists
	existingUser, err = s.userRepo.GetUserByUsername(username)
	if err != nil {
		logger.Log.Error("Failed to check username existence",
			zap.String("username", username),
			zap.Error(err),
		)
		return nil, "", err
	}
	if existingUser != nil {
		logger.Log.Warn("Username already exists",
			zap.String("username", username),
		)
		return nil, "", ErrUsernameAlreadyExists
	}

	// 4. Hash password (Argon2)
	hashStart := time.Now()
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		logger.Log.Error("Failed to hash password",
			zap.Error(err),
		)
		return nil, "", err
	}
	hashDuration := time.Since(hashStart)

	logger.Log.Debug("Password hashed successfully",
		zap.Duration("hash_duration", hashDuration),
	)

	// 5. Create user
	user := &models.User{
		ID:           uuid.New(),
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         models.RoleUser, // Default role
	}

	if err := s.userRepo.CreateUser(user); err != nil {
		logger.Log.Error("Failed to create user in database",
			zap.String("username", username),
			zap.String("email", email),
			zap.Error(err),
		)
		return nil, "", err
	}

	// 6. Generate JWT token
	token, err := utils.GenerateToken(user, s.jwtSecret, s.jwtExpiration)
	if err != nil {
		logger.Log.Error("Failed to generate JWT token",
			zap.String("user_id", user.ID.String()),
			zap.Error(err),
		)
		return nil, "", err
	}

	logger.Log.Info("User registered successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("username", username),
		zap.String("email", email),
		zap.Duration("hash_duration", hashDuration),
		zap.Duration("total_duration", time.Since(start)),
	)

	return user, token, nil
}

func (s *AuthService) Login(email, password string) (*models.User, string, error) {
	start := time.Now()

	logger.Log.Debug("Processing user login",
		zap.String("email", email),
	)

	// 1. Get user by email
	user, err := s.userRepo.GetUserByEmail(email)
	if err != nil {
		logger.Log.Error("Failed to get user by email",
			zap.String("email", email),
			zap.Error(err),
		)
		return nil, "", err
	}
	if user == nil {
		logger.Log.Warn("Login failed: user not found",
			zap.String("email", email),
		)
		return nil, "", ErrInvalidCredentials
	}

	// 2. Verify password
	verifyStart := time.Now()
	valid, err := utils.VerifyPassword(password, user.PasswordHash)
	if err != nil {
		logger.Log.Error("Failed to verify password",
			zap.String("email", email),
			zap.Error(err),
		)
		return nil, "", err
	}
	verifyDuration := time.Since(verifyStart)

	if !valid {
		logger.Log.Warn("Login failed: invalid password",
			zap.String("email", email),
			zap.String("user_id", user.ID.String()),
		)
		return nil, "", ErrInvalidCredentials
	}

	// 3. Generate JWT token
	token, err := utils.GenerateToken(user, s.jwtSecret, s.jwtExpiration)
	if err != nil {
		logger.Log.Error("Failed to generate JWT token",
			zap.String("user_id", user.ID.String()),
			zap.Error(err),
		)
		return nil, "", err
	}

	logger.Log.Info("User logged in successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("username", user.Username),
		zap.Duration("password_verify_duration", verifyDuration),
		zap.Duration("total_duration", time.Since(start)),
	)

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

// GetAllUsers returns all users (including soft-deleted ones)
func (s *AuthService) GetAllUsers() ([]*models.User, error) {
	logger.Log.Debug("Fetching all users (including deleted)")

	users, err := s.userRepo.GetAllUsers()
	if err != nil {
		logger.Log.Error("Failed to fetch all users",
			zap.Error(err),
		)
		return nil, err
	}

	logger.Log.Info("Fetched all users",
		zap.Int("count", len(users)),
	)

	return users, nil
}

// BanUser soft deletes a user (sets DeletedAt)
func (s *AuthService) BanUser(userID, adminID, reason string) error {
	logger.Log.Info("Banning user",
		zap.String("user_id", userID),
		zap.String("admin_id", adminID),
		zap.String("reason", reason),
	)

	// Parse UUID
	uid, err := uuid.Parse(userID)
	if err != nil {
		logger.Log.Warn("Invalid user ID format",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return errors.New("invalid user ID format")
	}

	// Soft delete user
	if err := s.userRepo.SoftDeleteUser(uid); err != nil {
		logger.Log.Error("Failed to ban user",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return err
	}

	// TODO: Delete all user's messages (will be implemented in message service)

	logger.Log.Info("User banned successfully",
		zap.String("user_id", userID),
		zap.String("admin_id", adminID),
	)

	return nil
}

// BanBulk bans multiple users at once
func (s *AuthService) BanBulk(userIDs []string, adminID, reason string) error {
	logger.Log.Info("Bulk banning users",
		zap.Int("count", len(userIDs)),
		zap.String("admin_id", adminID),
		zap.String("reason", reason),
	)

	var uuids []uuid.UUID
	for _, id := range userIDs {
		uid, err := uuid.Parse(id)
		if err != nil {
			logger.Log.Warn("Invalid user ID in bulk ban",
				zap.String("user_id", id),
				zap.Error(err),
			)
			continue // Skip invalid IDs
		}
		uuids = append(uuids, uid)
	}

	if len(uuids) == 0 {
		return errors.New("no valid user IDs provided")
	}

	// Bulk soft delete
	if err := s.userRepo.BulkSoftDelete(uuids); err != nil {
		logger.Log.Error("Failed to bulk ban users",
			zap.Error(err),
		)
		return err
	}

	// TODO: Delete all messages from these users

	logger.Log.Info("Users banned successfully",
		zap.Int("count", len(uuids)),
	)

	return nil
}
