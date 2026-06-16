package hlssync

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sahiy/sahiy-stream/pkg/storage"
	"go.uber.org/zap"
)

// SegmentUploader mirrors local HLS output to object storage.
type SegmentUploader struct {
	storage  storage.ObjectStorage
	localDir string
	streamID uuid.UUID
	interval time.Duration
	uploaded sync.Map
	log      *zap.Logger
}

func NewSegmentUploader(store storage.ObjectStorage, localDir string, streamID uuid.UUID, interval time.Duration, log *zap.Logger) *SegmentUploader {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &SegmentUploader{
		storage:  store,
		localDir: localDir,
		streamID: streamID,
		interval: interval,
		log:      log,
	}
}

func (u *SegmentUploader) Run(ctx context.Context) {
	ticker := time.NewTicker(u.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			u.syncOnce(ctx)
		}
	}
}

func (u *SegmentUploader) syncOnce(ctx context.Context) {
	_ = filepath.Walk(u.localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		if !storage.IsMediaFile(name) {
			return nil
		}

		rel, err := filepath.Rel(u.localDir, path)
		if err != nil {
			return nil
		}
		key := u.storage.ResolveKey(u.streamID.String(), filepath.ToSlash(rel))

		if storage.IsPlaylist(name) {
			return u.upload(ctx, key, path, name)
		}
		if _, exists := u.uploaded.Load(key); exists {
			return nil
		}
		if time.Since(info.ModTime()) < 500*time.Millisecond {
			return nil
		}
		if err := u.upload(ctx, key, path, name); err != nil {
			return nil
		}
		u.uploaded.Store(key, true)
		return nil
	})
}

func (u *SegmentUploader) upload(ctx context.Context, key, path, name string) error {
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		return nil
	}
	if err := u.storage.UploadFile(ctx, key, path, storage.ContentType(name)); err != nil {
		u.log.Warn("segment upload failed", zap.String("key", key), zap.Error(err))
		return err
	}
	return nil
}
