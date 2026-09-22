# NEXORA Video Player and Media Experience — Architecture

> **Status:** Implemented on `feature/videojs-player-migration`.
> **Scope:** how playback is assembled today, what the streaming codec strategy is,
> what the player actually selects (quality / audio / subtitles), and the limits we
> did **not** cross.

This document records the current implementation. Anything marked `[PLANNED]` is a
direction only and is not part of the running system.

---

## 1. Baseline found in the repository (before this round)

| Area | State found | File |
|---|---|---|
| Player engine | Video.js 8 behind a NEXORA control shell | `client/src/components/NexoraPlayer.jsx` |
| Watch screen | `/watch/:id` inside `CustomerCinemaLayout` | `client/src/pages/WatchPage.jsx` |
| Floating player | survives navigation | `client/src/components/MiniPlayerDock.jsx`, `client/src/context/PlaybackContext.jsx` |
| Streaming | Go `http.ServeContent` + HTTP Range by DB-resolved file id | `server/internal/api/handlers_stream.go` |
| Subtitles | external sidecar discovery + SRT→WebVTT | `server/internal/media/subtitles.go` |
| Progress | `localStorage` key `nexora:playback:{fileId|src}` | `client/src/lib/watchContent.js` |
| Related | provider-ID relationship graph, no TMDB call while browsing | `server/internal/db/repository_related.go` |
| Episode index | separate Meilisearch index `media_episodes` | `server/internal/search/episode_projection.go` |

The migration branch already delivered the engine, the watch screen, the type-aware
shell, the square episode cards, the related rail, the caption/speed pickers, the
next-episode countdown and the fullscreen queue. This document covers the gaps that
remained and how they were closed.

## 2. Gaps closed in this round

1. **Open-the-player cost.** `WatchPage` issued `/api/media/{id}` **plus**
   `/api/search/episodes?work=` (paged, up to 50 requests) before it could render a
   source. Every episode detail was pulled before the first frame.
2. **No real quality/source selection.** A file with several releases was a single
   opaque row; nothing told the viewer which resolution/codec was playing.
3. **No real audio-track selection.** `video_files.audio_tracks` was inspected by
   FFprobe and persisted, but the UI never read it.
4. **No technical information panel.**
5. **`WatchPage` could not switch seasons**, so a work with 20 seasons paginated
   hundreds of cards into one scrolling column.
6. **Backend stream responses leaked Go's mime table** for containers Go does not
   know (`.mkv`, `.ts`, `.m2ts`), and errors were answered as JSON on a media
   request — which some browsers surface as a generic decode failure.

## 3. Player engine

The engine stays **Video.js 8**, hosting a plain `<video>` element. This was an
explicit decision, not a default:

- It provides the text-track layer that `addRemoteTextTrack` feeds, the
  `playbackRate` plumbing, PiP/fullscreen APIs and one event loop
  (`play`/`pause`/`timeupdate`/`progress`/`loadedmetadata`/`videoTracks`).
- It does not force HLS/DASH, transcoding or a bundler story onto the LAN path.
- The cost is the `player` chunk (≈693 kB raw / 207 kB gzip). It is already isolated
  into its own `manualChunks` entry in `client/vite.config.js` and is only pulled by
  the watch screen, the dev harness and the floating dock — all lazy routes.

Video.js' own control bar, big-play button and hotkeys stay disabled
(`controls: false`, `bigPlayButton: false`, `controlBar: false`,
`userActions: { hotkeys: false, click: false }`) because NEXORA draws exactly one
control surface and defines its own keyboard map. Leaving both enabled duplicated
every button and stacked two timelines; it also fired the same gesture twice.

### 3.1 Component split

The player was one 997-line component. It is now one module directory:

