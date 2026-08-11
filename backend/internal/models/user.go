package models

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleGuest     UserRole = "guest"
	RoleUser      UserRole = "user"
	RoleModerator UserRole = "moderator"
	RoleAdmin     UserRole = "admin"
)

type University string

const (
	UniSechenov  University = "sechenov"
	UniPirogov   University = "pirogov"
	UniEvdokimov University = "evdokimov"
	UniOther     University = "other"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	// Login - отдельный от email и nickname идентификатор для входа: задаётся
	// при регистрации, уникален, меняется только через подтверждение кодом на
	// email (см. AuthService.Login и UserService.RequestLoginChange). Nickname,
	// в отличие от него, - просто отображаемое имя, свободно редактируемое и
	// для входа не используемое.
	Login           string      `json:"login"`
	Nickname        string      `json:"nickname"`
	Role            UserRole    `json:"role"`
	University      *University `json:"university,omitempty"`
	Course          *int        `json:"course,omitempty"`
	Faculty         *string     `json:"faculty,omitempty"`
	EmailVerifiedAt *time.Time  `json:"email_verified_at,omitempty"`
	BannedAt        *time.Time  `json:"banned_at,omitempty"`
	BanReason       *string     `json:"ban_reason,omitempty"`
	BannedBy        *uuid.UUID  `json:"banned_by,omitempty"`
	DeletedAt       *time.Time  `json:"-"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

func (u *User) IsBanned() bool {
	return u.BannedAt != nil
}

func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}
