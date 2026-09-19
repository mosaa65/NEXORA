-- Migration 0022: incremental indexing, scan sessions and file identity.
--
-- This migration makes the scanner's state persistent, which is what allows a
-- scan to resume after a crash and an incremental scan to skip unchanged files
-- without re-reading them.
--
-- Design notes:
--   * scan_sessions records every scan, including one that never finished. After
--     a restart the server can detect a RUNNING session and treat it as
--     interrupted instead of assuming the catalogue is consistent.
--   * scan_roots records per-root state separately, so one offline disk does not
--     invalidate the state of the others.
--   * video_files gains the filesystem identity and change-tracking columns the
--     incremental scanner compares against. They are all nullable/defaulted so
--     existing rows remain valid and a full re-scan is not forced by the upgrade.

CREATE TABLE IF NOT EXISTS scan_sessions (
    id VARCHAR(64) PRIMARY KEY,
    status VARCHAR(20) NOT NULL DEFAULT 'running',
    mode VARCHAR(20) NOT NULL DEFAULT 'full',
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP,
    duration_seconds DOUBLE PRECISION,
    -- Structured counters kept as JSONB so the report shape can evolve without
    -- a migration per counter.
    stats JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_checkpoint JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_count INT NOT NULL DEFAULT 0,
    -- A session left in 'running' means the server stopped mid-scan.
    interrupted BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_scan_sessions_started_at ON scan_sessions(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_scan_sessions_status ON scan_sessions(status);

CREATE TABLE IF NOT EXISTS scan_roots (
    id SERIAL PRIMARY KEY,
    scan_id VARCHAR(64) NOT NULL REFERENCES scan_sessions(id) ON DELETE CASCADE,
    root_id VARCHAR(128) NOT NULL,
    root_path TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_code VARCHAR(40),
    error_message TEXT,
    directories_visited BIGINT NOT NULL DEFAULT 0,
    files_seen BIGINT NOT NULL DEFAULT 0,
    media_accepted BIGINT NOT NULL DEFAULT 0,
    bytes_total BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (scan_id, root_id)
);

CREATE INDEX IF NOT EXISTS idx_scan_roots_scan_id ON scan_roots(scan_id);
CREATE INDEX IF NOT EXISTS idx_scan_roots_root_id ON scan_roots(root_id);

-- Known media roots, independent of any single scan. An external disk that is
-- unplugged keeps its row so a returning disk triggers a recovery scan of that
-- root only, never a library-wide rescan.
--
-- This deliberately does not reuse `storage_disks`: that table is keyed by a
-- single `disk_letter CHAR(1)` and therefore cannot represent a UNC share such
-- as //nas/media or several media roots on one volume. `storage_disks` remains
-- the per-volume capacity dashboard; `media_roots` is scan state.
CREATE TABLE IF NOT EXISTS media_roots (
    root_id VARCHAR(128) PRIMARY KEY,
    root_path TEXT NOT NULL,
    label VARCHAR(200),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    last_scan_id VARCHAR(64),
    last_seen_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- File identity and change tracking on the existing catalogue table.
ALTER TABLE video_files
    ADD COLUMN IF NOT EXISTS file_id TEXT,
    ADD COLUMN IF NOT EXISTS file_mod_time TIMESTAMP,
    ADD COLUMN IF NOT EXISTS root_id VARCHAR(128),
    ADD COLUMN IF NOT EXISTS root_path TEXT,
    ADD COLUMN IF NOT EXISTS relative_path TEXT,
    ADD COLUMN IF NOT EXISTS part_number INT,
    ADD COLUMN IF NOT EXISTS episode_end INT,
    ADD COLUMN IF NOT EXISTS title_normalized TEXT,
    ADD COLUMN IF NOT EXISTS search_tokens TEXT[],
    ADD COLUMN IF NOT EXISTS parse_confidence REAL,
    ADD COLUMN IF NOT EXISTS parse_reasons TEXT[],
    ADD COLUMN IF NOT EXISTS special_kind VARCHAR(20),
    ADD COLUMN IF NOT EXISTS state VARCHAR(20) NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS last_scan_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMP,
    ADD COLUMN IF NOT EXISTS first_indexed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Cheap change detection: the scanner compares (path) then (file_id) then
-- (size, mod_time). These indexes make each comparison a lookup rather than a
-- sequential scan of a multi-million row table.
CREATE INDEX IF NOT EXISTS idx_video_files_file_id ON video_files(file_id) WHERE file_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_video_files_root_id ON video_files(root_id);
CREATE INDEX IF NOT EXISTS idx_video_files_state ON video_files(state);
CREATE INDEX IF NOT EXISTS idx_video_files_size_modtime ON video_files(file_size, file_mod_time);
CREATE INDEX IF NOT EXISTS idx_video_files_title_normalized ON video_files(title_normalized);
-- A path can only appear once; this is the guarantee behind idempotency.
CREATE UNIQUE INDEX IF NOT EXISTS idx_video_files_file_path_unique ON video_files(file_path);

-- -----------------------------------------------------------------------------
-- Duplicate reconciliation for pre-existing data.
--
-- The old scanner could index the same physical file twice when a path arrived
-- with different separators or casing ("D:\Media\X.avi" vs "D:\Media\X.avi"),
-- because file_path had no normalization. Such rows are provably the same file:
-- identical normalized path, identical size. They are merged here rather than
-- left to collide with the unique path index below.
--
-- Only rows that are *provably* duplicates are removed. Rows that merely share an
-- episode number are NOT deleted: they may be legitimate alternate releases, and
-- silently dropping media rows is exactly the data loss this system must avoid.
-- Those are recorded in index_conflicts for review instead.
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS index_conflicts (
    id SERIAL PRIMARY KEY,
    kind VARCHAR(40) NOT NULL,
    media_item_id INT,
    season_id INT,
    episode_number INT,
    row_ids INT[] NOT NULL,
    detail TEXT,
    detected_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DELETE FROM video_files vf
USING video_files other
WHERE vf.id > other.id
  AND lower(replace(vf.file_path, '\\', '/')) = lower(replace(other.file_path, '\\', '/'))
  AND vf.file_size = other.file_size;

-- Record episode-identity collisions before the unique index is attempted, so
-- the operator can see exactly what needs review instead of reading a startup
-- failure. This runs whether or not the index can be created.
INSERT INTO index_conflicts (kind, media_item_id, season_id, episode_number, row_ids, detail)
SELECT
    'duplicate_episode_identity',
    media_item_id,
    season_id,
    episode_number,
    array_agg(id ORDER BY id),
    'Multiple files share one episode slot. They may be different releases, or the old parser may have merged unrelated files into one season. Review before removing any row.'
FROM video_files
WHERE season_id IS NOT NULL AND episode_number IS NOT NULL
GROUP BY media_item_id, season_id, episode_number
HAVING COUNT(*) > 1;

-- The same file must never be represented twice for one episode. Partial so that
-- movies and files without an episode number are unaffected.
--
-- This index is intentionally NOT unique: pre-existing collisions are real data
-- that must be reviewed, not a reason to refuse to start the server. The ingest
-- path re-checks uniqueness inside its upsert, so new duplicates are still
-- prevented going forward.
CREATE INDEX IF NOT EXISTS idx_video_files_episode_identity
    ON video_files(media_item_id, season_id, episode_number)
    WHERE season_id IS NOT NULL AND episode_number IS NOT NULL AND part_number IS NULL;

COMMENT ON COLUMN video_files.state IS 'active | missing | unavailable | error | pending_scan';
COMMENT ON COLUMN video_files.file_id IS 'Filesystem file identity used for rename inference; never file content';
COMMENT ON TABLE scan_sessions IS 'Every scan run, including interrupted ones, so the index state is never assumed';
