# ADR-014: Work details as a single-source TMDB layout

- **Status:** Accepted / Implemented
- **Date:** 2026-09-20
- **Revised:** 2026-09-20 — the superseded design is removed **but the episode
  selection and copy flow is carried over**, and the seasons/episodes section is
  rendered in the **platform card template**, not a plain row list. See §4 and §6.
- **Depends on:** ADR-005 (derived search index), ADR-010 (logical media model),
  ADR-011 (rebuildable projection), ADR-012 (separate episode index)

## Context

The work-details screen had grown to 1,024 lines and held **three parallel
sources** for the same question:

```
seasonsList             local seasons and files from /api/media/{id}
seasonSnapshots         the provider's stored seasons, in Arabic
englishSeasonsByNumber  the same provider seasons, in English
```

All three were reconciled in the browser, in the same render. When they
disagreed the screen disagreed with itself: a season could be shown twice, an
episode counted from one source and titled from another, and the Arabic and
English names came from different maps that were never joined.

ADR-012 already solved the underlying problem on the server. The episode index
returns one document per episode, already enriched from the provider snapshots
stored locally and already carrying the parent work's titles, so a single hit
renders "One Piece — S01E04" without a second lookup. The screen was simply not
using it.

Two components belonged to the superseded design: `PlayableFilesExplorer`
(the local file tree with its own view modes) and `ViewModeMenu` (the
"icons / details" toggle). Both were used by the details page and by nothing
else.

## Decision

### 1. One source for seasons and episodes

`MediaDetailsPage` reads its seasons and episodes from the episode index alone:

```
GET /api/media/{id}                     the work's own facts
GET /api/episodes/search?work={id}      every episode, enriched and merged
```

There is no client-side reconciliation. A season is a fold over the hits by
`season_number`. The viewer's chosen season is tracked by **season number**, not
by array index, so a re-group cannot point at a different season. The default is
the first season whose number is greater than 0 — provider season 0 is the
specials bucket and sorts first, and opening a show on its specials is the wrong
default.

**The provider season snapshots are still read, for one purpose only: season
posters.** Artwork is not an episode fact, the episode index does not carry it,
and no logic is derived from it. The distinction matters — a presentation-only
read that cannot disagree with the source of truth is not a second source of
truth.

### 2. An absent episode is shown in place, not hidden

An episode the library has not acquired (`has_local_file: false`) renders in its
normal position, greyed, with no play button and a quiet "قيد الإضافة" label
(not-added-yet). It is not moved to a separate section and it is not hidden
behind a toggle.

Order is the reason: episode 5 exists at the provider, so it belongs between 4
and 6. A separate "coming soon" section would preserve the count and destroy the
sequence, which is the thing a viewer actually reads.

### 3. Search inside one work

The source allows filtering episodes by title without a new endpoint, so the
season header carries a search box. It filters the already-loaded episodes
locally, so typing does not hit the server.

### 4. The superseded design is removed, but its selection feature is carried over
`PlayableFilesExplorer.jsx` and `ViewModeMenu.jsx` are deleted: they are the old
design, they had no other consumer (verified), and the view-mode toggle is being
rebuilt deliberately rather than left half-working.

**The episode selection and copy flow is NOT deleted — it moves onto the new
cards.** Per-episode selection controls plus the "انسخ من إلى" range popover
(`RangeSelectionBar`) let an operator pick episodes and send them through the
existing transfer system (ADR-008). Selection is held as a **Set of episode
ids**, so a filter or a season switch cannot move the selection onto a different
episode, and **only an episode the library holds can be selected or copied** — a
provider-only episode has no file to send.

`MediaCollection` is untouched: it renders the catalogue grid, not a work's
episodes.

### 5. A film keeps a file list
A film has no episode rows. When the index returns no episodes, the work's own
files are listed directly. A file carrying no season or episode number is never
dropped either — it is listed under "other files" beneath the episode list, so a
file the library owns stays reachable.

### 6. The platform card template is kept, extended with the local facts

The section renders in the **platform card template**: a horizontal rail of
vertical season cards (poster, gradient, episode-count badge, amber ring on the
selected season) and a horizontal rail of `aspect-video` episode cards
(episode-number badge, dual titles, metadata row). This is the design the owner
asked to preserve, and it is reused as-is.

Because the episode index already carries the library's own facts, each card
also shows **resolution**, **size**, **runtime** and **how many release files
back the episode** ("N إصدارات"). No extra request is made. A latin-valued fact
row is marked `dir="ltr"`, because in an RTL context `2.0 KB` otherwise renders
as `KB 2.0`.

