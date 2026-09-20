package search

import (
	"context"
	"errors"
	"testing"
)

// episodeFakeStore serves episode documents for the projector tests.
type episodeFakeStore struct {
	documents []EpisodeDocument
	cursors   map[string]int64
	pageCalls int
	saves     int
	resets    int
	failAfter int
}

func newEpisodeFakeStore(count int) *episodeFakeStore {
	docs := make([]EpisodeDocument, 0, count)
	for i := 1; i <= count; i++ {
		docs = append(docs, EpisodeDocument{
			ID:             int64(i),
			WorkID:         int64((i / 10) + 1),
			SeasonNumber:   (i % 5) + 1,
			EpisodeNumber:  i,
			EpisodeTitleEN: "Episode " + itoa(i),
		})
	}
	return &episodeFakeStore{documents: docs, cursors: map[string]int64{}}
}

func (s *episodeFakeStore) ListEpisodeDocumentPage(ctx context.Context, afterID int64, limit int) ([]EpisodeDocument, error) {
	s.pageCalls++
	if s.failAfter > 0 && s.pageCalls > s.failAfter {
		return nil, errors.New("simulated page failure")
	}
	if limit <= 0 {
		limit = 100
	}
	page := make([]EpisodeDocument, 0, limit)
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

func (s *episodeFakeStore) LiveEpisodeIDs(ctx context.Context) ([]int64, error) {
	ids := make([]int64, 0, len(s.documents))
	for _, doc := range s.documents {
		ids = append(ids, doc.ID)
	}
	return ids, nil
}

func (s *episodeFakeStore) LoadProjectionCursors(ctx context.Context) (map[string]int64, error) {
	out := make(map[string]int64, len(s.cursors))
	for kind, id := range s.cursors {
		out[kind] = id
	}
	return out, nil
}

func (s *episodeFakeStore) SaveProjectionCursor(ctx context.Context, kind string, lastID int64, documentCount int64) error {
	s.cursors[kind] = lastID
	s.saves++
	return nil
}

func (s *episodeFakeStore) ResetProjectionCursor(ctx context.Context, kind string) error {
	s.cursors[kind] = 0
	s.resets++
	return nil
}

// episodeFakeSink records what was written to the episode index.
type episodeFakeSink struct {
	batches     [][]EpisodeDocument
	deleted     []int64
	configured  int
	existingIDs []int64
}

func (s *episodeFakeSink) IndexEpisodeDocuments(ctx context.Context, documents []EpisodeDocument) (SyncResult, error) {
	batch := append([]EpisodeDocument(nil), documents...)
	s.batches = append(s.batches, batch)
	return SyncResult{Indexed: len(documents)}, nil
}

func (s *episodeFakeSink) EpisodeDocumentIDs(ctx context.Context) ([]int64, error) {
	return s.existingIDs, nil
}

func (s *episodeFakeSink) DeleteEpisodeDocuments(ctx context.Context, ids []int64) (SyncResult, error) {
	s.deleted = append(s.deleted, ids...)
	return SyncResult{Indexed: len(ids)}, nil
}

func (s *episodeFakeSink) ConfigureEpisodeIndex(ctx context.Context) error {
	s.configured++
	return nil
}

// TestEpisodeProjectionPagesThroughTheWholeIndex proves the episode index has no
// size ceiling, which is the property the work index already had.
func TestEpisodeProjectionPagesThroughTheWholeIndex(t *testing.T) {
	const total = 12000
	store := newEpisodeFakeStore(total)
	sink := &episodeFakeSink{}
	projector := NewEpisodeProjector(sink, store, 1000, nil)

	result, err := projector.Rebuild(context.Background(), false)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	if result.Documents != total {
		t.Errorf("projected %d documents, want %d", result.Documents, total)
	}
	if result.Pages < 12 {
		t.Errorf("used %d pages for %d documents; paging is not working", result.Pages, total)
	}

	// Every episode must appear exactly once, in ascending id order.
	seen := make(map[int64]int, total)
	previous := int64(0)
	for _, batch := range sink.batches {
		for _, doc := range batch {
			seen[doc.ID]++
			if doc.ID <= previous {
				t.Fatalf("episodes are not strictly ascending: %d after %d", doc.ID, previous)
			}
			previous = doc.ID
		}
	}
	if len(seen) != total {
		t.Errorf("indexed %d distinct episodes, want %d", len(seen), total)
	}
	for id, count := range seen {
		if count != 1 {
			t.Fatalf("episode %d was indexed %d times", id, count)
		}
	}
}

// TestEpisodeProjectionResumesFromPersistedCursor proves an interrupted episode
// rebuild continues rather than restarting.
func TestEpisodeProjectionResumesFromPersistedCursor(t *testing.T) {
	store := newEpisodeFakeStore(5000)
	store.cursors["episodes"] = 4000
	sink := &episodeFakeSink{}
	projector := NewEpisodeProjector(sink, store, 1000, nil)

	result, err := projector.Rebuild(context.Background(), false)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	if !result.Resumed || result.FromID != 4000 {
		t.Errorf("resume reported resumed=%v from=%d, want true/4000", result.Resumed, result.FromID)
	}
	if result.Documents != 1000 {
		t.Errorf("projected %d episodes, want the remaining 1000", result.Documents)
	}
	for _, batch := range sink.batches {
		for _, doc := range batch {
			if doc.ID <= 4000 {
				t.Fatalf("episode %d was re-indexed although it was already projected", doc.ID)
			}
		}
	}
}

// TestEpisodeProjectionResetStartsFromZero covers the drop-and-rebuild operation.
func TestEpisodeProjectionResetStartsFromZero(t *testing.T) {
	store := newEpisodeFakeStore(300)
	store.cursors["episodes"] = 250
	sink := &episodeFakeSink{}
	projector := NewEpisodeProjector(sink, store, 100, nil)

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
		t.Errorf("projected %d episodes after reset, want all 300", result.Documents)
	}
}

