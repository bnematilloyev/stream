package transcode

import (
	"context"
	"strings"
)

// RoutingBackend remuxes RTMP (OBS) and encodes RTSP (WHIP/WebRTC) to HLS.
type RoutingBackend struct {
	Passthrough Backend
	Encode      Backend
}

func NewRoutingBackend(passthrough, encode Backend) *RoutingBackend {
	return &RoutingBackend{Passthrough: passthrough, Encode: encode}
}

func (b *RoutingBackend) Start(ctx context.Context, req StartRequest) (*RunningJob, error) {
	if strings.HasPrefix(req.InputURL, "rtsp://") {
		return b.Encode.Start(ctx, req)
	}
	return b.Passthrough.Start(ctx, req)
}

func (b *RoutingBackend) Stop(ctx context.Context, streamID, reason string) error {
	_ = b.Passthrough.Stop(ctx, streamID, reason)
	_ = b.Encode.Stop(ctx, streamID, reason)
	return nil
}
