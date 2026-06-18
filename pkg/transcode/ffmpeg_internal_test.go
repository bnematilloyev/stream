package transcode

import (
	"strings"
	"testing"
)

func TestPassthroughArgsCopyVideoToSingleHLSPlaylist(t *testing.T) {
	runner := NewRunner("ffmpeg", VideoEncoder{})
	args := strings.Join(runner.passthroughArgs("rtmp://127.0.0.1:1935/live/key", "/tmp/hls/stream", "ultra-low"), " ")

	for _, want := range []string{
		"-c:v copy",
		"-c:a aac",
		"-hls_time 3.00",
		"-hls_list_size 0",
		"/tmp/hls/stream/master.m3u8",
	} {
		if !strings.Contains(args, want) {
			t.Fatalf("passthrough args missing %q in: %s", want, args)
		}
	}
}
