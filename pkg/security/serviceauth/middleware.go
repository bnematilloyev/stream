package serviceauth

import (
	"crypto/subtle"
	"net/http"
	"strings"

	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
)

const headerServiceToken = "X-Service-Token"

// Middleware protects service-to-service HTTP endpoints.
func Middleware(secret string) func(http.Handler) http.Handler {
	secret = strings.TrimSpace(secret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret == "" {
				httputil.Error(w, apperrors.Forbidden("service token not configured"))
				return
			}
			token := strings.TrimSpace(r.Header.Get(headerServiceToken))
			if subtle.ConstantTimeCompare([]byte(token), []byte(secret)) != 1 {
				httputil.Error(w, apperrors.Forbidden("forbidden"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
