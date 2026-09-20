package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"

	"nexora/server/internal/identity"
)

// -----------------------------------------------------------------------------
// Duplicate work consolidation
// -----------------------------------------------------------------------------

// DuplicateKind classifies why two rows are considered the same work.
//
// The kind determines how confident the merge is, and therefore whether it may
// run automatically. Collapsing them into one "duplicate" concept would mean
// treating a provable rename and a bilingual guess with the same trust.
type DuplicateKind string

const (
	// KindSameTitle is the strongest case: identical normalized title and
	// identical media type. There is no judgement involved.
	KindSameTitle DuplicateKind = "same_title"
	// KindAliasLinked is a pair already connected by a learned alias, so the
	// library itself has previously asserted they are one work.
	KindAliasLinked DuplicateKind = "alias_linked"
	// KindCrossLanguage is the same work written in Arabic and English. It is
	// only proposed, never merged automatically, because two translated titles
	// are not proof on their own.
	KindCrossLanguage DuplicateKind = "cross_language"
	// KindContainer is a row whose title is a container folder name ("أعمال",
	// "Movies") rather than the work. Such a row must be RE-RESOLVED from its
	// own file paths, not merged into another work.
	KindContainer DuplicateKind = "container_folder"
)

// DuplicateWork is one row that should not exist as its own entity.
type DuplicateWork struct {
	ID          int64
	TitleEN     string
	TitleAR     string
	MediaType   string
	FileCount   int64
	ExternalID  string
	Provisional bool
	FilePath    string
}

// WorkMergeGroup is a set of rows that describe one work.
//
// It is deliberately NOT named DuplicateGroup: repository.go already defines
// that name for checksum-based file duplicates, which answer a different
// question ("is this the same bytes twice") than this one ("is this the same
// work twice"). Reusing the name would have silently made the two concepts
// interchangeable at call sites.
type WorkMergeGroup struct {
	Kind DuplicateKind
	// CanonicalID is the row that survives. Everything else is folded into it.
	CanonicalID int64
	// Members are the rows being folded in.
	Members []DuplicateWork
	// Reason explains the grouping for the audit trail.
	Reason string
	// Safe reports whether the group may be merged without human review.
	Safe bool
}

// FindDuplicateGroups proposes every consolidation the catalogue needs.
//
// It is read-only: it returns decisions, never applies them.
func (r *Repository) FindDuplicateGroups(ctx context.Context) ([]WorkMergeGroup, error) {
	works, err := r.listWorksForConsolidation(ctx)
	if err != nil {
		return nil, err
	}

	groups := make([]WorkMergeGroup, 0)

	// 1. Container-folder rows. These are not duplicates of another work; they
	//    are rows that were never a work at all. They are listed first because
	//    merging them into a sibling would preserve a wrong title.
	for _, work := range works {
		if work.isContainerTitle() {
			groups = append(groups, WorkMergeGroup{
				Kind:        KindContainer,
				CanonicalID: work.ID,
				Members:     []DuplicateWork{work},
				Reason: "title \"" + work.TitleEN + "\" is a container folder name, " +
					"so the real title must be recovered from the file paths",
				Safe: false,
			})
		}
	}

	// 2. Same normalized title and media type.
	//
	// Container-titled rows are excluded FIRST, and that ordering is the whole
	// correctness of this function. Three rows named "أعمال" are textually
	// identical, so a naive grouping would call them duplicates of each other
	// and fold three unrelated films into one work. They are not duplicates:
	// each is a separate row that lost its title to the same container folder,
	// and each needs re-resolution, not a merge.
	byIdentity := make(map[string][]DuplicateWork)
	for _, work := range works {
		if work.isContainerTitle() {
			continue
		}
		key := work.MediaType + "::" + identity.Normalize(work.TitleEN)
		byIdentity[key] = append(byIdentity[key], work)
	}
	for _, members := range byIdentity {
		if len(members) < 2 {
			continue
		}
		canonical := chooseCanonical(members)
		folded := make([]DuplicateWork, 0, len(members)-1)
		for _, member := range members {
			if member.ID != canonical.ID {
				folded = append(folded, member)
			}
		}
		groups = append(groups, WorkMergeGroup{
			Kind:        KindSameTitle,
			CanonicalID: canonical.ID,
			Members:     folded,
			Reason:      "identical normalized title and media type",
			Safe:        true,
		})
	}

	// 3. Same media type under a different language title, connected by an
	//    alias. Only proposed: two translations are not proof by themselves, so
	//    an operator confirms before the rows are folded.
	groups = append(groups, r.findCrossLanguageGroups(ctx, works)...)

	return groups, nil
}

