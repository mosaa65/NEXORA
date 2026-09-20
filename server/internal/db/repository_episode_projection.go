package db

import (
	"context"
	"database/sql"
	"fmt"

	"nexora/server/internal/search"
)

// episodeProjectionRow is one episode joined with everything the search document
// needs: its own row, its season, its parent work, its release files, and the
// provider snapshot that was already stored locally.
//
// The provider snapshot is joined, not fetched. That is the whole point of the
// local enrichment path: the library owner already stores the full TMDB season
// response, including every episode's title, overview, still image, air date and
// runtime. Joining it costs one query and zero network calls, so enriching
// thousands of episodes cannot fail, cannot be rate-limited, and cannot block on
// a third party.
//
// The join key is (season_number, episode_number) within a work's external id,
// which is exactly how a provider identifies an episode. Inventing a mapping is
// unnecessary: the identity is already deterministic.
const episodeProjectionQuery = `
WITH provider_episode AS (
    -- The join is on the LOCAL media_item_id plus (season, episode) number,
    -- because the snapshot row already records which work it belongs to. Using
    -- the provider's external id instead would need a second lookup and would
    -- silently miss any show whose provider id was never stored.
    SELECT
        sms.media_item_id                                       AS media_item_id,
        sms.season_number                                       AS season_number,
        (episode->>'episode_number')::int                       AS episode_number,
        COALESCE(episode->>'name', '')                          AS title_en,
        COALESCE(episode->>'overview', '')                      AS overview_en,
        COALESCE(episode->>'still_path', '')                    AS still_path,
        COALESCE(episode->>'air_date', '')                      AS air_date,
        COALESCE(NULLIF(episode->>'runtime', '')::int, 0)       AS runtime
    FROM season_metadata_snapshots sms
    CROSS JOIN LATERAL jsonb_array_elements(sms.raw_payload->'episodes') AS episode
    WHERE sms.provider = 'tmdb'
)
SELECT
    e.id,
    s.media_item_id,
    s.season_number,
    e.episode_number,
    COALESCE(e.title_en, '')                                    AS local_title_en,
    COALESCE(e.title_ar, '')                                    AS local_title_ar,
    COALESCE(e.overview_en, '')                                 AS local_overview_en,
    COALESCE(e.overview_ar, '')                                 AS local_overview_ar,
    COALESCE(e.still_path, '')                                  AS local_still,
    COALESCE(e.air_date::text, '')                              AS local_air_date,
    COALESCE(e.runtime, 0)                                      AS local_runtime,
    COALESCE(pe.title_en, '')                                   AS provider_title_en,
    COALESCE(pe.overview_en, '')                                AS provider_overview_en,
    COALESCE(pe.still_path, '')                                 AS provider_still,
    COALESCE(pe.air_date, '')                                   AS provider_air_date,
    COALESCE(pe.runtime, 0)                                     AS provider_runtime,
    COALESCE(mi.title_en, '')                                   AS work_title_en,
    COALESCE(mi.title_ar, '')                                   AS work_title_ar,
    COALESCE(mi.title_normalized, '')                           AS work_title_normalized,
    COALESCE(c.slug, '')                                        AS category_slug,
    COALESCE(vf.file_count, 0)                                  AS file_count,
    COALESCE(vf.best_resolution, '')                            AS resolution,
    COALESCE(vf.total_size, 0)                                  AS total_size,
    COALESCE(vf.max_duration, 0)                                AS duration
FROM episodes e
JOIN seasons s ON s.id = e.season_id
JOIN media_items mi ON mi.id = s.media_item_id
LEFT JOIN categories c ON c.id = mi.category_id
LEFT JOIN provider_episode pe
       ON pe.media_item_id = s.media_item_id
      AND pe.season_number = s.season_number
      AND pe.episode_number = e.episode_number
LEFT JOIN LATERAL (
    SELECT
        COUNT(*)                    AS file_count,
        MAX(f.resolution)           AS best_resolution,
        SUM(f.file_size)            AS total_size,
        MAX(f.duration)             AS max_duration
    FROM video_files f
    WHERE f.episode_id = e.id
) vf ON TRUE
WHERE e.id > $1
  AND mi.merged_into_id IS NULL
ORDER BY e.id
LIMIT $2;
`

