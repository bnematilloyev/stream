package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
)

// RateLimitRules applies per-path rate limits with a default fallback.
func RateLimitRules(redisClient *redis.Client, defaultRPM int, rules map[string]int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limit, rulePath := limitForPath(r.URL.Path, defaultRPM, rules)

			key := fmt.Sprintf("ratelimit:%s:%s", httputil.ClientIP(r), rulePath)
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

func limitForPath(path string, fallback int, rules map[string]int) (int, string) {
	if v, ok := rules[path]; ok {
		return v, path
	}
	for pattern, limit := range rules {
		if matchWildcardPath(pattern, path) {
			return limit, pattern
		}
	}
	return fallback, path
}

func matchWildcardPath(pattern, path string) bool {
	if !strings.Contains(pattern, "*") {
		return false
	}
	pp := strings.Split(strings.Trim(pattern, "/"), "/")
	sp := strings.Split(strings.Trim(path, "/"), "/")
	if len(pp) != len(sp) {
		return false
	}
	for i := range pp {
		if pp[i] == "*" {
			continue
		}
		if pp[i] != sp[i] {
			return false
		}
	}
	return true
}
