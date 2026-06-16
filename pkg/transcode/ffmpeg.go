package transcode

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Runner struct {
	bin     string
	encoder VideoEncoder
	quality string
}

func NewRunner(bin string, encoder VideoEncoder) *Runner {
	return NewRunnerWithQuality(bin, encoder, QualityProduction)
}

func NewRunnerWithQuality(bin string, encoder VideoEncoder, quality string) *Runner {
	if encoder.Codec == "" {
		encoder.Codec = "libx264"
	}
	return &Runner{bin: bin, encoder: encoder, quality: NormalizeQuality(quality)}
}

// StartForLatency encodes using the runner's quality profile and ladder.
func (r *Runner) StartForLatency(inputURL, outputDir, latencyMode string) (*exec.Cmd, error) {
	profile, ladder := ResolvePipeline(r.quality, latencyMode)
	return r.StartABR(inputURL, outputDir, profile, ladder)
}

// StartABR launches multi-bitrate LL-HLS (or standard HLS) transcoding.
func (r *Runner) StartABR(inputURL, outputDir string, profile Profile, ladder []Tier) (*exec.Cmd, error) {
	if len(ladder) == 0 {
		ladder = DefaultLadder
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}
	for _, t := range ladder {
		if err := os.MkdirAll(filepath.Join(outputDir, t.Name), 0o755); err != nil {
			return nil, err
		}
	}

	n := len(ladder)
	filter := buildScaleFilter(n, ladder, profile.HighQuality)
	args := []string{
		"-hide_banner", "-loglevel", "warning",
		"-fflags", "+genpts+discardcorrupt",
	}

	if strings.HasPrefix(inputURL, "rtsp://") {
		args = append(args,
			"-use_wallclock_as_timestamps", "1",
			"-rtsp_transport", "tcp", "-timeout", "5000000",
			"-probesize", "32", "-analyzeduration", "0",
		)
	}
	if strings.HasPrefix(inputURL, "rtmp://") {
		args = append(args,
			"-rw_timeout", "45000000",
			"-probesize", "10000000",
			"-analyzeduration", "10000000",
		)
	}

	args = append(args, "-i", inputURL, "-filter_complex", filter)

	videoBase := r.encoder.BaseArgs(profile)
	for i := range ladder {
		args = append(args, "-map", fmt.Sprintf("[v%dout]", i+1))
	}
	args = append(args, "-map", "0:a?")
	args = append(args, videoBase...)
	for i, t := range ladder {
		args = append(args,
			fmt.Sprintf("-b:v:%d", i), t.Bitrate,
			fmt.Sprintf("-maxrate:v:%d", i), t.Maxrate,
			fmt.Sprintf("-bufsize:v:%d", i), t.Bufsize,
		)
	}
	// One AAC encode shared by all variants — avoids timestamp drift per ladder rung.
	args = append(args,
		"-af", "aresample=async=1:first_pts=0",
		"-c:a", "aac",
		"-b:a", ladder[0].AudioBR,
		"-ar", strconv.Itoa(profile.AudioRate),
	)

	args = append(args, "-f", "hls")
	args = append(args, "-hls_time", fmt.Sprintf("%.2f", profile.SegmentSec))
	args = append(args, "-hls_list_size", "12")
	args = append(args, "-hls_delete_threshold", "6")
	args = append(args, "-max_interleave_delta", "0")
	args = append(args, "-muxdelay", "0", "-muxpreload", "0")

	hlsFlags := []string{
		"delete_segments", "append_list", "program_date_time",
		"independent_segments", "temp_file",
	}
	if profile.UseLLHLS {
		hlsFlags = append(hlsFlags, "lhls")
	}
	args = append(args, "-hls_flags", strings.Join(hlsFlags, "+"))

	if profile.UseFMP4 {
		args = append(args, "-hls_segment_type", "fmp4", "-hls_fmp4_init_filename", "init.mp4")
		if profile.PartSec > 0 {
			args = append(args, "-hls_part_duration", fmt.Sprintf("%.4f", profile.PartSec))
		}
		segPattern := filepath.Join(outputDir, "%v/seg_%05d.m4s")
		args = append(args, "-hls_segment_filename", segPattern)
	} else {
		segPattern := filepath.Join(outputDir, "%v/seg_%05d.ts")
		args = append(args, "-hls_segment_filename", segPattern)
	}

	args = append(args, "-master_pl_name", "master.m3u8")
	audioIdx := n
	varStream := make([]string, n)
	for i := range ladder {
		varStream[i] = fmt.Sprintf("v:%d,a:%d", i, audioIdx)
	}
	args = append(args, "-var_stream_map", strings.Join(varStream, " "))
	args = append(args, filepath.Join(outputDir, "%v/playlist.m3u8"))

	cmd := exec.Command(r.bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w", err)
	}
	return cmd, nil
}

func buildScaleFilter(n int, ladder []Tier, highQuality bool) string {
	scaleFlags := ":force_divisible_by=2"
	if highQuality {
		scaleFlags += ":flags=lanczos"
	}
	if n == 1 {
		t := ladder[0]
		return fmt.Sprintf("[0:v]scale=w=%d:h=%d:force_original_aspect_ratio=decrease%s[v1out]",
			t.Width, t.Height, scaleFlags)
	}
	var b strings.Builder
	b.WriteString("[0:v]split=")
	b.WriteString(strconv.Itoa(n))
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "[v%d]", i)
	}
	b.WriteString(";")
	for i, t := range ladder {
		fmt.Fprintf(&b, "[v%d]scale=w=%d:h=%d:force_original_aspect_ratio=decrease%s[v%dout];",
			i+1, t.Width, t.Height, scaleFlags, i+1)
	}
	return strings.TrimSuffix(b.String(), ";")
}

func PID(cmd *exec.Cmd) int {
	if cmd == nil || cmd.Process == nil {
		return 0
	}
	return cmd.Process.Pid
}

func Stop(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	_ = cmd.Process.Signal(os.Interrupt)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
		return nil
	default:
		return cmd.Process.Kill()
	}
}
