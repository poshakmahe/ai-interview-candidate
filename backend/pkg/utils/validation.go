package utils

import (
	"errors"
	"path/filepath"
	"strings"
)

var (
	ErrInvalidFilename    = errors.New("invalid filename")
	ErrInvalidContentType = errors.New("invalid content type")
	ErrFileTooLarge       = errors.New("file too large")
)

// AllowedMIMETypes defines the whitelist of acceptable content types
var AllowedMIMETypes = map[string]bool{
	// Documents
	"application/pdf":                                                   true,
	"application/msword":                                                true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel":                                          true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"application/vnd.ms-powerpoint":                                     true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,

	// Text
	"text/plain":     true,
	"text/csv":       true,
	"text/markdown":  true,
	"application/json": true,
	"application/xml":  true,
	"text/xml":       true,

	// Images
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
	"image/svg+xml": true,

	// Archives
	"application/zip":         true,
	"application/x-zip-compressed": true,
	"application/gzip":        true,
	"application/x-tar":       true,
}

// SanitizeFilename removes potentially dangerous characters from filenames
// and prevents path traversal attacks
func SanitizeFilename(filename string) (string, error) {
	if filename == "" {
		return "", ErrInvalidFilename
	}

	// Get base name to prevent path traversal (e.g., "../../../etc/passwd")
	filename = filepath.Base(filename)

	// Remove any remaining path separators
	filename = strings.ReplaceAll(filename, "/", "")
	filename = strings.ReplaceAll(filename, "\\", "")

	// Remove null bytes
	filename = strings.ReplaceAll(filename, "\x00", "")

	// Trim whitespace
	filename = strings.TrimSpace(filename)

	// Check for empty result
	if filename == "" || filename == "." || filename == ".." {
		return "", ErrInvalidFilename
	}

	// Limit filename length (255 bytes is common filesystem limit)
	if len(filename) > 255 {
		// Keep extension intact
		ext := filepath.Ext(filename)
		maxBase := 255 - len(ext)
		if maxBase < 1 {
			return "", ErrInvalidFilename
		}
		base := filename[:maxBase]
		filename = base + ext
	}

	return filename, nil
}

// ValidateContentType checks if the MIME type is in the allowed list
func ValidateContentType(contentType string) error {
	// Remove parameters like charset (e.g., "text/plain; charset=utf-8")
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}

	contentType = strings.TrimSpace(strings.ToLower(contentType))

	if !AllowedMIMETypes[contentType] {
		return ErrInvalidContentType
	}

	return nil
}

// ValidateFileSize checks if file size is within acceptable limits
func ValidateFileSize(size, maxSize int64) error {
	if size <= 0 {
		return ErrFileTooLarge
	}
	if size > maxSize {
		return ErrFileTooLarge
	}
	return nil
}
