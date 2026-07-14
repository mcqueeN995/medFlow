package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
	"github.com/medflow/backend/internal/pkg/jwt"
)

func setupTestAuthService(userRepo *mockUserRepository, tokenRepo *mockTokenRepository) *AuthService {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:  "test_access_secret",
			RefreshSecret: "test_refresh_secret",
			AccessExpire:  15 * time.Minute,
			RefreshExpire: 30 * 24 * time.Hour,
		},
	}
	tokenService := NewTokenService(cfg)
	return NewAuthService(userRepo, tokenRepo, tokenService, cfg)
}

func TestAuthService_Register_Success(t *testing.T) {
	userRepo := &mockUserRepository{
		findByEmailFn: func(ctx context.Context, email string) (*models.User, error) {
			return nil, models.ErrUserNotFound
		},
		findByNicknameFn: func(ctx context.Context, nickname string) (*models.User, error) {
			return nil, models.ErrUserNotFound
		},
		createFn: func(ctx context.Context, user *models.User) error {
			user.CreatedAt = time.Now()
			user.UpdatedAt = time.Now()
			return nil
		},
	}

	tokenRepo := &mockTokenRepository{
		saveFn: func(ctx context.Context, token *models.RefreshToken) error {
			token.CreatedAt = time.Now()
			return nil
		},
	}

	service := setupTestAuthService(userRepo, tokenRepo)
	ctx := context.Background()

	input := dto.CreateUserRequest{
		Email:        "test@medflow.local",
		Password:     "password123",
		Nickname:     "testuser",
		AgreeToTerms: true,
	}

	resp, err := service.Register(ctx, input)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
	if resp.RefreshToken == "" {
		t.Error("RefreshToken is empty")
	}
	if resp.User.Email != input.Email {
		t.Errorf("User.Email = %v, want %v", resp.User.Email, input.Email)
	}
	if resp.User.Role != models.RoleUser {
		t.Errorf("User.Role = %v, want %v", resp.User.Role, models.RoleUser)
	}
	if resp.ExpiresIn != 900 {
		t.Errorf("ExpiresIn = %v, want 900", resp.ExpiresIn)
	}
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	userRepo := &mockUserRepository{
		findByEmailFn: func(ctx context.Context, email string) (*models.User, error) {
			return &models.User{ID: uuid.New(), Email: email}, nil
		},
	}

	service := setupTestAuthService(userRepo, &mockTokenRepository{})
	ctx := context.Background()

	input := dto.CreateUserRequest{
		Email:        "existing@medflow.local",
		Password:     "password123",
		Nickname:     "newuser",
		AgreeToTerms: true,
	}

	_, err := service.Register(ctx, input)
	if !errors.Is(err, ErrEmailExists) {
		t.Errorf("Register() error = %v, want %v", err, ErrEmailExists)
	}
}

func TestAuthService_Register_DuplicateNickname(t *testing.T) {
	userRepo := &mockUserRepository{
		findByEmailFn: func(ctx context.Context, email string) (*models.User, error) {
			return nil, models.ErrUserNotFound
		},
		findByNicknameFn: func(ctx context.Context, nickname string) (*models.User, error) {
			return &models.User{ID: uuid.New(), Nickname: nickname}, nil
		},
	}

	service := setupTestAuthService(userRepo, &mockTokenRepository{})
	ctx := context.Background()

	input := dto.CreateUserRequest{
		Email:        "new@medflow.local",
		Password:     "password123",
		Nickname:     "existing_nick",
		AgreeToTerms: true,
	}

	_, err := service.Register(ctx, input)
	if !errors.Is(err, ErrNicknameExists) {
		t.Errorf("Register() error = %v, want %v", err, ErrNicknameExists)
	}
}

