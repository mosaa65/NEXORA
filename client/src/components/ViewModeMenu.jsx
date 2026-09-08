import { useEffect, useState } from "react";
import Icon from "./Icon";
import CompactActionButton from "./CompactActionButton";
import usePopover from "../hooks/usePopover.js";

const VIEW_OPTIONS = [
  { id: "extralarge", label: "أيقونات كبيرة جداً", icon: "iconExtralarge" },
  { id: "large", label: "أيقونات كبيرة", icon: "iconLarge" },
  { id: "medium", label: "أيقونات متوسطة", icon: "iconMedium" },
  { id: "list", label: "قائمة", icon: "iconList" },
  { id: "details", label: "تفاصيل", icon: "iconDetails" },
];

/**
 * Windows 11-style "العرض كيف ما تشتي" (View Mode) dropdown.
 *
 * Renders the compact toolbar button that, when clicked, opens a popover menu
 * of view options. It closes on outside click, Escape, and item select. The
 * popover is positioned with `usePopover` so it stays on-screen in RTL and on
 * narrow viewports.
 *
 * @param {Object} props
 * @param {string} props.mode Current view mode id.
 * @param {(mode: string) => void} props.onSelect
 */
export default function ViewModeMenu({ mode, onSelect }) {
  const [open, setOpen] = useState(false);
  const { triggerRef, popoverRef, style } = usePopover({ open });

  useEffect(() => {
    if (!open) return;
    function onDocClick(event) {
      const insideTrigger = triggerRef.current && triggerRef.current.contains(event.target);
      const insidePopover = popoverRef.current && popoverRef.current.contains(event.target);
      if (!insideTrigger && !insidePopover) {
        setOpen(false);
      }
    }
    function onKey(event) {
      if (event.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onDocClick);
    document.addEventListener("touchstart", onDocClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDocClick);
      document.removeEventListener("touchstart", onDocClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [open, triggerRef, popoverRef]);

  return (
    <div ref={triggerRef} className="relative inline-flex">
      <CompactActionButton
        icon="settings"
        label="العرض كيف ما تشتي"
        active={open}
        onClick={() => setOpen((prev) => !prev)}
      />

      {open && (
        <div
          ref={popoverRef}
          role="menu"
          dir="rtl"
          style={style}
          className="min-w-[220px] overflow-hidden rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-1.5 shadow-[var(--shadow-lg)] backdrop-blur-xl"
        >
          <p className="px-3 pb-1.5 pt-1 text-[10px] font-black text-[var(--text-muted)]">
            طريقة العرض
          </p>
          {VIEW_OPTIONS.map((option) => {
            const isActive = mode === option.id;
            return (
              <button
                key={option.id}
                type="button"
                role="menuitemradio"
                aria-checked={isActive}
                onClick={() => {
                  onSelect(option.id);
                  setOpen(false);
                }}
                className={`flex w-full items-center gap-2.5 rounded-xl px-2.5 py-2 text-right text-xs font-bold transition ${
                  isActive
                    ? "bg-[var(--color-accent)]/10 text-[var(--color-accent)]"
                    : "text-[var(--text-primary)] hover:bg-[var(--bg-surface)]"
                }`}
              >
                <span className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border ${isActive ? "border-[var(--color-accent)]/40 bg-[var(--color-accent)]/10" : "border-[var(--border-default)] bg-[var(--bg-surface)]"}`}>
                  <Icon name={option.icon} className={`h-4 w-4 ${isActive ? "text-[var(--color-accent)]" : "text-[var(--text-secondary)]"}`} />
                </span>
                <span className="flex-1 truncate">{option.label}</span>
                {isActive && <Icon name="checkbox" className="h-4 w-4 shrink-0 text-[var(--color-accent)]" />}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}