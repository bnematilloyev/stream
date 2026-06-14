package transcode_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/sahiy/sahiy-stream/pkg/transcode"
)

func TestStartJobJSON(t *testing.T) {
	job := transcode.StartJob{
		JobID: "job-1", StreamID: "stream-1", InputURL: "rtmp://localhost/live/key",
		OutputDir: "/data/hls/stream-1", LatencyMode: "ultra-low", Encoder: "libx264",
		IssuedAt: time.Unix(1718366400, 0).UTC(),
	}
	data, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	var decoded transcode.StartJob
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.StreamID != job.StreamID || decoded.Encoder != job.Encoder {
		t.Fatalf("round-trip mismatch: %+v", decoded)
	}
}

func TestEvtSubject(t *testing.T) {
	got := transcode.EvtSubject("abc-123")
	want := "transcode.evt.abc-123"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestProfileForLatency(t *testing.T) {
	if !transcode.ProfileForLatency("ultra-low").UseLLHLS {
		t.Fatal("ultra-low should use LL-HLS")
	}
	if transcode.ProfileForLatency("standard").UseLLHLS {
		t.Fatal("standard should not use LL-HLS")
	}
}
