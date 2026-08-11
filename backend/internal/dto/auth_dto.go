package dto

import (
	"github.com/medflow/backend/internal/models"
	"time"
)

type CreateUserRequest struct {
	Email        string             `json:"email" binding:"required,email"`
	Login        string             `json:"login" binding:"required,min=3,max=50"`
	Password     string             `json:"password" binding:"required,min=8,max=100"`
	Nickname     string             `json:"nickname" binding:"required,min=3,max=50"`
	University   *models.University `json:"university,omitempty"`
	Course       *int               `json:"course,omitempty" binding:"omitempty,min=1,max=7"`
	Faculty      *string            `json:"faculty,omitempty" binding:"omitempty,max=100"`
	AgreeToTerms bool               `json:"agree_to_terms" binding:"required,eq=true"`
}

// LoginRequest.Login принимает email ИЛИ users.login (не nickname - тот для
// входа не используется, см. AuthService.findUserByLogin).
type LoginRequest struct {
	Login    string `json:"login" binding:"required,min=3,max=255"`
	Password string `json:"password" binding:"required"`
}

// RequestLoginChangeRequest - шаг 1 смены login: подтверждаем паролем,
// шлём код на текущий email. См. UserService.RequestLoginChange.
type RequestLoginChangeRequest struct {
	NewLogin        string `json:"new_login" binding:"required,min=3,max=50"`
	CurrentPassword string `json:"current_password" binding:"required"`
}

// ConfirmLoginChangeRequest - шаг 2: код из письма. См. UserService.ConfirmLoginChange.
type ConfirmLoginChangeRequest struct {
	Code string `json:"code" binding:"required"`
}

// RequestPasswordResetRequest - шаг 1 восстановления пароля: email или login.
// См. AuthService.RequestPasswordReset.
type RequestPasswordResetRequest struct {
	Login string `json:"login" binding:"required,min=3,max=255"`
}

// ConfirmPasswordResetRequest - шаг 2: код из письма + новый пароль.
// См. AuthService.ConfirmPasswordReset.
type ConfirmPasswordResetRequest struct {
	Code        string `json:"code" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=100"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AuthResponse struct {
	User         UserProfile `json:"user"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresIn    int64       `json:"expires_in"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type UserProfile struct {
	ID              string             `json:"id"`
	Email           string             `json:"email"`
	Login           string             `json:"login"`
	Nickname        string             `json:"nickname"`
	Role            models.UserRole    `json:"role"`
	University      *models.University `json:"university,omitempty"`
	Course          *int               `json:"course,omitempty"`
	Faculty         *string            `json:"faculty,omitempty"`
	EmailVerifiedAt *time.Time         `json:"email_verified_at,omitempty"`
	BannedAt        *time.Time         `json:"banned_at,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
}

type UpdateProfileRequest struct {
	Nickname   *string            `json:"nickname,omitempty" binding:"omitempty,min=3,max=50"`
	University *models.University `json:"university,omitempty"`
	Course     *int               `json:"course,omitempty" binding:"omitempty,min=1,max=7"`
	Faculty    *string            `json:"faculty,omitempty" binding:"omitempty,max=100"`
}

func ToUserProfile(user *models.User) UserProfile {
	return UserProfile{
		ID:              user.ID.String(),
		Email:           user.Email,
		Login:           user.Login,
		Nickname:        user.Nickname,
		Role:            user.Role,
		University:      user.University,
		Course:          user.Course,
		Faculty:         user.Faculty,
		EmailVerifiedAt: user.EmailVerifiedAt,
		BannedAt:        user.BannedAt,
		CreatedAt:       user.CreatedAt,
	}
}
