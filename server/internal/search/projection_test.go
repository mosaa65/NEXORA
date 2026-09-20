package search

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// fakeStore is an in-memory ProjectionStore that reproduces the paging contract
// of the real repository: strictly ascending ids, a cursor, and persistent
// bookkeeping.
type fakeStore struct {
	documents   []MediaDocument
	cursors     map[string]int64
	pageCalls   int
	cursorSaves int
	resets      int
	// failAfterPage simulates a database or search failure mid-rebuild.
	failAfterPage int
}

func newFakeStore(count int) *fakeStore {
	docs := make([]MediaDocument, 0, count)
	for i := 1; i <= count; i++ {
		docs = append(docs, MediaDocument{
			ID:      int64(i),
			TitleEN: fmt.Sprintf("Work %d", i),
			Type:    "movie",
		})
	}
	return &fakeStore{documents: docs, cursors: map[string]int64{}}
}

func (s *fakeStore) ListSearchDocumentPage(ctx context.Context, afterID int64, limit int) ([]MediaDocument, error) {
	s.pageCalls++
	if s.failAfterPage > 0 && s.pageCalls > s.failAfterPage {
		return nil, errors.New("simulated page failure")
	}
	if limit <= 0 {
		limit = 100
	}
	page := make([]MediaDocument, 0, limit)
	for _, doc := range s.documents {
		if doc.ID <= afterID {
			continue
		}
		page = append(page, doc)
		if len(page) == limit {
			break
		}
	}
	return page, nil
}

func (s *fakeStore) LoadProjectionCursors(ctx context.Context) (map[string]int64, error) {
	out := make(map[string]int64, len(s.cursors))
	for kind, id := range s.cursors {
		out[kind] = id
	}
	return out, nil
}

func (s *fakeStore) SaveProjectionCursor(ctx context.Context, kind string, lastID int64, documentCount int64) error {
	s.cursors[kind] = lastID
	s.cursorSaves++
	return nil
}

func (s *fakeStore) ResetProjectionCursor(ctx context.Context, kind string) error {
	s.cursors[kind] = 0
	s.resets++
	return nil
}

// LiveWorkIDs reports the works that still exist, which is the set a prune pass
// keeps. The fake exposes it so the orphan behaviour is testable without a
// database.
func (s *fakeStore) LiveWorkIDs(ctx context.Context) ([]int64, error) {
	ids := make([]int64, 0, len(s.documents))
	for _, doc := range s.documents {
		ids = append(ids, doc.ID)
	}
	return ids, nil
}

// fakeSink records what was indexed so the test can assert exact document sets.
type fakeSink struct {
	indexedBatches [][]MediaDocument
	deleted        []int64
	failAfterBatch int
	// cancelAfterFirst cancels the run once the first batch is indexed, so the
	// cancellation test exercises the mid-rebuild path.
	cancelAfterFirst context.CancelFunc
}

func (s *fakeSink) IndexDocuments(ctx context.Context, documents []MediaDocument) (SyncResult, error) {
	if s.failAfterBatch > 0 && len(s.indexedBatches) >= s.failAfterBatch {
		return SyncResult{}, errors.New("simulated index failure")
	}
	batch := append([]MediaDocument(nil), documents...)
	s.indexedBatches = append(s.indexedBatches, batch)
	if s.cancelAfterFirst != nil && len(s.indexedBatches) == 1 {
		s.cancelAfterFirst()
	}
	return SyncResult{Indexed: len(documents), TaskUID: "task-1"}, nil
}

func (s *fakeSink) DeleteDocuments(ctx context.Context, ids []int64) (SyncResult, error) {
	s.deleted = append(s.deleted, ids...)
	return SyncResult{Indexed: len(ids)}, nil
}

// DocumentIDs reports the ids currently indexed, so the orphan-prune test can
// simulate an index that accumulated documents whose rows are gone.
func (s *fakeSink) DocumentIDs(ctx context.Context) ([]int64, error) {
	seen := make(map[int64]struct{}, len(s.indexedBatches))
	ids := make([]int64, 0, len(s.indexedBatches))
	for _, batch := range s.indexedBatches {
		for _, doc := range batch {
			if _, exists := seen[doc.ID]; exists {
				continue
			}
			seen[doc.ID] = struct{}{}
			ids = append(ids, doc.ID)
		}
	}
	// Deleted ids are removed, so a prune test sees the index shrink.
	deleted := make(map[int64]struct{}, len(s.deleted))
	for _, id := range s.deleted {
		deleted[id] = struct{}{}
	}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, gone := deleted[id]; gone {
			continue
		}
		out = append(out, id)
	}
	return out, nil
}

