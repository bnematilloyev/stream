package pipeline

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	rtmpProbeInterval = 2 * time.Second
	rtmpProbeTimeout  = 45 * time.Second
)

func probeRTMP(ctx context.Context, inputURL string) error {
	probe, err := exec.LookPath("ffprobe")
	if err != nil {
		return fmt.Errorf("ffprobe not found: %w", err)
	}
	probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(
		probeCtx,
		probe,
		"-v", "error",
		"-rw_timeout", "5000000",
		"-show_entries", "stream=codec_type",
		"-of", "csv=p=0",
		inputURL,
	)
	return cmd.Run()
}

func waitForRTMPPublisher(ctx context.Context, baseURL, ingestName string, log *zap.Logger) (string, error) {
	candidates := rtmpInputCandidates(baseURL, ingestName)
	deadline := time.Now().Add(rtmpProbeTimeout)
	attempt := 0
	for time.Now().Before(deadline) {
		attempt++
		for _, url := range candidates {
			if err := probeRTMP(ctx, url); err == nil {
				if log != nil {
					log.Info("rtmp publisher ready",
						zap.String("url", url),
						zap.Int("attempt", attempt),
					)
				}
				return url, nil
			}
		}
		if log != nil && (attempt <= 3 || attempt%5 == 0) {
			log.Debug("waiting for rtmp publisher",
				zap.String("ingest", ingestName),
				zap.Int("attempt", attempt),
			)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(rtmpProbeInterval):
		}
	}
	return "", fmt.Errorf("rtmp publisher not ready after %s: %s", rtmpProbeTimeout, ingestName)
}

func rtmpInputCandidates(baseURL, ingestName string) []string {
	baseURL = strings.TrimRight(baseURL, "/")
	primary := fmt.Sprintf("%s/%s", baseURL, ingestName)
	seen := map[string]struct{}{primary: {}}
	out := []string{primary}

	if strings.Contains(baseURL, "127.0.0.1") {
		for _, host := range []string{"172.17.0.1", "172.24.0.1"} {
			alt := strings.Replace(baseURL, "127.0.0.1", host, 1) + "/" + ingestName
			if _, ok := seen[alt]; !ok {
				seen[alt] = struct{}{}
				out = append(out, alt)
			}
		}
	}
	return out
}
