package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/config"
)

// NewPostgres создает и настраивает пул подключений к PostgreSQL
func NewPostgres(cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}

	poolConfig.MaxConns = 20                      // Максимум соединений
	poolConfig.MinConns = 5                       // Минимум соединений
	poolConfig.MaxConnLifetime = 30 * time.Minute // Пересоздавать соединение каждые 30 мин
	poolConfig.MaxConnIdleTime = 5 * time.Minute  // Закрывать простое через 5 мин

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	slog.Info("database connection pool established",
		"host", cfg.Host,
		"port", cfg.Port,
		"db", cfg.DBName,
		"max_conns", poolConfig.MaxConns,
	)

	return pool, nil
}
