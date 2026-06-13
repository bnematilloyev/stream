package config

import (
	pkgconfig "github.com/sahiy/sahiy-stream/pkg/config"
)

type Config struct {
	AppEnv      string
	LogLevel    string
	GRPCAddr    string
	HTTPAddr    string
	DatabaseURL string
	RedisURL    string
	RTMPBaseURL string
	SRTBaseURL  string
}

func Load() Config {
	return Config{
		AppEnv:      pkgconfig.Get("APP_ENV", "development"),
		LogLevel:    pkgconfig.Get("LOG_LEVEL", "info"),
		GRPCAddr:    pkgconfig.Get("USER_GRPC_ADDR", ":50052"),
		HTTPAddr:    pkgconfig.Get("USER_HTTP_ADDR", ":9082"),
		DatabaseURL: pkgconfig.Get("DATABASE_URL", "postgres://sahiy:sahiy_secret@localhost:5433/sahiy_stream?sslmode=disable"),
		RedisURL:    pkgconfig.Get("REDIS_URL", "redis://localhost:6379/0"),
		RTMPBaseURL: pkgconfig.Get("RTMP_BASE_URL", "rtmp://ingest.sahiy.stream/live"),
		SRTBaseURL:  pkgconfig.Get("SRT_BASE_URL", "srt://ingest.sahiy.stream:9000"),
	}
}
