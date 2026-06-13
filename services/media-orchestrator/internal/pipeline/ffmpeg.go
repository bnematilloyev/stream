package pipeline

import (
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/transcode"
	"os/exec"
)

// FFmpegRunner wraps the transcode package for pipeline use.
type FFmpegRunner struct {
	inner *transcode.Runner
}

func NewFFmpegRunner(binPath string) *FFmpegRunner {
	return &FFmpegRunner{inner: transcode.NewRunner(binPath)}
}

func (f *FFmpegRunner) StartABR(inputURL, outputDir, latencyMode string) (*exec.Cmd, error) {
	profile := transcode.LLHLSProfile()
	if latencyMode == "standard" {
		profile = transcode.StandardProfile()
	}
	return f.inner.StartABR(inputURL, outputDir, profile, transcode.DefaultLadder)
}

func PID(cmd *exec.Cmd) int { return transcode.PID(cmd) }

func Stop(cmd *exec.Cmd) error { return transcode.Stop(cmd) }
