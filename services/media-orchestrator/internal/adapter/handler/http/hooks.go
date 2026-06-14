package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sahiy/sahiy-stream/pkg/health"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
	"github.com/sahiy/sahiy-stream/pkg/security/internalauth"
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/pipeline"
	"go.uber.org/zap"
)

type HookHandler struct {
	pipeline *pipeline.Manager
	log      *zap.Logger
	auth     internalauth.Config
}

func NewHookHandler(p *pipeline.Manager, log *zap.Logger, auth internalauth.Config) *HookHandler {
	return &HookHandler{pipeline: p, log: log, auth: auth}
}

func (h *HookHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/health", h.health)

	r.Group(func(protected chi.Router) {
		protected.Use(internalauth.Middleware(h.auth))
		protected.Post("/hooks/publish", h.onPublish)
		protected.Post("/hooks/publish_done", h.onPublishDone)
	})

	return r
}

func (h *HookHandler) health(w http.ResponseWriter, _ *http.Request) {
	health.Liveness(w, "media-orchestrator")
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
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
