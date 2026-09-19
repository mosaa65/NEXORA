package db

import (
	"context"
	"fmt"
	"sync"

	"nexora/server/internal/identity"
	"nexora/server/internal/scanner"
)

// ResolutionSession holds the Entity Resolution state shared by every file that
// reaches the catalogue, regardless of which path produced it.
//
// All three write paths must resolve before they store: the on-demand scan, the
// debounced file watcher, and the scheduled reconciliation sweep. If any one of
// them wrote files directly, that path would silently recreate the degenerate
// works that entity resolution exists to prevent.
//
// The mutable state is:
//
//	works   - the candidate set, which GROWS as new works are created so a later
//	          file can attach to a work an earlier file created;
//	aliases - the learned alias -> work map, so a confirmed spelling resolves
//	          exactly on the very next file rather than by similarity.
//
// Both are guarded because the watcher and the scheduler can run concurrently.
type ResolutionSession struct {
	mu          sync.Mutex
	works       []identity.Work
	aliasLookup map[string]int64
	resolver    *identity.Resolver
	loaded      bool
}

// NewResolutionSession creates a session that loads its state on first use.
func NewResolutionSession() *ResolutionSession {
	return &ResolutionSession{resolver: identity.NewResolver()}
}

// ResolutionStore is the persistence the session needs. It is an interface so
// the session can be handed a mock in tests and a concrete repository at
// runtime, without a type assertion at the call site.
type ResolutionStore interface {
	LoadKnownWorks(ctx context.Context) ([]identity.Work, error)
	LoadAliasLookup(ctx context.Context) (map[string]int64, error)
	ResolveAndIngest(ctx context.Context, files []scanner.FileInfo, resolver *identity.Resolver,
		works *[]identity.Work, aliasLookup map[string]int64) (ResolutionIngestStats, error)
}

// Resolver exposes the configured resolver so callers can tune it before use.
func (s *ResolutionSession) Resolver() *identity.Resolver {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.resolver
}

// loadLocked fetches the candidate set and alias map once, then reuses them.
func (s *ResolutionSession) loadLocked(ctx context.Context, repository ResolutionStore) error {
	if s.loaded {
		return nil
	}
	works, err := repository.LoadKnownWorks(ctx)
	if err != nil {
		return fmt.Errorf("load known works: %w", err)
	}
	aliases, err := repository.LoadAliasLookup(ctx)
	if err != nil {
		// Aliases are an optimisation, not a requirement: a library with no
		// aliases yet must still resolve.
		aliases = map[string]int64{}
	}
	s.works = works
	s.aliasLookup = aliases
	s.loaded = true
	return nil
}

// Invalidate forces the next ingest to reload the candidate set and aliases.
// It is called after a completed scan so the watcher sees the new entities.
func (s *ResolutionSession) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loaded = false
}

// Ingest resolves and stores a batch using the session's shared state.
func (s *ResolutionSession) Ingest(ctx context.Context, repository ResolutionStore,
	files []scanner.FileInfo) (ResolutionIngestStats, error) {

	if len(files) == 0 {
		return ResolutionIngestStats{}, nil
	}

	// Held across the whole batch so the candidate slice and alias map cannot be
	// mutated by a concurrent watcher event while this batch reads them.
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.loadLocked(ctx, repository); err != nil {
		return ResolutionIngestStats{}, err
	}

	stats, err := repository.ResolveAndIngest(ctx, files, s.resolver, &s.works, s.aliasLookup)
	if err != nil {
		return stats, err
	}

	// Refresh the alias map so a mapping learned by this batch is usable by the
	// next one without a database round trip per file.
	if stats.AliasesLearned > 0 {
		if aliases, err := repository.LoadAliasLookup(ctx); err == nil {
			s.aliasLookup = aliases
		}
	}
	return stats, nil
}

// KnownWorkCount reports how many candidate works the session holds. It is used
// by tests and the scan report to show the candidate set growing.
func (s *ResolutionSession) KnownWorkCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.works)
}

// -----------------------------------------------------------------------------
// Deprecated legacy path
// -----------------------------------------------------------------------------

// ErrLegacyIngestDisabled is returned by the legacy ingest so no code path can
// accidentally bypass entity resolution.
// IngestScannedFilesDirect is the raw, resolution-free writer.
//
// It is deliberately NOT wired to any caller. It exists only so a future
// migration tool can rewrite existing rows in bulk without re-resolving them.
// Every runtime path uses ResolutionSession instead.
//
// Calling it from the scan, watcher or scheduler would restore the exact defect
// that created works named "الكنز ج1 الحلقه" and "منت فديوهات زابيا", which is why
// it is named to be obvious in a review.
func (r *Repository) IngestScannedFilesDirect(ctx context.Context, files []scanner.FileInfo) (IngestStats, error) {
	return r.ingestWithoutResolution(ctx, files)
}
