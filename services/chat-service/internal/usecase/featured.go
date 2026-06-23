package usecase

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	pkgnats "github.com/sahiy/sahiy-stream/pkg/nats"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/domain"
)

// featuredEvent is broadcast over the chat bus so every connected viewer's
// player overlay updates the moment the broadcaster spotlights a product.
// A nil Product means the spotlight was cleared.
type featuredEvent struct {
	Type     string                  `json:"type"`
	StreamID string                  `json:"stream_id"`
	Product  *domain.FeaturedProduct `json:"product"`
}

const featuredEventType = "featured_product"

type FeaturedUseCase struct {
	store     domain.FeaturedStore
	moderator domain.StreamModerator
	bus       *pkgnats.ChatBus
}

func NewFeaturedUseCase(
	store domain.FeaturedStore,
	moderator domain.StreamModerator,
	bus *pkgnats.ChatBus,
) *FeaturedUseCase {
	return &FeaturedUseCase{store: store, moderator: moderator, bus: bus}
}

func (uc *FeaturedUseCase) authorize(ctx context.Context, streamID uuid.UUID, userID, role string) error {
	ok, err := uc.moderator.CanModerate(ctx, streamID, userID, role)
	if err != nil {
		return apperrors.Internal(err)
	}
	if !ok {
		return apperrors.Forbidden("only the broadcaster can feature products")
	}
	return nil
}

func (uc *FeaturedUseCase) Set(ctx context.Context, streamID uuid.UUID, userID, role string, product domain.FeaturedProduct) error {
	product.Title = strings.TrimSpace(product.Title)
	product.ProductID = strings.TrimSpace(product.ProductID)
	if product.ProductID == "" || product.Title == "" {
		return apperrors.Validation("product_id and title are required", nil)
	}
	if err := uc.authorize(ctx, streamID, userID, role); err != nil {
		return err
	}
	if err := uc.store.Set(ctx, streamID, product); err != nil {
		return apperrors.Internal(err)
	}
	return uc.broadcast(ctx, streamID, &product)
}

func (uc *FeaturedUseCase) Clear(ctx context.Context, streamID uuid.UUID, userID, role string) error {
	if err := uc.authorize(ctx, streamID, userID, role); err != nil {
		return err
	}
	if err := uc.store.Clear(ctx, streamID); err != nil {
		return apperrors.Internal(err)
	}
	return uc.broadcast(ctx, streamID, nil)
}

func (uc *FeaturedUseCase) Get(ctx context.Context, streamID uuid.UUID) (*domain.FeaturedProduct, error) {
	product, err := uc.store.Get(ctx, streamID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return product, nil
}

func (uc *FeaturedUseCase) broadcast(ctx context.Context, streamID uuid.UUID, product *domain.FeaturedProduct) error {
	payload, err := json.Marshal(featuredEvent{
		Type:     featuredEventType,
		StreamID: streamID.String(),
		Product:  product,
	})
	if err != nil {
		return apperrors.Internal(err)
	}
	if err := uc.bus.Publish(ctx, streamID.String(), payload); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}
