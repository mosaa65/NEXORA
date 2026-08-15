package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"nexora/server/internal/media"
	"nexora/server/internal/metadata"
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

type CategorySummary struct {
	ID         int64  `json:"id"`
	NameAR     string `json:"name_ar"`
	NameEN     string `json:"name_en"`
	Slug       string `json:"slug"`
	MediaCount int    `json:"media_count"`
	FileCount  int    `json:"file_count"`
}

type VideoFile struct {
	ID            int64           `json:"id"`
	MediaItemID   int64           `json:"media_item_id"`
	SeasonID      int64           `json:"season_id,omitempty"`
	EpisodeNumber int             `json:"episode_number,omitempty"`
	TitleAR       string          `json:"title_ar,omitempty"`
	TitleEN       string          `json:"title_en,omitempty"`
	FilePath      string          `json:"-"`
	FileSize      int64           `json:"file_size"`
	Duration      int             `json:"duration,omitempty"`
	Resolution    string          `json:"resolution,omitempty"`
	VideoCodec    string          `json:"video_codec,omitempty"`
	AudioTracks   json.RawMessage `json:"audio_tracks,omitempty"`
	Subtitles     json.RawMessage `json:"subtitles,omitempty"`
	StreamURL     string          `json:"stream_url,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// DuplicateGroup contains files with the same verified SHA-256 checksum.
// Checksums are intentionally calculated on demand so routine ingest stays fast
// even for very large libraries.
type DuplicateGroup struct {
	Checksum string      `json:"checksum"`
	FileSize int64       `json:"file_size"`
	Files    []VideoFile `json:"files"`
}

type MissingEpisode struct {
	MediaItemID  int64 `json:"media_item_id"`
	SeasonID     int64 `json:"season_id"`
	SeasonNumber int   `json:"season_number"`
	Episode      int   `json:"episode_number"`
}

type ChecksumResult struct {
	Scanned int `json:"scanned"`
	Updated int `json:"updated"`
	Failed  int `json:"failed"`
}

type SeasonDetail struct {
	ID           int64       `json:"id"`
	SeasonNumber int         `json:"season_number"`
	TitleAR      string      `json:"title_ar,omitempty"`
	TitleEN      string      `json:"title_en,omitempty"`
	Episodes     []VideoFile `json:"episodes"`
}

type MediaItemDetail struct {
	ID           int64          `json:"id"`
	CategoryID   int64          `json:"category_id"`
	CategorySlug string         `json:"category_slug,omitempty"`
	CategoryAR   string         `json:"category_ar,omitempty"`
	CategoryEN   string         `json:"category_en,omitempty"`
	TitleAR      string         `json:"title_ar,omitempty"`
	TitleEN      string         `json:"title_en"`
	Type         string         `json:"type"`
	PlotAR       string         `json:"plot_ar,omitempty"`
	PlotEN       string         `json:"plot_en,omitempty"`
	ReleaseYear  int            `json:"release_year,omitempty"`
	Rating       float64        `json:"rating,omitempty"`
	PosterPath   string         `json:"poster_path,omitempty"`
	BannerPath   string         `json:"banner_path,omitempty"`
	Genres       []string       `json:"genres,omitempty"`
	Status       string         `json:"status,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	FileCount    int            `json:"file_count"`
	Seasons      []SeasonDetail `json:"seasons,omitempty"`
	Files        []VideoFile    `json:"files,omitempty"`
}

type ListMediaOptions struct {
	CategorySlug string
	Type         string
	Search       string
	Sort         string
	Limit        int
	Offset       int
}

