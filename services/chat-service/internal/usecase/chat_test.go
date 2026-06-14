package usecase_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/domain"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/usecase"
)

type stubRepo struct {
	deleteErr error
}

func (s stubRepo) Insert(context.Context, *domain.Message) error { return nil }
func (s stubRepo) ListHistory(context.Context, uuid.UUID, int64, int) ([]domain.Message, error) {
	return nil, nil
}
func (s stubRepo) Delete(context.Context, uuid.UUID, int64) error { return s.deleteErr }
func (s stubRepo) IsBanned(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}

type stubModerator struct{ ok bool }

func (s stubModerator) CanModerate(context.Context, uuid.UUID, string, string) (bool, error) {
	return s.ok, nil
}

func TestDeleteMessageNotFound(t *testing.T) {
	uc := usecase.NewChatUseCase(
		stubRepo{deleteErr: domain.ErrNotFound},
		nil,
		stubModerator{ok: true},
		nil,
		nil,
	)
	err := uc.DeleteMessage(context.Background(), uuid.New(), 1, uuid.New().String(), "user")
	appErr, ok := apperrors.IsAppError(err)
	if !ok || appErr.HTTPStatus != 404 {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestDeleteMessageForbidden(t *testing.T) {
	uc := usecase.NewChatUseCase(stubRepo{}, nil, stubModerator{ok: false}, nil, nil)
	err := uc.DeleteMessage(context.Background(), uuid.New(), 1, uuid.New().String(), "user")
	appErr, ok := apperrors.IsAppError(err)
	if !ok || appErr.HTTPStatus != 403 {
		t.Fatalf("expected forbidden, got %v", err)
	}
}
