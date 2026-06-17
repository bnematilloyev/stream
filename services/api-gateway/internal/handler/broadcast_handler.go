package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
	authv1 "github.com/sahiy/sahiy-stream/proto/gen/auth/v1"
	streamv1 "github.com/sahiy/sahiy-stream/proto/gen/stream/v1"
	userv1 "github.com/sahiy/sahiy-stream/proto/gen/user/v1"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type BroadcastHandler struct {
	auth                *client.AuthClient
	user                *client.UserClient
	stream              *client.StreamClient
	secret              string
	whipBase            string
	marketWebhookURL    string
	marketWebhookSecret string
	webhookClient       *http.Client
}

func NewBroadcastHandler(
	auth *client.AuthClient,
	user *client.UserClient,
	stream *client.StreamClient,
	provisionSecret, whipBaseURL, marketWebhookURL, marketWebhookSecret string,
) *BroadcastHandler {
	return &BroadcastHandler{
		auth: auth, user: user, stream: stream, secret: provisionSecret, whipBase: whipBaseURL,
		marketWebhookURL: marketWebhookURL, marketWebhookSecret: marketWebhookSecret,
		webhookClient: &http.Client{Timeout: 3 * time.Second},
	}
}

type provisionChannelRequest struct {
	MarketplaceSellerID int64  `json:"marketplace_seller_id"`
	MarketplaceShopID   int64  `json:"marketplace_shop_id"`
	Title               string `json:"title"`
}

func (h *BroadcastHandler) ProvisionChannel(w http.ResponseWriter, r *http.Request) {
	var req provisionChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, decodeError(err))
		return
	}
	if req.MarketplaceSellerID <= 0 || req.MarketplaceShopID <= 0 {
		httputil.Error(w, apperrors.Validation("marketplace_seller_id and marketplace_shop_id are required", nil))
		return
	}
	if req.Title == "" {
		httputil.Error(w, apperrors.Validation("title is required", nil))
		return
	}

	existing, err := h.user.Channel.GetChannelByMarketplaceSeller(r.Context(), &userv1.GetChannelByMarketplaceSellerRequest{
		MarketplaceSellerId: req.MarketplaceSellerID,
	})
	if err != nil && status.Code(err) != codes.NotFound {
		httputil.Error(w, grpcError(err))
		return
	}
	if existing != nil {
		httputil.JSON(w, http.StatusOK, map[string]any{
			"slug":                  existing.GetSlug(),
			"marketplace_seller_id": req.MarketplaceSellerID,
			"marketplace_shop_id":   req.MarketplaceShopID,
			"provisioned":           false,
		})
		return
	}

	email := fmt.Sprintf("seller-%d@broadcast.internal.sahiy", req.MarketplaceSellerID)
	username := fmt.Sprintf("seller_%d", req.MarketplaceSellerID)
	password := provisionPassword(h.secret, req.MarketplaceSellerID)
	slug := fmt.Sprintf("shop-%d", req.MarketplaceShopID)

	authResp, err := h.auth.Register(r.Context(), &authv1.RegisterRequest{
		Email:       email,
		Username:    username,
		DisplayName: req.Title,
		Password:    password,
	})
	if err != nil {
		if status.Code(err) != codes.AlreadyExists && status.Code(err) != codes.FailedPrecondition {
			httputil.Error(w, grpcError(err))
			return
		}
		authResp, err = h.auth.Login(r.Context(), &authv1.LoginRequest{
			Email:    email,
			Password: password,
		})
		if err != nil {
			httputil.Error(w, grpcError(err))
			return
		}
	}

	ch, err := h.user.Channel.CreateChannel(r.Context(), &userv1.CreateChannelRequest{
		UserId:              authResp.GetUser().GetId(),
		Slug:                slug,
		Title:               req.Title,
		MarketplaceSellerId: req.MarketplaceSellerID,
		MarketplaceShopId:   req.MarketplaceShopID,
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}

	httputil.JSON(w, http.StatusCreated, map[string]any{
		"slug":                  ch.GetSlug(),
		"marketplace_seller_id": req.MarketplaceSellerID,
		"marketplace_shop_id":   req.MarketplaceShopID,
		"provisioned":           true,
	})
}

func (h *BroadcastHandler) ListMarketplaceLive(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	resp, err := h.stream.Stream.ListMarketplaceLiveStreams(r.Context(), &streamv1.ListMarketplaceLiveStreamsRequest{
		Page:  int32(page),
		Limit: int32(limit),
	})
	if err != nil {
		httputil.Error(w, grpcError(err))
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=2, stale-while-revalidate=5")
	httputil.JSON(w, http.StatusOK, streamsListToJSON(resp))
}

func provisionPassword(secret string, sellerID int64) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", secret, sellerID)))
	return hex.EncodeToString(sum[:])
}
