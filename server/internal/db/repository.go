package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/lib/pq"

	"nexora/server/internal/media"
	"nexora/server/internal/metadata"
	"nexora/server/internal/scanner"
	"nexora/server/internal/search"
)

type Repository struct {
	db            *sql.DB
	assetImageDir string
}

func NewRepository(database *sql.DB) *Repository {
	return &Repository{db: database}
}

func (r *Repository) SetAssetImageDir(dir string) {
	r.assetImageDir = dir
}

func (r *Repository) CacheLocalArtwork(sourcePath string) string {
	if sourcePath == "" {
		return ""
	}
	if r.assetImageDir == "" {
		return "/api/stream/image?path=" + url.QueryEscape(sourcePath)
	}

	targetDir := filepath.Join(r.assetImageDir, "local")
	_ = os.MkdirAll(targetDir, 0o755)

	ext := strings.ToLower(filepath.Ext(sourcePath))
	if ext == "" {
		ext = ".jpg"
	}
	hash := sha256.Sum256([]byte(filepath.Clean(sourcePath)))
	hashHex := hex.EncodeToString(hash[:])[:16]
	destFileName := "local_" + hashHex + ext
	destPath := filepath.Join(targetDir, destFileName)

	srcStat, err := os.Stat(sourcePath)
	if err != nil {
		return "/api/stream/image?path=" + url.QueryEscape(sourcePath)
	}
	destStat, err := os.Stat(destPath)
	if err != nil || destStat.Size() != srcStat.Size() {
		srcFile, err := os.Open(sourcePath)
		if err == nil {
			defer srcFile.Close()
			destFile, err := os.Create(destPath)
			if err == nil {
				defer destFile.Close()
				_, _ = io.Copy(destFile, srcFile)
			}
		}
	}
	return "/assets/images/local/" + destFileName
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
	ID                    int64           `json:"id"`
	MediaItemID           int64           `json:"media_item_id"`
	SeasonID              int64           `json:"season_id,omitempty"`
	EpisodeNumber         int             `json:"episode_number,omitempty"`
	TitleAR               string          `json:"title_ar,omitempty"`
	TitleEN               string          `json:"title_en,omitempty"`
	FilePath              string          `json:"-"`
	FileSize              int64           `json:"file_size"`
	Duration              int             `json:"duration,omitempty"`
	Resolution            string          `json:"resolution,omitempty"`
	VideoCodec            string          `json:"video_codec,omitempty"`
	AudioTracks           json.RawMessage `json:"audio_tracks,omitempty"`
	Subtitles             json.RawMessage `json:"subtitles,omitempty"`
	VerificationStatus    string          `json:"verification_status,omitempty"`
	VerificationError     string          `json:"verification_error,omitempty"`
	VerificationCheckedAt *time.Time      `json:"verification_checked_at,omitempty"`
	StreamURL             string          `json:"stream_url,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
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

// CorruptedFile is a persisted FFmpeg verification failure for an indexed file.
type CorruptedFile struct {
	ID          int64     `json:"id"`
	MediaItemID int64     `json:"media_item_id"`
	Title       string    `json:"title"`
	FilePath    string    `json:"file_path"`
	Error       string    `json:"error,omitempty"`
	CheckedAt   time.Time `json:"checked_at"`
}

type ChecksumResult struct {
	Scanned int `json:"scanned"`
	Updated int `json:"updated"`
	Failed  int `json:"failed"`
}

// MetadataSnapshot is the locally cached, provider-owned detail document.
// The API exposes this only through the NEXORA server, never directly from TMDB.
type MetadataSnapshot struct {
	Provider   string          `json:"provider"`
	ExternalID string          `json:"externalId"`
	Locale     string          `json:"locale"`
	Payload    json.RawMessage `json:"payload"`
	FetchedAt  time.Time       `json:"fetchedAt"`
	ExpiresAt  time.Time       `json:"expiresAt"`
}

// SeasonMetadataSnapshot retains a whole TMDB season response locally. Its
// payload includes the provider's episode list and season-level extras.
type SeasonMetadataSnapshot struct {
	Provider     string          `json:"provider"`
	ExternalID   string          `json:"externalId"`
	Locale       string          `json:"locale"`
	SeasonNumber int             `json:"seasonNumber"`
	Payload      json.RawMessage `json:"payload"`
	FetchedAt    time.Time       `json:"fetchedAt"`
	ExpiresAt    time.Time       `json:"expiresAt"`
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

// mediaCardSummary is deliberately returned with every catalogue/search item.
// It is calculated from persisted database data only; browser card rendering
// never needs to contact TMDB or inspect individual files.
type mediaCardSummary struct {
	Status             string
	SeasonCount        int
	TMDBSeasonCount    int
	TMDBEpisodeCount   int
	TotalSize          int64
	BestResolution     string
	RuntimeMinutes     int
	HasArabicAudio     bool
	HasArabicSubtitles bool
}

type ListMediaOptions struct {
	CategorySlug string
	Type         string
	Types        []string
	Categories   []string
	TagsAny      []string
	YearFrom     int
	YearTo       int
	RatingGTE    float64
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

// ShowcaseOptions selects locally persisted editorial collections and media
// summaries for the reusable hero. It never consults a remote metadata source.
type ShowcaseOptions struct {
	Context      string
	CategorySlug string
	Limit        int
}

type ShowcaseTarget struct {
	Category string          `json:"category,omitempty"`
	Filters  json.RawMessage `json:"filters,omitempty"`
}

type ShowcaseSlide struct {
	ID              string          `json:"id"`
	Kind            string          `json:"kind"`
	MediaID         int64           `json:"media_id,omitempty"`
	TitleAR         string          `json:"title_ar,omitempty"`
	TitleEN         string          `json:"title_en,omitempty"`
	DescriptionAR   string          `json:"description_ar,omitempty"`
	DescriptionEN   string          `json:"description_en,omitempty"`
	ArtworkPath     string          `json:"artwork_path,omitempty"`
	ArtworkPosition string          `json:"artwork_position,omitempty"`
	Accent          string          `json:"accent,omitempty"`
	ItemCount       int             `json:"item_count,omitempty"`
	Type            string          `json:"type,omitempty"`
	Status          string          `json:"status,omitempty"`
	ReleaseYear     int             `json:"release_year,omitempty"`
	Rating          float64         `json:"rating,omitempty"`
	BestResolution  string          `json:"best_resolution,omitempty"`
	Genres          []string        `json:"genres,omitempty"`
	Target          *ShowcaseTarget `json:"target,omitempty"`
}

type ShowcaseResult struct {
	Context string          `json:"context"`
	Slides  []ShowcaseSlide `json:"slides"`
}

type Collection struct {
	ID                 int64           `json:"id"`
	Slug               string          `json:"slug"`
	TitleAR            string          `json:"title_ar"`
	TitleEN            string          `json:"title_en"`
	DescriptionAR      string          `json:"description_ar"`
	DescriptionEN      string          `json:"description_en"`
	ArtworkPath        string          `json:"artwork_path"`
	ArtworkPosition    string          `json:"artwork_position"`
	Accent             string          `json:"accent"`
	TargetCategorySlug string          `json:"target_category_slug"`
	TargetFilters      json.RawMessage `json:"target_filters"`
	Priority           int             `json:"priority"`
	IsActive           bool            `json:"is_active"`
	ItemCount          int             `json:"item_count"`
}

type CollectionRequest struct {
	Slug               string          `json:"slug"`
	TitleAR            string          `json:"title_ar"`
	TitleEN            string          `json:"title_en"`
	DescriptionAR      string          `json:"description_ar"`
	DescriptionEN      string          `json:"description_en"`
	ArtworkPath        string          `json:"artwork_path"`
	ArtworkPosition    string          `json:"artwork_position"`
	Accent             string          `json:"accent"`
	TargetCategorySlug string          `json:"target_category_slug"`
	TargetFilters      json.RawMessage `json:"target_filters"`
	Priority           int             `json:"priority"`
	IsActive           bool            `json:"is_active"`
}

type HubRule struct {
	Types      []string `json:"types,omitempty"`
	Categories []string `json:"categories,omitempty"`
	TagsAny    []string `json:"tags_any,omitempty"`
	YearFrom   int      `json:"year_from,omitempty"`
	YearTo     int      `json:"year_to,omitempty"`
	RatingGTE  float64  `json:"rating_gte,omitempty"`
}

type SmartHub struct {
	ID              string   `json:"id"`
	Slug            string   `json:"slug"`
	Source          string   `json:"source"`
	Scope           string   `json:"scope"`
	TitleAR         string   `json:"title_ar"`
	TitleEN         string   `json:"title_en"`
	DescriptionAR   string   `json:"description_ar"`
	DescriptionEN   string   `json:"description_en"`
	ArtworkPath     string   `json:"artwork_path"`
	ArtworkPosition string   `json:"artwork_position"`
	Accent          string   `json:"accent"`
	Icon            string   `json:"icon"`
	Rule            HubRule  `json:"rule"`
	Priority        int      `json:"priority"`
	ItemCount       int      `json:"item_count"`
	PreviewArtwork  []string `json:"preview_artwork,omitempty"`
	IsActive        bool     `json:"is_active"`
	MinItemCount    int      `json:"min_item_count"`
}

type SmartHubRequest struct {
	Slug            string  `json:"slug"`
	Scope           string  `json:"scope"`
	TitleAR         string  `json:"title_ar"`
	TitleEN         string  `json:"title_en"`
	DescriptionAR   string  `json:"description_ar"`
	DescriptionEN   string  `json:"description_en"`
	ArtworkPath     string  `json:"artwork_path"`
	ArtworkPosition string  `json:"artwork_position"`
	Accent          string  `json:"accent"`
	Icon            string  `json:"icon"`
	Rule            HubRule `json:"rule"`
	Priority        int     `json:"priority"`
	IsActive        bool    `json:"is_active"`
	MinItemCount    int     `json:"min_item_count"`
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
	CorruptedFilesCount  int64             `json:"corrupted_files_count"`
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
		WITH file_summary AS (
			SELECT media_item_id, COUNT(*)::int AS file_count, COALESCE(SUM(file_size), 0)::bigint AS total_size,
				MAX(duration) FILTER (WHERE episode_number IS NULL)::int / 60 AS local_runtime_minutes,
				CASE MAX(CASE WHEN resolution ~* '(2160|4k)' THEN 4 WHEN resolution ~* '1440' THEN 3 WHEN resolution ~* '1080' THEN 2 WHEN resolution ~* '720' THEN 1 ELSE 0 END)
					WHEN 4 THEN '4K' WHEN 3 THEN '1440p' WHEN 2 THEN '1080p' WHEN 1 THEN '720p' ELSE '' END AS best_resolution,
				BOOL_OR(LOWER(COALESCE(audio_tracks::text, '')) LIKE '%"language":"ara"%' OR LOWER(COALESCE(audio_tracks::text, '')) LIKE '%"language":"ar"%') AS has_arabic_audio,
				BOOL_OR(LOWER(COALESCE(subtitles::text, '')) LIKE '%"language":"ara"%' OR LOWER(COALESCE(subtitles::text, '')) LIKE '%"language":"ar"%') AS has_arabic_subtitles
			FROM video_files GROUP BY media_item_id
		), season_summary AS (
			SELECT media_item_id, COUNT(*)::int AS season_count FROM seasons GROUP BY media_item_id
		)
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
			COALESCE(fs.file_count, 0), mi.status, COALESCE(ss.season_count, 0),
			COALESCE(NULLIF(mi.metadata_facets->>'number_of_seasons', '')::int, 0), COALESCE(NULLIF(mi.metadata_facets->>'number_of_episodes', '')::int, 0), COALESCE(fs.total_size, 0),
			COALESCE(fs.best_resolution, ''), COALESCE(NULLIF(mi.metadata_facets->>'runtime', '')::int, fs.local_runtime_minutes, 0),
			COALESCE(fs.has_arabic_audio, false), COALESCE(fs.has_arabic_subtitles, false)
		FROM media_items mi
		LEFT JOIN categories c ON c.id = mi.category_id
		LEFT JOIN file_summary fs ON fs.media_item_id = mi.id
		LEFT JOIN season_summary ss ON ss.media_item_id = mi.id
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
		var fileCount int
		var summary mediaCardSummary

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
			&fileCount, &summary.Status, &summary.SeasonCount, &summary.TMDBSeasonCount, &summary.TMDBEpisodeCount, &summary.TotalSize, &summary.BestResolution,
			&summary.RuntimeMinutes, &summary.HasArabicAudio, &summary.HasArabicSubtitles,
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
		doc.FileCount = fileCount
		doc.Status = summary.Status
		doc.SeasonCount = summary.SeasonCount
		doc.TMDBSeasonCount = summary.TMDBSeasonCount
		doc.TMDBEpisodeCount = summary.TMDBEpisodeCount
		doc.TotalSize = summary.TotalSize
		doc.BestResolution = summary.BestResolution
		doc.RuntimeMinutes = summary.RuntimeMinutes
		doc.HasArabicAudio = summary.HasArabicAudio
		doc.HasArabicSubtitles = summary.HasArabicSubtitles
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

func (r *Repository) CreateCategory(ctx context.Context, nameAR, nameEN, slug string) (*CategorySummary, error) {
	nameAR = strings.TrimSpace(nameAR)
	nameEN = strings.TrimSpace(nameEN)
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(nameEN, " ", "-"))
	}
	if nameEN == "" {
		nameEN = nameAR
	}
	if nameAR == "" {
		nameAR = nameEN
	}

	var newID int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO categories (name_ar, name_en, slug)
		VALUES ($1, $2, $3)
		ON CONFLICT (slug) DO UPDATE SET name_ar = EXCLUDED.name_ar, name_en = EXCLUDED.name_en
		RETURNING id;
	`, nameAR, nameEN, slug).Scan(&newID)
	if err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}

	return &CategorySummary{
		ID:         newID,
		NameAR:     nameAR,
		NameEN:     nameEN,
		Slug:       slug,
		MediaCount: 0,
		FileCount:  0,
	}, nil
}

