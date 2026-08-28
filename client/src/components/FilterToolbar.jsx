import React, { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
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
  { id: "oldest", label: "الأقدم إضافة" },
  { id: "rating", label: "الأعلى تقييماً" },
  { id: "rating_low", label: "الأقل تقييماً" },
  { id: "year", label: "سنة الإنتاج (الأحدث)" },
  { id: "year_old", label: "سنة الإنتاج (الأقدم)" },
  { id: "title", label: "الاسم (أ - ي)" },
  { id: "title_desc", label: "الاسم (ي - أ)" },
  { id: "files", label: "الأكثر ملفات" },
  { id: "runtime", label: "الأطول مدة" },
];

export const typeOptions = [
  { id: "all", label: "كل الأنواع" },
  { id: "movie", label: "أفلام" },
  { id: "series", label: "مسلسلات" },
  { id: "anime", label: "أنمي" },
  { id: "documentary", label: "وثائقيات" },
  { id: "play", label: "مسرحيات" },
];

export const qualityOptions = [
  { id: "all", label: "كل الجودات" },
  { id: "4K", label: "4K" },
  { id: "1440p", label: "1440p" },
  { id: "1080p", label: "1080p" },
  { id: "720p", label: "720p" },
];

export const yearOptions = [
  { id: "all", label: "كل السنوات" },
  { id: "2025+", label: "2025 وما بعد" },
  { id: "2020-2024", label: "2020 - 2024" },
  { id: "2010-2019", label: "2010 - 2019" },
  { id: "2000-2009", label: "2000 - 2009" },
  { id: "classic", label: "قبل 2000" },
];

export const ratingOptions = [
  { id: "all", label: "كل التقييمات" },
  { id: "8", label: "8 فأعلى" },
  { id: "7", label: "7 فأعلى" },
  { id: "6", label: "6 فأعلى" },
  { id: "5", label: "5 فأعلى" },
];

