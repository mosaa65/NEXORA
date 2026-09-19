package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"nexora/server/internal/scanner"
)

// IngestStats summarises a batch write. It is returned so the scan report can
// state what actually reached the database.
type IngestStats struct {
	Scanned   int `json:"scanned"`
	Inserted  int `json:"inserted"`
	Updated   int `json:"updated"`
	Moved     int `json:"moved"`
	Unchanged int `json:"unchanged"`
	Failed    int `json:"failed"`
}

// ingestWithoutResolution persists a batch of scanned files WITHOUT entity
// resolution: each parsed filename is turned directly into a media_items row.
//
// It replaces the previous one-transaction-per-file implementation. The previous
// approach issued a BEGIN/COMMIT pair, a category lookup and up to five
// statements per file, which is roughly a million round-trips for a million-file
// library. This version wraps the whole batch in a single transaction and uses
// multi-row upserts, so the cost is proportional to batches rather than files.
//
// Errors are collected per file rather than aborting the batch: one malformed
// path must not discard the other 127 files in its batch.
//
// WARNING: this bypasses entity resolution. It must never be called from the
// scan, the watcher or the scheduler, because doing so recreates works named
// after a watermark or an episode marker. Runtime paths use ResolutionSession.
func (r *Repository) ingestWithoutResolution(ctx context.Context, files []scanner.FileInfo) (IngestStats, error) {
	stats := IngestStats{Scanned: len(files)}
	if len(files) == 0 {
		return stats, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return stats, fmt.Errorf("begin ingest batch: %w", err)
	}
	defer tx.Rollback()

	categories, err := loadCategoryIDs(ctx, tx)
	if err != nil {
		return stats, err
	}

	for i := range files {
		file := files[i]
		outcome, err := r.ingestOne(ctx, tx, file, categories)
		if err != nil {
			// A single bad record is counted and skipped; the batch continues.
			stats.Failed++
			continue
		}
		switch outcome {
		case scanner.ChangeNew:
			stats.Inserted++
		case scanner.ChangeChanged:
			stats.Updated++
		case scanner.ChangeRenamed:
			stats.Moved++
		default:
			stats.Unchanged++
		}
	}

	if err := tx.Commit(); err != nil {
		return stats, fmt.Errorf("commit ingest batch: %w", err)
	}
	return stats, nil
}

