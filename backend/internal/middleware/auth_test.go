package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestNewAuthMiddleware(t *testing.T) {
	secret := "test-secret-key"
	auth := NewAuthMiddleware(secret)

	if auth == nil {
		t.Fatal("NewAuthMiddleware returned nil")
	}

	if string(auth.jwtSecret) != secret {
		t.Errorf("jwtSecret = %q, want %q", string(auth.jwtSecret), secret)
	}
}

func TestGenerateToken(t *testing.T) {
	auth := NewAuthMiddleware("test-secret-key-12345")
	userID := uuid.New()
	email := "test@example.com"

	token, err := auth.GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateToken() returned empty token")
	}

	// Validate the token structure
	claims, err := auth.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("claims.UserID = %v, want %v", claims.UserID, userID)
	}

	if claims.Email != email {
		t.Errorf("claims.Email = %q, want %q", claims.Email, email)
	}
}

func TestValidateToken_Valid(t *testing.T) {
	auth := NewAuthMiddleware("test-secret-key-12345")
	userID := uuid.New()
	email := "test@example.com"

	token, err := auth.GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := auth.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("claims.UserID = %v, want %v", claims.UserID, userID)
	}

	if claims.Email != email {
		t.Errorf("claims.Email = %q, want %q", claims.Email, email)
	}

	// Check that token expires in ~24 hours
	expiresIn := time.Until(claims.ExpiresAt.Time)
	if expiresIn < 23*time.Hour || expiresIn > 25*time.Hour {
		t.Errorf("Token expiration = %v, expected ~24 hours", expiresIn)
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	auth := NewAuthMiddleware("test-secret-key-12345")

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{
			name:    "empty token",
			token:   "",
			wantErr: ErrInvalidToken,
		},
		{
			name:    "malformed token",
			token:   "not-a-valid-jwt",
			wantErr: ErrInvalidToken,
		},
		{
			name:    "wrong number of segments",
			token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0",
			wantErr: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := auth.ValidateToken(tt.token)
			if err != tt.wantErr {
				t.Errorf("ValidateToken(%q) error = %v, wantErr = %v", tt.token, err, tt.wantErr)
			}
		})
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	auth1 := NewAuthMiddleware("secret-key-1")
	auth2 := NewAuthMiddleware("secret-key-2")

	token, err := auth1.GenerateToken(uuid.New(), "test@example.com")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Try to validate with different secret
	_, err = auth2.ValidateToken(token)
	if err != ErrInvalidToken {
		t.Errorf("ValidateToken() with wrong secret error = %v, want ErrInvalidToken", err)
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	auth := NewAuthMiddleware("test-secret-key-12345")
	userID := uuid.New()

	// Create an expired token manually
	claims := Claims{
		UserID: userID,
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(auth.jwtSecret)
	if err != nil {
		t.Fatalf("Failed to create expired token: %v", err)
	}

	_, err = auth.ValidateToken(tokenString)
	if err != ErrExpiredToken {
		t.Errorf("ValidateToken(expired) error = %v, want ErrExpiredToken", err)
	}
}

func TestValidateToken_WrongSigningMethod(t *testing.T) {
	auth := NewAuthMiddleware("test-secret-key-12345")

	// Create token with different signing method (none)
	claims := Claims{
		UserID: uuid.New(),
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}

	// Create an unsigned token (algorithm "none" attack)
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	_, err := auth.ValidateToken(tokenString)
	if err != ErrInvalidToken {
		t.Errorf("ValidateToken(none algorithm) error = %v, want ErrInvalidToken", err)
	}
}

func TestAuthenticate_Success(t *testing.T) {
	auth := NewAuthMiddleware("test-secret-key-12345")
	userID := uuid.New()
	email := "test@example.com"

	token, err := auth.GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	// Run middleware
	handler := auth.Authenticate()
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Response status = %d, want %d", w.Code, http.StatusOK)
	}

	// Check context values
	contextUserID, exists := c.Get("user_id")
	if !exists {
		t.Error("user_id not set in context")
	} else if contextUserID.(uuid.UUID) != userID {
		t.Errorf("user_id = %v, want %v", contextUserID, userID)
	}

	contextEmail, exists := c.Get("user_email")
	if !exists {
		t.Error("user_email not set in context")
	} else if contextEmail.(string) != email {
		t.Errorf("user_email = %q, want %q", contextEmail, email)
	}
}

