package httputil

import (
	"encoding/json"
	"net/http"

	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
)

type ErrorBody struct {
	Error apperrors.AppError `json:"error"`
}

func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func Error(w http.ResponseWriter, err error) {
	if appErr, ok := apperrors.IsAppError(err); ok {
		JSON(w, appErr.HTTPStatus, ErrorBody{Error: *appErr})
		return
	}
	JSON(w, http.StatusInternalServerError, ErrorBody{
		Error: *apperrors.Internal(err),
	})
}
