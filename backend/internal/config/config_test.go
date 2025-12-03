package config

import (
	"os"
	"testing"
)

func TestLoad_WithJWTSecret(t *testing.T) {
	// Set required JWT_SECRET
	t.Setenv("JWT_SECRET", "test-secret-key")

	// Clear other env vars to test defaults
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("UPLOAD_DIR", "")
	t.Setenv("MAX_FILE_SIZE", "")
	t.Setenv("ALLOWED_ORIGINS", "")

	cfg := Load()

	if cfg == nil {
		t.Fatal("Load() returned nil")
	}

	if cfg.JWTSecret != "test-secret-key" {
		t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, "test-secret-key")
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Set required JWT_SECRET
	t.Setenv("JWT_SECRET", "test-secret")

	// Explicitly unset optional env vars
	os.Unsetenv("PORT")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("UPLOAD_DIR")
	os.Unsetenv("MAX_FILE_SIZE")
	os.Unsetenv("ALLOWED_ORIGINS")

	cfg := Load()

	// Test default values
	if cfg.Port != "8080" {
		t.Errorf("Default Port = %q, want %q", cfg.Port, "8080")
	}

	expectedDBURL := "postgres://postgres:postgres@localhost:5432/docvault?sslmode=disable"
	if cfg.DatabaseURL != expectedDBURL {
		t.Errorf("Default DatabaseURL = %q, want %q", cfg.DatabaseURL, expectedDBURL)
	}

	if cfg.UploadDir != "./uploads" {
		t.Errorf("Default UploadDir = %q, want %q", cfg.UploadDir, "./uploads")
	}

	if cfg.MaxFileSize != 10485760 { // 10MB
		t.Errorf("Default MaxFileSize = %d, want %d", cfg.MaxFileSize, 10485760)
	}

	if cfg.AllowedOrigins != "http://localhost:3000" {
		t.Errorf("Default AllowedOrigins = %q, want %q", cfg.AllowedOrigins, "http://localhost:3000")
	}
}

func TestLoad_CustomValues(t *testing.T) {
	// Set all env vars
	t.Setenv("JWT_SECRET", "custom-jwt-secret")
	t.Setenv("PORT", "9000")
	t.Setenv("DATABASE_URL", "postgres://custom:password@db:5432/mydb")
	t.Setenv("UPLOAD_DIR", "/custom/uploads")
	t.Setenv("MAX_FILE_SIZE", "52428800") // 50MB
	t.Setenv("ALLOWED_ORIGINS", "https://example.com,https://api.example.com")

	cfg := Load()

	if cfg.Port != "9000" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9000")
	}

	if cfg.DatabaseURL != "postgres://custom:password@db:5432/mydb" {
		t.Errorf("DatabaseURL = %q, want custom value", cfg.DatabaseURL)
	}

	if cfg.JWTSecret != "custom-jwt-secret" {
		t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, "custom-jwt-secret")
	}

	if cfg.UploadDir != "/custom/uploads" {
		t.Errorf("UploadDir = %q, want %q", cfg.UploadDir, "/custom/uploads")
	}

	if cfg.MaxFileSize != 52428800 {
		t.Errorf("MaxFileSize = %d, want %d", cfg.MaxFileSize, 52428800)
	}

	if cfg.AllowedOrigins != "https://example.com,https://api.example.com" {
		t.Errorf("AllowedOrigins = %q, want custom value", cfg.AllowedOrigins)
	}
}

func TestLoad_InvalidMaxFileSize(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("MAX_FILE_SIZE", "not-a-number")

	cfg := Load()

	// Should use 0 when parsing fails (strconv.ParseInt returns 0 on error)
	if cfg.MaxFileSize != 0 {
		t.Errorf("MaxFileSize with invalid value = %d, want 0", cfg.MaxFileSize)
	}
}

func TestLoad_MissingJWTSecret_Panics(t *testing.T) {
	// Unset JWT_SECRET
	os.Unsetenv("JWT_SECRET")
	t.Setenv("JWT_SECRET", "") // Also set empty to be sure

	defer func() {
		if r := recover(); r == nil {
			t.Error("Load() should panic when JWT_SECRET is not set")
		}
	}()

	Load()
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		setEnv       bool
		expected     string
	}{
		{
			name:         "returns env value when set",
			key:          "TEST_VAR_1",
			defaultValue: "default",
			envValue:     "custom",
			setEnv:       true,
			expected:     "custom",
		},
		{
			name:         "returns default when not set",
			key:          "TEST_VAR_2",
			defaultValue: "default",
			envValue:     "",
			setEnv:       false,
			expected:     "default",
		},
		{
			name:         "returns empty string when set to empty",
			key:          "TEST_VAR_3",
			defaultValue: "default",
			envValue:     "",
			setEnv:       true,
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up env var
			os.Unsetenv(tt.key)

			if tt.setEnv {
				t.Setenv(tt.key, tt.envValue)
			}

			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnv(%q, %q) = %q, want %q", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestConfigStruct(t *testing.T) {
	// Test that Config struct has all expected fields
	cfg := &Config{
		Port:           "8080",
		DatabaseURL:    "postgres://localhost/db",
		JWTSecret:      "secret",
		UploadDir:      "./uploads",
		MaxFileSize:    10485760,
		AllowedOrigins: "http://localhost:3000",
	}

	if cfg.Port == "" {
		t.Error("Config.Port should be accessible")
	}
	if cfg.DatabaseURL == "" {
		t.Error("Config.DatabaseURL should be accessible")
	}
	if cfg.JWTSecret == "" {
		t.Error("Config.JWTSecret should be accessible")
	}
	if cfg.UploadDir == "" {
		t.Error("Config.UploadDir should be accessible")
	}
	if cfg.MaxFileSize == 0 {
		t.Error("Config.MaxFileSize should be accessible")
	}
	if cfg.AllowedOrigins == "" {
		t.Error("Config.AllowedOrigins should be accessible")
	}
}
