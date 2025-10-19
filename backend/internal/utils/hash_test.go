package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants
const (
	testPassword        = "SecurePassword123!"
	testWrongPassword   = "WrongPassword456!"
	testSpecialPassword = "P@ssw0rd!#$%"
)

func TestHashPassword_Success(t *testing.T) {
	// Arrange
	password := testPassword

	// Act
	hash, err := HashPassword(password)

	// Assert
	require.NoError(t, err, "HashPassword should not return error for valid password")
	assert.NotEmpty(t, hash, "Hash should not be empty")
	assert.NotEqual(t, password, hash, "Hash should be different from password")
	assert.Contains(t, hash, "$argon2id$", "Hash should contain Argon2id identifier")
}

func TestVerifyPassword_Correct(t *testing.T) {
	// Arrange
	password := testPassword
	hash, err := HashPassword(password)
	require.NoError(t, err, "Setup: HashPassword should not fail")

	// Act
	match, err := VerifyPassword(password, hash)

	// Assert
	require.NoError(t, err, "VerifyPassword should not return error")
	assert.True(t, match, "Password should match its hash")
}

func TestVerifyPassword_Incorrect(t *testing.T) {
	// Arrange
	password := testPassword
	wrongPassword := testWrongPassword
	hash, err := HashPassword(password)
	require.NoError(t, err, "Setup: HashPassword should not fail")

	// Act
	match, err := VerifyPassword(wrongPassword, hash)

	// Assert
	require.NoError(t, err, "VerifyPassword should not return error")
	assert.False(t, match, "Wrong password should not match hash")
}

func TestHashPassword_UniqueHashes(t *testing.T) {
	// Arrange
	password := testPassword

	// Act
	hash1, err1 := HashPassword(password)
	hash2, err2 := HashPassword(password)

	// Assert
	require.NoError(t, err1, "First HashPassword should not fail")
	require.NoError(t, err2, "Second HashPassword should not fail")
	assert.NotEqual(t, hash1, hash2, "Same password should produce different hashes due to unique salt")
}

func TestHashPassword_EmptyPassword(t *testing.T) {
	// Arrange
	password := ""

	// Act
	hash, err := HashPassword(password)

	// Assert
	require.NoError(t, err, "Argon2 should handle empty passwords")
	assert.NotEmpty(t, hash, "Hash should be generated even for empty password")
}

func TestHashPassword_VeryLongPassword(t *testing.T) {
	// Arrange
	// Test with 1000 character password
	password := strings.Repeat("a", 1000)

	// Act
	hash, err := HashPassword(password)

	// Assert
	require.NoError(t, err, "HashPassword should handle very long passwords")
	assert.NotEmpty(t, hash, "Hash should be generated for very long password")

	// Verify it can be verified
	match, err := VerifyPassword(password, hash)
	require.NoError(t, err)
	assert.True(t, match, "Very long password should match its hash")
}

func TestHashPassword_UnicodeCharacters(t *testing.T) {
	// Arrange
	unicodePasswords := []string{
		"パスワード123",           // Japanese
		"Şifre123!",             // Turkish
		"Пароль123",             // Russian
		"🔒🔑Password123",        // Emoji
		"Contraseña_ñ_ü_ç_ş",   // Mixed special chars
	}

	for _, password := range unicodePasswords {
		t.Run(password, func(t *testing.T) {
			// Act
			hash, err := HashPassword(password)

			// Assert
			require.NoError(t, err, "HashPassword should handle unicode characters")
			assert.NotEmpty(t, hash)

			// Verify
			match, err := VerifyPassword(password, hash)
			require.NoError(t, err)
			assert.True(t, match, "Unicode password should match its hash")
		})
	}
}

func TestVerifyPassword_InvalidHashFormat(t *testing.T) {
	// Arrange
	password := testPassword
	invalidHashes := []string{
		"",                                    // Empty
		"plain-text-not-hash",                 // Plain text
		"$invalid$format$",                    // Invalid format
		"$argon2id$v=19$m=65536",             // Incomplete
		"$argon2id$v=19$m=65536$corrupted",   // Corrupted
	}

	for _, invalidHash := range invalidHashes {
		t.Run(invalidHash, func(t *testing.T) {
			// Act
			match, err := VerifyPassword(password, invalidHash)

			// Assert
			assert.Error(t, err, "VerifyPassword should return error for invalid hash format")
			assert.False(t, match, "Match should be false for invalid hash")
		})
	}
}

func TestVerifyPassword_EmptyPassword(t *testing.T) {
	// Arrange
	password := ""
	hash, err := HashPassword(password)
	require.NoError(t, err, "Setup: HashPassword should not fail")

	// Act
	match, err := VerifyPassword(password, hash)

	// Assert
	require.NoError(t, err, "VerifyPassword should handle empty password")
	assert.True(t, match, "Empty password should match its hash")
}

func TestVerifyPassword_EmptyHash(t *testing.T) {
	// Arrange
	password := testPassword

	// Act
	match, err := VerifyPassword(password, "")

	// Assert
	assert.Error(t, err, "VerifyPassword should return error for empty hash")
	assert.False(t, match, "Match should be false for empty hash")
}

// Table-driven test for comprehensive coverage
func TestVerifyPassword_TableDriven(t *testing.T) {
	testCases := []struct {
		name        string
		password    string
		testPass    string
		expectMatch bool
		description string
	}{
		{
			name:        "correct_password",
			password:    testPassword,
			testPass:    testPassword,
			expectMatch: true,
			description: "Same password should match",
		},
		{
			name:        "incorrect_password",
			password:    testPassword,
			testPass:    testWrongPassword,
			expectMatch: false,
			description: "Different password should not match",
		},
		{
			name:        "empty_password",
			password:    "",
			testPass:    "",
			expectMatch: true,
			description: "Empty password should match itself",
		},
		{
			name:        "special_characters",
			password:    testSpecialPassword,
			testPass:    testSpecialPassword,
			expectMatch: true,
			description: "Special characters should be handled correctly",
		},
		{
			name:        "case_sensitive",
			password:    "Password123",
			testPass:    "password123",
			expectMatch: false,
			description: "Password verification should be case-sensitive",
		},
		{
			name:        "whitespace_matters",
			password:    "Password123 ",
			testPass:    "Password123",
			expectMatch: false,
			description: "Trailing whitespace should matter",
		},
		{
			name:        "unicode_password",
			password:    "Şifre123!",
			testPass:    "Şifre123!",
			expectMatch: true,
			description: "Unicode characters should work correctly",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			hash, err := HashPassword(tc.password)
			require.NoError(t, err, "Setup: HashPassword should not fail")

			// Act
			match, err := VerifyPassword(tc.testPass, hash)

			// Assert
			require.NoError(t, err, "VerifyPassword should not return error")
			assert.Equal(t, tc.expectMatch, match, tc.description)
		})
	}
}

// Benchmark tests
func BenchmarkHashPassword(b *testing.B) {
	password := testPassword
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = HashPassword(password)
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	password := testPassword
	hash, _ := HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = VerifyPassword(password, hash)
	}
}
