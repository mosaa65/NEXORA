-- The playback-segment feature was removed. This is intentionally safe for
-- fresh installations where the old table was never created.
DROP TABLE IF EXISTS playback_segments;
