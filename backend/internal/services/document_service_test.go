package services

import (
	"bytes"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestNewDocumentService(t *testing.T) {
	db, _ := newMockDB(t)
	defer db.Close()

	tempDir := t.TempDir()
	service := NewDocumentService(db, tempDir)

	if service == nil {
		t.Fatal("NewDocumentService returned nil")
	}

	if service.db != db {
		t.Error("DocumentService.db not set correctly")
	}

	if service.uploadDir != tempDir {
		t.Errorf("DocumentService.uploadDir = %q, want %q", service.uploadDir, tempDir)
	}

	// Verify directory was created
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Upload directory should be created")
	}
}

func TestNewDocumentService_CreatesDirectory(t *testing.T) {
	db, _ := newMockDB(t)
	defer db.Close()

	tempDir := filepath.Join(t.TempDir(), "nested", "uploads")
	service := NewDocumentService(db, tempDir)

	if service == nil {
		t.Fatal("NewDocumentService returned nil")
	}

	// Verify nested directory was created
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Nested upload directory should be created")
	}
}

func TestDocumentService_Create_Success(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	tempDir := t.TempDir()
	service := NewDocumentService(db, tempDir)

	ownerID := uuid.New()
	name := "Test Document"
	originalName := "original.pdf"
	mimeType := "application/pdf"
	fileContent := []byte("test file content")
	fileData := bytes.NewReader(fileContent)

	mock.ExpectExec(`INSERT INTO documents`).
		WithArgs(
			sqlmock.AnyArg(), // id
			ownerID,
			name,
			originalName,
			int64(len(fileContent)),
			mimeType,
			"AES-256-GCM",
			sqlmock.AnyArg(), // file_path
			false,
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	doc, err := service.Create(ownerID, name, originalName, mimeType, int64(len(fileContent)), fileData)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if doc == nil {
		t.Fatal("Create() returned nil document")
	}

	if doc.OwnerID != ownerID {
		t.Errorf("doc.OwnerID = %v, want %v", doc.OwnerID, ownerID)
	}

	if doc.Name != name {
		t.Errorf("doc.Name = %q, want %q", doc.Name, name)
	}

	if doc.OriginalName != originalName {
		t.Errorf("doc.OriginalName = %q, want %q", doc.OriginalName, originalName)
	}

	if doc.MimeType != mimeType {
		t.Errorf("doc.MimeType = %q, want %q", doc.MimeType, mimeType)
	}

	// Verify file was created
	if _, err := os.Stat(doc.FilePath); os.IsNotExist(err) {
		t.Error("File should be created on disk")
	}

	// Verify file content
	content, err := os.ReadFile(doc.FilePath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}
	if !bytes.Equal(content, fileContent) {
		t.Error("File content doesn't match")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}

	// Cleanup
	os.Remove(doc.FilePath)
}

func TestDocumentService_Create_InvalidContentType(t *testing.T) {
	db, _ := newMockDB(t)
	defer db.Close()

	tempDir := t.TempDir()
	service := NewDocumentService(db, tempDir)

	ownerID := uuid.New()
	fileData := bytes.NewReader([]byte("test"))

	_, err := service.Create(ownerID, "test", "test.exe", "application/x-executable", 4, fileData)

	if err == nil {
		t.Error("Create() should reject invalid content type")
	}
}

func TestDocumentService_Create_InvalidFilename(t *testing.T) {
	db, _ := newMockDB(t)
	defer db.Close()

	tempDir := t.TempDir()
	service := NewDocumentService(db, tempDir)

	ownerID := uuid.New()
	fileData := bytes.NewReader([]byte("test"))

	// Test with empty original filename
	_, err := service.Create(ownerID, "test", "", "application/pdf", 4, fileData)

	if err == nil {
		t.Error("Create() should reject empty filename")
	}
}

func TestDocumentService_Create_PathTraversalPrevention(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	tempDir := t.TempDir()
	service := NewDocumentService(db, tempDir)

	ownerID := uuid.New()
	maliciousName := "../../../etc/passwd"
	fileContent := []byte("test")
	fileData := bytes.NewReader(fileContent)

	mock.ExpectExec(`INSERT INTO documents`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	doc, err := service.Create(ownerID, maliciousName, "test.pdf", "application/pdf", int64(len(fileContent)), fileData)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Name should be sanitized
	if doc.Name == maliciousName {
		t.Error("Document name should be sanitized")
	}

	// File path should be within upload directory
	if !filepath.HasPrefix(doc.FilePath, tempDir) {
		t.Error("File path should be within upload directory")
	}

	// Cleanup
	os.Remove(doc.FilePath)
}

func TestDocumentService_Create_DBError_CleansUpFile(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	tempDir := t.TempDir()
	service := NewDocumentService(db, tempDir)

	ownerID := uuid.New()
	fileContent := []byte("test content")
	fileData := bytes.NewReader(fileContent)

	dbError := errors.New("database error")
	mock.ExpectExec(`INSERT INTO documents`).
		WillReturnError(dbError)

	_, err := service.Create(ownerID, "test", "test.pdf", "application/pdf", int64(len(fileContent)), fileData)

	if err == nil {
		t.Error("Create() should return error on DB failure")
	}

	// File should be cleaned up
	files, _ := os.ReadDir(tempDir)
	if len(files) != 0 {
		t.Error("File should be cleaned up after DB error")
	}
}

func TestDocumentService_GetByID_Success(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	service := NewDocumentService(db, t.TempDir())

	docID := uuid.New()
	ownerID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "original_name", "size", "mime_type",
		"encryption_algo", "file_path", "is_encrypted", "created_at", "updated_at", "deleted_at",
	}).AddRow(
		docID, ownerID, "Test Doc", "test.pdf", 1024, "application/pdf",
		"AES-256-GCM", "/path/to/file", false, time.Now(), time.Now(), nil,
	)

	mock.ExpectQuery(`SELECT .+ FROM documents WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(docID).
		WillReturnRows(rows)

	doc, err := service.GetByID(docID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if doc.ID != docID {
		t.Errorf("doc.ID = %v, want %v", doc.ID, docID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDocumentService_GetByID_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	service := NewDocumentService(db, t.TempDir())
	docID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM documents WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(docID).
		WillReturnError(sql.ErrNoRows)

	_, err := service.GetByID(docID)

	if err != ErrDocumentNotFound {
		t.Errorf("GetByID() error = %v, want ErrDocumentNotFound", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDocumentService_GetByOwner_Success(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	service := NewDocumentService(db, t.TempDir())
	ownerID := uuid.New()

	// Mock count query
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents WHERE owner_id = \$1`).
		WithArgs(ownerID).
		WillReturnRows(countRows)

	// Mock documents query
	docRows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "original_name", "size", "mime_type",
		"encryption_algo", "file_path", "is_encrypted", "created_at", "updated_at",
	}).
		AddRow(uuid.New(), ownerID, "Doc 1", "doc1.pdf", 1024, "application/pdf", "AES-256-GCM", "/path/1", false, time.Now(), time.Now()).
		AddRow(uuid.New(), ownerID, "Doc 2", "doc2.pdf", 2048, "application/pdf", "AES-256-GCM", "/path/2", false, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT .+ FROM documents WHERE owner_id = \$1 AND deleted_at IS NULL`).
		WithArgs(ownerID, 20, 0).
		WillReturnRows(docRows)

	docs, total, err := service.GetByOwner(ownerID, 1, 20)
	if err != nil {
		t.Fatalf("GetByOwner() error = %v", err)
	}

	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}

	if len(docs) != 2 {
		t.Errorf("len(docs) = %d, want 2", len(docs))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDocumentService_GetByOwner_Pagination(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	service := NewDocumentService(db, t.TempDir())
	ownerID := uuid.New()

	tests := []struct {
		name           string
		page           int
		perPage        int
		expectedOffset int
		expectedLimit  int
	}{
		{"page 1", 1, 20, 0, 20},
		{"page 2", 2, 20, 20, 20},
		{"invalid page defaults to 1", 0, 20, 0, 20},
		{"invalid perPage defaults to 20", 1, 0, 0, 20},
		{"perPage over 100 defaults to 20", 1, 200, 0, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
			mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents WHERE owner_id = \$1`).
				WithArgs(ownerID).
				WillReturnRows(countRows)

			docRows := sqlmock.NewRows([]string{
				"id", "owner_id", "name", "original_name", "size", "mime_type",
				"encryption_algo", "file_path", "is_encrypted", "created_at", "updated_at",
			})

			mock.ExpectQuery(`SELECT .+ FROM documents WHERE owner_id = \$1 AND deleted_at IS NULL`).
				WithArgs(ownerID, tt.expectedLimit, tt.expectedOffset).
				WillReturnRows(docRows)

			_, _, err := service.GetByOwner(ownerID, tt.page, tt.perPage)
			if err != nil {
				t.Fatalf("GetByOwner() error = %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestDocumentService_Delete_Success(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	tempDir := t.TempDir()
	service := NewDocumentService(db, tempDir)

	docID := uuid.New()
	ownerID := uuid.New()
	filePath := filepath.Join(tempDir, docID.String())

	// Create a temp file to delete
	os.WriteFile(filePath, []byte("test"), 0644)

	// Mock GetByID
	getRows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "original_name", "size", "mime_type",
		"encryption_algo", "file_path", "is_encrypted", "created_at", "updated_at", "deleted_at",
	}).AddRow(docID, ownerID, "Test", "test.pdf", 1024, "application/pdf", "AES-256-GCM", filePath, false, time.Now(), time.Now(), nil)

	mock.ExpectQuery(`SELECT .+ FROM documents WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(docID).
		WillReturnRows(getRows)

	// Mock soft delete
	mock.ExpectExec(`UPDATE documents SET deleted_at = \$1 WHERE id = \$2`).
		WithArgs(sqlmock.AnyArg(), docID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := service.Delete(docID, ownerID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("File should be deleted from disk")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDocumentService_Delete_NotOwner(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	service := NewDocumentService(db, t.TempDir())

	docID := uuid.New()
	ownerID := uuid.New()
	otherUserID := uuid.New()

	getRows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "original_name", "size", "mime_type",
		"encryption_algo", "file_path", "is_encrypted", "created_at", "updated_at", "deleted_at",
	}).AddRow(docID, ownerID, "Test", "test.pdf", 1024, "application/pdf", "AES-256-GCM", "/path", false, time.Now(), time.Now(), nil)

	mock.ExpectQuery(`SELECT .+ FROM documents WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(docID).
		WillReturnRows(getRows)

	err := service.Delete(docID, otherUserID)

	if err != ErrAccessDenied {
		t.Errorf("Delete() by non-owner error = %v, want ErrAccessDenied", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDocumentService_Share_Success(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	service := NewDocumentService(db, t.TempDir())

	docID := uuid.New()
	ownerID := uuid.New()
	sharedWithID := uuid.New()
	sharedWithEmail := "shared@example.com"

	// Mock GetByID
	getRows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "original_name", "size", "mime_type",
		"encryption_algo", "file_path", "is_encrypted", "created_at", "updated_at", "deleted_at",
	}).AddRow(docID, ownerID, "Test", "test.pdf", 1024, "application/pdf", "AES-256-GCM", "/path", false, time.Now(), time.Now(), nil)

	mock.ExpectQuery(`SELECT .+ FROM documents WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(docID).
		WillReturnRows(getRows)

	// Mock get shared user
	userRows := sqlmock.NewRows([]string{"id"}).AddRow(sharedWithID)
	mock.ExpectQuery(`SELECT id FROM users WHERE email = \$1`).
		WithArgs(sharedWithEmail).
		WillReturnRows(userRows)

	// Mock insert share
	mock.ExpectExec(`INSERT INTO document_shares`).
		WithArgs(docID, ownerID, sharedWithID, "view").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := service.Share(docID, ownerID, sharedWithEmail, "view")
	if err != nil {
		t.Fatalf("Share() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDocumentService_Share_UserNotFound(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	service := NewDocumentService(db, t.TempDir())

	docID := uuid.New()
	ownerID := uuid.New()

	// Mock GetByID
	getRows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "original_name", "size", "mime_type",
		"encryption_algo", "file_path", "is_encrypted", "created_at", "updated_at", "deleted_at",
	}).AddRow(docID, ownerID, "Test", "test.pdf", 1024, "application/pdf", "AES-256-GCM", "/path", false, time.Now(), time.Now(), nil)

	mock.ExpectQuery(`SELECT .+ FROM documents WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(docID).
		WillReturnRows(getRows)

	// Mock user not found
	mock.ExpectQuery(`SELECT id FROM users WHERE email = \$1`).
		WithArgs("nonexistent@example.com").
		WillReturnError(sql.ErrNoRows)

	err := service.Share(docID, ownerID, "nonexistent@example.com", "view")

	if err != ErrUserNotFound {
		t.Errorf("Share() with non-existent user error = %v, want ErrUserNotFound", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDocumentService_Share_NotOwner(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	service := NewDocumentService(db, t.TempDir())

	docID := uuid.New()
	ownerID := uuid.New()
	otherUserID := uuid.New()

	getRows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "original_name", "size", "mime_type",
		"encryption_algo", "file_path", "is_encrypted", "created_at", "updated_at", "deleted_at",
	}).AddRow(docID, ownerID, "Test", "test.pdf", 1024, "application/pdf", "AES-256-GCM", "/path", false, time.Now(), time.Now(), nil)

	mock.ExpectQuery(`SELECT .+ FROM documents WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(docID).
		WillReturnRows(getRows)

	err := service.Share(docID, otherUserID, "test@example.com", "view")

	if err != ErrAccessDenied {
		t.Errorf("Share() by non-owner error = %v, want ErrAccessDenied", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDocumentService_CanAccess_Owner(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	service := NewDocumentService(db, t.TempDir())

	docID := uuid.New()
	ownerID := uuid.New()

	getRows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "original_name", "size", "mime_type",
		"encryption_algo", "file_path", "is_encrypted", "created_at", "updated_at", "deleted_at",
	}).AddRow(docID, ownerID, "Test", "test.pdf", 1024, "application/pdf", "AES-256-GCM", "/path", false, time.Now(), time.Now(), nil)

	mock.ExpectQuery(`SELECT .+ FROM documents WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(docID).
		WillReturnRows(getRows)

	canAccess, permission, err := service.CanAccess(docID, ownerID)
	if err != nil {
		t.Fatalf("CanAccess() error = %v", err)
	}

	if !canAccess {
		t.Error("Owner should have access")
	}

	if permission != "owner" {
		t.Errorf("permission = %q, want %q", permission, "owner")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDocumentService_CanAccess_SharedUser(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	service := NewDocumentService(db, t.TempDir())

	docID := uuid.New()
	ownerID := uuid.New()
	sharedUserID := uuid.New()

	getRows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "original_name", "size", "mime_type",
		"encryption_algo", "file_path", "is_encrypted", "created_at", "updated_at", "deleted_at",
	}).AddRow(docID, ownerID, "Test", "test.pdf", 1024, "application/pdf", "AES-256-GCM", "/path", false, time.Now(), time.Now(), nil)

	mock.ExpectQuery(`SELECT .+ FROM documents WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(docID).
		WillReturnRows(getRows)

	// Mock share check
	shareRows := sqlmock.NewRows([]string{"permission"}).AddRow("view")
	mock.ExpectQuery(`SELECT permission FROM document_shares`).
		WithArgs(docID, sharedUserID).
		WillReturnRows(shareRows)

	canAccess, permission, err := service.CanAccess(docID, sharedUserID)
	if err != nil {
		t.Fatalf("CanAccess() error = %v", err)
	}

	if !canAccess {
		t.Error("Shared user should have access")
	}

	if permission != "view" {
		t.Errorf("permission = %q, want %q", permission, "view")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDocumentService_CanAccess_NoAccess(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	service := NewDocumentService(db, t.TempDir())

	docID := uuid.New()
	ownerID := uuid.New()
	otherUserID := uuid.New()

	getRows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "original_name", "size", "mime_type",
		"encryption_algo", "file_path", "is_encrypted", "created_at", "updated_at", "deleted_at",
	}).AddRow(docID, ownerID, "Test", "test.pdf", 1024, "application/pdf", "AES-256-GCM", "/path", false, time.Now(), time.Now(), nil)

	mock.ExpectQuery(`SELECT .+ FROM documents WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(docID).
		WillReturnRows(getRows)

	// Mock no share found
	mock.ExpectQuery(`SELECT permission FROM document_shares`).
		WithArgs(docID, otherUserID).
		WillReturnError(sql.ErrNoRows)

	canAccess, permission, err := service.CanAccess(docID, otherUserID)
	if err != nil {
		t.Fatalf("CanAccess() error = %v", err)
	}

	if canAccess {
		t.Error("Unshared user should not have access")
	}

	if permission != "" {
		t.Errorf("permission = %q, want empty string", permission)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDocumentService_Rename_Success(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	service := NewDocumentService(db, t.TempDir())

	docID := uuid.New()
	ownerID := uuid.New()
	newName := "New Document Name"

	// Mock GetByID for CanAccess
	getRows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "original_name", "size", "mime_type",
		"encryption_algo", "file_path", "is_encrypted", "created_at", "updated_at", "deleted_at",
	}).AddRow(docID, ownerID, "Old Name", "test.pdf", 1024, "application/pdf", "AES-256-GCM", "/path", false, time.Now(), time.Now(), nil)

	mock.ExpectQuery(`SELECT .+ FROM documents WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(docID).
		WillReturnRows(getRows)

	// Mock update
	mock.ExpectExec(`UPDATE documents SET name = \$1, updated_at = \$2 WHERE id = \$3`).
		WithArgs(newName, sqlmock.AnyArg(), docID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := service.Rename(docID, ownerID, newName)
	if err != nil {
		t.Fatalf("Rename() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDocumentService_Rename_NoEditPermission(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	service := NewDocumentService(db, t.TempDir())

	docID := uuid.New()
	ownerID := uuid.New()
	viewerID := uuid.New()

	// Mock GetByID for CanAccess
	getRows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "original_name", "size", "mime_type",
		"encryption_algo", "file_path", "is_encrypted", "created_at", "updated_at", "deleted_at",
	}).AddRow(docID, ownerID, "Old Name", "test.pdf", 1024, "application/pdf", "AES-256-GCM", "/path", false, time.Now(), time.Now(), nil)

	mock.ExpectQuery(`SELECT .+ FROM documents WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(docID).
		WillReturnRows(getRows)

	// Mock share with view-only permission
	shareRows := sqlmock.NewRows([]string{"permission"}).AddRow("view")
	mock.ExpectQuery(`SELECT permission FROM document_shares`).
		WithArgs(docID, viewerID).
		WillReturnRows(shareRows)

	err := service.Rename(docID, viewerID, "New Name")

	if err != ErrAccessDenied {
		t.Errorf("Rename() with view permission error = %v, want ErrAccessDenied", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDocumentService_GetFilePath_Owner(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	service := NewDocumentService(db, t.TempDir())

	docID := uuid.New()
	ownerID := uuid.New()
	expectedPath := "/uploads/test-file"

	getRows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "original_name", "size", "mime_type",
		"encryption_algo", "file_path", "is_encrypted", "created_at", "updated_at", "deleted_at",
	}).AddRow(docID, ownerID, "Test", "test.pdf", 1024, "application/pdf", "AES-256-GCM", expectedPath, false, time.Now(), time.Now(), nil)

	mock.ExpectQuery(`SELECT .+ FROM documents WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(docID).
		WillReturnRows(getRows)

	filePath, err := service.GetFilePath(docID, ownerID)
	if err != nil {
		t.Fatalf("GetFilePath() error = %v", err)
	}

	if filePath != expectedPath {
		t.Errorf("filePath = %q, want %q", filePath, expectedPath)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDocumentService_GetFilePath_NoAccess(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	service := NewDocumentService(db, t.TempDir())

	docID := uuid.New()
	ownerID := uuid.New()
	otherUserID := uuid.New()

	getRows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "original_name", "size", "mime_type",
		"encryption_algo", "file_path", "is_encrypted", "created_at", "updated_at", "deleted_at",
	}).AddRow(docID, ownerID, "Test", "test.pdf", 1024, "application/pdf", "AES-256-GCM", "/path", false, time.Now(), time.Now(), nil)

	mock.ExpectQuery(`SELECT .+ FROM documents WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(docID).
		WillReturnRows(getRows)

	// Mock no share
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM document_shares`).
		WithArgs(docID, otherUserID).
		WillReturnRows(countRows)

	_, err := service.GetFilePath(docID, otherUserID)

	if err != ErrAccessDenied {
		t.Errorf("GetFilePath() without access error = %v, want ErrAccessDenied", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDocumentService_RemoveShare_Success(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	service := NewDocumentService(db, t.TempDir())

	docID := uuid.New()
	ownerID := uuid.New()
	sharedWithID := uuid.New()

	// Mock GetByID
	getRows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "original_name", "size", "mime_type",
		"encryption_algo", "file_path", "is_encrypted", "created_at", "updated_at", "deleted_at",
	}).AddRow(docID, ownerID, "Test", "test.pdf", 1024, "application/pdf", "AES-256-GCM", "/path", false, time.Now(), time.Now(), nil)

	mock.ExpectQuery(`SELECT .+ FROM documents WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(docID).
		WillReturnRows(getRows)

	// Mock delete share
	mock.ExpectExec(`DELETE FROM document_shares WHERE document_id = \$1 AND shared_with_id = \$2`).
		WithArgs(docID, sharedWithID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := service.RemoveShare(docID, ownerID, sharedWithID)
	if err != nil {
		t.Fatalf("RemoveShare() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDocumentService_RemoveShare_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	service := NewDocumentService(db, t.TempDir())

	docID := uuid.New()
	ownerID := uuid.New()
	sharedWithID := uuid.New()

	// Mock GetByID
	getRows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "original_name", "size", "mime_type",
		"encryption_algo", "file_path", "is_encrypted", "created_at", "updated_at", "deleted_at",
	}).AddRow(docID, ownerID, "Test", "test.pdf", 1024, "application/pdf", "AES-256-GCM", "/path", false, time.Now(), time.Now(), nil)

	mock.ExpectQuery(`SELECT .+ FROM documents WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(docID).
		WillReturnRows(getRows)

	// Mock delete share - no rows affected
	mock.ExpectExec(`DELETE FROM document_shares WHERE document_id = \$1 AND shared_with_id = \$2`).
		WithArgs(docID, sharedWithID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := service.RemoveShare(docID, ownerID, sharedWithID)

	if err != ErrShareNotFound {
		t.Errorf("RemoveShare() for non-existent share error = %v, want ErrShareNotFound", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}
