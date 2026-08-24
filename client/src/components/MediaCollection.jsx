import { useState } from "react";
import Icon from "./Icon";
import UnifiedMediaCard from "./UnifiedMediaCard";

/**
 * Shared responsive catalogue surface. Use it anywhere a collection of media
 * is shown; it owns only the visual view mode, never the data or navigation.
 */
export default function MediaCollection({ items = [], onOpen, defaultView = "grid", className = "" }) {
  const [view, setView] = useState(defaultView);

  return (
    <section className={`media-collection ${className}`} aria-label="نتائج المكتبة" dir="rtl">
      <div className="mb-4 flex items-center justify-between gap-3">
        <p className="text-xs font-semibold text-[var(--text-secondary)]">{items.length} عمل متاح</p>
        <div className="inline-flex rounded-xl border border-[var(--border-default)] bg-[var(--bg-surface)] p-1 shadow-[var(--shadow-sm)]" aria-label="طريقة عرض الأعمال">
          <button type="button" onClick={() => setView("grid")} aria-pressed={view === "grid"} title="عرض شبكي" className={`flex h-8 w-9 items-center justify-center rounded-lg transition ${view === "grid" ? "bg-[var(--bg-card)] text-[var(--color-accent)] shadow-[var(--shadow-sm)]" : "text-[var(--text-muted)] hover:text-[var(--text-primary)]"}`}><Icon name="grid" className="h-4 w-4" /></button>
          <button type="button" onClick={() => setView("list")} aria-pressed={view === "list"} title="عرض قائمة" className={`flex h-8 w-9 items-center justify-center rounded-lg transition ${view === "list" ? "bg-[var(--bg-card)] text-[var(--color-accent)] shadow-[var(--shadow-sm)]" : "text-[var(--text-muted)] hover:text-[var(--text-primary)]"}`}><Icon name="list" className="h-4 w-4" /></button>
        </div>
      </div>
      <div className={view === "grid" ? "grid grid-cols-[repeat(auto-fill,minmax(155px,1fr))] gap-3 sm:grid-cols-[repeat(auto-fill,minmax(180px,1fr))] sm:gap-4" : "space-y-2.5 sm:space-y-3"}>
        {items.map((media) => <UnifiedMediaCard key={media.id} media={media} onOpen={onOpen} layout={view} />)}
      </div>
    </section>
  );
}
