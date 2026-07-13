package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{
			"http://localhost:5173", // Vite dev server
			"http://localhost:3000", // альтернативный порт
			"https://medflow.local", // production (пример)
		},
		AllowedMethods: []string{
			"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
		},
		AllowedHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID",
		},
		AllowCredentials: true,
	}
}

// CORS middleware с whitelist доменов
func CORS(config CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Проверяем, есть ли origin в whitelist
		allowed := false
		for _, allowedOrigin := range config.AllowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}

		// Если origin разрешен - устанавливаем заголовки
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)

			if config.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}

			c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
			c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
			c.Header("Access-Control-Max-Age", "86400") // 24 часа
		}

		// Обрабатываем preflight запросы
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