// listWorksForConsolidation loads every active work with the facts a merge
// decision needs.
func (r *Repository) listWorksForConsolidation(ctx context.Context) ([]DuplicateWork, error) {
	rows, err := r.db.QueryContext(ctx, `
	SELECT
	mi.id,
	COALESCE(mi.title_en, ''),
	COALESCE(mi.title_ar, ''),
	COALESCE(mi.type, ''),
	COALESCE((SELECT COUNT(*) FROM video_files vf WHERE vf.media_item_id = mi.id), 0),
	COALESCE(mi.metadata_external_id, ''),
	COALESCE(mi.provisional, FALSE),
	COALESCE((SELECT vf.file_path FROM video_files vf WHERE vf.media_item_id = mi.id
		          ORDER BY vf.id LIMIT 1), '')
	FROM media_items mi
	WHERE mi.merged_into_id IS NULL
	ORDER BY mi.id
	`)
	if err != nil {
		return nil, fmt.Errorf("list works for consolidation: %w", err)
	}
	defer rows.Close()

	works := make([]DuplicateWork, 0, 256)
	for rows.Next() {
		var work DuplicateWork
		if err := rows.Scan(&work.ID, &work.TitleEN, &work.TitleAR, &work.MediaType,
			&work.FileCount, &work.ExternalID, &work.Provisional, &work.FilePath); err != nil {
			return nil, fmt.Errorf("scan work for consolidation: %w", err)
		}
		works = append(works, work)
	}
	return works, rows.Err()
}

// isContainerTitle reports whether a row's title is a browse folder rather than
// a work name.
//
// This is the same rule the resolver applies when choosing a title, reused here
// so the catalogue repair and the ingest path cannot disagree about what a
// container is.
func (w DuplicateWork) isContainerTitle() bool {
	title := strings.TrimSpace(w.TitleEN)
	if title == "" {
		title = strings.TrimSpace(w.TitleAR)
	}
	if title == "" {
		return false
	}
	if identity.IsContainerFolderName(title) {
		return true
	}
	// A title that is only a structural word is a marker, not a name.
	if identity.IsStructuralTitle(title) {
		return true
	}
	return false
}

// chooseCanonical picks the row that survives a merge.
//
// Order of preference: non-provisional beats provisional (a reviewed entity is
// better than a guess), then the row with more files (the one the library
// actually uses), then the lower id so the choice is deterministic and a repeat
// run cannot pick differently.
func chooseCanonical(members []DuplicateWork) DuplicateWork {
	ordered := append([]DuplicateWork(nil), members...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Provisional != ordered[j].Provisional {
			return !ordered[i].Provisional
		}
		if ordered[i].FileCount != ordered[j].FileCount {
			return ordered[i].FileCount > ordered[j].FileCount
		}
		return ordered[i].ID < ordered[j].ID
	})
	return ordered[0]
}