type MediaListResult struct {
	Total  int                    `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
	Items  []search.MediaDocument `json:"items"`
}

type StorageDisk struct {
	ID          int64      `json:"id"`
	DiskLetter  string     `json:"disk_letter"`
	DiskLabel   string     `json:"disk_label"`
	TotalSpace  int64      `json:"total_space"`
	FreeSpace   int64      `json:"free_space"`
	UsedSpace   int64      `json:"used_space"`
	UsedPercent float64    `json:"used_percent"`
	IsActive    bool       `json:"is_active"`
	LastScanned *time.Time `json:"last_scanned,omitempty"`
}

type DashboardStats struct {
	TotalMedia           int64             `json:"total_media"`
	TotalFiles           int64             `json:"total_files"`
	TotalStorageBytes    int64             `json:"total_storage_bytes"`
	MissingEpisodesCount int               `json:"missing_episodes_count"`
	DuplicatesCount      int               `json:"duplicates_count"`
	Categories           []CategorySummary `json:"categories"`
	Disks                []StorageDisk     `json:"disks"`
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
			&file.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan video file: %w", err)
		}
		file.AudioTracks = json.RawMessage(audioTracks)
		file.Subtitles = json.RawMessage(subtitles)
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

func (r *Repository) ListCategories(ctx context.Context) ([]CategorySummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			c.id,
			c.name_ar,
			c.name_en,
			c.slug,
			COUNT(DISTINCT mi.id) AS media_count,
			COUNT(vf.id) AS file_count
		FROM categories c
		LEFT JOIN media_items mi ON mi.category_id = c.id
		LEFT JOIN video_files vf ON vf.media_item_id = mi.id
		GROUP BY c.id
		ORDER BY c.name_en;
	`)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	defer rows.Close()

	categories := make([]CategorySummary, 0)
	for rows.Next() {
		var category CategorySummary
		var mediaCount, fileCount int64
		if err := rows.Scan(&category.ID, &category.NameAR, &category.NameEN, &category.Slug, &mediaCount, &fileCount); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		category.MediaCount = int(mediaCount)
		category.FileCount = int(fileCount)
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}
	return categories, nil
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
	titleAR := nullableTitleAR(file.Parsed.TitleAR)
	if !titleAR.Valid {
		titleAR = nullableTitleAR(title)
	}
	titleEN := file.Parsed.TitleEN
	if titleEN == "" {
		titleEN = title
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
				status
			)
			VALUES ($1, $2, $3, $4, NULLIF($5, 0), 'completed')
			RETURNING id;
		`, categoryID, titleAR, titleEN, mediaType, file.Parsed.ReleaseYear).Scan(&mediaID)
		if err != nil {
			return fmt.Errorf("insert media item %q: %w", title, err)
		}
	} else if err != nil {
		return fmt.Errorf("find media item %q: %w", title, err)
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
	case strings.Contains(lowerPath, "anime") || strings.Contains(path, "أنمي") || strings.Contains(path, "انمي"):
		return "anime", "anime"
	case strings.Contains(lowerPath, "kids") || strings.Contains(path, "أطفال") || strings.Contains(path, "اطفال") || strings.Contains(lowerPath, "cartoon") || strings.Contains(path, "كرتون") || strings.Contains(path, "رسوم"):
		if parsed.IsEpisode {
			return "kids", "series"
		}
		return "kids", "movie"
	case strings.Contains(lowerPath, "document") || strings.Contains(path, "وثائقي") || strings.Contains(path, "وثائقية") || strings.Contains(path, "وثائقيات"):
		if parsed.IsEpisode {
			return "documentaries", "series"
		}
		return "documentaries", "movie"
	case strings.Contains(lowerPath, "play") || strings.Contains(path, "مسرح") || strings.Contains(path, "مسرحيات") || strings.Contains(path, "مسرحية"):
		return "plays", "movie"
	case strings.Contains(lowerPath, "series") || strings.Contains(path, "مسلسل") || strings.Contains(path, "مسلسلات") || parsed.IsEpisode:
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

func (r *Repository) GetMediaItem(ctx context.Context, id int64) (*MediaItemDetail, error) {
	var item MediaItemDetail
	var titleAR, plotAR, plotEN, posterPath, bannerPath, categorySlug, categoryAR, categoryEN sql.NullString
	var releaseYear sql.NullInt64
	var rating sql.NullFloat64
	var genresText string
	var categoryID sql.NullInt64

	err := r.db.QueryRowContext(ctx, `
		SELECT
			mi.id,
			mi.category_id,
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
			mi.status,
			mi.created_at,
			c.slug,
			c.name_ar,
			c.name_en
		FROM media_items mi
		LEFT JOIN categories c ON c.id = mi.category_id
		WHERE mi.id = $1
	`, id).Scan(
		&item.ID,
		&categoryID,
		&titleAR,
		&item.TitleEN,
		&item.Type,
		&plotAR,
		&plotEN,
		&releaseYear,
		&rating,
		&posterPath,
		&bannerPath,
		&genresText,
		&item.Status,
		&item.CreatedAt,
		&categorySlug,
		&categoryAR,
		&categoryEN,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("get media item %d: %w", id, err)
	}

	if categoryID.Valid {
		item.CategoryID = categoryID.Int64
	}
	item.TitleAR = nullableString(titleAR)
	item.PlotAR = nullableString(plotAR)
	item.PlotEN = nullableString(plotEN)
	item.PosterPath = nullableString(posterPath)
	item.BannerPath = nullableString(bannerPath)
	item.CategorySlug = nullableString(categorySlug)
	item.CategoryAR = nullableString(categoryAR)
	item.CategoryEN = nullableString(categoryEN)
	if releaseYear.Valid {
		item.ReleaseYear = int(releaseYear.Int64)
	}
	if rating.Valid {
		item.Rating = rating.Float64
	}
	_ = json.Unmarshal([]byte(genresText), &item.Genres)

	// Fetch seasons if any
	seasonRows, err := r.db.QueryContext(ctx, `
		SELECT id, season_number, COALESCE(title_ar, ''), COALESCE(title_en, '')
		FROM seasons
		WHERE media_item_id = $1
		ORDER BY season_number ASC;
	`, id)
	if err == nil {
		defer seasonRows.Close()
		seasonsMap := make(map[int64]*SeasonDetail)
		for seasonRows.Next() {
			var s SeasonDetail
			if err := seasonRows.Scan(&s.ID, &s.SeasonNumber, &s.TitleAR, &s.TitleEN); err == nil {
				s.Episodes = make([]VideoFile, 0)
				item.Seasons = append(item.Seasons, s)
			}
		}
		for idx := range item.Seasons {
			seasonsMap[item.Seasons[idx].ID] = &item.Seasons[idx]
		}

		// Fetch video files
		files, fileErr := r.ListVideoFiles(ctx, id)
		if fileErr == nil {
			item.FileCount = len(files)
			for i := range files {
				files[i].StreamURL = fmt.Sprintf("/api/stream/file/%d", files[i].ID)
				if files[i].SeasonID > 0 && seasonsMap[files[i].SeasonID] != nil {
					seasonsMap[files[i].SeasonID].Episodes = append(seasonsMap[files[i].SeasonID].Episodes, files[i])
				} else {
					item.Files = append(item.Files, files[i])
				}
			}
		}
	}

	return &item, nil
}

func (r *Repository) ListMediaItems(ctx context.Context, opts ListMediaOptions) (*MediaListResult, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 24
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}

	whereClauses := []string{"1=1"}
	args := []any{}
	argIdx := 1

	if strings.TrimSpace(opts.CategorySlug) != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("c.slug = $%d", argIdx))
		args = append(args, strings.TrimSpace(opts.CategorySlug))
		argIdx++
	}

	if strings.TrimSpace(opts.Type) != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("mi.type = $%d", argIdx))
		args = append(args, strings.TrimSpace(opts.Type))
		argIdx++
	}

	if strings.TrimSpace(opts.Search) != "" {
		searchTerm := "%" + strings.TrimSpace(opts.Search) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(mi.title_en ILIKE $%d OR mi.title_ar ILIKE $%d)", argIdx, argIdx))
		args = append(args, searchTerm)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// Count total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT mi.id)
		FROM media_items mi
		LEFT JOIN categories c ON c.id = mi.category_id
		WHERE %s;
	`, whereSQL)

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count media items: %w", err)
	}

	orderBy := "mi.created_at DESC, mi.id DESC"
	switch opts.Sort {
	case "rating":
		orderBy = "mi.rating DESC NULLS LAST, mi.id DESC"
	case "year":
		orderBy = "mi.release_year DESC NULLS LAST, mi.id DESC"
	case "title":
		orderBy = "mi.title_en ASC"
	}

	query := fmt.Sprintf(`
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
		WHERE %s
		GROUP BY mi.id, c.id
		ORDER BY %s
		LIMIT $%d OFFSET $%d;
	`, whereSQL, orderBy, argIdx, argIdx+1)

	args = append(args, opts.Limit, opts.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list media items: %w", err)
	}
	defer rows.Close()

	items := make([]search.MediaDocument, 0)
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
			return nil, fmt.Errorf("scan list item: %w", err)
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
		_ = json.Unmarshal([]byte(genresText), &doc.Genres)

		items = append(items, doc)
	}

	return &MediaListResult{
		Total:  total,
		Limit:  opts.Limit,
		Offset: opts.Offset,
		Items:  items,
	}, nil
}