func (r *Repository) UpdateCategory(ctx context.Context, id int64, nameAR, nameEN, slug string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE categories
		SET
			name_ar = COALESCE(NULLIF($1, ''), name_ar),
			name_en = COALESCE(NULLIF($2, ''), name_en),
			slug = COALESCE(NULLIF($3, ''), slug)
		WHERE id = $4;
	`, strings.TrimSpace(nameAR), strings.TrimSpace(nameEN), strings.ToLower(strings.TrimSpace(slug)), id)
	return err
}

func (r *Repository) DeleteCategory(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id = $1`, id)
	return err
}

func (r *Repository) ListCollections(ctx context.Context) ([]Collection, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT c.id,c.slug,COALESCE(c.title_ar,''),c.title_en,COALESCE(c.description_ar,''),COALESCE(c.description_en,''),COALESCE(c.artwork_path,''),c.artwork_position,c.accent,COALESCE(c.target_category_slug,''),c.target_filters::text,c.priority,c.is_active,COUNT(ci.media_item_id)::int FROM collections c LEFT JOIN collection_items ci ON ci.collection_id=c.id GROUP BY c.id ORDER BY c.priority DESC,c.id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}
	defer rows.Close()
	items := []Collection{}
	for rows.Next() {
		var item Collection
		var filters string
		if err := rows.Scan(&item.ID, &item.Slug, &item.TitleAR, &item.TitleEN, &item.DescriptionAR, &item.DescriptionEN, &item.ArtworkPath, &item.ArtworkPosition, &item.Accent, &item.TargetCategorySlug, &filters, &item.Priority, &item.IsActive, &item.ItemCount); err != nil {
			return nil, err
		}
		item.TargetFilters = json.RawMessage(filters)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveCollection(ctx context.Context, id int64, req CollectionRequest) (*Collection, error) {
	if strings.TrimSpace(req.TitleEN) == "" {
		req.TitleEN = req.TitleAR
	}
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	if req.Slug == "" || strings.TrimSpace(req.TitleAR) == "" {
		return nil, errors.New("hub slug and Arabic title are required")
	}
	if req.Slug == "" {
		req.Slug = strings.ReplaceAll(strings.ToLower(req.TitleEN), " ", "-")
	}
	if len(req.TargetFilters) == 0 {
		req.TargetFilters = json.RawMessage(`{}`)
	}
	var saved Collection
	query := `INSERT INTO collections (slug,title_ar,title_en,description_ar,description_en,artwork_path,artwork_position,accent,target_category_slug,target_filters,priority,is_active) VALUES ($1,$2,$3,$4,$5,$6,COALESCE(NULLIF($7,''),'center center'),COALESCE(NULLIF($8,''),'violet'),NULLIF($9,''),$10::jsonb,$11,$12) ON CONFLICT (slug) DO UPDATE SET title_ar=EXCLUDED.title_ar,title_en=EXCLUDED.title_en,description_ar=EXCLUDED.description_ar,description_en=EXCLUDED.description_en,artwork_path=EXCLUDED.artwork_path,artwork_position=EXCLUDED.artwork_position,accent=EXCLUDED.accent,target_category_slug=EXCLUDED.target_category_slug,target_filters=EXCLUDED.target_filters,priority=EXCLUDED.priority,is_active=EXCLUDED.is_active,updated_at=CURRENT_TIMESTAMP RETURNING id,slug,title_ar,title_en,description_ar,description_en,artwork_path,artwork_position,accent,COALESCE(target_category_slug,''),target_filters::text,priority,is_active`
	if id > 0 {
		query = `UPDATE collections SET slug=$1,title_ar=$2,title_en=$3,description_ar=$4,description_en=$5,artwork_path=$6,artwork_position=COALESCE(NULLIF($7,''),'center center'),accent=COALESCE(NULLIF($8,''),'violet'),target_category_slug=NULLIF($9,''),target_filters=$10::jsonb,priority=$11,is_active=$12,updated_at=CURRENT_TIMESTAMP WHERE id=` + fmt.Sprint(id) + ` RETURNING id,slug,title_ar,title_en,description_ar,description_en,artwork_path,artwork_position,accent,COALESCE(target_category_slug,''),target_filters::text,priority,is_active`
	}
	var filters string
	err := r.db.QueryRowContext(ctx, query, req.Slug, req.TitleAR, req.TitleEN, req.DescriptionAR, req.DescriptionEN, req.ArtworkPath, req.ArtworkPosition, req.Accent, req.TargetCategorySlug, string(req.TargetFilters), req.Priority, req.IsActive).Scan(&saved.ID, &saved.Slug, &saved.TitleAR, &saved.TitleEN, &saved.DescriptionAR, &saved.DescriptionEN, &saved.ArtworkPath, &saved.ArtworkPosition, &saved.Accent, &saved.TargetCategorySlug, &filters, &saved.Priority, &saved.IsActive)
	if err != nil {
		return nil, err
	}
	saved.TargetFilters = json.RawMessage(filters)
	return &saved, nil
}

func (r *Repository) DeleteCollection(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM collections WHERE id=$1`, id)
	return err
}

func (r *Repository) ListSmartHubsAdmin(ctx context.Context) ([]SmartHub, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT slug,source,scope,title_ar,COALESCE(title_en,''),COALESCE(description_ar,''),COALESCE(description_en,''),COALESCE(artwork_path,''),artwork_position,accent,icon,rule::text,priority,is_active,min_item_count FROM hub_definitions ORDER BY priority DESC,id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SmartHub{}
	for rows.Next() {
		var h SmartHub
		var rule string
		if err := rows.Scan(&h.Slug, &h.Source, &h.Scope, &h.TitleAR, &h.TitleEN, &h.DescriptionAR, &h.DescriptionEN, &h.ArtworkPath, &h.ArtworkPosition, &h.Accent, &h.Icon, &rule, &h.Priority, &h.IsActive, &h.MinItemCount); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(rule), &h.Rule); err != nil {
			return nil, err
		}
		h.ID = h.Slug
		items = append(items, h)
	}
	return items, rows.Err()
}
func (r *Repository) SaveSmartHub(ctx context.Context, slug string, req SmartHubRequest) (*SmartHub, error) {
	if strings.TrimSpace(slug) != "" {
		req.Slug = slug
	}
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	if req.MinItemCount < 1 {
		req.MinItemCount = 1
	}
	rule, err := json.Marshal(req.Rule)
	if err != nil {
		return nil, err
	}
	var h SmartHub
	var raw string
	err = r.db.QueryRowContext(ctx, `INSERT INTO hub_definitions (slug,source,scope,title_ar,title_en,description_ar,description_en,artwork_path,artwork_position,accent,icon,rule,priority,is_active,min_item_count) VALUES ($1,'editorial',COALESCE(NULLIF($2,''),'all'),$3,$4,$5,$6,$7,COALESCE(NULLIF($8,''),'center center'),COALESCE(NULLIF($9,''),'violet'),COALESCE(NULLIF($10,''),'spark'),$11::jsonb,$12,$13,$14) ON CONFLICT (slug) DO UPDATE SET scope=EXCLUDED.scope,title_ar=EXCLUDED.title_ar,title_en=EXCLUDED.title_en,description_ar=EXCLUDED.description_ar,description_en=EXCLUDED.description_en,artwork_path=EXCLUDED.artwork_path,artwork_position=EXCLUDED.artwork_position,accent=EXCLUDED.accent,icon=EXCLUDED.icon,rule=EXCLUDED.rule,priority=EXCLUDED.priority,is_active=EXCLUDED.is_active,min_item_count=EXCLUDED.min_item_count,updated_at=CURRENT_TIMESTAMP RETURNING slug,source,scope,title_ar,COALESCE(title_en,''),COALESCE(description_ar,''),COALESCE(description_en,''),COALESCE(artwork_path,''),artwork_position,accent,icon,rule::text,priority,is_active,min_item_count`, req.Slug, req.Scope, req.TitleAR, req.TitleEN, req.DescriptionAR, req.DescriptionEN, req.ArtworkPath, req.ArtworkPosition, req.Accent, req.Icon, string(rule), req.Priority, req.IsActive, req.MinItemCount).Scan(&h.Slug, &h.Source, &h.Scope, &h.TitleAR, &h.TitleEN, &h.DescriptionAR, &h.DescriptionEN, &h.ArtworkPath, &h.ArtworkPosition, &h.Accent, &h.Icon, &raw, &h.Priority, &h.IsActive, &h.MinItemCount)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(raw), &h.Rule)
	h.ID = h.Slug
	return &h, nil
}

func (r *Repository) ListSmartHubs(ctx context.Context, scope string) ([]SmartHub, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT slug,source,scope,title_ar,COALESCE(title_en,''),COALESCE(description_ar,''),COALESCE(description_en,''),COALESCE(artwork_path,''),artwork_position,accent,icon,rule::text,priority,is_active,min_item_count FROM hub_definitions WHERE is_active=true AND ($1='' OR scope='all' OR scope=$1) ORDER BY priority DESC,id DESC`, strings.TrimSpace(scope))
	if err != nil {
		return nil, fmt.Errorf("list smart hubs: %w", err)
	}
	defer rows.Close()
	hubs := []SmartHub{}
	for rows.Next() {
		var h SmartHub
		var ruleText string
		var minItemCount int
		if err := rows.Scan(&h.Slug, &h.Source, &h.Scope, &h.TitleAR, &h.TitleEN, &h.DescriptionAR, &h.DescriptionEN, &h.ArtworkPath, &h.ArtworkPosition, &h.Accent, &h.Icon, &ruleText, &h.Priority, &h.IsActive, &minItemCount); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(ruleText), &h.Rule); err != nil {
			return nil, fmt.Errorf("decode hub rule %s: %w", h.Slug, err)
		}
		h.ID = h.Slug
		h.MinItemCount = minItemCount
		preview, err := r.ListMediaItems(ctx, listOptionsFromHubRule(h.Rule, ListMediaOptions{Limit: 3, Sort: "rating"}))
		if err != nil {
			return nil, err
		}
		h.ItemCount = preview.Total
		h.PreviewArtwork = hubPreviewArtwork(preview.Items)
		if h.ItemCount >= minItemCount {
			hubs = append(hubs, h)
		}
	}
	return hubs, rows.Err()
}

