package search

import (
	"context"
	"fmt"
	"log/slog"
)

// ProjectionStore is the persistence the projector needs.
//
// It is an interface so the projector has no database dependency and can be
// tested with a fake, and so the paging contract is explicit: a caller supplies
// the next page after a cursor plus the cursor bookkeeping.
type ProjectionStore interface {
	ListSearchDocumentPage(ctx context.Context, afterID int64, limit int) ([]MediaDocument, error)
	LoadProjectionCursors(ctx context.Context) (map[string]int64, error)
	SaveProjectionCursor(ctx context.Context, kind string, lastID int64, documentCount int64) error
	ResetProjectionCursor(ctx context.Context, kind string) error
}

// ProjectionResult reports what a rebuild did.
type ProjectionResult struct {
	Documents int    `json:"documents"`
	Pages     int    `json:"pages"`
	FromID    int64  `json:"fromId"`
	LastID    int64  `json:"lastId"`
	Resumed   bool   `json:"resumed"`
	TaskUID   string `json:"taskUid,omitempty"`
}

// DefaultProjectionPageSize is the page size used when none is chosen.
//
// It is exported from this package because the projection policy (how much is
// read per page) belongs with the projector, not with the caller.
const DefaultProjectionPageSize = 1000

// Projector builds and maintains the search index as a PROJECTION of PostgreSQL.
//
// The search index is derived, rebuildable data. Two properties make that true
// in practice rather than only in intent:
//
//   - It pages through the database by ascending id with a persisted cursor, so
//     a library of any size can be projected without holding it in memory, and a
//     rebuild interrupted by a restart resumes instead of starting over.
//   - It can update a single document, so adding one episode does not rebuild
//     the whole index.
//
// Nothing here ever reads the filesystem: the index is rebuilt from the database.
type Projector struct {
	client   ProjectionSink
	store    ProjectionStore
	pageSize int
	logger   *slog.Logger
}

// ProjectionSink is the search engine capability the projector needs.
//
// It is separate from *Client so the API can pass its own interface value
// directly, and so a test can substitute a recording sink without a network.
type ProjectionSink interface {
	IndexDocuments(ctx context.Context, documents []MediaDocument) (SyncResult, error)
	DeleteDocuments(ctx context.Context, ids []int64) (SyncResult, error)
}

// NewProjector wires a projector to the search engine and the database.
func NewProjector(client ProjectionSink, store ProjectionStore, pageSize int, logger *slog.Logger) *Projector {
	if pageSize <= 0 {
		pageSize = 1000
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Projector{client: client, store: store, pageSize: pageSize, logger: logger}
}

// Rebuild projects the whole catalogue into the search index.
//
// When `reset` is true the cursor starts from zero, which is the "drop the index
// and rebuild it from the database without touching the filesystem" operation.
// When false, the run resumes from the persisted cursor, so an interrupted
// rebuild continues instead of repeating work.
//
// The cursor is saved after every page, so progress survives a crash at any
// point.
func (p *Projector) Rebuild(ctx context.Context, reset bool) (ProjectionResult, error) {
	result := ProjectionResult{}

	if reset {
		if err := p.store.ResetProjectionCursor(ctx, "media_items"); err != nil {
			return result, fmt.Errorf("reset projection cursor: %w", err)
		}
	} else {
		cursors, err := p.store.LoadProjectionCursors(ctx)
		if err == nil {
			result.FromID = cursors["media_items"]
			result.Resumed = result.FromID > 0
		}
	}

	afterID := result.FromID
	documentsProjected := 0

	for {
		// Cooperative cancellation: a rebuild of a large library is a long job and
		// must stop promptly when the operator cancels a scan.
		if err := ctx.Err(); err != nil {
			return result, err
		}

		page, err := p.store.ListSearchDocumentPage(ctx, afterID, p.pageSize)
		if err != nil {
			return result, fmt.Errorf("read projection page after id %d: %w", afterID, err)
		}
		if len(page) == 0 {
			break
		}

		syncResult, err := p.client.IndexDocuments(ctx, page)
		if err != nil {
			// Return the progress already made so the caller's report is truthful
			// about how far the index got before failing.
			result.Documents = documentsProjected
			result.LastID = afterID
			return result, fmt.Errorf("index projection page: %w", err)
		}
		if syncResult.TaskUID != "" {
			result.TaskUID = syncResult.TaskUID
		}

		documentsProjected += len(page)
		result.Pages++
		afterID = page[len(page)-1].ID
		result.LastID = afterID

		if err := p.store.SaveProjectionCursor(ctx, "media_items", afterID, int64(documentsProjected)); err != nil {
			p.logger.Warn("could not persist projection cursor", slog.Any("error", err))
		}

		if len(page) < p.pageSize {
			break
		}
	}

	result.Documents = documentsProjected
	return result, nil
}

// ProjectWork updates the search document for exactly one work.
//
// This is the "add one episode, update one document" path. It is what keeps a
// single file addition from rebuilding an index of a million works.
func (p *Projector) ProjectWork(ctx context.Context, documents []MediaDocument) (SyncResult, error) {
	if len(documents) == 0 {
		return SyncResult{}, nil
	}
	return p.client.IndexDocuments(ctx, documents)
}

// DeleteWork removes works from the index.
//
// Meilisearch deletes by primary key, so removing a merged duplicate is a
// targeted operation rather than a full rebuild.
func (p *Projector) DeleteWork(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := p.client.DeleteDocuments(ctx, ids)
	if err != nil {
		return fmt.Errorf("delete search documents: %w", err)
	}
	return nil
}