// TestEpisodeProjectionReportsPartialProgressOnFailure ensures a failed rebuild
// still states how far it got.
func TestEpisodeProjectionReportsPartialProgressOnFailure(t *testing.T) {
	store := newEpisodeFakeStore(5000)
	store.failAfter = 2
	sink := &episodeFakeSink{}
	projector := NewEpisodeProjector(sink, store, 1000, nil)

	result, err := projector.Rebuild(context.Background(), false)
	if err == nil {
		t.Fatal("expected the rebuild to fail")
	}
	if result.Documents == 0 || result.LastID == 0 {
		t.Errorf("a partial failure must report progress: %+v", result)
	}
}

// TestEpisodePruneRemovesOrphans is the episode-side counterpart of the work
// orphan test: an episode row that no longer exists must not stay searchable.
func TestEpisodePruneRemovesOrphans(t *testing.T) {
	store := newEpisodeFakeStore(10)
	sink := &episodeFakeSink{
		// The index holds three rows whose catalogue entries are gone.
		existingIDs: []int64{1, 2, 3, 900, 901, 902},
	}
	projector := NewEpisodeProjector(sink, store, 100, nil)

	result, err := projector.Prune(context.Background())
	if err != nil {
		t.Fatalf("prune: %v", err)
	}

	if result.Indexed != 6 {
		t.Errorf("indexed = %d, want 6", result.Indexed)
	}
	if result.Live != 10 {
		t.Errorf("live = %d, want 10", result.Live)
	}
	if result.Orphans != 3 {
		t.Errorf("orphans = %d, want 3", result.Orphans)
	}
	if result.Deleted != 3 {
		t.Errorf("deleted = %d, want 3", result.Deleted)
	}
	if len(sink.deleted) != 3 {
		t.Errorf("sink received %d deletions, want 3", len(sink.deleted))
	}
}

// TestEpisodeProjectWorkUpdatesOnlyGivenEpisodes is the targeted-update
// guarantee: enriching one episode writes one document, not the whole index.
func TestEpisodeProjectWorkUpdatesOnlyGivenEpisodes(t *testing.T) {
	store := newEpisodeFakeStore(1000)
	sink := &episodeFakeSink{}
	projector := NewEpisodeProjector(sink, store, 1000, nil)

	_, err := projector.ProjectEpisodes(context.Background(), []EpisodeDocument{
		{ID: 42, EpisodeTitleEN: "Ozymandias"},
	})
	if err != nil {
		t.Fatalf("project episodes: %v", err)
	}

	if len(sink.batches) != 1 || len(sink.batches[0]) != 1 || sink.batches[0][0].ID != 42 {
		t.Fatalf("wrote %+v, want only episode 42", sink.batches)
	}
	// Crucially: the paged reader was never touched.
	if store.pageCalls != 0 {
		t.Errorf("a targeted update paged the catalogue %d times", store.pageCalls)
	}
}

// TestEpisodeProjectWorkWithNoDocumentsIsANoOp guards against an empty update
// reaching the search engine.
func TestEpisodeProjectWorkWithNoDocumentsIsANoOp(t *testing.T) {
	store := newEpisodeFakeStore(10)
	sink := &episodeFakeSink{}
	projector := NewEpisodeProjector(sink, store, 10, nil)

	if _, err := projector.ProjectEpisodes(context.Background(), nil); err != nil {
		t.Fatalf("project episodes: %v", err)
	}
	if len(sink.batches) != 0 {
		t.Errorf("an empty update reached the search engine: %+v", sink.batches)
	}
}

// TestEpisodeProjectionIsRebuildableFromDatabaseOnly documents the boundary: the
// episode projector's inputs are a database store and a search sink. No
// filesystem is involved, so the index can be dropped and rebuilt without
// re-reading a single media file.
func TestEpisodeProjectionIsRebuildableFromDatabaseOnly(t *testing.T) {
	store := newEpisodeFakeStore(50)
	sink := &episodeFakeSink{}
	projector := NewEpisodeProjector(sink, store, 10, nil)

	if _, err := projector.Rebuild(context.Background(), true); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if len(sink.batches) == 0 {
		t.Fatal("nothing was projected")
	}
}

// itoa avoids importing strconv for a test-only counter.
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var buffer [20]byte
	index := len(buffer)
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
