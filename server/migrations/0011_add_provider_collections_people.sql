-- Provider-backed catalog entities are intentionally separate from the
-- editorial `collections` table. They are keyed by stable provider IDs, never
-- by translated titles, so enrichment remains deterministic and offline
-- browsing can use local relational data only.

CREATE TABLE IF NOT EXISTS provider_collections (
    id BIGSERIAL PRIMARY KEY,
    provider VARCHAR(50) NOT NULL,
    external_id VARCHAR(100) NOT NULL,
    kind VARCHAR(50) NOT NULL DEFAULT 'movie_collection',
    slug VARCHAR(180) NOT NULL UNIQUE,
    title_ar TEXT,
    title_en TEXT NOT NULL,
    overview_ar TEXT,
    overview_en TEXT,
    poster_path TEXT,
    backdrop_path TEXT,
    parts_count INTEGER NOT NULL DEFAULT 0,
    local_item_count INTEGER NOT NULL DEFAULT 0,
    metadata_fetched_at TIMESTAMP,
    metadata_expires_at TIMESTAMP,
    is_featured BOOLEAN NOT NULL DEFAULT FALSE,
    is_hidden BOOLEAN NOT NULL DEFAULT FALSE,
    sort_priority INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (provider, external_id)
);

CREATE INDEX IF NOT EXISTS idx_provider_collections_visible_priority
    ON provider_collections (is_hidden, is_featured DESC, sort_priority DESC, title_en);

CREATE TABLE IF NOT EXISTS media_collection_links (
    media_item_id INTEGER NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    collection_id BIGINT NOT NULL REFERENCES provider_collections(id) ON DELETE CASCADE,
    source VARCHAR(20) NOT NULL DEFAULT 'tmdb' CHECK (source IN ('tmdb', 'manual')),
    tmdb_order INTEGER,
    linked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    verified_at TIMESTAMP,
    PRIMARY KEY (media_item_id, collection_id)
);

CREATE INDEX IF NOT EXISTS idx_media_collection_links_collection_order
    ON media_collection_links (collection_id, tmdb_order NULLS LAST, media_item_id);

CREATE TABLE IF NOT EXISTS collection_metadata_snapshots (
    id BIGSERIAL PRIMARY KEY,
    collection_id BIGINT NOT NULL REFERENCES provider_collections(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    external_id VARCHAR(100) NOT NULL,
    locale VARCHAR(20) NOT NULL,
    raw_payload JSONB NOT NULL,
    fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    UNIQUE (collection_id, provider, locale)
);

CREATE INDEX IF NOT EXISTS idx_collection_metadata_snapshots_expiry
    ON collection_metadata_snapshots (expires_at);

CREATE TABLE IF NOT EXISTS people (
    id BIGSERIAL PRIMARY KEY,
    provider VARCHAR(50) NOT NULL,
    external_id VARCHAR(100) NOT NULL,
    slug VARCHAR(180) NOT NULL UNIQUE,
    name_ar TEXT,
    name_en TEXT NOT NULL,
    known_for_department VARCHAR(80),
    profile_path TEXT,
    popularity NUMERIC(10,3),
    local_media_count INTEGER NOT NULL DEFAULT 0,
    metadata_fetched_at TIMESTAMP,
    metadata_expires_at TIMESTAMP,
    is_featured BOOLEAN NOT NULL DEFAULT FALSE,
    is_hidden BOOLEAN NOT NULL DEFAULT FALSE,
    sort_priority INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (provider, external_id)
);

CREATE INDEX IF NOT EXISTS idx_people_visible_priority
    ON people (is_hidden, is_featured DESC, sort_priority DESC, popularity DESC NULLS LAST, name_en);

CREATE TABLE IF NOT EXISTS media_credits (
    id BIGSERIAL PRIMARY KEY,
    media_item_id INTEGER NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    person_id BIGINT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_credit_id VARCHAR(120),
    credit_kind VARCHAR(20) NOT NULL CHECK (credit_kind IN ('cast', 'crew')),
    character_name TEXT,
    job VARCHAR(120),
    department VARCHAR(120),
    billing_order INTEGER,
    source VARCHAR(20) NOT NULL DEFAULT 'tmdb' CHECK (source IN ('tmdb', 'manual')),
    verified_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_media_credits_provider_identity
    ON media_credits (media_item_id, provider, provider_credit_id)
    WHERE provider_credit_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_media_credits_manual_identity
    ON media_credits (media_item_id, person_id, credit_kind, COALESCE(character_name, ''), COALESCE(job, ''))
    WHERE provider_credit_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_media_credits_person_media
    ON media_credits (person_id, billing_order NULLS LAST, media_item_id);
CREATE INDEX IF NOT EXISTS idx_media_credits_media_billing
    ON media_credits (media_item_id, credit_kind, billing_order NULLS LAST);

CREATE TABLE IF NOT EXISTS person_metadata_snapshots (
    id BIGSERIAL PRIMARY KEY,
    person_id BIGINT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    external_id VARCHAR(100) NOT NULL,
    locale VARCHAR(20) NOT NULL,
    raw_payload JSONB NOT NULL,
    fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    UNIQUE (person_id, provider, locale)
);

CREATE INDEX IF NOT EXISTS idx_person_metadata_snapshots_expiry
    ON person_metadata_snapshots (expires_at);
