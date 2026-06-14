package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	pkgnats "github.com/sahiy/sahiy-stream/pkg/nats"
)

type HealthHandler struct{ bus *pkgnats.TranscodeBus }

func NewHealthHandler(bus *pkgnats.TranscodeBus) *HealthHandler {
	return &HealthHandler{bus: bus}
}

func (h *HealthHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"transcode-worker"}`))
	})
	r.Get("/ready", func(w http.ResponseWriter, _ *http.Request) {
		if err := h.bus.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"not_ready"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})
	return r
}
