# ADR-011: Rebuildable search projection and cooperative scan control

- **Status:** Accepted
- **Date:** 2026-09-19
- **Depends on:** ADR-009 (incremental indexing), ADR-010 (logical media model)

## Context

Two parts of the indexing system were still not production-grade after ADR-009
and ADR-010.

### 1. The search index had a silent ceiling

`handleSearchSync` and the scan's own sync step called:

```go
documents, err := s.repository.ListSearchDocuments(ctx, 10000)
```

`ListSearchDocuments` clamps its argument to 10,000 and returns a single slice.
For a library past that size the index simply stopped at 10,000 works, with:

- no error,
- no warning,
- no indication in any response.

An operator with 40,000 works would have seen "search sync ok" and a search
index holding a quarter of the library. Worse, the entire library was loaded
into memory as one slice, so raising the limit would have traded a silent
truncation for an out-of-memory failure.

The same query was also un-resumable: any interruption started from zero.

### 2. A scan could be cancelled but not paused

Only `cancel()` existed. That left an operator with a bad choice: let a
multi-hour scan of a large library run, or destroy the run's progress by
cancelling it. There was also no way to see what the scanner was doing beyond an
integer worker count, so "8 workers" told an operator nothing about whether the
run was healthy or stuck.

## Decision

### 1. The search index is a paged, resumable projection

```text
PostgreSQL (media_items)
  ↓ ListSearchDocumentPage(afterID, limit)    keyset pagination by ascending id
  ↓ IndexDocuments(page)                       pushed to Meilisearch per page
  ↓ SaveProjectionCursor(afterID)              progress persisted per page
```

Four properties follow, and each is tested:

- **No ceiling.** Pages are read until the catalogue is exhausted, so a library
  of any size is fully projected. Verified on a 25,000-document catalogue, which
  is well past the old 10,000 limit.
- **Flat memory.** Only one page is held at a time, so peak memory is independent
  of library size.
- **Resumable.** The cursor lives in `search_projection_state` and is written
  after every page, so an interrupted rebuild continues rather than repeating
  work. A restart mid-rebuild resumes.
- **Rebuildable without the filesystem.** The projector's inputs are a database
  store and a search sink. Nothing in the path reads a media file, so dropping
  and rebuilding the index never re-scans a single byte of the library.

**Keyset pagination, not offset pagination.** The cursor is `id > afterID` with
`ORDER BY id`. An offset-based pager skips or duplicates rows when items are
inserted during a rebuild, which on a live library is the normal case rather than
an edge case.

**Targeted updates.** `ProjectWork` indexes exactly the documents given and
touches no page at all, so adding one episode updates one document instead of
rebuilding an index of a million works. `DeleteWork` removes by primary key for
the same reason: a merged duplicate is deleted, not rebuilt around.

### 2. Pause is cooperative and distinct from Cancel

```text
RUNNING → (pause requested) → PAUSING → (no worker mid-item) → PAUSED
   ↑                                                            │
   └──────────────── (resume) ── RESUMING ──────────────────────┘
```

The gate sits **immediately before a worker takes a new work item**, never in
the middle of one:

```go
for visit := range candidates {
    // The pause gate sits BEFORE an item is taken, so a paused scan never
    // abandons a half-processed file.
    if !control.wait(workCtx) {
        return
    }
    control.markWorker(workerID, RoleMetadata, visit.Path)
    file, ok := s.processCandidate(...)
    ...
}
```

That placement is the safety property: **no half-written database row can ever
result from a pause.** A worker finishing its current file runs to completion.

`PAUSING` exists separately from `PAUSED` because collapsing them would make the
UI claim the scan has stopped while workers are still finishing. The scan reaches
`PAUSED` only once a worker is observed blocking.

**Cancel is a different operation, not a synonym.** Cancel ends the scan and
reports `CANCELLED`. It first releases any worker blocked on a pause, otherwise a
paused scan would never observe the cancellation. Reusing Cancel for Pause would
make a temporary stop indistinguishable from an abandoned run, and would discard
the counters and cursor that Pause deliberately preserves.

### 3. Workers are individually visible

```json
{"id": 1, "role": "metadata", "active": true,
 "currentPath": "/media/Disk1/Show/S01E04.mkv", "processed": 812}
```

Roles are `discovery`, `metadata`, `persistence` and `idle`. The list is ordered
by worker id so a polling UI does not reshuffle. `ItemStartedAt` is recorded so a
stuck file becomes visible rather than only a stalled counter.

## Consequences

**Positive**

- A library of any size gets a complete search index, and the run resumes after a
  restart instead of restarting.