func TestAuthService_Register_DBError(t *testing.T) {
	userRepo := &mockUserRepository{
		findByEmailFn: func(ctx context.Context, email string) (*models.User, error) {
			return nil, models.ErrUserNotFound
		},
		findByNicknameFn: func(ctx context.Context, nickname string) (*models.User, error) {
			return nil, models.ErrUserNotFound
		},
		createFn: func(ctx context.Context, user *models.User) error {
			return errors.New("database error")
		},
	}

	service := setupTestAuthService(userRepo, &mockTokenRepository{})
	ctx := context.Background()

	input := dto.CreateUserRequest{
		Email:        "test@medflow.local",
		Password:     "password123",
		Nickname:     "testuser",
		AgreeToTerms: true,
	}

	_, err := service.Register(ctx, input)
	if err == nil {
		t.Error("Register() expected error, got nil")
	}
	if errors.Is(err, ErrEmailExists) || errors.Is(err, ErrNicknameExists) {
		t.Errorf("Register() returned business error instead of DB error: %v", err)
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	hashedPassword := hashPasswordForTest("password123")

	user := &models.User{
		ID:           uuid.New(),
		Email:        "test@medflow.local",
		PasswordHash: hashedPassword,
		Nickname:     "testuser",
		Role:         models.RoleUser,
	}

	userRepo := &mockUserRepository{
		findByEmailFn: func(ctx context.Context, email string) (*models.User, error) {
			return user, nil
		},
	}

	tokenRepo := &mockTokenRepository{
		saveFn: func(ctx context.Context, token *models.RefreshToken) error {
			token.CreatedAt = time.Now()
			return nil
		},
	}

	service := setupTestAuthService(userRepo, tokenRepo)
	ctx := context.Background()

	input := dto.LoginRequest{
		Email:    "test@medflow.local",
		Password: "password123",
	}

	resp, err := service.Login(ctx, input)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
	if resp.RefreshToken == "" {
		t.Error("RefreshToken is empty")
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	hashedPassword := hashPasswordForTest("password123")

	user := &models.User{
		ID:           uuid.New(),
		Email:        "test@medflow.local",
		PasswordHash: hashedPassword,
		Role:         models.RoleUser,
	}

	userRepo := &mockUserRepository{
		findByEmailFn: func(ctx context.Context, email string) (*models.User, error) {
			return user, nil
		},
	}

	service := setupTestAuthService(userRepo, &mockTokenRepository{})
	ctx := context.Background()

	input := dto.LoginRequest{
		Email:    "test@medflow.local",
		Password: "wrongpassword",
	}

	_, err := service.Login(ctx, input)
	if !errors.Is(err, ErrInvalidCreds) {
		t.Errorf("Login() error = %v, want %v", err, ErrInvalidCreds)
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	userRepo := &mockUserRepository{
		findByEmailFn: func(ctx context.Context, email string) (*models.User, error) {
			return nil, models.ErrUserNotFound
		},
	}

	service := setupTestAuthService(userRepo, &mockTokenRepository{})
	ctx := context.Background()

	input := dto.LoginRequest{
		Email:    "nonexistent@medflow.local",
		Password: "password123",
	}

	_, err := service.Login(ctx, input)
	if !errors.Is(err, ErrInvalidCreds) {
		t.Errorf("Login() error = %v, want %v", err, ErrInvalidCreds)
	}
}

func TestAuthService_Login_UserBanned(t *testing.T) {
	hashedPassword := hashPasswordForTest("password123")
	bannedAt := time.Now()
	banReason := "violation of rules"

	user := &models.User{
		ID:           uuid.New(),
		Email:        "banned@medflow.local",
		PasswordHash: hashedPassword,
		Role:         models.RoleUser,
		BannedAt:     &bannedAt,
		BanReason:    &banReason,
	}

	userRepo := &mockUserRepository{
		findByEmailFn: func(ctx context.Context, email string) (*models.User, error) {
			return user, nil
		},
	}

	service := setupTestAuthService(userRepo, &mockTokenRepository{})
	ctx := context.Background()

	input := dto.LoginRequest{
		Email:    "banned@medflow.local",
		Password: "password123",
	}

	_, err := service.Login(ctx, input)

	var banErr *ErrUserBannedWithDetails
	if !errors.As(err, &banErr) {
		t.Fatalf("Login() error = %v, want *ErrUserBannedWithDetails", err)
	}
	if banErr.BanReason != banReason {
		t.Errorf("BanReason = %v, want %v", banErr.BanReason, banReason)
	}
	if banErr.BannedAt.IsZero() {
		t.Error("BannedAt is zero")
	}
}

func TestAuthService_Refresh_Success(t *testing.T) {
	user := &models.User{
		ID:       uuid.New(),
		Email:    "test@medflow.local",
		Nickname: "testuser",
		Role:     models.RoleUser,
	}

	tokenID := uuid.New()
	validRefresh, _ := jwt.GenerateRefresh(
		tokenID,
		user.ID,
		"test_refresh_secret",
		30*24*time.Hour,
	)

	storedToken := &models.RefreshToken{
		ID:        tokenID,
		UserID:    user.ID,
		TokenHash: HashToken(validRefresh),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return user, nil
		},
	}

	tokenRepo := &mockTokenRepository{
		findByHashFn: func(ctx context.Context, hash string) (*models.RefreshToken, error) {
			return storedToken, nil
		},
		deleteByIDFn: func(ctx context.Context, id uuid.UUID) error {
			return nil
		},
		saveFn: func(ctx context.Context, token *models.RefreshToken) error {
			token.CreatedAt = time.Now()
			return nil
		},
	}

	service := setupTestAuthService(userRepo, tokenRepo)
	ctx := context.Background()

	tokenPair, err := service.Refresh(ctx, validRefresh)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	if tokenPair.AccessToken == "" {
		t.Error("New AccessToken is empty")
	}
	if tokenPair.RefreshToken == "" {
		t.Error("New RefreshToken is empty")
	}
	if tokenPair.RefreshToken == validRefresh {
		t.Error("New RefreshToken should differ from old one (rotation)")
	}
}

func TestAuthService_Refresh_TokenReuseDetected(t *testing.T) {
	userID := uuid.New()
	tokenID := uuid.New()

	validRefresh, _ := jwt.GenerateRefresh(
		tokenID,
		userID,
		"test_refresh_secret",
		30*24*time.Hour,
	)

	deleteByUserIDCalled := false

	userRepo := &mockUserRepository{}
	tokenRepo := &mockTokenRepository{
		findByHashFn: func(ctx context.Context, hash string) (*models.RefreshToken, error) {
			return nil, models.ErrTokenNotFound
		},
		deleteByUserIDFn: func(ctx context.Context, uid uuid.UUID) error {
			deleteByUserIDCalled = true
			if uid != userID {
				t.Errorf("DeleteByUserID called with wrong userID")
			}
			return nil
		},
	}

	service := setupTestAuthService(userRepo, tokenRepo)
	ctx := context.Background()

	_, err := service.Refresh(ctx, validRefresh)
	if !errors.Is(err, ErrTokenCompromised) {
		t.Errorf("Refresh() error = %v, want %v", err, ErrTokenCompromised)
	}

	if !deleteByUserIDCalled {
		t.Error("DeleteByUserID should be called on token reuse detection")
	}
}

func TestAuthService_Refresh_InvalidToken(t *testing.T) {
	service := setupTestAuthService(&mockUserRepository{}, &mockTokenRepository{})
	ctx := context.Background()

	_, err := service.Refresh(ctx, "invalid_token")
	if !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("Refresh() error = %v, want %v", err, ErrTokenInvalid)
	}
}
func TestAuthService_Refresh_ExpiredToken(t *testing.T) {
	userID := uuid.New()
	tokenID := uuid.New()

	expiredRefresh, _ := jwt.GenerateRefresh(
		tokenID,
		userID,
		"test_refresh_secret",
		-1*time.Hour,
	)

	userRepo := &mockUserRepository{}

	tokenRepo := &mockTokenRepository{}

	service := setupTestAuthService(userRepo, tokenRepo)
	ctx := context.Background()

	_, err := service.Refresh(ctx, expiredRefresh)
	if !errors.Is(err, ErrTokenExpired) {
		t.Errorf("Refresh() error = %v, want %v", err, ErrTokenExpired)
	}
}

func TestAuthService_Refresh_UserBanned(t *testing.T) {
	userID := uuid.New()
	tokenID := uuid.New()

	validRefresh, _ := jwt.GenerateRefresh(
		tokenID,
		userID,
		"test_refresh_secret",
		30*24*time.Hour,
	)

	bannedAt := time.Now()
	banReason := "spam"
	user := &models.User{
		ID:        userID,
		Email:     "banned@medflow.local",
		Role:      models.RoleUser,
		BannedAt:  &bannedAt,
		BanReason: &banReason,
	}

	storedToken := &models.RefreshToken{
		ID:        tokenID,
		UserID:    userID,
		TokenHash: HashToken(validRefresh),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return user, nil
		},
	}

	tokenRepo := &mockTokenRepository{
		findByHashFn: func(ctx context.Context, hash string) (*models.RefreshToken, error) {
			return storedToken, nil
		},
		deleteByIDFn: func(ctx context.Context, id uuid.UUID) error {
			return nil
		},
	}

	service := setupTestAuthService(userRepo, tokenRepo)
	ctx := context.Background()

	_, err := service.Refresh(ctx, validRefresh)

	var banErr *ErrUserBannedWithDetails
	if !errors.As(err, &banErr) {
		t.Fatalf("Refresh() error = %v, want *ErrUserBannedWithDetails", err)
	}
	if banErr.BanReason != banReason {
		t.Errorf("BanReason = %v, want %v", banErr.BanReason, banReason)
	}
	if banErr.BannedAt.IsZero() {
		t.Error("BannedAt is zero")
	}
}

