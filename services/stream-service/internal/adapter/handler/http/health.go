package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sahiy/sahiy-stream/pkg/database"
)

type HealthHandler struct{ db *database.Router }

func NewHealthHandler(db *database.Router) *HealthHandler { return &HealthHandler{db: db} }

func (h *HealthHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "stream-service"})
	})
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		code := http.StatusOK
		status := "ready"
		if err := h.db.Primary().Ping(ctx); err != nil {
			code = http.StatusServiceUnavailable
			status = "not_ready"
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": status})
	})
	return r
}
