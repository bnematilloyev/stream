package config

import (
	"strings"
	"time"

	pkgconfig "github.com/sahiy/sahiy-stream/pkg/config"
)

type Config struct {
	AppEnv         string
	LogLevel       string
	HTTPAddr       string
	AuthService    string
	UserService    string
	StreamService  string
	RedisURL       string
	CORSOrigins    []string
	RateLimitRPM   int
	RequestTimeout time.Duration
	WhipBaseURL    string
}

func parseOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if o := strings.TrimSpace(p); o != "" {
			out = append(out, o)
		}
	}
	return out
}

func Load() Config {
	origins := pkgconfig.Get(
		"GATEWAY_CORS_ORIGINS",
		"http://localhost:3000,http://localhost:3001,http://127.0.0.1:3000,http://127.0.0.1:3001",
	)
	return Config{
		AppEnv:         pkgconfig.Get("APP_ENV", "development"),
		LogLevel:       pkgconfig.Get("LOG_LEVEL", "info"),
		HTTPAddr:       pkgconfig.Get("GATEWAY_HTTP_ADDR", ":8080"),
		AuthService:    pkgconfig.Get("AUTH_SERVICE_ADDR", "localhost:50051"),
		UserService:    pkgconfig.Get("USER_SERVICE_ADDR", "localhost:50052"),
		StreamService:  pkgconfig.Get("STREAM_SERVICE_ADDR", "localhost:50053"),
		RedisURL:       pkgconfig.Get("REDIS_URL", "redis://localhost:6379/0"),
		CORSOrigins:    parseOrigins(origins),
		RateLimitRPM:   100,
		RequestTimeout: 10 * time.Second,
		WhipBaseURL:    pkgconfig.Get("WHIP_BASE_URL", "http://localhost:8889"),
	}
}
