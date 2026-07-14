package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/medflow/backend/internal/service"
)

type ErrorResponse struct {
	Error struct {
		Code    string        `json:"code"`
		Message string        `json:"message"`
		Details []ErrorDetail `json:"details,omitempty"`
	} `json:"error"`
}

type ErrorDetail struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

type BanResponse struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		BanReason string `json:"ban_reason"`
		BannedAt  string `json:"banned_at"`
	} `json:"error"`
}

func RespondWithError(c *gin.Context, statusCode int, code, message string, details []ErrorDetail) {
	resp := ErrorResponse{}
	resp.Error.Code = code
	resp.Error.Message = message
	if details != nil {
		resp.Error.Details = details
	}
	c.JSON(statusCode, resp)
}

func RespondWithBanError(c *gin.Context, banErr *service.ErrUserBannedWithDetails) {
	resp := BanResponse{}
	resp.Error.Code = "BANNED"
	resp.Error.Message = "user is banned"
	resp.Error.BanReason = banErr.BanReason
	resp.Error.BannedAt = banErr.BannedAt.Format("2006-01-02T15:04:05Z07:00")
	c.JSON(http.StatusForbidden, resp)
}

func MapServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEmailExists):
		RespondWithError(c, http.StatusConflict, "EMAIL_EXISTS", "email already registered", nil)

	case errors.Is(err, service.ErrNicknameExists):
		RespondWithError(c, http.StatusConflict, "NICKNAME_EXISTS", "nickname already taken", nil)

	case errors.Is(err, service.ErrInvalidCreds):
		RespondWithError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password", nil)

	case errors.Is(err, service.ErrTokenInvalid):
		RespondWithError(c, http.StatusUnauthorized, "INVALID_TOKEN", "invalid refresh token", nil)

	case errors.Is(err, service.ErrTokenExpired):
		RespondWithError(c, http.StatusUnauthorized, "TOKEN_EXPIRED", "refresh token expired", nil)

	case errors.Is(err, service.ErrTokenCompromised):
		RespondWithError(c, http.StatusUnauthorized, "TOKEN_COMPROMISED", "token reuse detected - please login again", nil)

	default:
		RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

func MapBanError(c *gin.Context, err error) bool {
	var banErr *service.ErrUserBannedWithDetails
	if errors.As(err, &banErr) {
		RespondWithBanError(c, banErr)
		return true
	}
	return false
}
