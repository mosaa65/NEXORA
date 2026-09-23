import { useCallback, useEffect, useRef, useState } from "react";

const KEY_PREFIX = "nexora:playback:";
const RESUME_MIN_SECONDS = 30;
const RESUME_TAIL_SECONDS = 30;
const COMPLETED_RATIO = 0.95;
const SAVE_INTERVAL_SECONDS = 5;

export const progressKeyFor = (fileId, src) => `${KEY_PREFIX}${fileId || src}`;

/**
 * readSavedProgress — the stored position for one file, or null.
 *
 * Shared by the player and the episode cards so a card's resume bar and the
 * player's resume prompt can never disagree about what was watched.
 */
export function readSavedProgress(fileId, src) {
  try {
    const raw = localStorage.getItem(progressKeyFor(fileId, src));
    if (!raw) return null;
    const saved = JSON.parse(raw);
    if (!saved || !Number.isFinite(saved.duration) || saved.duration <= 0) return null;
    return saved;
  } catch {
    return null;
  }
}

/**
 * usePlayerProgress — ownership of playback position persistence.
 *
 * Extracted from the player component because it is the one behaviour with a
 * lifecycle of its own: it must write on a timer while playing, on pause, on the
 * tab going away, and on unmount — and never on every `timeupdate`, which would
 * hammer localStorage (and, before this change, the render path) several times a
 * second.
 *
 * @param {object} options
 * @param {string} options.fileId    the `video_files` id being played
 * @param {string} options.src       the stream URL, used when no id exists
 * @param {boolean} options.enabled  false while the source is only a probe
 */
export function usePlayerProgress({ fileId, src, enabled = true }) {
  const lastSavedAt = useRef(0);
  const resumeApplied = useRef(false);
  const [resumeAt, setResumeAt] = useState(0);
  const [askResume, setAskResume] = useState(false);

  const save = useCallback(
    (player, force = false) => {
      if (!enabled || !player || player.isDisposed()) return;
      const total = player.duration() || 0;
      const current = player.currentTime() || 0;
      if (!total) return;
      // Throttle while playing; a forced save (pause/ended/unmount) always writes.
      if (!force && Math.abs(current - lastSavedAt.current) < SAVE_INTERVAL_SECONDS) return;
      lastSavedAt.current = current;
      const completed = current / total >= COMPLETED_RATIO;
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
      } catch {
        // Storage can be full or blocked (private mode). Losing a resume point is
        // acceptable; breaking playback is not.
      }
    },
    [enabled, fileId, src]
  );

  /** Called on loadedmetadata: decide whether to offer a resume for this source. */
  const resolveResumePoint = useCallback(
    (player, { autoResume = false } = {}) => {
      if (resumeApplied.current) return;
      resumeApplied.current = true;
      const total = player.duration() || 0;
      const saved = readSavedProgress(fileId, src);
      if (!saved) return;
      const usable = saved.position > RESUME_MIN_SECONDS && saved.position < total - RESUME_TAIL_SECONDS && !saved.completed;
      if (!usable) return;
      if (autoResume) {
        // Re-mount into the floating dock: continue silently, never re-ask.
        player.currentTime(saved.position);
        return;
      }
      setResumeAt(saved.position);
      setAskResume(true);
    },
    [fileId, src]
  );

  /** Reset per-source state when the source changes. */
  const resetForNewSource = useCallback(() => {
    resumeApplied.current = false;
    lastSavedAt.current = 0;
    setResumeAt(0);
    setAskResume(false);
  }, []);

  const dismissResume = useCallback(() => setAskResume(false), []);

  return { save, resolveResumePoint, resetForNewSource, resumeAt, askResume, dismissResume };
}

/**
 * usePlayerProgressWriters — flush the position when the page or the player goes
 * away. `pagehide` is used alongside `beforeunload` because it is the event that
 * actually fires on mobile Safari, where `beforeunload` is unreliable.
 */
export function usePlayerProgressWriters(playerRef, save) {
  useEffect(() => {
    const flush = () => save(playerRef.current, true);
    const onVisibility = () => {
      if (document.visibilityState === "hidden") flush();
    };
    document.addEventListener("visibilitychange", onVisibility);
    window.addEventListener("pagehide", flush);
    window.addEventListener("beforeunload", flush);
    return () => {
      document.removeEventListener("visibilitychange", onVisibility);
      window.removeEventListener("pagehide", flush);
      window.removeEventListener("beforeunload", flush);
      // Unmount (closing the watch screen or the dock) is the last chance to write.
      flush();
    };
  }, [playerRef, save]);
}