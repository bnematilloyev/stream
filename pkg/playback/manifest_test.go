package playback_test

import (
	"strings"
	"testing"
	"time"

	"github.com/sahiy/sahiy-stream/pkg/playback"
)

func TestRewriteManifestMaster(t *testing.T) {
	signer := playback.NewSigner("playback-signing-secret-min-32-chars!", time.Hour)
	streamID := "4dc38c78-112c-4e17-ba0b-dbe8f4e3c7e9"
	exp := time.Now().Add(time.Hour).Unix()
	queryFor := func(resource string) string {
		return signer.QueryForResource(streamID, resource, exp)
	}

	body := []byte("#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1000000\n480p/playlist.m3u8\n")
	out := string(playback.RewriteManifest(body, "master.m3u8", queryFor))
	if !strings.Contains(out, "480p/playlist.m3u8?exp=") {
		t.Fatalf("expected signed variant playlist: %s", out)
	}
	if !strings.Contains(out, "&sig=") {
		t.Fatalf("expected signature in manifest: %s", out)
	}
}

func TestRewriteManifestVariant(t *testing.T) {
	signer := playback.NewSigner("playback-signing-secret-min-32-chars!", time.Hour)
	streamID := "4dc38c78-112c-4e17-ba0b-dbe8f4e3c7e9"
	exp := time.Now().Add(time.Hour).Unix()
	queryFor := func(resource string) string {
		return signer.QueryForResource(streamID, resource, exp)
	}

	body := []byte("#EXTM3U\n#EXT-X-MAP:URI=\"init.mp4\"\n#EXTINF:2.0,\nseg_00001.m4s\n")
	out := string(playback.RewriteManifest(body, "480p/playlist.m3u8", queryFor))
	if !strings.Contains(out, `URI="init.mp4?exp=`) {
		t.Fatalf("expected signed init segment map: %s", out)
	}
	if !strings.Contains(out, "seg_00001.m4s?exp=") {
		t.Fatalf("expected signed media segment: %s", out)
	}
}
