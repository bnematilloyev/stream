package ws

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sahiy/sahiy-stream/pkg/auth"
	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/domain"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/usecase"
	"go.uber.org/zap"
)

// FeaturedHandler serves the live-shopping spotlight endpoints. Mutating routes
// authenticate the broadcaster's bearer token directly (same model as the WS
// handler); the read route is public so viewers can hydrate on join.
type FeaturedHandler struct {
	uc        *usecase.FeaturedUseCase
	validator *auth.Validator
	log       *zap.Logger
}

func NewFeaturedHandler(uc *usecase.FeaturedUseCase, validator *auth.Validator, log *zap.Logger) *FeaturedHandler {
	return &FeaturedHandler{uc: uc, validator: validator, log: log}
}

func (h *FeaturedHandler) streamID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "streamID"))
	if err != nil {
		httputil.Error(w, apperrors.Validation("invalid stream id", nil))
		return uuid.Nil, false
	}
	return id, true
}

func (h *FeaturedHandler) principal(w http.ResponseWriter, r *http.Request) (*auth.Principal, bool) {
	token := extractToken(r)
	if token == "" {
		httputil.Error(w, apperrors.New(apperrors.CodeUnauthorized, "authentication required", http.StatusUnauthorized))
		return nil, false
	}
	principal, err := h.validator.ValidateAccess(r.Context(), token)
	if err != nil {
		httputil.Error(w, apperrors.New(apperrors.CodeUnauthorized, "invalid or expired token", http.StatusUnauthorized))
		return nil, false
	}
	return principal, true
}

func (h *FeaturedHandler) Get(w http.ResponseWriter, r *http.Request) {
	streamID, ok := h.streamID(w, r)
	if !ok {
		return
	}
	product, err := h.uc.Get(r.Context(), streamID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"data": product})
}

func (h *FeaturedHandler) Set(w http.ResponseWriter, r *http.Request) {
	streamID, ok := h.streamID(w, r)
	if !ok {
		return
	}
	principal, ok := h.principal(w, r)
	if !ok {
		return
	}
	var product domain.FeaturedProduct
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		httputil.Error(w, apperrors.Validation("invalid request body", nil))
		return
	}
	if err := h.uc.Set(r.Context(), streamID, principal.ID, principal.Role, product); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"data": product})
}

func (h *FeaturedHandler) Clear(w http.ResponseWriter, r *http.Request) {
	streamID, ok := h.streamID(w, r)
	if !ok {
		return
	}
	principal, ok := h.principal(w, r)
	if !ok {
		return
	}
	if err := h.uc.Clear(r.Context(), streamID, principal.ID, principal.Role); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]bool{"success": true})
}