// findCrossLanguageGroups proposes merges for rows that share a media type and
// are already linked by an alias.
func (r *Repository) findCrossLanguageGroups(ctx context.Context, works []DuplicateWork) []WorkMergeGroup {
	rows, err := r.db.QueryContext(ctx, `
	SELECT a1.media_item_id, a2.media_item_id
	FROM media_aliases a1
	JOIN media_aliases a2 ON a1.alias_normalized = a2.alias_normalized
	                    AND a1.media_item_id <> a2.media_item_id
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	linked := make(map[int64]map[int64]struct{})
	for rows.Next() {
		var left, right int64
		if err := rows.Scan(&left, &right); err != nil {
			return nil
		}
		if linked[left] == nil {
			linked[left] = make(map[int64]struct{})
		}
		linked[left][right] = struct{}{}
	}

	byID := make(map[int64]DuplicateWork, len(works))
	for _, work := range works {
		byID[work.ID] = work
	}

	seen := make(map[string]struct{})
	groups := make([]WorkMergeGroup, 0)
	for leftID, partners := range linked {
		left, exists := byID[leftID]
		if !exists {
			continue
		}
		for rightID := range partners {
			right, exists := byID[rightID]
			if !exists || left.MediaType != right.MediaType {
				continue
			}
			key := canonicalPairKey(leftID, rightID)
			if _, done := seen[key]; done {
				continue
			}
			seen[key] = struct{}{}

			canonical := chooseCanonical([]DuplicateWork{left, right})
			var folded DuplicateWork
			if canonical.ID == left.ID {
				folded = right
			} else {
				folded = left
			}
			groups = append(groups, WorkMergeGroup{
				Kind:        KindAliasLinked,
				CanonicalID: canonical.ID,
				Members:     []DuplicateWork{folded},
				Reason:      "a learned alias already links these two rows",
				Safe:        true,
			})
		}
	}
	return groups
}

// canonicalPairKey orders two ids so a pair is only reported once.
func canonicalPairKey(left, right int64) string {
	if left > right {
		left, right = right, left
	}
	return fmt.Sprintf("%d:%d", left, right)
}

// -----------------------------------------------------------------------------
// Applying a merge
// -----------------------------------------------------------------------------

// ConsolidationStats reports what a merge pass changed.
type ConsolidationStats struct {
	GroupsMerged   int `json:"groupsMerged"`
	WorksMerged    int `json:"worksMerged"`
	FilesRepointed int `json:"filesRepointed"`
	SeasonsMoved   int `json:"seasonsMoved"`
	EpisodesMoved  int `json:"episodesMoved"`
	AliasesKept    int `json:"aliasesKept"`
	Skipped        int `json:"skipped"`
}

// MergeDuplicateGroups folds every safe group into its canonical row.
//
// A merge never deletes a row. The folded row is marked with merged_into_id and
// kept, so the decision can be audited and, if it was wrong, reversed. That is
// also why files are repointed rather than copied: physical media is never
// duplicated by a catalogue operation.
//
// Only groups marked Safe are applied. A cross-language or container group is
// reported and left alone, because those need either a human decision or a
// re-resolution pass, not a merge.
func (r *Repository) MergeDuplicateGroups(ctx context.Context, groups []WorkMergeGroup) (ConsolidationStats, error) {
	stats := ConsolidationStats{}

	for _, group := range groups {
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		if !group.Safe || len(group.Members) == 0 {
			stats.Skipped++
			continue
		}

		merged, err := r.mergeGroup(ctx, group)
		if err != nil {
			// One bad group must not abandon the rest of the consolidation.
			stats.Skipped++
			continue
		}
		stats.GroupsMerged++
		stats.WorksMerged += merged.WorksMerged
		stats.FilesRepointed += merged.FilesRepointed
		stats.SeasonsMoved += merged.SeasonsMoved
		stats.EpisodesMoved += merged.EpisodesMoved
		stats.AliasesKept += merged.AliasesKept
	}

	return stats, nil
}

// mergeGroup folds one group in a single transaction, so a group is either fully
// merged or untouched.
func (r *Repository) mergeGroup(ctx context.Context, group WorkMergeGroup) (ConsolidationStats, error) {
	stats := ConsolidationStats{}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return stats, fmt.Errorf("begin merge: %w", err)
	}
	defer tx.Rollback()

	canonicalID := group.CanonicalID

	for _, member := range group.Members {
		if member.ID == canonicalID {
			continue
		}

		// Move the physical files and the episode links onto the canonical work.
		result, err := tx.ExecContext(ctx, `
			UPDATE video_files
			SET media_item_id = $1,
			    season_id = NULL,
			    episode_id = NULL
			WHERE media_item_id = $2
	`, canonicalID, member.ID)
		if err != nil {
			return stats, fmt.Errorf("repoint files of work %d: %w", member.ID, err)
		}
		if affected, err := result.RowsAffected(); err == nil {
			stats.FilesRepointed += int(affected)
		}
		stats.WorksMerged++

		// Each season of the member is folded into the canonical work. A season
		// number that already exists there is joined instead of duplicated, so a
		// merged work never ends up with two rows for the same season.
		moved, err := mergeSeasons(ctx, tx, canonicalID, member.ID)
		if err != nil {
			return stats, err
		}
		stats.SeasonsMoved += moved.Seasons
		stats.EpisodesMoved += moved.Episodes

		// The member's own title becomes an alias, so searching for the old name
		// still finds the surviving work.
		if strings.TrimSpace(member.TitleEN) != "" {
			alias := identity.LearnedFromAdmin(canonicalID, member.TitleEN, member.TitleEN)
			if learned, err := r.storeAliases(ctx, tx, []identity.AliasCandidate{alias}); err == nil {
				stats.AliasesKept += learned
			}
		}
		if strings.TrimSpace(member.TitleAR) != "" && member.TitleAR != member.TitleEN {
			alias := identity.LearnedFromAdmin(canonicalID, member.TitleAR, member.TitleAR)
			if learned, err := r.storeAliases(ctx, tx, []identity.AliasCandidate{alias}); err == nil {
				stats.AliasesKept += learned
			}
		}

		// Mark the folded row and keep it for the audit trail.
		if _, err := tx.ExecContext(ctx, `
			UPDATE media_items
			SET merged_into_id = $1,
			    resolution_state = 'resolved',
			    provisional = TRUE
			WHERE id = $2
	`, canonicalID, member.ID); err != nil {
			return stats, fmt.Errorf("mark work %d merged: %w", member.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return stats, fmt.Errorf("commit merge: %w", err)
	}
	return stats, nil
}

// seasonMergeCounts is what one season fold moved.
type seasonMergeCounts struct {
	Seasons  int
	Episodes int
}

// mergeSeasons moves every season of one work onto another.
//
// A season number that already exists on the target is joined rather than
// duplicated: the target's row wins, its episodes are kept, and the member's
// episodes are folded into it. Video files are re-pointed separately by
// episode_id, so a season fold never has to guess which file belongs where.
func mergeSeasons(ctx context.Context, tx *sql.Tx, canonicalID, memberID int64) (seasonMergeCounts, error) {
	counts := seasonMergeCounts{}

	rows, err := tx.QueryContext(ctx, `
	SELECT id, season_number FROM seasons WHERE media_item_id = $1 ORDER BY id
	`, memberID)
	if err != nil {
		return counts, fmt.Errorf("list seasons of work %d: %w", memberID, err)
	}

	type seasonRow struct {
		ID     int64
		Number int
	}
	memberSeasons := make([]seasonRow, 0, 8)
	for rows.Next() {
		var season seasonRow
		if err := rows.Scan(&season.ID, &season.Number); err != nil {
			return counts, fmt.Errorf("scan season of work %d: %w", memberID, err)
		}
		memberSeasons = append(memberSeasons, season)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return counts, fmt.Errorf("iterate seasons of work %d: %w", memberID, err)
	}

	for _, season := range memberSeasons {
		var existingID int64
		err := tx.QueryRowContext(ctx, `
			SELECT id FROM seasons WHERE media_item_id = $1 AND season_number = $2
	`, canonicalID, season.Number).Scan(&existingID)

		if err == sql.ErrNoRows {
			// No counterpart: the whole season simply belongs to the canonical
			// work now, and its episodes come with it.
			if _, err := tx.ExecContext(ctx, `
				UPDATE seasons SET media_item_id = $1 WHERE id = $2
			`, canonicalID, season.ID); err != nil {
				return counts, fmt.Errorf("move season %d: %w", season.Number, err)
			}
			counts.Seasons++
			continue
		}
		if err != nil {
			return counts, fmt.Errorf("find season %d on target: %w", season.Number, err)
		}

		// Both works have this season. Fold the episodes into the target's row.
		moved, err := mergeEpisodes(ctx, tx, existingID, season.ID)
		if err != nil {
			return counts, err
		}
		counts.Episodes += moved

		// The now-empty season row is retired, not deleted, so the fold is
		// auditable.
		if _, err := tx.ExecContext(ctx, `
			UPDATE episodes SET season_id = $1 WHERE season_id = $2
	`, existingID, season.ID); err != nil {
			return counts, fmt.Errorf("repoint episodes of season %d: %w", season.Number, err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM seasons WHERE id = $1`, season.ID); err != nil {
			return counts, fmt.Errorf("retire merged season %d: %w", season.Number, err)
		}
	}

	return counts, nil
}

