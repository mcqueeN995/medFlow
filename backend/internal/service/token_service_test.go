package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/models"
	"github.com/medflow/backend/internal/pkg/jwt"
)

func setupTestTokenService() (*TokenService, *config.Config) {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:  "test_access_secret",
			RefreshSecret: "test_refresh_secret",
			AccessExpire:  15 * time.Minute,
			RefreshExpire: 30 * 24 * time.Hour,
		},
	}
	return NewTokenService(cfg), cfg
}

func TestTokenService_GenerateAuthTokens_Success(t *testing.T) {
	service, _ := setupTestTokenService()
	ctx := context.Background()

	user := &models.User{
		ID:       uuid.New(),
		Email:    "test@medflow.local",
		Nickname: "testuser",
		Role:     models.RoleUser,
	}

	tokenSaved := false
	tokenRepo := &mockTokenRepository{
		saveFn: func(ctx context.Context, token *models.RefreshToken) error {
			tokenSaved = true
			if token.UserID != user.ID {
				t.Errorf("Save() UserID = %v, want %v", token.UserID, user.ID)
			}
			if token.TokenHash == "" {
				t.Error("Save() TokenHash is empty")
			}
			if token.ExpiresAt.Before(time.Now()) {
				t.Error("Save() ExpiresAt is in the past")
			}
			token.CreatedAt = time.Now()
			return nil
		},
	}

	resp, err := service.GenerateAuthTokens(ctx, user, tokenRepo)
	if err != nil {
		t.Fatalf("GenerateAuthTokens() error = %v", err)
	}

	if !tokenSaved {
		t.Error("TokenRepo.Save() was not called")
	}
	if resp.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
	if resp.RefreshToken == "" {
		t.Error("RefreshToken is empty")
	}
	if resp.User.Email != user.Email {
		t.Errorf("User.Email = %v, want %v", resp.User.Email, user.Email)
	}
	if resp.ExpiresIn != 900 { // 15 минут
		t.Errorf("ExpiresIn = %v, want 900", resp.ExpiresIn)
	}
}

func TestTokenService_GenerateAuthTokens_SaveError(t *testing.T) {
	service, _ := setupTestTokenService()
	ctx := context.Background()

	user := &models.User{
		ID:    uuid.New(),
		Email: "test@medflow.local",
		Role:  models.RoleUser,
	}

	expectedErr := errors.New("database error")
	tokenRepo := &mockTokenRepository{
		saveFn: func(ctx context.Context, token *models.RefreshToken) error {
			return expectedErr
		},
	}

	_, err := service.GenerateAuthTokens(ctx, user, tokenRepo)
	if !errors.Is(err, expectedErr) {
		t.Errorf("GenerateAuthTokens() error = %v, want %v", err, expectedErr)
	}
}

func TestTokenService_ParseRefreshToken_Success(t *testing.T) {
	service, cfg := setupTestTokenService()

	userID := uuid.New()
	tokenID := uuid.New()

	validToken, err := jwt.GenerateRefresh(
		tokenID,
		userID,
		cfg.JWT.RefreshSecret,
		30*24*time.Hour,
	)
	if err != nil {
		t.Fatalf("GenerateRefresh() error = %v", err)
	}

	claims, err := service.ParseRefreshToken(validToken)
	if err != nil {
		t.Fatalf("ParseRefreshToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.TokenID != tokenID {
		t.Errorf("TokenID = %v, want %v", claims.TokenID, tokenID)
	}
}

func TestTokenService_ParseRefreshToken_InvalidToken(t *testing.T) {
	service, _ := setupTestTokenService()

	_, err := service.ParseRefreshToken("invalid_token")
	if !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("ParseRefreshToken() error = %v, want %v", err, ErrTokenInvalid)
	}
}

func TestTokenService_ParseRefreshToken_ExpiredToken(t *testing.T) {
	service, cfg := setupTestTokenService()

	userID := uuid.New()
	tokenID := uuid.New()

	// Генерируем токен с отрицательным временем жизни (уже истек)
	expiredToken, err := jwt.GenerateRefresh(
		tokenID,
		userID,
		cfg.JWT.RefreshSecret,
		-1*time.Hour,
	)
	if err != nil {
		t.Fatalf("GenerateRefresh() error = %v", err)
	}

	_, err = service.ParseRefreshToken(expiredToken)
	if !errors.Is(err, ErrTokenExpired) {
		t.Errorf("ParseRefreshToken() error = %v, want %v", err, ErrTokenExpired)
	}
}

func TestTokenService_ParseRefreshToken_WrongSecret(t *testing.T) {
	service, _ := setupTestTokenService()

	userID := uuid.New()
	tokenID := uuid.New()

	// Генерируем токен с другим секретом
	wrongToken, _ := jwt.GenerateRefresh(
		tokenID,
		userID,
		"wrong_secret",
		30*24*time.Hour,
	)

	_, err := service.ParseRefreshToken(wrongToken)
	if !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("ParseRefreshToken() error = %v, want %v", err, ErrTokenInvalid)
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	token := "test_token_123"

	hash1 := HashToken(token)
	hash2 := HashToken(token)

	if hash1 != hash2 {
		t.Errorf("HashToken() is not deterministic: %v != %v", hash1, hash2)
	}
}

func TestHashToken_DifferentTokens(t *testing.T) {
	hash1 := HashToken("token_1")
	hash2 := HashToken("token_2")

	if hash1 == hash2 {
		t.Error("HashToken() produced same hash for different tokens")
	}
}

func TestHashToken_Format(t *testing.T) {
	hash := HashToken("test")

	// SHA-256 в hex = 64 символа
	if len(hash) != 64 {
		t.Errorf("HashToken() length = %v, want 64", len(hash))
	}
}

func TestGenerateRandomToken_Length(t *testing.T) {
	token, err := GenerateRandomToken(16)
	if err != nil {
		t.Fatalf("GenerateRandomToken() error = %v", err)
	}

	// 16 байт = 32 hex символа
	if len(token) != 32 {
		t.Errorf("GenerateRandomToken() length = %v, want 32", len(token))
	}
}

func TestGenerateRandomToken_Unique(t *testing.T) {
	token1, _ := GenerateRandomToken(16)
	token2, _ := GenerateRandomToken(16)

	if token1 == token2 {
		t.Error("GenerateRandomToken() produced same token twice")
	}
}

func TestGenerateRandomToken_DifferentLengths(t *testing.T) {
	tests := []struct {
		length   int
		expected int
	}{
		{8, 16},
		{16, 32},
		{32, 64},
	}

	for _, tt := range tests {
		token, err := GenerateRandomToken(tt.length)
		if err != nil {
			t.Fatalf("GenerateRandomToken(%d) error = %v", tt.length, err)
		}
		if len(token) != tt.expected {
			t.Errorf("GenerateRandomToken(%d) length = %v, want %v", tt.length, len(token), tt.expected)
		}
	}
}
