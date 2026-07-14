package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
	"github.com/medflow/backend/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	resp, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// Login POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		if MapBanError(c, err) {
			return
		}
		MapServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Refresh POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	resp, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if MapBanError(c, err) {
			return
		}
		MapServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Logout POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	err := h.authService.Logout(c.Request.Context(), req.RefreshToken)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "logout failed", nil)
		return
	}

	c.Status(http.StatusNoContent)
}

func GetUserID(c *gin.Context) string {
	if val, exists := c.Get("user_id"); exists {
		return val.(string)
	}
	return ""
}

func GetUserRole(c *gin.Context) models.UserRole {
	if val, exists := c.Get("user_role"); exists {
		return models.UserRole(val.(string))
	}
	return ""
}
