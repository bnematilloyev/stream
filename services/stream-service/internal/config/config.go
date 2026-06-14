package config

import (
	"strconv"
	"time"

	pkgconfig "github.com/sahiy/sahiy-stream/pkg/config"
	"github.com/sahiy/sahiy-stream/pkg/analytics"
	pkgredis "github.com/sahiy/sahiy-stream/pkg/redis"
	"github.com/sahiy/sahiy-stream/pkg/storage"
)

type Config struct {
	AppEnv               string
	LogLevel             string
	GRPCAddr             string
	HTTPAddr             string
	DatabaseURL          string
	DatabaseReplicaURL   string
	DBMaxConns           int32
	DBMinConns           int32
	Redis                pkgredis.Config
	PlaybackBaseURL      string
	PlaybackSignSecret   string
	PlaybackURLTTL       time.Duration
	ViewerWindow         time.Duration
	ViewerSyncInterval   time.Duration
	HLSLocalDir          string
	Storage              storage.Config
	StaleCleanupInterval time.Duration
	GRPCRequestTimeout   time.Duration
	ClickHouse           analytics.Config
}

func int32Env(key string, fallback int32) int32 {
	raw := pkgconfig.Get(key, "")
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return int32(v)
}

func Load() Config {
	hlsDir := pkgconfig.Get("HLS_OUTPUT_DIR", "./data/hls")
	return Config{
		AppEnv:             pkgconfig.Get("APP_ENV", "development"),
		LogLevel:           pkgconfig.Get("LOG_LEVEL", "info"),
		GRPCAddr:           pkgconfig.Get("STREAM_GRPC_ADDR", ":50053"),
		HTTPAddr:           pkgconfig.Get("STREAM_HTTP_ADDR", ":9083"),
		DatabaseURL:        pkgconfig.Get("DATABASE_URL", "postgres://sahiy:sahiy_secret@localhost:5433/sahiy_stream?sslmode=disable"),
		DatabaseReplicaURL: pkgconfig.Get("DATABASE_REPLICA_URL", ""),
		DBMaxConns:         int32Env("DB_MAX_CONNS", 25),
		DBMinConns:         int32Env("DB_MIN_CONNS", 5),
		Redis: pkgredis.Config{
			URL:          pkgconfig.Get("REDIS_URL", "redis://localhost:6379/0"),
			ClusterAddrs: pkgredis.ParseClusterAddrs(pkgconfig.Get("REDIS_CLUSTER_ADDRS", "")),
			PoolSize:     int(int32Env("REDIS_POOL_SIZE", 20)),
		},
		PlaybackBaseURL:      pkgconfig.Get("PLAYBACK_BASE_URL", "http://localhost:9083"),
		PlaybackSignSecret:   pkgconfig.Get("PLAYBACK_SIGNING_SECRET", "dev-playback-signing-secret-min-32!!"),
		PlaybackURLTTL:       pkgconfig.Duration("PLAYBACK_URL_TTL", 4*time.Hour),
		ViewerWindow:         pkgconfig.Duration("VIEWER_HEARTBEAT_WINDOW", 45*time.Second),
		ViewerSyncInterval:   pkgconfig.Duration("VIEWER_SYNC_INTERVAL", 15*time.Second),
		HLSLocalDir:          hlsDir,
		StaleCleanupInterval: pkgconfig.Duration("STREAM_STALE_CLEANUP_INTERVAL", 30*time.Second),
		GRPCRequestTimeout:   pkgconfig.Duration("GRPC_REQUEST_TIMEOUT", 10*time.Second),
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
		ClickHouse: analytics.Config{
			Enabled:  pkgconfig.Get("CLICKHOUSE_ENABLED", "false") == "true",
			Addr:     pkgconfig.Get("CLICKHOUSE_ADDR", "localhost:9009"),
			Database: pkgconfig.Get("CLICKHOUSE_DATABASE", "sahiy_analytics"),
			Username: pkgconfig.Get("CLICKHOUSE_USER", "default"),
			Password: pkgconfig.Get("CLICKHOUSE_PASSWORD", ""),
		},
	}
}

func (c Config) Validate() error {
	return pkgconfig.ValidateProductionSecrets(c.AppEnv, map[string]string{
		"PLAYBACK_SIGNING_SECRET": c.PlaybackSignSecret,
	})
}
