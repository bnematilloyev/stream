package httputil

import (
	"net/http"

	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
)

// MaxBody limits request body size to protect against oversized payloads.
func MaxBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body == nil || r.Method == http.MethodGet || r.Method == http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// BodyTooLargeError returns validation error when MaxBytesReader rejects input.
func BodyTooLargeError() *apperrors.AppError {
	return apperrors.Validation("request body too large", nil)
}
