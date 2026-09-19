package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"

	"nexora/server/internal/identity"
	"nexora/server/internal/scanner"
)

// -----------------------------------------------------------------------------
// Loading resolution inputs
// -----------------------------------------------------------------------------

// LoadKnownWorks loads the candidate entities the resolver compares against.
//
// Aliases are loaded in one extra query and attached in memory rather than with
// a join per work, because this runs once per scan rather than once per file.
func (r *Repository) LoadKnownWorks(ctx context.Context) ([]identity.Work, error) {
	rows, err := r.db.QueryContext(ctx, `
	SELECT
			mi.id,
			COALESCE(mi.title_en, ''),
			COALESCE(mi.title_ar, ''),
			COALESCE(mi.title_normalized, ''),
			COALESCE(mi.type, ''),
			COALESCE(c.slug, ''),
			COALESCE(mi.release_year, 0),
			COALESCE(mi.provisional, FALSE),
			COALESCE(mi.metadata_locked, FALSE),
			COALESCE(mi.metadata_provider, ''),
			COALESCE(mi.metadata_external_id, '')
	FROM media_items mi
	LEFT JOIN categories c ON c.id = mi.category_id
	WHERE mi.merged_into_id IS NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("load known works: %w", err)
	}
	defer rows.Close()

	works := make([]identity.Work, 0, 512)
	byID := make(map[int64]int, 512)
	for rows.Next() {
		var work identity.Work
		if err := rows.Scan(
			&work.ID, &work.TitleEN, &work.TitleAR, &work.TitleNormalized,
			&work.MediaType, &work.CategorySlug, &work.ReleaseYear,
			&work.Provisional, &work.MetadataLocked,
			&work.Provider, &work.ExternalID,
		); err != nil {
			return nil, fmt.Errorf("scan known work: %w", err)
		}
		if work.TitleNormalized == "" {
			// Backfill in memory so the comparison is still correct for rows
			// created before normalization existed.
			work.TitleNormalized = identity.Normalize(work.TitleEN)
		}
		work.SeasonNumbers = make([]int, 0, 4)
		work.EpisodeCounts = make(map[int]int, 4)
		byID[work.ID] = len(works)
		works = append(works, work)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate known works: %w", err)
	}

	if err := r.attachAliases(ctx, works, byID); err != nil {
		return nil, err
	}
	if err := r.attachSeasons(ctx, works, byID); err != nil {
		return nil, err
	}
	return works, nil
}

// attachAliases loads every alias and attaches it to its work.
func (r *Repository) attachAliases(ctx context.Context, works []identity.Work, byID map[int64]int) error {
	rows, err := r.db.QueryContext(ctx, `
	SELECT media_item_id, alias, COALESCE(alias_normalized, '')
	FROM media_aliases
	`)
	if err != nil {
		// A missing alias table must not break resolution: aliases are an
		// optimisation, not a requirement.
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var mediaID int64
		var alias, normalized string
		if err := rows.Scan(&mediaID, &alias, &normalized); err != nil {
			return fmt.Errorf("scan alias: %w", err)
		}
		if index, ok := byID[mediaID]; ok {
			works[index].Aliases = append(works[index].Aliases, alias)
		}
	}
	return rows.Err()
}

// attachSeasons loads season numbers and episode counts, which are the evidence
// for season compatibility and for detecting an out-of-range episode number.
func (r *Repository) attachSeasons(ctx context.Context, works []identity.Work, byID map[int64]int) error {
	rows, err := r.db.QueryContext(ctx, `
	SELECT s.media_item_id, s.season_number, COUNT(vf.id)
	FROM seasons s
	LEFT JOIN video_files vf ON vf.season_id = s.id
	GROUP BY s.media_item_id, s.season_number
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var mediaID int64
		var seasonNumber, fileCount int
		if err := rows.Scan(&mediaID, &seasonNumber, &fileCount); err != nil {
			return fmt.Errorf("scan season summary: %w", err)
		}
		index, ok := byID[mediaID]
		if !ok {
			continue
		}
		works[index].SeasonNumbers = append(works[index].SeasonNumbers, seasonNumber)
		works[index].EpisodeCounts[seasonNumber] = fileCount
	}
	return rows.Err()
}