func (r *Repository) UpdateMediaMetadata(ctx context.Context, id int64, meta metadata.Result) (*search.MediaDocument, error) {
	var genresArray any
	if len(meta.Genres) > 0 {
		genresArray = meta.Genres
	}

	poster := meta.CachedPosterPath
	if poster == "" {
		poster = meta.PosterPath
	}
	banner := meta.CachedBannerPath
	if banner == "" {
		banner = meta.BannerPath
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE media_items
		SET
			title_ar = COALESCE(NULLIF($1, ''), title_ar),
			title_en = COALESCE(NULLIF($2, ''), title_en),
			plot_en = COALESCE(NULLIF($3, ''), plot_en),
			release_year = COALESCE(NULLIF($4, 0), release_year),
			rating = COALESCE(NULLIF($5, 0.0), rating),
			poster_path = COALESCE(NULLIF($6, ''), poster_path),
			banner_path = COALESCE(NULLIF($7, ''), banner_path),
			genres = COALESCE($8, genres)
		WHERE id = $9;
	`, meta.Title, meta.Title, meta.Overview, meta.ReleaseYear, meta.Rating, poster, banner, genresArray, id)
	if err != nil {
		return nil, fmt.Errorf("update media metadata: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return nil, sql.ErrNoRows
	}

	// Fetch single document for search reindexing
	docs, err := r.ListSearchDocuments(ctx, 10000)
	if err != nil {
		return nil, err
	}
	for _, doc := range docs {
		if doc.ID == id {
			return &doc, nil
		}
	}
	return nil, nil
}

func (r *Repository) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	var stats DashboardStats

	// Total media
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_items`).Scan(&stats.TotalMedia)

	// Total files and size
	var totalFiles sql.NullInt64
	var totalSize sql.NullInt64
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(file_size), 0) FROM video_files`).Scan(&totalFiles, &totalSize)
	if totalFiles.Valid {
		stats.TotalFiles = totalFiles.Int64
	}
	if totalSize.Valid {
		stats.TotalStorageBytes = totalSize.Int64
	}

	// Categories summary
	categories, err := r.ListCategories(ctx)
	if err == nil {
		stats.Categories = categories
	}

	// Missing episodes count
	missing, err := r.ListMissingEpisodes(ctx)
	if err == nil {
		stats.MissingEpisodesCount = len(missing)
	}

	// Duplicate groups count
	duplicates, err := r.ListDuplicateGroups(ctx)
	if err == nil {
		stats.DuplicatesCount = len(duplicates)
	}

	// Disks
	disks, err := r.ListDisks(ctx)
	if err == nil {
		stats.Disks = disks
	}

	return &stats, nil
}

func (r *Repository) ListDisks(ctx context.Context) ([]StorageDisk, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, disk_letter, COALESCE(disk_label, ''), total_space, free_space, is_active, last_scanned
		FROM storage_disks
		ORDER BY disk_letter ASC;
	`)
	if err != nil {
		return nil, fmt.Errorf("list disks: %w", err)
	}
	defer rows.Close()

	disks := make([]StorageDisk, 0)
	for rows.Next() {
		var disk StorageDisk
		var lastScanned sql.NullTime
		if err := rows.Scan(&disk.ID, &disk.DiskLetter, &disk.DiskLabel, &disk.TotalSpace, &disk.FreeSpace, &disk.IsActive, &lastScanned); err != nil {
			return nil, fmt.Errorf("scan disk: %w", err)
		}
		if lastScanned.Valid {
			disk.LastScanned = &lastScanned.Time
		}
		disk.UsedSpace = disk.TotalSpace - disk.FreeSpace
		if disk.TotalSpace > 0 {
			disk.UsedPercent = float64(disk.UsedSpace) / float64(disk.TotalSpace) * 100.0
		}
		disks = append(disks, disk)
	}
	return disks, nil
}

func (r *Repository) SaveDisks(ctx context.Context, disks []StorageDisk) error {
	for _, disk := range disks {
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO storage_disks (disk_letter, disk_label, total_space, free_space, is_active, last_scanned)
			VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
			ON CONFLICT (disk_letter)
			DO UPDATE SET
				disk_label = EXCLUDED.disk_label,
				total_space = EXCLUDED.total_space,
				free_space = EXCLUDED.free_space,
				is_active = EXCLUDED.is_active,
				last_scanned = CURRENT_TIMESTAMP;
		`, disk.DiskLetter, disk.DiskLabel, disk.TotalSpace, disk.FreeSpace, disk.IsActive)
		if err != nil {
			return fmt.Errorf("save disk %s: %w", disk.DiskLetter, err)
		}
	}
	return nil
}

