import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import videojs from "video.js";
import "video.js/dist/video-js.css";
import EpisodeCard from "../watch/EpisodeCard.jsx";
import PlayerControls from "./PlayerControls.jsx";
import PlayerCenterControls from "./PlayerCenterControls.jsx";
import PlayerSettingsMenu from "./PlayerSettingsMenu.jsx";
import { PlayerNextOverlay, EndOfWorkNotice } from "./PlayerNextOverlay.jsx";
import PlayerGestureHud from "./PlayerGestureHud.jsx";
import useTouchGestures, { clamp01 } from "./useTouchGestures.js";
import {
  ResumePrompt,
  BufferingIndicator,
  PlayerToast,
  PlayerErrorState,
} from "./PlayerStaticOverlays.jsx";
import { usePlayerProgress, usePlayerProgressWriters } from "./usePlayerProgress.js";
import { clock } from "../../lib/watchContent.js";

/** True on a device whose primary input is a finger. */
const isTouchDevice = () => {
  if (typeof window === "undefined") return false;
  return window.matchMedia?.("(hover: none) and (pointer: coarse)")?.matches ?? false;
};

const SEEK_SECONDS = 10;
const NEXT_PROMPT_SECONDS = 30;
const NEXT_COUNTDOWN_SECONDS = 10;
const CONTROLS_IDLE_MS = 2500;
const SUBTITLE_STYLE_KEY = "nexora:subtitle-style";

const DEFAULT_SUBTITLE_STYLE = { size: "md", background: "soft" };
const SUBTITLE_SIZES = { sm: "0.85rem", md: "1.05rem", lg: "1.3rem", xl: "1.6rem" };

/** Human description of an audio track from FFprobe's persisted track object. */
function describeAudioTrack(track) {
  const parts = [];
  if (track.language) parts.push(track.language);
  if (track.channels) parts.push(track.channels === 1 ? "Mono" : track.channels === 2 ? "Stereo" : `${track.channels}.1`);
  if (track.title) parts.push(track.title);
  const label = parts.length > 0 ? parts.join(" • ") : `مسار ${track.index + 1}`;
  const detail = track.codec ? String(track.codec).toUpperCase() : "";
  return { label, detail };
}

/** Human description of an embedded subtitle track from the persisted array. */
function describeEmbeddedSubtitle(track) {
  const parts = [];
  if (track.language) parts.push(track.language);
  if (track.title) parts.push(track.title);
  return parts.length > 0 ? parts.join(" • ") : `ترجمة ${track.index + 1}`;
}

