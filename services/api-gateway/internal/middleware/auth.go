package middleware

import (
	"context"
	"net/http"
	"strings"

	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/client"
	authv1 "github.com/sahiy/sahiy-stream/proto/gen/auth/v1"
)

type contextKey string

const UserContextKey contextKey = "user"

func Authenticate(auth *client.AuthClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r.Header.Get("Authorization"))
			if token == "" {
				httputil.Error(w, apperrors.Unauthorized("missing authorization token"))
				return
			}

			resp, err := auth.ValidateToken(r.Context(), &authv1.ValidateTokenRequest{
				AccessToken: token,
			})
			if err != nil || resp == nil || !resp.Valid || resp.User == nil {
				httputil.Error(w, apperrors.Unauthorized("invalid or expired token"))
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, resp.User)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func OptionalAuth(auth *client.AuthClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r.Header.Get("Authorization"))
			if token != "" {
				resp, err := auth.ValidateToken(r.Context(), &authv1.ValidateTokenRequest{
					AccessToken: token,
				})
				if err == nil && resp != nil && resp.Valid && resp.User != nil {
					ctx := context.WithValue(r.Context(), UserContextKey, resp.User)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func GetUser(r *http.Request) *authv1.User {
	user, _ := r.Context().Value(UserContextKey).(*authv1.User)
	return user
}

func extractBearer(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
