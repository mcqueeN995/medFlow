package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/service"
)

type PushHandler struct {
	pushService *service.PushService
}

func NewPushHandler(pushService *service.PushService) *PushHandler {
	return &PushHandler{pushService: pushService}
}

func MapPushServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrPushSubscriptionNotFound):
		RespondWithError(c, http.StatusNotFound, "NOT_FOUND", "push subscription not found", nil)
	default:
		RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

// Subscribe POST /api/v1/push/subscribe
func (h *PushHandler) Subscribe(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var req dto.SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	id, err := h.pushService.Subscribe(c.Request.Context(), userID, req)
	if err != nil {
		MapPushServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// Unsubscribe DELETE /api/v1/push/unsubscribe?endpoint=...
func (h *PushHandler) Unsubscribe(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	endpoint := c.Query("endpoint")
	if endpoint == "" {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "endpoint is required", nil)
		return
	}

	if err := h.pushService.Unsubscribe(c.Request.Context(), userID, endpoint); err != nil {
		MapPushServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// UpdatePreferences PATCH /api/v1/push/preferences
func (h *PushHandler) UpdatePreferences(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var req dto.PushPreferences
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	prefs, err := h.pushService.UpdatePreferences(c.Request.Context(), userID, req)
	if err != nil {
		MapPushServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, prefs)
}