/**
 * NexoraPlayer — the Video.js host with the NEXORA control surface.
 *
 * Owned here:
 *   - the Video.js lifecycle (created once, source swapped, disposed on unmount);
 *   - the NEXORA keyboard map and the pointer gestures the engine does not ship;
 *   - source selection between the releases of one episode;
 *   - real subtitle and audio-track wiring from the catalogue's own metadata;
 *   - resume, progress persistence and the end-of-episode transition.
 *
 * Deliberately NOT owned here: the timeline's high-frequency painting
 * (`PlayerScrub` writes it straight to the DOM), the settings surface
 * (`PlayerSettingsMenu`), and the surrounding layout (`WatchPage`). The player
 * therefore re-renders on discrete events — play/pause, source change, menu open —
 * and not on every `timeupdate`.
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
  fullscreenTarget,
  onMinimize,
  onExit,
  autoResume = false,
  // Real selectable sources for the current episode (other releases of the same
  // episode). Empty or single-entry means the selector is not offered.
  sources = [],
  onSelectSource,
  // Technical facts persisted by FFprobe for the current file.
  technical = null,
  // Embedded tracks as persisted, used when the browser does not expose them.
  embeddedAudio = [],
  embeddedSubtitles = [],
  hasNext = false,
  hasPrevious = false,
  onPlayPrevious,
  // The catalogue's authoritative next entry (`next` in the playback plan). When
  // present it drives the next-episode card; the playlist is only a fallback.
  nextEpisode = null,
  // "season" | "work" — what to say when there is no next episode.
  endOfLabel = "work",
  onBrowseMore,
  // Compact surface (the floating dock): transport and captions only. The full
  // settings/quality surface needs reading room and belongs to the watch screen.
  compact = false,
}) {
  const containerRef = useRef(null);
  const playerRef = useRef(null);
  const hideTimer = useRef(null);
  const clickTimer = useRef(null);
  const remoteTracks = useRef([]);
  const captionPreference = useRef(-1);

  // High-frequency playback values live outside React. `PlayerScrub` reads them and
  // writes them to the DOM; nothing else subscribes to them.
  const liveStateRef = useRef({ time: 0, duration: 0, buffered: 0 });

  const [ready, setReady] = useState(false);
  const [playing, setPlaying] = useState(false);
  const [duration, setDuration] = useState(0);
  const [matchedSourceId, setMatchedSourceId] = useState(null);
  const [volume, setVolume] = useState(1);
  const [muted, setMuted] = useState(false);
  const [rate, setRate] = useState(1);
  const [visible, setVisible] = useState(true);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [buffering, setBuffering] = useState(false);
  const [captionIndex, setCaptionIndex] = useState(-1);
  const [audioTracks, setAudioTracks] = useState([]);
  const [audioIndex, setAudioIndex] = useState(-1);
  const [menu, setMenu] = useState(null); // null | "root" | "quality" | "audio" | "subtitle" | "rate" | "info"
  const [toast, setToast] = useState(null);
  const [nextCountdown, setNextCountdown] = useState(null);
  const [showNextCard, setShowNextCard] = useState(false);
  const [ended, setEnded] = useState(false);
  const [episodeDrawerOpen, setEpisodeDrawerOpen] = useState(false);
  const [error, setError] = useState(null);

  // Touch-only: a vertical drag on the right half sets volume, on the left half
  // dims the picture (a CSS filter — a page cannot change the panel backlight).
  const [brightness, setBrightness] = useState(1);
  const touchEnabled = useMemo(() => isTouchDevice(), []);
  const [subtitleStyle, setSubtitleStyle] = useState(() => {
    try {
      const saved = JSON.parse(localStorage.getItem(SUBTITLE_STYLE_KEY) || "null");
      if (saved?.size && saved?.background) return saved;
    } catch { }
    return DEFAULT_SUBTITLE_STYLE;
  });

  // Labels the timeline shows. Kept here (not in PlayerScrub) so the heavy child
  // stays prop-stable and only re-renders when a second actually matters.
  const [currentTimeLabel, setCurrentTimeLabel] = useState("0:00");
  const [remainingLabel, setRemainingLabel] = useState("0:00");
  const clockRef = useRef(0);

  const progress = usePlayerProgress({ fileId, src });
  const { save, resolveResumePoint, resetForNewSource, resumeAt, askResume, dismissResume } = progress;
  usePlayerProgressWriters(playerRef, save);

  // ---------------------------------------------------------------------------
  // Controls visibility
  // ---------------------------------------------------------------------------
  const showControls = useCallback(() => {
    setVisible(true);
    clearTimeout(hideTimer.current);
    const player = playerRef.current;
    if (player && !player.paused()) {
      hideTimer.current = setTimeout(() => setVisible(false), CONTROLS_IDLE_MS);
    }
  }, []);

  const hideControls = useCallback(() => {
    clearTimeout(hideTimer.current);
    const player = playerRef.current;
    if (!player || player.paused() || menu) return;
    setVisible(false);
  }, [menu]);

  const showToast = useCallback((message) => {
    setToast(message);
    clearTimeout(showToast._t);
    showToast._t = setTimeout(() => setToast(null), 1400);
  }, []);

  // ---------------------------------------------------------------------------
  // Transport
  // ---------------------------------------------------------------------------
  const seekBy = useCallback((seconds) => {
    const player = playerRef.current;
    if (!player || player.isDisposed()) return;
    const total = player.duration() || 0;
    const next = Math.max(0, Math.min(total, (player.currentTime() || 0) + seconds));
    player.currentTime(next);
  }, []);

  const togglePlay = useCallback(() => {
    const player = playerRef.current;
    if (!player || player.isDisposed()) return;
    if (player.paused()) {
      const attempt = player.play();
      if (attempt && typeof attempt.catch === "function") attempt.catch(() => { });
    } else {
      player.pause();
    }
  }, []);

  // Fullscreen targets the NEXORA shell: Video.js' own requestFullscreen() would
  // fullscreen `.video-js`, hiding every NEXORA layer (the bar, the centre
  // controls, the episode drawer) because they are siblings, not children.
  const toggleFullscreen = useCallback(() => {
    const root = fullscreenTarget?.current || containerRef.current;
    if (!root) return;
    if (document.fullscreenElement) {
      document.exitFullscreen?.().catch(() => { });
    } else {
      root.requestFullscreen?.().catch(() => { });
    }
  }, [fullscreenTarget]);

  const togglePiP = useCallback(() => {
    const player = playerRef.current;
    if (!player || player.isDisposed()) return;
    const el = player.el()?.querySelector?.("video") || player.tech?.()?.el?.();
    if (!el) return;
    if (document.pictureInPictureElement) {
      document.exitPictureInPicture?.().catch(() => { });
    } else {
      el.requestPictureInPicture?.().catch(() => { });
    }
  }, []);

  const setVolumeTo = useCallback((value) => {
    const player = playerRef.current;
    if (!player || player.isDisposed()) return;
    const next = clamp01(value);
    player.volume(next);
    // Raising the volume from a gesture should also lift a mute, otherwise the
    // HUD promises sound the viewer cannot hear.
    player.muted(next === 0);
  }, []);

  // Touch gestures (volume on the right half, brightness on the left half) are
  // attached to the player shell, so the whole video area is the surface.
  const { ref: gestureRef, hud } = useTouchGestures({
    enabled: touchEnabled && !error,
    volume,
    brightness,
    onVolume: setVolumeTo,
    onBrightness: setBrightness,
  });

  const toggleMute = useCallback(() => {
    const player = playerRef.current;
    if (!player || player.isDisposed()) return;
    player.muted(!player.muted());
  }, []);

  /** Advance to the next catalogue item, if one exists. */
  const goNext = useCallback(() => {
    setNextCountdown(null);
    setShowNextCard(false);
    // Guarded the same way as `goPrevious`: no next entry means no step, and the
    // button is disabled — this keeps a keyboard or remote call from going silent.
    if (hasNext) onNext?.();
  }, [hasNext, onNext]);

  /**
   * Previous: step back to the previous episode when the viewer is near the
   * start, otherwise restart the current one — the convention every streaming
   * platform uses, so a stray tap does not lose the episode.
   *
   * `hasPrevious` is checked BEFORE anything else: with no earlier entry there is
   * nothing to step to, and the button is disabled anyway, but a guard here keeps
   * a programmatic call (keyboard, remote) from doing nothing silently.
   */
  const goPrevious = useCallback(() => {
    const player = playerRef.current;
    if (!player || player.isDisposed()) return;
    const atStart = (player.currentTime() || 0) < 10;
    if (!atStart) {
      player.currentTime(0);
      showToast("من البداية");
      return;
    }
    if (hasPrevious) {
      onPlayPrevious?.();
      return;
    }
    player.currentTime(0);
    showToast("من البداية");
  }, [hasPrevious, onPlayPrevious, showToast]);

  // ---------------------------------------------------------------------------
  // Subtitle track selection
  // ---------------------------------------------------------------------------
  const applyCaptionIndex = useCallback((index) => {
    captionPreference.current = index;
    const player = playerRef.current;
    if (player && !player.isDisposed()) {
      const list = player.textTracks();
      if (list) {
        for (let i = 0; i < list.length; i += 1) {
          list[i].mode = i === index ? "showing" : "disabled";
        }
      }
    }
    setCaptionIndex(index);
  }, []);

  const cycleCaptions = useCallback(() => {
    const count = (tracks || []).length;
    if (count === 0) return;
    applyCaptionIndex(captionIndex + 1 >= count ? -1 : captionIndex + 1);
  }, [applyCaptionIndex, captionIndex, tracks]);

  // ---------------------------------------------------------------------------
  // Player creation (once per mounted node)
  // ---------------------------------------------------------------------------
  useEffect(() => {
    if (!containerRef.current || playerRef.current) return;

    const videoElement = document.createElement("video-js");
    videoElement.classList.add("vjs-big-play-centered");
    containerRef.current.appendChild(videoElement);

    playerRef.current = videojs(videoElement, {
      // Video.js' own bar and big-play are off: NEXORA draws the single control
      // surface. Both on duplicated every button and stacked two timelines.
      controls: false,
      bigPlayButton: false,
      controlBar: false,
      preload: "metadata",
      fluid: false,
      fill: true,
      playsinline: true,
      // Hotkeys are off so the NEXORA map cannot double-fire with Video.js', and
      // click is off because the shell owns single click (play/pause) and double
      // click (fullscreen).
      userActions: { hotkeys: false, click: false },
      playbackRates: [0.5, 0.75, 1, 1.25, 1.5, 1.75, 2],
    });
    setReady(true);
    // No disposal here: React StrictMode runs this cleanup between its two
    // development passes, and disposing would destroy a player we are about to
    // reuse. The unmount-only effect below owns the teardown.
  }, []);

  useEffect(() => {
    return () => {
      const player = playerRef.current;
      if (player && !player.isDisposed()) player.dispose();
      playerRef.current = null;
      clearTimeout(hideTimer.current);
      clearTimeout(clickTimer.current);
      clearTimeout(showToast._t);
    };
  }, [showToast]);

  // ---------------------------------------------------------------------------
  // Source switching. Resume is resolved per source, so a next-episode swap never
  // applies the previous episode's position.
  // ---------------------------------------------------------------------------
  useEffect(() => {
    const player = playerRef.current;
    if (!player || player.isDisposed() || !src) return;
    setError(null);
    setEnded(false);
    setShowNextCard(false);
    setNextCountdown(null);
    setEpisodeDrawerOpen(false);
    resetForNewSource();
    player.src({ src, type: "video/mp4" });
    if (poster) player.poster(poster);
    if (title) player.el()?.setAttribute("aria-label", title);
  }, [src, poster, title, ready, resetForNewSource]);

  // ---------------------------------------------------------------------------
  // Video.js events → React state. Every subscription is paired with an `off` in
  // the same cleanup.
  // ---------------------------------------------------------------------------
  useEffect(() => {
    const player = playerRef.current;
    if (!player || !ready) return undefined;

    const onPlay = () => {
      setPlaying(true);
      showControls();
    };
    const onPause = () => {
      setPlaying(false);
      setVisible(true);
      save(player, true);
    };

    const onTimeUpdate = () => {
      const current = player.currentTime() || 0;
      const total = player.duration() || 0;
      liveStateRef.current.time = current;
      liveStateRef.current.duration = total;
      // Persist on the shared throttle; the visible clock is updated once a second
      // so a 60fps timeupdate does not re-render the bar 60 times.
      save(player);
      const whole = Math.floor(current);
      if (whole !== clockRef.current) {
        clockRef.current = whole;
        setCurrentTimeLabel(clock(total ? current : 0));
        setRemainingLabel(`-${clock(Math.max(0, total - current))}`);
      }
      // Pre-empt the ending with the next-episode card.
      if (hasNext && total > 0 && total - current <= NEXT_PROMPT_SECONDS && !showNextCard && !ended) {
        setShowNextCard(true);
      }
    };

    const onProgress = () => {
      try {
        const ranges = player.buffered();
        let end = 0;
        const position = player.currentTime() || 0;
        for (let i = 0; i < ranges.length; i += 1) {
          if (ranges.start(i) <= position + 1) end = Math.max(end, ranges.end(i));
        }
        if (!end && ranges.length) end = ranges.end(ranges.length - 1);
        liveStateRef.current.buffered = end || 0;
      } catch {
        // buffered() throws in some browsers before metadata; the bar simply stays.
      }
    };

    const onLoadedMetadata = () => {
      const total = player.duration() || 0;
      setDuration(total);
      liveStateRef.current.duration = total;
      setRemainingLabel(`-${clock(total)}`);
      resolveResumePoint(player, { autoResume });
    };

    const onVolumeChange = () => {
      setVolume(player.volume() ?? 1);
      setMuted(Boolean(player.muted()));
    };
    const onRateChange = () => setRate(player.playbackRate() || 1);
    const onWaiting = () => setBuffering(true);
    const onPlaying = () => setBuffering(false);
    const onCanPlay = () => setBuffering(false);

    const onEnded = () => {
      setBuffering(false);
      setEnded(true);
      save(player, true);
      if (hasNext) {
        setShowNextCard(true);
        setNextCountdown(NEXT_COUNTDOWN_SECONDS);
      } else {
        setShowNextCard(true);
      }
    };

    const onError = () => {
      setBuffering(false);
      const mediaError = player.error?.();
      // Video.js normalises the element's MediaError; the codes below are the ones
      // a viewer can act on. Anything else falls back to a generic but honest line.
      let message = "تعذر قراءة الملف من المصدر.";
      if (mediaError?.code === 4 || mediaError?.code === 3) {
        message = "المصدر غير متوافق مع المتصفح (الحاوية أو ترميز الفيديو غير مدعوم).";
      } else if (mediaError?.code === 2) {
        message = "تعذر تحميل الفيديو من الخادم. تحقق من الاتصال ثم أعد المحاولة.";
      } else if (mediaError?.code === 1) {
        message = "تم إيقاف التحميل قبل انتهائه.";
      }
      setError({ code: mediaError?.code, message });
    };

    player.on("play", onPlay);
    player.on("pause", onPause);
    player.on("timeupdate", onTimeUpdate);
    player.on("progress", onProgress);
    player.on("loadedmetadata", onLoadedMetadata);
    player.on("durationchange", onLoadedMetadata);
    player.on("volumechange", onVolumeChange);
    player.on("ratechange", onRateChange);
    player.on("waiting", onWaiting);
    player.on("playing", onPlaying);
    player.on("canplay", onCanPlay);
    player.on("ended", onEnded);
    player.on("error", onError);

    // Seed so the strip does not read 0:00 until the first event.
    setDuration(player.duration() || 0);
    setVolume(player.volume() ?? 1);
    setMuted(Boolean(player.muted()));
    setRate(player.playbackRate() || 1);

    return () => {
      player.off("play", onPlay);
      player.off("pause", onPause);
      player.off("timeupdate", onTimeUpdate);
      player.off("progress", onProgress);
      player.off("loadedmetadata", onLoadedMetadata);
      player.off("durationchange", onLoadedMetadata);
      player.off("volumechange", onVolumeChange);
      player.off("ratechange", onRateChange);
      player.off("waiting", onWaiting);
      player.off("playing", onPlaying);
      player.off("canplay", onCanPlay);
      player.off("ended", onEnded);
      player.off("error", onError);
    };
  }, [ready, autoResume, hasNext, ended, showNextCard, save, showControls, resolveResumePoint]);

  // ---------------------------------------------------------------------------
  // Audio tracks — the real list, when the browser publishes one.
  // ---------------------------------------------------------------------------
  useEffect(() => {
    const player = playerRef.current;
    if (!player || !ready) return undefined;

    const list = player.audioTracks?.();
    if (!list) {
      setAudioTracks([]);
      setAudioIndex(-1);
      return undefined;
    }

    const read = () => {
      const next = [];
      let enabled = -1;
      for (let i = 0; i < list.length; i += 1) {
        const track = list[i];
        next.push({
          index: i,
          label: track.label || track.language || `مسار ${i + 1}`,
          detail: track.language && track.label ? track.language : "",
          selectable: true,
        });
        if (track.enabled) enabled = i;
      }
      setAudioTracks(next);
      setAudioIndex(enabled);
    };

    read();
    list.addEventListener?.("addtrack", read);
    list.addEventListener?.("removetrack", read);
    list.addEventListener?.("change", read);
    return () => {
      list.removeEventListener?.("addtrack", read);
      list.removeEventListener?.("removetrack", read);
      list.removeEventListener?.("change", read);
    };
  }, [ready, src]);

  // ---------------------------------------------------------------------------
  // Subtitle wiring: translate the NEXORA `tracks` prop into Video.js text tracks.
  // The previous file's tracks are removed first, otherwise the captions menu keeps
  // every episode's subtitles and a switch lands on a dead source.
  // ---------------------------------------------------------------------------
  useEffect(() => {
    const player = playerRef.current;
    if (!player || !ready || player.isDisposed()) return;

    remoteTracks.current.forEach((element) => {
      try {
        player.removeRemoteTextTrack(element);
      } catch { }
    });
    remoteTracks.current = [];

    (tracks || [])
      .filter((track) => track?.src)
      .forEach((track) => {
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

    // Re-apply the viewer's previous choice when that same label still exists.
    const list = player.textTracks();
    let restored = -1;
    if (list && captionPreference.current >= 0 && captionPreference.current < list.length) {
      restored = captionPreference.current;
    }
    applyCaptionIndex(restored);
  }, [tracks, ready, fileId, src, applyCaptionIndex]);

  // ---------------------------------------------------------------------------
  // Fullscreen state follows the DOM.
  // ---------------------------------------------------------------------------
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

  // ---------------------------------------------------------------------------
  // Keyboard map, scoped to the player element rather than `document`, so it never
  // steals keys from the page around it.
  // ---------------------------------------------------------------------------
  useEffect(() => {
    const root = containerRef.current;
    if (!root) return undefined;

    const onKeyDown = (event) => {
      const tag = event.target?.tagName;
      if (["INPUT", "TEXTAREA", "SELECT"].includes(tag) && event.target.type !== "range") return;
      const key = event.key === " " ? " " : event.key.toLowerCase();
      const handled = [" ", "k", "arrowleft", "arrowright", "j", "l", "arrowup", "arrowdown", "m", "f", "c", "n"];
      if (!handled.includes(key)) return;
      event.preventDefault();
      if (key === " " || key === "k") togglePlay();
      else if (key === "arrowleft" || key === "j") seekBy(-SEEK_SECONDS);
      else if (key === "arrowright" || key === "l") seekBy(SEEK_SECONDS);
      else if (key === "arrowup") setVolumeTo(Math.min(1, (playerRef.current?.volume() ?? 1) + 0.1));
      else if (key === "arrowdown") setVolumeTo(Math.max(0, (playerRef.current?.volume() ?? 1) - 0.1));
      else if (key === "m") toggleMute();
      else if (key === "f") toggleFullscreen();
      // On the compact dock the settings surface is not rendered, so the caption
      // key cycles directly instead of opening a menu that does not exist.
      else if (key === "c") cycleCaptions();
      else if (key === "n" && hasNext) goNext();
      showControls();
    };

    root.addEventListener("keydown", onKeyDown);
    return () => root.removeEventListener("keydown", onKeyDown);
  }, [
    cycleCaptions,
    goNext,
    hasNext,
    seekBy,
    setVolumeTo,
    showControls,
    toggleFullscreen,
    toggleMute,
    togglePlay,
  ]);

  // ---------------------------------------------------------------------------
  // Next-episode countdown.
  // ---------------------------------------------------------------------------
  useEffect(() => {
    if (nextCountdown === null) return undefined;
    if (nextCountdown <= 0) {
      setNextCountdown(null);
      goNext();
      return undefined;
    }
    const timer = setTimeout(() => setNextCountdown((n) => (n === null ? null : n - 1)), 1000);
    return () => clearTimeout(timer);
  }, [nextCountdown, goNext]);

  // ---------------------------------------------------------------------------
  // Subtitle styling → CSS custom properties on the player root, so Video.js' cue
  // layer picks them up without us touching its DOM.
  // ---------------------------------------------------------------------------
  useEffect(() => {
    const root = containerRef.current;
    if (!root) return;
    root.style.setProperty("--nx-caption-size", SUBTITLE_SIZES[subtitleStyle.size] || SUBTITLE_SIZES.md);
    const backgrounds = {
      none: "transparent",
      soft: "rgba(0,0,0,.62)",
      solid: "rgba(0,0,0,.95)",
    };
    root.style.setProperty("--nx-caption-bg", backgrounds[subtitleStyle.background] || backgrounds.soft);
    try {
      localStorage.setItem(SUBTITLE_STYLE_KEY, JSON.stringify(subtitleStyle));
    } catch { }
  }, [subtitleStyle]);

  // ---------------------------------------------------------------------------
  // Pointer gestures. Single click toggles play (debounced 220ms); double click
  // toggles fullscreen, matching the YouTube convention.
  // ---------------------------------------------------------------------------
  function onShellClick(event) {
    if (askResume || error) return;
    if (event.target.closest?.("button, input, select, .nexora-bar, .nexora-center, .nexora-next, .nexora-popover")) return;
    clearTimeout(clickTimer.current);
    clickTimer.current = setTimeout(() => togglePlay(), 220);
  }

  function onShellDoubleClick(event) {
    if (askResume || error) return;
    if (event.target.closest?.("button, input, select, .nexora-bar, .nexora-center, .nexora-next, .nexora-popover")) return;
    clearTimeout(clickTimer.current);
    toggleFullscreen();
  }

  function resolveResume(useResume) {
    const player = playerRef.current;
    if (!player || player.isDisposed()) return;
    player.currentTime(useResume ? resumeAt : 0);
    dismissResume();
    if (useResume) {
      const attempt = player.play();
      if (attempt && typeof attempt.catch === "function") attempt.catch(() => { });
    }
  }

  const retry = useCallback(() => {
    const player = playerRef.current;
    setError(null);
    if (!player || player.isDisposed()) return;
    player.src({ src, type: "video/mp4" });
    const attempt = player.play();
    if (attempt && typeof attempt.catch === "function") attempt.catch(() => { });
  }, [src]);

  // ---------------------------------------------------------------------------
  // Derived menu data
  // ---------------------------------------------------------------------------
  const openSubtitleTracks = useMemo(
    () =>
      (tracks || []).map((track) => ({
        label: track.label || track.srcLang || "ترجمة",
        source: track.srcLang ? String(track.srcLang).toUpperCase() : "",
      })),
    [tracks]
  );

  const audioOptions = useMemo(() => {
    if (audioTracks.length > 0) {
      return audioTracks.map((track) => ({
        index: track.index,
        label: track.label,
        detail: track.detail,
        selectable: true,
      }));
    }
    // The browser did not publish tracks: show what FFprobe found, read-only, so
    // the panel describes the file without pretending to switch it.
    return (embeddedAudio || []).map((track, index) => {
      const described = describeAudioTrack(track);
      return { index, label: described.label || describeEmbeddedSubtitle(track), detail: described.detail, selectable: false };
    });
  }, [audioTracks, embeddedAudio]);

  const technicalInfo = useMemo(() => {
    const containerSource = technical?.container || "";
    return {
      resolution: technical?.resolution || "",
      video_codec: technical?.video_codec || "",
      container: containerSource || "",
      durationLabel: duration ? clock(duration) : "",
      file_size: technical?.file_size || 0,
      audioCount: technical?.audio_track_count ?? (embeddedAudio?.length || 0),
      subtitleCount: technical?.subtitle_count ?? (embeddedSubtitles?.length || 0),
      audioSummary: audioOptions.length > 0 ? audioOptions.map((a) => a.label).join(" · ") : "",
    };
  }, [technical, duration, audioOptions, embeddedAudio, embeddedSubtitles]);

  const qualityOptions = useMemo(() => {
    const list = (sources || []).map((source) => ({
      video_file_id: source.video_file_id ?? source.id,
      label: source.label || source.resolution || "إصدار",
      resolution: source.resolution,
      file_size: source.file_size,
    }));
    return { sources: list };
  }, [sources]);

  const settingsOptions = useMemo(
    () => ({
      sources: qualityOptions.sources,
      audio: audioOptions,
      audioUnsupported: audioTracks.length === 0 && (embeddedAudio || []).length > 1,
      subtitle: openSubtitleTracks,
      technical: technicalInfo,
    }),
    [qualityOptions, audioOptions, audioTracks.length, embeddedAudio, openSubtitleTracks, technicalInfo]
  );

  const currentSelection = useMemo(
    () => ({
      qualityId: matchedSourceId,
      quality: sources?.find((s) => (s.video_file_id ?? s.id) === matchedSourceId)?.label || technicalInfo.resolution || "",
      audioIndex,
      audio: audioIndex >= 0 ? audioOptions[audioIndex]?.label || "" : "",
      subtitleIndex: captionIndex,
      subtitle: captionIndex >= 0 ? openSubtitleTracks[captionIndex]?.label || "" : "",
      rate,
    }),
    [matchedSourceId, sources, technicalInfo.resolution, audioIndex, audioOptions, captionIndex, openSubtitleTracks, rate]
  );

  const applySetting = useCallback(
    (kind, value) => {
      if (kind === "quality") {
        setMatchedSourceId(value.video_file_id);
        setMenu(null);
        onSelectSource?.(value);
      } else if (kind === "audio") {
        const player = playerRef.current;
        const list = player && !player.isDisposed() ? player.audioTracks?.() : null;
        if (list && list[value.index]) {
          for (let i = 0; i < list.length; i += 1) list[i].enabled = i === value.index;
        }
        setAudioIndex(value.index);
        setMenu(null);
      } else if (kind === "subtitle") {
        applyCaptionIndex(value);
        setMenu(null);
      } else if (kind === "rate") {
        const player = playerRef.current;
        if (player && !player.isDisposed()) player.playbackRate(value);
        setMenu(null);
      }
    },
    [applyCaptionIndex, onSelectSource]
  );

  const supportsPiP = typeof document !== "undefined" && Boolean(document.pictureInPictureEnabled);

  // Which release is playing, for the settings summary.
  useEffect(() => {
    if (!fileId) {
      setMatchedSourceId(null);
      return;
    }
    setMatchedSourceId(Number(fileId));
  }, [fileId]);

  // The playlist is the EPISODE list, so its identity is `id` (episode) while the
  // player holds a `video_file_id`. Matching on the wrong key silently returned -1,
  // which made "remaining episodes" the whole list and pointed the next-episode
  // card at episode 1. Both keys are checked, and a direct `file_path` match still
  // works for a caller that passes raw file rows.
  const currentPlaylistIndex = playlist.findIndex(
    (item) =>
      (item.video_file_id && String(item.video_file_id) === String(currentFileId)) ||
      (item.file_id && String(item.file_id) === String(currentFileId)) ||
      (item.id && !item.video_file_id && !item.file_id && String(item.id) === String(currentFileId)) ||
      (!item.id && !item.video_file_id && item.file_path === src)
  );
  const remainingEpisodes = currentPlaylistIndex >= 0 ? playlist.slice(currentPlaylistIndex + 1) : [];

  // The next item to play comes from the catalogue (the `next` field of the playback
  // plan) when the caller supplies it, because that ordering is authoritative. The
  // playlist is the fallback for callers that only pass a list.
  const nextItem = nextEpisode || (currentPlaylistIndex >= 0 ? playlist[currentPlaylistIndex + 1] : null) || null;

  const overlayVisible = visible && !askResume && !error;

  // The gesture surface is the shell itself, so the ref is attached to the root
  // element and the brightness filter rides on the same node.
  const shellStyle = brightness < 1 ? { filter: `brightness(${brightness})` } : undefined;

  return (
    <div
      ref={(node) => {
        containerRef.current = node;
        gestureRef.current = node;
      }}
      style={shellStyle}
      tabIndex={0}
      dir="rtl"
      role="region"
      aria-label={title || "مشغل الفيديو"}
      className={`nexora-vjs group relative aspect-video overflow-hidden bg-black ${visible ? "" : "nexora-vjs--idle"} ${touchEnabled ? "nexora-vjs--touch" : ""}`}
      onMouseMove={showControls}
      onMouseEnter={showControls}
      onMouseLeave={hideControls}
      onFocus={showControls}
      onClick={onShellClick}
      onDoubleClick={onShellDoubleClick}
      onTouchStart={showControls}
    >
      {error ? (
        <PlayerErrorState error={error} technical={technicalInfo} onRetry={retry} onBack={onExit} />
      ) : (
        <>
          <PlayerCenterControls
            visible={overlayVisible && !ended}
            playing={playing}
            onTogglePlay={togglePlay}
            onSeekBy={seekBy}
            onToast={showToast}
            // The ±10s rings need a known length to seek through; before the
            // player reports one there is nothing to jump across.
            seekable={Number.isFinite(duration) && duration > 0}
          />

          {askResume && (
            <ResumePrompt resumeAt={resumeAt} onResume={() => resolveResume(true)} onRestart={() => resolveResume(false)} />
          )}

          {/* Fullscreen episode queue — a grid of the same cards the watch screen
              uses, so the list looks like a library rather than a debug list. */}
          {isFullscreen && remainingEpisodes.length > 0 && episodeDrawerOpen && (
            <section className="nexora-queue" dir="rtl" aria-label="الحلقات المتبقية">
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
              <div className="nexora-queue-grid">
                {remainingEpisodes.map((episode, index) => (
                  <EpisodeCard
                    key={episode.id || episode.file_path || index}
                    episode={episode}
                    index={currentPlaylistIndex + index + 1}
                    type="series"
                    onPlay={(ep) => {
                      onSelectFile?.(ep);
                      setEpisodeDrawerOpen(false);
                    }}
                  />
                ))}
              </div>
            </section>
          )}

          {/* Next-episode card / honest ending, below the controls. */}
          {!ended && showNextCard && hasNext && (
            <PlayerNextOverlay
              next={nextItem}
              countdown={nextCountdown}
              onPlayNow={goNext}
              onCancel={() => {
                setNextCountdown(null);
                setShowNextCard(false);
              }}
            />
          )}
          {ended && !hasNext && <EndOfWorkNotice kind={endOfLabel} onBack={onBrowseMore} />}

          <BufferingIndicator visible={buffering} />
          <PlayerToast message={toast} />
          <PlayerGestureHud hud={hud} />

          {overlayVisible && (
            <PlayerControls
              playerRef={playerRef}
              liveStateRef={liveStateRef}
              playing={playing}
              duration={duration}
              muted={muted}
              volume={volume}
              isFullscreen={isFullscreen}
              supportsPiP={supportsPiP}
              captionsAvailable={(tracks || []).length}
              activeCaptionLabel={captionIndex >= 0}
              episodeListAvailable={remainingEpisodes.length}
              episodeListOpen={episodeDrawerOpen}
              onToggleEpisodeList={() => setEpisodeDrawerOpen((open) => !open)}
              onTogglePlay={togglePlay}
              onSeekBy={seekBy}
              onSeekStart={(current, total) => {
                liveStateRef.current.time = current;
                liveStateRef.current.duration = total;
              }}
              onSeek={(value) => {
                const total = liveStateRef.current.duration || duration;
                setCurrentTimeLabel(clock(value));
                setRemainingLabel(`-${clock(Math.max(0, total - value))}`);
              }}
              onToggleMute={toggleMute}
              onSetVolume={setVolumeTo}
              onToggleFullscreen={toggleFullscreen}
              onTogglePiP={togglePiP}
              onOpenSettings={compact ? null : setMenu}
              onExit={onExit || toggleFullscreen}
              onMinimize={onMinimize || toggleFullscreen}
              fileId={fileId}
              hasNext={hasNext}
              hasPrevious={hasPrevious}
              onNext={goNext}
              onPrevious={goPrevious}
              menuOpen={compact ? null : menu}
              compact={compact}
              currentTimeLabel={currentTimeLabel}
              remainingLabel={remainingLabel}
              settingsMenu={
                compact ? null : (
                  <PlayerSettingsMenu
                    section={menu}
                    onClose={() => setMenu(null)}
                    onSelectSection={setMenu}
                    options={settingsOptions}
                    current={currentSelection}
                    onApply={applySetting}
                    subtitleStyle={subtitleStyle}
                    onSubtitleStyle={setSubtitleStyle}
                  />
                )
              }
            />
          )}
        </>
      )}
    </div>
  );
}