package utils

import (
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  error
	}{
		{
			name:     "valid simple filename",
			input:    "document.pdf",
			expected: "document.pdf",
			wantErr:  nil,
		},
		{
			name:     "filename with spaces",
			input:    "my document.pdf",
			expected: "my document.pdf",
			wantErr:  nil,
		},
		{
			name:     "path traversal attack - relative path",
			input:    "../../../etc/passwd",
			expected: "passwd",
			wantErr:  nil,
		},
		{
			name:     "path traversal attack - absolute path",
			input:    "/etc/passwd",
			expected: "passwd",
			wantErr:  nil,
		},
		{
			name:     "path traversal attack - windows style backslashes removed",
			input:    "..\\..\\windows\\system32\\config",
			expected: "....windowssystem32config", // On Unix, backslash is not a path separator, so filepath.Base keeps the string, then backslashes are removed
			wantErr:  nil,
		},
		{
			name:     "filename with null byte",
			input:    "document\x00.pdf",
			expected: "document.pdf",
			wantErr:  nil,
		},
		{
			name:     "empty filename",
			input:    "",
			expected: "",
			wantErr:  ErrInvalidFilename,
		},
		{
			name:     "dot only",
			input:    ".",
			expected: "",
			wantErr:  ErrInvalidFilename,
		},
		{
			name:     "double dot only",
			input:    "..",
			expected: "",
			wantErr:  ErrInvalidFilename,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: "",
			wantErr:  ErrInvalidFilename,
		},
		{
			name:     "filename with leading/trailing spaces",
			input:    "  document.pdf  ",
			expected: "document.pdf",
			wantErr:  nil,
		},
		{
			name:     "hidden file",
			input:    ".gitignore",
			expected: ".gitignore",
			wantErr:  nil,
		},
		{
			name:     "filename with multiple extensions",
			input:    "document.tar.gz",
			expected: "document.tar.gz",
			wantErr:  nil,
		},
		{
			name:     "unicode filename",
			input:    "文档.pdf",
			expected: "文档.pdf",
			wantErr:  nil,
		},
		{
			name:     "embedded path separators",
			input:    "folder/subfolder/file.txt",
			expected: "file.txt",
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizeFilename(tt.input)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("SanitizeFilename(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("SanitizeFilename(%q) unexpected error = %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename_LongFilename(t *testing.T) {
	// Test filename longer than 255 bytes
	longName := strings.Repeat("a", 300) + ".pdf"
	result, err := SanitizeFilename(longName)

	if err != nil {
		t.Errorf("SanitizeFilename(long filename) unexpected error = %v", err)
		return
	}

	if len(result) > 255 {
		t.Errorf("SanitizeFilename(long filename) result length = %d, want <= 255", len(result))
	}

	// Verify extension is preserved
	if !strings.HasSuffix(result, ".pdf") {
		t.Errorf("SanitizeFilename(long filename) should preserve extension, got %q", result)
	}
}

func TestSanitizeFilename_VeryLongExtension(t *testing.T) {
	// Extension so long that base would be < 1 character
	longExt := "a." + strings.Repeat("x", 260)
	_, err := SanitizeFilename(longExt)

	if err != ErrInvalidFilename {
		t.Errorf("SanitizeFilename(very long extension) error = %v, want ErrInvalidFilename", err)
	}
}

func TestValidateContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		wantErr     error
	}{
		// Valid document types
		{name: "PDF", contentType: "application/pdf", wantErr: nil},
		{name: "Word doc", contentType: "application/msword", wantErr: nil},
		{name: "Word docx", contentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", wantErr: nil},
		{name: "Excel xls", contentType: "application/vnd.ms-excel", wantErr: nil},
		{name: "Excel xlsx", contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", wantErr: nil},
		{name: "PowerPoint ppt", contentType: "application/vnd.ms-powerpoint", wantErr: nil},
		{name: "PowerPoint pptx", contentType: "application/vnd.openxmlformats-officedocument.presentationml.presentation", wantErr: nil},

		// Valid text types
		{name: "Plain text", contentType: "text/plain", wantErr: nil},
		{name: "CSV", contentType: "text/csv", wantErr: nil},
		{name: "Markdown", contentType: "text/markdown", wantErr: nil},
		{name: "JSON", contentType: "application/json", wantErr: nil},
		{name: "XML application", contentType: "application/xml", wantErr: nil},
		{name: "XML text", contentType: "text/xml", wantErr: nil},

		// Valid image types
		{name: "JPEG", contentType: "image/jpeg", wantErr: nil},
		{name: "PNG", contentType: "image/png", wantErr: nil},
		{name: "GIF", contentType: "image/gif", wantErr: nil},
		{name: "WebP", contentType: "image/webp", wantErr: nil},
		{name: "SVG", contentType: "image/svg+xml", wantErr: nil},

		// Valid archive types
		{name: "ZIP", contentType: "application/zip", wantErr: nil},
		{name: "ZIP compressed", contentType: "application/x-zip-compressed", wantErr: nil},
		{name: "GZIP", contentType: "application/gzip", wantErr: nil},
		{name: "TAR", contentType: "application/x-tar", wantErr: nil},

		// Content type with charset parameter
		{name: "text with charset", contentType: "text/plain; charset=utf-8", wantErr: nil},
		{name: "JSON with charset", contentType: "application/json; charset=utf-8", wantErr: nil},

		// Content type with extra spaces
		{name: "content type with spaces", contentType: "  text/plain  ", wantErr: nil},

		// Invalid types
		{name: "executable", contentType: "application/x-executable", wantErr: ErrInvalidContentType},
		{name: "HTML", contentType: "text/html", wantErr: ErrInvalidContentType},
		{name: "JavaScript", contentType: "application/javascript", wantErr: ErrInvalidContentType},
		{name: "shell script", contentType: "application/x-sh", wantErr: ErrInvalidContentType},
		{name: "octet-stream", contentType: "application/octet-stream", wantErr: ErrInvalidContentType},
		{name: "empty", contentType: "", wantErr: ErrInvalidContentType},
		{name: "random string", contentType: "not-a-mime-type", wantErr: ErrInvalidContentType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContentType(tt.contentType)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("ValidateContentType(%q) error = %v, wantErr = %v", tt.contentType, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateContentType(%q) unexpected error = %v", tt.contentType, err)
			}
		})
	}
}

func TestValidateContentType_CaseInsensitive(t *testing.T) {
	tests := []string{
		"APPLICATION/PDF",
		"Application/Pdf",
		"TEXT/PLAIN",
		"Image/JPEG",
	}

	for _, ct := range tests {
		t.Run(ct, func(t *testing.T) {
			err := ValidateContentType(ct)
			if err != nil {
				t.Errorf("ValidateContentType(%q) should be case-insensitive, got error = %v", ct, err)
			}
		})
	}
}

func TestValidateFileSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int64
		maxSize int64
		wantErr error
	}{
		{
			name:    "valid size within limit",
			size:    1024,
			maxSize: 10485760, // 10MB
			wantErr: nil,
		},
		{
			name:    "size equals max",
			size:    10485760,
			maxSize: 10485760,
			wantErr: nil,
		},
		{
			name:    "size exceeds max",
			size:    10485761,
			maxSize: 10485760,
			wantErr: ErrFileTooLarge,
		},
		{
			name:    "zero size",
			size:    0,
			maxSize: 10485760,
			wantErr: ErrFileTooLarge,
		},
		{
			name:    "negative size",
			size:    -1,
			maxSize: 10485760,
			wantErr: ErrFileTooLarge,
		},
		{
			name:    "1 byte file",
			size:    1,
			maxSize: 10485760,
			wantErr: nil,
		},
		{
			name:    "very small max size",
			size:    100,
			maxSize: 50,
			wantErr: ErrFileTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileSize(tt.size, tt.maxSize)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("ValidateFileSize(%d, %d) error = %v, wantErr = %v", tt.size, tt.maxSize, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateFileSize(%d, %d) unexpected error = %v", tt.size, tt.maxSize, err)
			}
		})
	}
}

// Benchmark tests
func BenchmarkSanitizeFilename(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SanitizeFilename("../../../etc/passwd")
	}
}

func BenchmarkValidateContentType(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ValidateContentType("application/pdf; charset=utf-8")
	}
}

func BenchmarkValidateFileSize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ValidateFileSize(1024, 10485760)
	}
}
