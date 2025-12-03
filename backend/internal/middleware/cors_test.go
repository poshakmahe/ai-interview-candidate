package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORS_AllowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		allowedOrigins string
		requestOrigin  string
		expectHeader   string
	}{
		{
			name:           "single allowed origin - match",
			allowedOrigins: "http://localhost:3000",
			requestOrigin:  "http://localhost:3000",
			expectHeader:   "http://localhost:3000",
		},
		{
			name:           "single allowed origin - no match",
			allowedOrigins: "http://localhost:3000",
			requestOrigin:  "http://evil.com",
			expectHeader:   "",
		},
		{
			name:           "multiple allowed origins - first match",
			allowedOrigins: "http://localhost:3000,http://example.com",
			requestOrigin:  "http://localhost:3000",
			expectHeader:   "http://localhost:3000",
		},
		{
			name:           "multiple allowed origins - second match",
			allowedOrigins: "http://localhost:3000,http://example.com",
			requestOrigin:  "http://example.com",
			expectHeader:   "http://example.com",
		},
		{
			name:           "multiple allowed origins - no match",
			allowedOrigins: "http://localhost:3000,http://example.com",
			requestOrigin:  "http://evil.com",
			expectHeader:   "",
		},
		{
			name:           "wildcard allows all",
			allowedOrigins: "*",
			requestOrigin:  "http://any-origin.com",
			expectHeader:   "http://any-origin.com",
		},
		{
			name:           "origins with spaces",
			allowedOrigins: "http://localhost:3000, http://example.com",
			requestOrigin:  "http://example.com",
			expectHeader:   "http://example.com",
		},
		{
			name:           "no origin header",
			allowedOrigins: "http://localhost:3000",
			requestOrigin:  "",
			expectHeader:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(CORS(tt.allowedOrigins))
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}
			router.ServeHTTP(w, req)

			got := w.Header().Get("Access-Control-Allow-Origin")
			if got != tt.expectHeader {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.expectHeader)
			}
		})
	}
}

func TestCORS_Headers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS("http://localhost:3000"))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(w, req)

	// Check all CORS headers are set
	expectedHeaders := map[string]string{
		"Access-Control-Allow-Methods":     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		"Access-Control-Allow-Headers":     "Origin, Content-Type, Accept, Authorization",
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Max-Age":           "86400",
	}

	for header, expected := range expectedHeaders {
		got := w.Header().Get(header)
		if got != expected {
			t.Errorf("%s = %q, want %q", header, got, expected)
		}
	}
}

func TestCORS_OptionsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS("http://localhost:3000"))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(w, req)

	// OPTIONS should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("OPTIONS request status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Body should be empty
	if w.Body.Len() != 0 {
		t.Errorf("OPTIONS request body should be empty, got %q", w.Body.String())
	}
}

func TestCORS_PreflightRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS("http://localhost:3000"))
	router.POST("/api/data", func(c *gin.Context) {
		c.String(http.StatusOK, "data received")
	})

	// Simulate CORS preflight request
	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/api/data", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Preflight request status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Verify CORS headers are present
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Error("Preflight missing Access-Control-Allow-Origin header")
	}

	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Preflight missing Access-Control-Allow-Methods header")
	}
}

func TestCORS_RegularRequestPassesThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handlerCalled := false
	router := gin.New()
	router.Use(CORS("http://localhost:3000"))
	router.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.String(http.StatusOK, "handler executed")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Handler should be called for non-OPTIONS requests")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	if w.Body.String() != "handler executed" {
		t.Errorf("Body = %q, want %q", w.Body.String(), "handler executed")
	}
}

func TestCORS_OptionsDoesNotCallHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handlerCalled := false
	router := gin.New()
	router.Use(CORS("http://localhost:3000"))
	router.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.String(http.StatusOK, "should not see this")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(w, req)

	if handlerCalled {
		t.Error("Handler should NOT be called for OPTIONS requests")
	}
}

func TestCORS_DisallowedOriginStillGetsOtherHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS("http://localhost:3000"))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	router.ServeHTTP(w, req)

	// Origin header should NOT be set for disallowed origins
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("Disallowed origin should not get Access-Control-Allow-Origin header")
	}

	// Other CORS headers should still be set
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Access-Control-Allow-Methods should still be set")
	}
}

func TestCORS_EmptyAllowedOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS(""))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://any-origin.com")
	router.ServeHTTP(w, req)

	// Should not crash, origin should not be allowed
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("Empty allowed origins should not allow any origin")
	}
}

// Benchmark
func BenchmarkCORS(b *testing.B) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS("http://localhost:3000,http://example.com,http://test.com"))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
