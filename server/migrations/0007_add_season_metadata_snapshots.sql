CREATE TABLE IF NOT EXISTS season_metadata_snapshots (
    id BIGSERIAL PRIMARY KEY,
    media_item_id INT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    external_id VARCHAR(100) NOT NULL,
    season_number INT NOT NULL,
    locale VARCHAR(20) NOT NULL,
    raw_payload JSONB NOT NULL,
    fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    UNIQUE (media_item_id, provider, season_number, locale)
);

CREATE INDEX IF NOT EXISTS idx_season_metadata_snapshots_lookup
    ON season_metadata_snapshots (media_item_id, season_number, locale);

CREATE INDEX IF NOT EXISTS idx_season_metadata_snapshots_payload_gin
    ON season_metadata_snapshots USING GIN (raw_payload jsonb_path_ops);
