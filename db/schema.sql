-- One row per indexed file, independent of how many chunks it currently
-- has (including zero, e.g. an empty file) - this is what lets sync detect
-- unchanged files without touching chunks, and detect deleted files by
-- diffing this table's paths against a fresh directory walk.
CREATE TABLE IF NOT EXISTS files (
    id INTEGER PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    modified_at INTEGER NOT NULL,
    -- Whole-file content hash (sha256, 32 bytes). Catches content changes
    -- even if a filesystem doesn't reliably update modified_at.
    file_hash BLOB NOT NULL,
    status TEXT NOT NULL DEFAULT 'ok',
    last_error TEXT
);

CREATE TABLE IF NOT EXISTS chunks (
    id INTEGER PRIMARY KEY,
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,
    content TEXT NOT NULL,
    -- Per-chunk content hash (sha256, 32 bytes), used to skip re-embedding
    -- chunks whose content hasn't changed across a reindex.
    content_hash BLOB NOT NULL,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    -- Identifies which embedding model produced this chunk's vector, so a
    -- future model swap can detect and re-embed stale vectors instead of
    -- silently mixing incomparable vectors in search.
    embedding_model TEXT NOT NULL,
    UNIQUE(file_id, chunk_index)
);

-- The actual embedding vectors live here, not on chunks, keyed by the same
-- rowid as chunks.id (application-assigned on insert). Note this rowid
-- link is not a real foreign key - sqlite-vec's vec0 virtual table doesn't
-- support FK constraints, so deleting a chunks row does not automatically
-- delete its vec_chunks row; that has to be done explicitly together.
--
-- float[384] matches the current embedding model's output dimension. A
-- future model with a different dimension (e.g. a CLIP variant) would need
-- its own vec0 table, since a vec0 table's vector width is fixed.
CREATE VIRTUAL TABLE IF NOT EXISTS vec_chunks USING vec0(
    embedding float[384]
);
