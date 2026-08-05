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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/database"
	"github.com/medflow/backend/internal/handler"
	"github.com/medflow/backend/internal/repository"
	"github.com/medflow/backend/internal/service"
)

type Server struct {
	cfg    *config.Config
	http   *http.Server
	pool   *pgxpool.Pool
	router http.Handler
}

func New(cfg *config.Config) (*Server, error) {
	pool, err := database.NewPostgres(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	slog.Info("connected to postgres")

	userRepo := repository.NewUserRepo(pool)
	tokenRepo := repository.NewTokenRepo(pool)
	threadRepo := repository.NewThreadRepo(pool)
	commentRepo := repository.NewCommentRepo(pool)
	reactionRepo := repository.NewReactionRepo(pool)
	reportRepo := repository.NewReportRepo(pool)

	tokenService := service.NewTokenService(cfg)
	authService := service.NewAuthService(userRepo, tokenRepo, tokenService, cfg)
	forumService := service.NewForumService(threadRepo, commentRepo, reactionRepo, reportRepo)
	userService := service.NewUserService(userRepo, tokenRepo)

	authHandler := handler.NewAuthHandler(authService)
	forumHandler := handler.NewForumHandler(forumService)
	userHandler := handler.NewUserHandler(userService)

	router := SetupRouter(cfg, authHandler, forumHandler, userHandler)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.App.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		cfg:    cfg,
		http:   httpServer,
		pool:   pool,
		router: router,
	}, nil
}

func (s *Server) Run() error {
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

	<-quit
	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.http.Shutdown(ctx); err != nil {
		slog.Error("http shutdown error", "error", err)
	}

	// Закрываем пул БД
	s.pool.Close()
	slog.Info("server stopped gracefully")

	return nil
}
