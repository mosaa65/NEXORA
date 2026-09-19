-- Migration 0023: separate the logical media model from physical files.
--
-- The core correction this migration enables: a physical file is NOT a work.
-- Until now the ingest path derived a `media_items` row directly from a parsed
-- filename, which produced degenerate "works" for real data such as
-- "الكنز ج1 الحلقه", "Fate_Apocrypha", "الكنزنت" and "منور فديوهات زابيا",
-- and filed 12 One Piece episodes under type = 'movie'.
--
-- After this migration the pipeline is:
--
--   file -> parse -> candidate -> ENTITY RESOLUTION -> work / season / episode -> file
--
-- Everything here is additive with defaults, so existing rows stay valid. No
-- destructive change is made to media_items or video_files.

-- -----------------------------------------------------------------------------
-- 1. Value provenance
-- -----------------------------------------------------------------------------
-- Every important value must know where it came from, so a Full Scan can never
-- silently overwrite an administrator's correction with a raw parser guess.
--
-- An enum keeps the precedence order enforceable and auditable.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'value_source') THEN
        CREATE TYPE value_source AS ENUM (
            'filesystem',   -- facts read from the disk: path, size, mtime, identity
            'parser',       -- derived from the filename/path heuristics
            'resolver',     -- chosen by entity resolution
            'database',     -- carried over from an existing record
            'admin',        -- an operator decision; always wins
            'tmdb'          -- provider metadata
        );
    END IF;
END $$;

-- -----------------------------------------------------------------------------
-- 2. Resolution lifecycle state
-- -----------------------------------------------------------------------------
-- "indexed / not indexed" is not a sufficient status. These states describe how
-- far a file has travelled through the pipeline and what a human must still do.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'resolution_state') THEN
        CREATE TYPE resolution_state AS ENUM (
            'discovered',           -- seen on disk, not yet parsed
            'parsed',               -- metadata extracted
            'resolved',             -- attached to a work/season/episode with confidence
            'unresolved',           -- no work could be determined
            'needs_review',         -- ambiguous; queued for an operator
            'enrichment_pending',   -- resolved but awaiting provider metadata
            'enriched',             -- provider metadata applied
            'indexed',              -- projected into the search index
            'error'                 -- deterministic failure
        );
    END IF;
END $$;

-- -----------------------------------------------------------------------------
-- 3. Work aliases: the rule-based library memory
-- -----------------------------------------------------------------------------
-- This table is what lets "Breaking Bad", "Breaking.Bad", "Breaking_Bad",
-- "بريكنغ باد" and "Breaking Bad Arabic" resolve to ONE work without any AI.
-- Aliases are learned from successful resolutions and from operator decisions.
CREATE TABLE IF NOT EXISTS media_aliases (
    id SERIAL PRIMARY KEY,
    media_item_id INT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    alias TEXT NOT NULL,
    -- normalized form, computed by the resolver's normalizer, never by SQL
    alias_normalized TEXT NOT NULL,
    -- where this alias came from, so a learned alias can be distinguished from
    -- an operator-provided one
    source value_source NOT NULL DEFAULT 'resolver',
    -- how many times this alias successfully resolved a file; a learned alias
    -- that never matches again is harmless but visible
    hit_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (media_item_id, alias_normalized)
);

-- One alias string maps to at most one work. This is the constraint that makes
-- alias lookup deterministic; a conflict means a human must decide.
CREATE UNIQUE INDEX IF NOT EXISTS idx_media_aliases_normalized
    ON media_aliases(alias_normalized);
CREATE INDEX IF NOT EXISTS idx_media_aliases_media_item_id ON media_aliases(media_item_id);

