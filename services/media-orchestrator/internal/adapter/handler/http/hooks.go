package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/pipeline"
	"go.uber.org/zap"
)

type HookHandler struct {
	pipeline *pipeline.Manager
	log      *zap.Logger
}

func NewHookHandler(p *pipeline.Manager, log *zap.Logger) *HookHandler {
	return &HookHandler{pipeline: p, log: log}
}

func (h *HookHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/hooks/publish", h.onPublish)
	r.Post("/hooks/publish_done", h.onPublishDone)
	r.Get("/health", h.health)
	return r
}

func (h *HookHandler) health(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HookHandler) onPublish(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if name == "" {
		name = r.URL.Query().Get("name")
	}
	if name == "" {
		http.Error(w, "missing name", http.StatusBadRequest)
		return
	}
	source := r.FormValue("source")
	if source == "" {
		source = r.URL.Query().Get("source")
	}
	if source == "" {
		source = "rtmp"
	}
	if err := h.pipeline.OnPublish(r.Context(), name, source); err != nil {
		h.log.Warn("publish rejected", zap.String("name", name), zap.Error(err))
		http.Error(w, "rejected", http.StatusForbidden)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *HookHandler) onPublishDone(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if name == "" {
		name = r.URL.Query().Get("name")
	}
	if name != "" {
		if err := h.pipeline.OnPublishDone(r.Context(), name); err != nil {
			h.log.Warn("publish_done error", zap.String("name", name), zap.Error(err))
		}
	}
	w.WriteHeader(http.StatusOK)
}
