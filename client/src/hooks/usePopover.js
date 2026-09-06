import { useLayoutEffect, useRef, useState } from "react";

/**
 * Keeps a popover anchored to its trigger and fully on-screen.
 *
 * The popover is rendered as `position: fixed` at coordinates derived from the
 * trigger's bounding rect, re-measured on resize/scroll so it never clips on
 * narrow screens or RTL layouts. If there isn't enough space below, it flips
 * above the trigger.
 *
 * @param {Object} opts
 * @param {boolean} opts.open Whether the popover is visible.
 * @param {number} [opts.offset=8] Vertical gap between trigger and popover.
 * @param {number} [opts.gap=10] Minimum distance to the viewport edges.
 * @returns {{ triggerRef: import("react").RefObject, popoverRef: import("react").RefObject, style: React.CSSProperties }}
 */
export default function usePopover({ open, offset = 8, gap = 10 }) {
  const triggerRef = useRef(null);
  const popoverRef = useRef(null);
  const [style, setStyle] = useState(null);

  useLayoutEffect(() => {
    if (!open) {
      setStyle(null);
      return;
    }

    const update = () => {
      const trigger = triggerRef.current;
      const popover = popoverRef.current;
      if (!trigger || !popover) return;

      const triggerRect = trigger.getBoundingClientRect();
      const width = popover.offsetWidth || triggerRect.width;
      const height = popover.offsetHeight || 0;
      const vw = window.innerWidth || document.documentElement.clientWidth;
      const vh = window.innerHeight || document.documentElement.clientHeight;

      // Anchor the popover's start edge with the trigger's start edge, then
      // clamp so it always stays inside the horizontal viewport.
      let left = triggerRect.left;
      left = Math.max(gap, Math.min(left, vw - width - gap));

      let top = triggerRect.bottom + offset;
      if (top + height > vh - gap) {
        top = Math.max(gap, triggerRect.top - height - offset);
      }

      setStyle({ position: "fixed", left, top, zIndex: 60, maxWidth: `calc(100vw - ${gap * 2}px)` });
    };

    update();

    window.addEventListener("resize", update);
    window.addEventListener("scroll", update, true);
    return () => {
      window.removeEventListener("resize", update);
      window.removeEventListener("scroll", update, true);
    };
  }, [open, offset, gap]);

  return { triggerRef, popoverRef, style };
}