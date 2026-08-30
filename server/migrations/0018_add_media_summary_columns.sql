-- Migration 0018: Add precalculated media summary columns and automatic maintenance triggers
-- This optimizes media listings and eliminates heavy runtime table-scan aggregations (CTEs)
-- for massive 100+ TB libraries under 250+ concurrent client load.

ALTER TABLE media_items
    ADD COLUMN IF NOT EXISTS file_count INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS season_count INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_file_size BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS best_resolution VARCHAR(50) DEFAULT '',
    ADD COLUMN IF NOT EXISTS runtime_minutes INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS has_arabic_audio BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS has_arabic_subtitles BOOLEAN DEFAULT false;

-- Optimized sorting and filtering indexes
CREATE INDEX IF NOT EXISTS idx_media_items_sort_created ON media_items(created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_media_items_sort_rating ON media_items(rating DESC NULLS LAST, id DESC);
CREATE INDEX IF NOT EXISTS idx_media_items_sort_year ON media_items(release_year DESC NULLS LAST, id DESC);
CREATE INDEX IF NOT EXISTS idx_media_items_genres_gin ON media_items USING GIN (genres);

-- Function to recalculate summary for a single media_item
CREATE OR REPLACE FUNCTION update_single_media_summary(target_id BIGINT)
RETURNS VOID AS $$
BEGIN
    IF target_id IS NULL THEN
        RETURN;
    END IF;

    UPDATE media_items mi
    SET
        file_count = COALESCE(fs.file_count, 0),
        season_count = COALESCE(ss.season_count, 0),
        total_file_size = COALESCE(fs.total_size, 0),
        best_resolution = COALESCE(fs.best_resolution, ''),
        runtime_minutes = COALESCE(fs.local_runtime_minutes, 0),
        has_arabic_audio = COALESCE(fs.has_arabic_audio, false),
        has_arabic_subtitles = COALESCE(fs.has_arabic_subtitles, false)
    FROM (
        SELECT
            COUNT(*)::int AS file_count,
            COALESCE(SUM(file_size), 0)::bigint AS total_size,
            MAX(duration) FILTER (WHERE episode_number IS NULL)::int / 60 AS local_runtime_minutes,
            CASE MAX(CASE WHEN resolution ~* '(2160|4k)' THEN 4 WHEN resolution ~* '1440' THEN 3 WHEN resolution ~* '1080' THEN 2 WHEN resolution ~* '720' THEN 1 ELSE 0 END)
                WHEN 4 THEN '4K' WHEN 3 THEN '1440p' WHEN 2 THEN '1080p' WHEN 1 THEN '720p' ELSE '' END AS best_resolution,
            BOOL_OR(LOWER(COALESCE(audio_tracks::text, '')) LIKE '%"language":"ara"%' OR LOWER(COALESCE(audio_tracks::text, '')) LIKE '%"language":"ar"%') AS has_arabic_audio,
            BOOL_OR(LOWER(COALESCE(subtitles::text, '')) LIKE '%"language":"ara"%' OR LOWER(COALESCE(subtitles::text, '')) LIKE '%"language":"ar"%') AS has_arabic_subtitles
        FROM video_files
        WHERE media_item_id = target_id
    ) fs
    CROSS JOIN (
        SELECT COUNT(*)::int AS season_count
        FROM seasons
        WHERE media_item_id = target_id
    ) ss
    WHERE mi.id = target_id;
END;
$$ LANGUAGE plpgsql;

-- Trigger on video_files to maintain media_items summary
CREATE OR REPLACE FUNCTION trg_video_files_summary_update()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM update_single_media_summary(OLD.media_item_id);
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.media_item_id <> NEW.media_item_id THEN
            PERFORM update_single_media_summary(OLD.media_item_id);
        END IF;
        PERFORM update_single_media_summary(NEW.media_item_id);
    ELSE
        PERFORM update_single_media_summary(NEW.media_item_id);
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_video_files_summary ON video_files;
CREATE TRIGGER trg_video_files_summary
AFTER INSERT OR UPDATE OR DELETE ON video_files
FOR EACH ROW EXECUTE FUNCTION trg_video_files_summary_update();

-- Trigger on seasons to maintain media_items summary
CREATE OR REPLACE FUNCTION trg_seasons_summary_update()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM update_single_media_summary(OLD.media_item_id);
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.media_item_id <> NEW.media_item_id THEN
            PERFORM update_single_media_summary(OLD.media_item_id);
        END IF;
        PERFORM update_single_media_summary(NEW.media_item_id);
    ELSE
        PERFORM update_single_media_summary(NEW.media_item_id);
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_seasons_summary ON seasons;
CREATE TRIGGER trg_seasons_summary
AFTER INSERT OR UPDATE OR DELETE ON seasons
FOR EACH ROW EXECUTE FUNCTION trg_seasons_summary_update();

-- Initial batch backfill for all existing media_items
UPDATE media_items mi
SET
    file_count = COALESCE(fs.file_count, 0),
    season_count = COALESCE(ss.season_count, 0),
    total_file_size = COALESCE(fs.total_size, 0),
    best_resolution = COALESCE(fs.best_resolution, ''),
    runtime_minutes = COALESCE(fs.local_runtime_minutes, 0),
    has_arabic_audio = COALESCE(fs.has_arabic_audio, false),
    has_arabic_subtitles = COALESCE(fs.has_arabic_subtitles, false)
FROM (
    SELECT
        media_item_id,
        COUNT(*)::int AS file_count,
        COALESCE(SUM(file_size), 0)::bigint AS total_size,
        MAX(duration) FILTER (WHERE episode_number IS NULL)::int / 60 AS local_runtime_minutes,
        CASE MAX(CASE WHEN resolution ~* '(2160|4k)' THEN 4 WHEN resolution ~* '1440' THEN 3 WHEN resolution ~* '1080' THEN 2 WHEN resolution ~* '720' THEN 1 ELSE 0 END)
            WHEN 4 THEN '4K' WHEN 3 THEN '1440p' WHEN 2 THEN '1080p' WHEN 1 THEN '720p' ELSE '' END AS best_resolution,
        BOOL_OR(LOWER(COALESCE(audio_tracks::text, '')) LIKE '%"language":"ara"%' OR LOWER(COALESCE(audio_tracks::text, '')) LIKE '%"language":"ar"%') AS has_arabic_audio,
        BOOL_OR(LOWER(COALESCE(subtitles::text, '')) LIKE '%"language":"ara"%' OR LOWER(COALESCE(subtitles::text, '')) LIKE '%"language":"ar"%') AS has_arabic_subtitles
    FROM video_files
    GROUP BY media_item_id
) fs
FULL OUTER JOIN (
    SELECT media_item_id, COUNT(*)::int AS season_count
    FROM seasons
    GROUP BY media_item_id
) ss ON ss.media_item_id = fs.media_item_id
WHERE mi.id = COALESCE(fs.media_item_id, ss.media_item_id);
