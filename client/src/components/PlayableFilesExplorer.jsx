import { useEffect, useMemo, useState } from "react";
import Icon from "./Icon";
import CompactActionButton from "./CompactActionButton";
import ViewModeMenu from "./ViewModeMenu";
import RangeSelectionBar from "./RangeSelectionBar";
import useSelection from "../hooks/useSelection.js";
import useViewMode from "../hooks/useViewMode.js";

const VIEW_CLASSES = {
  extralarge: "grid-cols-1 sm:grid-cols-2 lg:grid-cols-2 xl:grid-cols-3",
  large: "grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4",
  medium: "grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4",
  list: "grid-cols-1",
  details: "grid-cols-1",
};

/**
 * File-Explorer-style surface for a list of playable episodes/files.
 *
 * Hosts the three Windows 11 inspired toolbar actions beside the section
 * heading:
 *  1. "العرض كيف ما تشتي"  → View mode menu (Extra Large → Details).
 *  2. "تحديد"              → Multi-select mode with per-item checkboxes and a
 *                            "تحديد الكل" bar.
 *  3. "انسخ من إلى"        → Range (inclusive) selection + direct copy.
 *
 * Selection is fully driven by the standalone `useSelection` hook and the
 * view mode by `useViewMode` (persisted in localStorage).
 *
 * @param {Object} props
 * @param {Array<Object>} props.items Episodes/files to render.
 * @param {string} props.title Section heading text.
 * @param {string} [props.icon] Icon name shown beside the heading (default "film").
 * @param {string} [props.countBadge] Optional count badge text next to the title.
 * @param {(item: Object, index: number) => void} props.onQuickPlay
 * @param {(items: Array<Object>) => void} [props.onCopySelected] Called when the
 *   user initiates a copy of a group of selected items (used by the range bar).
 * @param {string} [props.storageKey] localStorage key for the view mode.
 * @param {string} [props.defaultMode] Fallback view mode (default "medium").
 * @param {string} [props.className]
 */
