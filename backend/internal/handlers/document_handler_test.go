package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/katim/secure-doc-vault/internal/database"
	"github.com/katim/secure-doc-vault/internal/middleware"
	"github.com/katim/secure-doc-vault/internal/models"
	"github.com/katim/secure-doc-vault/internal/services"
)

func setupDocumentRouter(db *database.DB) (*gin.Engine, *AuthHandler, *DocumentHandler, string) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create temp upload dir
	uploadDir, _ := os.MkdirTemp("", "docvault-test-*")

	userService := services.NewUserService(db)
	documentService := services.NewDocumentService(db, uploadDir)
	authMiddleware := middleware.NewAuthMiddleware("test-secret")

	authHandler := NewAuthHandler(userService, authMiddleware)
	documentHandler := NewDocumentHandler(documentService, 10*1024*1024) // 10MB

	// Auth routes
	auth := router.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// Document routes
	documents := router.Group("/documents")
	documents.Use(authMiddleware.Authenticate())
	{
		documents.GET("", documentHandler.ListDocuments)
		documents.POST("", documentHandler.UploadDocument)
		documents.GET("/:id", documentHandler.GetDocument)
		documents.PATCH("/:id", documentHandler.RenameDocument)
		documents.DELETE("/:id", documentHandler.DeleteDocument)
		documents.GET("/:id/download", documentHandler.DownloadDocument)
		documents.POST("/:id/share", documentHandler.ShareDocument)
	}

	router.GET("/shared", authMiddleware.Authenticate(), documentHandler.ListSharedDocuments)

	return router, authHandler, documentHandler, uploadDir
}

func registerAndLogin(router *gin.Engine, email, password, name string) string {
	// Register
	regBody, _ := json.Marshal(models.RegisterRequest{
		Email:    email,
		Password: password,
		Name:     name,
	})
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(regBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var authResp models.AuthResponse
	json.Unmarshal(w.Body.Bytes(), &authResp)
	return authResp.Token
}

func createTestFile(content string) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	// Create a form file with proper Content-Type header (text/plain instead of default application/octet-stream)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="test.txt"`)
	h.Set("Content-Type", "text/plain")
	part, _ := writer.CreatePart(h)
	io.WriteString(part, content)
	writer.Close()
	return body, writer.FormDataContentType()
}

func TestListDocuments_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _, _, uploadDir := setupDocumentRouter(db)
	defer os.RemoveAll(uploadDir)

	token := registerAndLogin(router, "list@example.com", "password123", "Test User")

	req, _ := http.NewRequest("GET", "/documents", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.PaginatedResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Total != 0 {
		t.Errorf("Expected 0 documents, got %d", response.Total)
	}
}

func TestUploadDocument_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _, _, uploadDir := setupDocumentRouter(db)
	defer os.RemoveAll(uploadDir)

	token := registerAndLogin(router, "upload@example.com", "password123", "Test User")

	body, contentType := createTestFile("Hello, World!")
	req, _ := http.NewRequest("POST", "/documents", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var doc models.Document
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if doc.OriginalName != "test.txt" {
		t.Errorf("Expected original name 'test.txt', got %s", doc.OriginalName)
	}
}

