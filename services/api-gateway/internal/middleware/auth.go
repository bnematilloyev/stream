package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/sahiy/sahiy-stream/pkg/auth"
	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
)

type contextKey string

const UserContextKey contextKey = "user"

func Authenticate(validator *auth.Validator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r.Header.Get("Authorization"))
			if token == "" {
				httputil.Error(w, apperrors.Unauthorized("missing authorization token"))
				return
			}

			user, err := validator.ValidateAccess(r.Context(), token)
			if err != nil {
				if appErr, ok := apperrors.IsAppError(err); ok {
					httputil.Error(w, appErr)
					return
				}
				httputil.Error(w, apperrors.Unauthorized("invalid or expired token"))
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func OptionalAuth(validator *auth.Validator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r.Header.Get("Authorization"))
			if token != "" {
				if user, err := validator.ValidateAccess(r.Context(), token); err == nil && user != nil {
					ctx := context.WithValue(r.Context(), UserContextKey, user)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func GetUser(r *http.Request) *auth.Principal {
	user, _ := r.Context().Value(UserContextKey).(*auth.Principal)
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
