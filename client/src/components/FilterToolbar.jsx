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
      className="space-y-4 rounded-3xl border border-white/12 bg-[#0C0B18]/85 p-4 sm:p-5 shadow-2xl backdrop-blur-2xl transition-all"
      dir="rtl"
    >
      {/* Top Controls Row: Search, Results Counter, Sort & Reset */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-1 border-b border-white/5">
        {/* Search Field */}
        {showSearch && (
          <div className="relative flex-1 sm:max-w-xs md:max-w-sm">
            <div className="flex w-full items-center gap-2.5 rounded-full border border-white/15 bg-black/60 px-3.5 py-2 text-white shadow-inner transition focus-within:border-fuchsia-500/80 focus-within:bg-black/90 focus-within:ring-2 focus-within:ring-fuchsia-500/25">
              <Icon name="search" className="h-4 w-4 text-fuchsia-400 shrink-0" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => onSearchChange && onSearchChange(e.target.value)}
                placeholder="بحث فوري في هذا القسم..."
                className="w-full bg-transparent text-xs sm:text-sm font-semibold text-white outline-none placeholder:text-white/50 text-right selection:bg-fuchsia-600"
              />
              {searchQuery && (
                <button
                  type="button"
                  onClick={() => onSearchChange && onSearchChange("")}
                  className="flex h-5 w-5 items-center justify-center rounded-full bg-white/10 text-xs text-white/70 hover:bg-white/20 hover:text-white transition"
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
          <div className="inline-flex items-center gap-1.5 rounded-full border border-white/10 bg-white/5 px-3 py-1.5 text-xs font-semibold text-white/70">
            <span>النتائج:</span>
            <span className="font-mono font-black text-fuchsia-300">{resultCount}</span>
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
              <span className="text-xs text-white/50 hidden md:inline">الترتيب:</span>
              <div className="relative">
                <select
                  value={activeSort}
                  onChange={(e) => onSelectSort && onSelectSort(e.target.value)}
                  className="appearance-none rounded-full border border-white/15 bg-black/60 py-1.5 pl-8 pr-4 text-xs font-bold text-white shadow-sm outline-none transition hover:border-white/30 focus:border-fuchsia-500 cursor-pointer text-right"
                >
                  {sorts.map((opt) => (
                    <option key={opt.id} value={opt.id} className="bg-[#0C0B18] text-white">
                      {opt.label}
                    </option>
                  ))}
                </select>
                <div className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-white/50 text-[10px]">
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
          <div className="flex items-center gap-2 overflow-x-auto pb-1.5 pt-0.5 scrollbar-thin scrollbar-thumb-white/10">
            <span className="shrink-0 text-xs font-bold text-white/50 ml-1">المنشأ:</span>
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
                      : "bg-white/[0.04] text-white/70 border border-white/8 hover:bg-white/10 hover:text-white hover:border-white/20"
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
          <div className="flex items-center gap-2 overflow-x-auto pb-1 pt-0.5 scrollbar-thin scrollbar-thumb-white/10">
            <span className="shrink-0 text-xs font-bold text-white/50 ml-1">التصنيف:</span>
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
                      : "bg-white/[0.03] text-white/60 border border-white/5 hover:bg-white/10 hover:text-white hover:border-white/15"
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