// TestProjectionPagesThroughTheWholeCatalogue is the test for the removed
// ceiling.
//
// The previous sync read at most 10,000 documents and stopped silently. This
// projects a catalogue larger than that and asserts every document was indexed.
func TestProjectionPagesThroughTheWholeCatalogue(t *testing.T) {
	const total = 25000
	store := newFakeStore(total)
	sink := &fakeSink{}
	projector := NewProjector(sink, store, 1000, nil)

	result, err := projector.Rebuild(context.Background(), false)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	if result.Documents != total {
		t.Errorf("projected %d documents, want %d", result.Documents, total)
	}
	if result.Pages < 25 {
		t.Errorf("used %d pages for %d documents; paging is not working", result.Pages, total)
	}
	if result.LastID != total {
		t.Errorf("last id = %d, want %d", result.LastID, total)
	}

	// Every document must appear exactly once, in ascending order.
	seen := make(map[int64]int, total)
	previous := int64(0)
	for _, batch := range sink.indexedBatches {
		for _, doc := range batch {
			seen[doc.ID]++
			if doc.ID <= previous {
				t.Fatalf("documents are not strictly ascending: %d after %d", doc.ID, previous)
			}
			previous = doc.ID
		}
	}
	if len(seen) != total {
		t.Errorf("indexed %d distinct documents, want %d", len(seen), total)
	}
	for id, count := range seen {
		if count != 1 {
			t.Fatalf("document %d was indexed %d times", id, count)
		}
	}
}

// TestProjectionResumesFromPersistedCursor proves an interrupted rebuild
// continues instead of starting over.
//
// This is what makes a crash mid-rebuild cheap: the work already done is kept.
func TestProjectionResumesFromPersistedCursor(t *testing.T) {
	store := newFakeStore(5000)
	store.cursors["media_items"] = 3000
	sink := &fakeSink{}
	projector := NewProjector(sink, store, 1000, nil)

	result, err := projector.Rebuild(context.Background(), false)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	if !result.Resumed {
		t.Error("the run did not report resuming from the stored cursor")
	}
	if result.FromID != 3000 {
		t.Errorf("resumed from %d, want 3000", result.FromID)
	}
	if result.Documents != 2000 {
		t.Errorf("projected %d documents, want the remaining 2000", result.Documents)
	}

	// Nothing below the cursor may be re-indexed.
	for _, batch := range sink.indexedBatches {
		for _, doc := range batch {
			if doc.ID <= 3000 {
				t.Fatalf("document %d was re-indexed although it was already projected", doc.ID)
			}
		}
	}
}

// TestProjectionResetStartsFromZero covers the rebuild-from-the-database
// operation: drop the index, project everything again.
func TestProjectionResetStartsFromZero(t *testing.T) {
	store := newFakeStore(300)
	store.cursors["media_items"] = 250
	sink := &fakeSink{}
	projector := NewProjector(sink, store, 100, nil)

	result, err := projector.Rebuild(context.Background(), true)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	if store.resets != 1 {
		t.Errorf("cursor resets = %d, want 1", store.resets)
	}
	if result.Resumed {
		t.Error("a reset run must not report resuming")
	}
	if result.Documents != 300 {
		t.Errorf("projected %d documents after reset, want all 300", result.Documents)
	}
}

// TestProjectionPersistsCursorPerPage verifies progress survives a crash at any
// point, which is the whole reason the cursor is written per page.
func TestProjectionPersistsCursorPerPage(t *testing.T) {
	store := newFakeStore(3500)
	sink := &fakeSink{}
	projector := NewProjector(sink, store, 1000, nil)

	if _, err := projector.Rebuild(context.Background(), false); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	// Four pages of 1000/1000/1000/500 -> four cursor saves.
	if store.cursorSaves < 4 {
		t.Errorf("cursor saved %d times, want at least one save per page", store.cursorSaves)
	}
	if got := store.cursors["media_items"]; got != 3500 {
		t.Errorf("final cursor = %d, want 3500", got)
	}
}