```text
client/src/components/player/
  NexoraPlayer.jsx        engine host, events → state, keyboard, persistence
  PlayerIcons.jsx         the player's icon set (NEXORA stroke language)
  PlayerControls.jsx      bottom bar: transport, time, volume, menu buttons
  PlayerScrub.jsx         progress bar: played / buffered / hover + preview
  PlayerCenterControls.jsx  big play/pause and ±10s
  PlayerSettingsMenu.jsx  one settings surface (quality / audio / subtitles / speed / info)
  PlayerNextOverlay.jsx   next-episode card with countdown
  PlayerStaticOverlays.jsx  resume prompt, buffering, toast, fullscreen chrome
  usePlayerProgress.js    throttled localStorage progress + resume resolution
```

`NexoraPlayer.jsx` is re-exported from the old path so no importer changed.

**State isolation.** `time` and `buffered` are the only values that change on every
`timeupdate`/`progress` tick. They are pushed into a dedicated `stateRef` and
written to the DOM through refs by `PlayerScrub`, so a playing video does not
re-render the episode list, the related rail, or the page around it. `duration`,
`volume`, `rate` and `muted` are React state because they change rarely.

## 4. Source strategy — Direct Play first

```text
user opens /watch/:id(?file=videoFileId)
  → GET /api/media/{id}/playback        (ONE request: work header, source,
                                         episode list, siblings, next/prev, subtitle summary)
  → player src = /api/stream/file/{videoFileId}
  → browser issues GET (usually with Range)
  → Go: GetVideoFilePath → os.Open → http.ServeContent
  → browser decodes directly
```

There is **no transcoding and no HLS/DASH in the running system.** The strategy is:

| Level | When | What happens |
|---|---|---|
| 1 — Direct Play | container/codec supported by the browser (MP4/H.264/AAC, WebM, …) | `http.ServeContent` streams the original bytes with Range. No CPU cost on the server. |
| 2 — Remux | container the browser rejects but codecs it accepts (typical MKV/H.264) | **Not implemented.** Tracked as `[PLANNED]` in §12. |
| 3 — Transcode | codec the browser cannot decode (e.g. HEVC in Chrome) | **Not implemented.** The player reports the real reason instead of pretending (`PlayerErrorState`). |

Transcoding every file would put FFmpeg in the playback hot path and destroy the
zero-copy streaming property the backend is built on, so it is deliberately out of
scope here and requires an owner decision (see ADR-002, ADR-016).

## 5. Codec and technical facts

FFprobe already runs at scan time and its result is persisted, so opening the player
never re-probes a file:

- `video_files.resolution`, `video_files.video_codec`
- `video_files.audio_tracks` (JSONB: index, codec, language, title, channels)
- `video_files.subtitles` (JSONB: index, codec, language, title)

`GET /api/media/{id}/playback` exposes those plus a `technical` summary per file so
the player can label what is actually playing (`4K · HEVC`), show a technical panel,
and list the sibling releases of the same episode as a real version selector.

HDR/10-bit/Dolby Vision badges are **not** emitted: colour transfer and bit depth are
not captured by the current FFprobe call, and claiming them from resolution+codec
would be fabricated data. `[PLANNED]` in §12.

## 6. Quality / source selection

There is exactly one physical source per stream URL, so the selector lists what the
catalogue really holds:

- **Sibling releases** — every `video_file` attached to the same episode
  (`video_files.episode_id`). Choosing one re-points the player at that file and
  preserves the playback position.
- **Current source line** — always shown, from the persisted technical metadata.

No "Auto 2160p / 1080p / 720p" ladder is invented. An ABR ladder would require
multiple rendered representations, which do not exist.

## 7. Audio tracks

Video.js exposes `player.audioTracks()` for tracks it can see. `NexoraPlayer`
subscribes to `addtrack`/`removetrack`/`change` on that list and renders the real
tracks (label = `language · channels · codec`, falling back to the persisted FFprobe
track names). Selecting a track sets `.enabled`.

Browser reality that the UI states instead of hiding: embedded audio track
enumeration depends on the container and the browser. When the browser does not
publish tracks, the settings panel shows the FFprobe-reported tracks and says they
are description-only — it never renders a switcher that cannot switch.

## 8. Subtitles

```text
GET /api/stream/file/{id}/subtitles           → external sidecars (srt/vtt/ass/sub)
GET /api/stream/file/{id}/subtitles/{subId}   → SRT converted to WebVTT, others copied
NexoraPlayer: tracks prop → player.addRemoteTextTrack(...)
             (previous episode's element tracks removed first)
```

