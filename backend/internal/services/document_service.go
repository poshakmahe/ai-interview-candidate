package services

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/katim/secure-doc-vault/internal/database"
	"github.com/katim/secure-doc-vault/internal/models"
	"github.com/katim/secure-doc-vault/pkg/utils"
)

var (
	ErrDocumentNotFound = errors.New("document not found")
	ErrAccessDenied     = errors.New("access denied")
	ErrShareNotFound    = errors.New("share not found")
)

type DocumentService struct {
	db        *database.DB
	uploadDir string
}

func NewDocumentService(db *database.DB, uploadDir string) *DocumentService {
	// Ensure upload directory exists
	os.MkdirAll(uploadDir, 0755)
	return &DocumentService{db: db, uploadDir: uploadDir}
}

func (s *DocumentService) Create(ownerID uuid.UUID, name, originalName, mimeType string, size int64, fileData io.Reader) (*models.Document, error) {
	// Validate content type
	if err := utils.ValidateContentType(mimeType); err != nil {
		return nil, fmt.Errorf("invalid file type: %w", err)
	}

	// Sanitize filenames to prevent path traversal and other attacks
	sanitizedOriginalName, err := utils.SanitizeFilename(originalName)
	if err != nil {
		return nil, fmt.Errorf("invalid filename: %w", err)
	}

	sanitizedName := name
	if name != "" {
		sanitizedName, err = utils.SanitizeFilename(name)
		if err != nil {
			return nil, fmt.Errorf("invalid document name: %w", err)
		}
	} else {
		sanitizedName = sanitizedOriginalName
	}

	doc := &models.Document{
		ID:             uuid.New(),
		OwnerID:        ownerID,
		Name:           sanitizedName,
		OriginalName:   sanitizedOriginalName,
		Size:           size,
		MimeType:       mimeType,
		EncryptionAlgo: "AES-256-GCM",
		IsEncrypted:    false, // Encryption would be implemented in production
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Create file path using UUID only (never user input)
	doc.FilePath = filepath.Join(s.uploadDir, doc.ID.String())

	// Save file to disk first
	file, err := os.Create(doc.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, fileData); err != nil {
		os.Remove(doc.FilePath) // Clean up on failure
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Save to database (if this fails, file is cleaned up)
	_, err = s.db.Exec(
		`INSERT INTO documents (id, owner_id, name, original_name, size, mime_type, encryption_algo, file_path, is_encrypted, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		doc.ID, doc.OwnerID, doc.Name, doc.OriginalName, doc.Size, doc.MimeType,
		doc.EncryptionAlgo, doc.FilePath, doc.IsEncrypted, doc.CreatedAt, doc.UpdatedAt,
	)
	if err != nil {
		os.Remove(doc.FilePath) // Clean up file if DB insert fails
		return nil, fmt.Errorf("failed to save document metadata: %w", err)
	}

	return doc, nil
}

func (s *DocumentService) GetByID(id uuid.UUID) (*models.Document, error) {
	doc := &models.Document{}
	err := s.db.QueryRow(
		`SELECT id, owner_id, name, original_name, size, mime_type, encryption_algo, file_path, is_encrypted, created_at, updated_at, deleted_at
		 FROM documents WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&doc.ID, &doc.OwnerID, &doc.Name, &doc.OriginalName, &doc.Size, &doc.MimeType,
		&doc.EncryptionAlgo, &doc.FilePath, &doc.IsEncrypted, &doc.CreatedAt, &doc.UpdatedAt, &doc.DeletedAt)

	if err == sql.ErrNoRows {
		return nil, ErrDocumentNotFound
	}
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *DocumentService) GetByOwner(ownerID uuid.UUID, page, perPage int) ([]models.Document, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	// Get total count
	var total int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM documents WHERE owner_id = $1 AND deleted_at IS NULL`,
		ownerID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get documents
	rows, err := s.db.Query(
		`SELECT id, owner_id, name, original_name, size, mime_type, encryption_algo, file_path, is_encrypted, created_at, updated_at
		 FROM documents WHERE owner_id = $1 AND deleted_at IS NULL
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		ownerID, perPage, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var documents []models.Document
	for rows.Next() {
		var doc models.Document
		if err := rows.Scan(&doc.ID, &doc.OwnerID, &doc.Name, &doc.OriginalName, &doc.Size, &doc.MimeType,
			&doc.EncryptionAlgo, &doc.FilePath, &doc.IsEncrypted, &doc.CreatedAt, &doc.UpdatedAt); err != nil {
			return nil, 0, err
		}
		documents = append(documents, doc)
	}

	return documents, total, nil
}

func (s *DocumentService) GetSharedWithUser(userID uuid.UUID, page, perPage int) ([]models.DocumentResponse, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	// Get total count
	var total int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM document_shares ds
		 JOIN documents d ON ds.document_id = d.id
		 WHERE ds.shared_with_id = $1 AND d.deleted_at IS NULL
		 AND (ds.expires_at IS NULL OR ds.expires_at > NOW())`,
		userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get shared documents with owner info
	rows, err := s.db.Query(
		`SELECT d.id, d.owner_id, d.name, d.original_name, d.size, d.mime_type, d.encryption_algo,
		        d.is_encrypted, d.created_at, d.updated_at, u.name as owner_name, ds.permission
		 FROM document_shares ds
		 JOIN documents d ON ds.document_id = d.id
		 JOIN users u ON d.owner_id = u.id
		 WHERE ds.shared_with_id = $1 AND d.deleted_at IS NULL
		 AND (ds.expires_at IS NULL OR ds.expires_at > NOW())
		 ORDER BY ds.created_at DESC LIMIT $2 OFFSET $3`,
		userID, perPage, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var documents []models.DocumentResponse
	for rows.Next() {
		var doc models.DocumentResponse
		var permission string
		if err := rows.Scan(&doc.ID, &doc.OwnerID, &doc.Name, &doc.OriginalName, &doc.Size, &doc.MimeType,
			&doc.EncryptionAlgo, &doc.IsEncrypted, &doc.CreatedAt, &doc.UpdatedAt, &doc.OwnerName, &permission); err != nil {
			return nil, 0, err
		}
		documents = append(documents, doc)
	}

	return documents, total, nil
}

func (s *DocumentService) Delete(id, userID uuid.UUID) error {
	// Check ownership
	doc, err := s.GetByID(id)
	if err != nil {
		return err
	}
	if doc.OwnerID != userID {
		return ErrAccessDenied
	}

	// Mark as deleted in database (soft delete for referential integrity)
	now := time.Now()
	_, err = s.db.Exec(
		`UPDATE documents SET deleted_at = $1 WHERE id = $2`,
		now, id,
	)
	if err != nil {
		return err
	}

	// Delete the actual file from disk
	// Note: If this fails, file remains but document is marked deleted
	// A cleanup job could handle orphaned files
	if err := os.Remove(doc.FilePath); err != nil && !os.IsNotExist(err) {
		// Log the error but don't fail the operation
		// In production, you'd want proper logging here
		fmt.Printf("Warning: failed to delete file %s: %v\n", doc.FilePath, err)
	}

	return nil
}

func (s *DocumentService) Share(documentID, ownerID uuid.UUID, sharedWithEmail, permission string) error {
	// Verify ownership
	doc, err := s.GetByID(documentID)
	if err != nil {
		return err
	}
	if doc.OwnerID != ownerID {
		return ErrAccessDenied
	}

	// Get shared user
	var sharedWithID uuid.UUID
	err = s.db.QueryRow(`SELECT id FROM users WHERE email = $1`, sharedWithEmail).Scan(&sharedWithID)
	if err == sql.ErrNoRows {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}

	// Create share
	_, err = s.db.Exec(
		`INSERT INTO document_shares (document_id, shared_by_id, shared_with_id, permission)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (document_id, shared_with_id) DO UPDATE SET permission = $4`,
		documentID, ownerID, sharedWithID, permission,
	)
	return err
}

func (s *DocumentService) RemoveShare(documentID, ownerID, sharedWithID uuid.UUID) error {
	// Verify ownership
	doc, err := s.GetByID(documentID)
	if err != nil {
		return err
	}
	if doc.OwnerID != ownerID {
		return ErrAccessDenied
	}

	result, err := s.db.Exec(
		`DELETE FROM document_shares WHERE document_id = $1 AND shared_with_id = $2`,
		documentID, sharedWithID,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrShareNotFound
	}

	return nil
}

func (s *DocumentService) GetFilePath(id, userID uuid.UUID) (string, error) {
	doc, err := s.GetByID(id)
	if err != nil {
		return "", err
	}

	// Check if user is owner
	if doc.OwnerID == userID {
		return doc.FilePath, nil
	}

	// Check if document is shared with user
	var count int
	err = s.db.QueryRow(
		`SELECT COUNT(*) FROM document_shares
		 WHERE document_id = $1 AND shared_with_id = $2
		 AND (expires_at IS NULL OR expires_at > NOW())`,
		id, userID,
	).Scan(&count)
	if err != nil {
		return "", err
	}
	if count == 0 {
		return "", ErrAccessDenied
	}

	return doc.FilePath, nil
}

func (s *DocumentService) CanAccess(documentID, userID uuid.UUID) (bool, string, error) {
	doc, err := s.GetByID(documentID)
	if err != nil {
		return false, "", err
	}

	// Owner has full access
	if doc.OwnerID == userID {
		return true, "owner", nil
	}

	// Check share permissions
	var permission string
	err = s.db.QueryRow(
		`SELECT permission FROM document_shares
		 WHERE document_id = $1 AND shared_with_id = $2
		 AND (expires_at IS NULL OR expires_at > NOW())`,
		documentID, userID,
	).Scan(&permission)
	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}

	return true, permission, nil
}

func (s *DocumentService) Rename(id, userID uuid.UUID, newName string) error {
	canAccess, permission, err := s.CanAccess(id, userID)
	if err != nil {
		return err
	}
	if !canAccess || (permission != "owner" && permission != "edit") {
		return ErrAccessDenied
	}

	_, err = s.db.Exec(
		`UPDATE documents SET name = $1, updated_at = $2 WHERE id = $3`,
		newName, time.Now(), id,
	)
	return err
}
