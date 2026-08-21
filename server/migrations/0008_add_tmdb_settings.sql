-- 0008_add_tmdb_settings.sql
-- Table for TMDB Settings singleton
CREATE TABLE IF NOT EXISTS tmdb_settings (
    id INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    fetch_mode VARCHAR(30) NOT NULL DEFAULT 'standard',
    image_mode VARCHAR(30) NOT NULL DEFAULT 'hybrid',
    preferred_language VARCHAR(20) NOT NULL DEFAULT 'ar-SA',
    fallback_language VARCHAR(20) NOT NULL DEFAULT 'en-US',
    include_image_language VARCHAR(50) NOT NULL DEFAULT 'ar,en,null',
    daily_bandwidth_mb BIGINT NOT NULL DEFAULT 500,
    enable_rate_limit_delay BOOLEAN NOT NULL DEFAULT TRUE,
    rate_limit_requests_per_sec INT NOT NULL DEFAULT 35,
    poster_size VARCHAR(20) NOT NULL DEFAULT 'w500',
    backdrop_size VARCHAR(20) NOT NULL DEFAULT 'original',
    profile_size VARCHAR(20) NOT NULL DEFAULT 'w185',
    still_size VARCHAR(20) NOT NULL DEFAULT 'w300',
    modules JSONB NOT NULL DEFAULT '{}'::jsonb,
    remote_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Table for tracking API request count & bandwidth logs
CREATE TABLE IF NOT EXISTS tmdb_usage_log (
    id BIGSERIAL PRIMARY KEY,
    media_item_id INT REFERENCES media_items(id) ON DELETE SET NULL,
    request_kind VARCHAR(50) NOT NULL, -- "search", "details", "season", "image", "config"
    endpoint VARCHAR(300) NOT NULL,
    status_code INT NOT NULL DEFAULT 200,
    bytes_downloaded BIGINT NOT NULL DEFAULT 0,
    images_downloaded INT NOT NULL DEFAULT 0,
    duration_ms INT NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tmdb_usage_log_created_at
    ON tmdb_usage_log (created_at);

CREATE INDEX IF NOT EXISTS idx_tmdb_usage_log_media_item
    ON tmdb_usage_log (media_item_id);

-- Initialize default row if not present
INSERT INTO tmdb_settings (id, fetch_mode, image_mode, modules)
VALUES (
    1,
    'standard',
    'hybrid',
    '{
        "fetch_title": true,
        "fetch_overview": true,
        "fetch_genres": true,
        "fetch_keywords": true,
        "fetch_release_dates": true,
        "fetch_content_ratings": true,
        "fetch_external_ids": true,
        "fetch_translations": true,
        "fetch_alternative_titles": true,
        "fetch_trailers": true,
        "fetch_credits_text": true,
        "fetch_recommendations": true,
        "fetch_similar": true,
        "fetch_reviews": false,
        "fetch_watch_providers": false,
        "fetch_poster": true,
        "fetch_backdrop": true,
        "max_cast_images": 0,
        "max_gallery_posters": 0,
        "max_gallery_backdrops": 0,
        "max_gallery_logos": 0,
        "max_related_posters": 0,
        "fetch_season_overview": true,
        "fetch_season_posters": false,
        "fetch_episode_overview": true,
        "fetch_episode_stills": false
    }'::jsonb
)
ON CONFLICT (id) DO NOTHING;
