package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
	"github.com/medflow/backend/internal/pkg/password"
)

// Бизнес-ошибки сервисного слоя
var (
	ErrEmailExists      = errors.New("email already registered")
	ErrNicknameExists   = errors.New("nickname already taken")
	ErrInvalidCreds     = errors.New("invalid email or password")
	ErrTokenInvalid     = errors.New("invalid refresh token")
	ErrTokenExpired     = errors.New("refresh token expired")
	ErrTokenCompromised = errors.New("token reuse detected")
)

type ErrUserBannedWithDetails struct {
	BanReason string
	BannedAt  time.Time
}

func (e *ErrUserBannedWithDetails) Error() string {
	return "user is banned"
}

type AuthService struct {
	userRepo     UserRepository
	tokenRepo    TokenRepository
	tokenService *TokenService
	cfg          *config.Config
}

func NewAuthService(
	userRepo UserRepository,
	tokenRepo TokenRepository,
	tokenService *TokenService,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		tokenRepo:    tokenRepo,
		tokenService: tokenService,
		cfg:          cfg,
	}
}

func (s *AuthService) Register(ctx context.Context, input dto.CreateUserRequest) (*dto.AuthResponse, error) {
	if _, err := s.userRepo.FindByEmail(ctx, input.Email); err == nil {
		return nil, ErrEmailExists
	}
	if _, err := s.userRepo.FindByNickname(ctx, input.Nickname); err == nil {
		return nil, ErrNicknameExists
	}

	hash, err := password.Hash(input.Password)
	if err != nil {
		return nil, err
	}

	role := models.RoleUser
	if s.isAdminEmail(input.Email) {
		role = models.RoleAdmin
		slog.Info("admin user registered", "email", input.Email)
	}

	user := &models.User{
		ID:           uuid.New(),
		Email:        input.Email,
		PasswordHash: hash,
		Nickname:     input.Nickname,
		Role:         role,
		University:   input.University,
		Course:       input.Course,
		Faculty:      input.Faculty,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		switch {
		case errors.Is(err, models.ErrEmailAlreadyExists):
			return nil, ErrEmailExists
		case errors.Is(err, models.ErrNicknameExists):
			return nil, ErrNicknameExists
		default:
			return nil, err
		}
	}

	return s.tokenService.GenerateAuthTokens(ctx, user, s.tokenRepo)
}

func (s *AuthService) Login(ctx context.Context, input dto.LoginRequest) (*dto.AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			return nil, ErrInvalidCreds
		}
		return nil, err
	}

	if user.IsBanned() {
		reason := ""
		bannedAt := time.Time{}
		if user.BanReason != nil {
			reason = *user.BanReason
		}
		if user.BannedAt != nil {
			bannedAt = *user.BannedAt
		}
		return nil, &ErrUserBannedWithDetails{
			BanReason: reason,
			BannedAt:  bannedAt,
		}
	}

	valid, err := password.Compare(input.Password, user.PasswordHash)
	if err != nil || !valid {
		return nil, ErrInvalidCreds
	}

	return s.tokenService.GenerateAuthTokens(ctx, user, s.tokenRepo)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*dto.TokenPair, error) {
	claims, err := s.tokenService.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	tokenHash := HashToken(refreshToken)

	stored, err := s.tokenRepo.FindByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, models.ErrTokenNotFound) {
			slog.Error("token reuse detected - possible theft",
				"user_id", claims.UserID,
				"token_id", claims.TokenID,
			)
			_ = s.tokenRepo.DeleteByUserID(ctx, claims.UserID)
			return nil, ErrTokenCompromised
		}
		return nil, err
	}

	if stored.IsExpired() {
		_ = s.tokenRepo.DeleteByID(ctx, stored.ID)
		return nil, ErrTokenExpired
	}

	if err := s.tokenRepo.DeleteByID(ctx, stored.ID); err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if user.IsBanned() {
		reason := ""
		bannedAt := time.Time{}
		if user.BanReason != nil {
			reason = *user.BanReason
		}
		if user.BannedAt != nil {
			bannedAt = *user.BannedAt
		}
		return nil, &ErrUserBannedWithDetails{
			BanReason: reason,
			BannedAt:  bannedAt,
		}
	}

	newTokens, err := s.tokenService.GenerateAuthTokens(ctx, user, s.tokenRepo)
	if err != nil {
		return nil, err
	}

	return &dto.TokenPair{
		AccessToken:  newTokens.AccessToken,
		RefreshToken: newTokens.RefreshToken,
		ExpiresIn:    newTokens.ExpiresIn,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := HashToken(refreshToken)

	stored, err := s.tokenRepo.FindByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, models.ErrTokenNotFound) {
			return nil // идемпотентность
		}
		return err
	}

	return s.tokenRepo.DeleteByID(ctx, stored.ID)
}

func (s *AuthService) isAdminEmail(email string) bool {
	adminEmails := os.Getenv("ADMIN_EMAILS")
	if adminEmails == "" {
		return false
	}
	for _, e := range strings.Split(adminEmails, ",") {
		if strings.TrimSpace(e) == email {
			return true
		}
	}
	return false
}
