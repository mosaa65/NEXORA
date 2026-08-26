ALTER TABLE provider_collections
    ADD COLUMN IF NOT EXISTS rating NUMERIC(4,2);