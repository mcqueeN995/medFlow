package jwt

import (
	"errors"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// AccessClaims - claims для access токена
type AccessClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
	jwtlib.RegisteredClaims
}

// RefreshClaims - claims для refresh токена
type RefreshClaims struct {
	TokenID uuid.UUID `json:"token_id"` // ID токена в БД (для ротации)
	UserID  uuid.UUID `json:"user_id"`
	jwtlib.RegisteredClaims
}

// GenerateAccess создает access токен
func GenerateAccess(userID uuid.UUID, role string, secret string, expire time.Duration) (string, error) {
	now := time.Now()
	claims := AccessClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwtlib.RegisteredClaims{
			ExpiresAt: jwtlib.NewNumericDate(now.Add(expire)),
			IssuedAt:  jwtlib.NewNumericDate(now),
			NotBefore: jwtlib.NewNumericDate(now),
			Issuer:    "medflow",
			Subject:   userID.String(),
			ID:        uuid.New().String(),
		},
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefresh создает refresh токен
func GenerateRefresh(tokenID, userID uuid.UUID, secret string, expire time.Duration) (string, error) {
	now := time.Now()
	claims := RefreshClaims{
		TokenID: tokenID,
		UserID:  userID,
		RegisteredClaims: jwtlib.RegisteredClaims{
			ExpiresAt: jwtlib.NewNumericDate(now.Add(expire)),
			IssuedAt:  jwtlib.NewNumericDate(now),
			NotBefore: jwtlib.NewNumericDate(now),
			Issuer:    "medflow",
			Subject:   userID.String(),
			ID:        tokenID.String(),
		},
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseAccess валидирует access токен и возвращает claims
func ParseAccess(tokenString, secret string) (*AccessClaims, error) {
	token, err := jwtlib.ParseWithClaims(tokenString, &AccessClaims{}, func(token *jwtlib.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwtlib.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*AccessClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ParseRefresh валидирует refresh токен и возвращает claims
func ParseRefresh(tokenString, secret string) (*RefreshClaims, error) {
	token, err := jwtlib.ParseWithClaims(tokenString, &RefreshClaims{}, func(token *jwtlib.Token) (interface{}, error) {
		// Проверяем алгоритм подписи
		if _, ok := token.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwtlib.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*RefreshClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