export default function PlayableFilesExplorer({
  items = [],
  title = "ملفات الفيديو المتاحة للتشغيل",
  icon = "film",
  countBadge = "",
  onQuickPlay,
  onCopySelected,
  storageKey = "nexora_files_view_mode",
  defaultMode = "medium",
  className = "",
}) {
  const { mode, setMode, isGrid, isList, isDetails } = useViewMode(storageKey, defaultMode);
  const selection = useSelection(items);
  const [selectionMode, setSelectionMode] = useState(false);
  const [rangeApplied, setRangeApplied] = useState(null);

  // Switching out of selection mode discards any local selection.
  useEffect(() => {
    if (!selectionMode) selection.clearSelection();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectionMode]);

  // A different item list (e.g. a season change) invalidates old keys.
  useEffect(() => {
    selection.clearSelection();
    setRangeApplied(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [items]);

  const total = items.length;
  const selectedCount = selection.count;

  const rangeItems = useMemo(() => {
    if (!rangeApplied) return [];
    const lo = Math.min(rangeApplied.from, rangeApplied.to);
    const hi = Math.max(rangeApplied.from, rangeApplied.to);
    return items.slice(lo - 1, hi);
  }, [items, rangeApplied]);

  function handleRangeApply(from, to) {
    if (from == null || to == null) {
      setRangeApplied(null);
      return;
    }
    // Normalize the range to 1-based episode numbers for display, but select
    // using array index (index = episode number - 1).
    const lo = Math.min(from, to);
    const hi = Math.max(from, to);
    selection.selectRange(Math.max(0, lo - 1), Math.min(total - 1, hi - 1));
    setRangeApplied({ from: lo, to: hi });
  }

  function handleRangeCopy() {
    setRangeApplied(null);
    selection.clearSelection();
    onCopySelected?.(rangeItems);
  }

  function handleSelectAll() {
    if (selection.allSelected) selection.clearSelection();
    else selection.selectAll();
  }

  const modeTitle = {
    extralarge: "أيقونات كبيرة جداً",
    large: "أيقونات كبيرة",
    medium: "أيقونات متوسطة",
    list: "قائمة",
    details: "تفاصيل",
  }[mode];

  const itemNumber = (item, idx) => item?.episode_number || item?.episodeNumber || idx + 1;
  const itemTitle = (item, idx) =>
    item?.title_ar || item?.title_en || `الحلقة ${itemNumber(item, idx)}`;

  const renderCardActions = (item, idx) => {
    const selected = selection.isSelected(item);
    return (
      <>
        {selectionMode && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              selection.toggle(item);
            }}
            aria-pressed={selected}
            title={selected ? "إلغاء التحديد" : "تحديد"}
            className={`absolute left-2 top-2 z-10 flex h-6 w-6 items-center justify-center rounded-lg border-2 transition ${
              selected
                ? "border-[var(--color-accent)] bg-[var(--color-accent)] text-white shadow-md"
                : "border-white/60 bg-black/40 text-transparent hover:border-[var(--color-accent)]"
            }`}
          >
            <Icon name="checkbox" className="h-3.5 w-3.5" />
          </button>
        )}
        <button
          type="button"
          onClick={() => onCopySelected?.([item])}
          title="نسخ فوري إلى الهاتف"
          className={`absolute bottom-2 left-2 z-10 flex h-7 w-7 items-center justify-center rounded-lg border border-emerald-500/30 bg-emerald-950/60 text-xs text-emerald-300 transition hover:bg-emerald-900 hover:text-white ${selectionMode ? "hidden" : ""}`}
        >
          📱
        </button>
      </>
    );
  };

  const renderPlayableItem = (item, idx) => {
    const selected = selection.isSelected(item);
    const number = itemNumber(item, idx);
    const resolution = item?.resolution || "1080p";
    const codec = item?.video_codec || "HEVC/H264";

    // List & details share a row layout; details adds more metadata columns.
    if (isList || isDetails) {
      return (
        <article
          key={item.id || idx}
          onClick={() => selectionMode && selection.toggle(item)}
          className={`group relative flex items-center gap-3 rounded-2xl border bg-[var(--bg-elevated)] p-2.5 transition ${
            selected
              ? "border-[var(--color-accent)] bg-[var(--color-accent)]/5 shadow-md"
              : "border-[var(--border-default)] hover:border-fuchsia-500/60 hover:bg-[var(--bg-card)] hover:shadow-md"
          }`}
        >
          {selectionMode && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                selection.toggle(item);
              }}
              aria-pressed={selected}
              className={`flex h-6 w-6 shrink-0 items-center justify-center rounded-lg border-2 transition ${
                selected
                  ? "border-[var(--color-accent)] bg-[var(--color-accent)] text-white"
                  : "border-white/50 bg-transparent text-transparent hover:border-[var(--color-accent)]"
              }`}
            >
              <Icon name="checkbox" className="h-3.5 w-3.5" />
            </button>
          )}
          <button
            type="button"
            onClick={() => !selectionMode && onQuickPlay?.(item, idx)}
            className="relative h-16 w-20 shrink-0 overflow-hidden rounded-xl bg-black/40 border border-white/10"
            aria-label={`تشغيل ${itemTitle(item, idx)}`}
          >
            <div className="absolute inset-0 flex items-center justify-center bg-black/35">
              <span className="flex h-6 w-6 items-center justify-center rounded-full bg-gradient-to-tr from-fuchsia-600 to-purple-600 text-white text-[10px] font-black">▶</span>
            </div>
          </button>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-1.5">
              <span className="rounded-md bg-gradient-to-r from-fuchsia-600 to-purple-600 px-2 py-0.5 text-[10px] font-black text-white shrink-0">حلقة {number}</span>
              <span className="rounded bg-white/10 px-1.5 py-0.5 text-[9px] font-bold text-[var(--text-muted)] truncate">{resolution}</span>
            </div>
            <p className="mt-1 truncate text-xs font-bold text-[var(--text-primary)] group-hover:text-fuchsia-300 transition">
              {itemTitle(item, idx)}
            </p>
            {isDetails && (
              <p className="mt-0.5 text-[10px] text-[var(--text-muted)]">
                {codec} • {item?.file_size || item?.size ? `${(Number(item?.file_size || item?.size) / (1024 * 1024)).toFixed(1)} MB` : "تشغيل فوري"}
              </p>
            )}
          </div>
        </article>
      );
    }

    return (
      <article
        key={item.id || idx}
        onClick={() => selectionMode && selection.toggle(item)}
        className={`group flex items-center gap-2 rounded-2xl border p-2.5 text-right transition ${mode === "large" ? "lg:flex-col lg:items-stretch" : ""} ${
          selected
            ? "border-[var(--color-accent)] bg-[var(--color-accent)]/5 shadow-md"
            : "border-[var(--border-default)] bg-[var(--bg-elevated)] hover:border-fuchsia-500/60 hover:bg-[var(--bg-card)] hover:shadow-md"
        }`}
      >
        {renderCardActions(item, idx)}
        <button
          type="button"
          onClick={() => !selectionMode && onQuickPlay?.(item, idx)}
          className={`relative shrink-0 overflow-hidden rounded-xl bg-black/40 border border-white/10 ${
            mode === "extralarge"
              ? "h-24 w-36 sm:h-28 sm:w-44"
              : mode === "large"
                ? "h-20 w-32 sm:h-24 sm:w-40"
                : "h-14 w-20"
          }`}
          aria-label={`تشغيل ${itemTitle(item, idx)}`}
        >
          <img
            src="/nexora-episode-placeholder.PNG"
            alt=""
            className="h-full w-full object-cover opacity-60 group-hover:opacity-90 transition duration-300"
            onError={(e) => { e.currentTarget.style.display = "none"; }}
          />
          <div className="absolute inset-0 flex items-center justify-center bg-black/35 group-hover:bg-black/15 transition">
            <span className={`flex items-center justify-center rounded-full bg-gradient-to-tr from-fuchsia-600 to-purple-600 text-white shadow text-[10px] font-black ${
              mode === "extralarge" ? "h-9 w-9 sm:h-10 sm:w-10 text-base" : mode === "large" ? "h-7 w-7 sm:h-8 sm:w-8 text-xs" : "h-6 w-6"
            }`}>
              ▶
            </span>
          </div>
        </button>

        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5 mb-1">
            <span className="rounded-md bg-gradient-to-r from-fuchsia-600 to-purple-600 px-2 py-0.5 text-[10px] font-black text-white shadow-sm shrink-0">
              حلقة {number}
            </span>
            <span className="rounded bg-white/10 px-1.5 py-0.5 text-[9px] font-bold text-[var(--text-muted)]">{resolution}</span>
          </div>
          <p className="truncate text-xs font-bold text-[var(--text-primary)] group-hover:text-fuchsia-300 transition">{itemTitle(item, idx)}</p>
          {isGrid && mode !== "extralarge" && (
            <p className="mt-0.5 text-[10px] text-[var(--text-muted)] truncate">{codec} • تشغيل فوري</p>
          )}
        </div>
      </article>
    );
  };

  return (
    <div className={`playable-files-explorer ${className}`} dir="rtl">
      {/* Toolbar: heading + three compact actions */}
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2.5">
        <div className="flex min-w-0 items-center gap-2">
          {icon && (
            <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-lg bg-[var(--bg-elevated)] text-[var(--color-accent)]">
              <Icon name={icon} className="h-3.5 w-3.5" />
            </span>
          )}
          <h3 className="truncate text-sm font-black text-[var(--text-primary)]">{title}</h3>
          {countBadge && (
            <span className="shrink-0 rounded-full bg-[var(--bg-elevated)] px-2.5 py-0.5 text-[10px] font-black text-[var(--text-secondary)] border border-[var(--border-subtle)]">{countBadge}</span>
          )}
        </div>

        <div className="flex items-center gap-1.5">
          <ViewModeMenu mode={mode} onSelect={setMode} />

          <CompactActionButton
            icon="checkbox"
            label="تحديد"
            active={selectionMode}
            onClick={() => setSelectionMode((prev) => !prev)}
            title="وضع التحديد"
          />

          <RangeSelectionBar items={items} onApplyRange={handleRangeApply} onCopy={handleRangeCopy} />
        </div>
      </div>

      {/* Selection-mode utility bar */}
      {selectionMode && (
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2 rounded-xl border border-[var(--color-accent)]/30 bg-[var(--color-accent)]/5 px-3 py-2">
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={handleSelectAll}
              className="inline-flex items-center gap-1.5 rounded-lg border border-[var(--border-default)] bg-[var(--bg-surface)] px-2.5 py-1.5 text-[10px] font-black text-[var(--text-primary)] transition hover:border-[var(--color-accent)]/50"
            >
              <Icon name="selectAll" className="h-3.5 w-3.5 text-[var(--color-accent)]" />
              {selection.allSelected ? "إلغاء تحديد الكل" : "تحديد الكل"}
            </button>
            <span className="text-[11px] font-bold text-[var(--text-secondary)]">
              <span className="font-black text-[var(--color-accent)]">{selectedCount}</span> من {total} محدد
            </span>
          </div>
          {selectedCount > 0 && onCopySelected && (
            <button
              type="button"
              onClick={() => {
                const copyItems = selection.selectedItems;
                selection.clearSelection();
                onCopySelected?.(copyItems);
              }}
              className="inline-flex items-center gap-1.5 rounded-lg bg-gradient-to-r from-fuchsia-600 to-purple-600 px-3 py-1.5 text-[10px] font-black text-white shadow-sm transition hover:brightness-110 active:scale-[0.98]"
            >
              <span>📲</span>
              نسخ {selectedCount} المختارة
            </button>
          )}
        </div>
      )}

      {/* Range applied hint, shown above the grid */}
      {rangeApplied && !selectionMode && (
        <div className="mb-3 flex items-center justify-between gap-2 rounded-xl border border-emerald-500/30 bg-emerald-950/30 px-3 py-2">
          <span className="flex items-center gap-1.5 text-[10px] font-black text-emerald-300">
            <Icon name="checkbox" className="h-3.5 w-3.5" />
            تم تحديد {rangeApplied.to - rangeApplied.from + 1} حلقة ضمن المدى ({rangeApplied.from} → {rangeApplied.to})
          </span>
          {onCopySelected && (
            <button
              type="button"
              onClick={handleRangeCopy}
              className="inline-flex items-center gap-1.5 rounded-lg border border-emerald-500/40 bg-emerald-950/60 px-2.5 py-1.5 text-[10px] font-black text-emerald-200 transition hover:bg-emerald-900/70"
            >
              <span>📲</span>
              بدء النسخ
            </button>
          )}
        </div>
      )}

      {/* Items grid/list */}
      {total === 0 ? (
        <p className="rounded-2xl border border-dashed border-[var(--border-default)] bg-[var(--bg-surface)] px-4 py-8 text-center text-xs text-[var(--text-muted)]">
          لا توجد ملفات فيديو متاحة ضمن هذا القسم.
        </p>
      ) : (
        <div className={`grid gap-2.5 ${VIEW_CLASSES[mode]}`}>
          {items.map((item, idx) => renderPlayableItem(item, idx))}
        </div>
      )}

      {/* Small caption of the current mode */}
      <p className="mt-2 text-[10px] text-[var(--text-muted)]">
        طريقة العرض: {modeTitle}
      </p>
    </div>
  );
}