func (r *Repository) GetSmartHub(ctx context.Context, slug string) (*SmartHub, error) {
	var h SmartHub
	var ruleText string
	var minItemCount int
	err := r.db.QueryRowContext(ctx, `SELECT slug,source,scope,title_ar,COALESCE(title_en,''),COALESCE(description_ar,''),COALESCE(description_en,''),COALESCE(artwork_path,''),artwork_position,accent,icon,rule::text,priority,min_item_count FROM hub_definitions WHERE slug=$1 AND is_active=true`, strings.TrimSpace(slug)).Scan(&h.Slug, &h.Source, &h.Scope, &h.TitleAR, &h.TitleEN, &h.DescriptionAR, &h.DescriptionEN, &h.ArtworkPath, &h.ArtworkPosition, &h.Accent, &h.Icon, &ruleText, &h.Priority, &minItemCount)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(ruleText), &h.Rule); err != nil {
		return nil, err
	}
	h.ID = h.Slug
	count, err := r.ListMediaItems(ctx, listOptionsFromHubRule(h.Rule, ListMediaOptions{Limit: 3, Sort: "rating"}))
	if err != nil {
		return nil, err
	}
	h.ItemCount = count.Total
	h.PreviewArtwork = hubPreviewArtwork(count.Items)
	if h.ItemCount < minItemCount {
		return nil, sql.ErrNoRows
	}
	return &h, nil
}

