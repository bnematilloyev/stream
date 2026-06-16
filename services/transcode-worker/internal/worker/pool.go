package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"

	pkgnats "github.com/sahiy/sahiy-stream/pkg/nats"
	"github.com/sahiy/sahiy-stream/pkg/storage"
	"github.com/sahiy/sahiy-stream/pkg/transcode"
	"go.uber.org/zap"
)

type Pool struct {
	mu       sync.Mutex
	jobs     map[string]*activeJob
	maxJobs  int
	workerID string
	runner   *jobRunner
	bus      *pkgnats.TranscodeBus
	log      *zap.Logger
}

type activeJob struct {
	job            transcode.StartJob
	cmd            *exec.Cmd
	start          time.Time
	uploaderCancel context.CancelFunc
}

func NewPool(workerID, ffmpeg, encoder, quality, hlsOutputDir string, maxJobs int, bus *pkgnats.TranscodeBus, store storage.ObjectStorage, log *zap.Logger) *Pool {
	if maxJobs <= 0 {
		maxJobs = 4
	}
	if hlsOutputDir == "" {
		hlsOutputDir = "/tmp/hls"
	}
	return &Pool{
		jobs: make(map[string]*activeJob), maxJobs: maxJobs,
		workerID: workerID, bus: bus, log: log,
		runner: &jobRunner{
			ffmpeg: ffmpeg, quality: quality, hlsOutputDir: hlsOutputDir,
			workerID: workerID, bus: bus, storage: store, log: log,
		},
	}
}

func (p *Pool) Run(ctx context.Context) error {
	cmdSub, err := p.bus.SubscribeCommands(p.handleCommand)
	if err != nil {
		return err
	}

	go p.heartbeatLoop(ctx)

	<-ctx.Done()
	_ = cmdSub.Unsubscribe()
	p.stopAll()
	return nil
}

func (p *Pool) handleCommand(subject string, data []byte) error {
	switch subject {
	case transcode.CmdStartSubject:
		var job transcode.StartJob
		if err := json.Unmarshal(data, &job); err != nil {
			return err
		}
		return p.startJob(context.Background(), job)
	case transcode.CmdStopSubject:
		var job transcode.StopJob
		if err := json.Unmarshal(data, &job); err != nil {
			return err
		}
		p.stopJob(job.StreamID)
		return nil
	default:
		return fmt.Errorf("unknown subject: %s", subject)
	}
}

func (p *Pool) startJob(ctx context.Context, job transcode.StartJob) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.jobs) >= p.maxJobs {
		_ = p.runner.publish(ctx, transcode.JobEvent{
			JobID: job.JobID, StreamID: job.StreamID, Type: transcode.EventRejected,
			WorkerID: p.workerID, Error: "worker at capacity", At: time.Now().UTC(),
		})
		return fmt.Errorf("worker at capacity")
	}
	if _, exists := p.jobs[job.StreamID]; exists {
		return nil
	}

	active, err := p.runner.start(ctx, job)
	if err != nil {
		return err
	}
	p.jobs[job.StreamID] = active
	return nil
}

func (p *Pool) stopJob(streamID string) {
	p.mu.Lock()
	active, ok := p.jobs[streamID]
	if ok {
		delete(p.jobs, streamID)
	}
	p.mu.Unlock()

	if !ok {
		return
	}
	p.runner.stop(active, streamID)
}

func (p *Pool) stopAll() {
	p.mu.Lock()
	ids := make([]string, 0, len(p.jobs))
	for id := range p.jobs {
		ids = append(ids, id)
	}
	p.mu.Unlock()
	for _, id := range ids {
		p.stopJob(id)
	}
}

func (p *Pool) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.mu.Lock()
			for streamID, active := range p.jobs {
				_ = p.runner.publish(ctx, transcode.JobEvent{
					JobID: active.job.JobID, StreamID: streamID, Type: transcode.EventHeartbeat,
					WorkerID: p.workerID, FFmpegPID: transcode.PID(active.cmd), At: time.Now().UTC(),
				})
			}
			p.mu.Unlock()
		}
	}
}
