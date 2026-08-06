package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
	"github.com/medflow/backend/internal/service"
)

type CardHandler struct {
	cardService *service.CardService
}

func NewCardHandler(cardService *service.CardService) *CardHandler {
	return &CardHandler{cardService: cardService}
}

func MapCardServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCardTaskNotFound), errors.Is(err, service.ErrCardNotFound), errors.Is(err, service.ErrTextbookNotFound):
		RespondWithError(c, http.StatusNotFound, "NOT_FOUND", "not found", nil)
	case errors.Is(err, service.ErrForbidden):
		RespondWithError(c, http.StatusForbidden, "FORBIDDEN", "you are not the owner of this task", nil)
	case errors.Is(err, service.ErrTooManyActiveTasks):
		RespondWithError(c, http.StatusTooManyRequests, "RATE_LIMITED", "too many active card tasks", nil)
	case errors.Is(err, service.ErrPDFUploadNotFound):
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "referenced pdf upload not found or expired", nil)
	case errors.Is(err, service.ErrPDFUploadWrongType):
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "referenced upload is not a pdf", nil)
	default:
		RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

// CreateTask POST /api/v1/cards/tasks
func (h *CardHandler) CreateTask(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req dto.CreateCardTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	task, err := h.cardService.CreateTask(c.Request.Context(), userID, req)
	if err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, task)
}

// ListTasks GET /api/v1/cards/tasks
func (h *CardHandler) ListTasks(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var q struct {
		Status string `form:"status"`
		Page   int    `form:"page,default=1"`
		Limit  int    `form:"limit,default=20"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid query params", nil)
		return
	}

	var status *models.CardTaskStatus
	if q.Status != "" {
		st := models.CardTaskStatus(q.Status)
		status = &st
	}

	pagination, items, err := h.cardService.ListTasks(c.Request.Context(), userID, status, q.Page, q.Limit)
	if err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"pagination": pagination, "data": items})
}

// GetTask GET /api/v1/cards/tasks/:id
func (h *CardHandler) GetTask(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}

	task, err := h.cardService.GetTask(c.Request.Context(), userID, id)
	if err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

// ListTaskCards GET /api/v1/cards/tasks/:id/cards
func (h *CardHandler) ListTaskCards(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}

	var q struct {
		Page  int `form:"page,default=1"`
		Limit int `form:"limit,default=50"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid query params", nil)
		return
	}

	pagination, items, err := h.cardService.ListTaskCards(c.Request.Context(), userID, id, q.Page, q.Limit)
	if err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"pagination": pagination, "data": items})
}

// Review GET /api/v1/cards/review
func (h *CardHandler) Review(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var q struct {
		Limit int `form:"limit,default=20"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid query params", nil)
		return
	}

	items, count, err := h.cardService.Review(c.Request.Context(), userID, q.Limit)
	if err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "count": count})
}

// RateCard POST /api/v1/cards/:id/rate
func (h *CardHandler) RateCard(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}

	var req dto.RateCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	info, err := h.cardService.RateCard(c.Request.Context(), userID, id, req.Grade)
	if err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, info)
}

// ReportCard POST /api/v1/cards/:id/report
func (h *CardHandler) ReportCard(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}

	var req dto.ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	report, err := h.cardService.ReportCard(c.Request.Context(), userID, id, req.Reason)
	if err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, report)
}

// Stats GET /api/v1/cards/stats
func (h *CardHandler) Stats(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	stats, err := h.cardService.Stats(c.Request.Context(), userID)
	if err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, stats)
}
