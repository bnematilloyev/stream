package handler

import (
	"encoding/json"
	"io"

	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func decodeError(err error) error {
	if err == io.EOF {
		return apperrors.Validation("request body is required", nil)
	}
	if _, ok := err.(*json.SyntaxError); ok {
		return apperrors.Validation("invalid JSON body", nil)
	}
	return apperrors.Validation("invalid request body", nil)
}

func validationError(message string) error {
	return apperrors.Validation(message, nil)
}

func grpcError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return apperrors.Internal(err)
	}

	switch st.Code() {
	case codes.InvalidArgument:
		return apperrors.Validation(st.Message(), nil)
	case codes.Unauthenticated:
		return apperrors.Unauthorized(st.Message())
	case codes.PermissionDenied:
		return apperrors.Forbidden(st.Message())
	case codes.NotFound:
		return apperrors.NotFound(st.Message())
	case codes.AlreadyExists:
		return apperrors.Conflict(apperrors.CodeConflict, st.Message())
	case codes.ResourceExhausted:
		return apperrors.RateLimited()
	case codes.Unavailable:
		return apperrors.ServiceUnavailable(st.Message())
	default:
		return apperrors.Internal(err)
	}
}