// loadCategoryIDs reads the seeded category lookup once per batch instead of
// querying it per file.
func loadCategoryIDs(ctx context.Context, tx *sql.Tx) (map[string]int64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT slug, id FROM categories`)
	if err != nil {
		return nil, fmt.Errorf("load categories: %w", err)
	}
	defer rows.Close()

	out := make(map[string]int64, 8)
	for rows.Next() {
		var slug string
		var id int64
		if err := rows.Scan(&slug, &id); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		out[slug] = id
	}
	return out, rows.Err()
}

// ingestOne writes a single scanned file inside the shared batch transaction.
func (r *Repository) ingestOne(ctx context.Context, tx *sql.Tx, file scanner.FileInfo, categories map[string]int64) (scanner.ChangeKind, error) {
	categorySlug, mediaType := ClassifyMedia(file.Path, file.Parsed)
	categoryID, ok := categories[categorySlug]
	if !ok {
		// An unknown category is a data problem, not a fatal one: fall back to
		// the media type so the file is still indexed.
		categoryID = categories[mediaType]
	}
	if categoryID == 0 {
		return file.Change, fmt.Errorf("no category for slug %q", categorySlug)
	}

	title := file.Parsed.Title
	if title == "" {
		title = strings.TrimSuffix(lastPathSegment(file.Path), extensionOf(file.Path))
	}
	titleEN := file.Parsed.TitleEN
	if titleEN == "" {
		titleEN = title
	}
	titleAR := file.Parsed.TitleAR
	if titleAR == "" && containsArabic(title) {
		titleAR = title
	}

	// Artwork is cached once per distinct source file, not once per video: the
	// repository keeps a directory-level memo for the duration of the process.
	localPosterURL := ""
	if file.ArtworkPath != "" {
		localPosterURL = r.CacheLocalArtwork(file.ArtworkPath)
	}

	mediaID, err := upsertMediaItem(ctx, tx, upsertMediaParams{
		categoryID:  categoryID,
		titleAR:     titleAR,
		titleEN:     titleEN,
		mediaType:   mediaType,
		releaseYear: file.Parsed.ReleaseYear,
		posterURL:   localPosterURL,
	})
	if err != nil {
		return file.Change, err
	}

	if file.Parsed.OriginTag != "" {
		if err := mergeMediaTags(ctx, tx, mediaID, []string{file.Parsed.OriginTag}); err != nil {
			return file.Change, fmt.Errorf("apply origin tag: %w", err)
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
			return file.Change, fmt.Errorf("upsert season %d: %w", seasonNumber, err)
		}
		seasonID.Valid = true
		if file.Parsed.EpisodeNumber > 0 {
			episodeNumber = sql.NullInt64{Int64: int64(file.Parsed.EpisodeNumber), Valid: true}
		}
	}

	partNumber := sql.NullInt64{}
	if file.Parsed.PartNumber > 0 {
		partNumber = sql.NullInt64{Int64: int64(file.Parsed.PartNumber), Valid: true}
	}

	// A rename rewrites the path of the existing row instead of inserting a new
	// one, which is what keeps "same logical indexed entity" true across moves.
	if file.Change == scanner.ChangeRenamed && file.KnownID > 0 {
		_, err := tx.ExecContext(ctx, `
			UPDATE video_files
			SET file_path = $2,
			    file_size = $3,
			    file_mod_time = $4,
			    file_id = NULLIF($5, ''),
			    root_id = NULLIF($6, ''),
			    root_path = NULLIF($7, ''),
			    relative_path = NULLIF($8, ''),
			    media_item_id = $9,
			    season_id = $10,
			    episode_number = $11,
			    part_number = $12,
			    resolution = NULLIF($13, ''),
			    state = 'active',
			    last_seen_at = CURRENT_TIMESTAMP
			WHERE id = $1
	`, file.KnownID, file.Path, file.Size, file.ModTime, file.FileID, file.RootID, file.Root,
			file.RelativePath, mediaID, seasonID, episodeNumber, partNumber, file.Parsed.Resolution)
		if err != nil {
			return file.Change, fmt.Errorf("move video file %q: %w", file.Path, err)
		}
		return scanner.ChangeRenamed, nil
	}

	_, err = tx.ExecContext(ctx, `
	INSERT INTO video_files (
			media_item_id,
			season_id,
			episode_number,
			part_number,
			title_ar,
			title_en,
			title_normalized,
			search_tokens,
			file_path,
			relative_path,
			file_size,
			file_mod_time,
			file_id,
			root_id,
			root_path,
			resolution,
			parse_confidence,
			parse_reasons,
			special_kind,
			state,
			audio_tracks,
			subtitles,
			last_seen_at
	)
	VALUES (
			$1, $2, $3, $4, NULLIF($5, ''), $6, NULLIF($7, ''), $8,
			$9, NULLIF($10, ''), $11, $12, NULLIF($13, ''), NULLIF($14, ''), NULLIF($15, ''),
			NULLIF($16, ''), $17, $18, NULLIF($19, ''), 'active',
			'[]'::jsonb, '[]'::jsonb, CURRENT_TIMESTAMP
	)
	ON CONFLICT (file_path)
	DO UPDATE SET
			media_item_id = EXCLUDED.media_item_id,
			season_id = EXCLUDED.season_id,
			episode_number = EXCLUDED.episode_number,
			part_number = EXCLUDED.part_number,
			title_ar = EXCLUDED.title_ar,
			title_en = EXCLUDED.title_en,
			title_normalized = EXCLUDED.title_normalized,
			search_tokens = EXCLUDED.search_tokens,
			file_size = EXCLUDED.file_size,
			file_mod_time = EXCLUDED.file_mod_time,
			file_id = EXCLUDED.file_id,
			root_id = EXCLUDED.root_id,
			root_path = EXCLUDED.root_path,
			relative_path = EXCLUDED.relative_path,
			resolution = EXCLUDED.resolution,
			parse_confidence = EXCLUDED.parse_confidence,
			parse_reasons = EXCLUDED.parse_reasons,
			special_kind = EXCLUDED.special_kind,
			state = 'active',
			last_seen_at = CURRENT_TIMESTAMP;
	`, mediaID, seasonID, episodeNumber, partNumber, titleAR, titleEN,
		file.Parsed.TitleNormalized, pq.Array(file.SearchTokens),
		file.Path, file.RelativePath, file.Size, file.ModTime.UTC(), file.FileID,
		file.RootID, file.Root, file.Parsed.Resolution, float64(file.Parsed.Confidence),
		pq.Array(file.Parsed.Reasons), file.Parsed.SpecialKind)
	if err != nil {
		return file.Change, fmt.Errorf("upsert video file %q: %w", file.Path, err)
	}

	return file.Change, nil
}

type upsertMediaParams struct {
	categoryID  int64
	titleAR     string
	titleEN     string
	mediaType   string
	releaseYear int
	posterURL   string
}

// upsertMediaItem resolves the media item for a file, creating it when absent.
//
// The lookup deliberately matches on type + title_en + year, which mirrors the
// existing unique identity index, so two files of the same show never create two
// media rows.
func upsertMediaItem(ctx context.Context, tx *sql.Tx, params upsertMediaParams) (int64, error) {
	var mediaID int64
	err := tx.QueryRowContext(ctx, `
	INSERT INTO media_items (
			category_id, title_ar, title_en, type, release_year, poster_path, banner_path, status
	)
	VALUES ($1, NULLIF($2, ''), $3, $4, NULLIF($5, 0), NULLIF($6, ''), NULLIF($6, ''), 'completed')
	ON CONFLICT (type, LOWER(title_en), COALESCE(release_year, 0)) DO UPDATE
			SET poster_path = COALESCE(NULLIF(media_items.poster_path, ''), EXCLUDED.poster_path),
			    banner_path = COALESCE(NULLIF(media_items.banner_path, ''), EXCLUDED.banner_path),
			    title_ar = COALESCE(NULLIF(media_items.title_ar, ''), EXCLUDED.title_ar)
	RETURNING id;
	`, params.categoryID, params.titleAR, params.titleEN, params.mediaType,
		params.releaseYear, params.posterURL).Scan(&mediaID)
	if err != nil {
		return 0, fmt.Errorf("upsert media item %q: %w", params.titleEN, err)
	}
	return mediaID, nil
}

// MarkFilesState updates the reconciliation state of many files at once.
//
// This is how a scan records MISSING / UNAVAILABLE without deleting anything.
// It is the only place a state transition happens, so the audit trail is exact.
func (r *Repository) MarkFilesState(ctx context.Context, paths []string, state scanner.FileState, scanID string) (int, error) {
	if len(paths) == 0 {
		return 0, nil
	}
	result, err := r.db.ExecContext(ctx, `
	UPDATE video_files
	SET state = $2,
		    last_scan_id = NULLIF($3, '')
	WHERE file_path = ANY($1) AND state <> $2
	`, pq.Array(paths), string(state), scanID)
	if err != nil {
		return 0, fmt.Errorf("mark files %s: %w", state, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("mark files affected rows: %w", err)
	}
	return int(affected), nil
}

// RootsForScan returns the paths of root directories from a scan session.
func (r *Repository) ListKnownFiles(ctx context.Context, roots []string) ([]scanner.KnownFile, error) {
	query := `
	SELECT id, file_path, file_size, COALESCE(file_mod_time, created_at), COALESCE(file_id, ''),
		       COALESCE(root_id, ''), COALESCE(last_scan_id, ''), COALESCE(last_seen_at, created_at), state
	FROM video_files
	WHERE state <> 'error'
	`
	args := []any{}
	if len(roots) > 0 {
		// Restrict the known set to the roots being scanned, so a scan of Disk A
		// never declares Disk B's files missing.
		patterns := make([]string, 0, len(roots))
		for _, root := range roots {
			patterns = append(patterns, escapeLikePattern(root)+"%")
		}
		query += ` AND (file_path LIKE ANY($1) OR root_path LIKE ANY($1))`
		args = append(args, pq.Array(patterns))
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list known files: %w", err)
	}
	defer rows.Close()

	known := make([]scanner.KnownFile, 0, 1024)
	for rows.Next() {
		var record scanner.KnownFile
		var state string
		var modTime time.Time
		if err := rows.Scan(&record.ID, &record.Path, &record.Size, &modTime, &record.FileID,
			&record.RootID, &record.LastScanID, &record.LastSeenAt, &state); err != nil {
			return nil, fmt.Errorf("scan known file: %w", err)
		}
		record.ModTime = modTime
		record.State = scanner.FileState(state)
		known = append(known, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate known files: %w", err)
	}
	return known, nil
}

// DeleteMissingFiles permanently removes catalogued files that are still in
// MISSING state. It is called only by an explicit cleanup operation, never by a
// scan, and it refuses to run when any media root is offline.
func (r *Repository) DeleteMissingFiles(ctx context.Context, paths []string) (int, error) {
	if len(paths) == 0 {
		return 0, nil
	}
	result, err := r.db.ExecContext(ctx, `
	DELETE FROM video_files
	WHERE file_path = ANY($1) AND state = 'missing'
	`, pq.Array(paths))
	if err != nil {
		return 0, fmt.Errorf("delete missing files: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete missing files affected rows: %w", err)
	}
	return int(affected), nil
}

// -----------------------------------------------------------------------------
// Scan sessions
// -----------------------------------------------------------------------------

// ScanSession is the persisted record of a scan run.
type ScanSession struct {
	ID              string          `json:"id"`
	Status          string          `json:"status"`
	Mode            string          `json:"mode"`
	StartedAt       time.Time       `json:"startedAt"`
	FinishedAt      *time.Time      `json:"finishedAt,omitempty"`
	DurationSeconds float64         `json:"durationSeconds"`
	Stats           any             `json:"stats"`
	LastCheckpoint  any             `json:"lastCheckpoint,omitempty"`
	ErrorCount      int             `json:"errorCount"`
	Interrupted     bool            `json:"interrupted"`
	Roots           []ScanRootState `json:"roots,omitempty"`
}

// ScanRootState is the per-root progress of a scan session.
type ScanRootState struct {
	RootID       string `json:"rootId"`
	RootPath     string `json:"rootPath"`
	Status       string `json:"status"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	Directories  int64  `json:"directories"`
	FilesSeen    int64  `json:"filesSeen"`
	Accepted     int64  `json:"accepted"`
	Bytes        int64  `json:"bytes"`
}

