package middleware

import (
	"fmt"
	"net/http"
	"time"

	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
	"github.com/redis/go-redis/v9"
)

// RateLimitRules applies per-path rate limits with a default fallback.
func RateLimitRules(redisClient *redis.Client, defaultRPM int, rules map[string]int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limit := defaultRPM
			if v, ok := rules[r.URL.Path]; ok {
				limit = v
			}

			key := fmt.Sprintf("ratelimit:%s:%s", httputil.ClientIP(r), r.URL.Path)
			count, err := redisClient.Incr(r.Context(), key).Result()
			if err != nil {
				httputil.Error(w, apperrors.ServiceUnavailable("rate limiter unavailable"))
				return
			}
			if count == 1 {
				_ = redisClient.Expire(r.Context(), key, time.Minute).Err()
			}
			if int(count) > limit {
				httputil.Error(w, apperrors.RateLimited())
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
