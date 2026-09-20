package db

import (
	"context"
	"database/sql"
	"fmt"
)

// EnrichmentStats reports what a local enrichment pass changed.
type EpisodeEnrichmentStats struct {
	WorkID int64 `json:"work_id"`
	// SeasonsCreated counts provider seasons that had no local row yet.
	SeasonsCreated int `json:"seasons_created"`
	// EpisodesCreated counts provider episodes materialised as local rows. These
	// are the "coming soon" entries: the library knows they exist because the
	// provider says so, even though no file has been acquired.
	EpisodesCreated int `json:"episodes_created"`
	// EpisodesEnriched counts local rows that received provider titles,
	// overviews, stills, air dates or runtimes.
	EpisodesEnriched int `json:"episodes_enriched"`
	// LocalEpisodesLinked counts local episodes whose release files were attached
	// to an existing provider episode rather than duplicated.
	LocalEpisodesLinked int `json:"local_episodes_linked"`
}

// EnrichFromLocalSnapshots fills seasons and episodes from the TMDB snapshots
// already stored in this database.
//
// This is the local enrichment path the library owner asked for, and it is the
// right one technically as well:
//
//   - It makes ZERO external requests. The full provider season payload,
//     including every episode's title, overview, still image, air date and
//     runtime, is already persisted in season_metadata_snapshots.
//   - It cannot be rate-limited, cannot time out, and cannot fail because a
//     provider is unreachable.
//   - The match is deterministic, not guessed: a provider episode is identified
//     by (media_item_id, season_number, episode_number), which is exactly how
//     the provider itself numbers episodes. No fuzzy title matching is involved.
//
// Running it also produces the two states the UI needs:
//
//   - an episode that exists locally with a file is playable
//   - an episode that exists only because the provider lists it has no file, and
//     is rendered as "coming soon" rather than hidden
//
// Passing a workID of 0 enriches every work that has a stored snapshot.
func (r *Repository) EnrichFromLocalSnapshots(ctx context.Context, workID int64, limit int) ([]EpisodeEnrichmentStats, error) {
	workIDs, err := r.worksWithSeasonSnapshots(ctx, workID, limit)
	if err != nil {
		return nil, err
	}

	results := make([]EpisodeEnrichmentStats, 0, len(workIDs))
	for _, id := range workIDs {
		if err := ctx.Err(); err != nil {
			return results, err
		}
		stats, err := r.enrichWorkFromSnapshots(ctx, id)
		if err != nil {
			// One bad work must not abandon the rest of the library.
			continue
		}
		results = append(results, stats)
	}
	return results, nil
}

// worksWithSeasonSnapshots lists works that have at least one stored season
// payload, so enrichment only visits works where it can actually do something.
func (r *Repository) worksWithSeasonSnapshots(ctx context.Context, workID int64, limit int) ([]int64, error) {
	query := `
	SELECT DISTINCT sms.media_item_id
	FROM season_metadata_snapshots sms
	JOIN media_items mi ON mi.id = sms.media_item_id
	WHERE mi.merged_into_id IS NULL
	`
	args := []any{}
	if workID > 0 {
		query += ` AND sms.media_item_id = $1`
		args = append(args, workID)
	}
	query += ` ORDER BY sms.media_item_id`
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list works with season snapshots: %w", err)
	}
	defer rows.Close()

	ids := make([]int64, 0, 64)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan snapshot work id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// providerSeason is one stored season payload, decoded enough to walk its
// episodes without re-parsing the JSON per episode.
type providerSeason struct {
	SeasonNumber int
	Name         string
	Overview     string
	PosterPath   string
	AirDate      string
	Episodes     []providerEpisode
}

// providerEpisode is one episode inside a stored season payload.
type providerEpisode struct {
	EpisodeNumber int
	Title         string
	Overview      string
	StillPath     string
	AirDate       string
	Runtime       int
}

// loadProviderSeasons reads and decodes every stored season for one work.
func (r *Repository) loadProviderSeasons(ctx context.Context, workID int64) ([]providerSeason, error) {
	rows, err := r.db.QueryContext(ctx, `
	SELECT season_number, raw_payload::text
	FROM season_metadata_snapshots
	WHERE provider = 'tmdb' AND media_item_id = $1
	ORDER BY season_number
	`, workID)
	if err != nil {
		return nil, fmt.Errorf("load provider seasons: %w", err)
	}
	defer rows.Close()

	seasons := make([]providerSeason, 0, 8)
	for rows.Next() {
		var seasonNumber int
		var payload string
		if err := rows.Scan(&seasonNumber, &payload); err != nil {
			return nil, fmt.Errorf("scan provider season: %w", err)
		}
		season, err := decodeProviderSeason(seasonNumber, []byte(payload))
		if err != nil {
			continue
		}
		seasons = append(seasons, season)
	}
	return seasons, rows.Err()
}