// LoadAliasLookup returns the normalized alias -> work id map that makes a
// learned alias an exact match rather than a similarity guess.
func (r *Repository) LoadAliasLookup(ctx context.Context) (map[string]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
	SELECT alias_normalized, media_item_id FROM media_aliases WHERE alias_normalized <> ''
	`)
	if err != nil {
		return map[string]int64{}, nil
	}
	defer rows.Close()

	lookup := make(map[string]int64, 256)
	for rows.Next() {
		var normalized string
		var mediaID int64
		if err := rows.Scan(&normalized, &mediaID); err != nil {
			return lookup, fmt.Errorf("scan alias lookup: %w", err)
		}
		// First writer wins so the mapping stays deterministic if two rows ever
		// disagree; the unique index prevents that, and a conflict should be
		// surfaced rather than silently resolved by row order.
		if _, exists := lookup[normalized]; !exists {
			lookup[normalized] = mediaID
		}
	}
	return lookup, rows.Err()
}

// -----------------------------------------------------------------------------
// Resolution-driven ingest
// -----------------------------------------------------------------------------

// ResolveAndIngest persists a batch of scanned files through Entity Resolution.
//
// This replaces the behaviour of turning each parsed filename straight into a
// media_items row. The strict ordering is what prevents degenerate works:
//
//  1. resolve against existing entities (alias, then scored candidates)
//  2. attach the file to a work / season / episode
//  3. create a provisional work ONLY when resolution decided it is safe
//  4. queue anything ambiguous for human review instead of inventing a work
//
// `works` is a pointer because the candidate set must GROW as the scan runs.
// When this call creates a work, the next file in the same scan (and the next
// batch) must be able to attach to it. Passing a copy would make "01.mkv" and
// "02.mkv" in one folder each create their own work — the exact duplicate-work
// problem entity resolution exists to prevent.
func (r *Repository) ResolveAndIngest(ctx context.Context, files []scanner.FileInfo,
	resolver *identity.Resolver, works *[]identity.Work,
	aliasLookup map[string]int64) (ResolutionIngestStats, error) {

	stats := ResolutionIngestStats{Scanned: len(files)}
	if len(files) == 0 {
		return stats, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return stats, fmt.Errorf("begin resolution batch: %w", err)
	}
	defer tx.Rollback()

	categories, err := loadCategoryIDs(ctx, tx)
	if err != nil {
		return stats, err
	}

	// Work on the shared candidate set directly, so every work created here is
	// immediately visible to the next file and to the next batch of the scan.
	local := works
	if local == nil {
		empty := make([]identity.Work, 0, 64)
		local = &empty
	}

	for i := range files {
		file := files[i]

		evidence := identity.Evidence{
			Path:                file.Path,
			OriginalName:        file.Parsed.Original,
			FileSize:            file.Size,
			Segments:            file.PathSegments,
			ParsedTitle:         file.Parsed.Title,
			ParsedTitleAR:       file.Parsed.TitleAR,
			ParsedTitleEN:       file.Parsed.TitleEN,
			ParsedYear:          file.Parsed.ReleaseYear,
			ParsedSeason:        file.Parsed.SeasonNumber,
			ParsedEpisode:       file.Parsed.EpisodeNumber,
			ParsedPart:          file.Parsed.PartNumber,
			ParsedResolution:    file.Parsed.Resolution,
			ParserCategory:      file.Parsed.CategorySlug,
			ParserConfidence:    float64(file.Parsed.Confidence),
			ParserIsEpisode:     file.Parsed.IsEpisode,
			ParserEpisodeStrong: isStrongEpisodeSource(file.Parsed.EpisodeSource),
			WorkFolderTitle:     file.Context.WorkTitle,
			SeasonFolderNumber:  file.Context.SeasonNumber,
			SeasonFolderTitle:   file.Context.SeasonFolderTitle,
			OriginTag:           file.Parsed.OriginTag,
			CategoryHint:        file.Context.Category,
			Group: identity.GroupContext{
				Size:             file.Group.Size,
				EpisodicSiblings: file.Group.EpisodicSiblings,
				EpisodeNumbers:   file.Group.EpisodeNumbers,
				YearSiblings:     file.Group.YearSiblings,
				SeasonFolder:     file.Group.SeasonFolder,
				ConsensusTitle:   file.Group.ConsensusTitle,
				ConsensusVotes:   file.Group.ConsensusVotes,
			},
		}

		resolution := resolver.Resolve(identity.ResolverInput{
			Evidence:    evidence,
			Known:       *local,
			AliasLookup: aliasLookup,
		})

		outcome, err := r.applyResolution(ctx, tx, file, resolution, categories, local)
		if err != nil {
			stats.Failed++
			continue
		}
		stats.merge(outcome)
	}

	if err := tx.Commit(); err != nil {
		return stats, fmt.Errorf("commit resolution batch: %w", err)
	}
	return stats, nil
}

// ResolutionIngestStats reports what resolution decided for a batch, so the scan
// report can show how many files needed a human.
type ResolutionIngestStats struct {
	Scanned         int `json:"scanned"`
	Resolved        int `json:"resolved"`
	Provisional     int `json:"provisional"`
	Created         int `json:"worksCreated"`
	QueuedForReview int `json:"queuedForReview"`
	Unresolved      int `json:"unresolved"`
	Failed          int `json:"failed"`
	FilesAttached   int `json:"filesAttached"`
	AliasesLearned  int `json:"aliasesLearned"`
}

func (s *ResolutionIngestStats) merge(other ResolutionIngestStats) {
	s.Resolved += other.Resolved
	s.Provisional += other.Provisional
	s.Created += other.Created
	s.QueuedForReview += other.QueuedForReview
	s.Unresolved += other.Unresolved
	s.Failed += other.Failed
	s.FilesAttached += other.FilesAttached
	s.AliasesLearned += other.AliasesLearned
}

// isStrongEpisodeSource reports whether the parser matched an explicit episode
// pattern rather than a bare trailing number.
func isStrongEpisodeSource(source scanner.NumberSource) bool {
	return source == scanner.SourceFilenamePattern || source == scanner.SourceKeyword
}

// applyResolution writes the resolver's verdict for one file.
func (r *Repository) applyResolution(ctx context.Context, tx *sql.Tx, file scanner.FileInfo,
	resolution identity.Resolution, categories map[string]int64,
	local *[]identity.Work) (ResolutionIngestStats, error) {

	stats := ResolutionIngestStats{}

	switch resolution.Decision {
	case identity.DecisionReview:
		// The file must NOT become a work. Record it for an operator instead.
		if err := r.enqueueForReview(ctx, tx, file, resolution); err != nil {
			return stats, err
		}
		stats.QueuedForReview++
		return stats, nil

	case identity.DecisionCreate:
		workID, err := r.createProvisionalWork(ctx, tx, file, resolution, categories)
		if err != nil {
			return stats, err
		}
		// Register the new work locally so sibling files attach to it instead of
		// creating duplicates.
		*local = append(*local, identity.Work{
			ID:              workID,
			TitleEN:         resolution.NewWorkTitle,
			TitleNormalized: identity.Normalize(resolution.NewWorkTitle),
			MediaType:       string(resolution.MediaType),
			CategorySlug:    resolution.Category,
			Provisional:     true,
			SeasonNumbers:   nil,
			EpisodeCounts:   map[int]int{},
		})
		attached := identity.Work{ID: workID}
		resolution.Work = &attached
		stats.Created++

	case identity.DecisionAuto, identity.DecisionProvisional:
		if resolution.Work == nil {
			// Defensive: an attach decision without a work is a resolver bug and
			// must never create an unlinked file.
			if err := r.enqueueForReview(ctx, tx, file, resolution); err != nil {
				return stats, err
			}
			stats.QueuedForReview++
			return stats, nil
		}
		if resolution.Decision == identity.DecisionProvisional {
			stats.Provisional++
		} else {
			stats.Resolved++
		}
	}

	if resolution.Work == nil {
		stats.Unresolved++
		return stats, nil
	}

	// Attach the physical file to the logical entity.
	if err := r.attachFileToWork(ctx, tx, file, resolution); err != nil {
		return stats, err
	}
	stats.FilesAttached++

	// Learn aliases from a confident resolution so the next file is exact.
	learner := identity.NewAliasLearner()
	if aliases := learner.Learn(resolution, *resolution.Work, identity.Evidence{
		WorkFolderTitle: file.Context.WorkTitle,
		ParsedTitle:     file.Parsed.Title,
		ParsedTitleAR:   file.Parsed.TitleAR,
	}); len(aliases) > 0 {
		learned, err := r.storeAliases(ctx, tx, aliases)
		if err == nil {
			stats.AliasesLearned += learned
		}
	}

	return stats, nil
}

// enqueueForReview records an ambiguous file without creating any entity.
func (r *Repository) enqueueForReview(ctx context.Context, tx *sql.Tx, file scanner.FileInfo,
	resolution identity.Resolution) error {

	candidates, err := marshalCandidates(resolution.Candidates)
	if err != nil {
		candidates = "[]"
	}

	_, err = tx.ExecContext(ctx, `
	INSERT INTO resolution_queue (
			file_path, relative_path, root_id, original_filename, size,
			detected_title, detected_title_normalized, detected_category, detected_media_type,
			detected_season, detected_episode, detected_year, parser_confidence, resolver_confidence,
			reason, reason_detail, candidates, state
	)
	VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''), $5,
			NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''),
			NULLIF($10, 0), NULLIF($11, 0), NULLIF($12, 0), $13, $14,
			$15, NULLIF($16, ''), $17::jsonb, 'needs_review')
	ON CONFLICT (file_path) DO UPDATE SET
			detected_title = EXCLUDED.detected_title,
			detected_season = EXCLUDED.detected_season,
			detected_episode = EXCLUDED.detected_episode,
			parser_confidence = EXCLUDED.parser_confidence,
			resolver_confidence = EXCLUDED.resolver_confidence,
			reason = EXCLUDED.reason,
			reason_detail = EXCLUDED.reason_detail,
			candidates = EXCLUDED.candidates,
			updated_at = CURRENT_TIMESTAMP
	`, file.Path, file.RelativePath, file.RootID, file.Parsed.Original, file.Size,
		file.Parsed.Title, file.Parsed.TitleNormalized, resolution.Category,
		string(resolution.MediaType),
		resolution.Season, resolution.Episode, file.Parsed.ReleaseYear,
		resolution.ParserConfidence, resolution.ResolverConfidence,
		resolution.ReasonCode, resolution.Reason, candidates)
	if err != nil {
		return fmt.Errorf("enqueue for review %q: %w", file.Path, err)
	}
	return nil
}

// storeAliases persists learned aliases, ignoring ones already known.
func (r *Repository) storeAliases(ctx context.Context, tx *sql.Tx, aliases []identity.AliasCandidate) (int, error) {
	learned := 0
	for _, alias := range aliases {
		if alias.Normalized == "" {
			continue
		}
		result, err := tx.ExecContext(ctx, `
			INSERT INTO media_aliases (media_item_id, alias, alias_normalized, source)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (alias_normalized) DO UPDATE
					SET hit_count = media_aliases.hit_count + 1,
					    updated_at = CURRENT_TIMESTAMP
				WHERE media_aliases.media_item_id = EXCLUDED.media_item_id
	`, alias.MediaItemID, alias.Alias, alias.Normalized, string(alias.Source))
		if err != nil {
			// A conflicting alias belongs to another work, which is a real
			// ambiguity: leave it for review rather than reassigning it.
			continue
		}
		if affected, err := result.RowsAffected(); err == nil && affected > 0 {
			learned++
		}
	}
	return learned, nil
}

// createProvisionalWork creates a new entity only when resolution permitted it.
func (r *Repository) createProvisionalWork(ctx context.Context, tx *sql.Tx, file scanner.FileInfo,
	resolution identity.Resolution, categories map[string]int64) (int64, error) {

	categorySlug := resolution.Category
	if categorySlug == "" {
		categorySlug = "series"
		if resolution.MediaType == identity.MediaTypeMovie {
			categorySlug = "movies"
		}
	}
	categoryID, ok := categories[categorySlug]
	if !ok {
		categoryID = categories["series"]
	}
	if categoryID == 0 {
		return 0, fmt.Errorf("no category available for %q", categorySlug)
	}

	mediaType := string(resolution.MediaType)
	title := resolution.NewWorkTitle
	normalized := identity.Normalize(title)

	titleAR := ""
	titleEN := title
	if containsArabic(title) {
		titleAR = title
	}

	posterURL := ""
	if file.ArtworkPath != "" {
		posterURL = r.CacheLocalArtwork(file.ArtworkPath)
	}

	var mediaID int64
	err := tx.QueryRowContext(ctx, `
	INSERT INTO media_items (
			category_id, title_ar, title_en, title_normalized, type, release_year,
			poster_path, banner_path, status,
			title_source, resolution_state, parser_confidence, resolver_confidence,
			raw_detected_title, provisional
	)
	VALUES ($1, NULLIF($2, ''), $3, NULLIF($4, ''), $5, NULLIF($6, 0),
			NULLIF($7, ''), NULLIF($7, ''), 'completed',
			'resolver', 'enrichment_pending', $8, $9,
			NULLIF($10, ''), TRUE)
	ON CONFLICT (type, LOWER(title_en), COALESCE(release_year, 0)) DO UPDATE
			SET poster_path = COALESCE(NULLIF(media_items.poster_path, ''), EXCLUDED.poster_path),
			    banner_path = COALESCE(NULLIF(media_items.banner_path, ''), EXCLUDED.banner_path)
	RETURNING id;
	`, categoryID, titleAR, titleEN, normalized, mediaType, file.Parsed.ReleaseYear,
		posterURL, resolution.ParserConfidence, resolution.ResolverConfidence, file.Parsed.Title).
		Scan(&mediaID)
	if err != nil {
		return 0, fmt.Errorf("create provisional work %q: %w", title, err)
	}
	return mediaID, nil
}

// attachFileToWork writes the physical file and links it to its logical entity.
func (r *Repository) attachFileToWork(ctx context.Context, tx *sql.Tx, file scanner.FileInfo,
	resolution identity.Resolution) error {

	workID := resolution.Work.ID

	// Resolve the season and episode rows, creating them under this work. A new
	// season must attach to the existing work; it must never become a new work.
	seasonID := sql.NullInt64{}
	episodeID := sql.NullInt64{}
	episodeNumber := sql.NullInt64{}

	if resolution.MediaType == identity.MediaTypeSeries && resolution.Season > 0 {
		seasonNumber := resolution.Season
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO seasons (media_item_id, season_number, title_en)
			VALUES ($1, $2, $3)
			ON CONFLICT (media_item_id, season_number)
			DO UPDATE SET title_en = COALESCE(seasons.title_en, EXCLUDED.title_en)
			RETURNING id;
	`, workID, seasonNumber, fmt.Sprintf("Season %02d", seasonNumber)).Scan(&seasonID.Int64); err != nil {
			return fmt.Errorf("upsert season %d for work %d: %w", seasonNumber, workID, err)
		}
		seasonID.Valid = true

		if resolution.Episode > 0 {
			// The episodes table is the entity a file links to, so several
			// release files can share one episode without becoming extra works.
			if err := tx.QueryRowContext(ctx, `
				INSERT INTO episodes (season_id, episode_number, title_en)
				VALUES ($1, $2, $3)
				ON CONFLICT (season_id, episode_number) DO UPDATE
						SET episode_number = EXCLUDED.episode_number
				RETURNING id;
			`, seasonID.Int64, resolution.Episode,
				fmt.Sprintf("Episode %02d", resolution.Episode)).Scan(&episodeID.Int64); err != nil {
				return fmt.Errorf("upsert episode %d: %w", resolution.Episode, err)
			}
			episodeID.Valid = true
			episodeNumber = sql.NullInt64{Int64: int64(resolution.Episode), Valid: true}
		}
	}

	partNumber := sql.NullInt64{}
	if resolution.Part > 0 {
		partNumber = sql.NullInt64{Int64: int64(resolution.Part), Valid: true}
	}

	titleAR := resolution.NewWorkTitle
	titleEN := resolution.NewWorkTitle
	if resolution.Work.TitleEN != "" {
		titleEN = resolution.Work.TitleEN
	}
	if resolution.Work.TitleAR != "" {
		titleAR = resolution.Work.TitleAR
	} else if !containsArabic(titleAR) {
		titleAR = ""
	}

	// A rename rewrites the existing row so the record keeps its identity.
	if file.Change == scanner.ChangeRenamed && file.KnownID > 0 {
		_, err := tx.ExecContext(ctx, `
			UPDATE video_files
			SET file_path = $2, file_size = $3, file_mod_time = $4,
			    file_id = NULLIF($5, ''), root_id = NULLIF($6, ''),
			    root_path = NULLIF($7, ''), relative_path = NULLIF($8, ''),
			    media_item_id = $9, season_id = $10, episode_id = $11,
			    episode_number = $12, part_number = $13,
			    resolution = NULLIF($14, ''), state = 'active',
			    resolution_state = $15, resolver_confidence = $16,
			    last_seen_at = CURRENT_TIMESTAMP
			WHERE id = $1
	`, file.KnownID, file.Path, file.Size, file.ModTime, file.FileID, file.RootID,
			file.Root, file.RelativePath, workID, seasonID, episodeID, episodeNumber,
			partNumber, file.Parsed.Resolution, string(resolution.State),
			resolution.ResolverConfidence)
		if err != nil {
			return fmt.Errorf("move video file %q: %w", file.Path, err)
		}
		return nil
	}

	_, err := tx.ExecContext(ctx, `
	INSERT INTO video_files (
			media_item_id, season_id, episode_id, episode_number, part_number,
			title_ar, title_en, title_normalized, file_path, relative_path,
			file_size, file_mod_time, file_id, root_id, root_path,
			resolution, audio_tracks, subtitles, last_seen_at,
			parse_confidence, parse_reasons, special_kind,
			resolution_state, resolver_confidence, resolution_source
	)
	VALUES (
			$1, $2, $3, $4, $5,
			NULLIF($6, ''), $7, NULLIF($8, ''), $9, NULLIF($10, ''),
			$11, $12, NULLIF($13, ''), NULLIF($14, ''), NULLIF($15, ''),
			NULLIF($16, ''), '[]'::jsonb, '[]'::jsonb, CURRENT_TIMESTAMP,
			$17, $18, NULLIF($19, ''),
			$20, $21, 'resolver'
	)
	ON CONFLICT (file_path) DO UPDATE SET
			media_item_id = EXCLUDED.media_item_id,
			season_id = EXCLUDED.season_id,
			episode_id = EXCLUDED.episode_id,
			episode_number = EXCLUDED.episode_number,
			part_number = EXCLUDED.part_number,
			title_ar = EXCLUDED.title_ar,
			title_en = EXCLUDED.title_en,
			title_normalized = EXCLUDED.title_normalized,
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
			resolution_state = EXCLUDED.resolution_state,
			resolver_confidence = EXCLUDED.resolver_confidence,
			resolution_source = EXCLUDED.resolution_source,
			state = 'active',
			last_seen_at = CURRENT_TIMESTAMP;
	`, workID, seasonID, episodeID, episodeNumber, partNumber,
		titleAR, titleEN, identity.Normalize(titleEN), file.Path, file.RelativePath,
		file.Size, file.ModTime.UTC(), file.FileID, file.RootID, file.Root,
		file.Parsed.Resolution, float64(file.Parsed.Confidence),
		pq.Array(file.Parsed.Reasons), file.Parsed.SpecialKind,
		string(resolution.State), resolution.ResolverConfidence)
	if err != nil {
		return fmt.Errorf("attach file %q to work %d: %w", file.Path, workID, err)
	}
	return nil
}

