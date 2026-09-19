# ADR-009: Incremental, fault-tolerant media indexing pipeline

- **Status:** Accepted
- **Date:** 2026-08-30
- **Supersedes:** the implicit "scan every file into one slice, then insert" model documented in `PROJECT_ANALYSIS.md` §10, §23, §28.

## Context

The catalogue is the foundation of NEXORA: search, hubs, playback and quality
reports all read from it. The previous indexing path had properties that do not
survive a real library (hundreds of thousands of files, tens of terabytes, several
disks including NAS and removable storage):

1. **One unreadable directory aborted the entire scan.** `walkRoot` returned any
   `walkErr` to a shared `setErr`, which cancelled the shared context, so a single
   permission-denied folder left the library half-indexed.
2. **One failed root cancelled every other root.** All roots shared one context
   and one `sync.Once` error latch, so an unplugged disk stopped the scans of the
   disks that were present.
3. **Every run re-parsed and re-wrote every file.** There was no persisted file
   identity and no change detection, so a library of 100k files performed 100k
   parse operations and 100k transactions on every run.
4. **One database transaction per file.** `ingestScannedFile` did `BeginTx` /
   `Commit` per file, i.e. ~1M round-trips for 1M files.
5. **FFprobe ran synchronously for every file on every scan**, inside the
   request's callback, which saturated CPU and I/O.
6. **Deletions never reached the catalogue, and renames became delete+create.**
   The watcher logged removes and did nothing; there was no identity to detect a
   move, so a rename risked a duplicate row and lost watch progress.
7. **A large download was ingested dozens of times**, because every `Write` event
   triggered an ingest and there was no debounce or stability check.
8. **Artwork performed `os.ReadDir` per video file**, so a season folder with
   100k episodes performed 100k directory reads to find one poster.
9. **Category detection used `strings.Contains(fullPath, keyword)`**, so
   `/NotMovies/` classified as `movies`.
10. **Any filename ending in digits became an episode**, so `Toy Story 2` was
    indexed as episode 2.
11. **`Scan()` accumulated `[]FileInfo` in memory**, which cannot hold millions of
    files.
12. **No scan state was persisted**, so a crash at 72% was invisible and the
    catalogue was assumed consistent.

## Decision

Replace the ad-hoc walk with an explicit, bounded, staged pipeline, and make the
filesystem-to-catalogue transition reconcile-based rather than scan-based.

```text
Filesystem
  ↓  DISCOVERY (per-root goroutines, bounded channel, ignore rules, symlink policy)
  ↓  METADATA WORKERS (bounded pool: stat, fingerprint, parse, classify)
  ↓  IDENTITY / CHANGE DETECTION (path, then file ID, then size+mtime)
  ↓  RECONCILIATION (new / changed / renamed / missing / unavailable)
  ↓  PERSISTENCE (batched upserts, single transaction per batch)
  ↓  WATCHER (fast path, debounced, stability-checked) + periodic reconciliation (source of truth)
```

Concretely:

- **Root isolation.** Each media root has its own status and its own walker. A
  root failure marks only that root `UNAVAILABLE`; siblings continue.
- **Per-directory error isolation.** A directory that cannot be read is recorded
  and classified, and traversal continues with its siblings. Only an unreadable
  *root* is treated as a root-level failure.
- **Streaming, bounded stages.** Discovery feeds a fixed-size channel, a bounded
  worker pool parses, and `emit` runs on exactly one goroutine. Nothing is
  accumulated per file.
- **Change detection without hashing.** Identity is `path` → `file_id` (Windows
  file index / POSIX inode) → `size + mod_time`. File *content* is never read
  during a scan; hashing remains an explicit, separate operation.
- **Renames are moves, not delete+create.** When a file identity is seen at a new
  path and its old path was not visited in the same pass, the existing record's
  path is rewritten and its identity preserved.
- **States, not deletions.** Records move to `MISSING` only when the owning root
  was readable, and to `UNAVAILABLE` when it was not. Deletion exists only as an
  explicit cleanup with a minimum-age gate, an offline-root refusal and a
  per-run cap.
