package config

import (
	"fmt"

	pkgconfig "github.com/sahiy/sahiy-stream/pkg/config"
	"github.com/sahiy/sahiy-stream/pkg/storage"
)

type Config struct {
	AppEnv             string
	LogLevel           string
	HTTPAddr           string
	DatabaseURL        string
	StreamServiceAddr  string
	RTMPInternalURL    string
	RTMPWorkerURL      string
	RTSPInternalURL    string
	RTSPWorkerURL      string
	HLSOutputDir       string
	HLSBaseURL         string
	FFmpegPath         string
	FFmpegVideoEncoder string
	TranscodeQuality   string
	HookSecret         string
	TranscodeMode      string
	NATSURL            string
	Storage            storage.Config
}

func Load() Config {
	hlsDir := pkgconfig.Get("HLS_OUTPUT_DIR", "./data/hls")
	return Config{
		AppEnv:             pkgconfig.Get("APP_ENV", "development"),
		LogLevel:           pkgconfig.Get("LOG_LEVEL", "info"),
		HTTPAddr:           pkgconfig.Get("MEDIA_HTTP_ADDR", ":9084"),
		DatabaseURL:        pkgconfig.Get("DATABASE_URL", "postgres://sahiy:sahiy_secret@localhost:5433/sahiy_stream?sslmode=disable"),
		StreamServiceAddr:  pkgconfig.Get("STREAM_SERVICE_ADDR", "localhost:50053"),
		RTMPInternalURL:    pkgconfig.Get("RTMP_INTERNAL_URL", "rtmp://127.0.0.1:1935/live"),
		RTMPWorkerURL:      pkgconfig.Get("RTMP_BASE_URL", pkgconfig.Get("RTMP_INTERNAL_URL", "rtmp://127.0.0.1:1935/live")),
		RTSPInternalURL:    pkgconfig.Get("RTSP_INTERNAL_URL", "rtsp://127.0.0.1:8554"),
		RTSPWorkerURL:      pkgconfig.Get("RTSP_WORKER_URL", pkgconfig.Get("RTSP_INTERNAL_URL", "rtsp://127.0.0.1:8554")),
		HLSOutputDir:       hlsDir,
		HLSBaseURL:         pkgconfig.Get("HLS_BASE_URL", "http://localhost:8090/hls"),
		FFmpegPath:         pkgconfig.Get("FFMPEG_PATH", "ffmpeg"),
		FFmpegVideoEncoder: pkgconfig.Get("FFMPEG_VIDEO_ENCODER", "libx264"),
		TranscodeQuality:   pkgconfig.Get("TRANSCODE_QUALITY", "production"),
		HookSecret:         pkgconfig.Get("MEDIA_HOOK_SECRET", "dev-media-hook-secret-min-32-chars!!"),
		TranscodeMode:      pkgconfig.Get("TRANSCODE_MODE", "local"),
		NATSURL:            pkgconfig.Get("NATS_URL", "nats://localhost:4222"),
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

func (c Config) Validate() error {
	if err := pkgconfig.ValidateProductionSecrets(c.AppEnv, map[string]string{
		"MEDIA_HOOK_SECRET": c.HookSecret,
	}); err != nil {
		return err
	}
	if c.TranscodeMode == "queue" {
		if c.NATSURL == "" {
			return fmt.Errorf("TRANSCODE_MODE=queue requires NATS_URL")
		}
		if c.Storage.Backend != storage.BackendS3 {
			return fmt.Errorf("TRANSCODE_MODE=queue requires HLS_STORAGE_BACKEND=s3 (worker uploads segments)")
		}
	}
	return nil
}

func (c Config) HookAuth() (secret string, requireSecret bool, allowInternal bool) {
	requireSecret = pkgconfig.IsProduction(c.AppEnv)
	allowInternal = !requireSecret
	return c.HookSecret, requireSecret, allowInternal
}

func (c Config) SyncSegments() bool {
	return c.Storage.Backend == storage.BackendS3
}