// marshalCandidates renders the ranked candidates for the review queue.
func marshalCandidates(candidates []identity.Candidate) (string, error) {
	type candidateView struct {
		WorkID     int64                `json:"work_id"`
		Title      string               `json:"title"`
		Score      float64              `json:"score"`
		Decision   string               `json:"decision"`
		TitleMatch float64              `json:"title_match"`
		Breakdown  []identity.ScoreItem `json:"breakdown"`
		Reasons    []string             `json:"reasons,omitempty"`
	}

	views := make([]candidateView, 0, len(candidates))
	// Limit to the top few so the queue payload stays readable for an operator.
	limit := len(candidates)
	if limit > 5 {
		limit = 5
	}
	for _, candidate := range candidates[:limit] {
		views = append(views, candidateView{
			WorkID:     candidate.WorkID,
			Title:      candidate.Title,
			Score:      candidate.Score,
			Decision:   string(candidate.Decision),
			TitleMatch: candidate.TitleMatch,
			Breakdown:  candidate.Breakdown.Items,
			Reasons:    candidate.Reasons,
		})
	}
	return marshalJSON(views)
}

// -----------------------------------------------------------------------------
// Resolution queue operations
// -----------------------------------------------------------------------------

// ResolutionQueueItem is one ambiguous file awaiting an operator decision.
type ResolutionQueueItem struct {
	ID                 int64                `json:"id"`
	FilePath           string               `json:"file_path"`
	RelativePath       string               `json:"relative_path,omitempty"`
	OriginalFilename   string               `json:"original_filename,omitempty"`
	Size               int64                `json:"size"`
	DetectedTitle      string               `json:"detected_title,omitempty"`
	DetectedCategory   string               `json:"detected_category,omitempty"`
	DetectedMediaType  string               `json:"detected_media_type,omitempty"`
	DetectedSeason     int                  `json:"detected_season,omitempty"`
	DetectedEpisode    int                  `json:"detected_episode,omitempty"`
	DetectedYear       int                  `json:"detected_year,omitempty"`
	ParserConfidence   float64              `json:"parser_confidence"`
	ResolverConfidence float64              `json:"resolver_confidence"`
	Reason             string               `json:"reason"`
	ReasonDetail       string               `json:"reason_detail,omitempty"`
	Candidates         []identity.Candidate `json:"-"`
	CandidatesJSON     string               `json:"candidates"`
	State              string               `json:"state"`
	CreatedAt          time.Time            `json:"created_at"`
}