A plain row list from the single source was built first and rejected: it
discards the artwork and density that make a long show scannable.

## The paging defect this exposed

The episode list is only correct if it holds **every** episode of the work, and
verifying it against the live library showed it did not:

| Defect | Effect on the screen | Fix |
|---|---|---|
| The endpoint had no `offset` | only the first page of a work's episodes could be requested at all | added `offset` to `/api/episodes/search` and paged the whole work in the client |
| Meilisearch stops at `maxTotalHits: 1000` by default | a work with more episodes silently lost everything past the 1000th hit | set `pagination.maxTotalHits` on the episode index |

Both were found by measuring, not by reading. A work with 1,181 episodes
returned 200 hits for a `limit=200` request and 1,000 hits for a paged read —
against 1,220 rows actually in the index for that work. The screen would have
rendered **six seasons of twenty-three** and reported nothing wrong.

The 1,000 hit ceiling is a Meilisearch default that only becomes visible on a
work large enough to cross it. The live library has exactly one such work, which
is why the defect survived the previous round.

## Consequences

**Positive**

- The screen has one answer per question, so it cannot contradict itself.
- `MediaDetailsPage` went from 1,024 lines to about 810, and the removed lines
  were the reconciliation logic and the second season section, not features.
  The selection controls and the range copy popover were added back on the new
  cards.
- Filtering by season is a property of the loaded list rather than a second
  fetch, and the in-work search is free.
- "Coming soon" is a fact the index already knew (`has_local_file`) rather than a
  guess made by comparing two client-side collections.
- Two components and their view-mode storage keys are gone.

**Negative / accepted trade-offs**

- The episode list loads the whole work up front. For 1,220 episodes that is one
  request per 200, measured at well under a second each, and it is bounded: a
  work cannot grow past the paging guard.
- Season posters still come from the provider snapshots. If those are missing the
  season card falls back to the placeholder image; the episode data is
  unaffected. Accepted: the alternative is storing artwork on the episode
  document.
- `maxTotalHits: 10000` raises the ceiling for every query against the index, not
  just the paged one. Accepted: the index is filtered by `work_id` in the common
  case, and the limit exists to bound deep pagination rather than to bound the
  data.

## Verification

Checked in a real headless browser (Chromium) against the live server, on
`One Piece` — 23 seasons, 1,181 episodes:

| Check | Result |
|---|---|
| Work renders | yes |
| Season cards rendered | **24** (23 seasons + specials) |
| Old provider section heading | absent |
| Episode cards rendered | yes, with titles, overview and stills |
| "قيد الإضافة" state | present, on episodes with `has_local_file: false` |
| Play action | present, on episodes the library holds |
| In-work episode search | present |
| View-mode control (old design) | absent |

On a work carrying local files, the added features were checked by acting on
them:

| Check | Result |
|---|---|
| Default season | **Season 1**, not specials |
| Resolution badge | **1080p** shown |
| Size badge | shown, latin order correct |
| Release-count badge | "2 إصدارات" shown |
| Selection controls | 5 rendered |
| Selecting an episode | selection count **0 → 1**, copy bar **false → true** |
| Episode rail | horizontal (`overflow-x: auto`) |

Episodes returned by paging for that work: **1,220**, matching the index.

`go build ./...`, `go vet ./...`, `npm run build` all pass.

Console errors were limited to `ERR_CONNECTION_REFUSED` on `127.0.0.1:32145`,
which is the optional local Copy Bridge not running in this environment
(ADR-008), not a page error.

## Alternatives considered

1. **Keep three sources and reconcile them more carefully.** Rejected: the
   reconciliation is the defect. Making it more careful keeps 300 lines of
   logic whose only job is to paper over a disagreement the server already
   resolved.
2. **Fetch episodes per season instead of per work.** Rejected: the common case
   is browsing a work's episodes, and one request per season on a 23-season show
   is 23 round trips to render one page.
3. **Hide episodes the library lacks.** Rejected: the provider's episode count is
   what tells a viewer the show continues. Hiding them makes the catalogue look
   complete when it is not.
4. **Add an `offset` to the endpoint but leave `maxTotalHits` alone.** Rejected:
   paging past a silent ceiling returns fewer rows without an error, which is the
   hardest kind of defect to notice — it is exactly how this one was found.
5. **Delete the copy flow with the explorer.** Rejected: the transfer system
   (ADR-008) is a working feature, and losing the ability to pick episodes and
   send them would be a regression. The selection UI is rebuilt on the new cards
   instead.
6. **Keep `PlayableFilesExplorer` as a second, local section beside the new
   one.** Rejected: two sections describing one thing is the defect being
   removed.
