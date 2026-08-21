import React from "react";
import Icon from "./Icon";

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
  resultCount = 0,
}) {
  return (
    <div className="space-y-3 p-4 rounded-3xl bg-[#0D0C1A]/80 border border-white/10 backdrop-blur-xl" dir="rtl">
      {/* Top Row: Search Input & Sort Dropdown */}
      <div className="flex flex-col sm:flex-row items-center justify-between gap-3">
        {/* Search Field */}
        <div className="relative w-full sm:w-80">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => onSearchChange && onSearchChange(e.target.value)}
            placeholder="بحث فوري في هذا القسم..."
            className="w-full bg-slate-900/90 border border-slate-700/80 rounded-2xl pr-10 pl-4 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-fuchsia-500 transition-all shadow-inner"
          />
          <div className="absolute right-3.5 top-2.5 text-gray-400">
            <Icon name="search" className="w-4 h-4" />
          </div>
          {searchQuery && (
            <button
              onClick={() => onSearchChange && onSearchChange("")}
              className="absolute left-3 top-2.5 text-gray-400 hover:text-white text-xs"
            >
              ✕
            </button>
          )}
        </div>

        {/* Results Counter & Sort Selector */}
        <div className="flex items-center gap-3 w-full sm:w-auto justify-between sm:justify-end">
          <span className="text-xs font-semibold text-gray-400">
            النتائج: <strong className="text-fuchsia-300 font-mono">{resultCount}</strong>
          </span>

          <div className="flex items-center gap-2">
            <span className="text-xs text-gray-400 hidden sm:inline">الترتيب:</span>
            <select
              value={activeSort}
              onChange={(e) => onSelectSort && onSelectSort(e.target.value)}
              className="bg-slate-900 border border-slate-700 rounded-xl px-3 py-1.5 text-xs text-white font-medium focus:outline-none focus:border-fuchsia-500"
            >
              {sortOptions.map((opt) => (
                <option key={opt.id} value={opt.id}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>
        </div>
      </div>

      {/* Origin/Country Badges Bar */}
      {showOriginFilter && (
        <div className="flex items-center gap-1.5 overflow-x-auto pb-1 scrollbar-none">
          <span className="text-xs font-bold text-gray-400 ml-2 shrink-0">المنشأ:</span>
          {originsList.map((origin) => {
            const isSelected = activeOrigin === origin.id;
            return (
              <button
                key={origin.id}
                onClick={() => onSelectOrigin && onSelectOrigin(origin.id)}
                className={`px-3 py-1 rounded-xl text-xs font-bold transition shrink-0 border ${
                  isSelected
                    ? "bg-gradient-to-r from-fuchsia-600 to-purple-600 text-white border-fuchsia-400 shadow-md shadow-fuchsia-900/40"
                    : "bg-white/[0.04] text-white/70 border-white/5 hover:bg-white/10 hover:text-white"
                }`}
              >
                {origin.label}
              </button>
            );
          })}
        </div>
      )}

      {/* Genre Categories Bar */}
      <div className="flex items-center gap-1.5 overflow-x-auto pb-1 scrollbar-none pt-1 border-t border-white/5">
        <span className="text-xs font-bold text-gray-400 ml-2 shrink-0">التصنيف:</span>
        {genresList.map((genre) => {
          const isSelected = activeGenre === genre.id;
          return (
            <button
              key={genre.id}
              onClick={() => onSelectGenre && onSelectGenre(genre.id)}
              className={`px-3 py-1 rounded-xl text-xs font-medium transition shrink-0 border ${
                isSelected
                  ? "bg-purple-600 text-white border-purple-400 shadow-sm"
                  : "bg-white/[0.03] text-white/60 border-transparent hover:bg-white/10 hover:text-white"
              }`}
            >
              {genre.label}
            </button>
          );
        })}
      </div>
    </div>
  );
}
