# ADR-010: Logical media model and entity resolution

- **Status:** Accepted
- **Date:** 2026-08-30
- **Depends on:** ADR-009 (incremental indexing pipeline)
- **Supersedes:** the implicit model where every parsed filename became a work.

## Context

ADR-009 fixed how the filesystem is traversed. It did not fix what the result
*means*. The ingest path still did this:

```text
File → ParsedName → Database Record
```

so every file became a work. Running that against the real library produced
these rows in `media_items`:

| Work created | What it actually is |
|---|---|
| `الكنز ج1 الحلقه` | "The Treasure season 1 episode" — a **season+episode marker** became a work name |
| `الكنزنت` | a site watermark concatenated onto the title |
| `منور فديوهات زابيا` | "Munawar videos Zabyia" — a **channel description** became a work |
| `Fate_Apocrypha` | separators never normalized, so it can never match `Fate Apocrypha` |
| `Fate Stay Night -` | a **dangling separator** left by the parser |
| `ONE PIECE` | 12 **episodes** filed under `type = 'movie'` |
| `الكنز` | one work holding 88 files, including five files claiming episode 1 of season 1 |

None of these are works. They are parsing artifacts and container folders that
were promoted to entities, and once promoted they poisoned search, hubs, and
every later scan.

Two further problems share the same root cause:

- **The same work appeared many times.** `Breaking Bad`, `Breaking.Bad` and
  `Breaking_Bad` are one show but compared unequal, because identity was exact
  string equality.
- **A correct operator decision could be destroyed.** Nothing recorded where a
  value came from, so a Full Scan overwrote an admin's edit with the raw parser
  guess.

## Decision

Separate the **physical file** from the **logical work**, and insert an explicit
resolution stage between them.

```text
Filesystem file
  ↓ parse
Media candidate
  ↓ ENTITY RESOLUTION          (evidence → ranked candidates → decision)
Logical work  →  season  →  episode  →  physical file
```

### 1. Two axes, not one

`media_type` (movie | series | unknown) and `category` (movies, series, anime,
kids, documentaries, plays) are independent. `anime` is a category that is
*usually* a series, but an anime film is a movie. Category therefore never
determines data shape.

Media type is decided from **structure**, not from naming:

| Series evidence | Movie evidence |
|---|---|
| explicit episode marker (`S01E04`, `1x04`, `الحلقة 4`) | category folder is movies/plays |
| a season folder | a release year in the filename |
| a contiguous sibling episode run (`01..04`) | a part/CD/disc marker |
| ≥3 episodic siblings | siblings carrying years, not numbers |

When the two sides are close, the result is `unknown` and the file goes to
review. Guessing here is exactly what filed series as films.

### 2. Evidence-weighted candidate matching

A new file is **never** turned into a work immediately. It is scored against
existing entities:

| Evidence | Weight |
|---|---|
| exact normalized title | +40 |
| folder title match | +25 |
| learned alias match | +20 |
| existing episode relationship | +20 |
| group consensus (≥2 siblings agree) | +18 |
| series-shaped neighbourhood | +14 |
| season compatible | +10 |
| year compatible | +10 |
| category compatible | +10 |
| provider identity | +45 |
| contradictory evidence | −30 |
| type conflict | −25 |

Every point is a **named item in the breakdown**, which is what the review queue
displays. A decision is never an opaque number.

Thresholds: ≥70 auto-attach, ≥45 provisional (attach + review), otherwise review
or create-provisional. When the top two candidates are within 8 points the
decision is **downgraded to review regardless of absolute score** — a confident
score means nothing when a rival scores the same, which is the `Silo` vs
`Silo (2023)` case.

### 3. Never create a work too quickly

Creating an entity is the last resort, not the first behaviour:

1. Reject a title that is not a title (watermark, bare number, dangling
   separator, structural keyword, long unbroken run).
2. Look up a learned alias — exact knowledge beats similarity.
3. Score candidates and apply the ambiguity margin.
4. Create a provisional work **only** when the media type is certain and the
   title is usable.
5. Otherwise write to `resolution_queue` and leave nothing else behind.

An unresolved file is recoverable. A polluted library is not.

### 4. Group resolution

Resolving a file in isolation is the main cause of invention. `01.mkv` says
nothing, but `01..04.mkv` inside `Silo/` says almost everything. The scanner now
builds a per-directory summary (size, episodic siblings, episode numbers, year
siblings, season-shape, consensus title from ≥2 agreeing files) and every file
carries its neighbourhood. The collector is **bounded** (default 20 000
directories, oldest-half eviction) so a library with a directory per file cannot
grow memory without limit; eviction is safe because a summary is an optimisation.

