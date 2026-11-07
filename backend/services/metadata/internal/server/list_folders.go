package server

import (
	"context"
	"encoding/base64"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	metadatav1 "github.com/Stephnn0/stephanos-distributed-drive/backend/gen/drive/metadata/v1"
)

const (
	defaultPageSize = 50
	maxPageSize     = 1000
)

// ListFolders lists the folders directly inside a parent, one page at a time.
func (s *Server) ListFolders(ctx context.Context, req *metadatav1.ListFoldersRequest) (*metadatav1.ListFoldersResponse, error) {
	// 1. Check the request.
	owner, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	pageSize := int(req.GetPageSize())
	switch {
	case pageSize < 0:
		return nil, status.Error(codes.InvalidArgument, "page_size must not be negative")
	case pageSize == 0:
		pageSize = defaultPageSize
	case pageSize > maxPageSize:
		pageSize = maxPageSize
	}
	afterID, err := decodePageToken(req.GetPageToken())
	if err != nil {
		return nil, err
	}
	parentID, err := s.folderID(ctx, owner, "parent", req.GetParent())
	if err != nil {
		return nil, err
	}

	// 2. Call the store.
	folders, more, err := s.store.ListFolders(ctx, owner, parentID, afterID, pageSize)
	if err != nil {
		return nil, toStatus(err)
	}

	// 3. Convert the result.
	resp := &metadatav1.ListFoldersResponse{}
	for _, f := range folders {
		resp.Folders = append(resp.Folders, toProto(f))
	}
	if more {
		resp.NextPageToken = encodePageToken(folders[len(folders)-1].ID)
	}
	return resp, nil
}

// A page token is the ID of the last folder on the previous page, base64
// encoded so callers treat it as an opaque string.

func encodePageToken(lastID string) string {
	u := uuid.MustParse(lastID) // IDs come from the database, so always valid
	return base64.RawURLEncoding.EncodeToString(u[:])
}

// decodePageToken returns the ID to continue after. An empty token means
// the first page, which starts after the all-zero UUID (smaller than any ID).
func decodePageToken(token string) (string, error) {
	if token == "" {
		return uuid.Nil.String(), nil
	}
	b, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", status.Error(codes.InvalidArgument, "invalid page_token")
	}
	u, err := uuid.FromBytes(b)
	if err != nil {
		return "", status.Error(codes.InvalidArgument, "invalid page_token")
	}
	return u.String(), nil
}
