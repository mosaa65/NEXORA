ALTER TABLE media_items
    ADD COLUMN IF NOT EXISTS metadata_facets JSONB NOT NULL DEFAULT '{}'::jsonb;

-- Supports future filtering/faceting without re-reading or re-downloading the
-- full provider snapshot. The snapshot itself remains in metadata_snapshots.
CREATE INDEX IF NOT EXISTS idx_media_items_metadata_facets_gin
    ON media_items USING GIN (metadata_facets jsonb_path_ops);
