package store

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Folder is one row of the folders table.
type Folder struct {
	ID          string // UUID
	ParentID    string // "" for the user's root folder
	DisplayName string
	OwnerID     string
	CreateTime  time.Time
	UpdateTime  time.Time
	DeleteTime  *time.Time // nil unless the folder is in the trash
}

// Errors the store returns. Callers check them with errors.Is. Their
// messages are safe to show to API users.
var (
	ErrNotFound    = errors.New("not found")
	ErrRootExists  = errors.New("you already have a root folder")
	ErrRootFolder  = errors.New("the root folder can't be moved or deleted")
	ErrInvalidMove = errors.New("a folder can't be moved into itself or one of its own subfolders")
)

// folderColumns is the column list every folder query selects, in the
// order scanFolder reads them.
const folderColumns = `id::text, parent_id::text, display_name, owner_id, create_time, update_time, delete_time`

// scanFolder reads one row selected with folderColumns.
func scanFolder(row pgx.Row) (*Folder, error) {
	var f Folder
	var parentID *string // NULL for the root folder
	err := row.Scan(&f.ID, &parentID, &f.DisplayName, &f.OwnerID, &f.CreateTime, &f.UpdateTime, &f.DeleteTime)
	if err != nil {
		return nil, err
	}
	if parentID != nil {
		f.ParentID = *parentID
	}
	return &f, nil
}
