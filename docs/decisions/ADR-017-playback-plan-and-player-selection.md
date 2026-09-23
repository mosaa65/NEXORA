# ADR-017: A Single Playback Read and the Player's Real Selection Surfaces

## Status

Accepted / Implemented

## Context

The watch screen (`/watch/:id`) is the busiest read path in the application and it
had three structural problems, all verifiable from the code as it stood:

1. **It could not start playing without paying for the whole show.** Opening it
   triggered `GET /api/media/{id}` and then `searchAllEpisodes(id)`, which pages the
   episode search index 200 hits at a time for up to 50 requests. Every episode's
   descriptive metadata was fetched before one source URL was rendered.

2. **Its selectors were not real.** `video_files.audio_tracks` and
   `video_files.subtitles` were already inspected by FFprobe and persisted, but the
   player never read them: captions came only from external sidecar files, and audio
   track switching did not exist. There was no way to see which resolution was
   playing, and a file with several releases of the same episode could not choose
   between them.

3. **The player component had become the whole feature.** One 997-line file owned the
   engine, the control bar, the timeline, the caption menu, the rate menu, the
   next-episode countdown, the fullscreen queue and progress persistence — with
   `time` in React state, so every `timeupdate` re-rendered all of it.

## Decision

### 1. One playback read: `GET /api/media/{id}/playback`

A new endpoint answers the playback question directly. `Repository.GetPlaybackPlan`
performs the work's header, its ordered playable files, its episode list, the season
roll-up, the sibling releases of the chosen episode and its next/previous entries —
and it does so in one repository call rather than a chain of round trips.

Ordering is the catalogue's own, never the filesystem's:

```sql
ORDER BY COALESCE(s.season_number, 0), COALESCE(vf.episode_number, 0),
         COALESCE(vf.part_number, 0), vf.id
```

A film part and an episode are both playable items in one ordered list, so next and
previous work for every kind of work without a second code path.

### 2. `?file=` resolves against BOTH id spaces

The work-details page navigates with an **episode** id; a stream URL, a player
bookmark and the copy bridge hold a **`video_files`** id. These are independent
sequences, so an episode id that happens to equal an unrelated file id would select
the wrong file if the server guessed. `SelectPlaybackSource` checks the file space
first, then the episode space, then falls back to the catalogue's first file so a
stale link still plays something.

### 3. Selectors only exist where a real choice exists

- **Quality / versions** lists the sibling releases of the current episode —
  another `video_files` row for the same `episode_id`, or the same
  season/episode/part for a film. A single-release item shows no selector.
- **Audio** is driven by `player.audioTracks()` and subscribed to
  `addtrack`/`removetrack`/`change`. When the browser does not publish tracks for a
  container, the panel lists the FFprobe-persisted tracks as read-only and says so,
  rather than rendering a switcher that cannot switch.
- **Subtitles** keeps the existing external-sidecar pipeline and adds an explicit
  list plus the subtitle styling controls.

No "Auto 2160p/1080p/720p" ladder is offered, because no such representations exist.

### 4. The player is a module, and playback position is not React state

`client/src/components/player/` holds the engine host, controls, timeline, settings,
overlays and the progress hook. `client/src/components/NexoraPlayer.jsx` remains as
the re-exported entry point, so no importer changed.

`PlayerScrub` owns the high-frequency surface: it subscribes to `timeupdate` and
`progress`, coalesces them through one `requestAnimationFrame`, and writes the
played/buffered widths straight into CSS custom properties and element styles. `time`
and `buffered` never enter React state, so a playing video does not re-render the
episode list, the related rail or the page around it.

### 5. Stream responses carry a real media content type

Go's mime table has no entry for `.mkv`, `.ts`, `.m2ts`, `.avi`, `.wmv` or `.flv`.
`mime.TypeByExtension` stays authoritative where it is more specific (the MP4
family), and a small container map fills the gaps. `HEAD` is answered by the same
handler, and a failing media request is answered as `text/plain` with
`Cache-Control: no-store` instead of JSON.

## Evidence from Current Code

- `server/internal/db/repository_playback.go` — `GetPlaybackPlan`,
  `SelectPlaybackSource`, `PlaybackSiblings`, `PlaybackSiblingLabel` and the three
  queries, all documented with the ordering and identity rules they enforce.
- `server/internal/api/handlers_playback.go` — the endpoint and its validation.
- `server/internal/api/server.go` — the route, and `HEAD /api/stream/file/{id}`.
- `server/internal/api/handlers_stream.go` — `mediaContentType` and
  `writeStreamError`.
- `client/src/components/player/*` — the module; `PlayerScrub.jsx` holds the
  rAF-coalesced writer.
- `client/src/components/player/usePlayerProgress.js` — throttled persistence with
  writers on pause, `ended`, `visibilitychange`, `pagehide` and unmount.
- `client/src/pages/WatchPage.jsx` — one playback request, a season selector, and a
  related rail that loads in the background.
- `server/internal/db/repository_playback_test.go` — the id-space resolution, the
  sibling rules and the "no invented label" rule.
- `server/internal/api/server_test.go` — the endpoint contract, the container MIME
  map, and Range/HEAD/416 behaviour against a real file.

## Consequences

- Opening a work costs one request instead of a request plus a paged search.
- `WatchPage` no longer imports `searchAllEpisodes`; the episode index remains the
  source for the details page and search.
- Progress for an episode is keyed by `video_files` id as before, so
  `nexora:playback:{fileId}` positions saved by the previous player still resume.
- Switching a release keeps the episode and the playback position, because the
  source swap resets only the per-source resume state, not the player.
- The `player` chunk stays deferred: the engine is fetched when a watch screen or
  the dock opens, not on first paint.

## What This Does Not Decide

- No remux or transcode fallback, and therefore no server-side handling of a codec
  the browser cannot decode. See the levels table in
  [`VIDEO_PLAYER_ARCHITECTURE.md`](../VIDEO_PLAYER_ARCHITECTURE.md).
- No embedded-subtitle extraction to WebVTT.
- No HDR / bit-depth metadata, so no HDR badge: the current FFprobe call does not
  capture colour transfer or bit depth and inventing it from resolution and codec
  would be fabricated data.
- No server-side watch progress; progress stays browser-local.
- No change to `mediaPathAllowed` or to the existing path-authorization posture.
