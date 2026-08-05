package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
	"github.com/medflow/backend/internal/pkg/jwt"
	"github.com/medflow/backend/internal/pkg/password"
	"github.com/medflow/backend/internal/service"
)

type mockUserRepo struct {
	findByEmailFn    func(ctx context.Context, email string) (*models.User, error)
	findByNicknameFn func(ctx context.Context, nickname string) (*models.User, error)
	createFn         func(ctx context.Context, user *models.User) error
	findByIDFn       func(ctx context.Context, id uuid.UUID) (*models.User, error)
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(ctx, email)
	}
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepo) FindByNickname(ctx context.Context, nickname string) (*models.User, error) {
	if m.findByNicknameFn != nil {
		return m.findByNicknameFn(ctx, nickname)
	}
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepo) Create(ctx context.Context, user *models.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	return nil
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepo) Update(ctx context.Context, id uuid.UUID, nickname string, university *models.University, course *int, faculty *string) (*models.User, error) {
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockUserRepo) FindPublicByID(ctx context.Context, id uuid.UUID) (*models.PublicUser, error) {
	return nil, models.ErrUserNotFound
}

type mockTokenRepo struct {
	saveFn           func(ctx context.Context, token *models.RefreshToken) error
	findByHashFn     func(ctx context.Context, hash string) (*models.RefreshToken, error)
	deleteByIDFn     func(ctx context.Context, id uuid.UUID) error
	deleteByUserIDFn func(ctx context.Context, userID uuid.UUID) error
}

func (m *mockTokenRepo) Save(ctx context.Context, token *models.RefreshToken) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, token)
	}
	return nil
}

func (m *mockTokenRepo) FindByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	if m.findByHashFn != nil {
		return m.findByHashFn(ctx, hash)
	}
	return nil, models.ErrTokenNotFound
}

func (m *mockTokenRepo) DeleteByID(ctx context.Context, id uuid.UUID) error {
	if m.deleteByIDFn != nil {
		return m.deleteByIDFn(ctx, id)
	}
	return nil
}

func (m *mockTokenRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(ctx, userID)
	}
	return nil
}

func (m *mockTokenRepo) DeleteExpired(ctx context.Context) (int64, error) {
	return 0, nil
}

func setupTestHandler() (*AuthHandler, *mockUserRepo, *mockTokenRepo) {
	userRepo := &mockUserRepo{}
	tokenRepo := &mockTokenRepo{}

	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:  "test_access_secret",
			RefreshSecret: "test_refresh_secret",
			AccessExpire:  15 * time.Minute,
			RefreshExpire: 30 * 24 * time.Hour,
		},
	}

	tokenService := service.NewTokenService(cfg)
	authService := service.NewAuthService(userRepo, tokenRepo, tokenService, cfg)

	return NewAuthHandler(authService), userRepo, tokenRepo
}

func performRequest(r http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonData, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonData)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	return recorder
}

