package middleware

import (
	"net/http"

	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
)

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r)
			if user == nil {
				httputil.Error(w, apperrors.Unauthorized("unauthorized"))
				return
			}
			if _, ok := allowed[user.Role]; !ok {
				httputil.Error(w, apperrors.Forbidden("admin access required"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