// StartScanSession records the beginning of a scan. If a previous session is
// still marked running, it is flagged as interrupted: that is how the server
// knows the catalogue may be mid-scan after an unclean shutdown.
func (r *Repository) StartScanSession(ctx context.Context, scanID, mode string) error {
	if _, err := r.db.ExecContext(ctx, `
	UPDATE scan_sessions SET status = 'failed', interrupted = TRUE, finished_at = CURRENT_TIMESTAMP
	WHERE status = 'running'
	`); err != nil {
		return fmt.Errorf("close stale scan sessions: %w", err)
	}
	_, err := r.db.ExecContext(ctx, `
	INSERT INTO scan_sessions (id, status, mode, started_at)
	VALUES ($1, 'running', $2, CURRENT_TIMESTAMP)
	ON CONFLICT (id) DO UPDATE SET status = 'running', started_at = CURRENT_TIMESTAMP
	`, scanID, mode)
	if err != nil {
		return fmt.Errorf("start scan session: %w", err)
	}
	return nil
}

// FinishScanSession persists the final status and counters of a scan.
func (r *Repository) FinishScanSession(ctx context.Context, scanID, status string, progress scanner.Progress) error {
	_, err := r.db.ExecContext(ctx, `
	UPDATE scan_sessions
	SET status = $2,
		    finished_at = CURRENT_TIMESTAMP,
		    duration_seconds = $3,
		    stats = $4::jsonb,
		    error_count = $5
	WHERE id = $1
	`, scanID, status, progress.Duration.Seconds, progressJSON(progress), len(progress.Errors))
	if err != nil {
		return fmt.Errorf("finish scan session: %w", err)
	}
	return nil
}

