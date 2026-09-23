import { useEffect, useRef, useState } from "react";

/**
 * useTouchGestures — phone volume/brightness gestures over the video surface.
 *
 * Every streaming app on a phone lets a viewer swipe to change volume and
 * brightness, because the device's own controls are out of reach in fullscreen.
 * This hook reproduces that: a vertical drag on the RIGHT half drives volume, a
 * vertical drag on the LEFT half drives brightness, and a tap (no movement) is
 * left alone so it still toggles play.
 *
 * Design decisions:
 *
 * - **Brightness is a CSS filter on the player element**, not a real backlight
 *   change. A web page cannot dim the panel; dimming the content is the only
 *   honest thing it can do, and it is what every web player ships.
 * - **The gesture only claims vertical intent.** A drag must first clear a small
 *   vertical threshold with the horizontal axis smaller than the vertical one, so
 *   horizontal scrubbing across the timeline can be added later without the two
 *   fighting. Until it clears that test, the touch is untouched and taps, long
 *   presses and future horizontal gestures all keep working.
 * - **Volume is clamped to the player.** The hook reports a target value; the
 *   caller applies it through the player API.
 * - **No `preventDefault` unless the gesture is claimed**, which keeps passive
 *   scrolling and click-to-pause intact.
 *
 * The element must opt into `touch-action: pan-y` (or `none`) for vertical
 * gestures to be observed without the browser scrolling the page first.
 *
 * @param {object}   options
 * @param {boolean}  options.enabled     only on touch devices with a live player
 * @param {number}   options.volume      0..1, current volume
 * @param {number}   options.brightness  0..1, current brightness override
 * @param {(v:number)=>void} options.onVolume
 * @param {(b:number)=>void} options.onBrightness
 * @param {(label:string)=>void} [options.onFeedback] called with a HUD label
 */
const CLAIM_PX = 14; // vertical distance before the gesture is claimed
const FULL_SWEEP_PX = 220; // drag distance that covers the whole 0..1 range

export function clamp01(value) {
  if (!Number.isFinite(value)) return 0;
  return Math.min(1, Math.max(0, value));
}

export default function useTouchGestures({
  enabled,
  volume = 1,
  brightness = 1,
  onVolume,
  onBrightness,
  onFeedback,
}) {
  const ref = useRef(null);
  const state = useRef(null);
  const [hud, setHud] = useState(null);

  // Keep the newest handlers/values reachable from the listeners without
  // re-binding them on every render (a re-bind mid-gesture would drop it).
  const latest = useRef({ enabled, volume, brightness, onVolume, onBrightness, onFeedback });
  latest.current = { enabled, volume, brightness, onVolume, onBrightness, onFeedback };

  useEffect(() => {
    const el = ref.current;
    if (!el) return undefined;

    const showHud = (kind, value) => {
      const percent = Math.round(value * 100);
      setHud({ kind, percent });
      latest.current.onFeedback?.(`${kind === "volume" ? "الصوت" : "السطوع"} ${percent}%`);
    };

    const onStart = (event) => {
      const { enabled: on } = latest.current;
      if (!on || event.touches.length !== 1) return;
      const touch = event.touches[0];
      const box = el.getBoundingClientRect();
      // Right half in RTL still means the physically right side of the screen.
      const onRightHalf = touch.clientX - box.left > box.width / 2;
      state.current = {
        startX: touch.clientX,
        startY: touch.clientY,
        kind: onRightHalf ? "volume" : "brightness",
        startValue: onRightHalf ? latest.current.volume : latest.current.brightness,
        claimed: false,
      };
    };

    const onMove = (event) => {
      const s = state.current;
      if (!s || !latest.current.enabled || event.touches.length !== 1) return;
      const touch = event.touches[0];
      const dx = touch.clientX - s.startX;
      const dy = touch.clientY - s.startY;

      if (!s.claimed) {
        const vertical = Math.abs(dy) > Math.abs(dx);
        if (Math.abs(dy) < CLAIM_PX || !vertical) return;
        s.claimed = true;
      }

      // Swiping up increases; down decreases. `dy` is positive downwards.
      const delta = -dy / FULL_SWEEP_PX;
      const next = clamp01(s.startValue + delta);

      if (s.kind === "volume") {
        latest.current.onVolume?.(next);
        showHud("volume", next);
      } else {
        latest.current.onBrightness?.(next);
        showHud("brightness", next);
      }

      // Only now does the page stop treating this as a scroll.
      if (event.cancelable) event.preventDefault();
    };

    const onEnd = () => {
      state.current = null;
      // The HUD fades itself out; clear it shortly after the finger lifts.
      window.setTimeout(() => setHud(null), 550);
    };

    el.addEventListener("touchstart", onStart, { passive: true });
    el.addEventListener("touchmove", onMove, { passive: false });
    el.addEventListener("touchend", onEnd, { passive: true });
    el.addEventListener("touchcancel", onEnd, { passive: true });

    return () => {
      el.removeEventListener("touchstart", onStart);
      el.removeEventListener("touchmove", onMove);
      el.removeEventListener("touchend", onEnd);
      el.removeEventListener("touchcancel", onEnd);
    };
  }, []);

  return { ref, hud };
}
