package search

import (
	"context"
	"fmt"
	"log/slog"
)

// EpisodeDocument is the search projection of one episode.
//
// It lives in its OWN index, not alongside works. That separation is what makes
// searching inside a work fast and unambiguous: a work query never has to
// filter episodes out, and an episode hit never has to be distinguished from a
// work by inspecting its shape.
//
// The document carries the parent work's titles as well as the episode's own, so
// one hit is enough to render "Breaking Bad — S05E14 Ozymandias" without a
// second lookup.
type EpisodeDocument struct {
	// ID is the episode row id from the episodes table, so a hit can be opened
	// directly.
	ID int64 `json:"id"`
	// WorkID is the parent work, for filtering to one show.
	WorkID int64 `json:"work_id"`
	// SeasonNumber and EpisodeNumber are the identity that matters to a viewer.
	SeasonNumber  int `json:"season_number"`
	EpisodeNumber int `json:"episode_number"`

	// Episode titles. Empty until enrichment runs.
	EpisodeTitleEN string `json:"episode_title_en,omitempty"`
	EpisodeTitleAR string `json:"episode_title_ar,omitempty"`
	// EpisodeTitleNormalized is the folded form used for search only.
	EpisodeTitleNormalized string `json:"episode_title_normalized,omitempty"`

	// Parent work titles, denormalised so a hit renders without a join.
	WorkTitleEN         string `json:"work_title_en,omitempty"`
	WorkTitleAR         string `json:"work_title_ar,omitempty"`
	WorkTitleNormalized string `json:"work_title_normalized,omitempty"`

	OverviewEN string `json:"overview_en,omitempty"`
	OverviewAR string `json:"overview_ar,omitempty"`
	StillPath  string `json:"still_path,omitempty"`
	AirDate    string `json:"air_date,omitempty"`
	Runtime    int    `json:"runtime,omitempty"`

	// CategorySlug lets an episode list be scoped to a category.
	CategorySlug string `json:"category_slug,omitempty"`

	// HasLocalFile reports whether a physical file exists for this episode.
	// False is the "coming soon" state the UI renders for a provider episode
	// that the library has not acquired yet.
	HasLocalFile bool `json:"has_local_file"`
	// FileCount is how many release files back this episode. More than one is
	// normal (different resolutions of the same episode).
	FileCount int `json:"file_count"`

	// Resolution and FileSize come from the local file when one exists, so the
	// card can show the library's own facts beside the provider's metadata.
	Resolution string `json:"resolution,omitempty"`
	FileSize   int64  `json:"file_size,omitempty"`
	Duration   int    `json:"duration,omitempty"`

	// EnrichedFrom records where the descriptive fields came from. It is the
	// data-lineage marker: "provider" means a TMDB snapshot already stored
	// locally, "local" means the library's own filename parsing.
	EnrichedFrom string `json:"enriched_from,omitempty"`
}

// EpisodeProjectionStore is the persistence the episode projector reads.
type EpisodeProjectionStore interface {
	ListEpisodeDocumentPage(ctx context.Context, afterID int64, limit int) ([]EpisodeDocument, error)
	LiveEpisodeIDs(ctx context.Context) ([]int64, error)
	LoadProjectionCursors(ctx context.Context) (map[string]int64, error)
	SaveProjectionCursor(ctx context.Context, kind string, lastID int64, documentCount int64) error
	ResetProjectionCursor(ctx context.Context, kind string) error
}

// EpisodeIndexName is the Meilisearch index that holds episodes.
//
// It is a SEPARATE index from the work index, and that is the point. Sharing one
// index would collide on the primary key (work 42 and episode 42 are different
// rows) and would force every work query to filter episodes out.
const EpisodeIndexName = "media_episodes"

// EpisodeSink is the search-engine capability the episode projector needs.
//
// It is a separate interface from ProjectionSink because the document type
// differs: an episode and a work share no key space, so they must not share a
// writer that assumes one shape.
type EpisodeSink interface {
	IndexEpisodeDocuments(ctx context.Context, documents []EpisodeDocument) (SyncResult, error)
	EpisodeDocumentIDs(ctx context.Context) ([]int64, error)
	DeleteEpisodeDocuments(ctx context.Context, ids []int64) (SyncResult, error)
	ConfigureEpisodeIndex(ctx context.Context) error
}

