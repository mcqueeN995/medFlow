package dto

import (
	"github.com/medflow/backend/internal/models"
	"time"
)

type CreateUserRequest struct {
	Email        string             `json:"email" binding:"required,email"`
	Password     string             `json:"password" binding:"required,min=8,max=100"`
	Nickname     string             `json:"nickname" binding:"required,min=3,max=50"`
	University   *models.University `json:"university,omitempty"`
	Course       *int               `json:"course,omitempty" binding:"omitempty,min=1,max=7"`
	Faculty      *string            `json:"faculty,omitempty" binding:"omitempty,max=100"`
	AgreeToTerms bool               `json:"agree_to_terms" binding:"required,eq=true"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
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
