package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Every query filters by owner_id, so a user can only ever touch their own
// folders. Trashed folders (delete_time set) are treated as not found.

// RootFolderID returns the ID of the user's root folder.
func (s *Store) RootFolderID(ctx context.Context, owner string) (string, error) {
	var id string
	err := s.db.QueryRow(ctx,
		`SELECT id::text FROM folders WHERE owner_id = $1 AND parent_id IS NULL AND delete_time IS NULL`,
		owner).Scan(&id)
	return id, notFound(err, "root folder")
}

// CreateRootFolder creates the user's root folder. Each user has one.
func (s *Store) CreateRootFolder(ctx context.Context, owner, displayName string) (*Folder, error) {
	row := s.db.QueryRow(ctx,
		`INSERT INTO folders (owner_id, display_name) VALUES ($1, $2) RETURNING `+folderColumns,
		owner, displayName)
	f, err := scanFolder(row)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == "folders_one_root_per_owner" {
		return nil, ErrRootExists
	}
	return f, err
}

// CreateFolder creates a folder inside parentID.
func (s *Store) CreateFolder(ctx context.Context, owner, parentID, displayName string) (*Folder, error) {
	// Inserts only if the parent exists, belongs to the user and isn't in
	// the trash. Otherwise no row comes back.
	row := s.db.QueryRow(ctx, `
		INSERT INTO folders (owner_id, parent_id, display_name)
		SELECT $1, id, $3 FROM folders
		WHERE id = $2 AND owner_id = $1 AND delete_time IS NULL
		RETURNING `+folderColumns,
		owner, parentID, displayName)
	f, err := scanFolder(row)
	return f, notFound(err, "parent folder")
}

// GetFolder returns one folder.
func (s *Store) GetFolder(ctx context.Context, owner, id string) (*Folder, error) {
	row := s.db.QueryRow(ctx,
		`SELECT `+folderColumns+` FROM folders WHERE id = $1 AND owner_id = $2 AND delete_time IS NULL`,
		id, owner)
	f, err := scanFolder(row)
	return f, notFound(err, "folder")
}

// ListFolders returns up to limit folders inside parentID with an ID
// greater than afterID, oldest first. more is true if there are more after
// them.
//
// This is keyset pagination: the next page continues from the last ID
// returned. Unlike OFFSET it stays fast however deep you page, because it
// walks the (parent_id, id) index directly.
func (s *Store) ListFolders(ctx context.Context, owner, parentID, afterID string, limit int) (folders []*Folder, more bool, err error) {
	var exists bool
	err = s.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM folders WHERE id = $1 AND owner_id = $2 AND delete_time IS NULL)`,
		parentID, owner).Scan(&exists)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, fmt.Errorf("parent folder %w", ErrNotFound)
	}

	// Ask for one extra row: if it comes back, there is another page.
	rows, err := s.db.Query(ctx, `
		SELECT `+folderColumns+` FROM folders
		WHERE parent_id = $1 AND owner_id = $2 AND delete_time IS NULL AND id > $3
		ORDER BY id
		LIMIT $4`,
		parentID, owner, afterID, limit+1)
	if err != nil {
		return nil, false, err
	}
	folders, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Folder, error) {
		return scanFolder(row)
	})
	if err != nil {
		return nil, false, err
	}
	if len(folders) > limit {
		return folders[:limit], true, nil
	}
	return folders, false, nil
}

// UpdateFolder renames and/or moves a folder. A nil newName or newParentID
// leaves that field unchanged.
func (s *Store) UpdateFolder(ctx context.Context, owner, id string, newName, newParentID *string) (*Folder, error) {
	// A transaction, so the move check and the update see the same data.
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) // does nothing once committed

	if newParentID != nil {
		if err := checkMove(ctx, tx, owner, id, *newParentID); err != nil {
			return nil, err
		}
	}
	row := tx.QueryRow(ctx, `
		UPDATE folders SET
			display_name = COALESCE($3, display_name),
			parent_id    = COALESCE($4::uuid, parent_id),
			update_time  = now()
		WHERE id = $1 AND owner_id = $2 AND delete_time IS NULL
		RETURNING `+folderColumns,
		id, owner, newName, newParentID)
	f, err := scanFolder(row)
	if err != nil {
		return nil, notFound(err, "folder")
	}
	return f, tx.Commit(ctx)
}

// checkMove rejects moves that would break the folder tree: moving the
// root, moving into a missing or trashed folder, or moving a folder into
// itself or one of its own subfolders (which would create a loop).
//
// TODO(scale): two moves at the same time (A into B, B into A) can each
// pass this check and create a loop together. Fix by locking the rows
// involved, or with a SERIALIZABLE transaction and retries.
func checkMove(ctx context.Context, q querier, owner, id, newParentID string) error {
	var isRoot, parentExists, wouldLoop bool
	// ancestors walks up from the new parent to the root. If the folder
	// being moved is on that path, the new parent is inside it.
	err := q.QueryRow(ctx, `
		WITH RECURSIVE ancestors AS (
			SELECT id, parent_id FROM folders WHERE id = $1
			UNION ALL
			SELECT f.id, f.parent_id FROM folders f JOIN ancestors a ON f.id = a.parent_id
		)
		SELECT
			EXISTS (SELECT 1 FROM folders WHERE id = $2 AND owner_id = $3 AND parent_id IS NULL),
			EXISTS (SELECT 1 FROM folders WHERE id = $1 AND owner_id = $3 AND delete_time IS NULL),
			EXISTS (SELECT 1 FROM ancestors WHERE id = $2)`,
		newParentID, id, owner).Scan(&isRoot, &parentExists, &wouldLoop)
	switch {
	case err != nil:
		return err
	case isRoot:
		return ErrRootFolder
	case !parentExists:
		return fmt.Errorf("new parent folder %w", ErrNotFound)
	case wouldLoop:
		return ErrInvalidMove
	}
	return nil
}

// TrashFolder moves a folder to the trash (soft delete) and returns it.
//
// Only the folder itself is marked. Its subfolders can't be reached by
// listing anymore, since their parent is hidden. A background job will
// purge trashed trees later.
func (s *Store) TrashFolder(ctx context.Context, owner, id string) (*Folder, error) {
	row := s.db.QueryRow(ctx, `
		UPDATE folders SET delete_time = now(), update_time = now()
		WHERE id = $1 AND owner_id = $2 AND delete_time IS NULL AND parent_id IS NOT NULL
		RETURNING `+folderColumns,
		id, owner)
	f, err := scanFolder(row)
	if !errors.Is(err, pgx.ErrNoRows) {
		return f, err
	}
	// Nothing was updated: either the folder doesn't exist, or it's the root.
	var isRoot bool
	err = s.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM folders WHERE id = $1 AND owner_id = $2 AND parent_id IS NULL)`,
		id, owner).Scan(&isRoot)
	switch {
	case err != nil:
		return nil, err
	case isRoot:
		return nil, ErrRootFolder
	}
	return nil, fmt.Errorf("folder %w", ErrNotFound)
}

// notFound turns "no rows" into ErrNotFound, naming what was missing, e.g.
// "parent folder not found". Other errors pass through unchanged.
func notFound(err error, what string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s %w", what, ErrNotFound)
	}
	return err
}
