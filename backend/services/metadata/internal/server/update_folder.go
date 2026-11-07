package server

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	metadatav1 "github.com/Stephnn0/stephanos-distributed-drive/backend/gen/drive/metadata/v1"
)

// UpdateFolder renames and/or moves a folder.
func (s *Server) UpdateFolder(ctx context.Context, req *metadatav1.UpdateFolderRequest) (*metadatav1.Folder, error) {
	// 1. Check the request.
	owner, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	in := req.GetFolder()
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "folder is required")
	}
	id, err := s.folderID(ctx, owner, "folder.name", in.GetName())
	if err != nil {
		return nil, err
	}

	// update_mask lists the fields to change. With no mask, change every
	// supported field that was set.
	paths := req.GetUpdateMask().GetPaths()
	if len(paths) == 0 {
		if in.GetDisplayName() != "" {
			paths = append(paths, "display_name")
		}
		if in.GetParent() != "" {
			paths = append(paths, "parent")
		}
	}
	if len(paths) == 0 {
		return nil, status.Error(codes.InvalidArgument, "nothing to update: set display_name and/or parent")
	}

	// nil means "leave unchanged".
	var newName, newParentID *string
	for _, p := range paths {
		switch p {
		case "display_name":
			if err := validateDisplayName(in.GetDisplayName()); err != nil {
				return nil, err
			}
			newName = &in.DisplayName
		case "parent":
			parentID, err := s.folderID(ctx, owner, "folder.parent", in.GetParent())
			if err != nil {
				return nil, err
			}
			newParentID = &parentID
		default:
			return nil, status.Errorf(codes.InvalidArgument, "update_mask path %q is not supported", p)
		}
	}

	// 2. Call the store.
	f, err := s.store.UpdateFolder(ctx, owner, id, newName, newParentID)
	if err != nil {
		return nil, toStatus(err)
	}

	// 3. Convert the result.
	return toProto(f), nil
}
