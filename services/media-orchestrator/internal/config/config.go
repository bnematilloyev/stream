package config

import (
	pkgconfig "github.com/sahiy/sahiy-stream/pkg/config"
)

type Config struct {
	AppEnv           string
	LogLevel         string
	HTTPAddr         string
	DatabaseURL      string
	StreamServiceAddr string
	RTMPInternalURL  string
	RTSPInternalURL  string
	HLSOutputDir     string
	HLSBaseURL       string
	FFmpegPath       string
}

func Load() Config {
	return Config{
		AppEnv:            pkgconfig.Get("APP_ENV", "development"),
		LogLevel:          pkgconfig.Get("LOG_LEVEL", "info"),
		HTTPAddr:          pkgconfig.Get("MEDIA_HTTP_ADDR", ":9084"),
		DatabaseURL:       pkgconfig.Get("DATABASE_URL", "postgres://sahiy:sahiy_secret@localhost:5433/sahiy_stream?sslmode=disable"),
		StreamServiceAddr: pkgconfig.Get("STREAM_SERVICE_ADDR", "localhost:50053"),
		RTMPInternalURL:   pkgconfig.Get("RTMP_INTERNAL_URL", "rtmp://127.0.0.1:1935/live"),
		RTSPInternalURL:   pkgconfig.Get("RTSP_INTERNAL_URL", "rtsp://127.0.0.1:8554"),
		HLSOutputDir:      pkgconfig.Get("HLS_OUTPUT_DIR", "./data/hls"),
		HLSBaseURL:        pkgconfig.Get("HLS_BASE_URL", "http://localhost:8090/hls"),
		FFmpegPath:        pkgconfig.Get("FFMPEG_PATH", "ffmpeg"),
	}
}
