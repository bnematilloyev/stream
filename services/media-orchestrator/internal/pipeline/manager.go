package pipeline

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sahiy/sahiy-stream/pkg/hlssync"
	"github.com/sahiy/sahiy-stream/pkg/storage"
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/domain"
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/transcode"
	"go.uber.org/zap"
)

type StreamClient interface {
	ValidateStreamKey(ctx context.Context, key string) (valid bool, channelID, streamID string, err error)
	StartIngest(ctx context.Context, channelID string) (streamID string, err error)
	EndIngest(ctx context.Context, streamID string) error
	GetStream(ctx context.Context, streamID string) (latencyMode, status string, err error)
}

type Manager struct {
	mu             sync.Mutex
	active         map[string]*session
	backend        transcode.Backend
	encoder        string
	quality        string
	storageBackend string
	media          domain.StreamMediaRepository
	stream         StreamClient
	storage        storage.ObjectStorage
	syncSegments   bool
	rtmpBase       string
	rtspBase       string
	hlsDir         string
	log            *zap.Logger
}

type session struct {
	streamID       uuid.UUID
	ingestName     string
	jobID          string
	cmd            *exec.Cmd
	uploaderCancel context.CancelFunc
	startedAt      time.Time
}

func NewManager(
	backend transcode.Backend,
	encoder, quality string,
	media domain.StreamMediaRepository,
	stream StreamClient,
	store storage.ObjectStorage,
	syncSegments bool,
	rtmpBase, rtspBase, hlsDir string,
	log *zap.Logger,
) *Manager {
	storageBackend := storage.BackendLocal
	if store != nil {
		storageBackend = store.Backend()
	}
	return &Manager{
		active:         make(map[string]*session),
		backend:        backend,
		encoder:        encoder,
		quality:        quality,
		storageBackend: storageBackend,
		media:          media,
		stream:         stream,
		storage:        store,
		syncSegments:   syncSegments,
		rtmpBase:       rtmpBase,
		rtspBase:       rtspBase,
		hlsDir:         hlsDir,
		log:            log,
	}
}

func (m *Manager) OnPublish(ctx context.Context, ingestName, source string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.active[ingestName]; ok {
		return nil
	}

	streamID, latencyMode, err := m.resolveStream(ctx, ingestName)
	if err != nil {
		return err
	}

	sid, err := uuid.Parse(streamID)
	if err != nil {
		return fmt.Errorf("invalid stream id: %w", err)
	}

	inputURL, latencyMode := m.buildInputURL(ingestName, source, latencyMode)
	outDir := filepath.Join(m.hlsDir, sid.String())

	var job *transcode.RunningJob
	var uploaderCancel context.CancelFunc
	job, err = m.backend.Start(ctx, transcode.StartRequest{
		StreamID: sid.String(), IngestName: ingestName, InputURL: inputURL,
		OutputDir: outDir, LatencyMode: latencyMode, Quality: m.quality,
		Encoder: m.encoder, Storage: m.storageBackend,
	})
	if err != nil {
		// WHIP ultra-low can play via WHEP without HLS when FFmpeg is unavailable.
		if source == "whip" {
			m.log.Warn("ffmpeg unavailable, whip-only ingest (no HLS)",
				zap.String("stream_id", sid.String()),
				zap.Error(err),
			)
			job = &transcode.RunningJob{JobID: sid.String(), StreamID: sid.String()}
		} else {
			return err
		}
	}

	if job.Cmd != nil && m.syncSegments && m.storage != nil && m.storage.Backend() == storage.BackendS3 {
		var uploaderCtx context.Context
		uploaderCtx, uploaderCancel = context.WithCancel(context.Background())
		go hlssync.NewSegmentUploader(m.storage, outDir, sid, 2*time.Second, m.log).Run(uploaderCtx)
	}

	now := time.Now()
	hlsPath := outDir
	playbackResource := fmt.Sprintf("%s/master.m3u8", sid.String())
	pid := job.FFmpegPID

	if err := m.media.Upsert(ctx, &domain.StreamMedia{
		StreamID: sid, Status: domain.StatusIngesting,
		HLSPath: &hlsPath, PlaybackURL: &playbackResource, IngestName: &ingestName,
		FFmpegPID: &pid, StartedAt: &now,
	}); err != nil {
		if uploaderCancel != nil {
			uploaderCancel()
		}
		if job.Cmd != nil {
			_ = transcode.StopCmd(job.Cmd)
		}
		if job.Cmd != nil {
			_ = m.backend.Stop(ctx, sid.String(), "rollback")
		}
		return err
	}

	m.active[ingestName] = &session{
		streamID: sid, ingestName: ingestName, jobID: job.JobID, cmd: job.Cmd,
		uploaderCancel: uploaderCancel, startedAt: now,
	}
	m.log.Info("ingest started",
		zap.String("stream_id", sid.String()),
		zap.String("job_id", job.JobID),
		zap.String("ingest", ingestName),
		zap.String("source", source),
		zap.String("latency_mode", latencyMode),
		zap.String("storage", m.storageBackend),
	)
	return nil
}

