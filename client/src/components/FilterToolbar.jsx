import React from "react";
import Icon from "./Icon.jsx";

export const originsList = [
  { id: "all", label: "جميع البلدان" },
  { id: "تركي", label: "👑 تركي" },
  { id: "عربي", label: "🌟 عربي" },
  { id: "أجنبي", label: "🎬 أجنبي" },
  { id: "كوري", label: "🌸 كوري" },
  { id: "هندي", label: "⚡ هندي" },
  { id: "ياباني", label: "⚔️ ياباني" },
];

export const genresList = [
  { id: "all", label: "جميع التصنيفات" },
  { id: "أكشن", label: "أكشن" },
  { id: "مغامرة", label: "مغامرة" },
  { id: "دراما", label: "دراما" },
  { id: "كوميديا", label: "كوميديا" },
  { id: "غموض", label: "غموض" },
  { id: "جريمة", label: "جريمة" },
  { id: "خيال علمي", label: "خيال علمي" },
  { id: "رعب", label: "رعب" },
  { id: "أنمي", label: "أنمي" },
  { id: "عائلي", label: "عائلي" },
];

export const sortOptions = [
  { id: "newest", label: "الأحدث إضافة" },
  { id: "rating", label: "الأعلى تقييماً ⭐" },
  { id: "year", label: "سنة الإنتاج (الأحدث)" },
  { id: "title", label: "أبجدياً (A - Z)" },
];

