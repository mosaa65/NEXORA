package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"nexora/server/internal/search"
)

// decodeJSONField decodes a JSON text column into a target value.
//
// A malformed value degrades to leaving the target untouched rather than failing
// the whole document: one bad column must not make a work unsearchable.
func decodeJSONField(raw string, target any) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return json.Unmarshal([]byte(raw), target)
}

// ProjectionPageSize bounds how many media items are read from PostgreSQL in a
// single projection page.
//
// Paging is the whole point: the previous sync read at most 10,000 documents and
// silently stopped, so a larger library had a permanently incomplete search
// index with no error to notice. Paging removes the ceiling and keeps peak
// memory flat regardless of library size.
const ProjectionPageSize = 1000

// DefaultProjectionPageSize is the page size the API uses when none is chosen.
const DefaultProjectionPageSize = ProjectionPageSize

// SearchProjectionKind identifies which projection stream a cursor belongs to.
type SearchProjectionKind string

const (
	ProjectionMediaItems SearchProjectionKind = "media_items"
	ProjectionEpisodes   SearchProjectionKind = "episodes"
)

// ProjectionCursor is the persisted resume position for one projection stream.
//
// It lives in the database rather than in memory so a rebuild interrupted by a
// restart resumes instead of starting from zero, and so the operator can see how
// far the index has been built.
type ProjectionCursor struct {
	Kind            SearchProjectionKind `json:"kind"`
	LastProjectedID int64                `json:"lastProjectedId"`
	DocumentCount   int64                `json:"documentCount"`
}

// ProjectionStats reports one projection run.
type ProjectionStats struct {
	Documents int              `json:"documents"`
	Pages     int              `json:"pages"`
	Reset     bool             `json:"reset"`
	Cursor    ProjectionCursor `json:"cursor"`
}

