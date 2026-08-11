package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// PerRouteRedisLimiter - в отличие от RateLimiter (in-memory, глобальный на
// всё приложение), это фиксированное окно на конкретный роут поверх Redis -
// нужен там, где предел должен быть куда строже общего (напр. /auth/register
// против массовой регистрации ботов) и общим для всех инстансов бэкенда за
// балансировщиком (in-memory limiter per-instance такую защиту не даёт).
// Fail-open при недоступности Redis - как и LoginGuard, доступность выше
// полноты защиты при деградации инфраструктуры.
func PerRouteRedisLimiter(redisClient *redis.Client, prefix string, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		key := fmt.Sprintf("ratelimit:%s:%s", prefix, c.ClientIP())

		count, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}
		if count == 1 {
			_ = redisClient.Expire(ctx, key, window).Err()
		}

		if count > int64(limit) {
			ttl, err := redisClient.TTL(ctx, key).Result()
			if err != nil || ttl < 0 {
				ttl = window
			}
			retryAfter := int(ttl.Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": map[string]interface{}{
					"code":    "RATE_LIMITED",
					"message": "слишком много запросов, попробуйте позже",
				},
			})
			return
		}

		c.Next()
	}
}
