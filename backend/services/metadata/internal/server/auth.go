package server

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// callerID returns the ID of the user making the call, read from the
// "x-user-id" gRPC header.
//
// TODO(auth): replace with the user ID from a verified auth token, set by
// a shared interceptor. Today any caller can claim to be any user.
func callerID(ctx context.Context) (string, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if ids := md.Get("x-user-id"); len(ids) > 0 && ids[0] != "" {
		return ids[0], nil
	}
	return "", status.Error(codes.Unauthenticated, "missing x-user-id header")
}
