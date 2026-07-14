package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery перехватывает паники, логирует их и возвращает стандартный 500 ответ
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stackTrace := string(debug.Stack())
				requestID := c.GetString("request_id")

				slog.Error("panic recovered",
					"error", err,
					"stack", stackTrace,
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
					"request_id", requestID,
				)

				// Формат ошибки строго по OpenAPI спецификации
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": map[string]interface{}{
						"code":    "INTERNAL_ERROR",
						"message": "internal server error",
					},
				})
			}
		}()

		c.Next()
	}
}
