package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
	streamv1 "github.com/sahiy/sahiy-stream/proto/gen/stream/v1"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/client"
)

type StreamHandler struct {
	stream   *client.StreamClient
	whipBase string
}

func NewStreamHandler(stream *client.StreamClient, whipBaseURL string) *StreamHandler {
	return &StreamHandler{stream: stream, whipBase: whipBaseURL}
}

func (h *StreamHandler) Create(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var body struct {
		ChannelSlug     string   `json:"channel_slug"`
		Title           string   `json:"title"`
		Description     string   `json:"description"`
		IngestProtocol  string   `json:"ingest_protocol"`
		LatencyMode     string   `json:"latency_mode"`
		Visibility      string   `json:"visibility"`
		CategoryID      string   `json:"category_id"`
		Tags            []string `json:"tags"`
		ScheduledAtUnix int64    `json:"scheduled_at_unix"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, decodeError(err))
		return
	}
	st, err := h.stream.Stream.CreateStream(r.Context(), &streamv1.CreateStreamRequest{
		UserId: u.ID, ChannelSlug: body.ChannelSlug, Title: body.Title, Description: body.Description,
		IngestProtocol: body.IngestProtocol, LatencyMode: body.LatencyMode, Visibility: body.Visibility,
		CategoryId: body.CategoryID, Tags: body.Tags, ScheduledAtUnix: body.ScheduledAtUnix,
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusCreated, streamToJSON(st))
}

func (h *StreamHandler) Get(w http.ResponseWriter, r *http.Request) {
	st, err := h.stream.Stream.GetStream(r.Context(), &streamv1.GetStreamRequest{StreamId: chi.URLParam(r, "id")})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, streamToJSON(st))
}

func (h *StreamHandler) Update(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var body struct {
		Title       *string  `json:"title"`
		Description *string  `json:"description"`
		Visibility  *string  `json:"visibility"`
		CategoryID  *string  `json:"category_id"`
		Tags        []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, decodeError(err))
		return
	}
	st, err := h.stream.Stream.UpdateStream(r.Context(), &streamv1.UpdateStreamRequest{
		UserId: u.ID, StreamId: chi.URLParam(r, "id"),
		Title: body.Title, Description: body.Description, Visibility: body.Visibility,
		CategoryId: body.CategoryID, Tags: body.Tags,
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, streamToJSON(st))
}

func (h *StreamHandler) Delete(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	_, err := h.stream.Stream.DeleteStream(r.Context(), &streamv1.DeleteStreamRequest{
		UserId: u.ID, StreamId: chi.URLParam(r, "id"),
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *StreamHandler) ListLive(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	resp, err := h.stream.Stream.ListLiveStreams(r.Context(), &streamv1.ListLiveStreamsRequest{Page: int32(page), Limit: int32(limit)})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=2, stale-while-revalidate=5")
	httputil.JSON(w, http.StatusOK, streamsListToJSON(resp))
}

func (h *StreamHandler) ListByChannel(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	resp, err := h.stream.Stream.ListChannelStreams(r.Context(), &streamv1.ListChannelStreamsRequest{
		ChannelSlug: chi.URLParam(r, "slug"), Page: int32(page), Limit: int32(limit), Status: r.URL.Query().Get("status"),
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, streamsListToJSON(resp))
}

func (h *StreamHandler) Start(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	st, err := h.stream.Stream.StartStream(r.Context(), &streamv1.StartStreamRequest{UserId: u.ID, StreamId: chi.URLParam(r, "id")})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, streamToJSON(st))
}

func (h *StreamHandler) Playback(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "id")
	st, stErr := h.stream.Stream.GetStream(r.Context(), &streamv1.GetStreamRequest{StreamId: streamID})

	resp, err := h.stream.Stream.GetPlayback(r.Context(), &streamv1.GetPlaybackRequest{StreamId: streamID})
	if err != nil {
		// Ultra-low WHIP: WHEP works via MediaMTX before HLS/transcode is ready.
		if stErr == nil && st != nil && st.Status == "live" && st.LatencyMode == "ultra-low" && h.whipBase != "" {
			w.Header().Set("Cache-Control", "public, max-age=2, stale-while-revalidate=5")
			httputil.JSON(w, http.StatusOK, map[string]any{
				"stream_id":       streamID,
				"whep_url":        h.whipBase + "/" + streamID + "/whep",
				"format":          "webrtc",
				"playback_mode":   "whep",
				"status":          "live",
				"latency_mode":    st.LatencyMode,
				"expires_at_unix": 0,
			})
			return
		}
		httputil.Error(w, grpcError(err))
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=5, stale-while-revalidate=30")
	out := map[string]any{
		"stream_id": resp.StreamId, "url": resp.Url, "format": "ll-hls",
		"status": resp.Status, "expires_at_unix": resp.ExpiresAtUnix,
	}
	if stErr == nil && st != nil {
		out["latency_mode"] = st.LatencyMode
		if st.LatencyMode == "ultra-low" && h.whipBase != "" {
			out["whep_url"] = h.whipBase + "/" + streamID + "/whep"
			out["playback_mode"] = "dual"
		} else {
			out["playback_mode"] = "ll-hls"
		}
	}
	httputil.JSON(w, http.StatusOK, out)
}

func (h *StreamHandler) End(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	st, err := h.stream.Stream.EndStream(r.Context(), &streamv1.EndStreamRequest{UserId: u.ID, StreamId: chi.URLParam(r, "id")})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, streamToJSON(st))
}

func (h *StreamHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, decodeError(err))
		return
	}
	if body.SessionID == "" {
		httputil.Error(w, validationError("session_id is required"))
		return
	}
	resp, err := h.stream.Stream.RecordViewerHeartbeat(r.Context(), &streamv1.RecordViewerHeartbeatRequest{
		StreamId: chi.URLParam(r, "id"), SessionId: body.SessionID,
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, viewerStatsToJSON(resp))
}

func (h *StreamHandler) ViewerStats(w http.ResponseWriter, r *http.Request) {
	resp, err := h.stream.Stream.GetViewerStats(r.Context(), &streamv1.GetViewerStatsRequest{
		StreamId: chi.URLParam(r, "id"),
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, viewerStatsToJSON(resp))
}

func viewerStatsToJSON(resp *streamv1.ViewerStatsResponse) map[string]any {
	return map[string]any{
		"stream_id": resp.StreamId, "concurrent": resp.Concurrent, "unique": resp.Unique,
	}
}

func streamToJSON(st *streamv1.Stream) map[string]any {
	return map[string]any{
		"id": st.Id, "channel_id": st.ChannelId, "channel_slug": st.ChannelSlug, "channel_title": st.ChannelTitle,
		"title": st.Title, "description": st.Description, "thumbnail_url": st.ThumbnailUrl, "status": st.Status,
		"ingest_protocol": st.IngestProtocol, "latency_mode": st.LatencyMode, "visibility": st.Visibility,
		"category_id": st.CategoryId, "tags": st.Tags, "viewer_count": st.ViewerCount, "peak_viewers": st.PeakViewers,
		"scheduled_at_unix": st.ScheduledAtUnix, "started_at_unix": st.StartedAtUnix, "ended_at_unix": st.EndedAtUnix,
		"created_at_unix": st.CreatedAtUnix, "updated_at_unix": st.UpdatedAtUnix,
		"marketplace_seller_id": st.MarketplaceSellerId, "marketplace_shop_id": st.MarketplaceShopId,
	}
}

func streamsListToJSON(resp *streamv1.ListStreamsResponse) map[string]any {
	items := make([]map[string]any, 0, len(resp.Streams))
	for _, st := range resp.Streams {
		items = append(items, streamToJSON(st))
	}
	return map[string]any{"data": items, "pagination": map[string]any{"page": resp.Page, "limit": resp.Limit, "total": resp.Total}}
}
