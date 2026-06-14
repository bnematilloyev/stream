package transcode

import (
	"context"
	"os/exec"

	pkgtranscode "github.com/sahiy/sahiy-stream/pkg/transcode"
)

// LocalBackend runs FFmpeg in-process (default dev mode).
type LocalBackend struct {
	runner *pkgtranscode.Runner
}

func NewLocalBackend(ffmpegPath, encoder, quality string) *LocalBackend {
	return &LocalBackend{
		runner: pkgtranscode.NewRunnerWithQuality(
			ffmpegPath,
			pkgtranscode.VideoEncoder{Codec: encoder},
			quality,
		),
	}
}

func (b *LocalBackend) Start(_ context.Context, req StartRequest) (*RunningJob, error) {
	cmd, err := b.runner.StartForLatency(req.InputURL, req.OutputDir, req.LatencyMode)
	if err != nil {
		return nil, err
	}
	return &RunningJob{
		JobID: req.StreamID, StreamID: req.StreamID, Cmd: cmd,
		FFmpegPID: pkgtranscode.PID(cmd),
	}, nil
}

func (b *LocalBackend) Stop(_ context.Context, _ string, _ string) error {
	return nil
}

// StopCmd stops a specific FFmpeg process (used by manager with local backend).
func StopCmd(cmd *exec.Cmd) error { return pkgtranscode.Stop(cmd) }

func PID(cmd *exec.Cmd) int { return pkgtranscode.PID(cmd) }
