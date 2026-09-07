import { useEffect, useState } from "react";
import Icon from "./Icon";
import UnifiedMediaCard from "./UnifiedMediaCard";
import ViewModeMenu from "./ViewModeMenu";
import CompactActionButton from "./CompactActionButton";
import useSelection from "../hooks/useSelection.js";
import useViewMode from "../hooks/useViewMode.js";

const DEFAULT_VIEW_MAP = { grid: "large", list: "list" };

/**
 * Shared responsive catalogue surface. Use it anywhere a collection of media
 * is shown; it owns only the visual view mode (Windows 11 style menu, persisted
 * in localStorage) and the optional multi-select mode, never the data itself.
 *
 * @param {Object} props
 * @param {Array<Object>} props.items Media items.
 * @param {(media: Object) => void} [props.onOpen]
 * @param {string} [props.defaultView] "grid" or "list" (legacy values mapped
 *   onto the richer view modes).
 * @param {Function} [props.cardActions] Renders extra controls over grid items.
 * @param {(items: Array<Object>) => void} [props.onCopySelected] When provided,
 *   enables the "تحديد" (selection mode) toggle plus a "تحديد الكل" bar.
 * @param {boolean} [props.enableSelection] Forces the selection UI even without
 *   a copy action (just checkboxes + select-all).
 * @param {string} [props.storageKey] localStorage key for the view mode.
 */
export default function MediaCollection({
  items = [],
  onOpen,
  defaultView = "grid",
  className = "",
  cardActions,
  onCopySelected,
  enableSelection = false,
  storageKey = "nexora_view_mode",
}) {
  const { mode, setMode, isGrid, isList, isDetails } = useViewMode(
    storageKey,
    DEFAULT_VIEW_MAP[defaultView] || "large"
  );

  const selection = useSelection(items);
  const [selectionMode, setSelectionMode] = useState(false);

  useEffect(() => {
    if (!selectionMode) selection.clearSelection();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectionMode]);

  const selectionActive = (enableSelection || Boolean(onCopySelected)) && selectionMode;
  const onCardOpen = selectionActive ? (media) => selection.toggle(media) : onOpen;

  const gridClass =
    isList || isDetails
      ? "space-y-2.5 sm:space-y-3"
      : mode === "extralarge"
        ? "grid grid-cols-2 gap-3 sm:grid-cols-[repeat(auto-fill,minmax(230px,1fr))] sm:gap-5"
        : mode === "medium"
          ? "grid grid-cols-2 gap-3 sm:grid-cols-[repeat(auto-fill,minmax(150px,1fr))] sm:gap-3"
          : "grid grid-cols-2 gap-3 sm:grid-cols-[repeat(auto-fill,minmax(190px,1fr))] sm:gap-4";

  return (
    <section className={`media-collection ${className}`} aria-label="نتائج المكتبة" dir="rtl">
      <div className="mb-4 flex items-center justify-between gap-3">
        <p className="text-xs font-semibold text-[var(--text-secondary)]">{items.length} عمل متاح</p>
        <div className="flex items-center gap-1.5">
          {selectionActive || enableSelection || onCopySelected ? (
            <CompactActionButton
              icon="checkbox"
              label="تحديد"
              active={selectionMode}
              onClick={() => setSelectionMode((prev) => !prev)}
              title="وضع التحديد"
            />
          ) : null}
          <ViewModeMenu mode={mode} onSelect={setMode} />
        </div>
      </div>

      {/* Selection-mode utility bar (Select All) */}
      {selectionActive && (
        <div className="mb-4 flex flex-wrap items-center justify-between gap-2 rounded-xl border border-[var(--color-accent)]/30 bg-[var(--color-accent)]/5 px-3 py-2">
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={() => (selection.allSelected ? selection.clearSelection() : selection.selectAll())}
              className="inline-flex items-center gap-1.5 rounded-lg border border-[var(--border-default)] bg-[var(--bg-surface)] px-2.5 py-1.5 text-[10px] font-black text-[var(--text-primary)] transition hover:border-[var(--color-accent)]/50"
            >
              <Icon name="selectAll" className="h-3.5 w-3.5 text-[var(--color-accent)]" />
              {selection.allSelected ? "إلغاء تحديد الكل" : "تحديد الكل"}
            </button>
            <span className="text-[11px] font-bold text-[var(--text-secondary)]">
              <span className="font-black text-[var(--color-accent)]">{selection.count}</span> من {items.length} محدد
            </span>
          </div>
          {selection.count > 0 && onCopySelected && (
            <button
              type="button"
              onClick={() => {
                const copyItems = selection.selectedItems;
                selection.clearSelection();
                onCopySelected(copyItems);
              }}
              className="inline-flex items-center gap-1.5 rounded-lg bg-gradient-to-r from-fuchsia-600 to-purple-600 px-3 py-1.5 text-[10px] font-black text-white shadow-sm transition hover:brightness-110 active:scale-[0.98]"
            >
              <span>📲</span>
              نسخ {selection.count} المختارة
            </button>
          )}
        </div>
      )}

      <div className={gridClass}>
        {items.map((media) => {
          const selected = selection.isSelected(media);
          return (
            <div key={media.id} className={`${isGrid ? "min-w-0" : ""} relative ${cardActions && isGrid ? "group/card" : ""}`}>
              <UnifiedMediaCard media={media} onOpen={onCardOpen} layout={isGrid ? "grid" : "list"} />
              {cardActions && isGrid ? <div className="absolute left-2.5 top-12 z-10 flex gap-1.5">{cardActions(media)}</div> : null}
              {selectionActive && (
                <button
                  type="button"
                  onClick={() => selection.toggle(media)}
                  aria-pressed={selected}
                  title={selected ? "إلغاء التحديد" : "تحديد"}
                  className={`absolute left-2.5 top-2.5 z-10 flex h-6 w-6 items-center justify-center rounded-lg border-2 backdrop-blur-sm transition ${
                    selected
                      ? "border-[var(--color-accent)] bg-[var(--color-accent)] text-white shadow-md"
                      : "border-white/70 bg-black/45 text-transparent hover:border-[var(--color-accent)]"
                  }`}
                >
                  <Icon name="checkbox" className="h-3.5 w-3.5" />
                </button>
              )}
            </div>
          );
        })}
      </div>
    </section>
  );
}