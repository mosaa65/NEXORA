package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"unicode"

	"github.com/lib/pq"

	"nexora/server/internal/scanner"
)

// ClassifyMedia maps a path and its parse result to a category slug and a media
// type. Category detection is segment-based, so a folder like "/NotMovies/" can
// no longer be misread as "movies".
func ClassifyMedia(path string, parsed scanner.ParsedName) (categorySlug string, mediaType string) {
	detectedSlug := parsed.CategorySlug
	if detectedSlug == "" {
		detectedSlug = scanner.DetectCategoryFromPath(path)
	}

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

// progressJSON serialises a scan progress snapshot for the session stats column.
// An encoding failure must never fail a scan, so it degrades to "{}".
func progressJSON(progress scanner.Progress) string {
	encoded, err := json.Marshal(progress)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}
