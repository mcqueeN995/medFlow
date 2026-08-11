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
	findByLoginFn    func(ctx context.Context, login string) (*models.User, error)
	findByNicknameFn func(ctx context.Context, nickname string) (*models.User, error)
	createFn         func(ctx context.Context, user *models.User) error
	findByIDFn       func(ctx context.Context, id uuid.UUID) (*models.User, error)
	updatePasswordFn func(ctx context.Context, id uuid.UUID, passwordHash string) error
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(ctx, email)
	}
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepo) FindByLogin(ctx context.Context, login string) (*models.User, error) {
	if m.findByLoginFn != nil {
		return m.findByLoginFn(ctx, login)
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

func (m *mockUserRepo) UpdateLogin(ctx context.Context, id uuid.UUID, login string) (*models.User, error) {
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	if m.updatePasswordFn != nil {
		return m.updatePasswordFn(ctx, id, passwordHash)
	}
	return nil
}

func (m *mockUserRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockUserRepo) FindPublicByID(ctx context.Context, id uuid.UUID) (*models.PublicUser, error) {
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepo) AdminList(ctx context.Context, f models.AdminUserListFilter) ([]models.User, int, error) {
	return nil, 0, nil
}

func (m *mockUserRepo) ChangeRole(ctx context.Context, id uuid.UUID, role models.UserRole) (*models.User, error) {
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepo) Ban(ctx context.Context, id, bannedBy uuid.UUID, reason string) (*models.User, error) {
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepo) Unban(ctx context.Context, id uuid.UUID) (*models.User, error) {
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

type mockPasswordResetRepo struct {
	saveFn           func(ctx context.Context, req *models.PasswordResetRequest) error
	findByCodeHashFn func(ctx context.Context, codeHash string) (*models.PasswordResetRequest, error)
	deleteByIDFn     func(ctx context.Context, id uuid.UUID) error
	deleteByUserIDFn func(ctx context.Context, userID uuid.UUID) error
}

func (m *mockPasswordResetRepo) Save(ctx context.Context, req *models.PasswordResetRequest) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, req)
	}
	return nil
}

func (m *mockPasswordResetRepo) FindByCodeHash(ctx context.Context, codeHash string) (*models.PasswordResetRequest, error) {
	if m.findByCodeHashFn != nil {
		return m.findByCodeHashFn(ctx, codeHash)
	}
	return nil, models.ErrPasswordResetRequestNotFound
}

func (m *mockPasswordResetRepo) DeleteByID(ctx context.Context, id uuid.UUID) error {
	if m.deleteByIDFn != nil {
		return m.deleteByIDFn(ctx, id)
	}
	return nil
}

func (m *mockPasswordResetRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(ctx, userID)
	}
	return nil
}

type mockEmailSenderHandler struct {
	sendFn func(to, subject, body string) error
	sent   []string
}

func (m *mockEmailSenderHandler) Send(to, subject, body string) error {
	m.sent = append(m.sent, to)
	if m.sendFn != nil {
		return m.sendFn(to, subject, body)
	}
	return nil
}

// mockLoginGuard ручной мок LoginRateLimiter - по умолчанию никогда не
// блокирует, чтобы существующие тесты Login/Register не знали про
// rate-limiting; тесты самого лимитера переопределяют нужные *Fn.
type mockLoginGuard struct {
	checkLockedFn   func(ctx context.Context, ip, login string) (bool, time.Duration, error)
	recordFailureFn func(ctx context.Context, ip, login string) error
	resetFn         func(ctx context.Context, ip, login string) error
	failureCount    int
	resetCount      int
}

func (m *mockLoginGuard) CheckLocked(ctx context.Context, ip, login string) (bool, time.Duration, error) {
	if m.checkLockedFn != nil {
		return m.checkLockedFn(ctx, ip, login)
	}
	return false, 0, nil
}

func (m *mockLoginGuard) RecordFailure(ctx context.Context, ip, login string) error {
	m.failureCount++
	if m.recordFailureFn != nil {
		return m.recordFailureFn(ctx, ip, login)
	}
	return nil
}

func (m *mockLoginGuard) Reset(ctx context.Context, ip, login string) error {
	m.resetCount++
	if m.resetFn != nil {
		return m.resetFn(ctx, ip, login)
	}
	return nil
}

