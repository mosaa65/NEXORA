import { useEffect, useState } from "react";
import Icon from "./Icon";
import CompactActionButton from "./CompactActionButton";
import usePopover from "../hooks/usePopover.js";

/**
 * "انسخ من إلى" — range selection popover.
 *
 * Lets the user pick an inclusive episode range (من حلقة → إلى حلقة) that is
 * instantly reflected in the selection state. Handles invalid input (from > to,
 * out of bounds, empty) with an inline error message, and exposes a direct
 * "بدء النسخ" action once a valid range is applied.
 *
 * @param {Object} props
 * @param {Array<Object>} props.items The ordered episode/file list (index-based matching).
 * @param {(from: number, to: number) => void} props.onApplyRange Called with 1-based episode numbers.
 * @param {(files: Array<Object>) => void} [props.onCopy] Direct copy of the resolved range files.
 */
export default function RangeSelectionBar({ items = [], onApplyRange, onCopy }) {
  const [open, setOpen] = useState(false);
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [error, setError] = useState("");
  const [appliedRange, setAppliedRange] = useState(null);
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
      if (event.key === "Escape") {
        setOpen(false);
        onApplyRange?.(null);
      }
    }
    document.addEventListener("mousedown", onDocClick);
    document.addEventListener("touchstart", onDocClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDocClick);
      document.removeEventListener("touchstart", onDocClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [open, onApplyRange, triggerRef, popoverRef]);

  function evaluateRange() {
    const fromNum = Number.parseInt(from, 10);
    const toNum = Number.parseInt(to, 10);

    if (!from || !to || Number.isNaN(fromNum) || Number.isNaN(toNum)) {
      return { error: "", range: null };
    }
    if (fromNum <= 0 || toNum <= 0) {
      return { error: "أرقام الحلقات يجب أن تكون أكبر من صفر.", range: null };
    }
    if (fromNum > toNum) {
      return { error: "رقم البداية (من حلقة) لا يمكن أن يكون أكبر من رقم النهاية (إلى حلقة).", range: null };
    }
    if (fromNum > items.length || toNum > items.length) {
      return { error: `المدى المتاح هو من 1 إلى ${items.length} حلقة فقط.`, range: null };
    }
    return { error: "", range: { from: fromNum, to: toNum } };
  }

  function resetRange() {
    setFrom("");
    setTo("");
    setError("");
    setAppliedRange(null);
    onApplyRange?.(null);
  }

  function applyRange(range) {
    setError("");
    setAppliedRange(range);
    onApplyRange?.(range.from, range.to);
  }

  function handleApply() {
    const { error, range } = evaluateRange();
    if (error) {
      setError(error);
      return;
    }
    if (!range) {
      setError("يرجى إدخال رقمي البداية والنهاية معاً.");
      return;
    }
    applyRange(range);
  }

  // "بمجرد إدخال المدى" — auto-apply the inclusive range as soon as both
  // fields hold a valid value, so the selection updates live while typing.
  useEffect(() => {
    if (!open) return;
    const { error, range } = evaluateRange();
    if (error) {
      setError(error);
      return;
    }
    if (range) {
      setError("");
      setAppliedRange(range);
      onApplyRange?.(range.from, range.to);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, from, to, items.length]);

  const rangeCount = appliedRange ? appliedRange.to - appliedRange.from + 1 : 0;

  return (
    <div ref={triggerRef} className="relative inline-flex">
      <CompactActionButton
        icon="range"
        label="انسخ من إلى"
        active={open || Boolean(appliedRange)}
        onClick={() => setOpen((prev) => !prev)}
      />

      {open && (
        <div
          ref={popoverRef}
          dir="rtl"
          style={style}
          className="w-[300px] rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-3.5 shadow-[var(--shadow-lg)] backdrop-blur-xl"
        >
          <p className="mb-2.5 text-[11px] font-black text-[var(--text-primary)] flex items-center gap-1.5">
            <Icon name="range" className="h-3.5 w-3.5 text-[var(--color-accent)]" />
            انسخ الحلقات من مدى محدد
          </p>

          <div className="grid grid-cols-2 gap-2">
            <label className="block">
              <span className="mb-1 block text-[10px] font-bold text-[var(--text-muted)]">من حلقة</span>
              <input
                autoFocus
                type="number"
                min="1"
                value={from}
                onChange={(e) => setFrom(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleApply();
                }}
                placeholder="مثال: 15"
                className="w-full rounded-xl border border-[var(--border-default)] bg-[var(--bg-surface)] px-2.5 py-2 text-xs font-bold text-[var(--text-primary)] outline-none transition focus:border-[var(--color-accent)]/60 focus:ring-2 focus:ring-[var(--color-accent)]/20"
              />
            </label>
            <label className="block">
              <span className="mb-1 block text-[10px] font-bold text-[var(--text-muted)]">إلى حلقة</span>
              <input
                type="number"
                min="1"
                value={to}
                onChange={(e) => setTo(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleApply();
                }}
                placeholder="مثال: 30"
                className="w-full rounded-xl border border-[var(--border-default)] bg-[var(--bg-surface)] px-2.5 py-2 text-xs font-bold text-[var(--text-primary)] outline-none transition focus:border-[var(--color-accent)]/60 focus:ring-2 focus:ring-[var(--color-accent)]/20"
              />
            </label>
          </div>

          {error && (
            <p className="mt-2 flex items-start gap-1.5 rounded-xl border border-rose-500/30 bg-rose-950/30 px-2.5 py-2 text-[10px] font-bold leading-relaxed text-rose-300">
              <span className="mt-0.5 shrink-0">⚠️</span>
              {error}
            </p>
          )}

          {appliedRange && !error && (
            <p className="mt-2 flex items-center gap-1.5 rounded-xl border border-emerald-500/30 bg-emerald-950/30 px-2.5 py-2 text-[10px] font-black text-emerald-300">
              <Icon name="checkbox" className="h-3.5 w-3.5 shrink-0" />
              تم تحديد {rangeCount} حلقة ({appliedRange.from} → {appliedRange.to}) ضمن المدى المطلوب.
            </p>
          )}

          <div className="mt-3 flex items-center gap-2">
            <button
              type="button"
              onClick={handleApply}
              disabled={!from || !to}
              className="flex-1 rounded-xl bg-gradient-to-r from-fuchsia-600 to-purple-600 px-3 py-2 text-[11px] font-black text-white shadow-sm transition hover:brightness-110 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50"
            >
              تطبيق التحديد
            </button>
            {appliedRange && onCopy && (
              <button
                type="button"
                onClick={() => {
                  const startIdx = appliedRange.from - 1;
                  const endIdx = appliedRange.to - 1;
                  const rangeItems = items.slice(startIdx, endIdx + 1);
                  onCopy(rangeItems);
                }}
                className="flex items-center gap-1 rounded-xl border border-emerald-500/40 bg-emerald-950/60 px-3 py-2 text-[11px] font-black text-emerald-200 transition hover:bg-emerald-900/80"
              >
                <span>📲</span>
                بدء النسخ
              </button>
            )}
            <button
              type="button"
              onClick={resetRange}
              title="إلغاء المدى"
              className="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl border border-[var(--border-default)] bg-[var(--bg-surface)] text-[var(--text-muted)] transition hover:text-[var(--text-primary)]"
            >
              <Icon name="close" className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      )}

      {!open && appliedRange && (
        <span className="absolute -top-1.5 -left-1.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-[var(--color-accent)] px-1 text-[9px] font-black text-white shadow-sm">
          {rangeCount}
        </span>
      )}
    </div>
  );
}