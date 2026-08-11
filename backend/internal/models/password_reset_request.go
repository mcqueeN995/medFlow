package models

import (
	"time"

	"github.com/google/uuid"
)

// PasswordResetRequest - заявка на восстановление забытого пароля,
// подтверждается кодом, отправленным на email владельца аккаунта
// (см. AuthService.RequestPasswordReset/ConfirmPasswordReset).
type PasswordResetRequest struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"-"`
	CodeHash  string    `json:"-"`
	ExpiresAt time.Time `json:"-"`
	CreatedAt time.Time `json:"-"`
}

func (r *PasswordResetRequest) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}
