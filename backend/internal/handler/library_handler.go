package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
	"github.com/medflow/backend/internal/service"
)

type LibraryHandler struct {
	libraryService *service.LibraryService
}

func NewLibraryHandler(libraryService *service.LibraryService) *LibraryHandler {
	return &LibraryHandler{libraryService: libraryService}
}

func MapLibraryServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrTextbookNotFound):
		RespondWithError(c, http.StatusNotFound, "NOT_FOUND", "textbook not found", nil)
	case errors.Is(err, service.ErrNotDownloadable):
		RespondWithError(c, http.StatusForbidden, "FORBIDDEN", "download is only available for category A", nil)
	case errors.Is(err, service.ErrNoSourceLink):
		RespondWithError(c, http.StatusNotFound, "NOT_FOUND", "textbook has no source link", nil)
	case errors.Is(err, service.ErrPDFFileRequired):
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "pdf_file_id is required for storage_type A", nil)
	case errors.Is(err, service.ErrSourceURLRequired):
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "source_url is required for storage_type B", nil)
	case errors.Is(err, service.ErrPDFUploadNotFound):
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "referenced pdf upload not found or expired", nil)
	case errors.Is(err, service.ErrPDFUploadWrongType):
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "referenced upload is not a pdf", nil)
	default:
		RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

// ==================== PUBLIC ====================

// ListTextbooks GET /api/v1/library/textbooks
func (h *LibraryHandler) ListTextbooks(c *gin.Context) {
	var q struct {
		Q           string `form:"q"`
		Subject     string `form:"subject"`
		Course      int    `form:"course"`
		Department  string `form:"department"`
		StorageType string `form:"storage_type"`
		Sort        string `form:"sort"`
		Page        int    `form:"page,default=1"`
		Limit       int    `form:"limit,default=20"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid query params", nil)
		return
	}

	filter := models.TextbookListFilter{Sort: q.Sort, Page: q.Page, Limit: q.Limit}
	if q.Q != "" {
		filter.Query = &q.Q
	}
	if q.Subject != "" {
		filter.Subject = &q.Subject
	}
	if q.Course != 0 {
		filter.Course = &q.Course
	}
	if q.Department != "" {
		filter.Department = &q.Department
	}
	if q.StorageType != "" {
		st := models.TextbookStorageType(q.StorageType)
		filter.StorageType = &st
	}

	pagination, items, err := h.libraryService.ListTextbooks(c.Request.Context(), filter)
	if err != nil {
		MapLibraryServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"pagination": pagination, "data": items})
}

// GetTextbook GET /api/v1/library/textbooks/:id
func (h *LibraryHandler) GetTextbook(c *gin.Context) {
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	textbook, err := h.libraryService.GetTextbook(c.Request.Context(), id)
	if err != nil {
		MapLibraryServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, textbook)
}

// Download GET /api/v1/library/textbooks/:id/download
func (h *LibraryHandler) Download(c *gin.Context) {
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	url, err := h.libraryService.Download(c.Request.Context(), id)
	if err != nil {
		MapLibraryServiceError(c, err)
		return
	}
	c.Redirect(http.StatusFound, url)
}

// Source GET /api/v1/library/textbooks/:id/source
func (h *LibraryHandler) Source(c *gin.Context) {
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	url, err := h.libraryService.Source(c.Request.Context(), id)
	if err != nil {
		MapLibraryServiceError(c, err)
		return
	}
	c.Redirect(http.StatusFound, url)
}

// ==================== ADMIN ====================

// AdminListTextbooks GET /api/v1/admin/library/textbooks
func (h *LibraryHandler) AdminListTextbooks(c *gin.Context) {
	var q struct {
		StorageType   string `form:"storage_type"`
		IncludeHidden bool   `form:"include_hidden,default=false"`
		Page          int    `form:"page,default=1"`
		Limit         int    `form:"limit,default=20"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid query params", nil)
		return
	}

	filter := models.AdminTextbookListFilter{IncludeHidden: q.IncludeHidden, Page: q.Page, Limit: q.Limit}
	if q.StorageType != "" {
		st := models.TextbookStorageType(q.StorageType)
		filter.StorageType = &st
	}

	pagination, items, err := h.libraryService.AdminListTextbooks(c.Request.Context(), filter)
	if err != nil {
		MapLibraryServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"pagination": pagination, "data": items})
}

// AdminCreateTextbook POST /api/v1/admin/library/textbooks
func (h *LibraryHandler) AdminCreateTextbook(c *gin.Context) {
	actorID, ok := requireUserID(c)
	if !ok {
		return
	}
	var req dto.CreateTextbookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	textbook, err := h.libraryService.AdminCreateTextbook(c.Request.Context(), actorID, req)
	if err != nil {
		MapLibraryServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, textbook)
}

// AdminUpdateTextbook PATCH /api/v1/admin/library/textbooks/:id
func (h *LibraryHandler) AdminUpdateTextbook(c *gin.Context) {
	actorID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}

	var req dto.UpdateTextbookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	textbook, err := h.libraryService.AdminUpdateTextbook(c.Request.Context(), actorID, id, req)
	if err != nil {
		MapLibraryServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, textbook)
}

// AdminDeleteTextbook DELETE /api/v1/admin/library/textbooks/:id
func (h *LibraryHandler) AdminDeleteTextbook(c *gin.Context) {
	actorID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	if err := h.libraryService.AdminDeleteTextbook(c.Request.Context(), actorID, id); err != nil {
		MapLibraryServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