// LoadProjectionCursors reads the resume position for every projection stream,
// keyed by stream name.
//
// The key is a plain string so the repository satisfies the projector's store
// interface without the search package importing this one.
func (r *Repository) LoadProjectionCursors(ctx context.Context) (map[string]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
	SELECT kind, last_projected_id FROM search_projection_state
	`)
	if err != nil {
		// A missing projection table must not break search; it only means the
		// cursors start at zero.
		return map[string]int64{}, nil
	}
	defer rows.Close()

	cursors := make(map[string]int64, 2)
	for rows.Next() {
		var kind string
		var lastID int64
		if err := rows.Scan(&kind, &lastID); err != nil {
			return cursors, fmt.Errorf("scan projection cursor: %w", err)
		}
		cursors[kind] = lastID
	}
	return cursors, rows.Err()
}

// LiveWorkIDs returns every work id that still exists in the catalogue.
//
// Merged works are excluded: they have no standalone identity, so their index
// entry is a leftover that should be pruned rather than kept.
func (r *Repository) LiveWorkIDs(ctx context.Context) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
	SELECT id FROM media_items WHERE merged_into_id IS NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("list live work ids: %w", err)
	}
	defer rows.Close()

	ids := make([]int64, 0, 1024)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan live work id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SaveProjectionCursor persists the resume position after a page is indexed.
//
// It is written per page rather than at the end of the run, so an interrupted
// rebuild keeps the work it already did.
func (r *Repository) SaveProjectionCursor(ctx context.Context, kind string, lastID int64, documentCount int64) error {
	_, err := r.db.ExecContext(ctx, `
	INSERT INTO search_projection_state (kind, last_projected_id, last_run_at, document_count)
	VALUES ($1, $2, CURRENT_TIMESTAMP, $3)
	ON CONFLICT (kind) DO UPDATE SET
	last_projected_id = EXCLUDED.last_projected_id,
	last_run_at = CURRENT_TIMESTAMP,
		document_count = EXCLUDED.document_count
	`, kind, lastID, documentCount)
	if err != nil {
		return fmt.Errorf("save projection cursor: %w", err)
	}
	return nil
}

// ResetProjectionCursor sets a stream back to its start so the next run rebuilds
// the index from the beginning.
func (r *Repository) ResetProjectionCursor(ctx context.Context, kind string) error {
	_, err := r.db.ExecContext(ctx, `
	UPDATE search_projection_state SET last_projected_id = 0, document_count = 0
	WHERE kind = $1
	`, kind)
	if err != nil {
		return fmt.Errorf("reset projection cursor: %w", err)
	}
	return nil
}

// ListSearchDocumentPage reads one ordered page of media documents.
//
// Ordering is by id and the cursor is a strict "greater than", so pages never
// overlap and never skip a row even if items are inserted during the rebuild.
// An offset-based pager would misbehave on a live library for exactly that
// reason.
func (r *Repository) ListSearchDocumentPage(ctx context.Context, afterID int64, limit int) ([]search.MediaDocument, error) {
	if limit <= 0 || limit > ProjectionPageSize {
		limit = ProjectionPageSize
	}

	rows, err := r.db.QueryContext(ctx, `
	SELECT
	mi.id,
	mi.title_ar,
	mi.title_en,
	COALESCE(mi.title_normalized, ''),
	COALESCE((SELECT array_agg(a.alias) FROM media_aliases a WHERE a.media_item_id = mi.id), ARRAY[]::text[]),
	mi.type,
	mi.plot_ar,
	mi.plot_en,
	mi.release_year,
	mi.rating,
	mi.poster_path,
	mi.banner_path,
	COALESCE(array_to_json(mi.genres), '[]'::json)::text AS genres,
	COALESCE(mi.content_rating, mi.metadata_facets->>'content_rating', '') AS content_rating,
	c.slug,
	c.name_ar,
	c.name_en,
	COALESCE(mi.file_count, 0), mi.status, COALESCE(mi.season_count, 0),
	COALESCE(NULLIF(mi.metadata_facets->>'number_of_seasons', '')::int, 0),
	COALESCE(NULLIF(mi.metadata_facets->>'number_of_episodes', '')::int, 0),
	COALESCE(mi.total_file_size, 0),
	COALESCE(mi.best_resolution, ''),
	COALESCE(NULLIF(mi.metadata_facets->>'runtime', '')::int, mi.runtime_minutes, 0),
	COALESCE(mi.has_arabic_audio, false),
	COALESCE(mi.has_arabic_subtitles, false)
	FROM media_items mi
	LEFT JOIN categories c ON c.id = mi.category_id
	WHERE mi.id > $1
	  AND mi.merged_into_id IS NULL
	ORDER BY mi.id
	LIMIT $2;
	`, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("query search document page: %w", err)
	}
	defer rows.Close()

	documents := make([]search.MediaDocument, 0, limit)
	for rows.Next() {
		doc, err := scanMediaDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scan search document page: %w", err)
		}
		documents = append(documents, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search document page: %w", err)
	}
	return documents, nil
}

// SearchDocumentByID reads the projection row for one media item.
//
// This is what makes "add one episode, update one document" possible: the
// ingest path asks for exactly the work it touched instead of rebuilding the
// library's index.
func (r *Repository) SearchDocumentByID(ctx context.Context, mediaItemID int64) (*search.MediaDocument, error) {
	documents, err := r.materializeMediaDocuments(ctx, `
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
			COALESCE(mi.content_rating, mi.metadata_facets->>'content_rating', '') AS content_rating,
			c.slug,
			c.name_ar,
			c.name_en,
			COALESCE(mi.file_count, 0), mi.status, COALESCE(mi.season_count, 0),
			COALESCE(NULLIF(mi.metadata_facets->>'number_of_seasons', '')::int, 0),
			COALESCE(NULLIF(mi.metadata_facets->>'number_of_episodes', '')::int, 0),
			COALESCE(mi.total_file_size, 0),
			COALESCE(mi.best_resolution, ''),
			COALESCE(NULLIF(mi.metadata_facets->>'runtime', '')::int, mi.runtime_minutes, 0),
			COALESCE(mi.has_arabic_audio, false),
			COALESCE(mi.has_arabic_subtitles, false)
	FROM media_items mi
	LEFT JOIN categories c ON c.id = mi.category_id
	WHERE mi.id = $1
	`, mediaItemID)
	if err != nil {
		return nil, err
	}
	if len(documents) == 0 {
		return nil, sql.ErrNoRows
	}
	return &documents[0], nil
}

// materializeMediaDocuments runs a media-document query and decodes its rows.
//
// It exists so the paged reader and the single-document reader cannot drift
// apart: both decode through exactly the same code path.
func (r *Repository) materializeMediaDocuments(ctx context.Context, query string, args ...any) ([]search.MediaDocument, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query media documents: %w", err)
	}
	defer rows.Close()

	documents := make([]search.MediaDocument, 0, 16)
	for rows.Next() {
		doc, err := scanMediaDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scan media document: %w", err)
		}
		documents = append(documents, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate media documents: %w", err)
	}
	return documents, nil
}

// scanMediaDocument decodes one media-document row.
//
// Centralising the scan keeps the paged reader and the single-document reader
// from drifting apart: both call this, so a column can never be added to one
// query and forgotten in the other.
func scanMediaDocument(rows *sql.Rows) (search.MediaDocument, error) {
	var doc search.MediaDocument
	var titleAR, titleNormalized, plotAR, plotEN, posterPath, bannerPath sql.NullString
	var categorySlug, categoryAR, categoryEN, contentRating sql.NullString
	var releaseYear sql.NullInt64
	var rating sql.NullFloat64
	var genresText string
	var alternateTitles []string
	var fileCount int
	var summary mediaCardSummary

	if err := rows.Scan(
		&doc.ID,
		&titleAR,
		&doc.TitleEN,
		&titleNormalized,
		pq.Array(&alternateTitles),
		&doc.Type,
		&plotAR,
		&plotEN,
		&releaseYear,
		&rating,
		&posterPath,
		&bannerPath,
		&genresText,
		&contentRating,
		&categorySlug,
		&categoryAR,
		&categoryEN,
		&fileCount, &summary.Status, &summary.SeasonCount, &summary.TMDBSeasonCount,
		&summary.TMDBEpisodeCount, &summary.TotalSize, &summary.BestResolution,
		&summary.RuntimeMinutes, &summary.HasArabicAudio, &summary.HasArabicSubtitles,
	); err != nil {
		return doc, err
	}

	doc.TitleAR = nullableString(titleAR)
	doc.TitleNormalized = nullableString(titleNormalized)
	doc.AlternateTitles = alternateTitles
	doc.PlotAR = nullableString(plotAR)
	doc.PlotEN = nullableString(plotEN)
	doc.PosterPath = nullableString(posterPath)
	doc.BannerPath = nullableString(bannerPath)
	doc.ContentRating = nullableString(contentRating)
	doc.CategorySlug = nullableString(categorySlug)
	doc.CategoryAR = nullableString(categoryAR)
	doc.CategoryEN = nullableString(categoryEN)
	if releaseYear.Valid {
		doc.ReleaseYear = int(releaseYear.Int64)
	}
	if rating.Valid {
		doc.Rating = rating.Float64
	}
	if err := decodeJSONField(genresText, &doc.Genres); err != nil {
		doc.Genres = nil
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
	return doc, nil
}
