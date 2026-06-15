package playback_test

import (
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sahiy/sahiy-stream/pkg/playback"
)

func TestSignerSignAndValidate(t *testing.T) {
	signer := playback.NewSigner("playback-signing-secret-min-32-chars!", time.Hour)
	streamID := "11111111-1111-1111-1111-111111111111"
	resource := "master.m3u8"

	signed, expires := signer.Sign("http://localhost:9083/playback/"+streamID, streamID, resource)
	if !strings.Contains(signed, "sig=") || !strings.Contains(signed, "exp=") {
		t.Fatalf("signed url missing params: %s", signed)
	}
	if expires.Before(time.Now()) {
		t.Fatal("expires should be in the future")
	}

	parsed, err := url.Parse(signed)
	if err != nil {
		t.Fatal(err)
	}
	exp, err := strconv.ParseInt(parsed.Query().Get("exp"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	sig := parsed.Query().Get("sig")
	if err := signer.Validate(streamID, resource, exp, sig); err != nil {
		t.Fatalf("validate failed: %v", err)
	}
}

func TestSignerAcceptsMasterSignatureForVariantPlaylist(t *testing.T) {
	signer := playback.NewSigner("playback-signing-secret-min-32-chars!", time.Hour)
	streamID := "4dc38c78-112c-4e17-ba0b-dbe8f4e3c7e9"
	exp := time.Now().Add(time.Hour).Unix()
	signed, _ := signer.Sign("http://localhost:9083/playback/"+streamID, streamID, "master.m3u8")
	parsed, err := url.Parse(signed)
	if err != nil {
		t.Fatal(err)
	}
	sig := parsed.Query().Get("sig")
	if err := signer.Validate(streamID, "480p/playlist.m3u8", exp, sig); err != nil {
		t.Fatalf("master signature should authorize variant playlist: %v", err)
	}
}

func TestSignerRejectsTamperedSignature(t *testing.T) {
	signer := playback.NewSigner("playback-signing-secret-min-32-chars!", time.Hour)
	err := signer.Validate("stream-id", "master.m3u8", time.Now().Add(time.Hour).Unix(), "bad-signature")
	if err == nil {
		t.Fatal("expected validation error")
	}
}