func (r *Repository) ListSmartHubMedia(ctx context.Context, slug string, opts ListMediaOptions) (*MediaListResult, *SmartHub, error) {
	hub, err := r.GetSmartHub(ctx, slug)
	if err != nil {
		return nil, nil, err
	}
	result, err := r.ListMediaItems(ctx, listOptionsFromHubRule(hub.Rule, opts))
	if err != nil {
		return nil, nil, err
	}
	return result, hub, nil
}

func listOptionsFromHubRule(rule HubRule, opts ListMediaOptions) ListMediaOptions {
	opts.Types = rule.Types
	opts.Categories = rule.Categories
	opts.TagsAny = rule.TagsAny
	opts.YearFrom = rule.YearFrom
	opts.YearTo = rule.YearTo
	opts.RatingGTE = rule.RatingGTE
	return opts
}

func hubPreviewArtwork(items []search.MediaDocument) []string {
	paths := make([]string, 0, len(items))
	for _, item := range items {
		path := item.BannerPath
		if path == "" {
			path = item.PosterPath
		}
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

func countHubMatches(items []search.MediaDocument, rule HubRule) int {
	count := 0
	for _, item := range items {
		if matchesHubRule(item, rule) {
			count++
		}
	}
	return count
}
func matchesHubRule(item search.MediaDocument, rule HubRule) bool {
	in := func(values []string, value string) bool {
		for _, v := range values {
			if strings.EqualFold(strings.TrimSpace(v), strings.TrimSpace(value)) {
				return true
			}
		}
		return false
	}
	if len(rule.Types) > 0 && !in(rule.Types, item.Type) {
		return false
	}
	if len(rule.Categories) > 0 && !in(rule.Categories, item.CategorySlug) {
		return false
	}
	if rule.YearFrom > 0 && item.ReleaseYear < rule.YearFrom {
		return false
	}
	if rule.YearTo > 0 && item.ReleaseYear > rule.YearTo {
		return false
	}
	if rule.RatingGTE > 0 && item.Rating < rule.RatingGTE {
		return false
	}
	if len(rule.TagsAny) > 0 {
		found := false
		for _, tag := range item.Genres {
			for _, wanted := range rule.TagsAny {
				if strings.EqualFold(strings.TrimSpace(tag), strings.TrimSpace(wanted)) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
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
	if opts.Limit <= 0 || opts.Limit > 10000 {
		opts.Limit = 1000
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
	if len(opts.Types) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("mi.type = ANY($%d::text[])", argIdx))
		args = append(args, pq.Array(opts.Types))
		argIdx++
	}
	if len(opts.Categories) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("c.slug = ANY($%d::text[])", argIdx))
		args = append(args, pq.Array(opts.Categories))
		argIdx++
	}
	if len(opts.TagsAny) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("COALESCE(mi.genres::text[], ARRAY[]::text[]) && $%d::text[]", argIdx))
		args = append(args, pq.Array(opts.TagsAny))
		argIdx++
	}
	if opts.YearFrom > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("mi.release_year >= $%d", argIdx))
		args = append(args, opts.YearFrom)
		argIdx++
	}
	if opts.YearTo > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("mi.release_year <= $%d", argIdx))
		args = append(args, opts.YearTo)
		argIdx++
	}
	if opts.RatingGTE > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("mi.rating >= $%d", argIdx))
		args = append(args, opts.RatingGTE)
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
		WITH file_summary AS (
			SELECT media_item_id, COUNT(*)::int AS file_count, COALESCE(SUM(file_size), 0)::bigint AS total_size,
				MAX(duration) FILTER (WHERE episode_number IS NULL)::int / 60 AS local_runtime_minutes,
				CASE MAX(CASE WHEN resolution ~* '(2160|4k)' THEN 4 WHEN resolution ~* '1440' THEN 3 WHEN resolution ~* '1080' THEN 2 WHEN resolution ~* '720' THEN 1 ELSE 0 END)
					WHEN 4 THEN '4K' WHEN 3 THEN '1440p' WHEN 2 THEN '1080p' WHEN 1 THEN '720p' ELSE '' END AS best_resolution,
				BOOL_OR(LOWER(COALESCE(audio_tracks::text, '')) LIKE '%%"language":"ara"%%' OR LOWER(COALESCE(audio_tracks::text, '')) LIKE '%%"language":"ar"%%') AS has_arabic_audio,
				BOOL_OR(LOWER(COALESCE(subtitles::text, '')) LIKE '%%"language":"ara"%%' OR LOWER(COALESCE(subtitles::text, '')) LIKE '%%"language":"ar"%%') AS has_arabic_subtitles
			FROM video_files GROUP BY media_item_id
		), season_summary AS (
			SELECT media_item_id, COUNT(*)::int AS season_count FROM seasons GROUP BY media_item_id
		)
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
			COALESCE(fs.file_count, 0), mi.status, COALESCE(ss.season_count, 0),
			COALESCE(NULLIF(mi.metadata_facets->>'number_of_seasons', '')::int, 0), COALESCE(NULLIF(mi.metadata_facets->>'number_of_episodes', '')::int, 0), COALESCE(fs.total_size, 0),
			COALESCE(fs.best_resolution, ''), COALESCE(NULLIF(mi.metadata_facets->>'runtime', '')::int, fs.local_runtime_minutes, 0),
			COALESCE(fs.has_arabic_audio, false), COALESCE(fs.has_arabic_subtitles, false)
		FROM media_items mi
		LEFT JOIN categories c ON c.id = mi.category_id
		LEFT JOIN file_summary fs ON fs.media_item_id = mi.id
		LEFT JOIN season_summary ss ON ss.media_item_id = mi.id
		WHERE %s
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
		var fileCount int
		var summary mediaCardSummary

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
			&fileCount, &summary.Status, &summary.SeasonCount, &summary.TMDBSeasonCount, &summary.TMDBEpisodeCount, &summary.TotalSize, &summary.BestResolution,
			&summary.RuntimeMinutes, &summary.HasArabicAudio, &summary.HasArabicSubtitles,
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
		doc.FileCount = fileCount
		doc.Status = summary.Status
		doc.SeasonCount = summary.SeasonCount
		doc.TMDBSeasonCount = summary.TMDBSeasonCount
		doc.TMDBEpisodeCount = summary.TMDBEpisodeCount
		doc.TotalSize = summary.TotalSize
		doc.BestResolution = summary.BestResolution
		doc.RuntimeMinutes = summary.RuntimeMinutes
		doc.HasArabicAudio = summary.HasArabicAudio
		doc.HasArabicSubtitles = summary.HasArabicSubtitles
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

