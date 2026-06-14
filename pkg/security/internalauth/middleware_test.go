package internalauth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sahiy/sahiy-stream/pkg/security/internalauth"
)

func TestMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("valid header secret", func(t *testing.T) {
		handler := internalauth.Middleware(internalauth.Config{
			Secret:        "test-secret-minimum-32-characters-long",
			RequireSecret: true,
		})(next)

		req := httptest.NewRequest(http.MethodPost, "/hooks/publish", nil)
		req.Header.Set("X-Internal-Secret", "test-secret-minimum-32-characters-long")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("production rejects missing secret", func(t *testing.T) {
		handler := internalauth.Middleware(internalauth.Config{
			Secret:        "test-secret-minimum-32-characters-long",
			RequireSecret: true,
		})(next)

		req := httptest.NewRequest(http.MethodPost, "/hooks/publish", nil)
		req.RemoteAddr = "8.8.8.8:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("dev allows internal ip without secret", func(t *testing.T) {
		handler := internalauth.Middleware(internalauth.Config{
			Secret:        "test-secret-minimum-32-characters-long",
			AllowInternal: true,
		})(next)

		req := httptest.NewRequest(http.MethodPost, "/hooks/publish", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}