-- -----------------------------------------------------------------------------
-- 4. Resolution queue: where ambiguity goes instead of becoming a bad Work
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS resolution_queue (
    id SERIAL PRIMARY KEY,
    -- the file awaiting a decision
    file_path TEXT NOT NULL,
    relative_path TEXT,
    root_id VARCHAR(128),
    original_filename TEXT,
    size BIGINT,
    -- what the parser thought
    detected_title TEXT,
    detected_title_normalized TEXT,
    detected_category VARCHAR(60),
    detected_media_type VARCHAR(20),
    detected_season INT,
    detected_episode INT,
    detected_year INT,
    parser_confidence REAL,
    resolver_confidence REAL,
    -- why it could not be resolved
    reason VARCHAR(60) NOT NULL,
    reason_detail TEXT,
    -- ranked candidates with their scores, so the operator sees the reasoning
    candidates JSONB NOT NULL DEFAULT '[]'::jsonb,
    state resolution_state NOT NULL DEFAULT 'needs_review',
    -- operator decision, when made
    decision VARCHAR(40),
    decided_media_item_id INT REFERENCES media_items(id) ON DELETE SET NULL,
    decided_season_number INT,
    decided_episode_number INT,
    decided_by VARCHAR(100),
    decided_at TIMESTAMP,
    -- whether the decision was promoted into a durable alias
    learned_alias BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (file_path)
);

CREATE INDEX IF NOT EXISTS idx_resolution_queue_state ON resolution_queue(state);
CREATE INDEX IF NOT EXISTS idx_resolution_queue_reason ON resolution_queue(reason);
CREATE INDEX IF NOT EXISTS idx_resolution_queue_created_at ON resolution_queue(created_at DESC);