func TestAuthenticate_NoHeader(t *testing.T) {
	auth := NewAuthMiddleware("test-secret-key-12345")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	handler := auth.Authenticate()
	handler(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Response status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	if !c.IsAborted() {
		t.Error("Request should be aborted")
	}
}

func TestAuthenticate_InvalidHeaderFormat(t *testing.T) {
	auth := NewAuthMiddleware("test-secret-key-12345")

	tests := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		{"wrong prefix", "Basic eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		{"only bearer", "Bearer"},
		{"too many parts", "Bearer token extra"},
		{"empty after bearer", "Bearer "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)
			c.Request.Header.Set("Authorization", tt.header)

			handler := auth.Authenticate()
			handler(c)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Response status = %d, want %d", w.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestAuthenticate_InvalidToken(t *testing.T) {
	auth := NewAuthMiddleware("test-secret-key-12345")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer invalid-token")

	handler := auth.Authenticate()
	handler(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Response status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	if !c.IsAborted() {
		t.Error("Request should be aborted")
	}
}

func TestAuthenticate_ExpiredToken(t *testing.T) {
	auth := NewAuthMiddleware("test-secret-key-12345")

	// Create expired token
	claims := Claims{
		UserID: uuid.New(),
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(auth.jwtSecret)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+tokenString)

	handler := auth.Authenticate()
	handler(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Response status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticate_BearerCaseInsensitive(t *testing.T) {
	auth := NewAuthMiddleware("test-secret-key-12345")
	token, _ := auth.GenerateToken(uuid.New(), "test@example.com")

	// Test lowercase "bearer"
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "bearer "+token)

	handler := auth.Authenticate()
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Response status with lowercase bearer = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestGetUserID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Test when not set
	_, exists := GetUserID(c)
	if exists {
		t.Error("GetUserID should return false when not set")
	}

	// Test when set
	expectedID := uuid.New()
	c.Set("user_id", expectedID)

	userID, exists := GetUserID(c)
	if !exists {
		t.Error("GetUserID should return true when set")
	}
	if userID != expectedID {
		t.Errorf("GetUserID = %v, want %v", userID, expectedID)
	}
}

func TestGetUserID_WrongType(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Set wrong type
	c.Set("user_id", "not-a-uuid")

	_, ok := GetUserID(c)
	if ok {
		t.Error("GetUserID should return false when type is wrong")
	}
}

func TestGetUserEmail(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Test when not set
	_, exists := GetUserEmail(c)
	if exists {
		t.Error("GetUserEmail should return false when not set")
	}

	// Test when set
	expectedEmail := "test@example.com"
	c.Set("user_email", expectedEmail)

	email, exists := GetUserEmail(c)
	if !exists {
		t.Error("GetUserEmail should return true when set")
	}
	if email != expectedEmail {
		t.Errorf("GetUserEmail = %q, want %q", email, expectedEmail)
	}
}

func TestGetUserEmail_WrongType(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Set wrong type
	c.Set("user_email", 12345)

	_, ok := GetUserEmail(c)
	if ok {
		t.Error("GetUserEmail should return false when type is wrong")
	}
}

// Integration test: full auth flow
func TestAuthFlow_Integration(t *testing.T) {
	auth := NewAuthMiddleware("integration-test-secret")
	userID := uuid.New()
	email := "integration@test.com"

	// Step 1: Generate token
	token, err := auth.GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Step 2: Create a protected endpoint
	router := gin.New()
	router.GET("/protected", auth.Authenticate(), func(c *gin.Context) {
		id, _ := GetUserID(c)
		mail, _ := GetUserEmail(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id": id,
			"email":   mail,
		})
	})

	// Step 3: Make authenticated request
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Protected endpoint returned %d, want %d", w.Code, http.StatusOK)
	}

	// Step 4: Make unauthenticated request
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/protected", nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusUnauthorized {
		t.Errorf("Unauthenticated request returned %d, want %d", w2.Code, http.StatusUnauthorized)
	}
}

// Benchmark tests
func BenchmarkGenerateToken(b *testing.B) {
	auth := NewAuthMiddleware("benchmark-secret-key")
	userID := uuid.New()

	for i := 0; i < b.N; i++ {
		auth.GenerateToken(userID, "benchmark@test.com")
	}
}

func BenchmarkValidateToken(b *testing.B) {
	auth := NewAuthMiddleware("benchmark-secret-key")
	token, _ := auth.GenerateToken(uuid.New(), "benchmark@test.com")

	for i := 0; i < b.N; i++ {
		auth.ValidateToken(token)
	}
}
