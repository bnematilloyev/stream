package authadapter

import (
	"context"
	"time"

	"github.com/sahiy/sahiy-stream/pkg/auth"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/client"
	authv1 "github.com/sahiy/sahiy-stream/proto/gen/auth/v1"
)

// GRPCUserFetcher loads user details from auth-service on cache miss.
type GRPCUserFetcher struct {
	client *client.AuthClient
}

func NewGRPCUserFetcher(c *client.AuthClient) *GRPCUserFetcher {
	return &GRPCUserFetcher{client: c}
}

func (f *GRPCUserFetcher) FetchUser(ctx context.Context, userID string) (*auth.Principal, error) {
	u, err := f.client.GetUser(ctx, &authv1.GetUserRequest{UserId: userID})
	if err != nil {
		return nil, err
	}
	return &auth.Principal{
		ID:            u.Id,
		Email:         u.Email,
		Username:      u.Username,
		DisplayName:   u.DisplayName,
		Role:          u.Role,
		Status:        u.Status,
		EmailVerified: u.EmailVerified,
		CreatedAt:     time.Unix(u.CreatedAtUnix, 0).UTC(),
	}, nil
}