func TestUploadDocument_Unauthorized(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _, _, uploadDir := setupDocumentRouter(db)
	defer os.RemoveAll(uploadDir)

	body, contentType := createTestFile("Hello, World!")
	req, _ := http.NewRequest("POST", "/documents", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestGetDocument_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _, _, uploadDir := setupDocumentRouter(db)
	defer os.RemoveAll(uploadDir)

	token := registerAndLogin(router, "get@example.com", "password123", "Test User")

	// Upload a document
	body, contentType := createTestFile("Test content")
	req, _ := http.NewRequest("POST", "/documents", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var uploadedDoc models.Document
	json.Unmarshal(w.Body.Bytes(), &uploadedDoc)

	// Get the document
	req, _ = http.NewRequest("GET", "/documents/"+uploadedDoc.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var doc models.Document
	json.Unmarshal(w.Body.Bytes(), &doc)

	if doc.ID != uploadedDoc.ID {
		t.Errorf("Expected doc ID %s, got %s", uploadedDoc.ID, doc.ID)
	}
}

func TestGetDocument_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _, _, uploadDir := setupDocumentRouter(db)
	defer os.RemoveAll(uploadDir)

	token := registerAndLogin(router, "notfound@example.com", "password123", "Test User")

	req, _ := http.NewRequest("GET", "/documents/00000000-0000-0000-0000-000000000000", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDeleteDocument_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _, _, uploadDir := setupDocumentRouter(db)
	defer os.RemoveAll(uploadDir)

	token := registerAndLogin(router, "delete@example.com", "password123", "Test User")

	// Upload a document
	body, contentType := createTestFile("Test content")
	req, _ := http.NewRequest("POST", "/documents", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var doc models.Document
	json.Unmarshal(w.Body.Bytes(), &doc)

	// Delete the document
	req, _ = http.NewRequest("DELETE", "/documents/"+doc.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
	}

	// Verify document is no longer accessible
	req, _ = http.NewRequest("GET", "/documents/"+doc.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d after delete, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDeleteDocument_AccessDenied(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _, _, uploadDir := setupDocumentRouter(db)
	defer os.RemoveAll(uploadDir)

	// User 1 uploads a document
	token1 := registerAndLogin(router, "owner@example.com", "password123", "Owner")
	body, contentType := createTestFile("Test content")
	req, _ := http.NewRequest("POST", "/documents", body)
	req.Header.Set("Authorization", "Bearer "+token1)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var doc models.Document
	json.Unmarshal(w.Body.Bytes(), &doc)

	// User 2 tries to delete it
	token2 := registerAndLogin(router, "other@example.com", "password123", "Other")
	req, _ = http.NewRequest("DELETE", "/documents/"+doc.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token2)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestShareDocument_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _, _, uploadDir := setupDocumentRouter(db)
	defer os.RemoveAll(uploadDir)

	// User 1 uploads a document
	token1 := registerAndLogin(router, "sharer@example.com", "password123", "Sharer")
	body, contentType := createTestFile("Test content")
	req, _ := http.NewRequest("POST", "/documents", body)
	req.Header.Set("Authorization", "Bearer "+token1)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var doc models.Document
	json.Unmarshal(w.Body.Bytes(), &doc)

	// Create User 2
	registerAndLogin(router, "recipient@example.com", "password123", "Recipient")

	// Share with User 2
	shareBody, _ := json.Marshal(models.ShareRequest{
		Email:      "recipient@example.com",
		Permission: "view",
	})
	req, _ = http.NewRequest("POST", "/documents/"+doc.ID.String()+"/share", bytes.NewBuffer(shareBody))
	req.Header.Set("Authorization", "Bearer "+token1)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestShareDocument_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _, _, uploadDir := setupDocumentRouter(db)
	defer os.RemoveAll(uploadDir)

	token := registerAndLogin(router, "sharer2@example.com", "password123", "Sharer")

	// Upload a document
	body, contentType := createTestFile("Test content")
	req, _ := http.NewRequest("POST", "/documents", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var doc models.Document
	json.Unmarshal(w.Body.Bytes(), &doc)

	// Try to share with non-existent user
	shareBody, _ := json.Marshal(models.ShareRequest{
		Email:      "nonexistent@example.com",
		Permission: "view",
	})
	req, _ = http.NewRequest("POST", "/documents/"+doc.ID.String()+"/share", bytes.NewBuffer(shareBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestListSharedDocuments(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _, _, uploadDir := setupDocumentRouter(db)
	defer os.RemoveAll(uploadDir)

	// User 1 uploads and shares a document
	token1 := registerAndLogin(router, "owner2@example.com", "password123", "Owner")
	body, contentType := createTestFile("Shared content")
	req, _ := http.NewRequest("POST", "/documents", body)
	req.Header.Set("Authorization", "Bearer "+token1)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var doc models.Document
	json.Unmarshal(w.Body.Bytes(), &doc)

	// Create User 2
	token2 := registerAndLogin(router, "viewer@example.com", "password123", "Viewer")

	// Share with User 2
	shareBody, _ := json.Marshal(models.ShareRequest{
		Email:      "viewer@example.com",
		Permission: "view",
	})
	req, _ = http.NewRequest("POST", "/documents/"+doc.ID.String()+"/share", bytes.NewBuffer(shareBody))
	req.Header.Set("Authorization", "Bearer "+token1)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// User 2 lists shared documents
	req, _ = http.NewRequest("GET", "/shared", nil)
	req.Header.Set("Authorization", "Bearer "+token2)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.PaginatedResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Total != 1 {
		t.Errorf("Expected 1 shared document, got %d", response.Total)
	}
}

func TestRenameDocument_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _, _, uploadDir := setupDocumentRouter(db)
	defer os.RemoveAll(uploadDir)

	token := registerAndLogin(router, "rename@example.com", "password123", "Test User")

	// Upload a document
	body, contentType := createTestFile("Test content")
	req, _ := http.NewRequest("POST", "/documents", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var doc models.Document
	json.Unmarshal(w.Body.Bytes(), &doc)

	// Rename the document
	renameBody, _ := json.Marshal(map[string]string{"name": "new-name.txt"})
	req, _ = http.NewRequest("PATCH", "/documents/"+doc.ID.String(), bytes.NewBuffer(renameBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify the name changed
	req, _ = http.NewRequest("GET", "/documents/"+doc.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var updatedDoc models.Document
	json.Unmarshal(w.Body.Bytes(), &updatedDoc)

	if updatedDoc.Name != "new-name.txt" {
		t.Errorf("Expected name 'new-name.txt', got %s", updatedDoc.Name)
	}
}

func TestDownloadDocument_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router, _, _, uploadDir := setupDocumentRouter(db)
	defer os.RemoveAll(uploadDir)

	token := registerAndLogin(router, "download@example.com", "password123", "Test User")

	// Upload a document
	fileContent := "Hello, this is test content!"
	body, contentType := createTestFile(fileContent)
	req, _ := http.NewRequest("POST", "/documents", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var doc models.Document
	json.Unmarshal(w.Body.Bytes(), &doc)

	// Download the document
	req, _ = http.NewRequest("GET", "/documents/"+doc.ID.String()+"/download", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != fileContent {
		t.Errorf("Expected content '%s', got '%s'", fileContent, w.Body.String())
	}
}
