package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/client"
	userv1 "github.com/sahiy/sahiy-stream/proto/gen/user/v1"
)

type UserHandler struct{ user *client.UserClient }

func NewUserHandler(user *client.UserClient) *UserHandler { return &UserHandler{user: user} }

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	profile, err := h.user.User.GetProfile(r.Context(), &userv1.GetProfileRequest{UserId: u.ID})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, profileToJSON(profile))
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var body struct {
		DisplayName *string `json:"display_name"`
		AvatarURL   *string `json:"avatar_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, decodeError(err))
		return
	}
	profile, err := h.user.User.UpdateProfile(r.Context(), &userv1.UpdateProfileRequest{
		UserId: u.ID, DisplayName: body.DisplayName, AvatarUrl: body.AvatarURL,
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, profileToJSON(profile))
}

func (h *UserHandler) GetByUsername(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	profile, err := h.user.User.GetPublicProfile(r.Context(), &userv1.GetPublicProfileRequest{Username: username})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"id": profile.Id, "username": profile.Username, "display_name": profile.DisplayName,
	})
}

func profileToJSON(p *userv1.Profile) map[string]any {
	return map[string]any{
		"id": p.Id, "email": p.Email, "username": p.Username, "display_name": p.DisplayName,
		"avatar_url": p.AvatarUrl, "role": p.Role, "email_verified": p.EmailVerified,
		"created_at_unix": p.CreatedAtUnix,
	}
}