// EpisodeProjector builds and maintains the episode search index.
//
// It mirrors Projector exactly: keyset paging with a persisted cursor, so the
// index is complete at any size, flat in memory, resumable after a restart, and
// rebuildable from the database without reading a single media file.
type EpisodeProjector struct {
	client   EpisodeSink
	episodes EpisodeProjectionStore
	pageSize int
	logger   *slog.Logger
}

// NewEpisodeProjector wires an episode projector.
func NewEpisodeProjector(client EpisodeSink, episodes EpisodeProjectionStore,
	pageSize int, logger *slog.Logger) *EpisodeProjector {

	if pageSize <= 0 {
		pageSize = DefaultProjectionPageSize
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &EpisodeProjector{client: client, episodes: episodes, pageSize: pageSize, logger: logger}
}

// Rebuild projects every episode into the episode index.
//
// `reset` starts from the beginning, which is the "drop and rebuild" operation.
// Without it the run resumes from the persisted cursor, so an interrupted
// rebuild continues rather than repeating work.
func (p *EpisodeProjector) Rebuild(ctx context.Context, reset bool) (ProjectionResult, error) {
	result := ProjectionResult{}
	const kind = "episodes"

	if reset {
		if err := p.episodes.ResetProjectionCursor(ctx, kind); err != nil {
			return result, fmt.Errorf("reset episode projection cursor: %w", err)
		}
	} else {
		cursors, err := p.episodes.LoadProjectionCursors(ctx)
		if err == nil {
			result.FromID = cursors[kind]
			result.Resumed = result.FromID > 0
		}
	}

	afterID := result.FromID
	projected := 0

	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		page, err := p.episodes.ListEpisodeDocumentPage(ctx, afterID, p.pageSize)
		if err != nil {
			return result, fmt.Errorf("read episode projection page after id %d: %w", afterID, err)
		}
		if len(page) == 0 {
			break
		}

		if _, err := p.client.IndexEpisodeDocuments(ctx, page); err != nil {
			// Report the work already completed. `projected` counts pages that
			// succeeded, so the caller can say how far the index got instead of only
			// that it failed.
			result.Documents = projected
			result.LastID = afterID
			return result, fmt.Errorf("index episode projection page: %w", err)
		}

		projected += len(page)
		result.Pages++
		afterID = page[len(page)-1].ID
		result.LastID = afterID
		result.Documents = projected
		if err := p.episodes.SaveProjectionCursor(ctx, kind, afterID, int64(projected)); err != nil {
			p.logger.Warn("could not persist episode projection cursor", slog.Any("error", err))
		}

		if len(page) < p.pageSize {
			break
		}
	}

	result.Documents = projected
	return result, nil
}

// Prune removes episode documents whose episode row no longer exists.
//
// Same reasoning as the work index: indexing is additive, so a deleted episode
// would otherwise stay searchable forever.
func (p *EpisodeProjector) Prune(ctx context.Context) (PruneResult, error) {
	result := PruneResult{}

	indexed, err := p.client.EpisodeDocumentIDs(ctx)
	if err != nil {
		return result, fmt.Errorf("read episode index ids: %w", err)
	}
	result.Indexed = len(indexed)
	if len(indexed) == 0 {
		return result, nil
	}

	live, err := p.episodes.LiveEpisodeIDs(ctx)
	if err != nil {
		return result, fmt.Errorf("read live episode ids: %w", err)
	}
	result.Live = len(live)

	liveSet := make(map[int64]struct{}, len(live))
	for _, id := range live {
		liveSet[id] = struct{}{}
	}

	orphans := make([]int64, 0)
	for _, id := range indexed {
		if _, exists := liveSet[id]; !exists {
			orphans = append(orphans, id)
		}
	}
	result.Orphans = len(orphans)

	for start := 0; start < len(orphans); start += pruneBatchSize {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		end := start + pruneBatchSize
		if end > len(orphans) {
			end = len(orphans)
		}
		if _, err := p.client.DeleteEpisodeDocuments(ctx, orphans[start:end]); err != nil {
			result.Failures++
			continue
		}
		result.Deleted += end - start
	}

	return result, nil
}

// ProjectEpisodes indexes exactly the given episodes.
//
// This is the targeted update path: enriching one episode writes one document,
// rather than rebuilding an index of twelve thousand episodes.
func (p *EpisodeProjector) ProjectEpisodes(ctx context.Context, episodes []EpisodeDocument) (SyncResult, error) {
	if len(episodes) == 0 {
		return SyncResult{}, nil
	}
	return p.client.IndexEpisodeDocuments(ctx, episodes)
}
