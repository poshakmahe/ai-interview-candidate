package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // Never expose password in JSON
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Document struct {
	ID             uuid.UUID  `json:"id"`
	OwnerID        uuid.UUID  `json:"owner_id"`
	Name           string     `json:"name"`
	OriginalName   string     `json:"original_name"`
	Size           int64      `json:"size"`
	MimeType       string     `json:"mime_type"`
	EncryptionKey  string     `json:"-"` // Never expose encryption key
	EncryptionAlgo string     `json:"encryption_algo"`
	FilePath       string     `json:"-"` // Internal path, not exposed
	IsEncrypted    bool       `json:"is_encrypted"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

type DocumentShare struct {
	ID           uuid.UUID  `json:"id"`
	DocumentID   uuid.UUID  `json:"document_id"`
	SharedByID   uuid.UUID  `json:"shared_by_id"`
	SharedWithID uuid.UUID  `json:"shared_with_id"`
	Permission   string     `json:"permission"` // "view" or "edit"
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Request/Response DTOs
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required,min=2"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type ShareRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Permission string `json:"permission" binding:"required,oneof=view edit"`
}

type DocumentResponse struct {
	Document
	OwnerName  string `json:"owner_name,omitempty"`
	SharedWith []SharedUserInfo `json:"shared_with,omitempty"`
}

type SharedUserInfo struct {
	Email      string `json:"email"`
	Name       string `json:"name"`
	Permission string `json:"permission"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalPages int         `json:"total_pages"`
}
