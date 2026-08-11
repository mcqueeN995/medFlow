package models

import (
	"time"

	"github.com/google/uuid"
)

// LoginChangeRequest - заявка на смену users.login, подтверждается кодом,
// отправленным на текущий email владельца аккаунта (см. UserService).
type LoginChangeRequest struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"-"`
	NewLogin  string    `json:"-"`
	CodeHash  string    `json:"-"`
	ExpiresAt time.Time `json:"-"`
	CreatedAt time.Time `json:"-"`
}

func (r *LoginChangeRequest) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}
