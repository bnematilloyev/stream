package handler

import (
	"net/http"

	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/middleware"
	authv1 "github.com/sahiy/sahiy-stream/proto/gen/auth/v1"
)

func requireUser(w http.ResponseWriter, r *http.Request) *authv1.User {
	u := middleware.GetUser(r)
	if u == nil {
		httputil.Error(w, apperrors.Unauthorized("unauthorized"))
		return nil
	}
	return u
}