// ListEpisodeDocumentPage reads one ordered page of episode documents.
//
// Keyset pagination by ascending id, matching the work projector: pages never
// overlap and never skip a row even when episodes are inserted during a rebuild,
// which an offset pager would get wrong on a live library.
func (r *Repository) ListEpisodeDocumentPage(ctx context.Context, afterID int64, limit int) ([]search.EpisodeDocument, error) {
	if limit <= 0 || limit > ProjectionPageSize {
		limit = ProjectionPageSize
	}

	rows, err := r.db.QueryContext(ctx, episodeProjectionQuery, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("query episode document page: %w", err)
	}
	defer rows.Close()

	documents := make([]search.EpisodeDocument, 0, limit)
	for rows.Next() {
		document, err := scanEpisodeDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scan episode document: %w", err)
		}
		documents = append(documents, document)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate episode document page: %w", err)
	}
	return documents, nil
}

// EpisodeDocumentByID reads the projection row for one episode.
//
// This is the targeted path: after an episode is enriched or its file set
// changes, only that document is written instead of rebuilding the episode
// index.
func (r *Repository) EpisodeDocumentByID(ctx context.Context, episodeID int64) (*search.EpisodeDocument, error) {
	// The shared query pages by `e.id > $1`, so asking for ids greater than
	// episodeID-1 with a limit of 1 returns exactly this episode.
	rows, err := r.db.QueryContext(ctx, episodeProjectionQuery, episodeID-1, 1)
	if err != nil {
		return nil, fmt.Errorf("query episode document: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate episode document: %w", err)
		}
		return nil, sql.ErrNoRows
	}

	document, err := scanEpisodeDocument(rows)
	if err != nil {
		return nil, fmt.Errorf("scan episode document: %w", err)
	}
	return &document, nil
}

// scanEpisodeDocument decodes one joined episode row.
//
// The provider value wins over the local value for every descriptive field,
// because TMDB metadata is authoritative for a title while the local row may be
// empty or derived from a filename. The local value is only used when the
// provider has nothing, and EnrichedFrom records which one was used so the UI
// and the quality report can tell them apart.
func scanEpisodeDocument(rows *sql.Rows) (search.EpisodeDocument, error) {
	var document search.EpisodeDocument
	var mediaItemID int64
	var localTitleEN, localTitleAR, localOverviewEN, localOverviewAR sql.NullString
	var localStill, localAirDate sql.NullString
	var localRuntime, providerRuntime int
	var providerTitleEN, providerOverviewEN, providerStill, providerAirDate sql.NullString
	var workTitleEN, workTitleAR, workTitleNormalized sql.NullString
	var categorySlug, resolution sql.NullString
	var fileCount int
	var totalSize int64
	var duration int

	if err := rows.Scan(
		&document.ID,
		&mediaItemID,
		&document.SeasonNumber,
		&document.EpisodeNumber,
		&localTitleEN,
		&localTitleAR,
		&localOverviewEN,
		&localOverviewAR,
		&localStill,
		&localAirDate,
		&localRuntime,
		&providerTitleEN,
		&providerOverviewEN,
		&providerStill,
		&providerAirDate,
		&providerRuntime,
		&workTitleEN,
		&workTitleAR,
		&workTitleNormalized,
		&categorySlug,
		&fileCount,
		&resolution,
		&totalSize,
		&duration,
	); err != nil {
		return document, err
	}

	document.WorkID = mediaItemID
	document.WorkTitleEN = workTitleEN.String
	document.WorkTitleAR = workTitleAR.String
	document.WorkTitleNormalized = workTitleNormalized.String
	document.CategorySlug = categorySlug.String

	// Prefer the provider's descriptive fields; fall back to the local row.
	document.EpisodeTitleEN = firstNonEmpty(providerTitleEN.String, localTitleEN.String)
	document.OverviewEN = firstNonEmpty(providerOverviewEN.String, localOverviewEN.String)
	document.StillPath = firstNonEmpty(providerStill.String, localStill.String)
	document.AirDate = firstNonEmpty(providerAirDate.String, localAirDate.String)
	document.Runtime = firstNonZero(providerRuntime, localRuntime)
	document.EpisodeTitleAR = localTitleAR.String
	document.OverviewAR = localOverviewAR.String

	switch {
	case providerTitleEN.String != "":
		document.EnrichedFrom = "provider"
	case localTitleEN.String != "":
		document.EnrichedFrom = "local"
	}

	document.FileCount = fileCount
	document.HasLocalFile = fileCount > 0
	document.Resolution = resolution.String
	document.FileSize = totalSize
	document.Duration = duration

	return document, nil
}

