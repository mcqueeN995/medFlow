package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		requestID := c.GetString("request_id")

		var logFunc func(msg string, args ...any)
		switch {
		case status >= 500:
			logFunc = slog.Error
		case status >= 400:
			logFunc = slog.Warn
		default:
			logFunc = slog.Info
		}

		// Логируем
		logFunc("request",
			"request_id", requestID,
			"method", method,
			"path", path,
			"query", query,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"client_ip", clientIP,
			"bytes_in", c.Request.ContentLength,
			"bytes_out", c.Writer.Size(),
		)
	}
}
