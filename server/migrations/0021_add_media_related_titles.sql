-- TMDB recommendations and similar titles are provider-backed references,
-- not local media records. Stable provider IDs make the relationship safe
-- even when translated titles are duplicated or change over time.
CREATE TABLE IF NOT EXISTS media_related_titles (
    id BIGSERIAL PRIMARY KEY,
    source_media_item_id INTEGER NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    target_external_id VARCHAR(100) NOT NULL,
    target_kind VARCHAR(20) NOT NULL CHECK (target_kind IN ('movie', 'tv')),
    relation_type VARCHAR(20) NOT NULL CHECK (relation_type IN ('recommendation', 'similar')),
    provider_rank INTEGER NOT NULL DEFAULT 0,
    title_ar TEXT,
    title_en TEXT,
    original_title TEXT,
    overview_ar TEXT,
    overview_en TEXT,
    poster_path TEXT,
    release_year INTEGER,
    rating NUMERIC(4,2),
    fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (source_media_item_id, provider, target_kind, target_external_id, relation_type)
);

CREATE INDEX IF NOT EXISTS idx_media_related_titles_source_rank
    ON media_related_titles (source_media_item_id, relation_type, provider_rank, id);

CREATE INDEX IF NOT EXISTS idx_media_related_titles_target_identity
    ON media_related_titles (provider, target_kind, target_external_id);