- **Persistence is batched.** One transaction per batch (256 files) with
  multi-row upserts; per-file failures are counted instead of aborting the batch.
- **The watcher is the fast path, reconciliation is the source of truth.**
  fsnotify can overflow, miss events on network shares, and cannot see what
  happened while the server was down, so a periodic incremental sweep reconciles
  drift. A remove/rename event is reported and deferred, never acted on as a
  deletion.
- **Debounce plus stability.** A file is indexed only after its size and mtime
  have been unchanged for a stability window, which is what lets one download
  produce one record.
- **Evidence-based parsing.** Parsing collects folder, filename, season, episode,
  part and category evidence, then scores it. Weak evidence reduces confidence
  instead of inventing metadata, and a part/CD marker is never an episode number.
- **Directory-level artwork cache.** Artwork is resolved once per directory and
  reused, so per-file cost is O(1).
- **Scan sessions are persisted.** Every run is recorded, so a restart can detect
  a scan that never finished (`interrupted = TRUE`) rather than assume the
  catalogue is consistent.

## Consequences

**Positive**

- A library can be indexed with one unreadable folder, one offline disk or one
  corrupted file, without losing the rest of the work.
- An incremental re-scan of an unchanged library costs directory traversal plus
  one comparison per file, instead of a parse and a write per file.
- Deletion of catalogue data requires an explicit, capped, gated operation.
- The pipeline is bounded, so it is safe to run for days.
- The scan report is structured data, so the UI can show real progress and real
  problems.

**Negative / accepted trade-offs**

- Change detection trusts size + mtime when no filesystem identity exists. A
  same-size, same-mtime, different-content edit on such a filesystem is not
  detected. This is accepted deliberately: the alternative is hashing terabytes
  on every scan.
- Rename inference requires either a stable file ID or an unambiguous
  size+mtime match. On a filesystem with neither, a move appears as one missing
  record and one new record. The resolver refuses to guess when more than one
  candidate matches.
- The schema gains columns and tables. They are all additive with defaults, so
  existing data stays valid and no forced full re-scan is required.
- The old `GET /api/scan` JSON listing endpoint was removed and `POST /api/ingest`
  became an alias of `POST /api/index`. The admin UI used neither.

## Alternatives considered

1. **Keep the walk, add targeted patches (error isolation only).** Rejected: it
   fixes crash tolerance but leaves the O(files) re-parse and O(files)
   transactions that make a 100k+ library impractical.
2. **Content hashing for identity.** Rejected: reading terabytes on every scan is
   the exact cost this system must avoid. Hashing remains opt-in and targeted.
3. **A separate indexing service or message queue.** Rejected for now: it adds a
   deployment unit and failure mode without solving any problem that bounded
   in-process concurrency does not already solve. The package boundaries
   (`discovery`, `identity`, `pipeline`, `reconciler`, `watcher`, `progress`,
   `errors`) make that extraction possible later without a rewrite.
4. **Trust fsnotify as the source of truth.** Rejected: it demonstrably loses
   events and cannot span downtime. It is a latency optimisation, not a guarantee.
5. **Delete records on a remove event.** Rejected: a remove can be the first half
   of a rename, a disconnecting share or a replaced folder. This is the single
   most likely way to lose library data.

## Verification

- `go test ./...` and `go vet ./...` pass.
- A real data race was found by the benchmark and fixed: `identityIndex.seen` was
  written by every metadata worker. A stress test now drives 8 workers against
  200 candidate records and asserts each record is claimed at most once.
- Benchmarks on synthetic trees: 1,000 files ≈ 124 ms, 10,000 files ≈ 1.9 s
  (~5,250 files/s, linear scaling); incremental scan of 5,000 unchanged files
  ≈ 199 ms with 869 allocations, i.e. no per-file allocation.
- Fault tolerance is covered by explicit tests for an unreadable directory, an
  unavailable root beside a healthy root, and a context cancellation.
