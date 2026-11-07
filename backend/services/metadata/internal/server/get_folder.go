package server

import (
	"context"

	metadatav1 "github.com/Stephnn0/stephanos-distributed-drive/backend/gen/drive/metadata/v1"
)

// GetFolder returns one folder.
func (s *Server) GetFolder(ctx context.Context, req *metadatav1.GetFolderRequest) (*metadatav1.Folder, error) {
	// 1. Check the request.
	owner, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := s.folderID(ctx, owner, "name", req.GetName())
	if err != nil {
		return nil, err
	}

	// 2. Call the store.
	f, err := s.store.GetFolder(ctx, owner, id)
	if err != nil {
		return nil, toStatus(err)
	}

	// 3. Convert the result.
	return toProto(f), nil
}