func setupTestHandler() (*AuthHandler, *mockUserRepo, *mockTokenRepo) {
	handler, userRepo, tokenRepo, _, _, _ := setupTestHandlerWithReset()
	return handler, userRepo, tokenRepo
}

func setupTestHandlerWithReset() (*AuthHandler, *mockUserRepo, *mockTokenRepo, *mockPasswordResetRepo, *mockEmailSenderHandler, *mockLoginGuard) {
	userRepo := &mockUserRepo{}
	tokenRepo := &mockTokenRepo{}
	resetRepo := &mockPasswordResetRepo{}
	emailSender := &mockEmailSenderHandler{}
	loginGuard := &mockLoginGuard{}

	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:  "test_access_secret",
			RefreshSecret: "test_refresh_secret",
			AccessExpire:  15 * time.Minute,
			RefreshExpire: 30 * 24 * time.Hour,
		},
	}

	tokenService := service.NewTokenService(cfg)
	authService := service.NewAuthService(userRepo, tokenRepo, tokenService, cfg, resetRepo, emailSender)

	return NewAuthHandler(authService, loginGuard), userRepo, tokenRepo, resetRepo, emailSender, loginGuard
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
		Login:        "testlogin",
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
		Login:        "testlogin",
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
		Login:    "test@medflow.local",
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
		Login:    "banned@medflow.local",
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

func TestAuthHandler_Login_LockedByGuard_Returns429(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, userRepo, _, _, _, loginGuard := setupTestHandlerWithReset()

	loginGuard.checkLockedFn = func(ctx context.Context, ip, login string) (bool, time.Duration, error) {
		return true, 42 * time.Second, nil
	}
	// Если бы guard не сработал раньше, этот мок дал бы 200 - убеждаемся, что
	// сервис вообще не вызывается при блокировке.
	userRepo.findByEmailFn = func(ctx context.Context, email string) (*models.User, error) {
		t.Fatal("authService.Login должен быть пропущен при заблокированном guard")
		return nil, models.ErrUserNotFound
	}

	router := gin.New()
	router.POST("/auth/login", handler.Login)

	reqBody := dto.LoginRequest{Login: "locked@medflow.local", Password: "whatever123"}
	recorder := performRequest(router, "POST", "/auth/login", reqBody)

	if recorder.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, recorder.Code)
	}
	if recorder.Header().Get("Retry-After") != "42" {
		t.Errorf("Expected Retry-After=42, got %q", recorder.Header().Get("Retry-After"))
	}

	var resp ErrorResponse
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	if resp.Error.Code != "TOO_MANY_ATTEMPTS" {
		t.Errorf("Expected error code TOO_MANY_ATTEMPTS, got %s", resp.Error.Code)
	}
}

func TestAuthHandler_Login_WrongPassword_RecordsFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, userRepo, _, _, _, loginGuard := setupTestHandlerWithReset()

	hashedPwd, _ := password.Hash("correct-password")
	userRepo.findByEmailFn = func(ctx context.Context, email string) (*models.User, error) {
		return &models.User{ID: uuid.New(), Email: email, PasswordHash: hashedPwd, Nickname: "testuser", Role: models.RoleUser}, nil
	}

	router := gin.New()
	router.POST("/auth/login", handler.Login)

	reqBody := dto.LoginRequest{Login: "test@medflow.local", Password: "wrong-password"}
	recorder := performRequest(router, "POST", "/auth/login", reqBody)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
	if loginGuard.failureCount != 1 {
		t.Errorf("Expected RecordFailure() to be called once, got %d calls", loginGuard.failureCount)
	}
	if loginGuard.resetCount != 0 {
		t.Errorf("Expected Reset() not to be called on failed login, got %d calls", loginGuard.resetCount)
	}
}