`PlayerSettingsMenu` renders "إيقاف" plus every track with an explicit label, so
switching is one tap and never a blind cycle. The `C` key still cycles.
Subtitle styling (size / text shadow / background) is applied through CSS custom
properties on the player root, persisted in `localStorage`, and honoured by the
Video.js cue layer.

Embedded subtitle extraction to WebVTT is **not implemented** (`[PLANNED]`, §12);
the settings panel reports the embedded tracks as description-only.

## 9. Next / previous episode

The DTO's ordering is the catalogue's ordering, not the filesystem's:

```sql
ORDER BY s.season_number, ep.episode_number, ep.id, vf.id
```

`next` / `previous` are computed in `repository_playback.go` from that list plus the
work's file-level rows, so movies with several parts and files without an episode
row follow the same rule. `next` is `null` at the end of a season, of a movie and of
a work — the player then shows "انتهى الموسم / انتهى العمل" next to the related rail
instead of jumping somewhere that does not exist.

The last 30 seconds of a video show `PlayerNextOverlay`: poster, `S02E06`, title,
duration, "تشغيل الآن" and a 10→0 countdown with cancel.

`previous` restarts the current episode when the position is under 10 s, otherwise it
goes to the previous episode (platform convention).

## 10. Performance

| Concern | Implementation |
|---|---|
| Open latency | one DTO request instead of a detail call plus a paged episode search |
| Progress writes | throttled to one write per 5 s, plus forced writes on pause, `ended`, `visibilitychange`, `pagehide` and unmount |
| Episode lists | season-scoped; a work with a 1000-episode season renders its own season, and cards are lazy images |
| Re-renders while playing | `time`/`buffered` are written via refs into the DOM, not into React state |
| Timeline previews | 180 ms debounce + 10-second buckets, cached as immutable JPEGs on disk (unchanged) |
| FFmpeg | never in the playback path; preview generation only on first hover of a bucket |
| Bundle | `video.js` isolated in the `player` chunk; watch screen lazy-loaded |
| API cache | the DTO is a normal cacheable GET with the existing 3-minute TTL |

## 11. What is verified and how

- Frontend: `npm run build` (Vite production build, no type checker configured).
- Backend: `go build ./...` and `go test ./...` (Go toolchain required on the host).
- The player is exercised through `#/dev/player`, which drives the real catalogue.
- `[UNKNOWN]` concurrent-stream throughput: no load harness exists in the repository,
  and this document does not claim a measured concurrency figure.

## 12. Known limitations / PLANNED

1. `[PLANNED]` Remux and transcode fallbacks (§4 levels 2–3) and therefore no
   playback of codecs the browser cannot decode natively.
2. `[PLANNED]` Embedded subtitle extraction to WebVTT.
3. `[PLANNED]` HDR / bit depth metadata in FFprobe results, so HDR badges can be real.
4. `[PLANNED]` Server-side watch progress for cross-device resume.
5. `[PLANNED]` Storyboard sprites for preview hover (removes the first-hover FFmpeg).
6. `[PLANNED]` Skip intro / skip recap — no intro timing data exists in the schema, so
   no UI is shown for it.
7. `[KNOWN RISK]` Preview generation still has no single-flight lock for the first
   simultaneous hover of the same bucket (`PROJECT_ANALYSIS.md`).
8. `[KNOWN RISK]` `mediaPathAllowed` remains lenient for the client-supplied
   `?path=` stream endpoint; playback itself uses the DB-resolved path
   (`serveCataloguePath`) and never accepts a client path.

## 13. Decision references

- ADR-001 / ADR-016: player engine behind a NEXORA control shell.
- ADR-002: direct HTTP Range streaming through Go `http.ServeContent`.
- ADR-004: PostgreSQL as the catalogue source of truth.
- ADR-006: FFmpeg/FFprobe as external processing tools, not a playback engine.
- ADR-007: provider-ID related titles graph.
- ADR-012: local episode enrichment and the separate episode search index.