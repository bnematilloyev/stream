package worker

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sahiy/sahiy-stream/pkg/hlssync"
	pkgnats "github.com/sahiy/sahiy-stream/pkg/nats"
	"github.com/sahiy/sahiy-stream/pkg/storage"
	"github.com/sahiy/sahiy-stream/pkg/transcode"
	"go.uber.org/zap"
)

type jobRunner struct {
	ffmpeg   string
	quality  string
	workerID string
	bus      *pkgnats.TranscodeBus
	storage  storage.ObjectStorage
	log      *zap.Logger
}

func (r *jobRunner) start(ctx context.Context, job transcode.StartJob) (*activeJob, error) {
	runner := transcode.NewRunnerWithQuality(r.ffmpeg, transcode.VideoEncoder{Codec: job.Encoder}, job.Quality)
	cmd, err := runner.StartForLatency(job.InputURL, job.OutputDir, job.LatencyMode)
	if err != nil {
		_ = r.publish(ctx, transcode.JobEvent{
			JobID: job.JobID, StreamID: job.StreamID, Type: transcode.EventFailed,
			WorkerID: r.workerID, Error: err.Error(), At: time.Now().UTC(),
		})
		return nil, err
	}

	active := &activeJob{job: job, cmd: cmd, start: time.Now().UTC()}
	if r.storage != nil && job.Storage == storage.BackendS3 {
		sid, parseErr := uuid.Parse(job.StreamID)
		if parseErr == nil {
			uploaderCtx, cancel := context.WithCancel(context.Background())
			active.uploaderCancel = cancel
			go hlssync.NewSegmentUploader(r.storage, job.OutputDir, sid, 2*time.Second, r.log).Run(uploaderCtx)
		}
	}

	r.log.Info("transcode started",
		zap.String("stream_id", job.StreamID),
		zap.String("job_id", job.JobID),
		zap.Int("pid", transcode.PID(cmd)),
		zap.String("storage", job.Storage),
		zap.String("quality", transcode.NormalizeQuality(job.Quality)),
	)
	if err := r.publish(ctx, transcode.JobEvent{
		JobID: job.JobID, StreamID: job.StreamID, Type: transcode.EventStarted,
		WorkerID: r.workerID, FFmpegPID: transcode.PID(cmd), At: time.Now().UTC(),
	}); err != nil {
		_ = transcode.Stop(cmd)
		if active.uploaderCancel != nil {
			active.uploaderCancel()
		}
		return nil, err
	}
	return active, nil
}

func (r *jobRunner) stop(active *activeJob, streamID string) {
	if active.uploaderCancel != nil {
		active.uploaderCancel()
	}
	_ = transcode.Stop(active.cmd)
	_ = r.publish(context.Background(), transcode.JobEvent{
		JobID: active.job.JobID, StreamID: streamID, Type: transcode.EventStopped,
		WorkerID: r.workerID, At: time.Now().UTC(),
	})
	r.log.Info("transcode stopped", zap.String("stream_id", streamID))
}

func (r *jobRunner) publish(ctx context.Context, evt transcode.JobEvent) error {
	return r.bus.PublishEvent(ctx, evt)
}
