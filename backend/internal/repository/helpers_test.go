package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/database"
	"github.com/medflow/backend/internal/models"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	cfg := config.DatabaseConfig{
		User:     "medflow",
		Password: "medflow_secret",
		DBName:   "medflow_db",
		Host:     "localhost",
		Port:     "5433",
	}

	pool, err := database.NewPostgres(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

// createTestForumUser - как createTestUser в token_repo_test.go, но сама
// регистрирует cleanup через t.Cleanup (используется несколькими файлами
// forum-тестов, где вручную дублировать defer неудобно).
func createTestForumUser(t *testing.T, pool *pgxpool.Pool) *models.User {
	t.Helper()
	ctx := context.Background()
	user := createTestUser(t, NewUserRepo(pool), ctx)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", user.ID)
	})
	return user
}
