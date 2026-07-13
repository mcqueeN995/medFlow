package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/middleware"
)

type Server struct {
	cfg    *config.Config
	router *gin.Engine
	http   *http.Server
}

// New создает новый сервер
func New(cfg *config.Config) *Server {
	// Настраиваем Gin
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	router.Use(middleware.RequestID())
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	router.Use(middleware.RateLimiter(middleware.DefaultRateLimiterConfig()))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.App.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		cfg:    cfg,
		router: router,
		http:   httpServer,
	}
}

// Run запускает сервер с graceful shutdown
func (s *Server) Run() error {
	// Канал для сигналов ОС
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("starting server",
			"port", s.cfg.App.Port,
			"env", s.cfg.App.Env,
		)

		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Ждем сигнал завершения
	<-quit
	slog.Info("shutting down server...")

	// Graceful shutdown с таймаутом 30 секунд
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	slog.Info("server stopped gracefully")
	return nil
}