func (m *Manager) OnPublishDone(ctx context.Context, ingestName string) error {
	m.mu.Lock()
	sess, ok := m.active[ingestName]
	if !ok {
		m.mu.Unlock()
		return nil
	}
	delete(m.active, ingestName)
	m.mu.Unlock()

	if sess.uploaderCancel != nil {
		sess.uploaderCancel()
	}
	if sess.cmd != nil {
		_ = transcode.StopCmd(sess.cmd)
	}
	_ = m.backend.Stop(ctx, sess.streamID.String(), "publish_done")

	now := time.Now()
	stopped := domain.StatusStopped
	if err := m.media.Upsert(ctx, &domain.StreamMedia{
		StreamID: sess.streamID, Status: stopped, StoppedAt: &now,
	}); err != nil {
		m.log.Warn("media upsert on stop failed", zap.Error(err))
	}

	if err := m.stream.EndIngest(ctx, sess.streamID.String()); err != nil {
		m.log.Warn("end ingest failed", zap.Error(err))
	}

	m.log.Info("ingest stopped", zap.String("stream_id", sess.streamID.String()))
	return nil
}

func (m *Manager) resolveStream(ctx context.Context, ingestName string) (streamID, latencyMode string, err error) {
	if sid, parseErr := uuid.Parse(ingestName); parseErr == nil {
		lm, st, getErr := m.stream.GetStream(ctx, sid.String())
		if getErr != nil {
			return "", "", getErr
		}
		if st != "live" && st != "scheduled" {
			return "", "", fmt.Errorf("stream not publishable")
		}
		return sid.String(), lm, nil
	}

	valid, channelID, existingStreamID, err := m.stream.ValidateStreamKey(ctx, ingestName)
	if err != nil {
		return "", "", err
	}
	if !valid {
		return "", "", fmt.Errorf("invalid stream key")
	}

	streamID = existingStreamID
	if streamID == "" {
		streamID, err = m.stream.StartIngest(ctx, channelID)
		if err != nil {
			return "", "", err
		}
	}
	if lm, _, err := m.stream.GetStream(ctx, streamID); err == nil {
		latencyMode = lm
	}
	return streamID, latencyMode, nil
}

func (m *Manager) buildInputURL(ingestName, source, latencyMode string) (string, string) {
	if source == "whip" {
		time.Sleep(1200 * time.Millisecond)
		if latencyMode == "" || latencyMode == "ultra-low" {
			latencyMode = "ultra-low"
		}
		return fmt.Sprintf("%s/%s", m.rtspBase, ingestName), latencyMode
	}
	if latencyMode == "" {
		latencyMode = "standard"
	}
	return fmt.Sprintf("%s/%s", m.rtmpBase, ingestName), latencyMode
}