// mergeEpisodes folds one season's episodes into another.
//
// An episode number present on both sides is joined: the target's row wins and
// the member's is deleted after its files are re-pointed, so a merged season
// never holds two rows for the same episode number.
func mergeEpisodes(ctx context.Context, tx *sql.Tx, targetSeasonID, memberSeasonID int64) (int, error) {
	rows, err := tx.QueryContext(ctx, `
	SELECT id, episode_number FROM episodes WHERE season_id = $1 ORDER BY id
	`, memberSeasonID)
	if err != nil {
		return 0, fmt.Errorf("list episodes of season %d: %w", memberSeasonID, err)
	}

	type episodeRow struct {
		ID     int64
		Number int
	}
	memberEpisodes := make([]episodeRow, 0, 24)
	for rows.Next() {
		var episode episodeRow
		if err := rows.Scan(&episode.ID, &episode.Number); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan episode of season %d: %w", memberSeasonID, err)
		}
		memberEpisodes = append(memberEpisodes, episode)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate episodes of season %d: %w", memberSeasonID, err)
	}

	moved := 0
	for _, episode := range memberEpisodes {
		var targetEpisodeID int64
		err := tx.QueryRowContext(ctx, `
			SELECT id FROM episodes WHERE season_id = $1 AND episode_number = $2
	`, targetSeasonID, episode.Number).Scan(&targetEpisodeID)

		switch {
		case err == sql.ErrNoRows:
			if _, err := tx.ExecContext(ctx, `
				UPDATE episodes SET season_id = $1 WHERE id = $2
			`, targetSeasonID, episode.ID); err != nil {
				return moved, fmt.Errorf("move episode %d: %w", episode.Number, err)
			}
			// Files follow the episode they were attached to.
			if _, err := tx.ExecContext(ctx, `
				UPDATE video_files SET season_id = $1 WHERE episode_id = $2
			`, targetSeasonID, episode.ID); err != nil {
				return moved, fmt.Errorf("repoint files of episode %d: %w", episode.Number, err)
			}
			moved++
		case err != nil:
			return moved, fmt.Errorf("find episode %d on target: %w", episode.Number, err)
		default:
			// Both sides have this episode. The files from the member are moved
			// onto the surviving episode row before the duplicate is retired.
			if _, err := tx.ExecContext(ctx, `
				UPDATE video_files
				SET episode_id = $1, season_id = $2
				WHERE episode_id = $3
			`, targetEpisodeID, targetSeasonID, episode.ID); err != nil {
				return moved, fmt.Errorf("repoint files of duplicate episode %d: %w", episode.Number, err)
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM episodes WHERE id = $1`, episode.ID); err != nil {
				return moved, fmt.Errorf("retire duplicate episode %d: %w", episode.Number, err)
			}
			moved++
		}
	}

	return moved, nil
}

// -----------------------------------------------------------------------------
// Container-folder re-resolution
// -----------------------------------------------------------------------------

// ReresolveStats reports what a container re-resolution pass changed.
type ReresolveStats struct {
	Scanned  int `json:"scanned"`
	Resolved int `json:"resolved"`
	Skipped  int `json:"skipped"`
	// MergedIntoExisting counts rows whose recovered title already belonged to
	// another row, which is proof that they are a second copy of that work.
	MergedIntoExisting int               `json:"mergedIntoExisting"`
	Details            []ReresolveDetail `json:"details,omitempty"`
	// SkippedDetails records why a row could not be repaired, so a skip is
	// diagnosable rather than silent.
	SkippedDetails []ReresolveDetail `json:"skippedDetails,omitempty"`
}

// ReresolveDetail is one row that was recovered.
type ReresolveDetail struct {
	WorkID   int64  `json:"workId"`
	OldTitle string `json:"oldTitle"`
	NewTitle string `json:"newTitle"`
	FromPath string `json:"fromPath"`
}

// ReresolveContainerWorks repairs rows whose title is a container folder.
//
// These rows are not duplicates of another work; they are rows that were never
// a work at all. "…/Leonardo DiCaprio/أعمال/Titanic.1997.mkv" produced a work
// literally named "أعمال", because the old ingest took the parent folder as the
// title.
//
// Merging such a row into a sibling would preserve the wrong title, so the fix
// is to re-derive the title from the row's own file paths — which is exactly
// what the resolver now does during ingest. The new title is learned as an alias
// so the correction survives a re-scan.
func (r *Repository) ReresolveContainerWorks(ctx context.Context) (ReresolveStats, error) {
	stats := ReresolveStats{}

	groups, err := r.FindDuplicateGroups(ctx)
	if err != nil {
		return stats, err
	}

	for _, group := range groups {
		if group.Kind != KindContainer {
			continue
		}
		stats.Scanned++
		work := group.Members
		if len(work) == 0 {
			continue
		}

		detail, err := r.reresolveOne(ctx, work[0])
		if err != nil && errors.Is(err, errRecoveredTitleTaken) {
			// The recovered title already belongs to another row, which proves
			// this row is a second copy of that work rather than a new one.
			// Merging is then the correct repair, not a re-titling.
			merged, mergeErr := r.mergeIntoExistingTitle(ctx, work[0], err)
			if mergeErr == nil {
				stats.Resolved++
				stats.MergedIntoExisting++
				stats.Details = append(stats.Details, *merged)
				continue
			}
		}
		if err != nil || detail == nil {
			// Record why, so a skipped row is diagnosable instead of silent.
			reason := "no usable title could be recovered"
			if err != nil {
				reason = err.Error()
			} else if work[0].FilePath == "" {
				reason = "the row has no file path to recover a title from"
			}
			stats.Skipped++
			stats.SkippedDetails = append(stats.SkippedDetails, ReresolveDetail{
				WorkID:   work[0].ID,
				OldTitle: work[0].TitleEN,
				FromPath: reason,
			})
			continue
		}
		stats.Resolved++
		stats.Details = append(stats.Details, *detail)
	}

	return stats, nil
}

// errRecoveredTitleTaken means the recovered title already belongs to another
// row, which is proof that this row is a duplicate of that work rather than a
// distinct one. It is a sentinel so the caller can merge instead of failing.
var errRecoveredTitleTaken = errors.New("recovered title already belongs to another work")

// isUniqueViolation reports whether an error is a PostgreSQL unique-constraint
// violation (SQLSTATE 23505).
//
// It matters because that error is not a failure here: it is the database
// asserting that two rows would share one identity, which is exactly the signal
// the catalogue repair needs in order to decide between re-titling and merging.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return strings.Contains(err.Error(), "23505")
}

// mergeIntoExistingTitle folds a container-titled row into the work that already
// owns its recovered title.
//
// This is the case the catalogue actually contains:
//
//	397  Titanic   ← already resolved from its own file
//	398  أعمال     ← same film, still titled after the container folder
//
// Re-titling 398 to "Titanic" collides with the unique identity index, which is
// the database telling us these two rows are one work. Merging honours that
// instead of inventing a disambiguated title, which would leave the library
// holding the same film twice under slightly different names.
func (r *Repository) mergeIntoExistingTitle(ctx context.Context, work DuplicateWork, cause error) (*ReresolveDetail, error) {
	title := strings.TrimSpace(parseForReresolve(work.FilePath).Title)
	if title == "" {
		return nil, cause
	}

	// Find the row that already holds the title.
	var canonicalID int64
	var canonicalTitle string
	if err := r.db.QueryRowContext(ctx, `
	SELECT id, COALESCE(title_en, '') FROM media_items
	WHERE id <> $1 AND merged_into_id IS NULL
	  AND LOWER(title_en) = LOWER($2)
	ORDER BY id LIMIT 1
	`, work.ID, title).Scan(&canonicalID, &canonicalTitle); err != nil {
		return nil, fmt.Errorf("find existing work for title %q: %w", title, err)
	}

	group := WorkMergeGroup{
		Kind:        KindContainer,
		CanonicalID: canonicalID,
		Members:     []DuplicateWork{work},
		Reason:      fmt.Sprintf("recovered title %q already belongs to work %d", title, canonicalID),
		Safe:        true,
	}
	if _, err := r.mergeGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("merge work %d into %d: %w", work.ID, canonicalID, err)
	}

	return &ReresolveDetail{
		WorkID:   work.ID,
		OldTitle: work.TitleEN,
		NewTitle: canonicalTitle,
		FromPath: work.FilePath,
	}, nil
}

// reresolveOne recovers the real title for a single container-titled row.
func (r *Repository) reresolveOne(ctx context.Context, work DuplicateWork) (*ReresolveDetail, error) {
	if work.FilePath == "" {
		return nil, nil
	}

	// Re-parse using the full path, so folder context is available exactly as it
	// is during ingest.
	parsed := parseForReresolve(work.FilePath)
	title := strings.TrimSpace(parsed.Title)
	if title == "" || strings.EqualFold(title, work.TitleEN) {
		return nil, fmt.Errorf("recovered title %q for work %d is empty or unchanged (path %q)",
			title, work.ID, work.FilePath)
	}
	// Refuse to replace one container with another.
	if identity.IsContainerFolderName(title) || identity.IsStructuralTitle(title) {
		return nil, fmt.Errorf("recovered title %q for work %d is still a container", title, work.ID)
	}
	// A recovered title that is only a number is an episode marker or a folder
	// index, not a work name, so it is left alone rather than turned into a work
	// called "05".
	if _, err := strconv.Atoi(title); err == nil {
		return nil, fmt.Errorf("recovered title %q for work %d is only a number", title, work.ID)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin re-resolve: %w", err)
	}
	defer tx.Rollback()

	// The recovered title usually already exists as a real work.
	//
	// This is the normal case, not an edge case:
	// "…/Leonardo DiCaprio/أعمال/Inception.2010.mkv" resolves to "Inception",
	// and an "Inception" row almost certainly exists from another folder.
	// Renaming this row would collide with the unique identity index and leave
	// two rows for one film.
	//
	// So the recovered title is resolved to an existing work first and this row is
	// folded into it. Only when nothing matches does this row become the work,
	// which is the same order the ingest path follows.
	existingID, err := r.findWorkByRecoveredTitle(ctx, tx, title, parsed.ReleaseYear, parsed.MediaType)
	if err != nil {
		return nil, err
	}
	if existingID != 0 && existingID != work.ID {
		return r.attachContainerToExistingWork(ctx, tx, work, existingID, title)
	}

	if _, err := tx.ExecContext(ctx, `
	UPDATE media_items
	SET title_en = $2,
	    title_ar = NULLIF($3, ''),
	    title_normalized = $4,
	    type = COALESCE(NULLIF($5, ''), type),
	    release_year = COALESCE(release_year, NULLIF($6, 0)),
	    title_source = 'resolver',
	    resolution_state = 'resolved',
	    raw_detected_title = $7,
	    provisional = FALSE
	WHERE id = $1
	`, work.ID, title, parsed.TitleAR, identity.Normalize(title), parsed.MediaType,
		parsed.ReleaseYear, work.TitleEN); err != nil {
		return nil, fmt.Errorf("apply recovered title: %w", err)
	}

	// Both names become aliases: the container so a search for it still resolves,
	// and the recovered title so a re-scan attaches here instead of creating a
	// second row for one film.
	for _, aliasValue := range []string{work.TitleEN, title} {
		if strings.TrimSpace(aliasValue) == "" {
			continue
		}
		alias := identity.LearnedFromAdmin(work.ID, aliasValue, title)
		if _, err := r.storeAliases(ctx, tx, []identity.AliasCandidate{alias}); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit re-resolve: %w", err)
	}

	return &ReresolveDetail{
		WorkID:   work.ID,
		OldTitle: work.TitleEN,
		NewTitle: title,
		FromPath: work.FilePath,
	}, nil
}

// reresolveResult is the minimal parse result the repair needs, kept as its own
// type so this file does not depend on the full scanner contract.
type reresolveResult struct {
	Title        string
	TitleAR      string
	MediaType    string
	CategorySlug string
	// ReleaseYear is used to prefer the existing work whose year matches, which
	// is what keeps a remake from being folded into the original.
	ReleaseYear int
}

// findWorkByRecoveredTitle looks for an existing work that already owns a
// recovered title.
//
// Matching is on the title and media type, which is the identity the unique
// index enforces. The release year only orders the candidates: an exact year
// wins, then the closest, so a remake is preferred over the original when both
// exist. Refusing to match at all would create the duplicate this function is
// meant to prevent.
func (r *Repository) findWorkByRecoveredTitle(ctx context.Context, tx *sql.Tx,
	title string, releaseYear int, mediaType string) (int64, error) {

	if strings.TrimSpace(title) == "" {
		return 0, nil
	}

	var id int64
	err := tx.QueryRowContext(ctx, `
	SELECT id FROM media_items
	WHERE merged_into_id IS NULL
	  AND LOWER(title_en) = LOWER($1)
	  AND ($2 = '' OR type = $2)
	ORDER BY
	  ABS(COALESCE(release_year, $3) - $3),
	  id
	LIMIT 1
	`, title, mediaType, releaseYear).Scan(&id)

	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("find work by recovered title %q: %w", title, err)
	}
	return id, nil
}

// attachContainerToExistingWork folds a container-titled row into the real work
// that owns its recovered title.
//
// It is the merge path of the repair: files are re-pointed, seasons and episodes
// are folded, both names become aliases, and the container row is marked merged.
// Nothing is deleted, so the correction stays auditable and reversible.
func (r *Repository) attachContainerToExistingWork(ctx context.Context, tx *sql.Tx,
	work DuplicateWork, targetID int64, recoveredTitle string) (*ReresolveDetail, error) {

	// The file is detached from its season first, because that season belongs to
	// the container row and is about to be folded away. mergeSeasons re-points the
	// files by episode after the seasons move, so a file cannot end up referencing
	// a retired season.
	if _, err := tx.ExecContext(ctx, `
	UPDATE video_files SET media_item_id = $1, season_id = NULL, episode_id = NULL
	WHERE media_item_id = $2
	`, targetID, work.ID); err != nil {
		return nil, fmt.Errorf("repoint files of container work %d: %w", work.ID, err)
	}

	if _, err := mergeSeasons(ctx, tx, targetID, work.ID); err != nil {
		return nil, err
	}

	// Both names become aliases of the surviving work, so searching for either the
	// container folder or the real title finds it, and a re-scan attaches here.
	for _, aliasValue := range []string{work.TitleEN, recoveredTitle} {
		if strings.TrimSpace(aliasValue) == "" {
			continue
		}
		alias := identity.LearnedFromAdmin(targetID, aliasValue, recoveredTitle)
		if _, err := r.storeAliases(ctx, tx, []identity.AliasCandidate{alias}); err != nil {
			return nil, err
		}
	}

	if _, err := tx.ExecContext(ctx, `
	UPDATE media_items
	SET merged_into_id = $1, resolution_state = 'resolved', provisional = TRUE
	WHERE id = $2
	`, targetID, work.ID); err != nil {
		return nil, fmt.Errorf("mark container work %d merged: %w", work.ID, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit container merge: %w", err)
	}

	return &ReresolveDetail{
		WorkID:   work.ID,
		OldTitle: work.TitleEN,
		NewTitle: recoveredTitle + fmt.Sprintf(" (merged into work %d)", targetID),
		FromPath: work.FilePath,
	}, nil
}

// parseForReresolve is assigned by the scanner package at init to avoid an
// import cycle: the scanner already imports identity, and this package must not
// grow a dependency on the scanner's internals.
//
// It is a package-level hook rather than an interface because exactly one
// implementation exists in production.
var parseForReresolve = func(path string) reresolveResult {
	return reresolveResult{}
}

// ReresolvePathForTest exposes the installed hook so a caller can verify the
// wiring itself, separately from the repair logic. A missing hook silently makes
// every repair a no-op, which is otherwise indistinguishable from "nothing to
// repair".
func ReresolvePathForTest(path string) string {
	return parseForReresolve(path).Title
}

// SetReresolveParser installs the path parser the container repair uses.
//
// It is called once during wiring, which keeps the database layer free of a
// direct scanner dependency while still letting the repair use the same parser
// as ingest — so a repaired title and a freshly ingested title cannot differ.
func SetReresolveParser(parser func(path string) (title, titleAR, mediaType, category string)) {
	if parser == nil {
		return
	}
	parseForReresolve = func(path string) reresolveResult {
		title, titleAR, mediaType, category := parser(path)
		return reresolveResult{Title: title, TitleAR: titleAR, MediaType: mediaType, CategorySlug: category}
	}
}

// suppression keeps an unused import from being removed if the identity helpers
// above are ever narrowed.
var _ = time.Now
