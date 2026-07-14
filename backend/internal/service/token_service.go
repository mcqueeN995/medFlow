package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
	"github.com/medflow/backend/internal/pkg/jwt"
)

type TokenService struct {
	cfg *config.Config
}

func NewTokenService(cfg *config.Config) *TokenService {
	return &TokenService{cfg: cfg}
}

func (s *TokenService) GenerateAuthTokens(ctx context.Context, user *models.User, tokenRepo TokenRepository) (*dto.AuthResponse, error) {
	access, err := jwt.GenerateAccess(
		user.ID,
		string(user.Role),
		s.cfg.JWT.AccessSecret,
		s.cfg.JWT.AccessExpire,
	)
	if err != nil {
		return nil, err
	}

	tokenID := uuid.New()
	refresh, err := jwt.GenerateRefresh(
		tokenID,
		user.ID,
		s.cfg.JWT.RefreshSecret,
		s.cfg.JWT.RefreshExpire,
	)
	if err != nil {
		return nil, err
	}

	dbToken := &models.RefreshToken{
		ID:        tokenID,
		UserID:    user.ID,
		TokenHash: HashToken(refresh),
		ExpiresAt: time.Now().Add(s.cfg.JWT.RefreshExpire),
	}

	if err := tokenRepo.Save(ctx, dbToken); err != nil {
		return nil, err
	}

	userProfile := dto.ToUserProfile(user)

	return &dto.AuthResponse{
		User:         userProfile,
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(s.cfg.JWT.AccessExpire.Seconds()),
	}, nil
}

func (s *TokenService) ParseRefreshToken(tokenString string) (*jwt.RefreshClaims, error) {
	claims, err := jwt.ParseRefresh(tokenString, s.cfg.JWT.RefreshSecret)
	if err != nil {
		if errors.Is(err, jwt.ErrExpiredToken) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
