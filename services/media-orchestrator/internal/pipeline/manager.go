package pipeline

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/domain"
)

type StreamClient interface {
	ValidateStreamKey(ctx context.Context, key string) (valid bool, channelID, streamID string, err error)
	StartIngest(ctx context.Context, channelID string) (streamID string, err error)
	EndIngest(ctx context.Context, streamID string) error
	GetStream(ctx context.Context, streamID string) (latencyMode, status string, err error)
}

type Manager struct {
	mu       sync.Mutex
	active   map[string]*session
	ffmpeg   *FFmpegRunner
	media    domain.StreamMediaRepository
	stream   StreamClient
	rtmpBase string
	rtspBase string
	hlsDir   string
	hlsBase  string
	log      *zap.Logger
}

type session struct {
	streamID   uuid.UUID
	ingestName string
	cmd        *exec.Cmd
	startedAt  time.Time
}

func NewManager(
	ffmpeg *FFmpegRunner,
	media domain.StreamMediaRepository,
	stream StreamClient,
	rtmpBase, rtspBase, hlsDir, hlsBase string,
	log *zap.Logger,
) *Manager {
	return &Manager{
		active:   make(map[string]*session),
		ffmpeg:   ffmpeg,
		media:    media,
		stream:   stream,
		rtmpBase: rtmpBase,
		rtspBase: rtspBase,
		hlsDir:   hlsDir,
		hlsBase:  hlsBase,
		log:      log,
	}
}

func (m *Manager) OnPublish(ctx context.Context, ingestName, source string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.active[ingestName]; ok {
		return nil
	}

	var streamID string
	var latencyMode string

	if sid, err := uuid.Parse(ingestName); err == nil {
		lm, st, err := m.stream.GetStream(ctx, sid.String())
		if err != nil {
			return err
		}
		if st != "live" && st != "scheduled" {
			return fmt.Errorf("stream not publishable")
		}
		streamID = sid.String()
		latencyMode = lm
	} else {
		valid, channelID, existingStreamID, err := m.stream.ValidateStreamKey(ctx, ingestName)
		if err != nil {
			return err
		}
		if !valid {
			return fmt.Errorf("invalid stream key")
		}
		streamID = existingStreamID
		if streamID == "" {
			streamID, err = m.stream.StartIngest(ctx, channelID)
			if err != nil {
				return err
			}
		}
		if lm, _, err := m.stream.GetStream(ctx, streamID); err == nil {
			latencyMode = lm
		}
	}

	sid, err := uuid.Parse(streamID)
	if err != nil {
		return fmt.Errorf("invalid stream id: %w", err)
	}

	inputURL := fmt.Sprintf("%s/%s", m.rtmpBase, ingestName)
	if source == "whip" {
		path := ingestName
		if _, err := uuid.Parse(ingestName); err == nil {
			path = ingestName
		}
		inputURL = fmt.Sprintf("%s/%s", m.rtspBase, path)
		time.Sleep(1200 * time.Millisecond)
		if latencyMode == "" || latencyMode == "ultra-low" {
			latencyMode = "ultra-low"
		}
	}

	if latencyMode == "" {
		latencyMode = "standard"
	}

	outDir := filepath.Join(m.hlsDir, sid.String())
	cmd, err := m.ffmpeg.StartABR(inputURL, outDir, latencyMode)
	if err != nil {
		return err
	}

	now := time.Now()
	hlsPath := outDir
	playback := fmt.Sprintf("%s/%s/master.m3u8", m.hlsBase, sid.String())
	pid := PID(cmd)
	ingest := ingestName
	status := domain.StatusIngesting

	if err := m.media.Upsert(ctx, &domain.StreamMedia{
		StreamID: sid, Status: status,
		HLSPath: &hlsPath, PlaybackURL: &playback, IngestName: &ingest,
		FFmpegPID: &pid, StartedAt: &now,
	}); err != nil {
		_ = Stop(cmd)
		return err
	}

	m.active[ingestName] = &session{streamID: sid, ingestName: ingestName, cmd: cmd, startedAt: now}
	m.log.Info("ingest started",
		zap.String("stream_id", sid.String()),
		zap.String("ingest", ingestName),
		zap.String("source", source),
		zap.String("latency_mode", latencyMode),
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

	_ = Stop(sess.cmd)

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