func TestAuthService_Logout_Success(t *testing.T) {
	tokenID := uuid.New()
	storedToken := &models.RefreshToken{
		ID:        tokenID,
		UserID:    uuid.New(),
		TokenHash: "some_hash",
	}

	deleteByIDCalled := false

	userRepo := &mockUserRepository{}
	tokenRepo := &mockTokenRepository{
		findByHashFn: func(ctx context.Context, hash string) (*models.RefreshToken, error) {
			return storedToken, nil
		},
		deleteByIDFn: func(ctx context.Context, id uuid.UUID) error {
			deleteByIDCalled = true
			if id != tokenID {
				t.Errorf("DeleteByID called with wrong id")
			}
			return nil
		},
	}

	service := setupTestAuthService(userRepo, tokenRepo)
	ctx := context.Background()

	err := service.Logout(ctx, "some_refresh_token")
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	if !deleteByIDCalled {
		t.Error("DeleteByID should be called")
	}
}

func TestAuthService_Logout_TokenNotFound(t *testing.T) {
	userRepo := &mockUserRepository{}
	tokenRepo := &mockTokenRepository{
		findByHashFn: func(ctx context.Context, hash string) (*models.RefreshToken, error) {
			return nil, models.ErrTokenNotFound
		},
	}

	service := setupTestAuthService(userRepo, tokenRepo)
	ctx := context.Background()

	err := service.Logout(ctx, "nonexistent_token")
	if err != nil {
		t.Errorf("Logout() error = %v, want nil (idempotent)", err)
	}
}