export default function FilterToolbar({
  activeOrigin = "all",
  onSelectOrigin,
  activeGenre = "all",
  onSelectGenre,
  activeType = "all",
  onSelectType,
  activeQuality = "all",
  onSelectQuality,
  activeYear = "all",
  onSelectYear,
  activeRating = "all",
  onSelectRating,
  hasArabicAudio = false,
  onToggleArabicAudio,
  hasArabicSubtitles = false,
  onToggleArabicSubtitles,
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
  const [openPanel, setOpenPanel] = useState(null);
  const toolbarRef = useRef(null);
  const mobileOverlayRef = useRef(null);

  useEffect(() => {
    function closeOnOutside(event) {
      const clickedToolbar = toolbarRef.current?.contains(event.target);
      const clickedMobileSheet = mobileOverlayRef.current?.contains(event.target);
      if (!clickedToolbar && !clickedMobileSheet) setOpenPanel(null);
    }
    document.addEventListener("mousedown", closeOnOutside);
    return () => document.removeEventListener("mousedown", closeOnOutside);
  }, []);

  useEffect(() => {
    const closeOnEscape = (event) => {
      if (event.key === "Escape") setOpenPanel(null);
    };
    document.addEventListener("keydown", closeOnEscape);
    return () => document.removeEventListener("keydown", closeOnEscape);
  }, []);

  // The filter sheet is rendered at the document root on mobile, so it cannot
  // be clipped or sit behind the navigation drawer.
  const closePanel = () => setOpenPanel(null);

  const hasActiveFilters =
    (activeOrigin && activeOrigin !== "all") ||
    (activeGenre && activeGenre !== "all") ||
    (activeType && activeType !== "all") ||
    (activeQuality && activeQuality !== "all") ||
    (activeYear && activeYear !== "all") ||
    (activeRating && activeRating !== "all") ||
    hasArabicAudio || hasArabicSubtitles ||
    Boolean(searchQuery && searchQuery.trim());

  const handleReset = () => {
    if (onResetFilters) {
      onResetFilters();
    } else {
      if (onSelectOrigin) onSelectOrigin("all");
      if (onSelectGenre) onSelectGenre("all");
      if (onSelectType) onSelectType("all");
      if (onSelectQuality) onSelectQuality("all");
      if (onSelectYear) onSelectYear("all");
      if (onSelectRating) onSelectRating("all");
      if (onToggleArabicAudio && hasArabicAudio) onToggleArabicAudio(false);
      if (onToggleArabicSubtitles && hasArabicSubtitles) onToggleArabicSubtitles(false);
      if (onSearchChange) onSearchChange("");
    }
  };

  const selectedOrigin = origins.find((item) => item.id === activeOrigin);
  const selectedGenre = genres.find((item) => item.id === activeGenre);
  const selectedType = typeOptions.find((item) => item.id === activeType);
  const selectedQuality = qualityOptions.find((item) => item.id === activeQuality);
  const selectedYear = yearOptions.find((item) => item.id === activeYear);
  const selectedRating = ratingOptions.find((item) => item.id === activeRating);
  const selectedSort = sorts.find((item) => item.id === activeSort);
  const hasFilterOptions = showOriginFilter || showGenreFilter || onSelectType || onSelectQuality || onSelectYear || onSelectRating || onToggleArabicAudio || onToggleArabicSubtitles;

  const fieldClassName = "mt-1.5 h-11 w-full rounded-xl border border-[var(--border-default)] bg-[var(--bg-input)] px-3 text-sm font-bold text-[var(--text-primary)] outline-none transition focus:border-[var(--color-accent)] focus:ring-2 focus:ring-[var(--color-accent-light)]";
  const filterFields = <>
    {showOriginFilter && <label className="block text-xs font-bold text-[var(--text-secondary)]">المنشأ<select value={activeOrigin} onChange={(event) => onSelectOrigin && onSelectOrigin(event.target.value)} className={fieldClassName}>{origins.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}</select></label>}
    {showGenreFilter && <label className="block text-xs font-bold text-[var(--text-secondary)]">التصنيف<select value={activeGenre} onChange={(event) => onSelectGenre && onSelectGenre(event.target.value)} className={fieldClassName}>{genres.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}</select></label>}
    {onSelectType && <label className="block text-xs font-bold text-[var(--text-secondary)]">نوع العمل<select value={activeType} onChange={(event) => onSelectType(event.target.value)} className={fieldClassName}>{typeOptions.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}</select></label>}
    {onSelectQuality && <label className="block text-xs font-bold text-[var(--text-secondary)]">الجودة<select value={activeQuality} onChange={(event) => onSelectQuality(event.target.value)} className={fieldClassName}>{qualityOptions.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}</select></label>}
    {onSelectYear && <label className="block text-xs font-bold text-[var(--text-secondary)]">سنة الإنتاج<select value={activeYear} onChange={(event) => onSelectYear(event.target.value)} className={fieldClassName}>{yearOptions.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}</select></label>}
    {onSelectRating && <label className="block text-xs font-bold text-[var(--text-secondary)]">التقييم<select value={activeRating} onChange={(event) => onSelectRating(event.target.value)} className={fieldClassName}>{ratingOptions.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}</select></label>}
  </>;

  const filterPanel = <div className="max-h-[82dvh] overflow-y-auto rounded-t-[1.75rem] border border-[var(--border-default)] bg-[var(--bg-card)] p-4 pb-[max(1rem,env(safe-area-inset-bottom))] text-right shadow-[var(--shadow-xl)] sm:max-h-none sm:overflow-visible sm:rounded-2xl sm:p-3" dir="rtl">
    <div className="mx-auto mb-3 h-1 w-10 rounded-full bg-[var(--border-default)] sm:hidden" />
    <div className="mb-3 flex items-center justify-between gap-3"><div><strong className="block text-sm text-[var(--text-primary)]">خيارات الفلترة</strong><span className="block pt-0.5 text-[10px] text-[var(--text-muted)]">خصّص نتائج هذا القسم بسهولة</span></div><button type="button" onClick={closePanel} className="flex h-9 w-9 items-center justify-center rounded-full border border-[var(--border-default)] bg-[var(--bg-elevated)] text-[var(--text-secondary)] sm:hidden" aria-label="إغلاق الفلترة"><Icon name="close" className="h-4 w-4" /></button></div>
    <div className="grid gap-3 sm:gap-2.5">{filterFields}</div>
    {(onToggleArabicAudio || onToggleArabicSubtitles) && <div className="mt-4 grid gap-2 sm:grid-cols-2">{onToggleArabicAudio && <label className="flex min-h-11 cursor-pointer items-center gap-2 rounded-xl border border-[var(--border-default)] bg-[var(--bg-input)] px-3 text-xs font-bold text-[var(--text-secondary)]"><input type="checkbox" checked={hasArabicAudio} onChange={(event) => onToggleArabicAudio(event.target.checked)} className="h-4 w-4 accent-[var(--color-accent)]" />صوت عربي</label>}{onToggleArabicSubtitles && <label className="flex min-h-11 cursor-pointer items-center gap-2 rounded-xl border border-[var(--border-default)] bg-[var(--bg-input)] px-3 text-xs font-bold text-[var(--text-secondary)]"><input type="checkbox" checked={hasArabicSubtitles} onChange={(event) => onToggleArabicSubtitles(event.target.checked)} className="h-4 w-4 accent-[var(--color-accent)]" />ترجمة عربية</label>}</div>}
    <div className="mt-4 grid grid-cols-2 gap-2 sm:hidden"><button type="button" onClick={handleReset} className="min-h-11 rounded-xl border border-[var(--border-default)] text-xs font-bold text-[var(--text-secondary)]">إعادة تعيين</button><button type="button" onClick={closePanel} className="min-h-11 rounded-xl bg-[var(--color-accent)] px-4 text-xs font-bold text-white shadow-sm">عرض النتائج{resultCount ? ` (${resultCount})` : ""}</button></div>
  </div>;

  const mobileOverlay = openPanel === "filters" && createPortal(<div ref={mobileOverlayRef} className="fixed inset-0 z-[100] flex items-end bg-black/55 backdrop-blur-sm sm:hidden" role="dialog" aria-modal="true" aria-label="خيارات الفلترة"><button type="button" className="absolute inset-0 cursor-default" onClick={closePanel} aria-label="إغلاق الفلترة" /> <div className="relative w-full">{filterPanel}</div></div>, document.body);

  return (
    <div ref={toolbarRef} className="luminous-container relative z-50 rounded-2xl bg-[var(--bg-card)] p-2.5 shadow-[var(--shadow-md)] backdrop-blur-2xl sm:p-3" dir="rtl">
      <div className="flex items-center gap-2">
        {showSearch && <div className="min-w-0 flex-1">
          <div className="flex h-11 w-full items-center gap-2 rounded-full border border-[var(--border-default)] bg-[var(--bg-input)] px-4 text-[var(--text-primary)] transition focus-within:border-[var(--color-accent)] focus-within:ring-2 focus-within:ring-[var(--color-accent-light)]">
            <Icon name="search" className="h-4 w-4 shrink-0 text-[var(--color-accent)]" />
            <input type="text" value={searchQuery} onChange={(event) => onSearchChange && onSearchChange(event.target.value)} placeholder="بحث في هذا القسم..." className="w-full min-w-0 bg-transparent text-xs font-semibold text-[var(--text-primary)] outline-none placeholder:text-[var(--text-muted)] sm:text-sm" />
            {searchQuery && <button type="button" onClick={() => onSearchChange && onSearchChange("")} className="flex h-6 w-6 shrink-0 items-center justify-center rounded-lg text-[var(--text-muted)] transition hover:bg-[var(--bg-elevated)] hover:text-[var(--text-primary)]" title="مسح البحث" aria-label="مسح البحث"><Icon name="close" className="h-3.5 w-3.5" /></button>}
          </div>
        </div>}

        <div className="flex shrink-0 items-center gap-2">
          {hasFilterOptions && <div className="relative">
            <button type="button" onClick={() => setOpenPanel((value) => value === "filters" ? null : "filters")} className={`relative inline-flex h-11 w-11 items-center justify-center gap-2 rounded-full border text-xs font-bold transition sm:w-auto sm:px-3.5 ${openPanel === "filters" || hasActiveFilters ? "border-[var(--color-accent)] bg-[var(--color-accent-light)] text-[var(--color-accent)]" : "border-[var(--border-default)] bg-[var(--bg-surface)] text-[var(--text-secondary)] hover:border-[var(--color-accent)] hover:text-[var(--text-primary)]"}`} title="فتح الفلترة" aria-label="فتح الفلترة"><Icon name="filter" className="h-4 w-4" /><span className="hidden sm:inline">فلترة</span>{hasActiveFilters && <span className="absolute -right-0.5 -top-0.5 h-2 w-2 rounded-full bg-[var(--color-accent)] ring-2 ring-[var(--bg-card)]" />}</button>
            {openPanel === "filters" && <div className="absolute left-1/2 top-[calc(100%+0.6rem)] z-[70] hidden w-[min(18rem,calc(100vw-1rem))] -translate-x-1/2 sm:block">{filterPanel}</div>}
          </div>}

          {showSort && <div className="relative">
            <button type="button" onClick={() => setOpenPanel((value) => value === "sort" ? null : "sort")} className={`inline-flex h-11 w-11 items-center justify-center gap-2 rounded-full border text-xs font-bold transition sm:w-auto sm:px-3.5 ${openPanel === "sort" ? "border-[var(--color-accent)] bg-[var(--color-accent-light)] text-[var(--color-accent)]" : "border-[var(--border-default)] bg-[var(--bg-surface)] text-[var(--text-secondary)] hover:border-[var(--color-accent)] hover:text-[var(--text-primary)]"}`} title="فتح الفرز" aria-label="فتح الفرز"><Icon name="sort" className="h-4 w-4" /><span className="hidden sm:inline">فرز</span></button>
            {openPanel === "sort" && <div className="absolute left-1/2 top-[calc(100%+0.6rem)] z-[70] w-[min(18rem,calc(100vw-1rem))] -translate-x-1/2 rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-3 shadow-[var(--shadow-xl)]"><p className="px-2 py-2 text-xs font-bold text-[var(--text-muted)]">ترتيب الأعمال</p>{sorts.map((option) => <button type="button" key={option.id} onClick={() => { onSelectSort && onSelectSort(option.id); setOpenPanel(null); }} className={`flex min-h-11 w-full items-center justify-between rounded-full px-4 text-right text-sm font-bold transition ${activeSort === option.id ? "bg-[var(--color-accent-light)] text-[var(--color-accent)]" : "text-[var(--text-secondary)] hover:bg-[var(--bg-elevated)] hover:text-[var(--text-primary)]"}`}><span>{option.label}</span>{activeSort === option.id && <span>✓</span>}</button>)}</div>}
          </div>}

          {hasActiveFilters && <button type="button" onClick={handleReset} className="flex h-11 w-11 items-center justify-center rounded-full border border-[var(--color-accent)] bg-[var(--color-accent-light)] text-[var(--color-accent)] transition hover:bg-[var(--color-accent)] hover:text-white" title="إعادة تعيين الفلاتر" aria-label="إعادة تعيين الفلاتر"><Icon name="close" className="h-4 w-4" /></button>}
        </div>
      </div>
      {selectedSort && <p className="mt-2 px-1 text-[10px] text-[var(--text-muted)] sm:hidden">الفرز: <span className="font-bold text-[var(--text-secondary)]">{selectedSort.label}</span></p>}
      {mobileOverlay}
    </div>
  );
}
