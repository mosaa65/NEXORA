ALTER TABLE media_items
    ADD COLUMN IF NOT EXISTS metadata_provider VARCHAR(50),
    ADD COLUMN IF NOT EXISTS metadata_external_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS metadata_fetched_at TIMESTAMP,
    ADD COLUMN IF NOT EXISTS metadata_expires_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_media_items_metadata_source
    ON media_items (metadata_provider, metadata_external_id);

CREATE TABLE IF NOT EXISTS metadata_snapshots (
    id BIGSERIAL PRIMARY KEY,
    media_item_id INT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    external_id VARCHAR(100) NOT NULL,
    locale VARCHAR(20) NOT NULL DEFAULT 'en-US',
    raw_payload JSONB NOT NULL,
    fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    UNIQUE (media_item_id, provider, locale)
);

CREATE INDEX IF NOT EXISTS idx_metadata_snapshots_expiry
    ON metadata_snapshots (expires_at);
