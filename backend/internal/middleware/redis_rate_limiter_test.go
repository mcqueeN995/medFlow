package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func setupTestRedisForMiddleware(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "localhost:6380"})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("failed to connect to test redis: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestPerRouteRedisLimiter_AllowsUnderLimit_BlocksOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisClient := setupTestRedisForMiddleware(t)
	prefix := "test_register_" + uuid.New().String()[:8]

	router := gin.New()
	router.Use(PerRouteRedisLimiter(redisClient, prefix, 3, time.Minute))
	router.POST("/register", func(c *gin.Context) { c.Status(http.StatusCreated) })

	for i := 1; i <= 3; i++ {
		req, _ := http.NewRequest(http.MethodPost, "/register", nil)
		req.RemoteAddr = "9.9.9.9:1234"
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("request %d: status = %d, want 201 (within limit)", i, rec.Code)
		}
	}

	req, _ := http.NewRequest(http.MethodPost, "/register", nil)
	req.RemoteAddr = "9.9.9.9:1234"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("4th request: status = %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header on 429 response")
	}
}

func TestPerRouteRedisLimiter_DifferentIPs_TrackedIndependently(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisClient := setupTestRedisForMiddleware(t)
	prefix := "test_register_" + uuid.New().String()[:8]

	router := gin.New()
	router.Use(PerRouteRedisLimiter(redisClient, prefix, 1, time.Minute))
	router.POST("/register", func(c *gin.Context) { c.Status(http.StatusCreated) })

	req1, _ := http.NewRequest(http.MethodPost, "/register", nil)
	req1.RemoteAddr = "1.1.1.1:1234"
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("IP1 first request: status = %d, want 201", rec1.Code)
	}

	req2, _ := http.NewRequest(http.MethodPost, "/register", nil)
	req2.RemoteAddr = "2.2.2.2:1234"
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("IP2 first request: status = %d, want 201 (независимый лимит по IP)", rec2.Code)
	}
}

func TestPerRouteRedisLimiter_RedisUnavailable_FailsOpen(t *testing.T) {
	gin.SetMode(gin.TestMode)
	unreachable := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 200 * time.Millisecond,
	})
	t.Cleanup(func() { _ = unreachable.Close() })

	router := gin.New()
	router.Use(PerRouteRedisLimiter(unreachable, "test_unreachable", 1, time.Minute))
	router.POST("/register", func(c *gin.Context) { c.Status(http.StatusCreated) })

	req, _ := http.NewRequest(http.MethodPost, "/register", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (fail-open при недоступном Redis)", rec.Code)
	}
}
