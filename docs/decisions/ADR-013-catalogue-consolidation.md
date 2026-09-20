# ADR-013: Catalogue consolidation — duplicates, container titles and unlinked files

- **Status:** Accepted
- **Date:** 2026-09-19
- **Depends on:** ADR-010 (logical media model), ADR-012 (local episode enrichment)

## Context

Inspecting the live catalogue after the episode work exposed three defects that
the pre-ADR-010 ingest left behind. All three are catalogue-shape problems, not
indexing problems, so none of them could be fixed by the scanner.

### 1. Duplicate works

Measured on the live database, before repair:

```
Mission Impossible   → rows 305, 331, 340   (three works for one film)
Fast & Furious       → 3 rows
Attack on Titan      → 2 rows, one typed anime and one typed series
Moana, Dune Part Two, John Wick Chapter 4, …
16 same-title groups covering 19 rows
```

Each duplicate is a row that became its own work because the old ingest derived
a work per filename.

### 2. Container folders became works

```
work 397  "أعمال"   file …/Leonardo DiCaprio/أعمال/Titanic.1997.mkv
work 398  "أعمال"   file …/Leonardo DiCaprio/أعمال/Inception.2010.mkv
work 403  "أعمال"   file …/Robert Downey Jr/أعمال/Iron Man.2008.mkv
work 420  "Media"   file D:\Media\الانطلاقه نت 5.mp4
```

"أعمال" means "works" and is a browse grouping. The old ingest took the parent
folder as the title, so three different films in one actor's folder became three
rows named "works".

**These are not duplicates of each other.** Merging them would have preserved the
wrong title and produced one row claiming to be three films. They are rows that
were never a work at all.

### 3. Video files with no episode link

```
total files       → 5,049
linked to episode → 4,536
unlinked          → 513
  of which 77 carry a season and episode number
```

The 77 are files the library owns whose episode the provider never listed — an
unlisted special, a fan release, a numbering the provider does not use. Leaving
them unlinked means the catalogue cannot see episodes it actually holds.

## Decision

### 1. Consolidation is proposed, never assumed

`Repository.FindDuplicateGroups` classifies each group and marks whether it is
safe to apply:

| Kind | Meaning | Applied automatically? |
|---|---|---|
| `same_title` | identical normalized title and media type | yes |
| `alias_linked` | already connected by a learned alias | yes |
| `cross_language` | same work in two languages | no — proposed only |
| `container_folder` | the title is not a work name | no — needs re-resolution |

The distinction is the point. Collapsing these into one "duplicate" concept would
give a provable title match and a bilingual guess the same trust.

### 2. A merge folds, it never deletes

`MergeDuplicateGroups` runs only on safe groups, one transaction per group:

- files are re-pointed to the canonical work,
- seasons are folded, and a season number present on both sides is **joined**
  rather than duplicated,
- episodes are folded the same way, with the files moved onto the surviving row
  before the duplicate is retired,
- every folded row's title becomes an alias of the surviving work, so searching
  for the old name still finds it,
- the folded row is marked `merged_into_id` and **kept**.

Nothing is deleted, so the decision is auditable and reversible.

The canonical row is chosen by preference: non-provisional before provisional, then
more files, then lower id. The last criterion makes a repeat run deterministic —
running the tool twice cannot pick differently.

### 3. A container row is re-resolved, not merged

`ReresolveContainerWorks` re-parses the row's own file path with the same parser
ingest uses, recovering "Titanic" from a file named `Titanic.1997.mkv`.

The recovered title is then resolved to an **existing work** before anything is
created. This is not an edge case — it is the normal case: a film in an actor's
folder almost certainly already has a row from another folder. Renaming would
collide with the unique identity index and leave two rows for one film, so the row
is folded into the existing work instead, and only becomes the work when nothing
matches.

The container name and the recovered title both become aliases, so a search for
either resolves and a re-scan attaches here rather than creating a second row.

### 4. Two guards refuse to invent

A container row is left alone when:

- **it has no file**, so there is nothing to recover a title from;
- **the recovered title is only a number**, because that is an episode marker or
  a folder index, not a film name.

Both leave two rows unrepaired on the live library, on purpose. Inventing a title
for them would trade a visible gap for an invisible error.

