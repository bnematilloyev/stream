package transcode

import "time"

const (
	CmdStartSubject = "transcode.cmd.start"
	CmdStopSubject  = "transcode.cmd.stop"
	EvtSubjectPrefix = "transcode.evt."
)

// StartJob is dispatched by media-orchestrator to transcode workers.
type StartJob struct {
	JobID       string    `json:"job_id"`
	StreamID    string    `json:"stream_id"`
	IngestName  string    `json:"ingest_name"`
	InputURL    string    `json:"input_url"`
	OutputDir   string    `json:"output_dir"`
	LatencyMode string    `json:"latency_mode"`
	Quality     string    `json:"quality"`
	Encoder     string    `json:"encoder"`
	Storage     string    `json:"storage"`
	Priority    int       `json:"priority"`
	IssuedAt    time.Time `json:"issued_at"`
}

// StopJob stops a running transcode job.
type StopJob struct {
	JobID    string `json:"job_id"`
	StreamID string `json:"stream_id"`
	Reason   string `json:"reason"`
}

// Job event types.
const (
	EventStarted   = "started"
	EventStopped   = "stopped"
	EventFailed    = "failed"
	EventHeartbeat = "heartbeat"
	EventRejected  = "rejected"
)

// JobEvent is published by workers back to the control plane.
type JobEvent struct {
	JobID     string    `json:"job_id"`
	StreamID  string    `json:"stream_id"`
	Type      string    `json:"type"`
	WorkerID  string    `json:"worker_id"`
	FFmpegPID int       `json:"ffmpeg_pid,omitempty"`
	Error     string    `json:"error,omitempty"`
	At        time.Time `json:"at"`
}

func EvtSubject(streamID string) string {
	return EvtSubjectPrefix + streamID
}

// ProfileForLatency picks LL-HLS or standard packaging (balanced quality).
func ProfileForLatency(latencyMode string) Profile {
	p, _ := ResolvePipeline(QualityBalanced, latencyMode)
	return p
}
