// Команда seed создаёт тестовых пользователей для локальной разработки.
// Безопасна для запуска в docker-compose при каждом `up`: при APP_ENV=production
// ничего не делает (см. isProdEnv), поэтому тестовые аккаунты никогда не
// попадают в прод-окружение, даже если сервис случайно останется в compose.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/database"
	"github.com/medflow/backend/internal/models"
	"github.com/medflow/backend/internal/pkg/password"
)

type seedUser struct {
	email    string
	password string
	nickname string
	role     models.UserRole
}

// Фиксированные dev-аккаунты с известными паролями — только для отладки.
var seedUsers = []seedUser{
	{"admin@medflow.dev", "password123", "dev_admin", models.RoleAdmin},
	{"moderator@medflow.dev", "password123", "dev_moderator", models.RoleModerator},
	{"user@medflow.dev", "password123", "dev_user", models.RoleUser},
}

func isProdEnv(env string) bool {
	switch env {
	case "production", "prod":
		return true
	default:
		return false
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if isProdEnv(cfg.App.Env) {
		slog.Info("skipping test user seed: production environment", "app_env", cfg.App.Env)
		return
	}

	pool, err := database.NewPostgres(cfg.Database)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for _, su := range seedUsers {
		hash, err := password.Hash(su.password)
		if err != nil {
			slog.Error("failed to hash password", "email", su.email, "error", err)
			os.Exit(1)
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO users (id, email, password_hash, nickname, role, email_verified_at)
			VALUES ($1, $2, $3, $4, $5, now())
			ON CONFLICT (email) DO UPDATE
			SET password_hash = EXCLUDED.password_hash,
			    role = EXCLUDED.role,
			    email_verified_at = now(),
			    banned_at = NULL,
			    ban_reason = NULL,
			    banned_by = NULL,
			    deleted_at = NULL
		`, uuid.New(), su.email, hash, su.nickname, su.role)
		if err != nil {
			slog.Error("failed to seed user", "email", su.email, "error", err)
			os.Exit(1)
		}

		slog.Info("seeded test user", "email", su.email, "role", su.role)
	}

	fmt.Println("Test accounts (dev only, password for all: password123):")
	for _, su := range seedUsers {
		fmt.Printf("  %-9s %s\n", su.role, su.email)
	}
}
