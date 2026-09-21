import { useCallback, useEffect, useRef, useState } from "react";
import videojs from "video.js";
import "video.js/dist/video-js.css";

const SEEK_SECONDS = 10;
const RATES = [0.75, 1, 1.25, 1.5, 2];
const SPACE_KEY = " ";
const progressKeyFor = (fileId, src) => `nexora:playback:${fileId || src}`;

const clock = (value = 0) => {
  const seconds = Math.max(0, Math.floor(Number.isFinite(value) ? value : 0));
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return hours > 0
    ? `${hours}:${String(minutes).padStart(2, "0")}:${String(seconds % 60).padStart(2, "0")}`
    : `${minutes}:${String(seconds % 60).padStart(2, "0")}`;
};

function PlayerIcon({ name, className = "h-6 w-6" }) {
  const common = {
    viewBox: "0 0 24 24",
    className,
    "aria-hidden": true,
    fill: "none",
    stroke: "currentColor",
    strokeWidth: 2,
    strokeLinecap: "round",
    strokeLinejoin: "round",
  };
  if (name === "play") {
    return <svg {...common} fill="currentColor" stroke="none"><path d="M8 5.4v13.2c0 .78.86 1.26 1.53.86l10.2-6.6a1 1 0 000-1.72L9.53 4.54A1 1 0 008 5.4z" /></svg>;
  }
  if (name === "pause") {
    return <svg {...common} fill="currentColor" stroke="none"><rect x="7" y="5" width="3.5" height="14" rx="1" /><rect x="13.5" y="5" width="3.5" height="14" rx="1" /></svg>;
  }
  if (name === "rewind" || name === "forward") {
    return <svg {...common} className={`${className} ${name === "forward" ? "scale-x-[-1]" : ""}`}><path d="M4 10a8.5 8.5 0 1 1 1.35 7.1" /><path d="M4 4.5V10h5.5" /></svg>;
  }
  if (name === "volume") {
    return <svg {...common}><path d="M4 10v4h4l5 4V6L8 10H4z" /><path d="M16 9a4 4 0 010 6M18.5 6.5a7.5 7.5 0 010 11" /></svg>;
  }
  if (name === "mute") {
    return <svg {...common}><path d="M4 10v4h4l5 4V6L8 10H4zM17 10l4 4m0-4l-4" /></svg>;
  }
  if (name === "pip") {
    return <svg {...common}><rect x="3.5" y="5" width="17" height="14" rx="2" /><rect x="12.5" y="12" width="5" height="4" rx=".7" fill="currentColor" stroke="none" /></svg>;
  }
  if (name === "fullscreen") {
    return <svg {...common}><path d="M8 3H3v5m13-5h5v5M8 21H3v-5m13 5h5v-5" /></svg>;
  }
  if (name === "playlist") {
    return <svg {...common}><path d="M5 6h14M5 12h14M5 18h9" /><path d="M18 16v5m-2.5-2.5h5" /></svg>;
  }
  return null;
}

/**
 * NexoraPlayer — Video.js host with the NEXORA control surface.
 *
 * Owned here:
 *   - Video.js lifecycle: create once, switch source, dispose on unmount.
 *   - NEXORA behaviour Video.js does not ship: ±10s jump buttons, double-click
 *     half-screen seek, the NEXORA keyboard map (Space/K, ←/J, →/L, M, F, C),
 *     resume prompt, progress persistence, auto-next episode, timeline thumbnail
 *     previews, a fullscreen episode drawer, and the RTL/skin treatment.
 *   - The external `tracks` prop is translated into Video.js text tracks, so the
 *     NEXORA subtitle sources drive the built-in captions menu.
 *
 * Consumed by WatchPage, which owns the surrounding layout, the episode list
 * and the fullscreen controls.
 */
