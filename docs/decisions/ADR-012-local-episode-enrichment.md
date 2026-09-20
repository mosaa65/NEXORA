# ADR-012: Local episode enrichment and a separate episode search index

- **Status:** Accepted
- **Date:** 2026-09-19
- **Depends on:** ADR-010 (logical media model), ADR-011 (search projection)

## Context

Two problems surfaced when the search layer was inspected against the real
library.

### 1. The search index was append-only, so it accumulated orphans

`Projector.Rebuild` indexed and updated documents but never deleted one. Every
work that had been deleted or merged therefore stayed searchable forever.

Measured on the live database:

```
Meilisearch  → 382 documents
PostgreSQL   → 280 works
difference   → 102 orphaned documents
```

A user searching for a work could find a deleted one, click it, and get a 404.
The index is derived data by definition (ADR-011), and derived data that follows
insertions but not deletions is not really derived.

### 2. Episodes did not exist as data

The ingest path created an `episodes` row only when a video file carried an
episode number. Measured:

```
local episodes        → 1
local seasons         → 68
provider episodes     → 12,130 available in stored snapshots
```

So the library knew about its seasons but not its episodes. Consequences:

- searching for an episode title was impossible — there were no episode
  documents to find
- a season could not show how many episodes the provider says it has
- there was no way to express "the provider lists this episode but the library
  has not acquired it yet"

### 3. Enriching from the provider was rejected as the wrong tool

The original plan was to call TMDB to enrich episodes. That was rejected by the
library owner for reasons that hold up technically:

> Every episode has a name or it does not, and all of it is guesswork. It burns
> requests against the provider and most of them would fail anyway.

Inspection confirmed there was no need to call anyone: the database already
holds **350 stored season payloads containing 12,130 complete episodes**, each
with a title, overview, still image, air date and runtime.

## Decision

### 1. The index follows deletions, not only insertions

- `Client.DocumentIDs` and `Client.EpisodeDocumentIDs` enumerate index keys,
  **paginated**, because assuming one response holds the whole index silently
  truncates on a large library and leaves the tail forever unpruned.
- `Projector.Prune` and `EpisodeProjector.Prune` diff the index against the live
  row ids and delete the difference in batches.
- `POST /api/search/prune` exposes pruning explicitly.
- A full rebuild (`sync?reset=true`) prunes automatically, because the operator
  already asked for a from-scratch rebuild. A routine incremental sync never
  prunes, so no ordinary operation can delete from the index unasked.
- `DELETE /api/media/{id}` removes the document immediately. The cascade is
  best-effort and reports a warning rather than failing the delete, because the
  database deletion has already succeeded and index cleanup has a recovery path.

### 2. Search fields are ordered by significance

Meilisearch weights an earlier `searchableAttributes` entry above a later one.
The previous order placed `title_ar` before `title_en`, put the plot at the same
level as the title, and omitted the normalized title.

```
title_en → title_ar → title_normalized → alternate_titles
        → genres → category_en → category_ar → plot_en → plot_ar
```

`rankingRules` is now written out explicitly. They match Meilisearch's defaults
today, and stating them means an engine upgrade cannot silently reorder results.

### 3. Arabic search works because the index carries a folded title

`title_normalized` applies the normalization already proven in
`internal/identity`: Arabic letter variants fold to one form, diacritics and
tatweel are removed, and Arabic-Indic and Persian numerals become ASCII. So
`اسامة` matches `أسامة` and `الحلقة ١` matches `الحلقة 1`.

The original spelling is never replaced — `title_ar` and `title_en` keep it for
display, and `title_normalized` exists only for matching.

`alternate_titles` carries every learned alias, so a library that has learned
"ون بيس" for "One Piece" is findable by either name.

### 4. Episodes live in their own index

`media_episodes` is a separate Meilisearch index, not a second document type in
the work index. Sharing one index would:

- collide on the primary key (work 42 and episode 42 are different rows)
- force every work query to exclude episodes
- force every episode hit to be identified by inspecting its shape

The separation is what makes "episodes of this work" and "season 3 only" a
**filter** rather than a scan, which is the reason the index exists.

Ranking within the episode index puts the episode's own title first and the
parent work's name second, so `Ozymandias` finds the episode and `Breaking Bad`
finds everything from the show.

### 5. Enrichment is local and deterministic

`Repository.EnrichFromLocalSnapshots` materialises seasons and episodes from
`season_metadata_snapshots` with **zero external requests**.

