ALTER TABLE video_files
    ADD COLUMN IF NOT EXISTS verification_status VARCHAR(20) NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS verification_error TEXT,
    ADD COLUMN IF NOT EXISTS verification_checked_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_video_files_verification_status
    ON video_files (verification_status);
