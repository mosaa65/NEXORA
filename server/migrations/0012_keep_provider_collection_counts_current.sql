-- Keep the cached count aligned with the actual many-to-many links even when
-- a media item is removed through a cascading foreign-key delete.
CREATE OR REPLACE FUNCTION refresh_provider_collection_local_count()
RETURNS TRIGGER AS $$
DECLARE
    affected_collection_id BIGINT;
BEGIN
    affected_collection_id := COALESCE(NEW.collection_id, OLD.collection_id);
    UPDATE provider_collections
    SET local_item_count = (
            SELECT COUNT(*)
            FROM media_collection_links
            WHERE collection_id = affected_collection_id
        ),
        updated_at = CURRENT_TIMESTAMP
    WHERE id = affected_collection_id;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_refresh_provider_collection_count ON media_collection_links;
CREATE TRIGGER trg_refresh_provider_collection_count
AFTER INSERT OR DELETE OR UPDATE OF collection_id ON media_collection_links
FOR EACH ROW EXECUTE FUNCTION refresh_provider_collection_local_count();

-- Repair any counts produced before this trigger was introduced.
UPDATE provider_collections pc
SET local_item_count = (
        SELECT COUNT(*)
        FROM media_collection_links mcl
        WHERE mcl.collection_id = pc.id
    ),
    updated_at = CURRENT_TIMESTAMP;
