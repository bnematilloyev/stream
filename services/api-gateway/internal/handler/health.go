package handler

import (
	"net/http"

	"github.com/sahiy/sahiy-stream/pkg/httputil"
)

func Health(w http.ResponseWriter, _ *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "api-gateway",
	})
}