// enrichWorkFromSnapshots materialises seasons and episodes for one work.
//
// The whole pass runs in a single transaction per work, so a work is either
// fully enriched or unchanged. That matters because a half-enriched season would
// show some episodes as "coming soon" that are already owned.
func (r *Repository) enrichWorkFromSnapshots(ctx context.Context, workID int64) (EpisodeEnrichmentStats, error) {
	stats := EpisodeEnrichmentStats{WorkID: workID}

	seasons, err := r.loadProviderSeasons(ctx, workID)
	if err != nil {
		return stats, err
	}
	if len(seasons) == 0 {
		return stats, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return stats, fmt.Errorf("begin enrichment: %w", err)
	}
	defer tx.Rollback()

	for _, season := range seasons {
		// Season number 0 is how TMDB models "Specials". It is kept, because
		// dropping it would hide specials the library actually owns.
		seasonID, created, err := upsertProviderSeason(ctx, tx, workID, season)
		if err != nil {
			return stats, err
		}
		if created {
			stats.SeasonsCreated++
		}

		for _, episode := range season.Episodes {
			outcome, err := upsertProviderEpisode(ctx, tx, seasonID, episode)
			if err != nil {
				return stats, err
			}
			switch outcome {
			case episodeCreated:
				stats.EpisodesCreated++
			case episodeEnriched:
				stats.EpisodesEnriched++
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return stats, fmt.Errorf("commit enrichment: %w", err)
	}
	return stats, nil
}

// episodeOutcome says what the upsert did, so the report can count it.
type episodeOutcome int

const (
	episodeUnchanged episodeOutcome = iota
	episodeCreated
	episodeEnriched
)

// upsertProviderSeason creates or enriches one local season row.
func upsertProviderSeason(ctx context.Context, tx *sql.Tx, workID int64, season providerSeason) (int64, bool, error) {
	var existingID int64
	var currentTitle, currentOverview sql.NullString
	err := tx.QueryRowContext(ctx, `
	SELECT id, title_en, overview_en FROM seasons
	WHERE media_item_id = $1 AND season_number = $2
	`, workID, season.SeasonNumber).Scan(&existingID, &currentTitle, &currentOverview)

	if err == nil {
		// The row exists. Only fill descriptive gaps: an operator's own title is
		// never overwritten by provider metadata.
		if currentTitle.String == "" || currentOverview.String == "" {
			if _, err := tx.ExecContext(ctx, `
				UPDATE seasons
				SET title_en = COALESCE(NULLIF(title_en, ''), NULLIF($2, '')),
				    overview_en = COALESCE(NULLIF(overview_en, ''), NULLIF($3, '')),
				    poster_path = COALESCE(NULLIF(poster_path, ''), NULLIF($4, '')),
				    air_date = COALESCE(air_date, NULLIF($5, '')::date),
				    provider = COALESCE(provider, 'tmdb'),
				    external_id = COALESCE(external_id, $6)
				WHERE id = $1
			`, existingID, season.Name, season.Overview, season.PosterPath, season.AirDate,
				fmt.Sprintf("%d-%d", workID, season.SeasonNumber)); err != nil {
				return 0, false, fmt.Errorf("enrich season %d: %w", season.SeasonNumber, err)
			}
		}
		return existingID, false, nil
	}
	if err != sql.ErrNoRows {
		return 0, false, fmt.Errorf("find season %d: %w", season.SeasonNumber, err)
	}

	title := season.Name
	if title == "" {
		title = fmt.Sprintf("Season %02d", season.SeasonNumber)
	}
	if err := tx.QueryRowContext(ctx, `
	INSERT INTO seasons (media_item_id, season_number, title_en, overview_en, poster_path, air_date, provider, external_id)
	VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, '')::date, 'tmdb', $7)
	ON CONFLICT (media_item_id, season_number) DO UPDATE
	SET title_en = COALESCE(NULLIF(seasons.title_en, ''), EXCLUDED.title_en)
	RETURNING id
	`, workID, season.SeasonNumber, title, season.Overview, season.PosterPath, season.AirDate,
		fmt.Sprintf("%d-%d", workID, season.SeasonNumber)).Scan(&existingID); err != nil {
		return 0, false, fmt.Errorf("create season %d: %w", season.SeasonNumber, err)
	}
	return existingID, true, nil
}

// upsertProviderEpisode creates or enriches one local episode row.
//
// The row is created even when no file exists. That is the point: the provider
// says the episode exists, so the library records it and the UI can show it as
// "coming soon" instead of pretending the season has fewer episodes than it does.
func upsertProviderEpisode(ctx context.Context, tx *sql.Tx, seasonID int64, episode providerEpisode) (episodeOutcome, error) {
	var existingID int64
	var currentTitle, currentOverview sql.NullString
	err := tx.QueryRowContext(ctx, `
	SELECT id, title_en, overview_en FROM episodes
	WHERE season_id = $1 AND episode_number = $2
	`, seasonID, episode.EpisodeNumber).Scan(&existingID, &currentTitle, &currentOverview)

	if err == nil {
		if currentTitle.String == episode.Title && currentOverview.String == episode.Overview {
			return episodeUnchanged, nil
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE episodes
			SET title_en = COALESCE(NULLIF(title_en, ''), NULLIF($2, '')),
			    overview_en = COALESCE(NULLIF(overview_en, ''), NULLIF($3, '')),
			    still_path = COALESCE(NULLIF(still_path, ''), NULLIF($4, '')),
			    air_date = COALESCE(air_date, NULLIF($5, '')::date),
			    runtime = COALESCE(NULLIF(runtime, 0), NULLIF($6, 0)::int),
			    provider = COALESCE(provider, 'tmdb')
			WHERE id = $1
	`, existingID, episode.Title, episode.Overview, episode.StillPath,
			episode.AirDate, episode.Runtime); err != nil {
			return episodeUnchanged, fmt.Errorf("enrich episode %d: %w", episode.EpisodeNumber, err)
		}
		return episodeEnriched, nil
	}
	if err != sql.ErrNoRows {
		return episodeUnchanged, fmt.Errorf("find episode %d: %w", episode.EpisodeNumber, err)
	}

	if _, err := tx.ExecContext(ctx, `
	INSERT INTO episodes (
	season_id, episode_number, title_en, overview_en,
	still_path, air_date, runtime, provider
	)
	VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, '')::date, NULLIF($7, 0), 'tmdb')
	ON CONFLICT (season_id, episode_number) DO NOTHING
	`, seasonID, episode.EpisodeNumber, episode.Title, episode.Overview,
		episode.StillPath, episode.AirDate, episode.Runtime); err != nil {
		return episodeUnchanged, fmt.Errorf("create episode %d: %w", episode.EpisodeNumber, err)
	}
	return episodeCreated, nil
}

// LinkOrphanEpisodes attaches video files to their episode row, creating the
// episode row when the provider never listed that episode.
//
// Two situations produce an unlinked file:
//
//  1. The provider lists the episode and a row exists, but the file was written
//     before the rows existed. A plain link fixes it.
//  2. The provider does not list the episode at all — an unlisted special, a
//     fan release, a numbering the provider does not use. Here no row exists to
//     link to, and leaving the file unlinked means the library owns an episode
//     its own catalogue cannot see.
//
// Case 2 is why this is not a single UPDATE: the episode is materialised from
// the file's own season and episode number, with no descriptive fields, so the
// file becomes reachable while the provider metadata stays absent rather than
// invented. Both steps run in one transaction so a file is never left pointing at
// a half-created episode.
func (r *Repository) LinkOrphanEpisodes(ctx context.Context) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin episode linking: %w", err)
	}
	defer tx.Rollback()

	linkedTotal := 0
	// Step 1: link every file whose episode row already exists.
	result, err := tx.ExecContext(ctx, `
	UPDATE video_files vf
	SET episode_id = e.id
	FROM episodes e
	WHERE vf.episode_id IS NULL
	  AND vf.season_id = e.season_id
	  AND vf.episode_number = e.episode_number
	  AND vf.season_id IS NOT NULL
	`)
	if err != nil {
		return 0, fmt.Errorf("link existing episodes: %w", err)
	}
	if affected, err := result.RowsAffected(); err == nil {
		linkedTotal += int(affected)
	}

	// Step 2: create the missing episode rows for files that carry a number but
	// have no provider counterpart, then link them.
	//
	// DISTINCT ON keeps one row per (season, episode) even when several release
	// files describe the same episode, so a multi-resolution season does not
	// produce duplicate episode entities.
	result, err = tx.ExecContext(ctx, `
	WITH missing AS (
	    SELECT DISTINCT ON (vf.season_id, vf.episode_number)
	           vf.season_id,
	           vf.episode_number
	    FROM video_files vf
	    WHERE vf.episode_id IS NULL
	      AND vf.season_id IS NOT NULL
	      AND vf.episode_number IS NOT NULL
	      AND vf.episode_number > 0
	    ORDER BY vf.season_id, vf.episode_number
	),
	inserted AS (
	    INSERT INTO episodes (season_id, episode_number, provider)
	    SELECT season_id, episode_number, 'local'
	    FROM missing
	    ON CONFLICT (season_id, episode_number) DO NOTHING
	    RETURNING id, season_id, episode_number
	)
	UPDATE video_files vf
	SET episode_id = inserted.id
	FROM inserted
	WHERE vf.episode_id IS NULL
	  AND vf.season_id = inserted.season_id
	  AND vf.episode_number = inserted.episode_number
	`)
	if err != nil {
		return 0, fmt.Errorf("create and link missing episodes: %w", err)
	}
	if affected, err := result.RowsAffected(); err == nil {
		linkedTotal += int(affected)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit episode linking: %w", err)
	}
	return linkedTotal, nil
}
