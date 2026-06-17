package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
	authv1 "github.com/sahiy/sahiy-stream/proto/gen/auth/v1"
	streamv1 "github.com/sahiy/sahiy-stream/proto/gen/stream/v1"
	userv1 "github.com/sahiy/sahiy-stream/proto/gen/user/v1"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/client"
)

type AdminHandler struct {
	auth   *client.AuthClient
	user   *client.UserClient
	stream *client.StreamClient
}

func NewAdminHandler(auth *client.AuthClient, user *client.UserClient, stream *client.StreamClient) *AdminHandler {
	return &AdminHandler{auth: auth, user: user, stream: stream}
}

func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.auth.GetPlatformStats(r.Context(), &authv1.GetPlatformStatsRequest{})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	live, err := h.stream.Stream.ListLiveStreams(r.Context(), &streamv1.ListLiveStreamsRequest{Page: 1, Limit: 1})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	market, err := h.stream.Stream.ListMarketplaceLiveStreams(r.Context(), &streamv1.ListMarketplaceLiveStreamsRequest{Page: 1, Limit: 1})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"users": map[string]any{
			"total":     stats.TotalUsers,
			"active":    stats.UsersActive,
			"suspended": stats.UsersSuspended,
			"banned":    stats.UsersBanned,
			"admins":    stats.Admins,
		},
		"streams": map[string]any{
			"live_total":       live.Total,
			"live_marketplace": market.Total,
		},
	})
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	resp, err := h.auth.ListUsers(r.Context(), &authv1.ListUsersRequest{
		Status: r.URL.Query().Get("status"),
		Role:   r.URL.Query().Get("role"),
		Search: r.URL.Query().Get("search"),
		Page:   int32(page),
		Limit:  int32(limit),
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	users := make([]map[string]any, 0, len(resp.Users))
	for _, u := range resp.Users {
		users = append(users, adminUserToJSON(u))
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"data": users,
		"pagination": map[string]any{
			"page": resp.Page, "limit": resp.Limit, "total": resp.Total,
		},
	})
}

func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	actor := requireUser(w, r)
	if actor == nil {
		return
	}
	var body struct {
		Role   *string `json:"role"`
		Status *string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, decodeError(err))
		return
	}
	req := &authv1.UpdateUserAdminRequest{
		UserId:  chi.URLParam(r, "id"),
		ActorId: actor.ID,
		Role:    body.Role,
		Status:  body.Status,
	}
	user, err := h.auth.UpdateUserAdmin(r.Context(), req)
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, adminUserToJSON(user))
}

func (h *AdminHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	resp, err := h.auth.ListAuditLogs(r.Context(), &authv1.ListAuditLogsRequest{
		Page: int32(page), Limit: int32(limit),
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	logs := make([]map[string]any, 0, len(resp.Logs))
	for _, entry := range resp.Logs {
		logs = append(logs, map[string]any{
			"id": entry.Id, "actor_id": entry.ActorId, "action": entry.Action,
			"resource_type": entry.ResourceType, "resource_id": entry.ResourceId,
			"details": entry.DetailsJson, "created_at_unix": entry.CreatedAtUnix,
		})
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"data": logs,
		"pagination": map[string]any{
			"page": resp.Page, "limit": resp.Limit, "total": resp.Total,
		},
	})
}

func (h *AdminHandler) ListChannels(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	marketplaceOnly := r.URL.Query().Get("marketplace_only") == "true"
	resp, err := h.user.Channel.ListChannels(r.Context(), &userv1.ListChannelsRequest{
		Search:          r.URL.Query().Get("search"),
		MarketplaceOnly: marketplaceOnly,
		Page:            int32(page),
		Limit:           int32(limit),
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	channels := make([]map[string]any, 0, len(resp.Channels))
	for _, ch := range resp.Channels {
		channels = append(channels, adminChannelToJSON(ch))
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"data": channels,
		"pagination": map[string]any{
			"page": resp.Page, "limit": resp.Limit, "total": resp.Total,
		},
	})
}

func (h *AdminHandler) UpdateChannel(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IsVerified *bool `json:"is_verified"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, decodeError(err))
		return
	}
	if body.IsVerified == nil {
		httputil.Error(w, validationError("is_verified is required"))
		return
	}
	ch, err := h.user.Channel.AdminUpdateChannel(r.Context(), &userv1.AdminUpdateChannelRequest{
		Slug:       chi.URLParam(r, "slug"),
		IsVerified: body.IsVerified,
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, adminChannelToJSON(ch))
}

func (h *AdminHandler) ListLiveStreams(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	marketplaceOnly := r.URL.Query().Get("marketplace_only") == "true"

	var resp *streamv1.ListStreamsResponse
	var err error
	if marketplaceOnly {
		resp, err = h.stream.Stream.ListMarketplaceLiveStreams(r.Context(), &streamv1.ListMarketplaceLiveStreamsRequest{
			Page: int32(page), Limit: int32(limit),
		})
	} else {
		resp, err = h.stream.Stream.ListLiveStreams(r.Context(), &streamv1.ListLiveStreamsRequest{
			Page: int32(page), Limit: int32(limit),
		})
	}
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, streamsListToJSON(resp))
}

func (h *AdminHandler) ForceEndStream(w http.ResponseWriter, r *http.Request) {
	st, err := h.stream.Stream.AdminForceEndStream(r.Context(), &streamv1.AdminForceEndStreamRequest{
		StreamId: chi.URLParam(r, "id"),
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, streamToJSON(st))
}

func adminUserToJSON(u *authv1.User) map[string]any {
	return map[string]any{
		"id": u.Id, "email": u.Email, "username": u.Username, "display_name": u.DisplayName,
		"role": u.Role, "status": u.Status, "email_verified": u.EmailVerified,
		"created_at_unix": u.CreatedAtUnix,
	}
}

func adminChannelToJSON(ch *userv1.Channel) map[string]any {
	out := channelToJSON(ch)
	out["marketplace_seller_id"] = ch.MarketplaceSellerId
	out["marketplace_shop_id"] = ch.MarketplaceShopId
	return out
}
