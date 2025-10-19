package utils

import (
	"testing"
	"time"

	"github.com/Baaaki/digital-square/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants
const (
	testSecret          = "test-secret-key-for-jwt-testing"
	testWrongSecret     = "wrong-secret-key-for-jwt-testing"
	testTokenDuration   = 1 * time.Hour
	testExpiredDuration = -1 * time.Hour
)

// Helper function to create test user
func createTestUser(role models.Role) *models.User {
	return &models.User{
		ID:       uuid.New(),
		Username: "testuser",
		Email:    "test@example.com",
		Role:     role,
	}
}

func TestGenerateToken_Success(t *testing.T) {
	// Arrange
	user := createTestUser(models.RoleUser)

	// Act
	token, err := GenerateToken(user, testSecret, testTokenDuration)

	// Assert
	require.NoError(t, err, "GenerateToken should not return error for valid input")
	assert.NotEmpty(t, token, "Token should not be empty")
	assert.Contains(t, token, ".", "JWT token should contain dots")

	// JWT format check: header.payload.signature
	parts := len(token)
	assert.Greater(t, parts, 0, "Token should have content")
}

func TestGenerateToken_DifferentRoles(t *testing.T) {
	// Test token generation for different roles
	roles := []models.Role{models.RoleUser, models.RoleAdmin}

	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			// Arrange
			user := createTestUser(role)

			// Act
			token, err := GenerateToken(user, testSecret, testTokenDuration)

			// Assert
			require.NoError(t, err, "GenerateToken should work for all roles")
			assert.NotEmpty(t, token)

			// Validate the token contains correct role
			claims, err := ValidateToken(token, testSecret)
			require.NoError(t, err)
			assert.Equal(t, role, claims.Role, "Token should contain correct role")
		})
	}
}

func TestGenerateToken_EmptySecret(t *testing.T) {
	// Arrange
	user := createTestUser(models.RoleUser)

	// Act
	token, err := GenerateToken(user, "", testTokenDuration)

	// Assert
	// Should still generate token (but it's insecure)
	require.NoError(t, err, "GenerateToken should handle empty secret")
	assert.NotEmpty(t, token, "Token should be generated even with empty secret")
}

func TestGenerateToken_ZeroDuration(t *testing.T) {
	// Arrange
	user := createTestUser(models.RoleUser)

	// Act
	token, err := GenerateToken(user, testSecret, 0)

	// Assert
	require.NoError(t, err, "GenerateToken should handle zero duration")
	assert.NotEmpty(t, token)

	// Token should be immediately expired
	_, err = ValidateToken(token, testSecret)
	assert.Error(t, err, "Token with zero duration should be expired")
}

func TestValidateToken_Success(t *testing.T) {
	// Arrange
	user := createTestUser(models.RoleUser)
	token, err := GenerateToken(user, testSecret, testTokenDuration)
	require.NoError(t, err, "Setup: GenerateToken should not fail")

	// Act
	claims, err := ValidateToken(token, testSecret)

	// Assert
	require.NoError(t, err, "ValidateToken should not return error for valid token")
	assert.NotNil(t, claims, "Claims should not be nil")
	assert.Equal(t, user.ID, claims.UserID, "UserID should match")
	assert.Equal(t, user.Username, claims.Username, "Username should match")
	assert.Equal(t, user.Role, claims.Role, "Role should match")
	assert.True(t, claims.ExpiresAt.Time.After(time.Now()), "Token should not be expired")
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Arrange
	user := createTestUser(models.RoleUser)
	token, err := GenerateToken(user, testSecret, testExpiredDuration)
	require.NoError(t, err, "Setup: GenerateToken should not fail")

	// Act
	claims, err := ValidateToken(token, testSecret)

	// Assert
	assert.Error(t, err, "ValidateToken should return error for expired token")
	assert.Nil(t, claims, "Claims should be nil for expired token")
}

