// Package ratelimit - защита /auth/login от подбора пароля/логина: Redis
// хранит счётчик неудачных попыток по (IP, логин), фиксированное окно
// (INCR + EXPIRE на первой неудаче в окне - не sliding, см. RecordFailure).
package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	loginFailWindow    = 15 * time.Minute
	loginFailThreshold = 5
)

type LoginGuard struct {
	redis *redis.Client
}

func NewLoginGuard(redisClient *redis.Client) *LoginGuard {
	return &LoginGuard{redis: redisClient}
}

// loginFailKey - логин хэшируется, чтобы сырые email/login не оседали в
// ключах Redis (видны любому с доступом к INFO/MONITOR).
func loginFailKey(ip, login string) string {
	hash := sha256.Sum256([]byte(login))
	return "ratelimit:login_fail:" + ip + ":" + hex.EncodeToString(hash[:])
}

// CheckLocked - fail-open: недоступность Redis не должна блокировать вход
// (доступность логина важнее полноты защиты от подбора при деградации
// инфраструктуры). Вызывающий код обязан пропускать запрос при err != nil.
func (g *LoginGuard) CheckLocked(ctx context.Context, ip, login string) (locked bool, retryAfter time.Duration, err error) {
	key := loginFailKey(ip, login)
	count, err := g.redis.Get(ctx, key).Int()
	if err != nil {
		if err == redis.Nil {
			return false, 0, nil
		}
		return false, 0, err
	}
	if count < loginFailThreshold {
		return false, 0, nil
	}
	ttl, err := g.redis.TTL(ctx, key).Result()
	if err != nil || ttl < 0 {
		ttl = loginFailWindow
	}
	return true, ttl, nil
}

// RecordFailure - INCR + EXPIRE только на первой неудаче окна (count == 1),
// чтобы окно было честным "N неудач за 15 минут от первой", а не
// продлевалось каждой новой попыткой до бесконечности.
func (g *LoginGuard) RecordFailure(ctx context.Context, ip, login string) error {
	key := loginFailKey(ip, login)
	count, err := g.redis.Incr(ctx, key).Result()
	if err != nil {
		return err
	}
	if count == 1 {
		return g.redis.Expire(ctx, key, loginFailWindow).Err()
	}
	return nil
}

// Reset - вызывается после успешного входа, чтобы легитимный пользователь,
// однажды опечатавшийся, не унаследовал чужой счётчик неудач по тому же IP.
func (g *LoginGuard) Reset(ctx context.Context, ip, login string) error {
	return g.redis.Del(ctx, loginFailKey(ip, login)).Err()
}
