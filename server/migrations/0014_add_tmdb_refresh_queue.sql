ALTER TABLE tmdb_settings ADD COLUMN IF NOT EXISTS auto_refresh_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE tmdb_settings ADD COLUMN IF NOT EXISTS refresh_interval_days INT NOT NULL DEFAULT 30;
ALTER TABLE tmdb_settings ADD COLUMN IF NOT EXISTS refresh_on_open BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE tmdb_settings ADD COLUMN IF NOT EXISTS refresh_stale_days INT NOT NULL DEFAULT 7;
ALTER TABLE tmdb_settings ADD COLUMN IF NOT EXISTS queue_max_concurrent INT NOT NULL DEFAULT 1;

CREATE TABLE IF NOT EXISTS tmdb_refresh_queue (
    id BIGSERIAL PRIMARY KEY,
    media_item_id INT REFERENCES media_items(id) ON DELETE CASCADE,
    requested_by VARCHAR(30) NOT NULL DEFAULT 'admin',
    priority INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','running','succeeded','failed','cancelled')),
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    scheduled_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (media_item_id, status)
);
CREATE INDEX IF NOT EXISTS idx_tmdb_refresh_queue_status ON tmdb_refresh_queue(status, priority DESC, scheduled_at, id);