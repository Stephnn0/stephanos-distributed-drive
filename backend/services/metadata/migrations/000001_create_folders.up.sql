CREATE TABLE folders (
    -- UUIDv7 is time-ordered, so new rows go at the end of the index
    -- instead of random pages (random UUIDv4 slows inserts at scale).
    id           UUID PRIMARY KEY DEFAULT uuidv7(),

    -- Every query is scoped to a user; this is also the future shard key.
    owner_id     TEXT NOT NULL,

    -- NULL = the user's root folder. No ON DELETE CASCADE: deleting a huge
    -- tree in one transaction would lock it for a long time. Deletes are
    -- soft (delete_time) and a background job cleans up later.
    parent_id    UUID REFERENCES folders(id),

    display_name TEXT NOT NULL CHECK (length(display_name) BETWEEN 1 AND 255),

    create_time  TIMESTAMPTZ NOT NULL DEFAULT now(),
    update_time  TIMESTAMPTZ NOT NULL DEFAULT now(),
    delete_time  TIMESTAMPTZ -- NULL = live; set = in trash
);

-- "List what's inside this folder", paginated by id (keyset pagination).
-- Partial index: trashed rows are skipped, which keeps the index small.
CREATE INDEX folders_parent_id_idx ON folders (parent_id, id) WHERE delete_time IS NULL;

-- Each user has exactly one root folder.
CREATE UNIQUE INDEX folders_one_root_per_owner ON folders (owner_id) WHERE parent_id IS NULL;
