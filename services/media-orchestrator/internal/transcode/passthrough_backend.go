package transcode

import (
	"context"

	pkgtranscode "github.com/sahiy/sahiy-stream/pkg/transcode"
)

// PassthroughBackend remuxes OBS input to HLS without video transcoding.
type PassthroughBackend struct {
	runner *pkgtranscode.Runner
}

func NewPassthroughBackend(ffmpegPath string) *PassthroughBackend {
	return &PassthroughBackend{
		runner: pkgtranscode.NewRunnerWithQuality(
			ffmpegPath,
			pkgtranscode.VideoEncoder{Codec: "copy"},
			pkgtranscode.QualityProduction,
		),
	}
}

func (b *PassthroughBackend) Start(_ context.Context, req StartRequest) (*RunningJob, error) {
	cmd, err := b.runner.StartPassthroughHLS(req.InputURL, req.OutputDir, req.LatencyMode)
	if err != nil {
		return nil, err
	}
	return &RunningJob{
		JobID: req.StreamID, StreamID: req.StreamID, Cmd: cmd,
		FFmpegPID: pkgtranscode.PID(cmd),
	}, nil
}

func (b *PassthroughBackend) Stop(_ context.Context, _ string, _ string) error {
	return nil
}
