package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stackTrace := string(debug.Stack())

				slog.Error("panic recovered",
					"error", err,
					"stack", stackTrace,
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
					"request_id", c.GetString("request_id"),
				)

				// Возвращаем 500
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
			}
		}()

		c.Next()
	}
}