export default function FilterToolbar({
  activeOrigin = "all",
  onSelectOrigin,
  activeGenre = "all",
  onSelectGenre,
  activeSort = "newest",
  onSelectSort,
  searchQuery = "",
  onSearchChange,
  showOriginFilter = true,
  showGenreFilter = true,
  showSort = true,
  showSearch = true,
  resultCount = 0,
  origins = originsList,
  genres = genresList,
  sorts = sortOptions,
  onResetFilters,
}) {
  const hasActiveFilters =
    (activeOrigin && activeOrigin !== "all") ||
    (activeGenre && activeGenre !== "all") ||
    Boolean(searchQuery && searchQuery.trim());

  const handleReset = () => {
    if (onResetFilters) {
      onResetFilters();
    } else {
      if (onSelectOrigin) onSelectOrigin("all");
      if (onSelectGenre) onSelectGenre("all");
      if (onSearchChange) onSearchChange("");
    }
  };

  return (
    <div
      className="space-y-4 rounded-3xl border border-[var(--border-default)] bg-[var(--bg-card)] p-4 sm:p-5 shadow-[var(--shadow-lg)] backdrop-blur-2xl transition-all"
      dir="rtl"
    >
      {/* Top Controls Row: Search, Results Counter, Sort & Reset */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-1 border-b border-[var(--border-subtle)]">
        {/* Search Field */}
        {showSearch && (
          <div className="relative flex-1 sm:max-w-xs md:max-w-sm">
            <div className="flex w-full items-center gap-2.5 rounded-full border border-[var(--border-default)] bg-[var(--bg-input)] px-3.5 py-2 text-[var(--text-primary)] shadow-inner transition focus-within:border-[var(--color-accent)] focus-within:bg-[var(--bg-input)] focus-within:ring-2 focus-within:ring-[var(--color-accent-light)]">
              <Icon name="search" className="h-4 w-4 text-fuchsia-400 shrink-0" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => onSearchChange && onSearchChange(e.target.value)}
                placeholder="بحث فوري في هذا القسم..."
                className="w-full bg-transparent text-xs sm:text-sm font-semibold text-[var(--text-primary)] outline-none placeholder:text-[var(--text-muted)] text-right selection:bg-[var(--color-accent-light)]"
              />
              {searchQuery && (
                <button
                  type="button"
                  onClick={() => onSearchChange && onSearchChange("")}
                  className="flex h-5 w-5 items-center justify-center rounded-full bg-[var(--bg-elevated)] text-xs text-[var(--text-muted)] hover:bg-[var(--bg-surface)] hover:text-[var(--text-primary)] transition"
                  title="مسح البحث"
                >
                  ✕
                </button>
              )}
            </div>
          </div>
        )}

        {/* Right Area: Results Counter, Sort Dropdown, and Clear Filters Button */}
        <div className="flex flex-wrap items-center justify-between sm:justify-end gap-2.5">
          {/* Results Badge */}
          <div className="inline-flex items-center gap-1.5 rounded-full border border-[var(--border-default)] bg-[var(--bg-surface)] px-3 py-1.5 text-xs font-semibold text-[var(--text-secondary)]">
            <span>النتائج:</span>
            <span className="font-mono font-black text-fuchsia-400">{resultCount}</span>
          </div>

          {/* Reset Filters Action */}
          {hasActiveFilters && (
            <button
              type="button"
              onClick={handleReset}
              className="inline-flex items-center gap-1 rounded-full border border-fuchsia-500/30 bg-fuchsia-950/40 px-3 py-1.5 text-xs font-bold text-fuchsia-300 hover:bg-fuchsia-900/60 hover:text-white transition active:scale-95"
              title="إعادة تعيين جميع الفلاتر"
            >
              <span>إلغاء التصفية</span>
              <span className="text-[10px]">✕</span>
            </button>
          )}

          {/* Sort Selector */}
          {showSort && (
            <div className="flex items-center gap-2">
              <span className="text-xs text-[var(--text-muted)] hidden md:inline">الترتيب:</span>
              <div className="relative">
                <select
                  value={activeSort}
                  onChange={(e) => onSelectSort && onSelectSort(e.target.value)}
                  className="appearance-none rounded-full border border-[var(--border-default)] bg-[var(--bg-input)] py-1.5 pl-8 pr-4 text-xs font-bold text-[var(--text-primary)] shadow-sm outline-none transition hover:border-[var(--color-accent)] focus:border-[var(--color-accent)] cursor-pointer text-right"
                >
                  {sorts.map((opt) => (
                    <option key={opt.id} value={opt.id} className="bg-[var(--bg-card)] text-[var(--text-primary)]">
                      {opt.label}
                    </option>
                  ))}
                </select>
                <div className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-muted)] text-[10px]">
                  ▼
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Origin/Country Filters Row */}
      {showOriginFilter && (
        <div className="space-y-1.5">
          <div className="flex items-center gap-2 overflow-x-auto pb-1.5 pt-0.5 scrollbar-thin">
            <span className="shrink-0 text-xs font-bold text-[var(--text-muted)] ml-1">المنشأ:</span>
            {origins.map((origin) => {
              const isSelected = activeOrigin === origin.id;
              return (
                <button
                  key={origin.id}
                  type="button"
                  onClick={() => onSelectOrigin && onSelectOrigin(origin.id)}
                  className={`shrink-0 rounded-full px-3.5 py-1.5 text-xs font-bold transition-all duration-200 active:scale-95 ${
                    isSelected
                      ? "bg-gradient-to-r from-fuchsia-600 via-purple-600 to-fuchsia-700 text-white shadow-md shadow-fuchsia-950/60 border border-fuchsia-400 ring-2 ring-fuchsia-500/20"
                      : "bg-[var(--bg-surface)] text-[var(--text-secondary)] border border-[var(--border-subtle)] hover:bg-[var(--bg-elevated)] hover:text-[var(--text-primary)] hover:border-[var(--border-default)]"
                  }`}
                >
                  {origin.label}
                </button>
              );
            })}
          </div>
        </div>
      )}

      {/* Genre/Category Filters Row */}
      {showGenreFilter && (
        <div className="space-y-1.5">
          <div className="flex items-center gap-2 overflow-x-auto pb-1 pt-0.5 scrollbar-thin">
            <span className="shrink-0 text-xs font-bold text-[var(--text-muted)] ml-1">التصنيف:</span>
            {genres.map((genre) => {
              const isSelected = activeGenre === genre.id;
              return (
                <button
                  key={genre.id}
                  type="button"
                  onClick={() => onSelectGenre && onSelectGenre(genre.id)}
                  className={`shrink-0 rounded-full px-3.5 py-1.5 text-xs font-bold transition-all duration-200 active:scale-95 ${
                    isSelected
                      ? "bg-purple-600 text-white shadow-sm border border-purple-400 ring-2 ring-purple-500/20"
                      : "bg-[var(--bg-surface)] text-[var(--text-secondary)] border border-[var(--border-subtle)] hover:bg-[var(--bg-elevated)] hover:text-[var(--text-primary)] hover:border-[var(--border-default)]"
                  }`}
                >
                  {genre.label}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