### 5. Rule-based library memory, no AI

`media_aliases` maps every confirmed spelling to one work, with
`UNIQUE (alias_normalized)` so a mapping is deterministic and a conflict is
surfaced rather than silently resolved by row order. Aliases are learned
automatically **only** from an auto/provisional resolution and **only** when the
new spelling is ≥0.9 similar to a canonical name. A review-queue item never
teaches the library. Operator decisions are always stored, marked `admin`.

### 6. Provenance decides conflicts

Every important field carries its source, ranked
`admin(100) > tmdb(80) > database(60) > resolver(40) > parser(20) > filesystem(10)`.
`MergeField` discards an incoming value that loses to a locked or higher-ranked
existing value, and reports which fields actually changed so persistence issues a
minimal `UPDATE` instead of rewriting every column. This is what makes a Full
Scan unable to revert an operator's correction.

### 7. Physical file → logical entity

`video_files.episode_id` is the link that removes the ambiguity of matching on
`(season_id, episode_number)` alone. Several release files may attach to one
episode without becoming extra works. A new season attaches to its work and is
never promoted to a work of its own.

### 8. Search is a projection

`search_projection_state` tracks per-kind projection progress so the index can be
dropped and rebuilt **from the database without re-reading the filesystem**, and
so adding one episode updates one document rather than rebuilding the library.

## Consequences

**Positive**

- A file no longer creates a work by default; the default is review.
- `Toy Story 2`, `John Wick Chapter 4` and `01 - Batman Begins` are films, not
  episodes.
- `Breaking Bad`, `Breaking.Bad` and `بريكنغ باد` resolve to one work.
- Existing degenerate entities are marked `provisional` and can be merged
  (audit-preserving: `merged_into_id`, nothing deleted).
- Operator corrections survive every future scan.

**Negative / accepted trade-offs**

- Ambiguity is surfaced rather than hidden, so a new library produces review
  items. This is the intended cost of not polluting the catalogue.
- Learned aliases are persisted state; a wrong auto-learned alias would
  mis-resolve. Mitigated by the similarity gate and by never learning from a
  review item.
- The resolver compares a file against the whole candidate set. Bounded by
  loading the candidate set once per scan, not per file.
- Rename inference, subtitle handling and playback were not changed by this ADR.

## Verification

- `go build ./...`, `go vet ./...`, `go test ./...` all pass.
- `internal/identity` carries 31 tests and 36 subtests covering normalization
  unification, word-aligned similarity, the real rejection corpus, media-type
  detection, season/episode resolution, group resolution, alias learning,
  provenance precedence, ambiguity, and duplicate detection.
- **Migration 0023 applied to the live database on the first attempt**, with 253
  existing works correctly marked provisional and 543 files backfilled.
- `cmd/resolutionprobe` runs the real pipeline over the live library:

| Metric | Result |
|---|---|
| media files | 137 |
| works proposed | 68 |
| queued for review | 1 (0.7%) |
| **regressions vs the old ingest** | **0** |

The probe's regression check explicitly asserts that `01`, `03`, `x2`, `10`,
`الحلقة`, `الكنزنت`, `منور فديوهات زابيا` and `Fate Stay Night -` are **not**
created as works.

### Defects found by running on real data

Running against the actual library, rather than fixtures, exposed four defects
that unit tests had not:

1. Arabic-Indic digits never folded, because `unicode.IsDigit` was tested after
   `unicode.IsLetter` in the normalizer.
2. `releaseGroupRE` stripped `" - Batman Begins"` as a release-group suffix,
   destroying the title and leaving `01`.
3. A trailing count read a release year as an episode number, truncating the
   title to `01 Batman Begins`.
4. A `Part N` prefix inside a franchise folder let the trailing chapter number
   become an episode (`John Wick Chapter 3` → `John Wick Chapter`).

## Known refinements

Honestly recorded rather than silently accepted:

- `Franchises/Marvel/Part 1 ...` still groups under `Part 1`/`Part 2`; a
  `Part` folder should map to its parent franchise folder as the work.
- `John Wick Chapter` (no number) and `John Wick Chapter 2/4` are three works;
  sequel consolidation (`base title` + chapter number) is not yet implemented.
- `Broken Names/01`, `x2`, `10` still resolve as works inside a folder that is
  explicitly named "Broken Names"; the folder passes the title-quality check.
- `Days.of.Being.Wild`, `Ayla` and `Marvel` show the same fragment-shape as
  (1), meaning a short folder beside a longer filename can still win.

These are candidate-driven improvements to the same evidence model, not
architectural gaps. None creates a regressing entity, and each is visible in the
review queue or the probe rather than hidden in the catalogue.
