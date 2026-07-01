CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    name_ar VARCHAR(100) NOT NULL,
    name_en VARCHAR(100) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS media_items (
    id SERIAL PRIMARY KEY,
    category_id INT REFERENCES categories(id),
    title_ar VARCHAR(255),
    title_en VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    plot_ar TEXT,
    plot_en TEXT,
    release_year INT,
    rating NUMERIC(3, 1),
    poster_path VARCHAR(500),
    banner_path VARCHAR(500),
    genres VARCHAR(255)[],
    status VARCHAR(50) DEFAULT 'completed',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS seasons (
    id SERIAL PRIMARY KEY,
    media_item_id INT REFERENCES media_items(id) ON DELETE CASCADE,
    season_number INT NOT NULL,
    title_ar VARCHAR(150),
    title_en VARCHAR(150),
    UNIQUE(media_item_id, season_number)
);

CREATE TABLE IF NOT EXISTS video_files (
    id SERIAL PRIMARY KEY,
    media_item_id INT REFERENCES media_items(id) ON DELETE CASCADE,
    season_id INT REFERENCES seasons(id) ON DELETE CASCADE,
    episode_number INT,
    title_ar VARCHAR(255),
    title_en VARCHAR(255),
    file_path TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    duration INT,
    resolution VARCHAR(50),
    video_codec VARCHAR(50),
    audio_tracks JSONB,
    subtitles JSONB,
    checksum VARCHAR(64),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS storage_disks (
    id SERIAL PRIMARY KEY,
    disk_letter CHAR(1) UNIQUE NOT NULL,
    disk_label VARCHAR(100),
    total_space BIGINT NOT NULL,
    free_space BIGINT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    last_scanned TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_video_files_file_path ON video_files(file_path);
CREATE INDEX IF NOT EXISTS idx_media_items_category_id ON media_items(category_id);
CREATE INDEX IF NOT EXISTS idx_media_items_title_en ON media_items(title_en);
CREATE INDEX IF NOT EXISTS idx_media_items_type ON media_items(type);
CREATE UNIQUE INDEX IF NOT EXISTS idx_media_items_identity ON media_items(type, LOWER(title_en), COALESCE(release_year, 0));
CREATE INDEX IF NOT EXISTS idx_seasons_media_item_id ON seasons(media_item_id);
CREATE INDEX IF NOT EXISTS idx_video_files_media_item_id ON video_files(media_item_id);
CREATE INDEX IF NOT EXISTS idx_video_files_season_id ON video_files(season_id);
CREATE INDEX IF NOT EXISTS idx_video_files_episode_number ON video_files(episode_number);

INSERT INTO categories (name_ar, name_en, slug) VALUES
    ('أفلام', 'Movies', 'movies'),
    ('مسلسلات', 'Series', 'series'),
    ('أنمي', 'Anime', 'anime'),
    ('أطفال', 'Kids', 'kids'),
    ('مسرحيات', 'Plays', 'plays'),
    ('وثائقيات', 'Documentaries', 'documentaries')
ON CONFLICT (slug) DO NOTHING;
