package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
	authv1 "github.com/sahiy/sahiy-stream/proto/gen/auth/v1"
	streamv1 "github.com/sahiy/sahiy-stream/proto/gen/stream/v1"
	userv1 "github.com/sahiy/sahiy-stream/proto/gen/user/v1"
)

func (h *BroadcastHandler) sellerIDFromRequest(r *http.Request, bodySellerID int64) (int64, error) {
	if bodySellerID > 0 {
		return bodySellerID, nil
	}
	if id, err := strconv.ParseInt(chi.URLParam(r, "seller_id"), 10, 64); err == nil && id > 0 {
		return id, nil
	}
	return 0, apperrors.Validation("marketplace_seller_id is required", nil)
}

func (h *BroadcastHandler) sellerAuth(ctx context.Context, sellerID int64) (*authv1.AuthResponse, *userv1.Channel, error) {
	ch, err := h.user.Channel.GetChannelByMarketplaceSeller(ctx, &userv1.GetChannelByMarketplaceSellerRequest{
		MarketplaceSellerId: sellerID,
	})
	if err != nil {
		return nil, nil, err
	}

	email := fmt.Sprintf("seller-%d@broadcast.internal.sahiy", sellerID)
	authResp, err := h.auth.Login(ctx, &authv1.LoginRequest{
		Email:    email,
		Password: provisionPassword(h.secret, sellerID),
	})
	if err != nil {
		return nil, nil, err
	}
	return authResp, ch, nil
}

func (h *BroadcastHandler) ingestForSeller(ctx context.Context, userID, channelSlug string) (map[string]any, error) {
	ingest, err := h.user.Channel.GetIngestKey(ctx, &userv1.GetIngestKeyRequest{
		UserId: userID, ChannelSlug: channelSlug,
	})
	if err != nil {
		return nil, err
	}
	if ingest.GetStreamKey() == "" {
		ingest, err = h.user.Channel.RotateIngestKey(ctx, &userv1.RotateIngestKeyRequest{
			UserId: userID, ChannelSlug: channelSlug,
		})
		if err != nil {
			return nil, err
		}
	}
	return map[string]any{
		"stream_key": ingest.GetStreamKey(),
		"rtmp_url":   ingest.GetRtmpUrl(),
		"srt_url":    ingest.GetSrtUrl(),
		"key_prefix": ingest.GetKeyPrefix(),
	}, nil
}