// TestProjectionReportsPartialProgressOnFailure ensures a failed rebuild reports
// how far it got, rather than only that it failed.
func TestProjectionReportsPartialProgressOnFailure(t *testing.T) {
	store := newFakeStore(5000)
	sink := &fakeSink{failAfterBatch: 2}
	projector := NewProjector(sink, store, 1000, nil)

	result, err := projector.Rebuild(context.Background(), false)
	if err == nil {
		t.Fatal("expected the rebuild to fail")
	}
	if result.Documents == 0 {
		t.Error("a partial failure must report the documents already projected")
	}
	if result.LastID == 0 {
		t.Error("a partial failure must report the last projected id")
	}
	if result.Documents >= 5000 {
		t.Errorf("reported %d documents although the run failed early", result.Documents)
	}
}

// TestProjectionHonoursCancellation verifies a long rebuild stops promptly.
func TestProjectionHonoursCancellation(t *testing.T) {
	store := newFakeStore(100000)
	sink := &fakeSink{}
	projector := NewProjector(sink, store, 10, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Cancel after the first page is indexed.
	sink.cancelAfterFirst = cancel

	_, err := projector.Rebuild(ctx, false)
	if err == nil {
		t.Fatal("expected the rebuild to report cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

// TestProjectWorkUpdatesOnlyTheGivenDocuments is the "add one episode, update
// one document" guarantee: a targeted update must not reindex the library.
func TestProjectWorkUpdatesOnlyTheGivenDocuments(t *testing.T) {
	store := newFakeStore(1000)
	sink := &fakeSink{}
	projector := NewProjector(sink, store, 1000, nil)

	_, err := projector.ProjectWork(context.Background(), []MediaDocument{
		{ID: 42, TitleEN: "Breaking Bad", Type: "series"},
	})
	if err != nil {
		t.Fatalf("project work: %v", err)
	}

	if len(sink.indexedBatches) != 1 {
		t.Fatalf("indexed %d batches, want exactly 1", len(sink.indexedBatches))
	}
	if len(sink.indexedBatches[0]) != 1 || sink.indexedBatches[0][0].ID != 42 {
		t.Fatalf("indexed %+v, want only work 42", sink.indexedBatches[0])
	}
	// Crucially: the paged reader was never touched.
	if store.pageCalls != 0 {
		t.Errorf("a targeted update paged the catalogue %d times", store.pageCalls)
	}
}

// TestProjectWorkWithNoDocumentsIsANoOp guards against an empty update being
// sent to the search engine.
func TestProjectWorkWithNoDocumentsIsANoOp(t *testing.T) {
	store := newFakeStore(10)
	sink := &fakeSink{}
	projector := NewProjector(sink, store, 10, nil)

	if _, err := projector.ProjectWork(context.Background(), nil); err != nil {
		t.Fatalf("project work: %v", err)
	}
	if len(sink.indexedBatches) != 0 {
		t.Errorf("an empty update reached the search engine: %+v", sink.indexedBatches)
	}
}

// TestDeleteWorkTargetsGivenIDs verifies removals are targeted, not a rebuild.
func TestDeleteWorkTargetsGivenIDs(t *testing.T) {
	store := newFakeStore(10)
	sink := &fakeSink{}
	projector := NewProjector(sink, store, 10, nil)

	if err := projector.DeleteWork(context.Background(), []int64{7, 8}); err != nil {
		t.Fatalf("delete work: %v", err)
	}
	if len(sink.deleted) != 2 || sink.deleted[0] != 7 || sink.deleted[1] != 8 {
		t.Errorf("deleted %v, want [7 8]", sink.deleted)
	}
	if store.pageCalls != 0 {
		t.Error("a targeted delete paged the catalogue")
	}
}

// TestProjectionIsRebuildableFromDatabaseOnly documents the boundary: the
// projector's inputs are a database store and a search sink, and nothing else.
// No filesystem is involved, so the index can be dropped and rebuilt without
// re-reading a single media file.
func TestProjectionIsRebuildableFromDatabaseOnly(t *testing.T) {
	store := newFakeStore(50)
	sink := &fakeSink{}
	projector := NewProjector(sink, store, 10, nil)

	// This compiles and runs with no filesystem access anywhere: the store
	// supplies documents, the sink receives them.
	if _, err := projector.Rebuild(context.Background(), true); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if len(sink.indexedBatches) == 0 {
		t.Fatal("nothing was projected")
	}
}
