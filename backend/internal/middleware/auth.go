package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/pkg/jwt"
)

const (
	ContextUserID   = "user_id"
	ContextUserRole = "user_role"
)

func AuthRequired(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": map[string]interface{}{
					"code":    "UNAUTHORIZED",
					"message": "authorization header is required",
				},
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": map[string]interface{}{
					"code":    "UNAUTHORIZED",
					"message": "invalid authorization format, use: Bearer <token>",
				},
			})
			return
		}

		tokenString := parts[1]

		claims, err := jwt.ParseAccess(tokenString, cfg.JWT.AccessSecret)
		if err != nil {
			if errors.Is(err, jwt.ErrExpiredToken) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": map[string]interface{}{
						"code":    "TOKEN_EXPIRED",
						"message": "access token has expired",
					},
				})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": map[string]interface{}{
					"code":    "INVALID_TOKEN",
					"message": "invalid access token",
				},
			})
			return
		}

		c.Set(ContextUserID, claims.UserID.String())
		c.Set(ContextUserRole, claims.Role)

		ctx := context.WithValue(c.Request.Context(), ContextUserID, claims.UserID.String())
		ctx = context.WithValue(ctx, ContextUserRole, claims.Role)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func GetUserID(c *gin.Context) string {
	if val, exists := c.Get(ContextUserID); exists {
		return val.(string)
	}
	return ""
}

func GetUserRole(c *gin.Context) string {
	if val, exists := c.Get(ContextUserRole); exists {
		return val.(string)
	}
	return ""
}
