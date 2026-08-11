package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func setupTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "localhost:6380"})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("failed to connect to test redis: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})
	return client
}

func TestLoginGuard_FifthFailureStillAllows_SixthBlocks(t *testing.T) {
	redisClient := setupTestRedis(t)
	guard := NewLoginGuard(redisClient)
	ctx := context.Background()
	ip := "10.0.0." + uuid.New().String()[:8]
	login := "attacker@medflow.local"
	t.Cleanup(func() { _ = guard.Reset(ctx, ip, login) })

	for i := 1; i <= 5; i++ {
		locked, _, err := guard.CheckLocked(ctx, ip, login)
		if err != nil {
			t.Fatalf("CheckLocked() [attempt %d] error = %v", i, err)
		}
		if locked {
			t.Fatalf("CheckLocked() [attempt %d] locked = true, want false (5-я неудача должна ещё пропускать)", i)
		}
		if err := guard.RecordFailure(ctx, ip, login); err != nil {
			t.Fatalf("RecordFailure() [attempt %d] error = %v", i, err)
		}
	}

	locked, retryAfter, err := guard.CheckLocked(ctx, ip, login)
	if err != nil {
		t.Fatalf("CheckLocked() [6th] error = %v", err)
	}
	if !locked {
		t.Fatal("CheckLocked() [6th] locked = false, want true (6-я попытка должна блокироваться)")
	}
	if retryAfter <= 0 || retryAfter > loginFailWindow {
		t.Errorf("retryAfter = %v, want in (0, %v]", retryAfter, loginFailWindow)
	}
}

func TestLoginGuard_SuccessfulLoginResetsCounter(t *testing.T) {
	redisClient := setupTestRedis(t)
	guard := NewLoginGuard(redisClient)
	ctx := context.Background()
	ip := "10.0.0." + uuid.New().String()[:8]
	login := "user@medflow.local"

	for i := 0; i < loginFailThreshold; i++ {
		if err := guard.RecordFailure(ctx, ip, login); err != nil {
			t.Fatalf("RecordFailure() error = %v", err)
		}
	}
	locked, _, err := guard.CheckLocked(ctx, ip, login)
	if err != nil {
		t.Fatalf("CheckLocked() error = %v", err)
	}
	if !locked {
		t.Fatal("CheckLocked() locked = false, want true before reset")
	}

	if err := guard.Reset(ctx, ip, login); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	locked, _, err = guard.CheckLocked(ctx, ip, login)
	if err != nil {
		t.Fatalf("CheckLocked() after reset error = %v", err)
	}
	if locked {
		t.Fatal("CheckLocked() after reset locked = true, want false")
	}
}

func TestLoginGuard_DifferentLoginsOnSameIP_TrackedIndependently(t *testing.T) {
	redisClient := setupTestRedis(t)
	guard := NewLoginGuard(redisClient)
	ctx := context.Background()
	ip := "10.0.0." + uuid.New().String()[:8]
	loginA := "a@medflow.local"
	loginB := "b@medflow.local"
	t.Cleanup(func() {
		_ = guard.Reset(ctx, ip, loginA)
		_ = guard.Reset(ctx, ip, loginB)
	})

	for i := 0; i < loginFailThreshold; i++ {
		if err := guard.RecordFailure(ctx, ip, loginA); err != nil {
			t.Fatalf("RecordFailure() error = %v", err)
		}
	}

	lockedA, _, err := guard.CheckLocked(ctx, ip, loginA)
	if err != nil {
		t.Fatalf("CheckLocked() [loginA] error = %v", err)
	}
	if !lockedA {
		t.Error("CheckLocked() [loginA] locked = false, want true")
	}

	lockedB, _, err := guard.CheckLocked(ctx, ip, loginB)
	if err != nil {
		t.Fatalf("CheckLocked() [loginB] error = %v", err)
	}
	if lockedB {
		t.Error("CheckLocked() [loginB] locked = true, want false (независимый счётчик)")
	}
}

func TestLoginGuard_RedisUnavailable_FailsOpen(t *testing.T) {
	// Клиент указывает на несуществующий адрес - имитация недоступности Redis.
	unreachable := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 200 * time.Millisecond,
	})
	t.Cleanup(func() { _ = unreachable.Close() })
	guard := NewLoginGuard(unreachable)
	ctx := context.Background()

	_, _, err := guard.CheckLocked(ctx, "1.2.3.4", "someone@medflow.local")
	if err == nil {
		t.Fatal("CheckLocked() error = nil, want non-nil (caller обязан fail-open по ненулевой ошибке)")
	}
}
