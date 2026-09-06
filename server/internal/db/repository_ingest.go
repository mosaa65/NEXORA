package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/lib/pq"

	"nexora/server/internal/scanner"
)

func (r *Repository) IngestScannedFiles(ctx context.Context, files []scanner.FileInfo) (IngestResult, error) {
	result := IngestResult{Scanned: len(files)}
	for _, file := range files {
		if err := r.ingestScannedFile(ctx, file); err != nil {
			return result, err
		}
		result.Imported++
	}
	return result, nil
}

func (r *Repository) ingestScannedFile(ctx context.Context, file scanner.FileInfo) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin ingest: %w", err)
	}
	defer tx.Rollback()

	categorySlug, mediaType := ClassifyMedia(file.Path, file.Parsed)

	var categoryID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM categories WHERE slug = $1`, categorySlug).Scan(&categoryID); err != nil {
		return fmt.Errorf("find category %q: %w", categorySlug, err)
	}

	title := file.Parsed.Title
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(file.Path), filepath.Ext(file.Path))
	}
	titleAR := nullableTitleAR(file.Parsed.TitleAR)
	if !titleAR.Valid {
		titleAR = nullableTitleAR(title)
	}
	titleEN := file.Parsed.TitleEN
	if titleEN == "" {
		titleEN = title
	}

	localPosterURL := ""
	if file.ArtworkPath != "" {
		localPosterURL = r.CacheLocalArtwork(file.ArtworkPath)
	}

	var mediaID int64
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM media_items
		WHERE type = $1 AND (LOWER(title_en) = LOWER($2) OR (title_ar IS NOT NULL AND title_ar = $3))
		LIMIT 1;
	`, mediaType, titleEN, titleAR).Scan(&mediaID)

	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRowContext(ctx, `
			INSERT INTO media_items (
				category_id,
				title_ar,
				title_en,
				type,
				release_year,
				poster_path,
				banner_path,
				status
			)
			VALUES ($1, $2, $3, $4, NULLIF($5, 0), NULLIF($6, ''), NULLIF($6, ''), 'completed')
			RETURNING id;
		`, categoryID, titleAR, titleEN, mediaType, file.Parsed.ReleaseYear, localPosterURL).Scan(&mediaID)
		if err != nil {
			return fmt.Errorf("insert media item %q: %w", title, err)
		}
	} else if err != nil {
		return fmt.Errorf("find media item %q: %w", title, err)
	} else if localPosterURL != "" {
		_, _ = tx.ExecContext(ctx, `
			UPDATE media_items
			SET poster_path = COALESCE(NULLIF(poster_path, ''), $2),
			    banner_path = COALESCE(NULLIF(banner_path, ''), $2)
			WHERE id = $1 AND (poster_path IS NULL OR poster_path = '' OR poster_path LIKE '/images/placeholder%');
		`, mediaID, localPosterURL)
	}

	// A library folder such as "مسلسلات/عربي" is a stronger signal than a
	// metadata search. Keep that owner-provided classification as a tag while
	// preserving all existing genre tags already attached to the media item.
	if originTags := scanner.DetectOriginTagsFromPath(file.Path); len(originTags) > 0 {
		if err := mergeMediaTags(ctx, tx, mediaID, originTags); err != nil {
			return fmt.Errorf("apply origin tags for %q: %w", title, err)
		}
	}

	seasonID := sql.NullInt64{}
	episodeNumber := sql.NullInt64{}
	if file.Parsed.IsEpisode {
		seasonNumber := file.Parsed.SeasonNumber
		if seasonNumber <= 0 {
			seasonNumber = 1
		}
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO seasons (media_item_id, season_number, title_en)
			VALUES ($1, $2, $3)
			ON CONFLICT (media_item_id, season_number)
			DO UPDATE SET title_en = COALESCE(seasons.title_en, EXCLUDED.title_en)
			RETURNING id;
		`, mediaID, seasonNumber, fmt.Sprintf("Season %02d", seasonNumber)).Scan(&seasonID.Int64); err != nil {
			return fmt.Errorf("upsert season %d: %w", seasonNumber, err)
		}
		seasonID.Valid = true
		episodeNumber = sql.NullInt64{Int64: int64(file.Parsed.EpisodeNumber), Valid: file.Parsed.EpisodeNumber > 0}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO video_files (
			media_item_id,
			season_id,
			episode_number,
			title_ar,
			title_en,
			file_path,
			file_size,
			resolution,
			audio_tracks,
			subtitles
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), '[]'::jsonb, '[]'::jsonb)
		ON CONFLICT (file_path)
		DO UPDATE SET
			media_item_id = EXCLUDED.media_item_id,
			season_id = EXCLUDED.season_id,
			episode_number = EXCLUDED.episode_number,
			title_ar = EXCLUDED.title_ar,
			title_en = EXCLUDED.title_en,
			file_size = EXCLUDED.file_size,
			resolution = EXCLUDED.resolution;
	`, mediaID, seasonID, episodeNumber, titleAR, titleEN, file.Path, file.Size, file.Parsed.Resolution); err != nil {
		return fmt.Errorf("upsert video file %q: %w", file.Path, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit ingest: %w", err)
	}
	return nil
}