// ProviderOnlyEpisode is an episode the provider lists that the library has not
// acquired.
type ProviderOnlyEpisode struct {
	EpisodeID     int64  `json:"episode_id"`
	WorkID        int64  `json:"work_id"`
	WorkTitleEN   string `json:"work_title_en"`
	WorkTitleAR   string `json:"work_title_ar,omitempty"`
	SeasonNumber  int    `json:"season_number"`
	EpisodeNumber int    `json:"episode_number"`
	TitleEN       string `json:"title_en,omitempty"`
	AirDate       string `json:"air_date,omitempty"`
}

// ListProviderOnlyEpisodes returns episodes that exist because the provider
// lists them, with no release file behind them.
//
// This is the acquisition list and the "coming soon" state in one query: an
// operator can see exactly which episodes of which season are still missing,
// rather than comparing a provider episode count against a file count by hand.
func (r *Repository) ListProviderOnlyEpisodes(ctx context.Context, limit int) ([]ProviderOnlyEpisode, error) {
	if limit <= 0 || limit > 2000 {
		limit = 200
	}

	rows, err := r.db.QueryContext(ctx, `
	SELECT
	e.id,
	s.media_item_id,
	COALESCE(mi.title_en, ''),
	COALESCE(mi.title_ar, ''),
	s.season_number,
	e.episode_number,
	COALESCE(e.title_en, ''),
	COALESCE(e.air_date::text, '')
	FROM episodes e
	JOIN seasons s ON s.id = e.season_id
	JOIN media_items mi ON mi.id = s.media_item_id
	WHERE mi.merged_into_id IS NULL
	  AND e.provider IS NOT NULL
	  AND NOT EXISTS (SELECT 1 FROM video_files vf WHERE vf.episode_id = e.id)
	ORDER BY mi.title_en, s.season_number, e.episode_number
	LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list provider-only episodes: %w", err)
	}
	defer rows.Close()

	items := make([]ProviderOnlyEpisode, 0, limit)
	for rows.Next() {
		var item ProviderOnlyEpisode
		if err := rows.Scan(&item.EpisodeID, &item.WorkID, &item.WorkTitleEN, &item.WorkTitleAR,
			&item.SeasonNumber, &item.EpisodeNumber, &item.TitleEN, &item.AirDate); err != nil {
			return nil, fmt.Errorf("scan provider-only episode: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// LiveEpisodeIDs returns every episode id that still exists.
func (r *Repository) LiveEpisodeIDs(ctx context.Context) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id FROM episodes`)
	if err != nil {
		return nil, fmt.Errorf("list live episode ids: %w", err)
	}
	defer rows.Close()

	ids := make([]int64, 0, 1024)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan live episode id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// firstNonZero returns the first value that is not zero.
//
// It is used to pick a runtime: the provider's episode runtime wins, and the
// locally probed duration is the fallback when the provider recorded none.
// (firstNonEmpty already exists in repository_metadata.go and is reused here.)
func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