// ListShowcases returns editorial collections followed by featured local works.
// Both layers are read from PostgreSQL; no remote metadata is contacted here.
func (r *Repository) ListShowcases(ctx context.Context, opts ShowcaseOptions) (*ShowcaseResult, error) {
	if opts.Limit <= 0 || opts.Limit > 12 {
		opts.Limit = 6
	}
	categorySlug := strings.TrimSpace(opts.CategorySlug)
	result := &ShowcaseResult{Context: strings.TrimSpace(opts.Context), Slides: make([]ShowcaseSlide, 0, opts.Limit)}

	rows, err := r.db.QueryContext(ctx, `
		SELECT c.slug, COALESCE(c.title_ar, ''), c.title_en, COALESCE(c.description_ar, ''), COALESCE(c.description_en, ''),
			COALESCE(c.artwork_path, ''), c.artwork_position, c.accent,
			COUNT(ci.media_item_id)::int, COALESCE(c.target_category_slug, ''), c.target_filters::text
		FROM collections c
		LEFT JOIN collection_items ci ON ci.collection_id = c.id
		WHERE c.is_active = true
			AND ($1 = '' OR c.target_category_slug = '' OR c.target_category_slug = $1)
		GROUP BY c.id
		ORDER BY c.priority DESC, c.id DESC
		LIMIT $2`, categorySlug, opts.Limit)
	if err != nil {
		return nil, fmt.Errorf("list showcase collections: %w", err)
	}
	for rows.Next() {
		var slug, titleAR, titleEN, descriptionAR, descriptionEN, artworkPath, artworkPosition, accent, targetCategory, filtersText string
		var itemCount int
		if err := rows.Scan(&slug, &titleAR, &titleEN, &descriptionAR, &descriptionEN, &artworkPath, &artworkPosition, &accent, &itemCount, &targetCategory, &filtersText); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan showcase collection: %w", err)
		}
		result.Slides = append(result.Slides, ShowcaseSlide{
			ID: "collection-" + slug, Kind: "collection", TitleAR: titleAR, TitleEN: titleEN,
			DescriptionAR: descriptionAR, DescriptionEN: descriptionEN, ArtworkPath: artworkPath,
			ArtworkPosition: artworkPosition, Accent: accent, ItemCount: itemCount,
			Target: &ShowcaseTarget{Category: targetCategory, Filters: json.RawMessage(filtersText)},
		})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate showcase collections: %w", err)
	}
	rows.Close()

	remaining := opts.Limit - len(result.Slides)
	if remaining <= 0 {
		return result, nil
	}
	mediaResult, err := r.ListMediaItems(ctx, ListMediaOptions{CategorySlug: categorySlug, Sort: "rating", Limit: remaining})
	if err != nil {
		return nil, err
	}
	for _, item := range mediaResult.Items {
		artwork := item.BannerPath
		if artwork == "" {
			artwork = item.PosterPath
		}
		result.Slides = append(result.Slides, ShowcaseSlide{
			ID: fmt.Sprintf("media-%d", item.ID), Kind: "featured", MediaID: item.ID,
			TitleAR: item.TitleAR, TitleEN: item.TitleEN, DescriptionAR: item.PlotAR,
			DescriptionEN: item.PlotEN, ArtworkPath: artwork, ArtworkPosition: "center center",
			Accent: "violet", Type: item.Type, Status: item.Status, ReleaseYear: item.ReleaseYear,
			Rating: item.Rating, BestResolution: item.BestResolution, Genres: item.Genres,
		})
	}

	return result, nil
}

