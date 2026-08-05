package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func MapUserServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		RespondWithError(c, http.StatusNotFound, "NOT_FOUND", "user not found", nil)
	case errors.Is(err, service.ErrNicknameExists):
		RespondWithError(c, http.StatusConflict, "NICKNAME_EXISTS", "nickname already taken", nil)
	case errors.Is(err, service.ErrInvalidCreds):
		RespondWithError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid password", nil)
	default:
		RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

// Me GET /api/v1/users/me
func (h *UserHandler) Me(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	profile, err := h.userService.Me(c.Request.Context(), userID)
	if err != nil {
		MapUserServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

// UpdateMe PATCH /api/v1/users/me
func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	profile, err := h.userService.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		MapUserServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

// DeleteMe DELETE /api/v1/users/me?password=...
func (h *UserHandler) DeleteMe(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	pw := c.Query("password")
	if pw == "" {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "password is required", nil)
		return
	}

	if err := h.userService.DeleteAccount(c.Request.Context(), userID, pw); err != nil {
		MapUserServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// PublicProfile GET /api/v1/users/:id
func (h *UserHandler) PublicProfile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id", nil)
		return
	}
	profile, err := h.userService.PublicProfile(c.Request.Context(), id)
	if err != nil {
		MapUserServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, profile)
}
