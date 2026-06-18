package pipeline

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sahiy/sahiy-stream/pkg/hlsrecord"
	"github.com/sahiy/sahiy-stream/pkg/hlssync"
	"github.com/sahiy/sahiy-stream/pkg/storage"
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/domain"
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/transcode"
	"go.uber.org/zap"
)

type StreamClient interface {
	ValidateStreamKey(ctx context.Context, key string) (valid bool, channelID, streamID string, err error)
	StartIngest(ctx context.Context, channelID string) (streamID string, err error)
	StartIngestStream(ctx context.Context, streamID string) (startedStreamID string, err error)
	EndIngest(ctx context.Context, streamID string) error
	GetStream(ctx context.Context, streamID string) (channelID, latencyMode, status string, err error)
}

const whipEndIngestGrace = 90 * time.Second

type Manager struct {
	mu             sync.Mutex
	active         map[string]*session
	pendingEndMu   sync.Mutex
	pendingEnd     map[string]*time.Timer
	backend        transcode.Backend
	encoder        string
	quality        string
	storageBackend string
	media          domain.StreamMediaRepository
	stream         StreamClient
	storage        storage.ObjectStorage
	syncSegments   bool
	rtmpBase       string
	rtmpWorkerBase string
	rtspBase       string
	rtspWorkerBase string
	hlsDir         string
	log            *zap.Logger
}

type session struct {
	streamID       uuid.UUID
	ingestName     string
	jobID          string
	outputDir      string
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
	rtmpBase, rtmpWorkerBase, rtspBase, rtspWorkerBase, hlsDir string,
	log *zap.Logger,
) *Manager {
	storageBackend := storage.BackendLocal
	if store != nil {
		storageBackend = store.Backend()
	}
	return &Manager{
		active:         make(map[string]*session),
		pendingEnd:     make(map[string]*time.Timer),
		backend:        backend,
		encoder:        encoder,
		quality:        quality,
		storageBackend: storageBackend,
		media:          media,
		stream:         stream,
		storage:        store,
		syncSegments:   syncSegments,
		rtmpBase:       rtmpBase,
		rtmpWorkerBase: rtmpWorkerBase,
		rtspBase:       rtspBase,
		rtspWorkerBase: rtspWorkerBase,
		hlsDir:         hlsDir,
		log:            log,
	}
}

func (m *Manager) PreparePublish(ctx context.Context, ingestName string) error {
	_, _, err := m.resolveStream(ctx, ingestName)
	return err
}

func isWhipIngestName(ingestName string) bool {
	_, err := uuid.Parse(ingestName)
	return err == nil
}

func (m *Manager) cancelPendingEndIngest(ingestName string) {
	m.pendingEndMu.Lock()
	defer m.pendingEndMu.Unlock()
	if t, ok := m.pendingEnd[ingestName]; ok {
		t.Stop()
		delete(m.pendingEnd, ingestName)
	}
}

func (m *Manager) scheduleWhipEndIngest(ingestName, streamID string) {
	m.cancelPendingEndIngest(ingestName)
	m.pendingEndMu.Lock()
	defer m.pendingEndMu.Unlock()
	t := time.AfterFunc(whipEndIngestGrace, func() {
		m.pendingEndMu.Lock()
		delete(m.pendingEnd, ingestName)
		m.pendingEndMu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if sid, err := uuid.Parse(ingestName); err == nil {
			now := time.Now()
			outDir := filepath.Join(m.hlsDir, sid.String())
			if err := m.media.Upsert(ctx, &domain.StreamMedia{
				StreamID: sid, Status: domain.StatusStopped, StoppedAt: &now, HLSPath: &outDir,
			}); err != nil {
				m.log.Warn("whip grace media upsert failed", zap.Error(err))
			}
		}

		if err := m.stream.EndIngest(ctx, streamID); err != nil {
			m.log.Warn("delayed whip end ingest failed",
				zap.String("stream_id", streamID),
				zap.Error(err),
			)
		} else {
			m.log.Info("whip ingest ended after grace period",
				zap.String("stream_id", streamID),
			)
		}
	})
	m.pendingEnd[ingestName] = t
}