func TestAuthHandler_Login_Success_ResetsGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, userRepo, tokenRepo, _, _, loginGuard := setupTestHandlerWithReset()

	hashedPwd, _ := password.Hash("password123")
	userRepo.findByEmailFn = func(ctx context.Context, email string) (*models.User, error) {
		return &models.User{ID: uuid.New(), Email: email, PasswordHash: hashedPwd, Nickname: "testuser", Role: models.RoleUser}, nil
	}
	tokenRepo.saveFn = func(ctx context.Context, token *models.RefreshToken) error {
		token.CreatedAt = time.Now()
		return nil
	}

	router := gin.New()
	router.POST("/auth/login", handler.Login)

	reqBody := dto.LoginRequest{Login: "test@medflow.local", Password: "password123"}
	recorder := performRequest(router, "POST", "/auth/login", reqBody)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if loginGuard.resetCount != 1 {
		t.Errorf("Expected Reset() to be called once on successful login, got %d calls", loginGuard.resetCount)
	}
	if loginGuard.failureCount != 0 {
		t.Errorf("Expected RecordFailure() not to be called on successful login, got %d calls", loginGuard.failureCount)
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

func TestAuthHandler_RequestPasswordReset_SendsCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, userRepo, _, _, emailSender, _ := setupTestHandlerWithReset()

	userRepo.findByEmailFn = func(ctx context.Context, email string) (*models.User, error) {
		return &models.User{ID: uuid.New(), Email: email}, nil
	}

	router := gin.New()
	router.POST("/auth/password-reset", handler.RequestPasswordReset)

	reqBody := dto.RequestPasswordResetRequest{Login: "test@medflow.local"}
	recorder := performRequest(router, "POST", "/auth/password-reset", reqBody)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if len(emailSender.sent) != 1 || emailSender.sent[0] != "test@medflow.local" {
		t.Errorf("Expected reset code sent to test@medflow.local, got %v", emailSender.sent)
	}
}

func TestAuthHandler_RequestPasswordReset_UnknownLogin_StillReturns200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, userRepo, _, _, emailSender, _ := setupTestHandlerWithReset()

	userRepo.findByEmailFn = func(ctx context.Context, email string) (*models.User, error) {
		return nil, models.ErrUserNotFound
	}

	router := gin.New()
	router.POST("/auth/password-reset", handler.RequestPasswordReset)

	reqBody := dto.RequestPasswordResetRequest{Login: "unknown@medflow.local"}
	recorder := performRequest(router, "POST", "/auth/password-reset", reqBody)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status %d (не палим существование аккаунта), got %d", http.StatusOK, recorder.Code)
	}
	if len(emailSender.sent) != 0 {
		t.Errorf("Expected no email sent for unknown login, got %v", emailSender.sent)
	}
}

func TestAuthHandler_ConfirmPasswordReset_InvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, _, resetRepo, _, _ := setupTestHandlerWithReset()

	resetRepo.findByCodeHashFn = func(ctx context.Context, codeHash string) (*models.PasswordResetRequest, error) {
		return nil, models.ErrPasswordResetRequestNotFound
	}

	router := gin.New()
	router.POST("/auth/password-reset/confirm", handler.ConfirmPasswordReset)

	reqBody := dto.ConfirmPasswordResetRequest{Code: "000000", NewPassword: "newpassword123"}
	recorder := performRequest(router, "POST", "/auth/password-reset/confirm", reqBody)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var resp ErrorResponse
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	if resp.Error.Code != "INVALID_CODE" {
		t.Errorf("Expected error code INVALID_CODE, got %s", resp.Error.Code)
	}
}

func TestAuthHandler_ConfirmPasswordReset_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, userRepo, tokenRepo, resetRepo, _, _ := setupTestHandlerWithReset()

	userID := uuid.New()
	resetRepo.findByCodeHashFn = func(ctx context.Context, codeHash string) (*models.PasswordResetRequest, error) {
		return &models.PasswordResetRequest{ID: uuid.New(), UserID: userID, ExpiresAt: time.Now().Add(15 * time.Minute)}, nil
	}
	passwordUpdated := false
	userRepo.updatePasswordFn = func(ctx context.Context, id uuid.UUID, passwordHash string) error {
		passwordUpdated = id == userID
		return nil
	}
	tokensRevoked := false
	tokenRepo.deleteByUserIDFn = func(ctx context.Context, id uuid.UUID) error {
		tokensRevoked = id == userID
		return nil
	}

	router := gin.New()
	router.POST("/auth/password-reset/confirm", handler.ConfirmPasswordReset)

	reqBody := dto.ConfirmPasswordResetRequest{Code: "123456", NewPassword: "newpassword123"}
	recorder := performRequest(router, "POST", "/auth/password-reset/confirm", reqBody)

	if recorder.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if !passwordUpdated {
		t.Error("Expected password to be updated for the request's user")
	}
	if !tokensRevoked {
		t.Error("Expected refresh tokens to be revoked for the request's user")
	}
}