The match is deterministic, never guessed: a provider episode is identified by
`(media_item_id, season_number, episode_number)`, which is exactly how the
provider itself numbers episodes. No fuzzy title matching is involved, and no
mapping table has to be invented.

Rules that follow from that:

- A provider episode with no local file **still gets a row**. That row is what
  lets the UI render "coming soon" instead of showing a season with fewer
  episodes than it actually has.
- Provider values win over local values for descriptive fields, because TMDB is
  authoritative for a title while a local row may be empty or derived from a
  filename. `enriched_from` records which was used, so the quality report can
  tell them apart.
- An existing non-empty local title is **never** overwritten. An operator's own
  edit outranks provider metadata, consistent with ADR-010's provenance order.
- Season number `0` is kept, because that is how TMDB models "Specials"; dropping
  it would hide specials the library owns.
- An episode without a positive episode number is skipped rather than defaulted.
  A row that cannot be identified by number cannot be matched to a file, and
  inventing a number would corrupt the season's ordering.

`Repository.LinkOrphanEpisodes` then points each video file at its episode row by
`(season_id, episode_number)`, so a file is never duplicated across two rows.

## Consequences

**Positive**

- An index rebuild now removes what is gone, so search results stop leading to
  404s.
- Arabic search works on real spelling variation instead of exact bytes.
- Episodes are searchable, which was previously impossible.
- A season can show its true episode count and mark the ones not yet acquired.
- Enrichment of 6,066 episodes ran in **13 seconds with zero provider requests**,
  and cannot be rate-limited or time out.
- Re-running enrichment is idempotent: it fills gaps and creates what is missing,
  and changes nothing else.

**Negative / accepted trade-offs**

- The episode index duplicates the parent work's titles in every episode
  document. That is deliberate denormalisation: it lets one hit render
  "Breaking Bad — S05E14 Ozymandias" without a second lookup, at the cost of a
  larger index.
- Pruning reads every id in the index. On a very large library that is a real
  read cost, which is why it runs on request or as the last step of a rebuild
  rather than during every sync.
- Enrichment trusts the stored snapshot. If a snapshot is stale, the episode
  reflects that staleness until the snapshot is refreshed. This is the intended
  trade: the local snapshot is the source of truth for enrichment, and the
  network path is only used to refresh snapshots.
- Merged duplicate works are excluded from `LiveWorkIDs`, so their index entries
  are pruned. That is correct for a merged duplicate, but it means a work must
  not be marked merged while it should still be searchable.

## Verification

Measured on the live library, not on fixtures:

| Check | Result |
|---|---|
| Local enrichment (5 works) | 372 episodes created in 1.15s |
| **Local enrichment (full run)** | **6,066 episodes in 12.85s, 0 provider requests** |
| Episode index build | 12,130 documents projected across 13 pages |
| Episode index stored | 6,066 unique documents |
| Enrichment field coverage | title 100%, overview 85%, still 86%, air date 98%, runtime 89% |
| Search by episode title | `Ozymandias` → S05E14, via=provider |
| Search in Arabic | `الحلقة` → 1000 results |
| `has_local_file` | false for provider-only episodes, true where a file exists |

`go build ./...`, `go vet ./...` and `go test ./...` all pass across 12 packages.

## Known remaining item

The live database contains **12 or more duplicate works** created by the
pre-ADR-010 ingest, for example `Mission Impossible` exists three times.
`identity.DetectDuplicateWorks` already proposes the merges, but executing a
merge rewrites catalogue rows and is therefore left for an explicit owner
decision rather than run as part of this work.

## Alternatives considered

1. **Offset pagination for orphan detection.** Rejected: an offset pager skips
   and duplicates rows when documents are inserted during the scan, which on a
   live library is the normal case.
2. **Timestamp-based pruning** (delete anything not written by this run).
   Rejected: it deletes legitimate documents whenever a run is partial.
3. **One index for works and episodes.** Rejected: primary-key collision, and
   every work query would need to exclude episodes.
4. **Calling TMDB per episode.** Rejected by the owner, and unnecessary: the
   data is already stored locally and the match is deterministic. It would also
   burn provider quota and fail on rate limits for no benefit.
5. **Enriching only episodes that have a local file.** Rejected: it would hide
   episodes the provider knows about, making a season look shorter than it is and
   removing the "coming soon" state entirely.
