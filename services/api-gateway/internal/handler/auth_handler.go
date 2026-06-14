package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/sahiy/sahiy-stream/pkg/httputil"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/client"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/middleware"
	"github.com/sahiy/sahiy-stream/pkg/auth"
	authv1 "github.com/sahiy/sahiy-stream/proto/gen/auth/v1"
)

type AuthHandler struct {
	auth *client.AuthClient
}

func NewAuthHandler(auth *client.AuthClient) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerRequest struct {
	Email       string `json:"email"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type authResponse struct {
	User         userResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
}

type userResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Username      string `json:"username"`
	DisplayName   string `json:"display_name"`
	Role          string `json:"role"`
	Status        string `json:"status"`
	EmailVerified bool   `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, decodeError(err))
		return
	}

	resp, err := h.auth.Register(r.Context(), &authv1.RegisterRequest{
		Email:       req.Email,
		Username:    req.Username,
		DisplayName: req.DisplayName,
		Password:    req.Password,
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}

	setRefreshCookie(w, resp.RefreshToken)
	httputil.JSON(w, http.StatusCreated, toAuthResponse(resp))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, decodeError(err))
		return
	}

	resp, err := h.auth.Login(r.Context(), &authv1.LoginRequest{
		Email:      req.Email,
		Password:   req.Password,
		DeviceInfo: r.UserAgent(),
		IpAddress:  httputil.ClientIP(r),
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}

	setRefreshCookie(w, resp.RefreshToken)
	httputil.JSON(w, http.StatusOK, toAuthResponse(resp))
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	token := refreshTokenFromRequest(r)
	if token == "" {
		var req refreshRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		token = req.RefreshToken
	}
	if token == "" {
		httputil.Error(w, validationError("refresh_token is required"))
		return
	}

	resp, err := h.auth.Refresh(r.Context(), &authv1.RefreshRequest{RefreshToken: token})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}

	setRefreshCookie(w, resp.RefreshToken)
	httputil.JSON(w, http.StatusOK, toAuthResponse(resp))
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := refreshTokenFromRequest(r)
	if token == "" {
		var req refreshRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		token = req.RefreshToken
	}

	if token != "" {
		_, _ = h.auth.Logout(r.Context(), &authv1.LogoutRequest{RefreshToken: token})
	}

	clearRefreshCookie(w)
	httputil.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		httputil.JSON(w, http.StatusOK, map[string]any{})
		return
	}
	httputil.JSON(w, http.StatusOK, toPrincipalResponse(user))
}

func refreshTokenFromRequest(r *http.Request) string {
	if c, err := r.Cookie("refresh_token"); err == nil {
		return c.Value
	}
	return ""
}

func setRefreshCookie(w http.ResponseWriter, token string) {
	secure := os.Getenv("APP_ENV") == "production"
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/v1/auth",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
	})
}

func clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/v1/auth",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func toAuthResponse(resp *authv1.AuthResponse) authResponse {
	return authResponse{
		User:         toUserResponse(resp.User),
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresAt:    time.Unix(resp.ExpiresAtUnix, 0).UTC(),
	}
}

func toPrincipalResponse(u *auth.Principal) userResponse {
	return userResponse{
		ID:            u.ID,
		Email:         u.Email,
		Username:      u.Username,
		DisplayName:   u.DisplayName,
		Role:          u.Role,
		Status:        u.Status,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
	}
}

func toUserResponse(u *authv1.User) userResponse {
	return userResponse{
		ID:            u.Id,
		Email:         u.Email,
		Username:      u.Username,
		DisplayName:   u.DisplayName,
		Role:          u.Role,
		Status:        u.Status,
		EmailVerified: u.EmailVerified,
		CreatedAt:     time.Unix(u.CreatedAtUnix, 0).UTC(),
	}
}
