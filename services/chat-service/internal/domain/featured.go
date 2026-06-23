package domain

import (
	"context"

	"github.com/google/uuid"
)

// FeaturedProduct is an opaque, provider-agnostic snapshot of the product a
// broadcaster is currently spotlighting on screen. The marketplace owns the
// canonical product data; the stream platform only relays this card to viewers.
type FeaturedProduct struct {
	ProductID string `json:"product_id"`
	Title     string `json:"title"`
	ImageURL  string `json:"image_url,omitempty"`
	Price     int64  `json:"price,omitempty"`
	Currency  string `json:"currency,omitempty"`
	URL       string `json:"url,omitempty"`
}

// FeaturedStore persists the currently featured product per stream so that
// viewers who join mid-stream can fetch the active spotlight.
type FeaturedStore interface {
	Set(ctx context.Context, streamID uuid.UUID, product FeaturedProduct) error
	Get(ctx context.Context, streamID uuid.UUID) (*FeaturedProduct, error)
	Clear(ctx context.Context, streamID uuid.UUID) error
}
