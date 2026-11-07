package server

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Stephnn0/stephanos-distributed-drive/backend/services/metadata/internal/store"
)

// toStatus turns an error from the store into a gRPC error with the right
// code. Unexpected errors are logged, and the caller only sees "internal
// error", so database details never leak out of the service.
func toStatus(err error) error {
	if _, ok := status.FromError(err); ok {
		return err // already a gRPC error
	}
	switch {
	case errors.Is(err, store.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, store.ErrRootExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, store.ErrRootFolder):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, store.ErrInvalidMove):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	}
	slog.Error("unexpected error", "error", err)
	return status.Error(codes.Internal, "internal error")
}