### 5. Unlinked files without a provider episode are materialised locally

`LinkOrphanEpisodes` does two things in one transaction:

1. link every file whose episode row already exists;
2. create the missing episode row from the file's own season and episode number,
   marked `provider = 'local'`, then link it.

The episode is created with **no descriptive fields**. The provider metadata stays
absent rather than invented — the file becomes reachable, and the quality report
can tell a local-only episode from an enriched one.

`DISTINCT ON (season_id, episode_number)` keeps one episode per number even when
several release files describe it, so a season with a 1080p and a 4K release does
not produce duplicate episode entities.

## Consequences

**Positive**

- Duplicate works are gone, so search returns a film once instead of three times.
- Container-titled rows name the film they actually contain.
- Files the library owns are reachable from its own catalogue.
- Every repair is reversible: folded rows are kept, and the original title
  survives as an alias.
- Re-running any repair is idempotent.

**Negative / accepted trade-offs**

- `same_title` matches on title and type only. Two genuinely different works that
  happen to share a title and a type would be folded. The unique identity index
  already treats that pair as one work, so the merge makes the catalogue
  consistent with its own constraint rather than introducing a new assumption.
- The release year is used to *order* candidates, not to reject them, because
  regional releases of one film are routinely a year apart.
- A local-only episode has no title, overview or still until the provider lists
  it. This is the intended trade: a nameless reachable episode beats an
  unreachable named one.
- Two container rows remain unrepaired (no file, and a filename that is only a
  site watermark). They are reported rather than guessed at.

## Verification

Measured on the live library:

| Check | Before | After |
|---|---|---|
| Duplicate work groups | 16 same-title + 5 container | **0** |
| `Inception` rows | 2 (one named "أعمال") | **1 row, 4 files** |
| `Iron Man` rows | 3 | **1 row, 5 files** |
| `Titanic` | named "أعمال" | **own row, year 1997** |
| Files linked to an episode | 4,536 | **4,613** (+77) |
| Unlinked files carrying an episode number | 77 | **0** |
| Episodes | 6,066 | **6,132** |
| Works after merge | 280 | **259** |
| Work index | 382 docs, 102 orphans | **259 docs, 0 orphans** |
| Episode index | none | **12,180 docs** |

Container repair output on the live run:

```
work 397  "أعمال" -> "Titanic"
work 398  "أعمال" -> "Inception (merged into work 16)"
work 403  "أعمال" -> "Iron Man (merged into work 367)"
```

Two skipped, both correctly: work 265 has no file, and work 420's filename is a
site watermark with no title in it.

Every enrichment and consolidation step ran with **zero external requests**.

`go build ./...`, `go vet ./...` and `go test ./...` pass.

## A note on a transient test failure observed during this work
One full-suite run reported a failure in `internal/transfer`, and the package
then passed on its own and passed again as part of the complete suite. The cause
is a timeout, not a logic error:

- `TestGoIOSBackendRealDeviceSmoke` blocks for around 20 seconds waiting for a
  real iOS device that is not attached, which is the bulk of that package's
  runtime;
- the full suite therefore runs close to the default `go test` timeout, and a
  loaded machine can push it over.

The package is untouched by this work. It is recorded so the next person who sees
the failure knows it is environmental and where the time goes, rather than
treating it as a catalogue regression. The useful follow-up is to give the
real-device smoke test a build tag or a `-short` guard so it does not delay every
routine run.

## Alternatives considered

1. **Merge container rows with each other.** Rejected: they share only a folder
   name, not an identity, and merging would produce one row claiming to be three
   different films.
2. **Rename a container row to its recovered title without checking for an
   existing work.** Rejected: it either collides with the unique identity index or
   creates a second row for one film. Both are worse than the state being fixed.
3. **Delete folded rows.** Rejected: it destroys the audit trail and any watch
   progress, and makes a wrong merge unrecoverable.
4. **Skip the 77 unlinked files.** Rejected: the library owns them, so the
   catalogue must be able to see them. Creating the episode locally is the only
   option that does not wait for a provider that may never list it.
5. **Invent a title for the watermark-only row.** Rejected: it would trade a
   visible gap for an invisible error, which is the exact failure ADR-010 exists
   to prevent.
