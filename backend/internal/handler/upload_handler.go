package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/medflow/backend/internal/service"
)

type UploadHandler struct {
	uploadService *service.UploadService
}

func NewUploadHandler(uploadService *service.UploadService) *UploadHandler {
	return &UploadHandler{uploadService: uploadService}
}

func MapUploadServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidUploadType):
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid upload type", nil)
	case errors.Is(err, service.ErrInvalidFileType):
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid file type for this upload type", nil)
	default:
		RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

// Upload POST /api/v1/upload?type=pdf|image|avatar
func (h *UploadHandler) Upload(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	uploadType := c.Query("type")
	if uploadType == "" {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "type query param is required", nil)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "file is required", nil)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "could not read file", nil)
		return
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	res, err := h.uploadService.Upload(c.Request.Context(), userID, uploadType, fileHeader.Filename, contentType, fileHeader.Size, file)
	if err != nil {
		MapUploadServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}
