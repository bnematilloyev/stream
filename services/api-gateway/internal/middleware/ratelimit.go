package middleware

import (
	"fmt"
	"net/http"
	"time"

	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
	"github.com/redis/go-redis/v9"
)

func RateLimit(redisClient *redis.Client, maxPerMinute int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := fmt.Sprintf("ratelimit:%s:%s", clientIP(r), r.URL.Path)

			count, err := redisClient.Incr(r.Context(), key).Result()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if count == 1 {
				_ = redisClient.Expire(r.Context(), key, time.Minute).Err()
			}
			if int(count) > maxPerMinute {
				httputil.Error(w, apperrors.RateLimited())
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	return r.RemoteAddr
}