func TestAuthHandler_Register_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, userRepo, tokenRepo := setupTestHandler()

	userRepo.findByEmailFn = func(ctx context.Context, email string) (*models.User, error) {
		return nil, models.ErrUserNotFound
	}
	userRepo.findByNicknameFn = func(ctx context.Context, nickname string) (*models.User, error) {
		return nil, models.ErrUserNotFound
	}
	userRepo.createFn = func(ctx context.Context, user *models.User) error {
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
		return nil
	}
	tokenRepo.saveFn = func(ctx context.Context, token *models.RefreshToken) error {
		token.CreatedAt = time.Now()
		return nil
	}

	router := gin.New()
	router.POST("/auth/register", handler.Register)

	reqBody := dto.CreateUserRequest{
		Email:        "test@medflow.local",
		Password:     "password123",
		Nickname:     "testuser",
		AgreeToTerms: true,
	}

	recorder := performRequest(router, "POST", "/auth/register", reqBody)

	if recorder.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var resp dto.AuthResponse
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Error("Expected tokens in response")
	}
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, userRepo, _ := setupTestHandler()

	userRepo.findByEmailFn = func(ctx context.Context, email string) (*models.User, error) {
		return &models.User{ID: uuid.New(), Email: email}, nil
	}

	router := gin.New()
	router.POST("/auth/register", handler.Register)

	reqBody := dto.CreateUserRequest{
		Email:        "existing@medflow.local",
		Password:     "password123",
		Nickname:     "testuser",
		AgreeToTerms: true,
	}

	recorder := performRequest(router, "POST", "/auth/register", reqBody)

	if recorder.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, recorder.Code)
	}

	var resp ErrorResponse
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	if resp.Error.Code != "EMAIL_EXISTS" {
		t.Errorf("Expected error code EMAIL_EXISTS, got %s", resp.Error.Code)
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, userRepo, tokenRepo := setupTestHandler()

	hashedPwd, _ := password.Hash("password123")
	testUser := &models.User{
		ID:           uuid.New(),
		Email:        "test@medflow.local",
		PasswordHash: hashedPwd,
		Nickname:     "testuser",
		Role:         models.RoleUser,
	}

	userRepo.findByEmailFn = func(ctx context.Context, email string) (*models.User, error) {
		return testUser, nil
	}
	tokenRepo.saveFn = func(ctx context.Context, token *models.RefreshToken) error {
		token.CreatedAt = time.Now()
		return nil
	}

	router := gin.New()
	router.POST("/auth/login", handler.Login)

	reqBody := dto.LoginRequest{
		Email:    "test@medflow.local",
		Password: "password123",
	}

	recorder := performRequest(router, "POST", "/auth/login", reqBody)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestAuthHandler_Login_UserBanned(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, userRepo, _ := setupTestHandler()

	bannedAt := time.Now()
	banReason := "violation of rules"

	userRepo.findByEmailFn = func(ctx context.Context, email string) (*models.User, error) {
		return nil, &service.ErrUserBannedWithDetails{
			BanReason: banReason,
			BannedAt:  bannedAt,
		}
	}

	router := gin.New()
	router.POST("/auth/login", handler.Login)

	reqBody := dto.LoginRequest{
		Email:    "banned@medflow.local",
		Password: "password123",
	}

	recorder := performRequest(router, "POST", "/auth/login", reqBody)

	if recorder.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}

	var resp BanResponse
	json.Unmarshal(recorder.Body.Bytes(), &resp)

	if resp.Error.Code != "BANNED" {
		t.Errorf("Expected error code BANNED, got %s", resp.Error.Code)
	}
	if resp.Error.BanReason != banReason {
		t.Errorf("Expected ban reason '%s', got '%s'", banReason, resp.Error.BanReason)
	}
}

func TestAuthHandler_Refresh_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, userRepo, tokenRepo := setupTestHandler()

	testUser := &models.User{
		ID:       uuid.New(),
		Email:    "test@medflow.local",
		Nickname: "testuser",
		Role:     models.RoleUser,
	}

	tokenID := uuid.New()

	validRefresh, err := jwt.GenerateRefresh(
		tokenID,
		testUser.ID,
		"test_refresh_secret",
		30*24*time.Hour,
	)
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	storedToken := &models.RefreshToken{
		ID:        tokenID,
		UserID:    testUser.ID,
		TokenHash: service.HashToken(validRefresh),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	userRepo.findByIDFn = func(ctx context.Context, id uuid.UUID) (*models.User, error) {
		return testUser, nil
	}
	tokenRepo.findByHashFn = func(ctx context.Context, hash string) (*models.RefreshToken, error) {
		return storedToken, nil
	}
	tokenRepo.deleteByIDFn = func(ctx context.Context, id uuid.UUID) error {
		return nil
	}
	tokenRepo.saveFn = func(ctx context.Context, token *models.RefreshToken) error {
		token.CreatedAt = time.Now()
		return nil
	}

	router := gin.New()
	router.POST("/auth/refresh", handler.Refresh)

	reqBody := dto.RefreshRequest{
		RefreshToken: validRefresh,
	}

	recorder := performRequest(router, "POST", "/auth/refresh", reqBody)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var resp dto.TokenPair
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Error("Expected new tokens in response")
	}
}

func TestAuthHandler_Logout_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, tokenRepo := setupTestHandler()

	tokenRepo.findByHashFn = func(ctx context.Context, hash string) (*models.RefreshToken, error) {
		return &models.RefreshToken{ID: uuid.New()}, nil
	}
	tokenRepo.deleteByIDFn = func(ctx context.Context, id uuid.UUID) error {
		return nil
	}

	router := gin.New()
	router.POST("/auth/logout", handler.Logout)

	reqBody := dto.LogoutRequest{
		RefreshToken: "some_valid_token",
	}

	recorder := performRequest(router, "POST", "/auth/logout", reqBody)

	if recorder.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}
