-- Editorial showcases are intentionally independent from provider metadata.
-- Their artwork is a local NEXORA asset path and their target remains a
-- database-backed category/filter definition.
CREATE TABLE IF NOT EXISTS collections (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(120) NOT NULL UNIQUE,
    title_ar TEXT,
    title_en TEXT NOT NULL,
    description_ar TEXT,
    description_en TEXT,
    artwork_path TEXT,
    artwork_position VARCHAR(80) NOT NULL DEFAULT 'center center',
    accent VARCHAR(40) NOT NULL DEFAULT 'violet',
    target_category_slug VARCHAR(120),
    target_filters JSONB NOT NULL DEFAULT '{}'::jsonb,
    priority INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_collections_active_priority
    ON collections (is_active, priority DESC, id DESC);

CREATE TABLE IF NOT EXISTS collection_items (
    collection_id BIGINT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    media_item_id BIGINT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (collection_id, media_item_id)
);

CREATE INDEX IF NOT EXISTS idx_collection_items_order
    ON collection_items (collection_id, sort_order, media_item_id);
