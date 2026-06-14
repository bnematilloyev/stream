package storage

import (
	"context"
	"fmt"
	"io"
)

const (
	BackendLocal = "local"
	BackendS3    = "s3"
)

// Config holds object storage connection settings.
type Config struct {
	Backend       string
	Endpoint      string
	AccessKey     string
	SecretKey     string
	Bucket        string
	Region        string
	UseSSL        bool
	PublicBaseURL string
	LocalRoot     string
	KeyPrefix     string
}

// ObjectStorage abstracts HLS segment persistence (Strategy pattern).
type ObjectStorage interface {
	EnsureBucket(ctx context.Context) error
	UploadFile(ctx context.Context, key, localPath, contentType string) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	ObjectURL(key string) string
	ResolveKey(streamID, relativePath string) string
	LocalPath(streamID, relativePath string) string
	Backend() string
}

// New creates the configured storage backend.
func New(cfg Config) (ObjectStorage, error) {
	switch cfg.Backend {
	case BackendS3, "minio":
		return NewS3Storage(cfg)
	case BackendLocal, "":
		return NewLocalStorage(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported storage backend: %s", cfg.Backend)
	}
}

func defaultPrefix(prefix string) string {
	if prefix == "" {
		return "hls"
	}
	return prefix
}