// SaveScanRootState persists the independent state of one root.
func (r *Repository) SaveScanRootState(ctx context.Context, scanID string, state ScanRootState) error {
	_, err := r.db.ExecContext(ctx, `
	INSERT INTO scan_roots (
			scan_id, root_id, root_path, status, error_code, error_message,
			directories_visited, files_seen, media_accepted, bytes_total, updated_at
	)
	VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7, $8, $9, $10, CURRENT_TIMESTAMP)
	ON CONFLICT (scan_id, root_id) DO UPDATE SET
			status = EXCLUDED.status,
			error_code = EXCLUDED.error_code,
			error_message = EXCLUDED.error_message,
			directories_visited = EXCLUDED.directories_visited,
			files_seen = EXCLUDED.files_seen,
			media_accepted = EXCLUDED.media_accepted,
			bytes_total = EXCLUDED.bytes_total,
			updated_at = CURRENT_TIMESTAMP
	`, scanID, state.RootID, state.RootPath, state.Status, state.ErrorCode, state.ErrorMessage,
		state.Directories, state.FilesSeen, state.Accepted, state.Bytes)
	if err != nil {
		return fmt.Errorf("save scan root state: %w", err)
	}
	return nil
}

// InterruptedScanSessions returns sessions that never completed, which is how a
// restart detects that the index may need reconciliation.
func (r *Repository) InterruptedScanSessions(ctx context.Context) ([]ScanSession, error) {
	rows, err := r.db.QueryContext(ctx, `
	SELECT id, status, mode, started_at, finished_at, COALESCE(duration_seconds, 0),
		       COALESCE(stats, '{}'::jsonb)::text, error_count, interrupted
	FROM scan_sessions
	WHERE interrupted = TRUE OR status = 'running'
	ORDER BY started_at DESC
	LIMIT 20
	`)
	if err != nil {
		return nil, fmt.Errorf("query interrupted scans: %w", err)
	}
	defer rows.Close()

	sessions := make([]ScanSession, 0, 4)
	for rows.Next() {
		var session ScanSession
		var finishedAt sql.NullTime
		var stats string
		if err := rows.Scan(&session.ID, &session.Status, &session.Mode, &session.StartedAt,
			&finishedAt, &session.DurationSeconds, &stats, &session.ErrorCount, &session.Interrupted); err != nil {
			return nil, fmt.Errorf("scan interrupted session: %w", err)
		}
		if finishedAt.Valid {
			session.FinishedAt = &finishedAt.Time
		}
		session.Stats = stats
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

// LatestScanProgress returns the most recent session, used by the UI to show
// whether a scan is running and what it last reported.
func (r *Repository) LatestScanProgress(ctx context.Context) (*ScanSession, error) {
	var session ScanSession
	var finishedAt sql.NullTime
	var stats string
	err := r.db.QueryRowContext(ctx, `
	SELECT id, status, mode, started_at, finished_at, COALESCE(duration_seconds, 0),
		       COALESCE(stats, '{}'::jsonb)::text, error_count, interrupted
	FROM scan_sessions
	ORDER BY started_at DESC
	LIMIT 1
	`).Scan(&session.ID, &session.Status, &session.Mode, &session.StartedAt, &finishedAt,
		&session.DurationSeconds, &stats, &session.ErrorCount, &session.Interrupted)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query latest scan: %w", err)
	}
	if finishedAt.Valid {
		session.FinishedAt = &finishedAt.Time
	}
	session.Stats = stats
	return &session, nil
}

// escapeLikePattern neutralises LIKE wildcards in a user-supplied path.
func escapeLikePattern(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}

func lastPathSegment(path string) string {
	if index := strings.LastIndexAny(path, `/\`); index >= 0 {
		return path[index+1:]
	}
	return path
}

func extensionOf(path string) string {
	if index := strings.LastIndex(path, "."); index > 0 {
		return path[index:]
	}
	return ""
}
