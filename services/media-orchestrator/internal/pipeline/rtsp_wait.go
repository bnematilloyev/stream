package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	rtspProbeInterval = 1500 * time.Millisecond
	rtspProbeTimeout  = 45 * time.Second
)

func waitForRTSPPublisher(ctx context.Context, baseURL, ingestName string, log *zap.Logger) (string, error) {
	candidates := rtspInputCandidates(baseURL, ingestName)
	deadline := time.Now().Add(rtspProbeTimeout)
	attempt := 0
	for time.Now().Before(deadline) {
		attempt++
		for _, url := range candidates {
			if err := probeRTMP(ctx, url); err == nil {
				if log != nil {
					log.Info("rtsp publisher ready",
						zap.String("url", url),
						zap.Int("attempt", attempt),
					)
				}
				return url, nil
			}
		}
		if log != nil && (attempt <= 3 || attempt%5 == 0) {
			log.Debug("waiting for rtsp publisher",
				zap.String("ingest", ingestName),
				zap.Int("attempt", attempt),
			)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(rtspProbeInterval):
		}
	}
	return "", fmt.Errorf("rtsp publisher not ready after %s: %s", rtspProbeTimeout, ingestName)
}

func rtspInputCandidates(baseURL, ingestName string) []string {
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
