package handler

import (
	"net/http"

	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/auth"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/middleware"
)

func requireUser(w http.ResponseWriter, r *http.Request) *auth.Principal {
	u := middleware.GetUser(r)
	if u == nil {
		httputil.Error(w, apperrors.Unauthorized("unauthorized"))
		return nil
	}
	return u
}
