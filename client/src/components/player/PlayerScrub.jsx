import { memo, useCallback, useEffect, useRef, useState } from "react";

const PREVIEW_BUCKET_SECONDS = 10;
const PREVIEW_DEBOUNCE_MS = 180;

const clock = (value = 0) => {
  const seconds = Math.max(0, Math.floor(Number.isFinite(value) ? value : 0));
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return hours > 0
    ? `${hours}:${String(minutes).padStart(2, "0")}:${String(seconds % 60).padStart(2, "0")}`
    : `${minutes}:${String(seconds % 60).padStart(2, "0")}`;
};

/**
 * PlayerScrub — the timeline.
 *
 * This is the one surface that would otherwise re-render the whole player on every
 * `timeupdate`. It subscribes to the raw media events itself, writes the played and
 * buffered widths straight into CSS custom properties on its own DOM nodes, and
 * keeps React state only for the things a human actually perceives changing:
 * the hover position and its preview frame.
 *
 * The component is memoised and takes only stable props, so a re-render of the
 * player (a settings toggle, a caption switch) does not touch the timeline.
 */
function PlayerScrub({
  playerRef,
  liveStateRef,
  duration,
  onSeekStart,
  onSeek,
  fileId,
  currentTimeLabel,
  remainingLabel,
  disabled = false,
}) {
  const trackRef = useRef(null);
  const playedRef = useRef(null);
  const loadedRef = useRef(null);
  const inputRef = useRef(null);
  const previewTimer = useRef(null);
  const previewBucket = useRef(null);
  // The parent passes these inline, so a dependency on them would tear down and
  // rebuild the media subscriptions on every parent render — the exact churn this
  // component exists to prevent. They are read through a ref instead.
  const callbacksRef = useRef({ onSeekStart, onSeek });
  callbacksRef.current = { onSeekStart, onSeek };

  const [hoverTime, setHoverTime] = useState(null);
  const [previewSrc, setPreviewSrc] = useState("");
  const [seeking, setSeeking] = useState(false);

  // --- Live position painting -------------------------------------------------
  // A single rAF-coalesced writer: `timeupdate` can fire faster than the browser
  // paints, so intermediate values are dropped instead of forcing layout each time.
  useEffect(() => {
    const player = playerRef.current;
    const track = trackRef.current;
    if (!player || !track || player.isDisposed()) return undefined;

    let frame = 0;
    const paint = () => {
      frame = 0;
      const total = liveStateRef.current.duration || player.duration() || 0;
      const current = player.currentTime() || 0;
      const buffered = liveStateRef.current.buffered || 0;
      liveStateRef.current.time = current;
      const playedPercent = total ? (current / total) * 100 : 0;
      const loadedPercent = total ? (Math.max(buffered, current) / total) * 100 : 0;
      track.style.setProperty("--played", `${playedPercent}%`);
      track.style.setProperty("--loaded", `${loadedPercent}%`);
      if (playedRef.current) playedRef.current.style.width = `${playedPercent}%`;
      if (loadedRef.current) loadedRef.current.style.width = `${loadedPercent}%`;
      // The native input drives keyboard seeking and screen readers, so it has to
      // track the real position — but only while the user is not dragging it.
      if (inputRef.current && !inputRef.current.matches(":active")) {
        inputRef.current.value = String(current);
      }
      callbacksRef.current.onSeekStart?.(current, total);
    };
    const schedule = () => {
      if (frame) return;
      frame = requestAnimationFrame(paint);
    };

    player.on("timeupdate", schedule);
    player.on("progress", schedule);
    player.on("seeking", schedule);
    player.on("seeked", schedule);
    schedule();
    return () => {
      if (frame) cancelAnimationFrame(frame);
      player.off("timeupdate", schedule);
      player.off("progress", schedule);
      player.off("seeking", schedule);
      player.off("seeked", schedule);
    };
  }, [liveStateRef, playerRef]);

  // --- Hover preview ----------------------------------------------------------
  const handleHover = useCallback(
    (event) => {
      if (disabled) return;
      const box = event.currentTarget.getBoundingClientRect();
      if (!box.width) return;
      const total = liveStateRef.current.duration || duration || 0;
      const next = total
        ? Math.max(0, Math.min(total, ((event.clientX - box.left) / box.width) * total))
        : null;
      setHoverTime(next);
      if (!fileId || next === null) return;
      const bucket = Math.floor(next / PREVIEW_BUCKET_SECONDS) * PREVIEW_BUCKET_SECONDS;
      if (previewBucket.current === bucket) return;
      previewBucket.current = bucket;
      clearTimeout(previewTimer.current);
      previewTimer.current = setTimeout(
        () => setPreviewSrc(`/api/stream/file/${fileId}/preview?at=${bucket}`),
        PREVIEW_DEBOUNCE_MS
      );
    },
    [disabled, duration, fileId, liveStateRef]
  );

  const clearHover = useCallback(() => {
    setHoverTime(null);
    setPreviewSrc("");
    previewBucket.current = null;
    clearTimeout(previewTimer.current);
  }, []);

  useEffect(() => () => clearTimeout(previewTimer.current), []);

  // Reset hover state when the source changes (new episode, new file).
  useEffect(() => {
    clearHover();
  }, [fileId, clearHover]);

  const commitSeek = useCallback(
    (value) => {
      const player = playerRef.current;
      if (!player) return;
      const next = Number(value);
      player.currentTime(next);
      liveStateRef.current.time = next;
      callbacksRef.current.onSeek?.(next);
    },
    [liveStateRef, playerRef]
  );

  const hoverPercent = hoverTime !== null && duration ? Math.min(100, (hoverTime / duration) * 100) : 0;
  // The preview is clamped so its own width never pushes it off the bar's edge.
  const previewLeft = Math.max(0, Math.min(82, hoverPercent));

  return (
    <div
      dir="ltr"
      className="nexora-scrub-host relative mb-1.5"
      onMouseMove={handleHover}
      onMouseEnter={handleHover}
      onMouseLeave={clearHover}
      onTouchStart={() => setSeeking(true)}
      onTouchEnd={() => setSeeking(false)}
    >
      {hoverTime !== null && (
        <span
          className="nexora-timeline-preview absolute bottom-6 z-50 flex w-40 flex-col overflow-hidden rounded-lg border-white/20 bg-black/95 shadow-xl"
          style={{ left: `${previewLeft}%` }}
        >
          <span className="nexora-preview-frame">
            {previewSrc ? (
              <img
                src={previewSrc}
                alt=""
                className="aspect-video w-full object-cover"
                onError={() => setPreviewSrc("")}
              />
            ) : (
              <span className="nexora-preview-loading" aria-hidden="true" />
            )}
          </span>
          <span className="px-2 py-1 text-center text-[11px] font-bold text-white">{clock(hoverTime)}</span>
        </span>
      )}

      <div ref={trackRef} className="nexora-scrub">
        <span className="nexora-scrub-track" aria-hidden="true">
          <span ref={loadedRef} className="nexora-scrub-loaded" />
          <span ref={playedRef} className="nexora-scrub-played" />
        </span>
        <input
          ref={inputRef}
          dir="ltr"
          aria-label="شريط تقدم الفيديو"
          aria-valuetext={`${currentTimeLabel} من ${currentTimeLabel ? remainingLabel : ""}`.trim()}
          type="range"
          min="0"
          max={duration || 0}
          step="0.1"
          defaultValue={0}
          onChange={(event) => commitSeek(event.target.value)}
          disabled={disabled}
          className="nexora-scrub-input"
        />
        {hoverTime !== null && (
          <span className="nexora-timeline-cursor" style={{ left: `${hoverPercent}%` }} aria-hidden="true" />
        )}
      </div>

      {/* Time readout lives with the bar so the parent does not re-render for it. */}
      <div className="mt-1 flex items-center justify-between px-0.5 text-[11px] font-bold text-white/70 nexora-time-row">
        <span className="nexora-time">{currentTimeLabel}</span>
        <span className={`nexora-time ${seeking ? "is-seeking" : ""}`}>{remainingLabel}</span>
      </div>
    </div>
  );
}

export default memo(PlayerScrub);