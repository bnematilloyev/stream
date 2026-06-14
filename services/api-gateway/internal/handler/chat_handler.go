package handler

import (
	"fmt"
	"net/http"
	stdhttputil "net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/client"
	chatv1 "github.com/sahiy/sahiy-stream/proto/gen/chat/v1"
)

type ChatHandler struct {
	chat  *client.ChatClient
	proxy *stdhttputil.ReverseProxy
}

func NewChatHandler(chat *client.ChatClient, chatHTTPAddr string) (*ChatHandler, error) {
	host := strings.TrimPrefix(chatHTTPAddr, "http://")
	host = strings.TrimPrefix(host, "https://")
	target, err := url.Parse("http://" + host)
	if err != nil {
		return nil, fmt.Errorf("parse chat http addr: %w", err)
	}
	return &ChatHandler{
		chat:  chat,
		proxy: stdhttputil.NewSingleHostReverseProxy(target),
	}, nil
}

func (h *ChatHandler) History(w http.ResponseWriter, r *http.Request) {
	beforeID, _ := strconv.ParseInt(r.URL.Query().Get("cursor"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	resp, err := h.chat.Chat.GetHistory(r.Context(), &chatv1.GetHistoryRequest{
		StreamId: chi.URLParam(r, "streamID"),
		BeforeId: beforeID,
		Limit:    int32(limit),
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	items := make([]map[string]any, 0, len(resp.Messages))
	for _, m := range resp.Messages {
		items = append(items, map[string]any{
			"id": m.Id, "stream_id": m.StreamId, "user_id": m.UserId,
			"username": m.Username, "display_name": m.DisplayName,
			"content": m.Content, "type": m.Type, "created_at_unix": m.CreatedAtUnix,
		})
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"data": items, "has_more": resp.HasMore,
	})
}

func (h *ChatHandler) WebSocket(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "streamID")
	r.URL.Path = "/v1/chat/" + streamID
	h.proxy.ServeHTTP(w, r)
}

func (h *ChatHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	messageID, err := strconv.ParseInt(chi.URLParam(r, "messageID"), 10, 64)
	if err != nil || messageID <= 0 {
		httputil.Error(w, validationError("invalid message id"))
		return
	}
	_, err = h.chat.Chat.DeleteMessage(r.Context(), &chatv1.DeleteMessageRequest{
		StreamId:    chi.URLParam(r, "streamID"),
		MessageId:   messageID,
		ActorUserId: u.ID,
		ActorRole:   u.Role,
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]bool{"success": true})
}
