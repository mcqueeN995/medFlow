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
	"github.com/medflow/backend/internal/pkg/email"
	"github.com/medflow/backend/internal/pkg/llm"
	"github.com/medflow/backend/internal/pkg/queue"
	"github.com/medflow/backend/internal/pkg/ratelimit"
	"github.com/medflow/backend/internal/pkg/storage"
	"github.com/medflow/backend/internal/repository"
	"github.com/medflow/backend/internal/service"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	cfg    *config.Config
	http   *http.Server
	pool   *pgxpool.Pool
	router http.Handler
	queue  *queue.Client
	redis  *redis.Client
}

func New(cfg *config.Config) (*Server, error) {
	pool, err := database.NewPostgres(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	slog.Info("connected to postgres")

	s3Client, err := storage.New(cfg.S3)
	if err != nil {
		return nil, fmt.Errorf("connect to s3: %w", err)
	}

	llmProvider, err := llm.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("init llm provider: %w", err)
	}
	// Эмбеддинги карточек всегда идут через Ollama, независимо от того, кем
	// сконфигурирован llmProvider выше — см. llm.ErrEmbedNotSupported.
	embedProvider := llm.NewOllamaProvider(cfg.Ollama.Host, cfg.Ollama.GenerationModel, cfg.Ollama.EmbeddingModel)

	redisAddr := fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port)
	queueClient := queue.New(redisAddr)
	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	loginGuard := ratelimit.NewLoginGuard(redisClient)

	userRepo := repository.NewUserRepo(pool)
	tokenRepo := repository.NewTokenRepo(pool)
	loginChangeRepo := repository.NewLoginChangeRepo(pool)
	passwordResetRepo := repository.NewPasswordResetRepo(pool)
	threadRepo := repository.NewThreadRepo(pool)
	commentRepo := repository.NewCommentRepo(pool)
	reactionRepo := repository.NewReactionRepo(pool)
	reportRepo := repository.NewReportRepo(pool)
	textbookRepo := repository.NewTextbookRepo(pool)
	uploadRepo := repository.NewUploadRepo(pool)
	cardTaskRepo := repository.NewCardTaskRepo(pool)
	cardRepo := repository.NewCardRepo(pool)
	cardProgressRepo := repository.NewCardProgressRepo(pool)
	cardFavoriteRepo := repository.NewCardFavoriteRepo(pool)
	cardRatingRepo := repository.NewCardRatingRepo(pool)
	textbookChunkRepo := repository.NewTextbookChunkRepo(pool)
	poiRepo := repository.NewPOIRepo(pool)
	auditLogRepo := repository.NewAuditLogRepo(pool)
	adminStatsRepo := repository.NewAdminStatsRepo(pool)
	pushRepo := repository.NewPushRepo(pool)

	tokenService := service.NewTokenService(cfg)
	emailSender := email.NewSender(cfg.Email)
	authService := service.NewAuthService(userRepo, tokenRepo, tokenService, cfg, passwordResetRepo, emailSender)
	pushSender := service.NewWebPushSender()
	pushService := service.NewPushService(pushRepo, pushSender, cfg.VAPID)
	forumService := service.NewForumService(threadRepo, commentRepo, reactionRepo, reportRepo, auditLogRepo, pushService)
	userService := service.NewUserService(userRepo, tokenRepo, auditLogRepo, loginChangeRepo, emailSender)
	libraryService := service.NewLibraryService(textbookRepo, uploadRepo, s3Client, auditLogRepo)
	uploadService := service.NewUploadService(uploadRepo, s3Client)
	cardService := service.NewCardService(
		cardTaskRepo, cardRepo, cardProgressRepo, cardFavoriteRepo, cardRatingRepo, textbookChunkRepo, textbookRepo, uploadRepo, reportRepo,
		s3Client, llmProvider, embedProvider, queueClient, pushService,
	)
	poiService := service.NewPOIService(poiRepo, auditLogRepo)
	adminService := service.NewAdminService(reportRepo, auditLogRepo, adminStatsRepo, threadRepo, commentRepo, cardRepo)

	authHandler := handler.NewAuthHandler(authService, loginGuard)
	forumHandler := handler.NewForumHandler(forumService)
	userHandler := handler.NewUserHandler(userService)
	libraryHandler := handler.NewLibraryHandler(libraryService)
	uploadHandler := handler.NewUploadHandler(uploadService)
	cardHandler := handler.NewCardHandler(cardService)
	poiHandler := handler.NewPOIHandler(poiService)
	adminHandler := handler.NewAdminHandler(adminService)
	pushHandler := handler.NewPushHandler(pushService)

	router := SetupRouter(cfg, redisClient, authHandler, forumHandler, userHandler, libraryHandler, uploadHandler, cardHandler, poiHandler, adminHandler, pushHandler)

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
		queue:  queueClient,
		redis:  redisClient,
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

	if err := s.queue.Close(); err != nil {
		slog.Error("queue client close error", "error", err)
	}

	if err := s.redis.Close(); err != nil {
		slog.Error("redis client close error", "error", err)
	}

	// Закрываем пул БД
	s.pool.Close()
	slog.Info("server stopped gracefully")

	return nil
}
