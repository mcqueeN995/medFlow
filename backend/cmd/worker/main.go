package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/hibiken/asynq"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/database"
	"github.com/medflow/backend/internal/pkg/llm"
	"github.com/medflow/backend/internal/pkg/storage"
	"github.com/medflow/backend/internal/repository"
	"github.com/medflow/backend/internal/service"
	"github.com/medflow/backend/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	pool, err := database.NewPostgres(cfg.Database)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	s3Client, err := storage.New(cfg.S3)
	if err != nil {
		slog.Error("failed to init s3 client", "error", err)
		os.Exit(1)
	}

	llmProvider, err := llm.New(cfg)
	if err != nil {
		slog.Error("failed to init llm provider", "error", err)
		os.Exit(1)
	}

	cardTaskRepo := repository.NewCardTaskRepo(pool)
	cardRepo := repository.NewCardRepo(pool)
	cardProgressRepo := repository.NewCardProgressRepo(pool)
	textbookChunkRepo := repository.NewTextbookChunkRepo(pool)
	textbookRepo := repository.NewTextbookRepo(pool)
	uploadRepo := repository.NewUploadRepo(pool)
	reportRepo := repository.NewReportRepo(pool)
	pushRepo := repository.NewPushRepo(pool)
	pushService := service.NewPushService(pushRepo, service.NewWebPushSender(), cfg.VAPID)

	// enqueuer воркеру не нужен (он только выполняет задачи, не создаёт новые) -
	// nil безопасен, т.к. CardService.ProcessTask его не использует.
	cardService := service.NewCardService(
		cardTaskRepo, cardRepo, cardProgressRepo, textbookChunkRepo, textbookRepo, uploadRepo, reportRepo,
		s3Client, llmProvider, nil, pushService,
	)
	cardTaskHandler := worker.NewCardTaskHandler(cardService)

	redisAddr := fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port)

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"cards":         6,
				"notifications": 3,
				"email":         1,
			},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(service.TaskTypeGenerateCards, cardTaskHandler.HandleGenerate)

	slog.Info("starting asynq worker", "redis_addr", redisAddr)
	if err := srv.Run(mux); err != nil {
		slog.Error("worker error", "error", err)
		os.Exit(1)
	}
}
