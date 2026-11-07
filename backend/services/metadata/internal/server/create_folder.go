package server

import (
	"context"

	metadatav1 "github.com/Stephnn0/stephanos-distributed-drive/backend/gen/drive/metadata/v1"
	"github.com/Stephnn0/stephanos-distributed-drive/backend/services/metadata/internal/store"
)

// CreateFolder creates a folder, or the caller's root folder if parent is empty.
func (s *Server) CreateFolder(ctx context.Context, req *metadatav1.CreateFolderRequest) (*metadatav1.Folder, error) {
	// 1. Check the request.
	owner, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	name := req.GetFolder().GetDisplayName()
	if err := validateDisplayName(name); err != nil {
		return nil, err
	}

	// 2. Call the store.
	var f *store.Folder
	if req.GetParent() == "" {
		f, err = s.store.CreateRootFolder(ctx, owner, name)
	} else {
		var parentID string
		if parentID, err = s.folderID(ctx, owner, "parent", req.GetParent()); err != nil {
			return nil, err
		}
		f, err = s.store.CreateFolder(ctx, owner, parentID, name)
	}
	if err != nil {
		return nil, toStatus(err)
	}

	// 3. Convert the result.
	return toProto(f), nil
}
