package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/katim/secure-doc-vault/internal/database"
	"github.com/katim/secure-doc-vault/internal/middleware"
	"github.com/katim/secure-doc-vault/internal/models"
	"github.com/katim/secure-doc-vault/internal/services"
)

func getTestDatabaseURL() string {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		return url
	}
	// Default to port 5433 to avoid conflict with dev database on 5432
	return "postgres://postgres:postgres@localhost:5433/docvault_test?sslmode=disable"
}

func setupTestDB(t *testing.T) *database.DB {
	// Use test database
	db, err := database.New(getTestDatabaseURL())
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test database: %v", err)
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Clean up tables
	db.Exec("DELETE FROM document_shares")
	db.Exec("DELETE FROM documents")
	db.Exec("DELETE FROM users")

	return db
}

func setupAuthRouter(db *database.DB) (*gin.Engine, *AuthHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	userService := services.NewUserService(db)
	authMiddleware := middleware.NewAuthMiddleware("test-secret")
	authHandler := NewAuthHandler(userService, authMiddleware)

	auth := router.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.GET("/me", authMiddleware.Authenticate(), authHandler.GetMe)
	}

	return router, authHandler
}

func TestRegister_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _ := setupAuthRouter(db)

	reqBody := models.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response models.AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Token == "" {
		t.Error("Expected token in response")
	}
	if response.User.Email != reqBody.Email {
		t.Errorf("Expected email %s, got %s", reqBody.Email, response.User.Email)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _ := setupAuthRouter(db)

	reqBody := models.RegisterRequest{
		Email:    "duplicate@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	body, _ := json.Marshal(reqBody)

	// First registration
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Second registration with same email
	req, _ = http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _ := setupAuthRouter(db)

	reqBody := map[string]string{
		"email":    "invalid-email",
		"password": "password123",
		"name":     "Test User",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _ := setupAuthRouter(db)

	reqBody := map[string]string{
		"email":    "test@example.com",
		"password": "short",
		"name":     "Test User",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _ := setupAuthRouter(db)

	// First register a user
	regBody, _ := json.Marshal(models.RegisterRequest{
		Email:    "login@example.com",
		Password: "password123",
		Name:     "Test User",
	})
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(regBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Now login
	loginBody, _ := json.Marshal(models.LoginRequest{
		Email:    "login@example.com",
		Password: "password123",
	})
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Token == "" {
		t.Error("Expected token in response")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _ := setupAuthRouter(db)

	// First register a user
	regBody, _ := json.Marshal(models.RegisterRequest{
		Email:    "wrongpwd@example.com",
		Password: "password123",
		Name:     "Test User",
	})
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(regBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Login with wrong password
	loginBody, _ := json.Marshal(models.LoginRequest{
		Email:    "wrongpwd@example.com",
		Password: "wrongpassword",
	})
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _ := setupAuthRouter(db)

	loginBody, _ := json.Marshal(models.LoginRequest{
		Email:    "notfound@example.com",
		Password: "password123",
	})
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestGetMe_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _ := setupAuthRouter(db)

	// First register and get token
	regBody, _ := json.Marshal(models.RegisterRequest{
		Email:    "me@example.com",
		Password: "password123",
		Name:     "Test User",
	})
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(regBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var authResp models.AuthResponse
	json.Unmarshal(w.Body.Bytes(), &authResp)

	// Get current user
	req, _ = http.NewRequest("GET", "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+authResp.Token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var user models.User
	if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if user.Email != "me@example.com" {
		t.Errorf("Expected email me@example.com, got %s", user.Email)
	}
}

func TestGetMe_Unauthorized(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _ := setupAuthRouter(db)

	req, _ := http.NewRequest("GET", "/auth/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestGetMe_InvalidToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _ := setupAuthRouter(db)

	req, _ := http.NewRequest("GET", "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}