export default function NexoraPlayer({
  src,
  title,
  poster,
  fileId,
  tracks = [],
  onNext,
  playlist = [],
  currentFileId,
  onSelectFile,
  // Optional: the element to fullscreen. Defaults to the player shell. A page
  // that wraps the player in a larger stage can pass its own ref so fullscreen
  // covers that stage and keeps the page's own overlay controls in view.
  fullscreenTarget,
  onMinimize,
  onExit,
  // When true, resume the saved position automatically (no prompt) — used when
  // the player is re-mounted into the floating dock so playback continues where
  // it left off instead of asking the user again.
  autoResume = false,
}) {
  const containerRef = useRef(null);
  const playerRef = useRef(null);
  const hideTimer = useRef(null);
  const clickTimer = useRef(null);
  const previewTimer = useRef(null);
  const previewBucket = useRef(null);
  const lastSavedAt = useRef(0);
  const resumeApplied = useRef(false);
  const remoteTracks = useRef([]);

  const [ready, setReady] = useState(false);
  const [playing, setPlaying] = useState(false);
  const [duration, setDuration] = useState(0);
  const [time, setTime] = useState(0);
  const [volume, setVolume] = useState(1);
  const [muted, setMuted] = useState(false);
  const [rate, setRate] = useState(1);
  const [visible, setVisible] = useState(true);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [captionIndex, setCaptionIndex] = useState(-1);
  const [hoverTime, setHoverTime] = useState(null);
  const [previewSrc, setPreviewSrc] = useState("");
  const [resumeAt, setResumeAt] = useState(0);
  const [askResume, setAskResume] = useState(false);
  const [episodeDrawerOpen, setEpisodeDrawerOpen] = useState(false);

  // ---------------------------------------------------------------------------
  // Progress persistence (localStorage, same key/shape as the legacy player so
  // existing saved positions survive the migration).
  // ---------------------------------------------------------------------------
  const saveProgress = useCallback((force = false) => {
    const player = playerRef.current;
    if (!player || player.isDisposed()) return;
    const total = player.duration() || 0;
    const current = player.currentTime() || 0;
    if (!total || (!force && Math.abs(current - lastSavedAt.current) < 10)) return;
    lastSavedAt.current = current;
    const completed = current / total >= 0.95;
    try {
      localStorage.setItem(
        progressKeyFor(fileId, src),
        JSON.stringify({
          position: completed ? 0 : current,
          duration: total,
          completed,
          updatedAt: Date.now(),
        })
      );
    } catch {}
  }, [fileId, src]);

  // ---------------------------------------------------------------------------
  // NEXORA control actions
  // ---------------------------------------------------------------------------
  const showControls = useCallback(() => {
    setVisible(true);
    clearTimeout(hideTimer.current);
    const player = playerRef.current;
    if (player && !player.paused()) {
      hideTimer.current = setTimeout(() => setVisible(false), 2500);
    }
  }, []);

  const hideControls = useCallback(() => {
    clearTimeout(hideTimer.current);
    const player = playerRef.current;
    if (player && !player.paused() && !askResume) setVisible(false);
  }, [askResume]);

  const seekBy = useCallback((seconds) => {
    const player = playerRef.current;
    if (!player) return;
    const total = player.duration() || 0;
    const next = Math.max(0, Math.min(total, (player.currentTime() || 0) + seconds));
    player.currentTime(next);
    setTime(next);
  }, []);

  const togglePlay = useCallback(() => {
    const player = playerRef.current;
    if (!player) return;
    if (player.paused()) {
      const attempt = player.play();
      if (attempt && typeof attempt.catch === "function") attempt.catch(() => {});
    } else {
      player.pause();
    }
  }, []);

  // Fullscreen targets the whole NEXORA shell, not Video.js' inner element.
  // Video.js' own requestFullscreen() fullscreens `.video-js`, which would hide
  // every NEXORA layer (the bar, the centre controls, the episode drawer) since
  // they are siblings of `.video-js`, not children of it.
  const toggleFullscreen = useCallback(() => {
    const root = fullscreenTarget?.current || containerRef.current;
    if (!root) return;
    if (document.fullscreenElement) document.exitFullscreen?.().catch(() => {});
    else root.requestFullscreen?.().catch(() => {});
  }, [fullscreenTarget]);

  const togglePiP = useCallback(() => {
    const player = playerRef.current;
    if (!player) return;
    const el = player.el()?.querySelector?.("video") || player.tech?.()?.el?.();
    if (!el) return;
    if (document.pictureInPictureElement) {
      document.exitPictureInPicture?.().catch(() => {});
    } else {
      el.requestPictureInPicture?.().catch(() => {});
    }
  }, []);

  const setVolumeTo = useCallback((value) => {
    const player = playerRef.current;
    if (!player) return;
    player.volume(value);
    player.muted(value === 0);
  }, []);

  const toggleMute = useCallback(() => {
    const player = playerRef.current;
    if (!player) return;
    player.muted(!player.muted());
  }, []);

  const applyRate = useCallback((value) => {
    const player = playerRef.current;
    if (!player) return;
    player.playbackRate(value);
  }, []);

  const cycleCaptions = useCallback(() => {
    const player = playerRef.current;
    if (!player) return;
    const list = player.textTracks();
    if (!list || list.length === 0) return;
    let current = -1;
    for (let i = 0; i < list.length; i += 1) {
      if (list[i].mode === "showing") {
        current = i;
        break;
      }
    }
    const next = current + 1 >= list.length ? -1 : current + 1;
    for (let i = 0; i < list.length; i += 1) {
      list[i].mode = i === next ? "showing" : "disabled";
    }
    setCaptionIndex(next);
  }, []);

  // ---------------------------------------------------------------------------
  // Player creation. Runs once per mounted node.
  // ---------------------------------------------------------------------------
  useEffect(() => {
    if (!containerRef.current || playerRef.current) return;

    const videoElement = document.createElement("video-js");
    videoElement.classList.add("vjs-big-play-centered");
    containerRef.current.appendChild(videoElement);

    playerRef.current = videojs(videoElement, {
      // Video.js' own control bar and big-play button are off: NEXORA draws the
      // single control surface (see the strip/timeline below). Two control bars
      // on one element duplicated every button and stacked two timelines.
      controls: false,
      bigPlayButton: false,
      controlBar: false,
      preload: "metadata",
      fluid: false,
      fill: true,
      playsinline: true,
      // Video.js hotkeys are disabled: NEXORA defines its own map so the keys
      // match the legacy player exactly, and so a key cannot fire twice.
      // Click-to-play is disabled too, because the NEXORA shell owns the click
      // (single = play/pause, double = half-screen seek); leaving Video.js' own
      // click on would toggle play a second time for the same gesture.
      userActions: { hotkeys: false, click: false },
      playbackRates: RATES,
    });
    setReady(true);

    // No disposal here. React StrictMode runs this cleanup between its two
    // development effect passes, and disposing there would destroy a player we
    // are about to reuse. The unmount-only effect below owns the teardown.
  }, []);

  // Dedicated teardown: dispose the player and drop the reference so a later
  // mount builds a fresh instance instead of touching a disposed one.
  useEffect(() => {
    return () => {
      const player = playerRef.current;
      if (player && !player.isDisposed()) player.dispose();
      playerRef.current = null;
      clearTimeout(hideTimer.current);
      clearTimeout(clickTimer.current);
      clearTimeout(previewTimer.current);
    };
  }, []);

  // ---------------------------------------------------------------------------
  // Source switching, kept explicit so a next-episode swap cannot leave the
  // previous file attached to the element. Resume is resolved from localStorage
  // on each metadata load, exactly like the legacy player.
  // ---------------------------------------------------------------------------
  useEffect(() => {
    const player = playerRef.current;
    if (!player || player.isDisposed() || !src) return;
    setHoverTime(null);
    setPreviewSrc("");
    previewBucket.current = null;
    resumeApplied.current = false;
    player.src({ src, type: "video/mp4" });
    if (poster) player.poster(poster);
    if (title) player.el().setAttribute("aria-label", title);
  }, [src, poster, title, ready]);

  // ---------------------------------------------------------------------------
  // Video.js events → React state. Every subscription is paired with an `off`
  // in the same cleanup, which is what keeps this leak-free across swaps.
  // ---------------------------------------------------------------------------
  useEffect(() => {
    const player = playerRef.current;
    if (!player || !ready) return;

    const onPlay = () => {
      setPlaying(true);
      showControls();
    };
    const onPause = () => {
      setPlaying(false);
      setVisible(true);
      saveProgress(true);
    };
    const onTimeUpdate = () => {
      setTime(player.currentTime() || 0);
      saveProgress();
    };
    const onDurationChange = () => setDuration(player.duration() || 0);
    const onVolumeChange = () => {
      setVolume(player.volume() ?? 1);
      setMuted(Boolean(player.muted()));
    };
    const onRateChange = () => setRate(player.playbackRate() || 1);
    const onLoadedMetadata = () => {
      const total = player.duration() || 0;
      setDuration(total);
      if (resumeApplied.current) return;
      resumeApplied.current = true;
      try {
        const saved = JSON.parse(localStorage.getItem(progressKeyFor(fileId, src)) || "null");
        if (saved?.position > 30 && saved.position < total - 30 && !saved.completed) {
          if (autoResume) {
            // Dock re-mount: continue silently from where playback stopped.
            player.currentTime(saved.position);
          } else {
            setResumeAt(saved.position);
            setAskResume(true);
          }
        }
      } catch {}
    };
    const onEnded = () => {
      saveProgress(true);
      onNext?.();
    };

    player.on("play", onPlay);
    player.on("pause", onPause);
    player.on("timeupdate", onTimeUpdate);
    player.on("durationchange", onDurationChange);
    player.on("loadedmetadata", onLoadedMetadata);
    player.on("volumechange", onVolumeChange);
    player.on("ratechange", onRateChange);
    player.on("ended", onEnded);

    // Seed initial values so the strip does not read 0:00 until the first event.
    setDuration(player.duration() || 0);
    setVolume(player.volume() ?? 1);
    setMuted(Boolean(player.muted()));
    setRate(player.playbackRate() || 1);

    return () => {
      player.off("play", onPlay);
      player.off("pause", onPause);
      player.off("timeupdate", onTimeUpdate);
      player.off("durationchange", onDurationChange);
      player.off("loadedmetadata", onLoadedMetadata);
      player.off("volumechange", onVolumeChange);
      player.off("ratechange", onRateChange);
      player.off("ended", onEnded);
    };
  }, [ready, showControls, saveProgress, fileId, src, onNext, autoResume]);

  // Fullscreen state follows the DOM. Fullscreen may be applied to the shell
  // (containerRef) or, when the host page wraps us in a bigger stage, to that
  // stage; either counts as "the player is fullscreen" for our chrome.
  useEffect(() => {
    const syncFullscreen = () => {
      const active = document.fullscreenElement;
      const target = fullscreenTarget?.current || containerRef.current;
      const full = Boolean(active) && active === target;
      setIsFullscreen(full);
      if (!full) setEpisodeDrawerOpen(false);
      setVisible(true);
    };
    document.addEventListener("fullscreenchange", syncFullscreen);
    syncFullscreen();
    return () => document.removeEventListener("fullscreenchange", syncFullscreen);
  }, [fullscreenTarget]);

  // Persist the last position when the player unmounts (closing the modal or
  // switching episodes), matching the legacy `useEffect(() => () => …)` teardown.
  useEffect(() => () => saveProgress(true), [saveProgress]);

  // ---------------------------------------------------------------------------
  // Subtitle wiring: translate the NEXORA `tracks` prop into Video.js text
  // tracks. Video.js owns the captions menu; we only feed it the sources.
  // A track already present for the same src is left untouched so re-renders do
  // not reset the user's selected caption.
  // ---------------------------------------------------------------------------
  useEffect(() => {
    const player = playerRef.current;
    if (!player || !ready || player.isDisposed()) return;

    const wanted = (tracks || []).filter((track) => track?.src);

    // Drop the previous file's tracks first, otherwise the captions menu keeps
    // every episode's subtitles and cycling lands on a dead source.
    remoteTracks.current.forEach((element) => {
      try {
        player.removeRemoteTextTrack(element);
      } catch {}
    });
    remoteTracks.current = [];

    wanted.forEach((track) => {
      const element = player.addRemoteTextTrack(
        {
          kind: track.kind || "subtitles",
          label: track.label || "العربية",
          srclang: track.srcLang || "ar",
          src: track.src,
          default: Boolean(track.default),
        },
        false
      );
      if (element) remoteTracks.current.push(element);
    });

    setCaptionIndex(-1);
  }, [tracks, ready, fileId, src]);

  // ---------------------------------------------------------------------------
  // NEXORA keyboard map, scoped to the player element rather than `document`.
  // Video.js hotkeys are off (see creation options), so nothing double-fires.
  // ---------------------------------------------------------------------------
  useEffect(() => {
    const root = containerRef.current;
    if (!root) return;

    const onKeyDown = (event) => {
      const tag = event.target?.tagName;
      if (["INPUT", "TEXTAREA", "SELECT"].includes(tag)) return;
      const key = event.key.toLowerCase();
      if (![SPACE_KEY, "k", "arrowleft", "arrowright", "j", "l", "m", "f", "c"].includes(key)) return;
      event.preventDefault();
      if (key === SPACE_KEY || key === "k") togglePlay();
      if (key === "arrowleft" || key === "j") seekBy(-SEEK_SECONDS);
      if (key === "arrowright" || key === "l") seekBy(SEEK_SECONDS);
      if (key === "m") toggleMute();
      if (key === "f") toggleFullscreen();
      if (key === "c") cycleCaptions();
      showControls();
    };

    root.addEventListener("keydown", onKeyDown);
    return () => root.removeEventListener("keydown", onKeyDown);
  }, [cycleCaptions, seekBy, showControls, toggleFullscreen, toggleMute, togglePlay]);

  // Timeline hover: quantize the pointer position to a ten-second bucket and
  // fetch the cached FFmpeg frame for that bucket. The same endpoint the legacy
  // player used, guarded by `fileId`.
  function onHoverProgress(event) {
    const box = event.currentTarget.getBoundingClientRect();
    const nextTime = duration
      ? Math.max(0, Math.min(duration, ((event.clientX - box.left) / box.width) * duration))
      : null;
    setHoverTime(nextTime);
    if (!fileId || nextTime === null) return;
    const bucket = Math.floor(nextTime / 10) * 10;
    if (previewBucket.current === bucket) return;
    previewBucket.current = bucket;
    clearTimeout(previewTimer.current);
    previewTimer.current = setTimeout(
      () => setPreviewSrc(`/api/stream/file/${fileId}/preview?at=${bucket}`),
      180
    );
  }

  function onSeek(event) {
    const player = playerRef.current;
    const next = Number(event.target.value);
    if (player) player.currentTime(next);
    setTime(next);
  }

  function resolveResume(resume) {
    const player = playerRef.current;
    if (!player) return;
    player.currentTime(resume ? resumeAt : 0);
    setAskResume(false);
    if (resume) {
      const attempt = player.play();
      if (attempt && typeof attempt.catch === "function") attempt.catch(() => {});
    }
  }

  // Click toggles play, double-click jumps half a screen either side. Applied
  // to the shell so it covers the video area but never the control bar.
  function onShellClick(event) {
    if (askResume) return;
    if (event.target.closest?.(".vjs-control-bar, .vjs-big-play-button, button, input, select")) return;
    clearTimeout(clickTimer.current);
    clickTimer.current = setTimeout(() => togglePlay(), 220);
  }

  function onShellDoubleClick(event) {
    if (askResume) return;
    if (event.target.closest?.(".vjs-control-bar, .vjs-big-play-button, button, input, select")) return;
    clearTimeout(clickTimer.current);
    const box = event.currentTarget.getBoundingClientRect();
    seekBy(event.clientX < box.left + box.width / 2 ? -SEEK_SECONDS : SEEK_SECONDS);
  }

  const currentPlaylistIndex = playlist.findIndex(
    (item) => item.id === currentFileId || (!item.id && item.file_path === src)
  );
  const remainingEpisodes =
    currentPlaylistIndex >= 0 ? playlist.slice(currentPlaylistIndex + 1) : playlist;
  const supportsPiP = typeof document !== "undefined" && document.pictureInPictureEnabled;

  return (
    <div
      ref={containerRef}
      tabIndex="0"
      dir="rtl"
      className={`nexora-vjs group relative overflow-hidden bg-black shadow-panel ${visible ? "" : "nexora-vjs--idle"}`}
      onMouseMove={showControls}
      onMouseEnter={showControls}
      onMouseLeave={hideControls}
      onFocus={showControls}
      onClick={onShellClick}
      onDoubleClick={onShellDoubleClick}
    >
      {/* Fullscreen chrome — minimise / exit at the video's top-right, shown only
          while fullscreen. Falls back to plain fullscreen toggling when the host
          page does not pass its own handlers. */}
      {isFullscreen && (
        <div className="nexora-fs-chrome" dir="ltr">
          <button
            type="button"
            className="nexora-fs-btn"
            onClick={() => (onMinimize ? onMinimize() : toggleFullscreen())}
            aria-label="تصغير"
            title="تصغير"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 12h14" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" /></svg>
          </button>
          <button
            type="button"
            className="nexora-fs-btn nexora-fs-btn--exit"
            onClick={() => (onExit ? onExit() : toggleFullscreen())}
            aria-label="خروج"
            title="خروج"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 6l12 12M18 6L6 18" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" /></svg>
          </button>
        </div>
      )}

      {/* NEXORA centre controls — the ±10s jump buttons Video.js does not have.
          The layer itself never swallows pointer events (so clicks still reach
          the video and the bar below); only the buttons do. */}
      <div
        dir="ltr"
        className={`nexora-center pointer-events-none absolute inset-0 z-20 flex-col items-center justify-center gap-4 transition-opacity duration-500 ${visible && !askResume ? "opacity-100" : "opacity-0"}`}
      >
        <div className="pointer-events-auto flex items-center gap-3 sm:gap-5">
          <button
            type="button"
            className="nexora-center-button nexora-center-seek"
            onClick={() => seekBy(-SEEK_SECONDS)}
            aria-label="رجوع 10 ثوانٍ"
          >
            <PlayerIcon name="rewind" />
            <small>10</small>
          </button>
          <button
            type="button"
            className="nexora-center-button nexora-center-main"
            onClick={togglePlay}
            aria-label={playing ? "إيقاف مؤقت" : "تشغيل"}
          >
            <PlayerIcon name={playing ? "pause" : "play"} className="h-8 w-8" />
          </button>
          <button
            type="button"
            className="nexora-center-button nexora-center-seek"
            onClick={() => seekBy(SEEK_SECONDS)}
            aria-label="تقديم 10 ثوانٍ"
          >
            <PlayerIcon name="forward" />
            <small>10</small>
          </button>
        </div>
      </div>

      {/* Resume prompt — same wording and storage contract as the legacy player. */}
      {askResume && (
        <div className="absolute inset-0 z-40 flex items-center justify-center bg-black/55 p-4 backdrop-blur-sm" dir="rtl">
          <div className="rounded-2xl border-white/15 bg-[#151225]/95 p-5 text-center shadow-2xl">
            <p className="text-base font-black text-white">متابعة المشاهدة؟</p>
            <p className="mt-1 text-xs text-white/60">توقفت عند {clock(resumeAt)}</p>
            <div className="mt-4 flex justify-center gap-2">
              <button
                type="button"
                className="rounded-xl bg-fuchsia-600 px-4 py-2 text-xs font-black text-white"
                onClick={() => resolveResume(true)}
              >
                استئناف
              </button>
              <button
                type="button"
                className="rounded-xl bg-white/10 px-4 py-2 text-xs font-bold text-white"
                onClick={() => resolveResume(false)}
              >
                من البداية
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Fullscreen episode queue — YouTube-style thumbnail rail. Appears at the
          bottom of the screen while the control bar stays reachable above it. */}
      {isFullscreen && remainingEpisodes.length > 0 && episodeDrawerOpen && (
        <section
          className="nexora-queue"
          dir="rtl"
          aria-label="الحلقات المتبقية"
        >
          <div className="nexora-queue-head">
            <button
              type="button"
              className="nexora-queue-close"
              onClick={() => setEpisodeDrawerOpen(false)}
              aria-label="إغلاق قائمة الحلقات"
            >
              ×
            </button>
            <span className="nexora-queue-count">{remainingEpisodes.length} متبقية</span>
            <h3 className="nexora-queue-title">الحلقات المتبقية</h3>
          </div>
          <div className="nexora-queue-rail">
            {remainingEpisodes.map((episode, index) => {
              const number = currentPlaylistIndex + index + 2;
              const label =
                episode.title_ar ||
                episode.title_en ||
                (episode.episode_number ? `الحلقة ${episode.episode_number}` : `ملف ${number}`);
              // Real frame from the episode (cached by the server), with the
              // work poster as a fallback if the frame cannot be generated.
              const thumb = episode.id
                ? `/api/stream/file/${episode.id}/preview?at=0`
                : poster;
              return (
                <button
                  key={episode.id || episode.file_path || index}
                  type="button"
                  className="nexora-queue-card"
                  onClick={() => {
                    onSelectFile?.(episode);
                    setEpisodeDrawerOpen(false);
                  }}
                >
                  <span className="nexora-queue-thumb">
                    {thumb ? (
                      <img
                        src={thumb}
                        alt=""
                        loading="lazy"
                        onError={(event) => {
                          // Fall back to the work poster once; if that also fails
                          // (or there is none) hide the image and keep the frame.
                          const img = event.currentTarget;
                          if (poster && !img.dataset.fallback) {
                            img.dataset.fallback = "1";
                            img.src = poster;
                          } else {
                            img.style.display = "none";
                          }
                        }}
                      />
                    ) : null}
                    {episode.duration > 0 && (
                      <span className="nexora-queue-duration">{clock(episode.duration)}</span>
                    )}
                    <span className="nexora-queue-index">{number}</span>
                  </span>
                  <span className="nexora-queue-meta">
                    <b>{label}</b>
                    {title ? <small>{title}</small> : null}
                  </span>
                </button>
              );
            })}
          </div>
        </section>
      )}

      {/* NEXORA control bar — the single control surface. Video.js' own bar is
          disabled (see creation options), so nothing here is duplicated. Reads
          right-to-left: transport on the right, time inline, then volume, rate,
          captions, PiP and fullscreen running to the left edge. */}
      <div
        dir="rtl"
        className={`nexora-bar absolute inset-x-0 bottom-0 z-30 px-3 pb-2 pt-10 transition-opacity duration-500 ${visible && !askResume ? "opacity-100" : "pointer-events-none opacity-0"}`}
      >
        {/* Seek bar with hover thumbnail preview. Pinned LTR like the range
            itself, so the thumbnail tracks the cursor position exactly. */}
        <div
          dir="ltr"
          className="relative mb-1.5"
          onMouseMove={onHoverProgress}
          onMouseLeave={() => {
            setHoverTime(null);
            setPreviewSrc("");
            previewBucket.current = null;
          }}
        >
          {hoverTime !== null && (
            <span
              className="nexora-timeline-preview absolute bottom-5 z-50 flex w-40 flex-col overflow-hidden rounded-lg border-white/20 bg-black/95 shadow-xl"
              style={{ left: `${Math.max(0, Math.min(82, (hoverTime / (duration || 1)) * 100))}%` }}
            >
              {previewSrc && (
                <img
                  src={previewSrc}
                  alt="معاينة المشهد"
                  className="aspect-video w-full object-cover"
                  onError={() => setPreviewSrc("")}
                />
              )}
              <span className="px-2 py-1 text-center text-[11px] font-bold text-white">{clock(hoverTime)}</span>
            </span>
          )}
          <input
            dir="ltr"
            aria-label="شريط تقدم الفيديو"
            type="range"
            min="0"
            max={duration || 0}
            step="0.1"
            value={time}
            onChange={onSeek}
            className="nexora-player-progress w-full"
            style={{ "--player-progress": `${duration ? (time / duration) * 100 : 0}%` }}
          />
        </div>

        {isFullscreen && remainingEpisodes.length > 0 && (
          <div className="nexora-player-episodes-slot">
            <button
              type="button"
              className={`nexora-player-episodes-button ${episodeDrawerOpen ? "is-open" : ""}`}
              aria-expanded={episodeDrawerOpen}
              onClick={() => setEpisodeDrawerOpen((open) => !open)}
            >
              <PlayerIcon name="playlist" className="h-4 w-4" />
              الحلقات المتبقية <span>{remainingEpisodes.length}</span>
            </button>
          </div>
        )}

        <div className="flex items-center gap-1.5">
          <button
            type="button"
            className="nexora-bar-button"
            onClick={togglePlay}
            aria-label={playing ? "إيقاف مؤقت" : "تشغيل"}
          >
            <PlayerIcon name={playing ? "pause" : "play"} className="h-5 w-5" />
          </button>
          <button type="button" className="nexora-bar-button" onClick={() => seekBy(-SEEK_SECONDS)} aria-label="رجوع 10 ثوانٍ">
            <PlayerIcon name="rewind" className="h-5 w-5" />
          </button>
          <button type="button" className="nexora-bar-button" onClick={() => seekBy(SEEK_SECONDS)} aria-label="تقديم 10 ثوانٍ">
            <PlayerIcon name="forward" className="h-5 w-5" />
          </button>
          <span className="nexora-time tabular-nums">
            {clock(time)} <span className="opacity-40">/</span> {clock(duration)}
          </span>

          <div className="ms-auto flex items-center gap-1.5">
            <button type="button" className="nexora-bar-button" onClick={toggleMute} aria-label={muted ? "إلغاء الكتم" : "كتم الصوت"}>
              <PlayerIcon name={muted || volume === 0 ? "mute" : "volume"} className="h-5 w-5" />
            </button>
            <input
              aria-label="الصوت"
              type="range"
              min="0"
              max="1"
              step="0.05"
              value={muted ? 0 : volume}
              onChange={(event) => setVolumeTo(Number(event.target.value))}
              className="nexora-vol"
            />
            {tracks.length > 0 && (
              <button
                type="button"
                className={`nexora-bar-button ${captionIndex >= 0 ? "nexora-bar-button--active" : ""}`}
                onClick={cycleCaptions}
                aria-label="تبديل الترجمة"
              >
                CC
              </button>
            )}
            <select
              aria-label="سرعة التشغيل"
              value={rate}
              onChange={(event) => applyRate(Number(event.target.value))}
              className="nexora-rate"
            >
              {RATES.map((value) => (
                <option key={value} value={value}>{value}×</option>
              ))}
            </select>
            {supportsPiP && (
              <button type="button" className="nexora-bar-button" onClick={togglePiP} aria-label="نافذة مصغرة">
                <PlayerIcon name="pip" className="h-5 w-5" />
              </button>
            )}
            <button type="button" className="nexora-bar-button" onClick={toggleFullscreen} aria-label="ملء الشاشة">
              <PlayerIcon name="fullscreen" className="h-5 w-5" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
