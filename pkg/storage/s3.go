package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type s3Storage struct {
	client        *minio.Client
	bucket        string
	prefix        string
	publicBaseURL string
	localRoot     string
}

func NewS3Storage(cfg Config) (*s3Storage, error) {
	endpoint := cfg.Endpoint
	secure := cfg.UseSSL

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: secure,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}

	publicBase := strings.TrimRight(cfg.PublicBaseURL, "/")
	if publicBase == "" {
		scheme := "http"
		if secure {
			scheme = "https"
		}
		publicBase = fmt.Sprintf("%s://%s/%s", scheme, endpoint, cfg.Bucket)
	}

	return &s3Storage{
		client:        client,
		bucket:        cfg.Bucket,
		prefix:        defaultPrefix(cfg.KeyPrefix),
		publicBaseURL: publicBase,
		localRoot:     cfg.LocalRoot,
	}, nil
}

func (s *s3Storage) Backend() string { return BackendS3 }

func (s *s3Storage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
}

func (s *s3Storage) UploadFile(ctx context.Context, key, localPath, contentType string) error {
	_, err := s.client.FPutObject(ctx, s.bucket, key, localPath, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (s *s3Storage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (s *s3Storage) ObjectURL(key string) string {
	u, err := url.JoinPath(s.publicBaseURL, key)
	if err != nil {
		return s.publicBaseURL + "/" + key
	}
	return u
}

func (s *s3Storage) ResolveKey(streamID, relativePath string) string {
	return filepath.ToSlash(filepath.Join(s.prefix, streamID, relativePath))
}

func (s *s3Storage) LocalPath(streamID, relativePath string) string {
	root := s.localRoot
	if root == "" {
		root = "./data/hls"
	}
	return filepath.Join(root, streamID, filepath.FromSlash(relativePath))
}

func (s *s3Storage) PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if expiry <= 0 {
		expiry = time.Hour
	}
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, expiry, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func ObjectExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
