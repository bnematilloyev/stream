package http

import (
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/sahiy/sahiy-stream/pkg/playback"
	"github.com/sahiy/sahiy-stream/pkg/storage"
)

// DeliveryHandler serves signed HLS manifests and segments.
type DeliveryHandler struct {
	storage storage.ObjectStorage
	signer  *playback.Signer
}

func NewDeliveryHandler(store storage.ObjectStorage, signer *playback.Signer) *DeliveryHandler {
	return &DeliveryHandler{storage: store, signer: signer}
}

func (h *DeliveryHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{streamID}/*", h.serve)
	return r
}

func (h *DeliveryHandler) serve(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "streamID")
	resource := chi.URLParam(r, "*")
	if streamID == "" || resource == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	expRaw := r.URL.Query().Get("exp")
	sig := r.URL.Query().Get("sig")
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || sig == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := h.signer.Validate(streamID, resource, exp, sig); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	key := h.storage.ResolveKey(streamID, resource)
	reader, err := h.storage.Open(r.Context(), key)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer func() { _ = reader.Close() }()

	if storage.IsPlaylist(resource) {
		body, err := io.ReadAll(reader)
		if err != nil || len(body) == 0 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		queryFor := func(resource string) string {
			return h.signer.QueryForResource(streamID, resource, exp)
		}
		body = playback.RewriteManifest(body, resource, queryFor)

		w.Header().Set("Content-Type", storage.ContentType(resource))
		w.Header().Set("Cache-Control", "public, max-age=1, stale-while-revalidate=2")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
		return
	}

	w.Header().Set("Content-Type", storage.ContentType(resource))
	switch {
	case storage.IsSegment(resource):
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	default:
		w.Header().Set("Cache-Control", "public, max-age=60")
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}
