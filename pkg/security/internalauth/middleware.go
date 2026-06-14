package internalauth

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"

	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
)

const (
	headerSecret = "X-Internal-Secret"
	querySecret  = "internal_secret"
)

// Config controls internal endpoint authentication.
type Config struct {
	Secret         string
	AllowInternal  bool // allow RFC1918/loopback without secret (dev convenience)
	RequireSecret  bool // production: secret mandatory even for internal IPs
}

// Middleware protects internal-only HTTP endpoints (e.g. media ingest hooks).
func Middleware(cfg Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authorized(cfg, r) {
				next.ServeHTTP(w, r)
				return
			}
			httputil.Error(w, apperrors.Forbidden("forbidden"))
		})
	}
}

func authorized(cfg Config, r *http.Request) bool {
	secret := strings.TrimSpace(cfg.Secret)
	if secret != "" && validSecret(r, secret) {
		return true
	}
	if cfg.RequireSecret {
		return false
	}
	if cfg.AllowInternal && isInternalIP(httputil.ClientIP(r)) {
		return true
	}
	return secret == "" && cfg.AllowInternal
}

func validSecret(r *http.Request, expected string) bool {
	if header := r.Header.Get(headerSecret); header != "" {
		return subtle.ConstantTimeCompare([]byte(header), []byte(expected)) == 1
	}
	if query := r.URL.Query().Get(querySecret); query != "" {
		return subtle.ConstantTimeCompare([]byte(query), []byte(expected)) == 1
	}
	return false
}

func isInternalIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate()
}
