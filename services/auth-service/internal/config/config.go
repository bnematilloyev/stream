package config

import (
	"time"

	pkgconfig "github.com/sahiy/sahiy-stream/pkg/config"
)

type Config struct {
	AppEnv        string
	LogLevel      string
	GRPCAddr      string
	HTTPAddr      string
	DatabaseURL   string
	RedisURL      string
	JWTAccess     string
	JWTRefresh    string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration
}

func Load() Config {
	return Config{
		AppEnv:        pkgconfig.Get("APP_ENV", "development"),
		LogLevel:      pkgconfig.Get("LOG_LEVEL", "info"),
		GRPCAddr:      pkgconfig.Get("AUTH_GRPC_ADDR", ":50051"),
		HTTPAddr:      pkgconfig.Get("AUTH_HTTP_ADDR", ":9081"),
		DatabaseURL:   pkgconfig.Get("DATABASE_URL", "postgres://sahiy:sahiy_secret@localhost:5433/sahiy_stream?sslmode=disable"),
		RedisURL:      pkgconfig.Get("REDIS_URL", "redis://localhost:6379/0"),
		JWTAccess:     pkgconfig.Get("JWT_ACCESS_SECRET", "dev-access-secret-change-in-production-32"),
		JWTRefresh:    pkgconfig.Get("JWT_REFRESH_SECRET", "dev-refresh-secret-change-in-production-32"),
		JWTAccessTTL:  pkgconfig.Duration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL: pkgconfig.Duration("JWT_REFRESH_TTL", 168*time.Hour),
	}
}