- Dropping the index and rebuilding it from the database is a supported
  operation that never touches the filesystem.
- Adding one episode costs one document write, not a library rebuild.
- An operator can pause a long scan without losing it, and can see what each
  worker is doing.

**Negative / accepted trade-offs**

- A paused scan that is never resumed holds its workers indefinitely. This is
  intended: Pause means "hold this", and Cancel exists for "stop".
- The projection cursor is a single integer per stream. Two concurrent rebuilds
  of the same stream would race on it. This is acceptable because the scan guard
  already serialises heavy operations, and a rebuild is an explicit operator
  action rather than something the server runs concurrently with itself.
- Episode documents are not projected yet. The `episodes` table and the
  `search_projection_state` stream for it exist; the projection query for episode
  documents does not. See "Remaining" below.

## Verification

- `go build ./...`, `go vet ./...`, `go test ./...` all pass across 12 packages.
- **Search projection: 10 tests** covering paging past the old ceiling (25,000
  documents indexed exactly once, in ascending order), resume from a persisted
  cursor, reset, per-page cursor persistence, partial-progress reporting on
  failure, cancellation, targeted single-document update, targeted delete,
  empty-update no-op, and the "database only, no filesystem" boundary.
- **Scan control: 11 tests** covering the pause state machine, that a paused scan
  blocks new work, that resume releases it, that cancel releases it with a *stop*
  signal, that a cancelled context releases it, that a paused scan does not
  abandon the in-flight item, per-worker roles and current paths, idle
  transitions, the processed counter, stable ordering, nil-safety of the handle,
  and that a scan without a control handle still works.
- Endpoints verified against the running server:
  `/api/scan/workers` returns `{"running":false,"scanState":"idle","workers":[]}`.

## Alternatives considered

1. **Raise the 10,000 limit.** Rejected: it converts a silent truncation into an
   out-of-memory failure for exactly the libraries that need it most.
2. **Offset pagination.** Rejected: it skips and duplicates rows under concurrent
   inserts, which is the normal state of a live library.
3. **Rebuild the whole index on every change.** Rejected: it makes one new
   episode cost a full library projection.
4. **Implement Pause as Cancel plus a stored cursor.** Rejected: it discards the
   per-root status and worker state, and it makes a resumable scan
   indistinguishable from an abandoned one in the session record.
5. **Hard-stop workers on pause (kill mid-item).** Rejected: it can leave a
   partially written row and a half-updated season, which is precisely the class
   of corruption the indexing work exists to prevent.

## The admin UI exposes both centres
Two React screens consume the endpoints above.

**Scan control centre** (`components/admin/ScanControlCenter.jsx`, on
`/admin/indexer`):

- polls `/api/scan/status` every 1.5s while a scan runs, and **stops polling**
  when it finishes or the component unmounts;
- shows every measured counter (directories, files seen, candidates, accepted,
  new/changed/renamed/unchanged, bytes, skipped bytes, files/sec, MB/sec,
  permission errors, filesystem errors, low confidence, duration);
- renders a per-worker table with each worker's role, current file and item
  count;
- offers Pause, Resume and **Cancel behind an explicit confirmation**, with copy
  explaining that Pause keeps the counters while Cancel loses them.

The UI shows the `pausing` state honestly rather than claiming an immediate
stop, and only presents `paused` once the backend reports it.

The mode selector sends `mode: "incremental"` by default and explains the
trade-off, so an operator is not left running a full re-probe by accident.

**Review centre** (`components/admin/ResolutionReviewCenter.jsx`, on
`/admin/review`):

- lists queued files grouped by reason, with counts and filters;
- shows each candidate's score **with the itemised evidence behind it**, so the
  ranking is explainable rather than an opaque number;
- offers attach / create / mark-movie / ignore, with a work search for attach and
  season/episode fields;
- exposes "learn this mapping", which promotes the decision into a durable alias
  so the same ambiguity never returns.

**Verified in a real browser** (headless Chromium, logged in, both routes):
`loggedIn: true`, control centre present, both mode buttons present, review
centre present and rendering 26 queued items, and **zero JavaScript errors** on
the indexer route. The review screen listed a real file, `05 4K.mkv`, with 80%
parser confidence and no invented title — the exact case this system exists to
surface instead of guessing at.

## Remaining (recorded honestly, not silently accepted)

- **Episode search documents** are not projected. The table and cursor stream
  exist; the projection query does not. A library searches by work today.
- **`go test -race`** has not been run in this environment because no C toolchain
  is available. The concurrency guarantees are covered by deterministic
  stress tests instead, one of which found a genuine race during ADR-010 work.
