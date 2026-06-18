package config

import (
	"time"

	pkgconfig "github.com/sahiy/sahiy-stream/pkg/config"
)

type Config struct {
	AppEnv             string
	LogLevel           string
	GRPCAddr           string
	HTTPAddr           string
	DatabaseURL        string
	RedisURL           string
	JWTAccess          string
	JWTRefresh         string
	JWTAccessTTL       time.Duration
	JWTRefreshTTL      time.Duration
	UserCacheTTL       time.Duration
	GRPCRequestTimeout time.Duration
}

func Load() Config {
	return Config{
		AppEnv:             pkgconfig.Get("APP_ENV", "development"),
		LogLevel:           pkgconfig.Get("LOG_LEVEL", "info"),
		GRPCAddr:           pkgconfig.Get("AUTH_GRPC_ADDR", ":50051"),
		HTTPAddr:           pkgconfig.Get("AUTH_HTTP_ADDR", ":9081"),
		DatabaseURL:        pkgconfig.Get("DATABASE_URL", "postgres://sahiy:sahiy_secret@localhost:5433/sahiy_stream?sslmode=disable"),
		RedisURL:           pkgconfig.Get("REDIS_URL", "redis://localhost:6379/0"),
		JWTAccess:          pkgconfig.Get("JWT_ACCESS_SECRET", "dev-access-secret-change-in-production-32"),
		JWTRefresh:         pkgconfig.Get("JWT_REFRESH_SECRET", "dev-refresh-secret-change-in-production-32"),
		JWTAccessTTL:       pkgconfig.Duration("JWT_ACCESS_TTL", 24*time.Hour),
		JWTRefreshTTL:      pkgconfig.Duration("JWT_REFRESH_TTL", 720*time.Hour),
		UserCacheTTL:       pkgconfig.Duration("AUTH_USER_CACHE_TTL", 5*time.Minute),
		GRPCRequestTimeout: pkgconfig.Duration("GRPC_REQUEST_TIMEOUT", 10*time.Second),
	}
}

// Validate fails fast when production secrets are missing or insecure.
func (c Config) Validate() error {
	return pkgconfig.ValidateProductionSecrets(c.AppEnv, map[string]string{
		"JWT_ACCESS_SECRET":  c.JWTAccess,
		"JWT_REFRESH_SECRET": c.JWTRefresh,
	})
}