func (m *Manager) OnPublish(ctx context.Context, ingestName, source string) error {
	m.cancelPendingEndIngest(ingestName)

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

	if source == "whip" {
		now := time.Now()
		name := ingestName
		if err := m.media.Upsert(ctx, &domain.StreamMedia{
			StreamID: sid, Status: domain.StatusIngesting,
			IngestName: &name, StartedAt: &now,
		}); err != nil {
			m.log.Warn("whip early media upsert failed", zap.Error(err))
		}
	}

	inputURL, latencyMode := m.buildInputURL(ingestName, source, latencyMode)
	if source == "rtmp" {
		readyURL, err := waitForRTMPPublisher(ctx, m.rtmpBase, ingestName, m.log)
		if err != nil {
			return err
		}
		// Pull via the URL that passed ffprobe (usually 127.0.0.1), not the public
		// RTMP_BASE_URL shown to OBS — hairpin to the VPS public IP often fails.
		inputURL = readyURL
	}
	if source == "whip" {
		readyURL, err := waitForRTSPPublisher(ctx, m.rtspBase, ingestName, m.log)
		if err != nil {
			return err
		}
		inputURL = readyURL
	}
	outDir := filepath.Join(m.hlsDir, sid.String())

	var job *transcode.RunningJob
	var uploaderCancel context.CancelFunc
	job, err = m.backend.Start(ctx, transcode.StartRequest{
		StreamID: sid.String(), IngestName: ingestName, InputURL: inputURL,
		OutputDir: outDir, LatencyMode: latencyMode, Quality: m.quality,
		Encoder: m.encoder, Storage: m.storageBackend,
	})
	if err != nil {
		if source == "whip" && latencyMode == "ultra-low" {
			m.log.Warn("whip hls transcode unavailable, keeping whep-only ingest",
				zap.String("stream_id", sid.String()),
				zap.Error(err),
			)
			m.active[ingestName] = &session{
				streamID: sid, ingestName: ingestName, outputDir: outDir, startedAt: time.Now(),
			}
			return nil
		}
		return fmt.Errorf("start transcode for %s: %w", source, err)
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
		streamID: sid, ingestName: ingestName, jobID: job.JobID, outputDir: outDir, cmd: job.Cmd,
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
	whipIngest := isWhipIngestName(ingestName)

	m.mu.Lock()
	sess, ok := m.active[ingestName]
	if ok {
		delete(m.active, ingestName)
	}
	m.mu.Unlock()

	if !ok {
		if whipIngest {
			if sid, err := uuid.Parse(ingestName); err == nil {
				m.scheduleWhipEndIngest(ingestName, sid.String())
			}
		}
		return nil
	}

	if sess.uploaderCancel != nil {
		sess.uploaderCancel()
	}
	if sess.cmd != nil {
		_ = transcode.StopCmd(sess.cmd)
	}
	_ = m.backend.Stop(ctx, sess.streamID.String(), "publish_done")

	outDir := sess.outputDir
	if outDir == "" {
		outDir = filepath.Join(m.hlsDir, sess.streamID.String())
	}
	playlist := filepath.Join(outDir, "master.m3u8")
	if err := hlsrecord.FinalizePlaylist(playlist); err != nil {
		m.log.Warn("finalize hls playlist", zap.String("path", playlist), zap.Error(err))
	}

	streamID := sess.streamID.String()
	if whipIngest {
		// WHIP qisqa uzilsa WHEP tomoshabinlari saqlansin; qayta ulansa grace bekor.
		m.scheduleWhipEndIngest(ingestName, streamID)
		m.log.Info("whip publisher paused, grace before end ingest",
			zap.String("stream_id", streamID),
			zap.Duration("grace", whipEndIngestGrace),
		)
		return nil
	}

	now := time.Now()
	stopped := domain.StatusStopped
	if err := m.media.Upsert(ctx, &domain.StreamMedia{
		StreamID: sess.streamID, Status: stopped, StoppedAt: &now, HLSPath: &outDir,
	}); err != nil {
		m.log.Warn("media upsert on stop failed", zap.Error(err))
	}

	if err := m.stream.EndIngest(ctx, streamID); err != nil {
		m.log.Warn("end ingest failed", zap.Error(err))
	}

	m.log.Info("ingest stopped", zap.String("stream_id", streamID))
	return nil
}

func (m *Manager) resolveStream(ctx context.Context, ingestName string) (streamID, latencyMode string, err error) {
	if sid, parseErr := uuid.Parse(ingestName); parseErr == nil {
		_, lm, st, getErr := m.stream.GetStream(ctx, sid.String())
		if getErr != nil {
			return "", "", getErr
		}
		switch st {
		case "live":
			return sid.String(), lm, nil
		case "scheduled":
			startedID, startErr := m.stream.StartIngestStream(ctx, sid.String())
			if startErr != nil {
				return "", "", startErr
			}
			if startedID != sid.String() {
				return "", "", fmt.Errorf("scheduled stream is not the active ingest target")
			}
			return sid.String(), lm, nil
		default:
			return "", "", fmt.Errorf("stream not publishable")
		}
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
	if _, lm, _, err := m.stream.GetStream(ctx, streamID); err == nil {
		latencyMode = lm
	}
	return streamID, latencyMode, nil
}

func (m *Manager) buildInputURL(ingestName, source, latencyMode string) (string, string) {
	if source == "whip" {
		if latencyMode == "" || latencyMode == "ultra-low" {
			latencyMode = "ultra-low"
		}
		return m.streamInputURL(m.rtspWorkerBase, ingestName), latencyMode
	}
	if latencyMode == "" {
		latencyMode = "standard"
	}
	return m.streamInputURL(m.rtmpWorkerBase, ingestName), latencyMode
}

func (m *Manager) streamInputURL(base, ingestName string) string {
	return fmt.Sprintf("%s/%s", strings.TrimRight(base, "/"), ingestName)
}
