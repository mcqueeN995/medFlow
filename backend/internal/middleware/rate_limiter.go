package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiterConfig конфигурация rate limiter
type RateLimiterConfig struct {
	RequestsPerMinute int
	BurstSize         int
}

// DefaultRateLimiterConfig возвращает дефолтную конфигурацию
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		RequestsPerMinute: 100,
		BurstSize:         20,
	}
}

// visitor хранит limiter и время последнего запроса
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// ipRateLimiter хранит limiter'ы для каждого IP
type ipRateLimiter struct {
	ips map[string]*visitor
	mu  sync.RWMutex
	r   rate.Limit
	b   int
}

// newIPRateLimiter создает новый rate limiter
func newIPRateLimiter(rpm int, burst int) *ipRateLimiter {
	rl := &ipRateLimiter{
		ips: make(map[string]*visitor),
		r:   rate.Limit(float64(rpm) / 60.0), // конвертируем в запросы в секунду
		b:   burst,
	}

	// Запускаем горутину для очистки старых записей
	go rl.cleanup()

	return rl
}

// addVisitor добавляет нового посетителя или возвращает существующего
func (rl *ipRateLimiter) addVisitor(ip string) *rate.Limiter {
	rl.mu.RLock()
	v, exists := rl.ips[ip]
	rl.mu.RUnlock()

	if exists {
		v.lastSeen = time.Now()
		return v.limiter
	}

	// Создаем новый limiter
	limiter := rate.NewLimiter(rl.r, rl.b)

	rl.mu.Lock()
	rl.ips[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
	rl.mu.Unlock()

	return limiter
}

// cleanup удаляет старые записи
func (rl *ipRateLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)

		rl.mu.Lock()
		for ip, v := range rl.ips {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.ips, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimiter ограничивает количество запросов с одного IP
func RateLimiter(config RateLimiterConfig) gin.HandlerFunc {
	// Создаем отдельный limiter для этой конфигурации
	localLimiter := newIPRateLimiter(config.RequestsPerMinute, config.BurstSize)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		l := localLimiter.addVisitor(ip)

		if !l.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": map[string]interface{}{
					"code":    "RATE_LIMITED",
					"message": "too many requests, please try again later",
				},
			})
			return
		}

		c.Next()
	}
}
