package transcode

import (
	"context"
	"os/exec"
)

// Backend starts and stops transcoding jobs.
type Backend interface {
	Start(ctx context.Context, req StartRequest) (*RunningJob, error)
	Stop(ctx context.Context, streamID, reason string) error
}

type StartRequest struct {
	StreamID    string
	IngestName  string
	InputURL    string
	OutputDir   string
	LatencyMode string
	Quality     string
	Encoder     string
	Storage     string
}

type RunningJob struct {
	JobID     string
	StreamID  string
	Cmd       *exec.Cmd
	FFmpegPID int
}
