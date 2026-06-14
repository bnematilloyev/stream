package config

import (
	pkgconfig "github.com/sahiy/sahiy-stream/pkg/config"
	"github.com/sahiy/sahiy-stream/pkg/storage"
)

type Config struct {
	AppEnv       string
	LogLevel     string
	HTTPAddr     string
	WorkerID     string
	MaxJobs      int
	NATSURL      string
	FFmpegPath   string
	VideoEncoder string
	TranscodeQuality string
	HLSOutputDir string
	Storage      storage.Config
}

func Load() Config {
	hlsDir := pkgconfig.Get("HLS_OUTPUT_DIR", "./data/hls")
	return Config{
		AppEnv:       pkgconfig.Get("APP_ENV", "development"),
		LogLevel:     pkgconfig.Get("LOG_LEVEL", "info"),
		HTTPAddr:     pkgconfig.Get("WORKER_HTTP_ADDR", ":9086"),
		WorkerID:     pkgconfig.Get("WORKER_ID", ""),
		MaxJobs:      pkgconfig.IntEnv("WORKER_MAX_JOBS", 4),
		NATSURL:      pkgconfig.Get("NATS_URL", "nats://localhost:4222"),
		FFmpegPath:   pkgconfig.Get("FFMPEG_PATH", "ffmpeg"),
		VideoEncoder: pkgconfig.Get("FFMPEG_VIDEO_ENCODER", "libx264"),
		TranscodeQuality: pkgconfig.Get("TRANSCODE_QUALITY", "production"),
		HLSOutputDir: hlsDir,
		Storage: storage.Config{
			Backend:       pkgconfig.Get("HLS_STORAGE_BACKEND", storage.BackendLocal),
			Endpoint:      pkgconfig.Get("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey:     pkgconfig.Get("MINIO_ACCESS_KEY", "sahiy_minio"),
			SecretKey:     pkgconfig.Get("MINIO_SECRET_KEY", "sahiy_minio_secret"),
			Bucket:        pkgconfig.Get("MINIO_BUCKET", "sahiy-media"),
			Region:        pkgconfig.Get("MINIO_REGION", "us-east-1"),
			UseSSL:        pkgconfig.Get("MINIO_USE_SSL", "false") == "true",
			PublicBaseURL: pkgconfig.Get("CDN_BASE_URL", ""),
			LocalRoot:     hlsDir,
			KeyPrefix:     "hls",
		},
	}
}