func (r *Repository) UpdateMediaMetadata(ctx context.Context, id int64, meta metadata.Result) (*search.MediaDocument, error) {
	var genresArray any
	if len(meta.Genres) > 0 {
		genresArray = pq.Array(meta.Genres)
	}

	poster := meta.CachedPosterPath
	if poster == "" {
		poster = meta.PosterPath
	}
	banner := meta.CachedBannerPath
	if banner == "" {
		banner = meta.BannerPath
	}
	titleAR, titleEN, plotAR, plotEN := localizedMetadataFields(meta)
	metadataFacets := metadataFacetsJSON(meta)
	status := metadataStatus(meta)

	result, err := r.db.ExecContext(ctx, `
		UPDATE media_items
		SET
			title_ar = COALESCE(NULLIF($1, ''), title_ar),
			title_en = COALESCE(NULLIF($2, ''), title_en),
			plot_ar = COALESCE(NULLIF($3, ''), plot_ar),
			plot_en = COALESCE(NULLIF($4, ''), plot_en),
			release_year = COALESCE(NULLIF($5, 0), release_year),
			rating = COALESCE(NULLIF($6, 0.0), rating),
			poster_path = COALESCE(NULLIF($7, ''), poster_path),
			banner_path = COALESCE(NULLIF($8, ''), banner_path),
			genres = COALESCE($9, genres),
			metadata_provider = COALESCE(NULLIF($10, ''), metadata_provider),
			metadata_external_id = COALESCE(NULLIF($11, ''), metadata_external_id),
			metadata_facets = COALESCE($12::jsonb, metadata_facets),
			status = COALESCE(NULLIF($13, ''), status),
			metadata_fetched_at = CASE WHEN NULLIF($10, '') IS NULL THEN metadata_fetched_at ELSE CURRENT_TIMESTAMP END,
			metadata_expires_at = CASE WHEN NULLIF($10, '') IS NULL THEN metadata_expires_at ELSE CURRENT_TIMESTAMP + INTERVAL '30 days' END
		WHERE id = $14;
	`, titleAR, titleEN, plotAR, plotEN, meta.ReleaseYear, meta.Rating, poster, banner, genresArray, meta.Provider, meta.ExternalID, metadataFacets, status, id)
	if err != nil {
		return nil, fmt.Errorf("update media metadata: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return nil, sql.ErrNoRows
	}
	if len(meta.RawPayload) > 0 && meta.Provider != "" && meta.ExternalID != "" {
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO metadata_snapshots (media_item_id, provider, external_id, locale, raw_payload, expires_at)
			VALUES ($1, $2, $3, $4, $5::jsonb, CURRENT_TIMESTAMP + INTERVAL '30 days')
			ON CONFLICT (media_item_id, provider, locale)
			DO UPDATE SET external_id = EXCLUDED.external_id,
				raw_payload = EXCLUDED.raw_payload,
				fetched_at = CURRENT_TIMESTAMP,
				expires_at = EXCLUDED.expires_at
		`, id, meta.Provider, meta.ExternalID, firstNonEmptyLocale(meta.Locale), string(meta.RawPayload)); err != nil {
			return nil, fmt.Errorf("cache metadata snapshot: %w", err)
		}
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

// localizedMetadataFields prevents a locale-specific TMDB response from
// overwriting the other title/plot column. This is important for records where
// TMDB falls back to a third language when Arabic text is unavailable.
func localizedMetadataFields(meta metadata.Result) (titleAR, titleEN, plotAR, plotEN string) {
	locale := strings.ToLower(strings.TrimSpace(meta.Locale))
	if strings.HasPrefix(locale, "ar") {
		return meta.Title, "", meta.Overview, ""
	}
	return "", meta.Title, "", meta.Overview
}

// metadataFacetsJSON extracts stable, filterable English attributes from the
// complete TMDB document. The full source document remains in
// metadata_snapshots; this small JSONB projection is indexed on media_items.
func metadataFacetsJSON(meta metadata.Result) any {
	if !strings.HasPrefix(strings.ToLower(meta.Locale), "en") || len(meta.RawPayload) == 0 {
		return nil
	}
	var raw map[string]any
	if err := json.Unmarshal(meta.RawPayload, &raw); err != nil {
		return nil
	}
	mediaType := "movie"
	if raw["name"] != nil && raw["title"] == nil {
		mediaType = "tv"
	}
	facets := map[string]any{
		"provider":              meta.Provider,
		"external_id":           meta.ExternalID,
		"type":                  mediaType,
		"title":                 raw["title"],
		"original_title":        raw["original_title"],
		"original_language":     raw["original_language"],
		"release_date":          raw["release_date"],
		"runtime":               raw["runtime"],
		"status":                raw["status"],
		"number_of_seasons":     raw["number_of_seasons"],
		"number_of_episodes":    raw["number_of_episodes"],
		"episode_run_time":      raw["episode_run_time"],
		"adult":                 raw["adult"],
		"popularity":            raw["popularity"],
		"vote_average":          raw["vote_average"],
		"vote_count":            raw["vote_count"],
		"genres":                raw["genres"],
		"keywords":              raw["keywords"],
		"production_companies":  raw["production_companies"],
		"production_countries":  raw["production_countries"],
		"spoken_languages":      raw["spoken_languages"],
		"belongs_to_collection": raw["belongs_to_collection"],
	}
	encoded, err := json.Marshal(facets)
	if err != nil {
		return nil
	}
	return string(encoded)
}

// metadataStatus turns TMDB's provider status into one of the UI's stable
// database values. It is saved during enrichment, so browsing stays offline.
func metadataStatus(meta metadata.Result) string {
	if len(meta.RawPayload) == 0 {
		return ""
	}
	var raw struct {
		Status string `json:"status"`
	}
	if json.Unmarshal(meta.RawPayload, &raw) != nil {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(raw.Status)) {
	case "released", "ended":
		return "completed"
	case "returning series", "in production", "post production", "planned", "rumored", "pilot":
		return "ongoing"
	case "canceled", "cancelled":
		return "cancelled"
	default:
		return ""
	}
}

func (r *Repository) GetMetadataSnapshot(ctx context.Context, mediaItemID int64, locale string) (*MetadataSnapshot, error) {
	locale = firstNonEmptyLocale(locale)
	var snapshot MetadataSnapshot
	var payload string
	err := r.db.QueryRowContext(ctx, `
		SELECT provider, external_id, locale, raw_payload::text, fetched_at, expires_at
		FROM metadata_snapshots
		WHERE media_item_id = $1 AND locale = $2
	`, mediaItemID, locale).Scan(&snapshot.Provider, &snapshot.ExternalID, &snapshot.Locale, &payload, &snapshot.FetchedAt, &snapshot.ExpiresAt)
	if err != nil {
		return nil, err
	}
	snapshot.Payload = json.RawMessage(payload)
	return &snapshot, nil
}

func (r *Repository) SaveSeasonMetadataSnapshots(ctx context.Context, mediaItemID int64, snapshots []metadata.SeasonResult) error {
	for _, snapshot := range snapshots {
		if len(snapshot.RawPayload) == 0 || snapshot.Provider == "" || snapshot.ExternalID == "" {
			continue
		}
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO season_metadata_snapshots (media_item_id, provider, external_id, season_number, locale, raw_payload, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6::jsonb, CURRENT_TIMESTAMP + INTERVAL '30 days')
			ON CONFLICT (media_item_id, provider, season_number, locale)
			DO UPDATE SET external_id = EXCLUDED.external_id,
				raw_payload = EXCLUDED.raw_payload,
				fetched_at = CURRENT_TIMESTAMP,
				expires_at = EXCLUDED.expires_at
		`, mediaItemID, snapshot.Provider, snapshot.ExternalID, snapshot.SeasonNumber, firstNonEmptyLocale(snapshot.Locale), string(snapshot.RawPayload)); err != nil {
			return fmt.Errorf("save season metadata snapshot: %w", err)
		}
	}
	return nil
}

