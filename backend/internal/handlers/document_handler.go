package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/katim/secure-doc-vault/internal/middleware"
	"github.com/katim/secure-doc-vault/internal/models"
	"github.com/katim/secure-doc-vault/internal/services"
)

type DocumentHandler struct {
	documentService *services.DocumentService
	maxFileSize     int64
}

func NewDocumentHandler(documentService *services.DocumentService, maxFileSize int64) *DocumentHandler {
	return &DocumentHandler{
		documentService: documentService,
		maxFileSize:     maxFileSize,
	}
}

// ListDocuments godoc
// @Summary List user's documents
// @Description Get paginated list of documents owned by the current user
// @Tags documents
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} models.PaginatedResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /documents [get]
func (h *DocumentHandler) ListDocuments(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	documents, total, err := h.documentService.GetByOwner(userID, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch documents",
		})
		return
	}

	totalPages := (total + perPage - 1) / perPage
	c.JSON(http.StatusOK, models.PaginatedResponse{
		Data:       documents,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	})
}

// UploadDocument godoc
// @Summary Upload a new document
// @Description Upload a file to the secure vault
// @Tags documents
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formance file true "File to upload"
// @Param name formance string false "Document name"
// @Success 201 {object} models.Document
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 413 {object} models.ErrorResponse
// @Router /documents [post]
func (h *DocumentHandler) UploadDocument(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "file_required",
			Message: "A file is required",
		})
		return
	}
	defer file.Close()

	// Check file size
	if header.Size > h.maxFileSize {
		c.JSON(http.StatusRequestEntityTooLarge, models.ErrorResponse{
			Error:   "file_too_large",
			Message: "File exceeds maximum allowed size",
		})
		return
	}

	// Get document name (use original filename if not provided)
	name := c.PostForm("name")
	if name == "" {
		name = header.Filename
	}

	document, err := h.documentService.Create(
		userID,
		name,
		header.Filename,
		header.Header.Get("Content-Type"),
		header.Size,
		file,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "upload_failed",
			Message: "Failed to upload document",
		})
		return
	}

	c.JSON(http.StatusCreated, document)
}

// GetDocument godoc
// @Summary Get document details
// @Description Get details of a specific document
// @Tags documents
// @Security BearerAuth
// @Produce json
// @Param id path string true "Document ID"
// @Success 200 {object} models.Document
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /documents/{id} [get]
func (h *DocumentHandler) GetDocument(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
		return
	}

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid document ID",
		})
		return
	}

	canAccess, _, err := h.documentService.CanAccess(docID, userID)
	if err != nil {
		if errors.Is(err, services.ErrDocumentNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Document not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal_error"})
		return
	}

	if !canAccess {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "access_denied",
			Message: "You don't have access to this document",
		})
		return
	}

	document, err := h.documentService.GetByID(docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal_error"})
		return
	}

	c.JSON(http.StatusOK, document)
}

// DownloadDocument godoc
// @Summary Download a document
// @Description Download the file content of a document
// @Tags documents
// @Security BearerAuth
// @Produce octet-stream
// @Param id path string true "Document ID"
// @Success 200 {file} binary
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /documents/{id}/download [get]
func (h *DocumentHandler) DownloadDocument(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
		return
	}

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid document ID",
		})
		return
	}

	filePath, err := h.documentService.GetFilePath(docID, userID)
	if err != nil {
		if errors.Is(err, services.ErrDocumentNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "not_found"})
			return
		}
		if errors.Is(err, services.ErrAccessDenied) {
			c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "access_denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal_error"})
		return
	}

	document, _ := h.documentService.GetByID(docID)
	c.Header("Content-Disposition", "attachment; filename="+document.OriginalName)
	c.Header("Content-Type", document.MimeType)
	c.File(filePath)
}

// DeleteDocument godoc
// @Summary Delete a document
// @Description Soft delete a document (owner only)
// @Tags documents
// @Security BearerAuth
// @Param id path string true "Document ID"
// @Success 204 "No Content"
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /documents/{id} [delete]
func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
		return
	}

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid document ID",
		})
		return
	}

	err = h.documentService.Delete(docID, userID)
	if err != nil {
		if errors.Is(err, services.ErrDocumentNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "not_found"})
			return
		}
		if errors.Is(err, services.ErrAccessDenied) {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "access_denied",
				Message: "Only the document owner can delete it",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal_error"})
		return
	}

	c.Status(http.StatusNoContent)
}

// ShareDocument godoc
// @Summary Share a document
// @Description Share a document with another user
// @Tags documents
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Document ID"
// @Param request body models.ShareRequest true "Share details"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /documents/{id}/share [post]
func (h *DocumentHandler) ShareDocument(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
		return
	}

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid document ID",
		})
		return
	}

	var req models.ShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	err = h.documentService.Share(docID, userID, req.Email, req.Permission)
	if err != nil {
		if errors.Is(err, services.ErrDocumentNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "document_not_found"})
			return
		}
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "user_not_found",
				Message: "User with this email not found",
			})
			return
		}
		if errors.Is(err, services.ErrAccessDenied) {
			c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "access_denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document shared successfully"})
}

// ListSharedDocuments godoc
// @Summary List shared documents
// @Description Get documents shared with the current user
// @Tags documents
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} models.PaginatedResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /shared [get]
func (h *DocumentHandler) ListSharedDocuments(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	documents, total, err := h.documentService.GetSharedWithUser(userID, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch shared documents",
		})
		return
	}

	totalPages := (total + perPage - 1) / perPage
	c.JSON(http.StatusOK, models.PaginatedResponse{
		Data:       documents,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	})
}

// RenameDocument godoc
// @Summary Rename a document
// @Description Update the name of a document
// @Tags documents
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Document ID"
// @Param request body object{name=string} true "New name"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Router /documents/{id} [patch]
func (h *DocumentHandler) RenameDocument(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
		return
	}

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid document ID",
		})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	err = h.documentService.Rename(docID, userID, req.Name)
	if err != nil {
		if errors.Is(err, services.ErrDocumentNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "not_found"})
			return
		}
		if errors.Is(err, services.ErrAccessDenied) {
			c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "access_denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document renamed successfully"})
}