-- -----------------------------------------------------------------------------
-- 5. Canonical work metadata and provenance columns
-- -----------------------------------------------------------------------------
ALTER TABLE media_items
    ADD COLUMN IF NOT EXISTS title_source value_source DEFAULT 'resolver',
    ADD COLUMN IF NOT EXISTS resolution_state resolution_state DEFAULT 'resolved',
    ADD COLUMN IF NOT EXISTS resolver_confidence REAL,
    ADD COLUMN IF NOT EXISTS parser_confidence REAL,
    ADD COLUMN IF NOT EXISTS tmdb_confidence REAL,
    ADD COLUMN IF NOT EXISTS raw_detected_title TEXT,
    -- normalized canonical title, used for identity comparison
    ADD COLUMN IF NOT EXISTS title_normalized TEXT,
    -- true when an operator has edited this work; the scanner must not revert it
    ADD COLUMN IF NOT EXISTS metadata_locked BOOLEAN NOT NULL DEFAULT FALSE,
    -- a provisional entity is one created by the resolver before enrichment;
    -- it may be merged into a provider-backed entity later
    ADD COLUMN IF NOT EXISTS provisional BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS merged_into_id INT REFERENCES media_items(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_media_items_title_normalized ON media_items(title_normalized);
CREATE INDEX IF NOT EXISTS idx_media_items_resolution_state ON media_items(resolution_state);
CREATE INDEX IF NOT EXISTS idx_media_items_provisional ON media_items(provisional) WHERE provisional = TRUE;

-- -----------------------------------------------------------------------------
-- 6. Episode identity
-- -----------------------------------------------------------------------------
-- A season can hold provider-backed episodes even before a file exists for them,
-- which is what makes NEXORA Managed Mode possible: the operator creates the
-- episode, and the watcher later attaches the file to it.
CREATE TABLE IF NOT EXISTS episodes (
    id SERIAL PRIMARY KEY,
    season_id INT NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    episode_number INT NOT NULL,
    title_ar VARCHAR(255),
    title_en VARCHAR(255),
    overview_ar TEXT,
    overview_en TEXT,
    air_date DATE,
    runtime INT,
    still_path VARCHAR(500),
    provider VARCHAR(30),
    external_id VARCHAR(40),
    metadata_payload JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (season_id, episode_number)
);

CREATE INDEX IF NOT EXISTS idx_episodes_season_id ON episodes(season_id);
CREATE INDEX IF NOT EXISTS idx_episodes_external_id ON episodes(provider, external_id);

ALTER TABLE seasons
    ADD COLUMN IF NOT EXISTS overview_ar TEXT,
    ADD COLUMN IF NOT EXISTS overview_en TEXT,
    ADD COLUMN IF NOT EXISTS air_date DATE,
    ADD COLUMN IF NOT EXISTS poster_path VARCHAR(500),
    ADD COLUMN IF NOT EXISTS provider VARCHAR(30),
    ADD COLUMN IF NOT EXISTS external_id VARCHAR(40);

-- -----------------------------------------------------------------------------
-- 7. Physical file -> logical entity relation
-- -----------------------------------------------------------------------------
-- `episode_id` is the missing link that removes the ambiguity of matching a file
-- by (season_id, episode_number) alone. It is nullable so part files and the
-- existing rows remain valid.
ALTER TABLE video_files
    ADD COLUMN IF NOT EXISTS episode_id INT REFERENCES episodes(id) ON DELETE SET NULL,
    -- how this file was attached to its work
    ADD COLUMN IF NOT EXISTS resolution_state resolution_state DEFAULT 'indexed',
    ADD COLUMN IF NOT EXISTS resolver_confidence REAL,
    ADD COLUMN IF NOT EXISTS resolution_source value_source DEFAULT 'resolver',
    -- different releases of the same episode are legitimate; this lets the
    -- resolver attach them to one episode instead of inventing a work per file
    ADD COLUMN IF NOT EXISTS release_key TEXT;

CREATE INDEX IF NOT EXISTS idx_video_files_episode_id ON video_files(episode_id);
CREATE INDEX IF NOT EXISTS idx_video_files_resolution_state ON video_files(resolution_state);

-- The episode identity index from 0022 was non-unique precisely because legacy
-- data contains collisions that predate entity resolution. Now that files carry
-- episode_id, the meaningful uniqueness is per episode row, and multiple release
-- files per episode are allowed. Renamed for clarity instead of silently kept.
CREATE INDEX IF NOT EXISTS idx_video_files_season_episode_lookup
    ON video_files(season_id, episode_number)
    WHERE season_id IS NOT NULL AND episode_number IS NOT NULL;

-- -----------------------------------------------------------------------------
-- 8. Search projection tables
-- -----------------------------------------------------------------------------
-- The search index is a PROJECTION: it is rebuilt from these tables without ever
-- re-reading the filesystem. Storing the projection also makes "add one episode,
-- update one document" possible instead of rebuilding the library.
CREATE TABLE IF NOT EXISTS search_projection_state (
    -- one row per projected entity kind so progress is resumable
    kind VARCHAR(30) PRIMARY KEY,
    last_projected_id BIGINT NOT NULL DEFAULT 0,
    last_run_at TIMESTAMP,
    document_count BIGINT NOT NULL DEFAULT 0
);

INSERT INTO search_projection_state (kind) VALUES ('media_items'), ('episodes')
ON CONFLICT (kind) DO NOTHING;

-- -----------------------------------------------------------------------------
-- 9. Backfill existing rows so the new state columns describe reality
-- -----------------------------------------------------------------------------
-- Existing works were created by the old per-file ingest, so they are marked
-- provisional: they may be merged into a proper entity by a later resolution or
-- enrichment pass. Nothing is deleted and nothing is renamed here.
UPDATE media_items
SET provisional = TRUE,
    resolution_state = COALESCE(resolution_state, 'resolved'),
    title_source = COALESCE(title_source, 'parser')
WHERE provisional = FALSE
  AND title_en IS NOT NULL;

UPDATE video_files
SET resolution_state = COALESCE(resolution_state, 'indexed'),
    resolution_source = COALESCE(resolution_source, 'filesystem')
WHERE resolution_state IS NULL;

COMMENT ON TABLE media_aliases IS
    'Rule-based library memory: every spelling variant that maps to one work. Learned from resolutions and operator decisions; never from an LLM.';
COMMENT ON TABLE resolution_queue IS
    'Ambiguous files that must not become automatic works. An operator decision here is promoted into media_aliases when safe.';
COMMENT ON TABLE episodes IS
    'Provider-backed or operator-created episode entity. A file attaches to it, so one episode may have several release files.';
COMMENT ON COLUMN media_items.provisional IS
    'True when the work was derived from a filename and may be merged into a provider-backed entity later.';
COMMENT ON COLUMN media_items.metadata_locked IS
    'True when an operator edited this work. The scanner must never overwrite a locked work.';
COMMENT ON COLUMN video_files.episode_id IS
    'The episode entity this physical file belongs to. Several release files may share one episode.';
COMMENT ON COLUMN media_items.merged_into_id IS
    'Set when this provisional work was merged into a canonical work; kept for auditability instead of deleting history.';
