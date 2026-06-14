package config

import (
	"strings"
	"time"

	pkgconfig "github.com/sahiy/sahiy-stream/pkg/config"
	pkgredis "github.com/sahiy/sahiy-stream/pkg/redis"
)

type Config struct {
	AppEnv             string
	LogLevel           string
	GRPCAddr           string
	HTTPAddr           string
	DatabaseURL        string
	Redis              pkgredis.Config
	NATSURL            string
	AuthServiceAddr    string
	UserServiceAddr    string
	StreamServiceAddr  string
	CORSOrigins        []string
	ChatRateLimit      int
	JWTAccessSecret    string
	JWTRefreshSecret   string
	JWTAccessTTL       time.Duration
	JWTRefreshTTL      time.Duration
	UserCacheTTL       time.Duration
	GRPCRequestTimeout time.Duration
}

func Load() Config {
	origins := pkgconfig.Get(
		"GATEWAY_CORS_ORIGINS",
		"http://localhost:3000,http://localhost:3001,http://127.0.0.1:3000,http://127.0.0.1:3001",
	)
	return Config{
		AppEnv:            pkgconfig.Get("APP_ENV", "development"),
		LogLevel:          pkgconfig.Get("LOG_LEVEL", "info"),
		GRPCAddr:          pkgconfig.Get("CHAT_GRPC_ADDR", ":50054"),
		HTTPAddr:          pkgconfig.Get("CHAT_HTTP_ADDR", ":9085"),
		DatabaseURL:       pkgconfig.Get("DATABASE_URL", "postgres://sahiy:sahiy_secret@localhost:5433/sahiy_stream?sslmode=disable"),
		NATSURL:           pkgconfig.Get("NATS_URL", "nats://localhost:4222"),
		AuthServiceAddr:   pkgconfig.Get("AUTH_SERVICE_ADDR", "localhost:50051"),
		UserServiceAddr:   pkgconfig.Get("USER_SERVICE_ADDR", "localhost:50052"),
		StreamServiceAddr: pkgconfig.Get("STREAM_SERVICE_ADDR", "localhost:50053"),
		CORSOrigins:       parseOrigins(origins),
		ChatRateLimit:     pkgconfig.IntEnv("CHAT_RATE_LIMIT_PER_SEC", 5),
		JWTAccessSecret:   pkgconfig.Get("JWT_ACCESS_SECRET", "dev-access-secret-change-in-production-32"),
		JWTRefreshSecret:  pkgconfig.Get("JWT_REFRESH_SECRET", "dev-refresh-secret-change-in-production-32"),
		JWTAccessTTL:      pkgconfig.Duration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:     pkgconfig.Duration("JWT_REFRESH_TTL", 168*time.Hour),
		UserCacheTTL:      pkgconfig.Duration("AUTH_USER_CACHE_TTL", 5*time.Minute),
		GRPCRequestTimeout: pkgconfig.Duration("GRPC_REQUEST_TIMEOUT", 10*time.Second),
		Redis: pkgredis.Config{
			URL:      pkgconfig.Get("REDIS_URL", "redis://localhost:6379/0"),
			PoolSize: pkgconfig.IntEnv("REDIS_POOL_SIZE", 20),
		},
	}
}

func (c Config) Validate() error {
	return pkgconfig.ValidateProductionSecrets(c.AppEnv, map[string]string{
		"JWT_ACCESS_SECRET":  c.JWTAccessSecret,
		"JWT_REFRESH_SECRET": c.JWTRefreshSecret,
	})
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