func (h *BroadcastHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MarketplaceSellerID int64 `json:"marketplace_seller_id"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.Error(w, decodeError(err))
			return
		}
	}

	sellerID, err := h.sellerIDFromRequest(r, req.MarketplaceSellerID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	authResp, ch, err := h.sellerAuth(r.Context(), sellerID)
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}

	ingest, err := h.ingestForSeller(r.Context(), authResp.GetUser().GetId(), ch.GetSlug())
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]any{
		"access_token":          authResp.GetAccessToken(),
		"refresh_token":         authResp.GetRefreshToken(),
		"expires_at_unix":       authResp.GetExpiresAtUnix(),
		"channel_slug":          ch.GetSlug(),
		"marketplace_seller_id": sellerID,
		"marketplace_shop_id":   ch.GetMarketplaceShopId(),
		"ingest":                ingest,
	})
}

func (h *BroadcastHandler) CreateSellerStream(w http.ResponseWriter, r *http.Request) {
	sellerID, err := h.sellerIDFromRequest(r, 0)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	authResp, ch, err := h.sellerAuth(r.Context(), sellerID)
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}

	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, decodeError(err))
		return
	}
	if body.Title == "" {
		httputil.Error(w, apperrors.Validation("title is required", nil))
		return
	}

	st, err := h.stream.Stream.CreateStream(r.Context(), &streamv1.CreateStreamRequest{
		UserId: authResp.GetUser().GetId(), ChannelSlug: ch.GetSlug(),
		Title: body.Title, Description: body.Description, Visibility: "public",
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusCreated, streamToJSON(st))
}

func (h *BroadcastHandler) ListSellerStreams(w http.ResponseWriter, r *http.Request) {
	sellerID, err := h.sellerIDFromRequest(r, 0)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	_, ch, err := h.sellerAuth(r.Context(), sellerID)
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	resp, err := h.stream.Stream.ListChannelStreams(r.Context(), &streamv1.ListChannelStreamsRequest{
		ChannelSlug: ch.GetSlug(), Page: int32(page), Limit: int32(limit), Status: r.URL.Query().Get("status"),
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, streamsListToJSON(resp))
}

func (h *BroadcastHandler) StartSellerStream(w http.ResponseWriter, r *http.Request) {
	sellerID, err := h.sellerIDFromRequest(r, 0)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	authResp, ch, err := h.sellerAuth(r.Context(), sellerID)
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}

	st, err := h.stream.Stream.StartStream(r.Context(), &streamv1.StartStreamRequest{
		UserId: authResp.GetUser().GetId(), StreamId: chi.URLParam(r, "stream_id"),
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	h.sendMarketEvent(r.Context(), "stream.started", sellerID, ch.GetMarketplaceShopId(), st.GetId())
	httputil.JSON(w, http.StatusOK, streamToJSON(st))
}

func (h *BroadcastHandler) EndSellerStream(w http.ResponseWriter, r *http.Request) {
	sellerID, err := h.sellerIDFromRequest(r, 0)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	authResp, ch, err := h.sellerAuth(r.Context(), sellerID)
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}

	st, err := h.stream.Stream.EndStream(r.Context(), &streamv1.EndStreamRequest{
		UserId: authResp.GetUser().GetId(), StreamId: chi.URLParam(r, "stream_id"),
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	h.sendMarketEvent(r.Context(), "stream.ended", sellerID, ch.GetMarketplaceShopId(), st.GetId())
	httputil.JSON(w, http.StatusOK, streamToJSON(st))
}

func (h *BroadcastHandler) GetStreamPlayback(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "stream_id")
	st, stErr := h.stream.Stream.GetStream(r.Context(), &streamv1.GetStreamRequest{StreamId: streamID})

	resp, err := h.stream.Stream.GetPlayback(r.Context(), &streamv1.GetPlaybackRequest{StreamId: streamID})
	if err != nil {
		if stErr == nil && st != nil && st.Status == "live" && st.LatencyMode == "ultra-low" && h.whipBase != "" {
			httputil.JSON(w, http.StatusOK, map[string]any{
				"stream_id": streamID, "url": h.whipBase + "/" + streamID + "/whep",
				"format": "webrtc", "playback_mode": "whep", "status": "live",
			})
			return
		}
		httputil.Error(w, grpcError(err))
		return
	}

	out := map[string]any{
		"stream_id": resp.StreamId, "url": resp.Url, "format": resp.Format,
		"status": resp.Status, "expires_at_unix": resp.ExpiresAtUnix,
	}
	if stErr == nil && st != nil && st.LatencyMode == "ultra-low" && h.whipBase != "" {
		out["whep_url"] = h.whipBase + "/" + streamID + "/whep"
		out["playback_mode"] = "dual"
	}
	httputil.JSON(w, http.StatusOK, out)
}

func (h *BroadcastHandler) sendMarketEvent(_ context.Context, event string, sellerID, shopID int64, streamID string) {
	if h.marketWebhookURL == "" || h.marketWebhookSecret == "" || streamID == "" {
		return
	}
	body, err := json.Marshal(map[string]any{
		"event":                 event,
		"stream_id":             streamID,
		"marketplace_seller_id": sellerID,
		"marketplace_shop_id":   shopID,
	})
	if err != nil {
		return
	}
	// Use detached context so client disconnect won't drop platform event.
	sendCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	backoff := 150 * time.Millisecond
	for i := 0; i < 3; i++ {
		req, reqErr := http.NewRequestWithContext(sendCtx, http.MethodPost, h.marketWebhookURL, bytes.NewReader(body))
		if reqErr != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Stream-Secret", h.marketWebhookSecret)
		resp, doErr := h.webhookClient.Do(req)
		if doErr == nil && resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return
			}
		}
		select {
		case <-sendCtx.Done():
			return
		case <-time.After(backoff):
			backoff *= 2
		}
	}
}
