package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"nexora/server/internal/media"
)

func (r *Repository) ListVideoFiles(ctx context.Context, mediaItemID int64) ([]VideoFile, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id,
			media_item_id,
			COALESCE(season_id, 0),
			COALESCE(episode_number, 0),
			COALESCE(title_ar, ''),
			COALESCE(title_en, ''),
			file_path,
			file_size,
			COALESCE(duration, 0),
			COALESCE(resolution, ''),
			COALESCE(video_codec, ''),
			COALESCE(audio_tracks, '[]'::jsonb)::text,
			COALESCE(subtitles, '[]'::jsonb)::text,
			verification_status,
			COALESCE(verification_error, ''),
			verification_checked_at,
			created_at
		FROM video_files
		WHERE media_item_id = $1
		ORDER BY COALESCE(season_id, 0), COALESCE(episode_number, 0), title_en, file_path;
	`, mediaItemID)
	if err != nil {
		return nil, fmt.Errorf("query video files: %w", err)
	}
	defer rows.Close()

	files := make([]VideoFile, 0)
	for rows.Next() {
		var file VideoFile
		var audioTracks, subtitles string
		var verificationCheckedAt sql.NullTime
		if err := rows.Scan(
			&file.ID,
			&file.MediaItemID,
			&file.SeasonID,
			&file.EpisodeNumber,
			&file.TitleAR,
			&file.TitleEN,
			&file.FilePath,
			&file.FileSize,
			&file.Duration,
			&file.Resolution,
			&file.VideoCodec,
			&audioTracks,
			&subtitles,
			&file.VerificationStatus,
			&file.VerificationError,
			&verificationCheckedAt,
			&file.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan video file: %w", err)
		}
		file.AudioTracks = json.RawMessage(audioTracks)
		file.Subtitles = json.RawMessage(subtitles)
		if verificationCheckedAt.Valid {
			file.VerificationCheckedAt = &verificationCheckedAt.Time
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate video files: %w", err)
	}

	return files, nil
}

func (r *Repository) GetVideoFilePath(ctx context.Context, id int64) (string, error) {
	var path string
	if err := r.db.QueryRowContext(ctx, `SELECT file_path FROM video_files WHERE id = $1`, id).Scan(&path); err != nil {
		return "", fmt.Errorf("get video file path: %w", err)
	}
	return path, nil
}

func (r *Repository) GetVideoFileIDByPath(ctx context.Context, path string) (int64, error) {
	var id int64
	if err := r.db.QueryRowContext(ctx, `SELECT id FROM video_files WHERE file_path = $1`, path).Scan(&id); err != nil {
		return 0, fmt.Errorf("get video file id: %w", err)
	}
	return id, nil
}

func (r *Repository) UpdateVideoTechnicalDetails(ctx context.Context, id int64, details media.InspectResult) error {
	audioTracks, err := json.Marshal(details.AudioTracks)
	if err != nil {
		return fmt.Errorf("encode audio tracks: %w", err)
	}
	subtitles, err := json.Marshal(details.Subtitles)
	if err != nil {
		return fmt.Errorf("encode subtitles: %w", err)
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE video_files
		SET duration = NULLIF($1, 0), resolution = NULLIF($2, ''), video_codec = NULLIF($3, ''), audio_tracks = $4::jsonb, subtitles = $5::jsonb
		WHERE id = $6
	`, details.Duration, details.Resolution, details.VideoCodec, string(audioTracks), string(subtitles), id)
	if err != nil {
		return fmt.Errorf("update technical details: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("technical details affected rows: %w", err)
	}
	if updated == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// UpdateVideoVerification persists the result of a full FFmpeg decode check.
func (r *Repository) UpdateVideoVerification(ctx context.Context, id int64, result media.VerifyResult) error {
	status := "healthy"
	if !result.Healthy {
		status = "corrupted"
	}
	updated, err := r.db.ExecContext(ctx, `
		UPDATE video_files
		SET verification_status = $1,
			verification_error = NULLIF($2, ''),
			verification_checked_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`, status, result.ErrorOutput, id)
	if err != nil {
		return fmt.Errorf("update video verification: %w", err)
	}
	count, err := updated.RowsAffected()
	if err != nil {
		return fmt.Errorf("verification affected rows: %w", err)
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) ListCorruptedFiles(ctx context.Context) ([]CorruptedFile, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT vf.id, vf.media_item_id,
			COALESCE(NULLIF(vf.title_en, ''), NULLIF(vf.title_ar, ''), mi.title_en),
			vf.file_path, COALESCE(vf.verification_error, ''), vf.verification_checked_at
		FROM video_files vf
		JOIN media_items mi ON mi.id = vf.media_item_id
		WHERE vf.verification_status = 'corrupted'
		ORDER BY vf.verification_checked_at DESC NULLS LAST, vf.file_path ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query corrupted files: %w", err)
	}
	defer rows.Close()

	files := make([]CorruptedFile, 0)
	for rows.Next() {
		var file CorruptedFile
		if err := rows.Scan(&file.ID, &file.MediaItemID, &file.Title, &file.FilePath, &file.Error, &file.CheckedAt); err != nil {
			return nil, fmt.Errorf("scan corrupted file: %w", err)
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

func (r *Repository) ListDuplicateGroups(ctx context.Context) ([]DuplicateGroup, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, media_item_id, COALESCE(season_id, 0), COALESCE(episode_number, 0),
			COALESCE(title_ar, ''), COALESCE(title_en, ''), file_path, file_size,
			COALESCE(duration, 0), COALESCE(resolution, ''), COALESCE(video_codec, ''),
			COALESCE(audio_tracks, '[]'::jsonb)::text, COALESCE(subtitles, '[]'::jsonb)::text,
			created_at, checksum
		FROM video_files
		WHERE checksum IS NOT NULL AND checksum <> ''
		  AND checksum IN (SELECT checksum FROM video_files WHERE checksum IS NOT NULL AND checksum <> '' GROUP BY checksum HAVING COUNT(*) > 1)
		ORDER BY checksum, file_path;
	`)
	if err != nil {
		return nil, fmt.Errorf("query duplicates: %w", err)
	}
	defer rows.Close()

	groups := make([]DuplicateGroup, 0)
	byChecksum := make(map[string]int)
	for rows.Next() {
		var file VideoFile
		var audioTracks, subtitles, checksum string
		if err := rows.Scan(&file.ID, &file.MediaItemID, &file.SeasonID, &file.EpisodeNumber, &file.TitleAR, &file.TitleEN, &file.FilePath, &file.FileSize, &file.Duration, &file.Resolution, &file.VideoCodec, &audioTracks, &subtitles, &file.CreatedAt, &checksum); err != nil {
			return nil, fmt.Errorf("scan duplicate: %w", err)
		}
		file.AudioTracks, file.Subtitles = json.RawMessage(audioTracks), json.RawMessage(subtitles)
		index, exists := byChecksum[checksum]
		if !exists {
			index = len(groups)
			byChecksum[checksum] = index
			groups = append(groups, DuplicateGroup{Checksum: checksum, FileSize: file.FileSize})
		}
		groups[index].Files = append(groups[index].Files, file)
	}
	return groups, rows.Err()
}

func (r *Repository) ListMissingEpisodes(ctx context.Context) ([]MissingEpisode, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH numbered AS (
			SELECT season_id, MAX(episode_number) AS max_episode
			FROM video_files WHERE season_id IS NOT NULL AND episode_number IS NOT NULL
			GROUP BY season_id
		)
		SELECT s.media_item_id, s.id, s.season_number, expected.episode_number
		FROM seasons s
		JOIN numbered n ON n.season_id = s.id
		CROSS JOIN LATERAL generate_series(1, n.max_episode) AS expected(episode_number)
		LEFT JOIN video_files vf ON vf.season_id = s.id AND vf.episode_number = expected.episode_number
		WHERE vf.id IS NULL
		ORDER BY s.media_item_id, s.season_number, expected.episode_number;
	`)
	if err != nil {
		return nil, fmt.Errorf("query missing episodes: %w", err)
	}
	defer rows.Close()
	missing := make([]MissingEpisode, 0)
	for rows.Next() {
		var item MissingEpisode
		if err := rows.Scan(&item.MediaItemID, &item.SeasonID, &item.SeasonNumber, &item.Episode); err != nil {
			return nil, fmt.Errorf("scan missing episode: %w", err)
		}
		missing = append(missing, item)
	}
	return missing, rows.Err()
}

// CalculateChecksums hashes files incrementally, never loading video data into memory.
func (r *Repository) CalculateChecksums(ctx context.Context, mediaItemID int64) (ChecksumResult, error) {
	query := `SELECT id, file_path FROM video_files`
	args := []any{}
	if mediaItemID > 0 {
		query += ` WHERE media_item_id = $1`
		args = append(args, mediaItemID)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ChecksumResult{}, fmt.Errorf("query files for checksums: %w", err)
	}
	defer rows.Close()
	result := ChecksumResult{}
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			return result, fmt.Errorf("scan checksum file: %w", err)
		}
		result.Scanned++
		file, err := os.Open(path)
		if err != nil {
			result.Failed++
			continue
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil {
			result.Failed++
			continue
		}
		if _, err := r.db.ExecContext(ctx, `UPDATE video_files SET checksum = $1 WHERE id = $2`, fmt.Sprintf("%x", hash.Sum(nil)), id); err != nil {
			return result, fmt.Errorf("update checksum: %w", err)
		}
		result.Updated++
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("iterate checksum files: %w", err)
	}
	return result, nil
}
