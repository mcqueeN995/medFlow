package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/medflow/backend/internal/models"
)

func RequireRole(allowedRoles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleStr, exists := c.Get(ContextUserRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": map[string]interface{}{
					"code":    "UNAUTHORIZED",
					"message": "authentication required",
				},
			})
			return
		}

		userRole := models.UserRole(roleStr.(string))

		for _, allowed := range allowedRoles {
			if userRole == allowed {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": map[string]interface{}{
				"code":    "FORBIDDEN",
				"message": "insufficient permissions",
			},
		})
	}
}