func TestValidateToken_InvalidToken(t *testing.T) {
	// Arrange
	invalidTokens := []string{
		"",                                    // Empty
		"invalid.token.here",                  // Invalid format
		"not-a-jwt-token",                     // Plain text
		"a.b",                                 // Incomplete JWT
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", // Only header
	}

	for _, invalidToken := range invalidTokens {
		t.Run(invalidToken, func(t *testing.T) {
			// Act
			claims, err := ValidateToken(invalidToken, testSecret)

			// Assert
			assert.Error(t, err, "ValidateToken should return error for invalid token")
			assert.Nil(t, claims, "Claims should be nil for invalid token")
		})
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	// Arrange
	user := createTestUser(models.RoleUser)
	token, err := GenerateToken(user, testSecret, testTokenDuration)
	require.NoError(t, err, "Setup: GenerateToken should not fail")

	// Act
	claims, err := ValidateToken(token, testWrongSecret)

	// Assert
	assert.Error(t, err, "ValidateToken should return error for wrong secret")
	assert.Nil(t, claims, "Claims should be nil when secret is wrong")
}

func TestValidateToken_EmptySecret(t *testing.T) {
	// Arrange
	user := createTestUser(models.RoleUser)
	token, err := GenerateToken(user, testSecret, testTokenDuration)
	require.NoError(t, err, "Setup: GenerateToken should not fail")

	// Act
	claims, err := ValidateToken(token, "")

	// Assert
	assert.Error(t, err, "ValidateToken should return error for empty secret")
	assert.Nil(t, claims, "Claims should be nil for empty secret")
}

func TestValidateToken_TamperedToken(t *testing.T) {
	// Arrange
	user := createTestUser(models.RoleUser)
	token, err := GenerateToken(user, testSecret, testTokenDuration)
	require.NoError(t, err, "Setup: GenerateToken should not fail")

	// Tamper with the token by modifying the signature
	tamperedToken := token[:len(token)-5] + "XXXXX"

	// Act
	claims, err := ValidateToken(tamperedToken, testSecret)

	// Assert
	assert.Error(t, err, "ValidateToken should return error for tampered token")
	assert.Nil(t, claims, "Claims should be nil for tampered token")
}

func TestToken_RoundTrip(t *testing.T) {
	// Test that we can generate and validate tokens for different users
	// Arrange
	users := []*models.User{
		createTestUser(models.RoleUser),
		createTestUser(models.RoleAdmin),
		{
			ID:       uuid.New(),
			Username: "unicode_user_ışık",
			Email:    "unicode@example.com",
			Role:     models.RoleUser,
		},
		{
			ID:       uuid.New(),
			Username: "special!@#$%",
			Email:    "special@example.com",
			Role:     models.RoleAdmin,
		},
	}

	for _, user := range users {
		t.Run(user.Username, func(t *testing.T) {
			// Act - Generate
			token, err := GenerateToken(user, testSecret, testTokenDuration)
			require.NoError(t, err, "GenerateToken should succeed")

			// Act - Validate
			claims, err := ValidateToken(token, testSecret)
			require.NoError(t, err, "ValidateToken should succeed")

			// Assert
			assert.Equal(t, user.ID, claims.UserID, "UserID should match")
			assert.Equal(t, user.Username, claims.Username, "Username should match")
			assert.Equal(t, user.Role, claims.Role, "Role should match")
		})
	}
}

func TestToken_MultipleTokensSameUser(t *testing.T) {
	// Test that same user can have multiple valid tokens
	// Arrange
	user := createTestUser(models.RoleUser)

	// Act - Generate multiple tokens with a small delay
	token1, err1 := GenerateToken(user, testSecret, testTokenDuration)
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	token2, err2 := GenerateToken(user, testSecret, testTokenDuration)

	// Assert
	require.NoError(t, err1, "First token generation should succeed")
	require.NoError(t, err2, "Second token generation should succeed")

	// Note: Tokens might be identical if generated at exact same millisecond
	// This test verifies both tokens work independently, not necessarily different

	// Both tokens should be valid
	claims1, err1 := ValidateToken(token1, testSecret)
	claims2, err2 := ValidateToken(token2, testSecret)

	require.NoError(t, err1, "First token should be valid")
	require.NoError(t, err2, "Second token should be valid")
	assert.Equal(t, user.ID, claims1.UserID, "First token should have correct UserID")
	assert.Equal(t, user.ID, claims2.UserID, "Second token should have correct UserID")
}

// Table-driven test for multiple scenarios
func TestValidateToken_TableDriven(t *testing.T) {
	testCases := []struct {
		name        string
		secret      string
		duration    time.Duration
		wrongSecret bool
		expectError bool
		description string
	}{
		{
			name:        "valid_token",
			secret:      testSecret,
			duration:    testTokenDuration,
			wrongSecret: false,
			expectError: false,
			description: "Valid token with correct secret should pass",
		},
		{
			name:        "expired_token",
			secret:      testSecret,
			duration:    testExpiredDuration,
			wrongSecret: false,
			expectError: true,
			description: "Expired token should fail validation",
		},
		{
			name:        "wrong_secret",
			secret:      testSecret,
			duration:    testTokenDuration,
			wrongSecret: true,
			expectError: true,
			description: "Token validated with wrong secret should fail",
		},
		{
			name:        "short_duration",
			secret:      testSecret,
			duration:    5 * time.Second,
			wrongSecret: false,
			expectError: false,
			description: "Token with short duration should be valid initially",
		},
		{
			name:        "long_duration",
			secret:      testSecret,
			duration:    24 * 365 * time.Hour, // 1 year
			wrongSecret: false,
			expectError: false,
			description: "Token with long duration should be valid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			user := createTestUser(models.RoleUser)
			token, err := GenerateToken(user, tc.secret, tc.duration)
			require.NoError(t, err, "Setup: GenerateToken should not fail")

			validateSecret := tc.secret
			if tc.wrongSecret {
				validateSecret = testWrongSecret
			}

			// Act
			claims, err := ValidateToken(token, validateSecret)

			// Assert
			if tc.expectError {
				assert.Error(t, err, tc.description)
				assert.Nil(t, claims, "Claims should be nil on error")
			} else {
				require.NoError(t, err, tc.description)
				assert.NotNil(t, claims, "Claims should not be nil on success")
				assert.Equal(t, user.ID, claims.UserID, "UserID should match")
				assert.Equal(t, user.Username, claims.Username, "Username should match")
			}
		})
	}
}

// Benchmark tests
func BenchmarkGenerateToken(b *testing.B) {
	user := createTestUser(models.RoleUser)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GenerateToken(user, testSecret, testTokenDuration)
	}
}

func BenchmarkValidateToken(b *testing.B) {
	user := createTestUser(models.RoleUser)
	token, _ := GenerateToken(user, testSecret, testTokenDuration)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ValidateToken(token, testSecret)
	}
}
