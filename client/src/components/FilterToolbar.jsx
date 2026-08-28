import React, { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import Icon from "./Icon.jsx";

export const defaultOrigins = [
  { id: "all", label: "جميع البلدان" },
  { id: "تركي", label: "👑 تركي" },
  { id: "عربي", label: "🌟 عربي" },
  { id: "أجنبي", label: "🎬 أجنبي" },
  { id: "كوري", label: "🌸 كوري" },
  { id: "هندي", label: "⚡ هندي" },
  { id: "ياباني", label: "⚔️ ياباني" },
];

export const defaultGenres = [
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
  { id: "rating", label: "الأعلى تقييماً (★)" },
  { id: "rating_low", label: "الأقل تقييماً" },
  { id: "year", label: "سنة الإنتاج (الأحدث)" },
  { id: "year_old", label: "سنة الإنتاج (الأقدم)" },
  { id: "title", label: "الاسم (أ - ي)" },
  { id: "title_desc", label: "الاسم (ي - أ)" },
  { id: "files", label: "الأكثر ملفات وحلقات" },
  { id: "runtime", label: "الأطول مدة" },
];

export const defaultTypeOptions = [
  { id: "all", label: "كل الأنواع" },
  { id: "movie", label: "أفلام" },
  { id: "series", label: "مسلسلات" },
  { id: "anime", label: "أنمي" },
  { id: "documentary", label: "وثائقيات" },
  { id: "play", label: "مسرحيات" },
];

export const qualityOptions = [
  { id: "all", label: "كل الجودات" },
  { id: "4K", label: "4K UHD فائقة" },
  { id: "1440p", label: "1440p QHD" },
  { id: "1080p", label: "1080p FHD عالية" },
  { id: "720p", label: "720p HD متوسطة" },
];

export const yearOptions = [
  { id: "all", label: "كل السنوات" },
  { id: "2025+", label: "2025 وما بعد (حديث جداً)" },
  { id: "2024", label: "2024" },
  { id: "2023", label: "2023" },
  { id: "2020-2024", label: "2020 - 2024" },
  { id: "2010-2019", label: "2010 - 2019" },
  { id: "2000-2009", label: "2000 - 2009" },
  { id: "classic", label: "كلاسيكيات (قبل 2000)" },
];

export const ratingOptions = [
  { id: "all", label: "كل التقييمات" },
  { id: "9", label: "★ 9.0 فما فوق (تحف فنية)" },
  { id: "8", label: "★ 8.0 فما فوق (ممتاز جداً)" },
  { id: "7", label: "★ 7.0 فما فوق (جيد)" },
  { id: "6", label: "★ 6.0 فما فوق" },
];

export const contentRatingOptions = [
  { id: "all", label: "كافة الفئات العمرية" },
  { id: "G", label: "G (لجميع الأعمار)" },
  { id: "PG", label: "PG (إرشاد عائلي للأطفال)" },
  { id: "PG-13", label: "PG-13 (فوق 13 سنة)" },
  { id: "R", label: "R (للبالغين +17)" },
  { id: "NC-17", label: "NC-17 (للبالغين فقط)" },
  { id: "TV-Y", label: "TV-Y (للأطفال الصغار)" },
  { id: "TV-Y7", label: "TV-Y7 (أطفال +7)" },
  { id: "TV-G", label: "TV-G (عام تلفزيوني)" },
  { id: "TV-PG", label: "TV-PG (توجيه عائلي)" },
  { id: "TV-14", label: "TV-14 (فوق 14 سنة)" },
  { id: "TV-MA", label: "TV-MA (للبالغين فقط)" },
  { id: "family", label: "مناسب للعائلة (G / PG / TV-G)" },
  { id: "teen", label: "إشراف عائلي (+13 / PG-13 / TV-14)" },
  { id: "mature", label: "للبالغين (+18 / R / TV-MA)" },
];

export default function FilterToolbar({
  // Filter state values
  activeOrigin = "all",
  onSelectOrigin,
  activeGenre = "all",
  onSelectGenre,
  activeType = "all",
  onSelectType,
  activeFormat = "all",
  onSelectFormat,
  activeStatus = "all",
  onSelectStatus,
  activeSeasons = "all",
  onSelectSeasons,
  activeStudio = "all",
  onSelectStudio,
  activeTopic = "all",
  onSelectTopic,
  activePromotion = "all",
  onSelectPromotion,
  activeSeasonYear = "all",
  onSelectSeasonYear,
  activeEra = "all",
  onSelectEra,
  activeDuration = "all",
  onSelectDuration,
  activeAudioDub = "all",
  onSelectAudioDub,
  activeQuality = "all",
  onSelectQuality,
  activeYear = "all",
  onSelectYear,
  activeRating = "all",
  onSelectRating,
  activeContentRating = "all",
  onSelectContentRating,
  hasArabicAudio = false,
  onToggleArabicAudio,
  hasArabicSubtitles = false,
  onToggleArabicSubtitles,

  // Sort & Search
  activeSort = "newest",
  onSelectSort,
  searchQuery = "",
  onSearchChange,

  // Options lists (Dynamic from CategoryConfig or defaults)
  origins = defaultOrigins,
  genres = defaultGenres,
  types = defaultTypeOptions,
  formats = null,
  statuses = null,
  seasons = null,
  studios = null,
  topics = null,
  promotions = null,
  seasonYears = null,
  eras = null,
  durations = null,
  audioDubs = null,
  contentRatings = contentRatingOptions,
  qualities = qualityOptions,
  years = yearOptions,
  ratings = ratingOptions,
  sorts = sortOptions,

  // Visibility Flags
  showOriginFilter = true,
  showGenreFilter = true,
  showTypeFilter = true,
  showSort = true,
  showSearch = true,
  resultCount = 0,
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

  const closePanel = () => setOpenPanel(null);

  // Determine active filters count and tags for the Quick Chip Bar
  const activeChips = [];
  if (showOriginFilter && activeOrigin && activeOrigin !== "all") {
    const item = origins.find((o) => o.id === activeOrigin);
    activeChips.push({ key: "origin", label: item?.label || activeOrigin, onClear: () => onSelectOrigin && onSelectOrigin("all") });
  }
  if (showGenreFilter && activeGenre && activeGenre !== "all") {
    const item = genres.find((g) => g.id === activeGenre);
    activeChips.push({ key: "genre", label: item?.label || activeGenre, onClear: () => onSelectGenre && onSelectGenre("all") });
  }
  if (showTypeFilter && activeType && activeType !== "all") {
    const item = types.find((t) => t.id === activeType);
    activeChips.push({ key: "type", label: item?.label || activeType, onClear: () => onSelectType && onSelectType("all") });
  }
  if (formats && activeFormat && activeFormat !== "all") {
    const item = formats.find((f) => f.id === activeFormat);
    activeChips.push({ key: "format", label: item?.label || activeFormat, onClear: () => onSelectFormat && onSelectFormat("all") });
  }
  if (statuses && activeStatus && activeStatus !== "all") {
    const item = statuses.find((s) => s.id === activeStatus);
    activeChips.push({ key: "status", label: item?.label || activeStatus, onClear: () => onSelectStatus && onSelectStatus("all") });
  }
  if (seasons && activeSeasons && activeSeasons !== "all") {
    const item = seasons.find((s) => s.id === activeSeasons);
    activeChips.push({ key: "seasons", label: item?.label || activeSeasons, onClear: () => onSelectSeasons && onSelectSeasons("all") });
  }
  if (studios && activeStudio && activeStudio !== "all") {
    const item = studios.find((s) => s.id === activeStudio);
    activeChips.push({ key: "studio", label: item?.label || activeStudio, onClear: () => onSelectStudio && onSelectStudio("all") });
  }
  if (topics && activeTopic && activeTopic !== "all") {
    const item = topics.find((t) => t.id === activeTopic);
    activeChips.push({ key: "topic", label: item?.label || activeTopic, onClear: () => onSelectTopic && onSelectTopic("all") });
  }
  if (promotions && activePromotion && activePromotion !== "all") {
    const item = promotions.find((p) => p.id === activePromotion);
    activeChips.push({ key: "promotion", label: item?.label || activePromotion, onClear: () => onSelectPromotion && onSelectPromotion("all") });
  }
  if (seasonYears && activeSeasonYear && activeSeasonYear !== "all") {
    const item = seasonYears.find((sy) => sy.id === activeSeasonYear);
    activeChips.push({ key: "seasonYear", label: item?.label || activeSeasonYear, onClear: () => onSelectSeasonYear && onSelectSeasonYear("all") });
  }
  if (eras && activeEra && activeEra !== "all") {
    const item = eras.find((e) => e.id === activeEra);
    activeChips.push({ key: "era", label: item?.label || activeEra, onClear: () => onSelectEra && onSelectEra("all") });
  }
  if (durations && activeDuration && activeDuration !== "all") {
    const item = durations.find((d) => d.id === activeDuration);
    activeChips.push({ key: "duration", label: item?.label || activeDuration, onClear: () => onSelectDuration && onSelectDuration("all") });
  }
  if (audioDubs && activeAudioDub && activeAudioDub !== "all") {
    const item = audioDubs.find((ad) => ad.id === activeAudioDub);
    activeChips.push({ key: "audioDub", label: item?.label || activeAudioDub, onClear: () => onSelectAudioDub && onSelectAudioDub("all") });
  }
  if (activeContentRating && activeContentRating !== "all") {
    const item = contentRatings.find((cr) => cr.id === activeContentRating);
    activeChips.push({ key: "contentRating", label: item?.label || activeContentRating, onClear: () => onSelectContentRating && onSelectContentRating("all") });
  }
  if (activeQuality && activeQuality !== "all") {
    const item = qualities.find((q) => q.id === activeQuality);
    activeChips.push({ key: "quality", label: item?.label || activeQuality, onClear: () => onSelectQuality && onSelectQuality("all") });
  }
  if (activeYear && activeYear !== "all") {
    const item = years.find((y) => y.id === activeYear);
    activeChips.push({ key: "year", label: item?.label || activeYear, onClear: () => onSelectYear && onSelectYear("all") });
  }
  if (activeRating && activeRating !== "all") {
    const item = ratings.find((r) => r.id === activeRating);
    activeChips.push({ key: "rating", label: item?.label || activeRating, onClear: () => onSelectRating && onSelectRating("all") });
  }
  if (hasArabicAudio) {
    activeChips.push({ key: "arabicAudio", label: "صوت عربي", onClear: () => onToggleArabicAudio && onToggleArabicAudio(false) });
  }
  if (hasArabicSubtitles) {
    activeChips.push({ key: "arabicSubs", label: "ترجمة عربية", onClear: () => onToggleArabicSubtitles && onToggleArabicSubtitles(false) });
  }

  const hasActiveFilters = activeChips.length > 0 || Boolean(searchQuery && searchQuery.trim());

  const handleReset = () => {
    if (onResetFilters) {
      onResetFilters();
    } else {
      if (onSelectOrigin) onSelectOrigin("all");
      if (onSelectGenre) onSelectGenre("all");
      if (onSelectType) onSelectType("all");
      if (onSelectFormat) onSelectFormat("all");
      if (onSelectStatus) onSelectStatus("all");
      if (onSelectSeasons) onSelectSeasons("all");
      if (onSelectStudio) onSelectStudio("all");
      if (onSelectTopic) onSelectTopic("all");
      if (onSelectPromotion) onSelectPromotion("all");
      if (onSelectSeasonYear) onSelectSeasonYear("all");
      if (onSelectEra) onSelectEra("all");
      if (onSelectDuration) onSelectDuration("all");
      if (onSelectAudioDub) onSelectAudioDub("all");
      if (onSelectQuality) onSelectQuality("all");
      if (onSelectYear) onSelectYear("all");
      if (onSelectRating) onSelectRating("all");
      if (onSelectContentRating) onSelectContentRating("all");
      if (onToggleArabicAudio && hasArabicAudio) onToggleArabicAudio(false);
      if (onToggleArabicSubtitles && hasArabicSubtitles) onToggleArabicSubtitles(false);
      if (onSearchChange) onSearchChange("");
    }
  };

  const fieldClassName =
    "mt-1.5 h-10 w-full rounded-xl border border-[var(--border-default)] bg-[var(--bg-input)] px-3 text-xs font-bold text-[var(--text-primary)] outline-none transition focus:border-[var(--color-accent)] focus:ring-2 focus:ring-[var(--color-accent-light)]";

  const filterFields = (
    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
      {/* 1. Origin Filter */}
      {showOriginFilter && origins && origins.length > 0 && onSelectOrigin && (
        <label className="block text-xs font-bold text-[var(--text-secondary)]">
          المنشأ والبلد
          <select value={activeOrigin} onChange={(e) => onSelectOrigin(e.target.value)} className={fieldClassName}>
            {origins.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      )}

      {/* 2. Genre Filter */}
      {showGenreFilter && genres && genres.length > 0 && onSelectGenre && (
        <label className="block text-xs font-bold text-[var(--text-secondary)]">
          التصنيف الفني
          <select value={activeGenre} onChange={(e) => onSelectGenre(e.target.value)} className={fieldClassName}>
            {genres.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      )}

      {/* 3. Format / Type */}
      {formats && onSelectFormat && (
        <label className="block text-xs font-bold text-[var(--text-secondary)]">
          تنسيق العمل
          <select value={activeFormat} onChange={(e) => onSelectFormat(e.target.value)} className={fieldClassName}>
            {formats.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      )}

      {showTypeFilter && types && onSelectType && (
        <label className="block text-xs font-bold text-[var(--text-secondary)]">
          نوع العمل
          <select value={activeType} onChange={(e) => onSelectType(e.target.value)} className={fieldClassName}>
            {types.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      )}

      {/* 4. Status / Seasons */}
      {statuses && onSelectStatus && (
        <label className="block text-xs font-bold text-[var(--text-secondary)]">
          حالة العرض
          <select value={activeStatus} onChange={(e) => onSelectStatus(e.target.value)} className={fieldClassName}>
            {statuses.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      )}

      {seasons && onSelectSeasons && (
        <label className="block text-xs font-bold text-[var(--text-secondary)]">
          عدد المواسم
          <select value={activeSeasons} onChange={(e) => onSelectSeasons(e.target.value)} className={fieldClassName}>
            {seasons.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      )}

      {/* 5. Studios (Kids / Cartoons) */}
      {studios && onSelectStudio && (
        <label className="block text-xs font-bold text-[var(--text-secondary)]">
          الاستوديو والعالم
          <select value={activeStudio} onChange={(e) => onSelectStudio(e.target.value)} className={fieldClassName}>
            {studios.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      )}

      {/* 6. Topics (Documentaries) */}
      {topics && onSelectTopic && (
        <label className="block text-xs font-bold text-[var(--text-secondary)]">
          موضوع الوثائقي
          <select value={activeTopic} onChange={(e) => onSelectTopic(e.target.value)} className={fieldClassName}>
            {topics.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      )}

      {/* 7. Promotions (Wrestling/Sports) */}
      {promotions && onSelectPromotion && (
        <label className="block text-xs font-bold text-[var(--text-secondary)]">
          الاتحاد والبطولة
          <select value={activePromotion} onChange={(e) => onSelectPromotion(e.target.value)} className={fieldClassName}>
            {promotions.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      )}

      {/* 8. Season Year (Ramadan) */}
      {seasonYears && onSelectSeasonYear && (
        <label className="block text-xs font-bold text-[var(--text-secondary)]">
          الموسم الرمضاني
          <select value={activeSeasonYear} onChange={(e) => onSelectSeasonYear(e.target.value)} className={fieldClassName}>
            {seasonYears.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      )}

      {/* 9. Eras (Plays) */}
      {eras && onSelectEra && (
        <label className="block text-xs font-bold text-[var(--text-secondary)]">
          الحقبة المسرحية
          <select value={activeEra} onChange={(e) => onSelectEra(e.target.value)} className={fieldClassName}>
            {eras.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      )}

      {/* 10. Duration (Movies) */}
      {durations && onSelectDuration && (
        <label className="block text-xs font-bold text-[var(--text-secondary)]">
          مدة العمل
          <select value={activeDuration} onChange={(e) => onSelectDuration(e.target.value)} className={fieldClassName}>
            {durations.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      )}

      {/* 11. Audio Dubs */}
      {audioDubs && onSelectAudioDub && (
        <label className="block text-xs font-bold text-[var(--text-secondary)]">
          الدبلجة والصوت
          <select value={activeAudioDub} onChange={(e) => onSelectAudioDub(e.target.value)} className={fieldClassName}>
            {audioDubs.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      )}

      {/* 12. Content Rating */}
      {onSelectContentRating && contentRatings && (
        <label className="block text-xs font-bold text-[var(--text-secondary)]">
          التصنيف العمري (Content Rating)
          <select value={activeContentRating} onChange={(e) => onSelectContentRating(e.target.value)} className={fieldClassName}>
            {contentRatings.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      )}

      {/* 13. Quality */}
      {onSelectQuality && (
        <label className="block text-xs font-bold text-[var(--text-secondary)]">
          الجودة والدقة
          <select value={activeQuality} onChange={(e) => onSelectQuality(e.target.value)} className={fieldClassName}>
            {qualities.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      )}

      {/* 14. Year */}
      {onSelectYear && (
        <label className="block text-xs font-bold text-[var(--text-secondary)]">
          سنة الإنتاج
          <select value={activeYear} onChange={(e) => onSelectYear(e.target.value)} className={fieldClassName}>
            {years.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      )}

      {/* 15. Rating */}
      {onSelectRating && (
        <label className="block text-xs font-bold text-[var(--text-secondary)]">
          التقييم الجماهيري
          <select value={activeRating} onChange={(e) => onSelectRating(e.target.value)} className={fieldClassName}>
            {ratings.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      )}
    </div>
  );

  const filterPanel = (
    <div
      className="max-h-[82dvh] overflow-y-auto rounded-t-[1.75rem] border border-[var(--border-default)] bg-[var(--bg-card)] p-5 pb-[max(1.25rem,env(safe-area-inset-bottom))] text-right shadow-[var(--shadow-xl)] sm:max-h-none sm:overflow-visible sm:rounded-3xl sm:p-5"
      dir="rtl"
    >
      <div className="mx-auto mb-3 h-1.5 w-12 rounded-full bg-[var(--border-default)] sm:hidden" />
      <div className="mb-4 flex items-center justify-between gap-3 border-b border-[var(--border-default)] pb-3">
        <div>
          <strong className="block text-sm sm:text-base font-black text-[var(--text-primary)]">
            خيارات الفلترة والتخصيص الذكية
          </strong>
          <span className="block pt-0.5 text-[11px] text-[var(--text-muted)]">
            فلاتر سياقية فائقة الدقة مهيأة خصيصاً لهذا القسم
          </span>
        </div>
        <button
          type="button"
          onClick={closePanel}
          className="flex h-9 w-9 items-center justify-center rounded-full border border-[var(--border-default)] bg-[var(--bg-elevated)] text-[var(--text-secondary)] transition hover:text-[var(--text-primary)]"
          aria-label="إغلاق الفلترة"
        >
          <Icon name="close" className="h-4 w-4" />
        </button>
      </div>

      <div className="space-y-4">
        {filterFields}

        {/* Audio & Subtitle Quick Toggles */}
        {(onToggleArabicAudio || onToggleArabicSubtitles) && (
          <div className="mt-4 pt-3 border-t border-[var(--border-default)] grid grid-cols-2 gap-2">
            {onToggleArabicAudio && (
              <label className="flex min-h-11 cursor-pointer items-center justify-center gap-2 rounded-xl border border-[var(--border-default)] bg-[var(--bg-input)] px-3 text-xs font-bold text-[var(--text-secondary)] hover:border-[var(--color-accent)] transition">
                <input
                  type="checkbox"
                  checked={hasArabicAudio}
                  onChange={(e) => onToggleArabicAudio(e.target.checked)}
                  className="h-4 w-4 rounded accent-[var(--color-accent)]"
                />
                <span>🎙️ صوت ودبلجة عربية</span>
              </label>
            )}
            {onToggleArabicSubtitles && (
              <label className="flex min-h-11 cursor-pointer items-center justify-center gap-2 rounded-xl border border-[var(--border-default)] bg-[var(--bg-input)] px-3 text-xs font-bold text-[var(--text-secondary)] hover:border-[var(--color-accent)] transition">
                <input
                  type="checkbox"
                  checked={hasArabicSubtitles}
                  onChange={(e) => onToggleArabicSubtitles(e.target.checked)}
                  className="h-4 w-4 rounded accent-[var(--color-accent)]"
                />
                <span>📝 ترجمة عربية</span>
              </label>
            )}
          </div>
        )}

        <div className="mt-5 flex items-center justify-between gap-3 border-t border-[var(--border-default)] pt-4">
          <button
            type="button"
            onClick={handleReset}
            className="flex-1 min-h-11 rounded-xl border border-[var(--border-default)] bg-[var(--bg-surface)] text-xs font-bold text-[var(--text-secondary)] hover:bg-[var(--bg-elevated)] transition"
          >
            إعادة تعيين الفلاتر
          </button>
          <button
            type="button"
            onClick={closePanel}
            className="flex-1 min-h-11 rounded-xl bg-gradient-to-r from-fuchsia-600 to-indigo-600 px-4 text-xs font-bold text-white shadow-md hover:opacity-95 transition"
          >
            عرض النتائج {resultCount ? `(${resultCount})` : ""}
          </button>
        </div>
      </div>
    </div>
  );

  const mobileOverlay =
    openPanel === "filters" &&
    createPortal(
      <div
        ref={mobileOverlayRef}
        className="fixed inset-0 z-[100] flex items-end bg-black/60 backdrop-blur-sm sm:hidden"
        role="dialog"
        aria-modal="true"
        aria-label="خيارات الفلترة"
      >
        <button type="button" className="absolute inset-0 cursor-default" onClick={closePanel} aria-label="إغلاق الفلترة" />
        <div className="relative w-full">{filterPanel}</div>
      </div>,
      document.body
    );

  const selectedSort = sorts.find((item) => item.id === activeSort);

  return (
    <div
      ref={toolbarRef}
      className="luminous-container relative z-40 rounded-3xl bg-[var(--bg-card)]/90 p-3 sm:p-4 border border-[var(--border-default)] shadow-[var(--shadow-lg)] backdrop-blur-2xl transition space-y-3"
      dir="rtl"
    >
      {/* 1. Main Search & Action Bar */}
      <div className="flex items-center gap-2 sm:gap-3">
        {showSearch && (
          <div className="min-w-0 flex-1">
            <div className="flex h-11 sm:h-12 w-full items-center gap-2.5 rounded-2xl border border-[var(--border-default)] bg-[var(--bg-input)] px-4 text-[var(--text-primary)] transition focus-within:border-[var(--color-accent)] focus-within:ring-2 focus-within:ring-[var(--color-accent-light)]">
              <Icon name="search" className="h-4 w-4 shrink-0 text-[var(--color-accent)]" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => onSearchChange && onSearchChange(e.target.value)}
                placeholder="بحث فوري في هذا القسم بالعنوان، القصة، أو الكلمات المفتاحية..."
                className="w-full min-w-0 bg-transparent text-xs sm:text-sm font-semibold text-[var(--text-primary)] outline-none placeholder:text-[var(--text-muted)]"
              />
              {searchQuery && (
                <button
                  type="button"
                  onClick={() => onSearchChange && onSearchChange("")}
                  className="flex h-6 w-6 shrink-0 items-center justify-center rounded-lg text-[var(--text-muted)] transition hover:bg-[var(--bg-elevated)] hover:text-[var(--text-primary)]"
                  title="مسح البحث"
                  aria-label="مسح البحث"
                >
                  <Icon name="close" className="h-3.5 w-3.5" />
                </button>
              )}
            </div>
          </div>
        )}

        <div className="flex shrink-0 items-center gap-2">
          {/* Filters Toggle Button */}
          <div className="relative">
            <button
              type="button"
              onClick={() => setOpenPanel((val) => (val === "filters" ? null : "filters"))}
              className={`relative inline-flex h-11 sm:h-12 items-center justify-center gap-2 rounded-2xl border px-4 text-xs sm:text-sm font-black transition ${
                openPanel === "filters" || hasActiveFilters
                  ? "border-fuchsia-500 bg-fuchsia-500/15 text-fuchsia-400 shadow-[0_0_15px_rgba(217,70,239,0.25)]"
                  : "border-[var(--border-default)] bg-[var(--bg-surface)] text-[var(--text-secondary)] hover:border-fuchsia-500/50 hover:text-[var(--text-primary)]"
              }`}
              title="خيارات الفلترة"
              aria-label="خيارات الفلترة"
            >
              <Icon name="filter" className="h-4 w-4 text-fuchsia-400" />
              <span className="hidden sm:inline">فلترة متقدمة</span>
              {activeChips.length > 0 && (
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-fuchsia-600 text-[10px] font-black text-white">
                  {activeChips.length}
                </span>
              )}
            </button>

            {/* Desktop Filter Popover */}
            {openPanel === "filters" && (
              <div className="absolute left-0 top-[calc(100%+0.6rem)] z-[70] hidden w-[min(38rem,calc(100vw-2rem))] sm:block">
                {filterPanel}
              </div>
            )}
          </div>

          {/* Sort Button */}
          {showSort && (
            <div className="relative">
              <button
                type="button"
                onClick={() => setOpenPanel((val) => (val === "sort" ? null : "sort"))}
                className={`inline-flex h-11 sm:h-12 items-center justify-center gap-2 rounded-2xl border px-3.5 text-xs sm:text-sm font-bold transition ${
                  openPanel === "sort"
                    ? "border-[var(--color-accent)] bg-[var(--color-accent-light)] text-[var(--color-accent)]"
                    : "border-[var(--border-default)] bg-[var(--bg-surface)] text-[var(--text-secondary)] hover:border-[var(--color-accent)] hover:text-[var(--text-primary)]"
                }`}
                title="ترتيب النتائج"
                aria-label="ترتيب النتائج"
              >
                <Icon name="sort" className="h-4 w-4" />
                <span className="hidden md:inline">{selectedSort ? selectedSort.label : "فرز"}</span>
              </button>

              {/* Sort Dropdown Popover */}
              {openPanel === "sort" && (
                <div className="absolute left-0 top-[calc(100%+0.6rem)] z-[70] w-64 rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-2.5 shadow-[var(--shadow-xl)] backdrop-blur-2xl">
                  <p className="px-3 py-1.5 text-[11px] font-black text-[var(--text-muted)] border-b border-[var(--border-default)] mb-1">
                    ترتيب الأعمال حسب
                  </p>
                  <div className="max-h-72 overflow-y-auto space-y-1">
                    {sorts.map((option) => (
                      <button
                        type="button"
                        key={option.id}
                        onClick={() => {
                          onSelectSort && onSelectSort(option.id);
                          setOpenPanel(null);
                        }}
                        className={`flex w-full items-center justify-between rounded-xl px-3 py-2 text-right text-xs font-bold transition ${
                          activeSort === option.id
                            ? "bg-fuchsia-500/20 text-fuchsia-400 font-black"
                            : "text-[var(--text-secondary)] hover:bg-[var(--bg-elevated)] hover:text-[var(--text-primary)]"
                        }`}
                      >
                        <span>{option.label}</span>
                        {activeSort === option.id && <span className="text-fuchsia-400">✓</span>}
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Quick Clear All Button */}
          {hasActiveFilters && (
            <button
              type="button"
              onClick={handleReset}
              className="flex h-11 sm:h-12 w-11 sm:w-12 items-center justify-center rounded-2xl border border-rose-500/30 bg-rose-500/10 text-rose-400 transition hover:bg-rose-500 hover:text-white"
              title="مسح جميع الفلاتر"
              aria-label="مسح جميع الفلاتر"
            >
              <Icon name="close" className="h-4 w-4" />
            </button>
          )}
        </div>
      </div>

      {/* 2. Active Filter Badges Bar (Chips) */}
      {activeChips.length > 0 && (
        <div className="flex flex-wrap items-center gap-1.5 pt-1 text-xs">
          <span className="text-[11px] font-bold text-[var(--text-muted)] pl-1">الفلاتر النشطة:</span>
          {activeChips.map((chip) => (
            <span
              key={chip.key}
              className="inline-flex items-center gap-1.5 rounded-xl border border-fuchsia-500/40 bg-fuchsia-500/10 px-2.5 py-1 font-bold text-fuchsia-300 backdrop-blur-md"
            >
              <span>{chip.label}</span>
              <button
                type="button"
                onClick={chip.onClear}
                className="hover:text-white transition rounded-full p-0.5"
                title="إلغاء هذا الفلتر"
              >
                ✕
              </button>
            </span>
          ))}
          <button
            type="button"
            onClick={handleReset}
            className="text-[11px] font-bold text-rose-400 hover:text-rose-300 transition underline underline-offset-4 mr-2"
          >
            إعادة تعيين الكل
          </button>
        </div>
      )}

      {mobileOverlay}
    </div>
  );
}