func (r *Repository) GetSeasonMetadataSnapshots(ctx context.Context, mediaItemID int64, locale string) ([]SeasonMetadataSnapshot, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT provider, external_id, locale, season_number, raw_payload::text, fetched_at, expires_at
		FROM season_metadata_snapshots
		WHERE media_item_id = $1 AND locale = $2
		ORDER BY season_number ASC
	`, mediaItemID, firstNonEmptyLocale(locale))
	if err != nil {
		return nil, fmt.Errorf("query season metadata snapshots: %w", err)
	}
	defer rows.Close()

	snapshots := make([]SeasonMetadataSnapshot, 0)
	for rows.Next() {
		var snapshot SeasonMetadataSnapshot
		var payload string
		if err := rows.Scan(&snapshot.Provider, &snapshot.ExternalID, &snapshot.Locale, &snapshot.SeasonNumber, &payload, &snapshot.FetchedAt, &snapshot.ExpiresAt); err != nil {
			return nil, fmt.Errorf("scan season metadata snapshot: %w", err)
		}
		snapshot.Payload = json.RawMessage(payload)
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func firstNonEmptyLocale(locale string) string {
	if strings.TrimSpace(locale) == "" {
		return "en-US"
	}
	return locale
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
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM video_files WHERE verification_status = 'corrupted'`).Scan(&stats.CorruptedFilesCount)

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

type UpdateMediaRequest struct {
	TitleAR      string   `json:"title_ar"`
	TitleEN      string   `json:"title_en"`
	PlotAR       string   `json:"plot_ar"`
	PlotEN       string   `json:"plot_en"`
	ReleaseYear  int      `json:"release_year"`
	Rating       float64  `json:"rating"`
	PosterPath   string   `json:"poster_path"`
	BannerPath   string   `json:"banner_path"`
	Genres       []string `json:"genres"`
	Type         string   `json:"type"`
	CategorySlug string   `json:"category_slug"`
}

type CreateMediaRequest struct {
	TitleAR      string   `json:"title_ar"`
	TitleEN      string   `json:"title_en"`
	Type         string   `json:"type"`
	CategorySlug string   `json:"category_slug"`
	PlotAR       string   `json:"plot_ar"`
	PlotEN       string   `json:"plot_en"`
	ReleaseYear  int      `json:"release_year"`
	Rating       float64  `json:"rating"`
	PosterPath   string   `json:"poster_path"`
	BannerPath   string   `json:"banner_path"`
	Genres       []string `json:"genres"`
}

