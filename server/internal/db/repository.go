package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"nexora/server/internal/scanner"
	"nexora/server/internal/search"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(database *sql.DB) *Repository {
	return &Repository{db: database}
}

type Health struct {
	DatabaseOK bool      `json:"databaseOk"`
	CheckedAt  time.Time `json:"checkedAt"`
}

type IngestResult struct {
	Scanned  int `json:"scanned"`
	Imported int `json:"imported"`
}

func (r *Repository) Health(ctx context.Context) (Health, error) {
	if err := r.db.PingContext(ctx); err != nil {
		return Health{CheckedAt: time.Now().UTC()}, err
	}
	return Health{DatabaseOK: true, CheckedAt: time.Now().UTC()}, nil
}

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

func (r *Repository) ListSearchDocuments(ctx context.Context, limit int) ([]search.MediaDocument, error) {
	if limit <= 0 || limit > 10000 {
		limit = 1000
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			mi.id,
			mi.title_ar,
			mi.title_en,
			mi.type,
			mi.plot_ar,
			mi.plot_en,
			mi.release_year,
			mi.rating,
			mi.poster_path,
			mi.banner_path,
			COALESCE(array_to_json(mi.genres), '[]'::json)::text AS genres,
			c.slug,
			c.name_ar,
			c.name_en,
			COUNT(vf.id) AS file_count
		FROM media_items mi
		LEFT JOIN categories c ON c.id = mi.category_id
		LEFT JOIN video_files vf ON vf.media_item_id = mi.id
		GROUP BY mi.id, c.id
		ORDER BY mi.created_at DESC, mi.id DESC
		LIMIT $1;
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query search documents: %w", err)
	}
	defer rows.Close()

	documents := make([]search.MediaDocument, 0)
	for rows.Next() {
		var doc search.MediaDocument
		var titleAR, plotAR, plotEN, posterPath, bannerPath, categorySlug, categoryAR, categoryEN sql.NullString
		var releaseYear sql.NullInt64
		var rating sql.NullFloat64
		var genresText string
		var fileCount int64

		if err := rows.Scan(
			&doc.ID,
			&titleAR,
			&doc.TitleEN,
			&doc.Type,
			&plotAR,
			&plotEN,
			&releaseYear,
			&rating,
			&posterPath,
			&bannerPath,
			&genresText,
			&categorySlug,
			&categoryAR,
			&categoryEN,
			&fileCount,
		); err != nil {
			return nil, fmt.Errorf("scan search document: %w", err)
		}

		doc.TitleAR = nullableString(titleAR)
		doc.PlotAR = nullableString(plotAR)
		doc.PlotEN = nullableString(plotEN)
		doc.PosterPath = nullableString(posterPath)
		doc.BannerPath = nullableString(bannerPath)
		doc.CategorySlug = nullableString(categorySlug)
		doc.CategoryAR = nullableString(categoryAR)
		doc.CategoryEN = nullableString(categoryEN)
		if releaseYear.Valid {
			doc.ReleaseYear = int(releaseYear.Int64)
		}
		if rating.Valid {
			doc.Rating = rating.Float64
		}
		doc.FileCount = int(fileCount)
		if err := json.Unmarshal([]byte(genresText), &doc.Genres); err != nil {
			doc.Genres = nil
		}

		documents = append(documents, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search documents: %w", err)
	}

	return documents, nil
}

func nullableString(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func (r *Repository) ingestScannedFile(ctx context.Context, file scanner.FileInfo) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin ingest: %w", err)
	}
	defer tx.Rollback()

	categorySlug, mediaType := classifyMedia(file.Path, file.Parsed)

	var categoryID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM categories WHERE slug = $1`, categorySlug).Scan(&categoryID); err != nil {
		return fmt.Errorf("find category %q: %w", categorySlug, err)
	}

	title := file.Parsed.Title
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(file.Path), filepath.Ext(file.Path))
	}
	titleAR := nullableTitleAR(title)
	titleEN := title

	var mediaID int64
	if err := tx.QueryRowContext(ctx, `
		WITH existing AS (
			SELECT id
			FROM media_items
			WHERE type = $1
			  AND LOWER(title_en) = LOWER($2)
			  AND COALESCE(release_year, 0) = $3
			LIMIT 1
		),
		inserted AS (
			INSERT INTO media_items (
				category_id,
				title_ar,
				title_en,
				type,
				release_year,
				status
			)
			SELECT $4, $5, $2, $1, NULLIF($3, 0), 'completed'
			WHERE NOT EXISTS (SELECT 1 FROM existing)
			RETURNING id
		)
		SELECT id FROM inserted
		UNION ALL
		SELECT id FROM existing
		LIMIT 1;
	`, mediaType, titleEN, file.Parsed.ReleaseYear, categoryID, titleAR).Scan(&mediaID); err != nil {
		return fmt.Errorf("upsert media item %q: %w", title, err)
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

func classifyMedia(path string, parsed scanner.ParsedName) (categorySlug string, mediaType string) {
	lowerPath := strings.ToLower(filepath.ToSlash(path))
	switch {
	case strings.Contains(lowerPath, "anime") || strings.Contains(path, "أنمي"):
		return "anime", "anime"
	case strings.Contains(lowerPath, "kids") || strings.Contains(path, "أطفال"):
		if parsed.IsEpisode {
			return "kids", "series"
		}
		return "kids", "movie"
	case strings.Contains(lowerPath, "document") || strings.Contains(path, "وثائقي"):
		return "documentaries", "movie"
	case strings.Contains(lowerPath, "play") || strings.Contains(path, "مسرح"):
		return "plays", "movie"
	case parsed.IsEpisode:
		return "series", "series"
	default:
		return "movies", "movie"
	}
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
