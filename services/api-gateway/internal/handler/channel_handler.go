package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/client"
	userv1 "github.com/sahiy/sahiy-stream/proto/gen/user/v1"
)

type ChannelHandler struct {
	user        *client.UserClient
	whipBaseURL string
}

func NewChannelHandler(user *client.UserClient, whipBaseURL string) *ChannelHandler {
	return &ChannelHandler{user: user, whipBaseURL: whipBaseURL}
}

func (h *ChannelHandler) ingestToJSON(streamKey, rtmpURL, srtURL, keyPrefix string) map[string]any {
	out := map[string]any{
		"stream_key": streamKey, "rtmp_url": rtmpURL, "srt_url": srtURL,
		"key_prefix": keyPrefix, "whip_base_url": h.whipBaseURL,
	}
	if streamKey != "" {
		out["whip_url"] = h.whipBaseURL + "/" + streamKey + "/whip"
	}
	return out
}

func (h *ChannelHandler) Create(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var body struct {
		Slug        string `json:"slug"`
		Title       string `json:"title"`
		Description string `json:"description"`
		CategoryID  string `json:"category_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, decodeError(err))
		return
	}
	ch, err := h.user.Channel.CreateChannel(r.Context(), &userv1.CreateChannelRequest{
		UserId: u.Id, Slug: body.Slug, Title: body.Title, Description: body.Description, CategoryId: body.CategoryID,
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusCreated, channelToJSON(ch))
}

func (h *ChannelHandler) Get(w http.ResponseWriter, r *http.Request) {
	ch, err := h.user.Channel.GetChannel(r.Context(), &userv1.GetChannelRequest{Slug: chi.URLParam(r, "slug")})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, channelToJSON(ch))
}

func (h *ChannelHandler) GetMine(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	ch, err := h.user.Channel.GetMyChannel(r.Context(), &userv1.GetMyChannelRequest{UserId: u.Id})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, channelToJSON(ch))
}

func (h *ChannelHandler) Update(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var body struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		BannerURL   *string `json:"banner_url"`
		AvatarURL   *string `json:"avatar_url"`
		CategoryID  *string `json:"category_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, decodeError(err))
		return
	}
	ch, err := h.user.Channel.UpdateChannel(r.Context(), &userv1.UpdateChannelRequest{
		UserId: u.Id, Slug: chi.URLParam(r, "slug"),
		Title: body.Title, Description: body.Description, BannerUrl: body.BannerURL,
		AvatarUrl: body.AvatarURL, CategoryId: body.CategoryID,
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, channelToJSON(ch))
}

func (h *ChannelHandler) Follow(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	resp, err := h.user.Channel.Follow(r.Context(), &userv1.FollowRequest{UserId: u.Id, ChannelSlug: chi.URLParam(r, "slug")})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"success": resp.Success, "follower_count": resp.FollowerCount})
}

func (h *ChannelHandler) Unfollow(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	resp, err := h.user.Channel.Unfollow(r.Context(), &userv1.UnfollowRequest{UserId: u.Id, ChannelSlug: chi.URLParam(r, "slug")})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"success": resp.Success, "follower_count": resp.FollowerCount})
}

func (h *ChannelHandler) Followers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	resp, err := h.user.Channel.ListFollowers(r.Context(), &userv1.ListFollowersRequest{
		ChannelSlug: chi.URLParam(r, "slug"), Page: int32(page), Limit: int32(limit),
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *ChannelHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	resp, err := h.user.Channel.GetIngestKey(r.Context(), &userv1.GetIngestKeyRequest{UserId: u.Id, ChannelSlug: chi.URLParam(r, "slug")})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, h.ingestToJSON(resp.StreamKey, resp.RtmpUrl, resp.SrtUrl, resp.KeyPrefix))
}

func (h *ChannelHandler) RotateKey(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	resp, err := h.user.Channel.RotateIngestKey(r.Context(), &userv1.RotateIngestKeyRequest{UserId: u.Id, ChannelSlug: chi.URLParam(r, "slug")})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, h.ingestToJSON(resp.StreamKey, resp.RtmpUrl, resp.SrtUrl, resp.KeyPrefix))
}

func channelToJSON(ch *userv1.Channel) map[string]any {
	return map[string]any{
		"id": ch.Id, "user_id": ch.UserId, "slug": ch.Slug, "title": ch.Title,
		"description": ch.Description, "banner_url": ch.BannerUrl, "avatar_url": ch.AvatarUrl,
		"category_id": ch.CategoryId, "category_slug": ch.CategorySlug,
		"is_verified": ch.IsVerified, "is_live": ch.IsLive, "follower_count": ch.FollowerCount,
		"created_at_unix": ch.CreatedAtUnix, "updated_at_unix": ch.UpdatedAtUnix,
	}
}
