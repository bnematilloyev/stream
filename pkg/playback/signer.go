package playback

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultTTL = 4 * time.Hour

// Signer creates and validates HMAC-signed playback URLs.
type Signer struct {
	secret []byte
	ttl    time.Duration
}

func NewSigner(secret string, ttl time.Duration) *Signer {
	if ttl <= 0 {
		ttl = defaultTTL
	}
	return &Signer{secret: []byte(secret), ttl: ttl}
}

// Sign builds a signed URL for an HLS resource.
func (s *Signer) Sign(baseURL, streamID, resource string) (string, time.Time) {
	expires := time.Now().Add(s.ttl)
	sig := s.compute(streamID, resource, expires.Unix())
	u, _ := url.Parse(strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(resource, "/"))
	q := u.Query()
	q.Set("exp", strconv.FormatInt(expires.Unix(), 10))
	q.Set("sig", sig)
	u.RawQuery = q.Encode()
	return u.String(), expires
}

// Validate checks signature and expiry for a playback request.
func (s *Signer) Validate(streamID, resource string, exp int64, sig string) error {
	if exp <= time.Now().Unix() {
		return fmt.Errorf("playback token expired")
	}
	for _, candidate := range []string{resource, "master.m3u8", "*"} {
		expected := s.compute(streamID, candidate, exp)
		if hmac.Equal([]byte(expected), []byte(sig)) {
			return nil
		}
	}
	return fmt.Errorf("invalid playback signature")
}

func (s *Signer) compute(streamID, resource string, exp int64) string {
	payload := fmt.Sprintf("%s:%s:%d", streamID, normalizeResource(resource), exp)
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func normalizeResource(resource string) string {
	return strings.TrimPrefix(strings.TrimSpace(resource), "/")
}

// BuildPlaybackPath returns the unsigned playback path for a stream manifest.
func BuildPlaybackPath(streamID string) string {
	return fmt.Sprintf("%s/master.m3u8", streamID)
}
