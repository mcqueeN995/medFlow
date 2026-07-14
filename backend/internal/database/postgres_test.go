package database

import (
	"testing"

	"github.com/medflow/backend/internal/config"
)

func TestNewPostgres_Success(t *testing.T) {
	cfg := config.DatabaseConfig{
		User:     "medflow",
		Password: "medflow_secret",
		DBName:   "medflow_db",
		Host:     "localhost",
		Port:     "5433",
	}

	pool, err := NewPostgres(cfg)
	if err != nil {
		t.Fatalf("NewPostgres() error = %v", err)
	}
	defer pool.Close()

	if pool.Stat().TotalConns() == 0 {
		t.Error("Expected at least one connection in pool")
	}
}

func TestNewPostgres_InvalidCredentials(t *testing.T) {
	cfg := config.DatabaseConfig{
		User:     "wrong_user",
		Password: "wrong_password",
		DBName:   "medflow_db",
		Host:     "localhost",
		Port:     "5433",
	}

	_, err := NewPostgres(cfg)
	if err == nil {
		t.Fatal("Expected error for invalid credentials, got nil")
	}
}

func TestNewPostgres_UnreachableHost(t *testing.T) {
	cfg := config.DatabaseConfig{
		User:     "medflow",
		Password: "medflow_secret",
		DBName:   "medflow_db",
		Host:     "localhost",
		Port:     "9999",
	}

	_, err := NewPostgres(cfg)
	if err == nil {
		t.Fatal("Expected error for unreachable host, got nil")
	}
}
