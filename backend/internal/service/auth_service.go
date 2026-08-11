package service

import (
	"context"
	"errors"
	"fmt"
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
	ErrEmailExists              = errors.New("email already registered")
	ErrNicknameExists           = errors.New("nickname already taken")
	ErrLoginExists              = errors.New("login already taken")
	ErrInvalidCreds             = errors.New("invalid email or password")
	ErrTokenInvalid             = errors.New("invalid refresh token")
	ErrTokenExpired             = errors.New("refresh token expired")
	ErrTokenCompromised         = errors.New("token reuse detected")
	ErrPasswordResetCodeInvalid = errors.New("invalid or expired password reset code")
)

// passwordResetCodeTTL - см. loginChangeCodeTTL в user_service.go, тот же
// паттерн 6-значного кода на 15 минут.
const passwordResetCodeTTL = 15 * time.Minute

type ErrUserBannedWithDetails struct {
	BanReason string
	BannedAt  time.Time
}

func (e *ErrUserBannedWithDetails) Error() string {
	return "user is banned"
}

type AuthService struct {
	userRepo          UserRepository
	tokenRepo         TokenRepository
	tokenService      *TokenService
	cfg               *config.Config
	passwordResetRepo PasswordResetRepository
	emailSender       EmailSender
}

func NewAuthService(
	userRepo UserRepository,
	tokenRepo TokenRepository,
	tokenService *TokenService,
	cfg *config.Config,
	passwordResetRepo PasswordResetRepository,
	emailSender EmailSender,
) *AuthService {
	return &AuthService{
		userRepo:          userRepo,
		tokenRepo:         tokenRepo,
		tokenService:      tokenService,
		cfg:               cfg,
		passwordResetRepo: passwordResetRepo,
		emailSender:       emailSender,
	}
}

func (s *AuthService) Register(ctx context.Context, input dto.CreateUserRequest) (*dto.AuthResponse, error) {
	if _, err := s.userRepo.FindByEmail(ctx, input.Email); err == nil {
		return nil, ErrEmailExists
	}
	if _, err := s.userRepo.FindByLogin(ctx, input.Login); err == nil {
		return nil, ErrLoginExists
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
		Login:        input.Login,
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
		case errors.Is(err, models.ErrLoginExists):
			return nil, ErrLoginExists
		case errors.Is(err, models.ErrNicknameExists):
			return nil, ErrNicknameExists
		default:
			return nil, err
		}
	}

	return s.tokenService.GenerateAuthTokens(ctx, user, s.tokenRepo)
}

func (s *AuthService) Login(ctx context.Context, input dto.LoginRequest) (*dto.AuthResponse, error) {
	user, err := s.findUserByLogin(ctx, input.Login)
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

// RequestPasswordReset - шаг 1 восстановления забытого пароля: принимает то
// же, что и поле логина при входе (email или users.login), шлёт 6-значный
// код на email аккаунта. Намеренно не возвращает ошибку, если аккаунт не
// найден - чтобы ответ API не палил, зарегистрирован ли данный email/login
// (user enumeration). Прежние неиспользованные запросы того же пользователя
// удаляются - активным может быть только один.
func (s *AuthService) RequestPasswordReset(ctx context.Context, login string) error {
	user, err := s.findUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			return nil
		}
		return err
	}

	code, err := generateNumericCode(6)
	if err != nil {
		return err
	}

	if err := s.passwordResetRepo.DeleteByUserID(ctx, user.ID); err != nil {
		return err
	}
	resetReq := &models.PasswordResetRequest{
		ID:        uuid.New(),
		UserID:    user.ID,
		CodeHash:  HashToken(code),
		ExpiresAt: time.Now().Add(passwordResetCodeTTL),
	}
	if err := s.passwordResetRepo.Save(ctx, resetReq); err != nil {
		return err
	}

	body := fmt.Sprintf("Код для восстановления пароля: %s\n\nКод действителен 15 минут. Если вы не запрашивали восстановление пароля — проигнорируйте это письмо.", code)
	return s.emailSender.Send(user.Email, "medFlow: восстановление пароля", body)
}

// ConfirmPasswordReset - шаг 2: код из письма + новый пароль. Успех - пароль
// обновлён, использованная заявка удалена, все текущие refresh-токены
// отозваны (чтобы угнанная сессия не пережила смену пароля).
func (s *AuthService) ConfirmPasswordReset(ctx context.Context, code, newPassword string) error {
	resetReq, err := s.passwordResetRepo.FindByCodeHash(ctx, HashToken(code))
	if err != nil {
		return ErrPasswordResetCodeInvalid
	}
	if resetReq.IsExpired() {
		_ = s.passwordResetRepo.DeleteByID(ctx, resetReq.ID)
		return ErrPasswordResetCodeInvalid
	}

	hash, err := password.Hash(newPassword)
	if err != nil {
		return err
	}
	if err := s.userRepo.UpdatePassword(ctx, resetReq.UserID, hash); err != nil {
		return err
	}
	_ = s.passwordResetRepo.DeleteByID(ctx, resetReq.ID)

	return s.tokenRepo.DeleteByUserID(ctx, resetReq.UserID)
}

// findUserByLogin принимает то, что ввели в поле логина при входе: email
// (по "@") или users.login. Nickname сюда сознательно не входит - он для
// входа не используется, см. models.User.Login doc-комментарий.
func (s *AuthService) findUserByLogin(ctx context.Context, login string) (*models.User, error) {
	if strings.Contains(login, "@") {
		return s.userRepo.FindByEmail(ctx, login)
	}
	return s.userRepo.FindByLogin(ctx, login)
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
