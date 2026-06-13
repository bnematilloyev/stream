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
	HLSBaseURL  string
}

func Load() Config {
	return Config{
		AppEnv:      pkgconfig.Get("APP_ENV", "development"),
		LogLevel:    pkgconfig.Get("LOG_LEVEL", "info"),
		GRPCAddr:    pkgconfig.Get("STREAM_GRPC_ADDR", ":50053"),
		HTTPAddr:    pkgconfig.Get("STREAM_HTTP_ADDR", ":9083"),
		DatabaseURL: pkgconfig.Get("DATABASE_URL", "postgres://sahiy:sahiy_secret@localhost:5433/sahiy_stream?sslmode=disable"),
		HLSBaseURL:  pkgconfig.Get("HLS_BASE_URL", "http://localhost:8090/hls"),
	}
}
