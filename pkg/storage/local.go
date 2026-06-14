package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type localStorage struct {
	root   string
	prefix string
}

func NewLocalStorage(cfg Config) ObjectStorage {
	root := cfg.LocalRoot
	if root == "" {
		root = "./data/hls"
	}
	return &localStorage{root: root, prefix: defaultPrefix(cfg.KeyPrefix)}
}

func (s *localStorage) Backend() string { return BackendLocal }

func (s *localStorage) EnsureBucket(_ context.Context) error { return nil }

func (s *localStorage) UploadFile(_ context.Context, _, _, _ string) error { return nil }

func (s *localStorage) Open(_ context.Context, key string) (io.ReadCloser, error) {
	path, err := s.keyToPath(key)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (s *localStorage) ObjectURL(key string) string {
	return key
}

func (s *localStorage) ResolveKey(streamID, relativePath string) string {
	return filepath.ToSlash(filepath.Join(s.prefix, streamID, relativePath))
}

func (s *localStorage) LocalPath(streamID, relativePath string) string {
	return filepath.Join(s.root, streamID, filepath.FromSlash(relativePath))
}

func (s *localStorage) keyToPath(key string) (string, error) {
	key = filepath.ToSlash(key)
	prefix := s.prefix + "/"
	if !strings.HasPrefix(key, prefix) {
		return "", fmt.Errorf("invalid storage key: %s", key)
	}
	rest := strings.TrimPrefix(key, prefix)
	return filepath.Join(append([]string{s.root}, strings.Split(rest, "/")...)...), nil
}
