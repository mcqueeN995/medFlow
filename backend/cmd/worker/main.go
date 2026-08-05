package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/hibiken/asynq"
	"github.com/medflow/backend/internal/config"
)

// Скелет asynq-воркера: поднимает пул воркеров и graceful shutdown.
// Обработчики задач (генерация ИИ-карточек, письма, уведомления) добавляются
// в mux по мере реализации соответствующих модулей — см. Этап 3 плана.
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
	// mux.HandleFunc("cards:generate", cardHandler.HandleGenerate)

	slog.Info("starting asynq worker", "redis_addr", redisAddr)
	if err := srv.Run(mux); err != nil {
		slog.Error("worker error", "error", err)
		os.Exit(1)
	}
}