func ClassifyMedia(path string, parsed scanner.ParsedName) (categorySlug string, mediaType string) {
	// Use the centralized category detection from the scanner package.
	detectedSlug := scanner.DetectCategoryFromPath(path)

	switch detectedSlug {
	case "anime":
		return "anime", "anime"
	case "kids":
		if parsed.IsEpisode {
			return "kids", "series"
		}
		return "kids", "movie"
	case "documentaries":
		if parsed.IsEpisode {
			return "documentaries", "series"
		}
		return "documentaries", "movie"
	case "plays":
		return "plays", "movie"
	case "series":
		return "series", "series"
	case "movies":
		return "movies", "movie"
	default:
		// No category keyword found in path; fall back to episode detection.
		if parsed.IsEpisode {
			return "series", "series"
		}
		return "movies", "movie"
	}
}

func mergeMediaTags(ctx context.Context, tx *sql.Tx, mediaID int64, tags []string) error {
	if len(tags) == 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, `
		UPDATE media_items
		SET genres = ARRAY(
			SELECT DISTINCT tag
			FROM unnest(COALESCE(genres, ARRAY[]::text[]) || $1::text[]) AS tag
		)
		WHERE id = $2;
	`, pq.Array(tags), mediaID)
	return err
}

// ClassifyOriginsFromPaths repairs existing libraries without re-scanning or
// calling an external API. It reuses the actual source folders already saved
// for each video file, so "مسلسلات/عربي" is classified deterministically.
func (r *Repository) ClassifyOriginsFromPaths(ctx context.Context) (int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT media_item_id, file_path FROM video_files`)
	if err != nil {
		return 0, fmt.Errorf("list file paths for origin classification: %w", err)
	}
	defer rows.Close()

	tagsByMedia := make(map[int64]map[string]struct{})
	for rows.Next() {
		var mediaID int64
		var path string
		if err := rows.Scan(&mediaID, &path); err != nil {
			return 0, fmt.Errorf("scan file path for origin classification: %w", err)
		}
		for _, tag := range scanner.DetectOriginTagsFromPath(path) {
			if tagsByMedia[mediaID] == nil {
				tagsByMedia[mediaID] = make(map[string]struct{})
			}
			tagsByMedia[mediaID][tag] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate file paths for origin classification: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin origin classification: %w", err)
	}
	defer tx.Rollback()

	updated := 0
	for mediaID, tagsSet := range tagsByMedia {
		tags := make([]string, 0, len(tagsSet))
		for tag := range tagsSet {
			tags = append(tags, tag)
		}
		if err := mergeMediaTags(ctx, tx, mediaID, tags); err != nil {
			return 0, fmt.Errorf("classify origin for media %d: %w", mediaID, err)
		}
		updated++
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit origin classification: %w", err)
	}
	return updated, nil
}

func nullableTitleAR(title string) sql.NullString {
	if containsArabic(title) {
		return sql.NullString{String: title, Valid: true}
	}
	return sql.NullString{}
}

func containsArabic(input string) bool {
	for _, r := range input {
		if unicode.In(r, unicode.Arabic) {
			return true
		}
	}
	return false
}