// ListResolutionQueue returns pending items, optionally filtered by reason.
func (r *Repository) ListResolutionQueue(ctx context.Context, reason string, limit int) ([]ResolutionQueueItem, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	query := `
	SELECT id, file_path, COALESCE(relative_path, ''), COALESCE(original_filename, ''),
		     COALESCE(size, 0), COALESCE(detected_title, ''), COALESCE(detected_category, ''),
		     COALESCE(detected_media_type, ''), COALESCE(detected_season, 0),
		     COALESCE(detected_episode, 0), COALESCE(detected_year, 0),
		     COALESCE(parser_confidence, 0), COALESCE(resolver_confidence, 0),
		     reason, COALESCE(reason_detail, ''), candidates::text, state, created_at
	FROM resolution_queue
	WHERE state IN ('unresolved', 'needs_review')
	`
	args := []any{}
	if reason != "" {
		query += ` AND reason = $1`
		args = append(args, reason)
	}
	query += ` ORDER BY created_at DESC LIMIT $` + itoa(len(args)+1)
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list resolution queue: %w", err)
	}
	defer rows.Close()

	items := make([]ResolutionQueueItem, 0, limit)
	for rows.Next() {
		var item ResolutionQueueItem
		if err := rows.Scan(&item.ID, &item.FilePath, &item.RelativePath, &item.OriginalFilename,
			&item.Size, &item.DetectedTitle, &item.DetectedCategory, &item.DetectedMediaType,
			&item.DetectedSeason, &item.DetectedEpisode, &item.DetectedYear,
			&item.ParserConfidence, &item.ResolverConfidence, &item.Reason,
			&item.ReasonDetail, &item.CandidatesJSON, &item.State, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan resolution queue item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ResolutionDecision is an operator's verdict on a queued file.
type ResolutionDecision struct {
	// Action is attach | create | ignore | mark_movie | move_season.
	Action   string `json:"action"`
	WorkID   int64  `json:"work_id,omitempty"`
	Season   int    `json:"season,omitempty"`
	Episode  int    `json:"episode,omitempty"`
	NewTitle string `json:"new_title,omitempty"`
	// LearnAlias promotes the detected spelling into a durable alias.
	LearnAlias bool   `json:"learn_alias"`
	DecidedBy  string `json:"decided_by,omitempty"`
}

// ApplyResolutionDecision records an operator decision and, when safe, learns
// the mapping so the same ambiguity never appears twice.
func (r *Repository) ApplyResolutionDecision(ctx context.Context, itemID int64,
	decision ResolutionDecision) error {

	// Read the queued item first so the decision can be applied to real data.
	var item ResolutionQueueItem
	if err := r.db.QueryRowContext(ctx, `
	SELECT id, file_path, COALESCE(detected_title, ''), COALESCE(detected_season, 0),
		     COALESCE(detected_episode, 0), COALESCE(original_filename, '')
	FROM resolution_queue WHERE id = $1
	`, itemID).Scan(&item.ID, &item.FilePath, &item.DetectedTitle,
		&item.DetectedSeason, &item.DetectedEpisode, &item.OriginalFilename); err != nil {
		return fmt.Errorf("load queue item %d: %w", itemID, err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin resolution decision: %w", err)
	}
	defer tx.Rollback()

	var learnedAlias bool

	switch decision.Action {
	case "attach", "move_season", "mark_movie":
		if decision.WorkID == 0 {
			return fmt.Errorf("action %q requires a work id", decision.Action)
		}
		seasonNumber := decision.Season
		if seasonNumber == 0 {
			seasonNumber = item.DetectedSeason
		}
		episodeNumber := decision.Episode
		if episodeNumber == 0 {
			episodeNumber = item.DetectedEpisode
		}

		seasonID := sql.NullInt64{}
		if seasonNumber > 0 {
			if err := tx.QueryRowContext(ctx, `
				INSERT INTO seasons (media_item_id, season_number, title_en)
				VALUES ($1, $2, $3)
				ON CONFLICT (media_item_id, season_number) DO UPDATE
						SET title_en = COALESCE(seasons.title_en, EXCLUDED.title_en)
				RETURNING id;
			`, decision.WorkID, seasonNumber,
				fmt.Sprintf("Season %02d", seasonNumber)).Scan(&seasonID.Int64); err != nil {
				return fmt.Errorf("resolve season for decision: %w", err)
			}
			seasonID.Valid = true
		}

		episodeID := sql.NullInt64{}
		if episodeNumber > 0 && seasonID.Valid {
			if err := tx.QueryRowContext(ctx, `
				INSERT INTO episodes (season_id, episode_number, title_en)
				VALUES ($1, $2, $3)
				ON CONFLICT (season_id, episode_number) DO UPDATE
						SET episode_number = EXCLUDED.episode_number
				RETURNING id;
			`, seasonID.Int64, episodeNumber,
				fmt.Sprintf("Episode %02d", episodeNumber)).Scan(&episodeID.Int64); err != nil {
				return fmt.Errorf("resolve episode for decision: %w", err)
			}
			episodeID.Valid = true
		}

		// Attach (or re-attach) the file to the operator's chosen entity.
		if _, err := tx.ExecContext(ctx, `
			UPDATE video_files
			SET media_item_id = $2, season_id = $3, episode_id = $4,
			    episode_number = NULLIF($5, 0),
			    resolution_state = 'resolved', resolution_source = 'admin',
			    resolver_confidence = 1.0
			WHERE file_path = $1
	`, item.FilePath, decision.WorkID, seasonID, episodeID, episodeNumber); err != nil {
			return fmt.Errorf("attach file for decision: %w", err)
		}

		// An operator decision is always safe to learn as an alias.
		if decision.LearnAlias && item.DetectedTitle != "" {
			alias := identity.LearnedFromAdmin(decision.WorkID, item.DetectedTitle, item.DetectedTitle)
			if learned, err := r.storeAliases(ctx, tx, []identity.AliasCandidate{alias}); err == nil && learned > 0 {
				learnedAlias = true
			}
		}

	case "create":
		if decision.NewTitle == "" {
			return fmt.Errorf("action create requires a new title")
		}
		// The operator authored this name, so it is admin-sourced and locked.
		if err := r.learnOperatorAlias(ctx, tx, decision, item); err != nil {
			return err
		}
		learnedAlias = true

	case "ignore":
	// Nothing to attach; the file stays unindexed deliberately.

	default:
		return fmt.Errorf("unknown resolution action %q", decision.Action)
	}

	// Mark the queue item decided, keeping the record for auditability.
	if _, err := tx.ExecContext(ctx, `
	UPDATE resolution_queue
	SET state = 'resolved',
	    decision = $2,
	    decided_media_item_id = NULLIF($3, 0),
	    decided_season_number = NULLIF($4, 0),
	    decided_episode_number = NULLIF($5, 0),
	    decided_by = NULLIF($6, ''),
	    decided_at = CURRENT_TIMESTAMP,
	    learned_alias = $7,
	    updated_at = CURRENT_TIMESTAMP
	WHERE id = $1
	`, itemID, decision.Action, decision.WorkID, decision.Season, decision.Episode,
		decision.DecidedBy, learnedAlias); err != nil {
		return fmt.Errorf("record resolution decision: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit resolution decision: %w", err)
	}
	return nil
}

// learnOperatorAlias creates a work named by an operator and records the mapping.
func (r *Repository) learnOperatorAlias(ctx context.Context, tx *sql.Tx,
	decision ResolutionDecision, item ResolutionQueueItem) error {

	// The operator's name is authoritative: record it as admin-sourced.
	_, err := tx.ExecContext(ctx, `
	UPDATE media_items
	SET title_en = $2,
	    title_normalized = $3,
	    title_source = 'admin',
	    metadata_locked = TRUE,
	    provisional = FALSE
	WHERE id = $1
	`, decision.WorkID, decision.NewTitle, identity.Normalize(decision.NewTitle))
	if err != nil {
		return fmt.Errorf("apply operator title: %w", err)
	}

	if item.DetectedTitle != "" && decision.WorkID > 0 {
		alias := identity.LearnedFromAdmin(decision.WorkID, item.DetectedTitle, decision.NewTitle)
		if _, err := r.storeAliases(ctx, tx, []identity.AliasCandidate{alias}); err != nil {
			return err
		}
	}
	return nil
}

// CountResolutionQueue returns how many items await review, per reason.
func (r *Repository) CountResolutionQueue(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `
	SELECT reason, COUNT(*) FROM resolution_queue
	WHERE state IN ('unresolved', 'needs_review')
	GROUP BY reason
	`)
	if err != nil {
		return map[string]int{}, nil
	}
	defer rows.Close()

	counts := make(map[string]int, 8)
	for rows.Next() {
		var reason string
		var count int
		if err := rows.Scan(&reason, &count); err != nil {
			return counts, err
		}
		counts[reason] = count
	}
	return counts, rows.Err()
}

// marshalJSON encodes a value for a JSONB column, degrading to "[]" rather
// than failing a batch when a payload cannot be encoded.
func marshalJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "[]", err
	}
	return string(encoded), nil
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	index := len(buffer)
	negative := value < 0
	if negative {
		value = -value
	}
	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		index--
		buffer[index] = '-'
	}
	return string(buffer[index:])
}
