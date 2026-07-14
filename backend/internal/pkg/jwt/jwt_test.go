package jwt

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGenerateAccess(t *testing.T) {
	userID := uuid.New()
	role := "user"
	secret := "test_secret_key"
	expire := 15 * time.Minute

	token, err := GenerateAccess(userID, role, secret, expire)
	if err != nil {
		t.Fatalf("GenerateAccess() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateAccess() returned empty token")
	}

	// Проверяем, что токен состоит из 3 частей (header.payload.signature)
	parts := len(token)
	if parts == 0 {
		t.Error("GenerateAccess() returned invalid token format")
	}
}

func TestGenerateRefresh(t *testing.T) {
	tokenID := uuid.New()
	userID := uuid.New()
	secret := "test_secret_key"
	expire := 30 * 24 * time.Hour // 30 дней

	token, err := GenerateRefresh(tokenID, userID, secret, expire)
	if err != nil {
		t.Fatalf("GenerateRefresh() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateRefresh() returned empty token")
	}
}

func TestParseAccess(t *testing.T) {
	userID := uuid.New()
	role := "admin"
	secret := "test_secret_key"
	expire := 15 * time.Minute

	// Генерируем токен
	token, err := GenerateAccess(userID, role, secret, expire)
	if err != nil {
		t.Fatalf("GenerateAccess() error = %v", err)
	}

	// Парсим токен
	claims, err := ParseAccess(token, secret)
	if err != nil {
		t.Fatalf("ParseAccess() error = %v", err)
	}

	// Проверяем claims
	if claims.UserID != userID {
		t.Errorf("ParseAccess() userID = %v, want %v", claims.UserID, userID)
	}
	if claims.Role != role {
		t.Errorf("ParseAccess() role = %v, want %v", claims.Role, role)
	}
}

func TestParseRefresh(t *testing.T) {
	tokenID := uuid.New()
	userID := uuid.New()
	secret := "test_secret_key"
	expire := 30 * 24 * time.Hour

	// Генерируем токен
	token, err := GenerateRefresh(tokenID, userID, secret, expire)
	if err != nil {
		t.Fatalf("GenerateRefresh() error = %v", err)
	}

	// Парсим токен
	claims, err := ParseRefresh(token, secret)
	if err != nil {
		t.Fatalf("ParseRefresh() error = %v", err)
	}

	// Проверяем claims
	if claims.TokenID != tokenID {
		t.Errorf("ParseRefresh() tokenID = %v, want %v", claims.TokenID, tokenID)
	}
	if claims.UserID != userID {
		t.Errorf("ParseRefresh() userID = %v, want %v", claims.UserID, userID)
	}
}

func TestParseAccess_ExpiredToken(t *testing.T) {
	userID := uuid.New()
	role := "user"
	secret := "test_secret_key"
	expire := -1 * time.Minute // уже истек

	token, err := GenerateAccess(userID, role, secret, expire)
	if err != nil {
		t.Fatalf("GenerateAccess() error = %v", err)
	}

	_, err = ParseAccess(token, secret)
	if err != ErrExpiredToken {
		t.Errorf("ParseAccess() error = %v, want %v", err, ErrExpiredToken)
	}
}

func TestParseAccess_InvalidSecret(t *testing.T) {
	userID := uuid.New()
	role := "user"
	secret := "test_secret_key"
	wrongSecret := "wrong_secret"
	expire := 15 * time.Minute

	token, err := GenerateAccess(userID, role, secret, expire)
	if err != nil {
		t.Fatalf("GenerateAccess() error = %v", err)
	}

	_, err = ParseAccess(token, wrongSecret)
	if err != ErrInvalidToken {
		t.Errorf("ParseAccess() error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestParseAccess_InvalidToken(t *testing.T) {
	secret := "test_secret_key"

	_, err := ParseAccess("invalid_token", secret)
	if err != ErrInvalidToken {
		t.Errorf("ParseAccess() error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestParseRefresh_ExpiredToken(t *testing.T) {
	tokenID := uuid.New()
	userID := uuid.New()
	secret := "test_secret_key"
	expire := -1 * time.Minute

	token, err := GenerateRefresh(tokenID, userID, secret, expire)
	if err != nil {
		t.Fatalf("GenerateRefresh() error = %v", err)
	}

	_, err = ParseRefresh(token, secret)
	if err != ErrExpiredToken {
		t.Errorf("ParseRefresh() error = %v, want %v", err, ErrExpiredToken)
	}
}
