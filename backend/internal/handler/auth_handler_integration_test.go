package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Baaaki/digital-square/internal/handler"
	"github.com/Baaaki/digital-square/internal/models"
	"github.com/Baaaki/digital-square/internal/repository"
	"github.com/Baaaki/digital-square/internal/service"
	"github.com/Baaaki/digital-square/internal/testutil"
	"github.com/Baaaki/digital-square/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// AuthHandlerIntegrationTestSuite defines test suite
type AuthHandlerIntegrationTestSuite struct {
	suite.Suite
	testDB      *testutil.TestDatabase
	authHandler *handler.AuthHandler
	router      *gin.Engine
}

// SetupSuite runs before all tests
func (s *AuthHandlerIntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Initialize logger (required for handlers)
	logger.Init(false) // false = production mode (no verbose logs)

	// Start in-memory SQLite test database (migrations run automatically)
	s.testDB = testutil.SetupTestDatabase(s.T())

	// Setup repositories and services
	userRepo := repository.NewUserRepository(s.testDB.DB)
	authService := service.NewAuthService(userRepo, "test-secret-key", 1*time.Hour, "development")

	// Setup handler
	s.authHandler = handler.NewAuthHandler(authService)

	// Setup router
	s.router = gin.New()
	s.router.POST("/api/auth/register", s.authHandler.Register)
	s.router.POST("/api/auth/login", s.authHandler.Login)
}

// TearDownSuite runs after all tests
func (s *AuthHandlerIntegrationTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

// SetupTest runs before each test (clean database)
func (s *AuthHandlerIntegrationTestSuite) SetupTest() {
	testutil.CleanDatabase(s.T(), s.testDB.DB)
}

// TestRegisterSuccess tests successful user registration
func (s *AuthHandlerIntegrationTestSuite) TestRegisterSuccess() {
	// Prepare request
	reqBody := map[string]string{
		"username": "newuser",
		"email":    "newuser@example.com",
		"password": "SecurePass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Make request
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(s.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "User registered successfully", response["message"])

	// Check user data
	user := response["user"].(map[string]interface{})
	assert.Equal(s.T(), "newuser", user["username"])
	assert.Equal(s.T(), "newuser@example.com", user["email"])
	assert.Equal(s.T(), "user", user["role"])

	// Check cookie
	cookies := w.Result().Cookies()
	assert.NotEmpty(s.T(), cookies)
	var tokenCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "token" {
			tokenCookie = cookie
			break
		}
	}
	assert.NotNil(s.T(), tokenCookie)
	assert.True(s.T(), tokenCookie.HttpOnly)
	assert.Equal(s.T(), http.SameSiteLaxMode, tokenCookie.SameSite)
}

// TestRegisterDuplicateEmail tests registration with existing email
func (s *AuthHandlerIntegrationTestSuite) TestRegisterDuplicateEmail() {
	// Create existing user
	existingUser, _ := testutil.CreateTestUser("existing", "test@example.com", "Pass123", models.RoleUser)
	s.testDB.DB.Create(existingUser)

	// Try to register with same email
	reqBody := map[string]string{
		"username": "different",
		"email":    "test@example.com", // Same email
		"password": "SecurePass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(s.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(s.T(), response["error"], "email already exists")
}

// TestRegisterInvalidInput tests registration with invalid input
func (s *AuthHandlerIntegrationTestSuite) TestRegisterInvalidInput() {
	testCases := []struct {
		name     string
		reqBody  map[string]string
		expected string
	}{
		{
			name: "Short username",
			reqBody: map[string]string{
				"username": "ab",
				"email":    "test@example.com",
				"password": "Pass123456",
			},
			expected: "username must be at least 3 characters",
		},
		{
			name: "Invalid email",
			reqBody: map[string]string{
				"username": "testuser",
				"email":    "invalid-email",
				"password": "Pass123456",
			},
			expected: "invalid email format",
		},
		{
			name: "Short password",
			reqBody: map[string]string{
				"username": "testuser",
				"email":    "test@example.com",
				"password": "short",
			},
			expected: "password must be at least 8 characters",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			bodyBytes, _ := json.Marshal(tc.reqBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			s.router.ServeHTTP(w, req)

			assert.Equal(s.T(), http.StatusBadRequest, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			assert.Contains(s.T(), response["error"], tc.expected)
		})
	}
}

// TestLoginSuccess tests successful login
func (s *AuthHandlerIntegrationTestSuite) TestLoginSuccess() {
	// Create test user
	testUser, _ := testutil.CreateTestUser("loginuser", "login@example.com", "LoginPass123", models.RoleUser)
	s.testDB.DB.Create(testUser)

	// Login request
	reqBody := map[string]string{
		"email":    "login@example.com",
		"password": "LoginPass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(s.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "Login successful", response["message"])

	// Check user data
	user := response["user"].(map[string]interface{})
	assert.Equal(s.T(), "loginuser", user["username"])
	assert.Equal(s.T(), "login@example.com", user["email"])

	// Check cookie
	cookies := w.Result().Cookies()
	assert.NotEmpty(s.T(), cookies)
}

// TestLoginInvalidCredentials tests login with wrong password
func (s *AuthHandlerIntegrationTestSuite) TestLoginInvalidCredentials() {
	// Create test user
	testUser, _ := testutil.CreateTestUser("loginuser", "login@example.com", "CorrectPass123", models.RoleUser)
	s.testDB.DB.Create(testUser)

	// Login with wrong password
	reqBody := map[string]string{
		"email":    "login@example.com",
		"password": "WrongPass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(s.T(), response["error"], "invalid credentials")
}

// TestLoginNonExistentUser tests login with non-existent email
func (s *AuthHandlerIntegrationTestSuite) TestLoginNonExistentUser() {
	reqBody := map[string]string{
		"email":    "nonexistent@example.com",
		"password": "SomePass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(s.T(), response["error"], "invalid credentials")
}

// TestSuite runs all tests in the suite
func TestAuthHandlerIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AuthHandlerIntegrationTestSuite))
}
