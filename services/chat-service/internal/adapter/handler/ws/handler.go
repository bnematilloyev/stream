package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sahiy/sahiy-stream/pkg/auth"
	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/usecase"
	"go.uber.org/zap"
)

type Handler struct {
	uc        *usecase.ChatUseCase
	hub       *Hub
	validator *auth.Validator
	upgrader  websocket.Upgrader
	log       *zap.Logger
}

func NewHandler(
	uc *usecase.ChatUseCase,
	hub *Hub,
	validator *auth.Validator,
	allowedOrigins []string,
	appEnv string,
	log *zap.Logger,
) *Handler {
	return &Handler{
		uc:        uc,
		hub:       hub,
		validator: validator,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     originAllowed(allowedOrigins, appEnv),
		},
		log: log,
	}
}

func originAllowed(origins []string, appEnv string) func(*http.Request) bool {
	allowed := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		allowed[strings.TrimSpace(o)] = struct{}{}
	}
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		if _, ok := allowed[origin]; ok {
			return true
		}
		if appEnv == "development" {
			return strings.HasPrefix(origin, "http://localhost:") ||
				strings.HasPrefix(origin, "http://127.0.0.1:")
		}
		return false
	}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/v1/chat/{streamID}", h.ServeWS)
	return r
}

type inboundMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	streamID, err := uuid.Parse(chi.URLParam(r, "streamID"))
	if err != nil {
		http.Error(w, "invalid stream id", http.StatusBadRequest)
		return
	}

	var principal *auth.Principal
	if token := extractToken(r); token != "" {
		principal, err = h.validator.ValidateAccess(r.Context(), token)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Warn("ws upgrade failed", zap.Error(err))
		return
	}

	client := NewClient(conn)
	streamKey := streamID.String()
	h.hub.Join(streamKey, client)

	go client.WritePump()
	defer func() {
		h.hub.Leave(streamKey, client)
		client.Close()
	}()

	client.ReadPump(func(data []byte) {
		if principal == nil {
			h.writeError(client, "authentication required to send messages")
			return
		}
		var msg inboundMessage
		if err := json.Unmarshal(data, &msg); err != nil || msg.Type != "message" {
			h.writeError(client, "invalid message format")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := h.uc.Send(ctx, usecase.SendInput{
			StreamID:    streamID,
			UserID:      uuid.MustParse(principal.ID),
			Username:    principal.Username,
			DisplayName: principal.DisplayName,
			Content:     msg.Content,
		})
		if err != nil {
			if appErr, ok := apperrors.IsAppError(err); ok {
				h.writeError(client, appErr.Message)
				return
			}
			h.writeError(client, "failed to send message")
		}
	})
}

func extractToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header != "" {
		parts := strings.SplitN(header, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			return strings.TrimSpace(parts[1])
		}
	}
	return r.URL.Query().Get("token")
}

func (h *Handler) writeError(client *Client, message string) {
	payload, _ := json.Marshal(map[string]string{"type": "error", "content": message})
	client.Send(payload)
}

func (h *Handler) Broadcast(streamID string, payload []byte) {
	h.hub.Broadcast(streamID, payload)
}
