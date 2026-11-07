package server

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	metadatav1 "github.com/Stephnn0/stephanos-distributed-drive/backend/gen/drive/metadata/v1"
	"github.com/Stephnn0/stephanos-distributed-drive/backend/services/metadata/internal/store"
)

// The API names folders "folders/{id}". "folders/root" means the caller's
// root folder.
const (
	folderPrefix = "folders/"
	rootAlias    = "root"
)

// toProto converts a folder from the store into its API form.
func toProto(f *store.Folder) *metadatav1.Folder {
	out := &metadatav1.Folder{
		Name:        folderPrefix + f.ID,
		DisplayName: f.DisplayName,
		OwnerId:     f.OwnerID,
		CreateTime:  timestamppb.New(f.CreateTime),
		UpdateTime:  timestamppb.New(f.UpdateTime),
	}
	if f.ParentID != "" {
		out.Parent = folderPrefix + f.ParentID
	}
	if f.DeleteTime != nil {
		out.DeleteTime = timestamppb.New(*f.DeleteTime)
	}
	return out
}

// folderID turns a name like "folders/{uuid}" into the folder's ID, and
// "folders/root" into the ID of the caller's root folder. field is the
// request field being read, for error messages.
func (s *Server) folderID(ctx context.Context, owner, field, name string) (string, error) {
	id, ok := strings.CutPrefix(name, folderPrefix)
	if !ok {
		return "", status.Errorf(codes.InvalidArgument, "%s must look like folders/{folder_id}, got %q", field, name)
	}
	if id == rootAlias {
		rootID, err := s.store.RootFolderID(ctx, owner)
		if err != nil {
			return "", toStatus(err)
		}
		return rootID, nil
	}
	u, err := uuid.Parse(id)
	if err != nil {
		return "", status.Errorf(codes.InvalidArgument, "%s has an invalid folder ID %q", field, id)
	}
	return u.String(), nil
}

// validateDisplayName checks a folder name is 1 to 255 characters, the
// same rule the database enforces.
func validateDisplayName(name string) error {
	if strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > 255 {
		return status.Error(codes.InvalidArgument, "display_name must be 1 to 255 characters")
	}
	return nil
}
