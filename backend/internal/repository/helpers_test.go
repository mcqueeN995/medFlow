package repository

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/database"
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