func (r *Repository) CreateMediaItem(ctx context.Context, req CreateMediaRequest) (*search.MediaDocument, error) {
	categorySlug := req.CategorySlug
	if categorySlug == "" {
		categorySlug = "movies"
		if req.Type == "series" || req.Type == "anime" {
			categorySlug = req.Type
		}
	}

	var categoryID int64
	if err := r.db.QueryRowContext(ctx, `SELECT id FROM categories WHERE slug = $1`, categorySlug).Scan(&categoryID); err != nil {
		return nil, fmt.Errorf("find category %q: %w", categorySlug, err)
	}

	titleEN := strings.TrimSpace(req.TitleEN)
	if titleEN == "" {
		titleEN = req.TitleAR
	}
	if titleEN == "" {
		return nil, errors.New("title is required")
	}

	mediaType := req.Type
	if mediaType == "" {
		mediaType = "movie"
	}

	var genresArray any
	if len(req.Genres) > 0 {
		genresArray = pq.Array(req.Genres)
	}

	var newID int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO media_items (
			category_id,
			title_ar,
			title_en,
			type,
			plot_ar,
			plot_en,
			release_year,
			rating,
			poster_path,
			banner_path,
			genres,
			status
		)
		VALUES ($1, NULLIF($2, ''), $3, $4, NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, 0), NULLIF($8, 0.0), NULLIF($9, ''), NULLIF($10, ''), $11, 'completed')
		RETURNING id;
	`, categoryID, req.TitleAR, titleEN, mediaType, req.PlotAR, req.PlotEN, req.ReleaseYear, req.Rating, req.PosterPath, req.BannerPath, genresArray).Scan(&newID)
	if err != nil {
		return nil, fmt.Errorf("insert media item: %w", err)
	}

	docs, err := r.ListSearchDocuments(ctx, 10000)
	if err == nil {
		for _, doc := range docs {
			if doc.ID == newID {
				return &doc, nil
			}
		}
	}

	return &search.MediaDocument{
		ID:           newID,
		TitleAR:      req.TitleAR,
		TitleEN:      titleEN,
		Type:         mediaType,
		PlotAR:       req.PlotAR,
		PlotEN:       req.PlotEN,
		ReleaseYear:  req.ReleaseYear,
		Rating:       req.Rating,
		PosterPath:   req.PosterPath,
		BannerPath:   req.BannerPath,
		CategorySlug: categorySlug,
		Genres:       req.Genres,
	}, nil
}

func (r *Repository) UpdateMediaFull(ctx context.Context, id int64, req UpdateMediaRequest) (*search.MediaDocument, error) {
	var categoryID sql.NullInt64
	if req.CategorySlug != "" {
		var catID int64
		if err := r.db.QueryRowContext(ctx, `SELECT id FROM categories WHERE slug = $1`, req.CategorySlug).Scan(&catID); err == nil {
			categoryID = sql.NullInt64{Int64: catID, Valid: true}
		}
	}

	var genresArray any
	if len(req.Genres) > 0 {
		genresArray = pq.Array(req.Genres)
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE media_items
		SET
			category_id = COALESCE(NULLIF($1, 0), category_id),
			title_ar = COALESCE(NULLIF($2, ''), title_ar),
			title_en = COALESCE(NULLIF($3, ''), title_en),
			type = COALESCE(NULLIF($4, ''), type),
			plot_ar = COALESCE(NULLIF($5, ''), plot_ar),
			plot_en = COALESCE(NULLIF($6, ''), plot_en),
			release_year = COALESCE(NULLIF($7, 0), release_year),
			rating = COALESCE(NULLIF($8, 0.0), rating),
			poster_path = COALESCE(NULLIF($9, ''), poster_path),
			banner_path = COALESCE(NULLIF($10, ''), banner_path),
			genres = COALESCE($11, genres)
		WHERE id = $12;
	`, categoryID.Int64, req.TitleAR, req.TitleEN, req.Type, req.PlotAR, req.PlotEN, req.ReleaseYear, req.Rating, req.PosterPath, req.BannerPath, genresArray, id)
	if err != nil {
		return nil, fmt.Errorf("update media full: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return nil, sql.ErrNoRows
	}

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

func (r *Repository) DeleteMediaItem(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Delete associated video_files
	if _, err := tx.ExecContext(ctx, `DELETE FROM video_files WHERE media_item_id = $1`, id); err != nil {
		return fmt.Errorf("delete video files: %w", err)
	}

	// 2. Delete seasons
	if _, err := tx.ExecContext(ctx, `DELETE FROM seasons WHERE media_item_id = $1`, id); err != nil {
		return fmt.Errorf("delete seasons: %w", err)
	}

	// 3. Delete metadata snapshots
	if _, err := tx.ExecContext(ctx, `DELETE FROM metadata_snapshots WHERE media_item_id = $1`, id); err != nil {
		return fmt.Errorf("delete snapshots: %w", err)
	}

	// 4. Delete media item
	res, err := tx.ExecContext(ctx, `DELETE FROM media_items WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete media item: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil || affected == 0 {
		return sql.ErrNoRows
	}

	return tx.Commit()
}

func (r *Repository) GetTMDBSettings(ctx context.Context) (*metadata.TMDBSettings, error) {
	var s metadata.TMDBSettings
	var modulesRaw, remoteConfigRaw []byte
	var fetchMode, imageMode, prefLang, fallbackLang, incImgLang string
	var posterSize, backdropSize, profileSize, stillSize string
	var dailyBandwidthMB int64
	var enableRateLimitDelay bool
	var rateLimitReqPerSec int
	var updatedAt time.Time

	err := r.db.QueryRowContext(ctx, `
		SELECT
			fetch_mode, image_mode, preferred_language, fallback_language, include_image_language,
			daily_bandwidth_mb, enable_rate_limit_delay, rate_limit_requests_per_sec,
			poster_size, backdrop_size, profile_size, still_size,
			modules, remote_config, updated_at
		FROM tmdb_settings
		WHERE id = 1;
	`).Scan(
		&fetchMode, &imageMode, &prefLang, &fallbackLang, &incImgLang,
		&dailyBandwidthMB, &enableRateLimitDelay, &rateLimitReqPerSec,
		&posterSize, &backdropSize, &profileSize, &stillSize,
		&modulesRaw, &remoteConfigRaw, &updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			defaults := metadata.DefaultSettings()
			_ = r.SaveTMDBSettings(ctx, defaults)
			return &defaults, nil
		}
		return nil, fmt.Errorf("get tmdb settings: %w", err)
	}

	s.FetchMode = metadata.FetchMode(fetchMode)
	s.ImageMode = metadata.ImageMode(imageMode)
	s.PreferredLanguage = prefLang
	s.FallbackLanguage = fallbackLang
	s.IncludeImageLanguage = incImgLang
	s.DailyBandwidthMB = dailyBandwidthMB
	s.EnableRateLimitDelay = enableRateLimitDelay
	s.RateLimitRequestsPerSec = rateLimitReqPerSec
	s.PosterSize = posterSize
	s.BackdropSize = backdropSize
	s.ProfileSize = profileSize
	s.StillSize = stillSize
	s.UpdatedAt = updatedAt

	if len(modulesRaw) > 0 {
		_ = json.Unmarshal(modulesRaw, &s.Modules)
	}

	return &s, nil
}

func (r *Repository) SaveTMDBSettings(ctx context.Context, s metadata.TMDBSettings) error {
	modulesJSON, err := json.Marshal(s.Modules)
	if err != nil {
		return fmt.Errorf("marshal modules: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO tmdb_settings (
			id, fetch_mode, image_mode, preferred_language, fallback_language, include_image_language,
			daily_bandwidth_mb, enable_rate_limit_delay, rate_limit_requests_per_sec,
			poster_size, backdrop_size, profile_size, still_size, modules, updated_at
		) VALUES (
			1, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::jsonb, CURRENT_TIMESTAMP
		)
		ON CONFLICT (id) DO UPDATE SET
			fetch_mode = EXCLUDED.fetch_mode,
			image_mode = EXCLUDED.image_mode,
			preferred_language = EXCLUDED.preferred_language,
			fallback_language = EXCLUDED.fallback_language,
			include_image_language = EXCLUDED.include_image_language,
			daily_bandwidth_mb = EXCLUDED.daily_bandwidth_mb,
			enable_rate_limit_delay = EXCLUDED.enable_rate_limit_delay,
			rate_limit_requests_per_sec = EXCLUDED.rate_limit_requests_per_sec,
			poster_size = EXCLUDED.poster_size,
			backdrop_size = EXCLUDED.backdrop_size,
			profile_size = EXCLUDED.profile_size,
			still_size = EXCLUDED.still_size,
			modules = EXCLUDED.modules,
			updated_at = CURRENT_TIMESTAMP;
	`,
		string(s.FetchMode), string(s.ImageMode), s.PreferredLanguage, s.FallbackLanguage, s.IncludeImageLanguage,
		s.DailyBandwidthMB, s.EnableRateLimitDelay, s.RateLimitRequestsPerSec,
		s.PosterSize, s.BackdropSize, s.ProfileSize, s.StillSize, string(modulesJSON),
	)

	if err != nil {
		return fmt.Errorf("save tmdb settings: %w", err)
	}
	return nil
}

type TMDBLogEntry struct {
	MediaItemID      *int64
	RequestKind      string
	Endpoint         string
	StatusCode       int
	BytesDownloaded  int64
	ImagesDownloaded int
	DurationMS       int
	ErrorMessage     string
}

func (r *Repository) LogTMDBUsage(ctx context.Context, entry TMDBLogEntry) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tmdb_usage_log (
			media_item_id, request_kind, endpoint, status_code, bytes_downloaded, images_downloaded, duration_ms, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''));
	`, entry.MediaItemID, entry.RequestKind, entry.Endpoint, entry.StatusCode, entry.BytesDownloaded, entry.ImagesDownloaded, entry.DurationMS, entry.ErrorMessage)
	return err
}

func (r *Repository) GetTMDBUsageSummary(ctx context.Context) (*metadata.TMDBUsageSummary, error) {
	summary := &metadata.TMDBUsageSummary{}

	// 1. Overall & monthly request counts and total bytes
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(bytes_downloaded), 0),
			COALESCE(SUM(images_downloaded), 0),
			MAX(created_at)
		FROM tmdb_usage_log;
	`).Scan(&summary.TotalRequests, &summary.TotalBytesDownloaded, &summary.TotalImagesDownloaded, &summary.LastRequestAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("query total tmdb stats: %w", err)
	}

	// 2. Today's stats (since midnight UTC)
	_ = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(bytes_downloaded), 0),
			COALESCE(SUM(images_downloaded), 0)
		FROM tmdb_usage_log
		WHERE created_at >= CURRENT_DATE;
	`).Scan(&summary.RequestsToday, &summary.BytesToday, &summary.ImagesToday)

	summary.MBToday = float64(summary.BytesToday) / (1024.0 * 1024.0)

	// 3. This month's stats
	_ = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM tmdb_usage_log
		WHERE created_at >= DATE_TRUNC('month', CURRENT_DATE);
	`).Scan(&summary.RequestsThisMonth)

	// 4. Enriched vs Pending Media items count
	_ = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(CASE WHEN metadata_provider IS NOT NULL AND metadata_provider != '' THEN 1 END),
			COUNT(CASE WHEN metadata_provider IS NULL OR metadata_provider = '' THEN 1 END)
		FROM media_items;
	`).Scan(&summary.EnrichedMediaCount, &summary.PendingMediaCount)

	// 5. Get configured quota
	settings, err := r.GetTMDBSettings(ctx)
	if err == nil && settings != nil {
		summary.DailyQuotaMB = settings.DailyBandwidthMB
		if summary.DailyQuotaMB > 0 {
			summary.DailyQuotaUsedPercent = (summary.MBToday / float64(summary.DailyQuotaMB)) * 100.0
			if summary.DailyQuotaUsedPercent > 100.0 {
				summary.DailyQuotaUsedPercent = 100.0
			}
		}
	}

	return summary, nil
}
