-- Add content_rating to media_items
ALTER TABLE media_items
    ADD COLUMN IF NOT EXISTS content_rating VARCHAR(30);

CREATE INDEX IF NOT EXISTS idx_media_items_content_rating ON media_items(content_rating);

-- Cleanup duplicate English keywords from genres array on existing media_items
UPDATE media_items
SET genres = (
    SELECT array_agg(DISTINCT cleaned_tag)
    FROM (
        SELECT CASE
            WHEN tag ILIKE 'animation' THEN 'أنمي'
            WHEN tag ILIKE 'action' THEN 'أكشن'
            WHEN tag ILIKE 'adventure' THEN 'مغامرة'
            WHEN tag ILIKE 'comedy' THEN 'كوميديا'
            WHEN tag ILIKE 'drama' THEN 'دراما'
            WHEN tag ILIKE 'crime' THEN 'جريمة'
            WHEN tag ILIKE 'documentary' THEN 'وثائقي'
            WHEN tag ILIKE 'fantasy' THEN 'فانتازيا'
            WHEN tag ILIKE 'family' THEN 'عائلي'
            WHEN tag ILIKE 'horror' THEN 'رعب'
            WHEN tag ILIKE 'history' THEN 'تاريخي'
            WHEN tag ILIKE 'mystery' THEN 'غموض'
            WHEN tag ILIKE 'romance' THEN 'رومانسي'
            WHEN tag ILIKE 'science fiction' OR tag ILIKE 'sci-fi' THEN 'خيال علمي'
            WHEN tag ILIKE 'thriller' THEN 'إثارة'
            WHEN tag ILIKE 'war' THEN 'حرب'
            WHEN tag ILIKE 'music' THEN 'موسيقى'
            WHEN tag ILIKE 'western' THEN 'ويسترن'
            WHEN tag ILIKE 'kids' THEN 'أطفال'
            ELSE tag
        END AS cleaned_tag
        FROM unnest(genres) AS tag
        WHERE tag IS NOT NULL AND tag != ''
    ) cleaned_sub
    WHERE cleaned_tag IS NOT NULL AND cleaned_tag != ''
)
WHERE genres IS NOT NULL AND array_length(genres, 1) > 0;
