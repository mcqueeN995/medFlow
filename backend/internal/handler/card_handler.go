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
	case errors.Is(err, service.ErrCardTaskNotFound), errors.Is(err, service.ErrCardNotFound), errors.Is(err, service.ErrTextbookNotFound),
		errors.Is(err, service.ErrCardFavoriteNotFound), errors.Is(err, service.ErrCardRatingNotFound):
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

// ==================== Лента каталога ====================

// ListCatalogFeed GET /api/v1/cards/catalog
func (h *CardHandler) ListCatalogFeed(c *gin.Context) {
	var q struct {
		Q          string `form:"q"`
		TextbookID string `form:"textbook_id"`
		Difficulty string `form:"difficulty"`
		Page       int    `form:"page,default=1"`
		Limit      int    `form:"limit,default=20"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid query params", nil)
		return
	}

	var qPtr, textbookIDPtr *string
	var difficultyPtr *models.CardDifficulty
	if q.Q != "" {
		qPtr = &q.Q
	}
	if q.TextbookID != "" {
		textbookIDPtr = &q.TextbookID
	}
	if q.Difficulty != "" {
		d := models.CardDifficulty(q.Difficulty)
		difficultyPtr = &d
	}

	pagination, items, err := h.cardService.ListCatalogFeed(c.Request.Context(), qPtr, textbookIDPtr, difficultyPtr, q.Page, q.Limit)
	if err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"pagination": pagination, "data": items})
}

// ==================== Избранное ====================

// FavoriteCard POST /api/v1/cards/:id/favorite
func (h *CardHandler) FavoriteCard(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	if err := h.cardService.FavoriteCard(c.Request.Context(), userID, id); err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// UnfavoriteCard DELETE /api/v1/cards/:id/favorite
func (h *CardHandler) UnfavoriteCard(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	if err := h.cardService.UnfavoriteCard(c.Request.Context(), userID, id); err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListFavorites GET /api/v1/cards/favorites
func (h *CardHandler) ListFavorites(c *gin.Context) {
	userID, ok := requireUserID(c)
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

	pagination, items, err := h.cardService.ListFavorites(c.Request.Context(), userID, q.Page, q.Limit)
	if err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"pagination": pagination, "data": items})
}

// ReviewFavorites GET /api/v1/cards/favorites/review
func (h *CardHandler) ReviewFavorites(c *gin.Context) {
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

	items, count, err := h.cardService.ReviewFavorites(c.Request.Context(), userID, q.Limit)
	if err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "count": count})
}

// ==================== Рейтинг звёзд ====================

// RateCardStars POST /api/v1/cards/:id/stars
func (h *CardHandler) RateCardStars(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}

	var req dto.RateCardStarsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	if err := h.cardService.RateCardStars(c.Request.Context(), userID, id, req.Stars); err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// RemoveCardRating DELETE /api/v1/cards/:id/stars
func (h *CardHandler) RemoveCardRating(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	if err := h.cardService.RemoveCardRating(c.Request.Context(), userID, id); err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListRatedCards GET /api/v1/cards/rated
func (h *CardHandler) ListRatedCards(c *gin.Context) {
	userID, ok := requireUserID(c)
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

	pagination, items, err := h.cardService.ListRatedCards(c.Request.Context(), userID, q.Page, q.Limit)
	if err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"pagination": pagination, "data": items})
}

// ==================== Шеринг ====================

// ShareTask POST /api/v1/cards/tasks/:id/share
func (h *CardHandler) ShareTask(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	resp, err := h.cardService.ShareTask(c.Request.Context(), userID, id)
	if err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// UnshareTask DELETE /api/v1/cards/tasks/:id/share
func (h *CardHandler) UnshareTask(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	if err := h.cardService.UnshareTask(c.Request.Context(), userID, id); err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// GetSharedTask GET /api/v1/cards/shared/:token - публичный, без авторизации.
func (h *CardHandler) GetSharedTask(c *gin.Context) {
	token := c.Param("token")
	resp, err := h.cardService.GetSharedTask(c.Request.Context(), token)
	if err != nil {
		MapCardServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
