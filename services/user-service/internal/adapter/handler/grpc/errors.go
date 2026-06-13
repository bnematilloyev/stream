package grpc

import (
	"net/http"

	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toGRPCError(err error) error {
	appErr, ok := apperrors.IsAppError(err)
	if !ok {
		return status.Error(codes.Internal, "internal server error")
	}
	switch appErr.HTTPStatus {
	case http.StatusBadRequest:
		return status.Error(codes.InvalidArgument, appErr.Message)
	case http.StatusUnauthorized:
		return status.Error(codes.Unauthenticated, appErr.Message)
	case http.StatusForbidden:
		return status.Error(codes.PermissionDenied, appErr.Message)
	case http.StatusNotFound:
		return status.Error(codes.NotFound, appErr.Message)
	case http.StatusConflict:
		return status.Error(codes.AlreadyExists, appErr.Message)
	default:
		return status.Error(codes.Internal, appErr.Message)
	}
}
