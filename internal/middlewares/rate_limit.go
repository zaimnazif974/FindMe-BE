package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"findme/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func LiveLocationRateLimit(client *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := UserID(c)
		key := fmt.Sprintf("rate:live:%s", userID)
		count, err := incrementWithTTL(c.Request.Context(), client, key, window)
		if err == nil && count > int64(limit) {
			utils.ResponseFailed(c, http.StatusTooManyRequests, "RATE_LIMITED", "Too many live location updates")
			return
		}
		c.Next()
	}
}

func incrementWithTTL(ctx context.Context, client *redis.Client, key string, ttl time.Duration) (int64, error) {
	pipe := client.TxPipeline()
	count := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return count.Val(), err
}
