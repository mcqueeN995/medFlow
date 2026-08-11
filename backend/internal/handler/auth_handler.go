package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
	"github.com/medflow/backend/internal/service"
)

// LoginRateLimiter - узкий интерфейс над ratelimit.LoginGuard (см.
// internal/pkg/ratelimit), нужен только чтобы AuthHandler можно было
// тестировать без реального Redis - по тому же принципу, что и узкие
// репозиторные интерфейсы в service/interfaces.go.
type LoginRateLimiter interface {
	CheckLocked(ctx context.Context, ip, login string) (locked bool, retryAfter time.Duration, err error)
	RecordFailure(ctx context.Context, ip, login string) error
	Reset(ctx context.Context, ip, login string) error
}

type AuthHandler struct {
	authService *service.AuthService
	loginGuard  LoginRateLimiter
}

func NewAuthHandler(authService *service.AuthService, loginGuard LoginRateLimiter) *AuthHandler {
	return &AuthHandler{authService: authService, loginGuard: loginGuard}
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

// Login POST /api/v1/auth/login. Защищён от подбора пароля/логина
// LoginGuard'ом (см. internal/pkg/ratelimit): 5 неудачных попыток за 15
// минут на пару (IP, логин) блокируют 6-ю попытку 429-м, до истечения окна.
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	ip := c.ClientIP()
	if locked, retryAfter, err := h.loginGuard.CheckLocked(c.Request.Context(), ip, req.Login); err == nil && locked {
		c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
		RespondWithError(c, http.StatusTooManyRequests, "TOO_MANY_ATTEMPTS", "too many failed login attempts, try again later", nil)
		return
	}
	// err != nil (Redis недоступен) - fail-open, пропускаем к обычному входу.

	resp, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCreds) {
			_ = h.loginGuard.RecordFailure(c.Request.Context(), ip, req.Login)
		}
		if MapBanError(c, err) {
			return
		}
		MapServiceError(c, err)
		return
	}

	_ = h.loginGuard.Reset(c.Request.Context(), ip, req.Login)
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

// RequestPasswordReset POST /api/v1/auth/password-reset
func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	var req dto.RequestPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	if err := h.authService.RequestPasswordReset(c.Request.Context(), req.Login); err != nil {
		MapServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "if the account exists, a reset code has been sent"})
}

// ConfirmPasswordReset POST /api/v1/auth/password-reset/confirm
func (h *AuthHandler) ConfirmPasswordReset(c *gin.Context) {
	var req dto.ConfirmPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	if err := h.authService.ConfirmPasswordReset(c.Request.Context(), req.Code, req.NewPassword); err != nil {
		MapServiceError(c, err)
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
