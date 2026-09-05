import React, { useEffect, useMemo, useState, useRef, useCallback } from "react";
import ShowcaseHero from "../components/ShowcaseHero.jsx";
import FilterToolbar from "../components/FilterToolbar.jsx";
import MediaCollection from "../components/MediaCollection.jsx";
import SmartHubRail from "../components/SmartHubRail.jsx";
import Icon from "../components/Icon.jsx";
import { getMediaList } from "../lib/api.js";
import { getCategoryConfig } from "../data/categoryConfig.js";
import { useNavigationState } from "../context/NavigationStateContext.jsx";
import { useScrollRestoration } from "../hooks/useScrollRestoration.js";

const PAGE_SIZE = 36;

export default function CategoryPage({ selectedCategory = "series", onOpenMedia, onQuickPlay }) {
  const { getPageState, savePageState } = useNavigationState();
  const cacheKey = `category:${selectedCategory}`;

  // Check if we have cached state from previous navigation
  const cachedState = useMemo(() => getPageState(cacheKey), [cacheKey, getPageState]);

  const [items, setItems] = useState(() => cachedState?.items || []);
  const [totalCount, setTotalCount] = useState(() => cachedState?.totalCount || 0);
  const [loading, setLoading] = useState(() => !cachedState?.items?.length);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(() => (cachedState ? Boolean(cachedState.hasMore) : true));

  // Context-Aware Filter States (restored from cache if available)
  const [activeOrigin, setActiveOrigin] = useState(() => cachedState?.filters?.origin || "all");
  const [activeGenre, setActiveGenre] = useState(() => cachedState?.filters?.genre || "all");
  const [activeType, setActiveType] = useState(() => cachedState?.filters?.type || "all");
  const [activeFormat, setActiveFormat] = useState(() => cachedState?.filters?.format || "all");
  const [activeStatus, setActiveStatus] = useState(() => cachedState?.filters?.status || "all");
  const [activeSeasons, setActiveSeasons] = useState(() => cachedState?.filters?.seasons || "all");
  const [activeStudio, setActiveStudio] = useState(() => cachedState?.filters?.studio || "all");
  const [activeTopic, setActiveTopic] = useState(() => cachedState?.filters?.topic || "all");
  const [activePromotion, setActivePromotion] = useState(() => cachedState?.filters?.promotion || "all");
  const [activeSeasonYear, setActiveSeasonYear] = useState(() => cachedState?.filters?.seasonYear || "all");
  const [activeEra, setActiveEra] = useState(() => cachedState?.filters?.era || "all");
  const [activeDuration, setActiveDuration] = useState(() => cachedState?.filters?.duration || "all");
  const [activeAudioDub, setActiveAudioDub] = useState(() => cachedState?.filters?.audioDub || "all");
  const [activeQuality, setActiveQuality] = useState(() => cachedState?.filters?.quality || "all");
  const [activeYear, setActiveYear] = useState(() => cachedState?.filters?.year || "all");
  const [activeRating, setActiveRating] = useState(() => cachedState?.filters?.rating || "all");
  const [activeContentRating, setActiveContentRating] = useState(() => cachedState?.filters?.contentRating || "all");
  const [hasArabicAudio, setHasArabicAudio] = useState(() => cachedState?.filters?.hasArabicAudio || false);
  const [hasArabicSubtitles, setHasArabicSubtitles] = useState(() => cachedState?.filters?.hasArabicSubtitles || false);
  const [activeSort, setActiveSort] = useState(() => cachedState?.sort || "newest");
  const [searchQuery, setSearchQuery] = useState(() => cachedState?.searchQuery || "");

  // Intersection Observer Target for Infinite Scroll
  const observerTargetRef = useRef(null);
  const isFetchingRef = useRef(false);

  // Category Configuration
  const categoryConfig = useMemo(() => getCategoryConfig(selectedCategory), [selectedCategory]);
  const filterConfig = categoryConfig.filterConfig || {};

  // Current filters bundle
  const currentFilters = useMemo(
    () => ({
      origin: activeOrigin,
      genre: activeGenre,
      type: activeType,
      format: activeFormat,
      status: activeStatus,
      seasons: activeSeasons,
      studio: activeStudio,
      topic: activeTopic,
      promotion: activePromotion,
      seasonYear: activeSeasonYear,
      era: activeEra,
      duration: activeDuration,
      audioDub: activeAudioDub,
      quality: activeQuality,
      year: activeYear,
      rating: activeRating,
      contentRating: activeContentRating,
      hasArabicAudio,
      hasArabicSubtitles,
    }),
    [
      activeOrigin,
      activeGenre,
      activeType,
      activeFormat,
      activeStatus,
      activeSeasons,
      activeStudio,
      activeTopic,
      activePromotion,
      activeSeasonYear,
      activeEra,
      activeDuration,
      activeAudioDub,
      activeQuality,
      activeYear,
      activeRating,
      activeContentRating,
      hasArabicAudio,
      hasArabicSubtitles,
    ]
  );

  // Use Scroll Restoration hook
  useScrollRestoration(cacheKey, !loading, {
    items,
    totalCount,
    filters: currentFilters,
    sort: activeSort,
    searchQuery,
    hasMore,
  });

  const transformRawItem = useCallback((item) => ({
    id: item.id,
    titleAr: item.title_ar,
    titleEn: item.title_en,
    type: item.type,
    plot: item.plot_ar || item.plot_en || "",
    year: item.release_year,
    rating: item.rating,
    contentRating: item.content_rating || item.contentRating || "",
    posterPath: item.poster_path,
    bannerPath: item.banner_path,
    categorySlug: item.category_slug || selectedCategory,
    fileCount: item.file_count,
    status: item.status,
    seasonCount: item.season_count || (item.seasons ? item.seasons.length : 1),
    tmdbSeasonCount: item.tmdb_season_count,
    tmdbEpisodeCount: item.tmdb_episode_count,
    totalSize: item.total_size,
    bestResolution: item.best_resolution,
    runtimeMinutes: item.runtime_minutes || 0,
    hasArabicAudio: item.has_arabic_audio,
    hasArabicSubtitles: item.has_arabic_subtitles,
    genres: item.genres || [],
  }), [selectedCategory]);

  const loadInitialItems = useCallback(async () => {
    if (isFetchingRef.current) return;
    isFetchingRef.current = true;
    setLoading(true);

    try {
      const fetchParams = {
        sort:
          activeSort === "rating"
            ? "rating"
            : activeSort === "year"
            ? "year"
            : activeSort === "title"
            ? "title"
            : "",
        limit: PAGE_SIZE,
        offset: 0,
      };

      if (selectedCategory === "movies") {
        fetchParams.type = "movie";
      } else if (selectedCategory === "series") {
        fetchParams.type = "series";
      } else if (["family", "kids", "anime"].includes(selectedCategory)) {
        // Cross-category
      } else {
        fetchParams.category = selectedCategory;
      }

      const res = await getMediaList(fetchParams);
      const rawList = res?.items || [];
      const transformed = rawList.map(transformRawItem);
      const total = res?.total || transformed.length;

      setItems(transformed);
      setTotalCount(total);
      const canLoadMore = transformed.length < total && rawList.length >= PAGE_SIZE;
      setHasMore(canLoadMore);

      savePageState(cacheKey, {
        items: transformed,
        totalCount: total,
        filters: currentFilters,
        sort: activeSort,
        searchQuery,
        hasMore: canLoadMore,
      });
    } catch {
      setItems([]);
      setTotalCount(0);
      setHasMore(false);
    } finally {
      setLoading(false);
      isFetchingRef.current = false;
    }
  }, [activeSort, selectedCategory, transformRawItem, savePageState, cacheKey, currentFilters, searchQuery]);

  const loadMoreItems = useCallback(async () => {
    if (isFetchingRef.current || !hasMore || loading || loadingMore) return;
    isFetchingRef.current = true;
    setLoadingMore(true);

    try {
      const fetchParams = {
        sort:
          activeSort === "rating"
            ? "rating"
            : activeSort === "year"
            ? "year"
            : activeSort === "title"
            ? "title"
            : "",
        limit: PAGE_SIZE,
        offset: items.length,
      };

      if (selectedCategory === "movies") {
        fetchParams.type = "movie";
      } else if (selectedCategory === "series") {
        fetchParams.type = "series";
      } else if (["family", "kids", "anime"].includes(selectedCategory)) {
        // Cross-category
      } else {
        fetchParams.category = selectedCategory;
      }

      const res = await getMediaList(fetchParams);
      const rawList = res?.items || [];
      const transformed = rawList.map(transformRawItem);

      if (transformed.length === 0) {
        setHasMore(false);
      } else {
        setItems((prev) => {
          const existingIds = new Set(prev.map((x) => x.id));
          const newItems = transformed.filter((x) => !existingIds.has(x.id));
          const merged = [...prev, ...newItems];
          const canLoadMore = merged.length < (res?.total || totalCount) && transformed.length >= PAGE_SIZE;
          setHasMore(canLoadMore);

          savePageState(cacheKey, {
            items: merged,
            totalCount: res?.total || totalCount,
            filters: currentFilters,
            sort: activeSort,
            searchQuery,
            hasMore: canLoadMore,
          });

          return merged;
        });
      }
    } catch {
      setHasMore(false);
    } finally {
      setLoadingMore(false);
      isFetchingRef.current = false;
    }
  }, [hasMore, loading, loadingMore, activeSort, items.length, selectedCategory, transformRawItem, totalCount, savePageState, cacheKey, currentFilters, searchQuery]);

  // Load items when category or sort changes (unless restored from cache)
  useEffect(() => {
    const cached = getPageState(cacheKey);
    if (!cached || !cached.items?.length || cached.sort !== activeSort) {
      loadInitialItems();
    }
  }, [selectedCategory, activeSort, cacheKey, getPageState, loadInitialItems]);

  // Infinite Scroll Observer Setup
  useEffect(() => {
    const sentinel = observerTargetRef.current;
    if (!sentinel) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loading && !loadingMore) {
          loadMoreItems();
        }
      },
      { threshold: 0.1, rootMargin: "400px" }
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [hasMore, loading, loadingMore, loadMoreItems]);

  // 100% Context-Aware Multi-Dimensional Filtering Logic
  const filteredItems = useMemo(() => {
    return items.filter((item) => {
      const itemCat = (item.categorySlug || "").toLowerCase();
      const genresStr = (item.genres || []).join(" ").toLowerCase();
      const cr = String(item.contentRating || "").toUpperCase().trim();

      // 0. Multi-Indexing & Isolation Rules
      if (selectedCategory === "movies") {
        if (itemCat === "anime" || genresStr.includes("أنمي") || genresStr.includes("anime")) return false;
        if (item.type && item.type !== "movie") return false;
      } else if (selectedCategory === "series") {
        if (itemCat === "anime" || genresStr.includes("أنمي") || genresStr.includes("anime")) return false;
        if (item.type && item.type !== "series") return false;
      } else if (selectedCategory === "kids") {
        const isKids =
          itemCat === "kids" ||
          genresStr.includes("كرتون") ||
          genresStr.includes("رسوم متحركة") ||
          genresStr.includes("أطفال") ||
          genresStr.includes("ديزني") ||
          genresStr.includes("بيكسار") ||
          genresStr.includes("سبيستون") ||
          genresStr.includes("دريم وركس") ||
          genresStr.includes("animation");
        if (!isKids) return false;
        if (["R", "NC-17", "TV-MA", "18+", "18"].includes(cr)) return false;
      } else if (selectedCategory === "family") {
        const isAdult = ["R", "NC-17", "TV-MA", "18+", "18", "MA"].includes(cr);
        if (isAdult) return false;
        const isFamilySafe =
          ["G", "PG", "TV-G", "TV-Y", "TV-Y7", "TV-PG", "ALL"].includes(cr) ||
          genresStr.includes("عائلي") ||
          genresStr.includes("أطفال") ||
          genresStr.includes("كرتون") ||
          genresStr.includes("ديزني") ||
          genresStr.includes("بيكسار") ||
          genresStr.includes("family") ||
          genresStr.includes("animation");
        if (!isFamilySafe) return false;
      } else if (selectedCategory === "anime") {
        const isAnime = itemCat === "anime" || genresStr.includes("أنمي") || genresStr.includes("anime");
        if (!isAnime) return false;
      }

      // 1. Search Query
      if (searchQuery.trim()) {
        const query = searchQuery.trim().toLowerCase();
        const matchesTitle =
          (item.titleAr && item.titleAr.toLowerCase().includes(query)) ||
          (item.titleEn && item.titleEn.toLowerCase().includes(query)) ||
          (item.plot && item.plot.toLowerCase().includes(query));
        if (!matchesTitle) return false;
      }

      // 2. Audio & Subtitles Flags
      if (hasArabicAudio && !item.hasArabicAudio) return false;
      if (hasArabicSubtitles && !item.hasArabicSubtitles) return false;

      // 3. Year Filter
      if (activeYear !== "all") {
        const y = item.year;
        if (!y) return false;
        if (activeYear === "2026" && y !== 2026) return false;
        if (activeYear === "2025" && y !== 2025) return false;
        if (activeYear === "2024" && y !== 2024) return false;
        if (activeYear === "2023" && y !== 2023) return false;
        if (activeYear === "2020s" && (y < 2020 || y > 2029)) return false;
        if (activeYear === "2010s" && (y < 2010 || y > 2019)) return false;
        if (activeYear === "2000s" && (y < 2000 || y > 2009)) return false;
        if (activeYear === "1990s" && (y < 1990 || y > 1999)) return false;
        if (activeYear === "classic" && y >= 1990) return false;
      }

      // 4. Rating Filter
      if (activeRating !== "all") {
        const r = item.rating || 0;
        if (activeRating === "9+" && r < 9) return false;
        if (activeRating === "8+" && r < 8) return false;
        if (activeRating === "7+" && r < 7) return false;
        if (activeRating === "6+" && r < 6) return false;
      }

      // 5. Quality Filter
      if (activeQuality !== "all") {
        const res = (item.bestResolution || "").toUpperCase();
        if (activeQuality === "4k" && !res.includes("4K") && !res.includes("2160")) return false;
        if (activeQuality === "1080p" && !res.includes("1080")) return false;
        if (activeQuality === "720p" && !res.includes("720")) return false;
      }

      // 6. Content Rating
      if (activeContentRating !== "all") {
        if (cr !== activeContentRating.toUpperCase()) return false;
      }

      // 7. Dynamic Categorical Filters
      if (activeOrigin !== "all") {
        const originMatch =
          genresStr.includes(activeOrigin.toLowerCase()) ||
          (item.plot && item.plot.toLowerCase().includes(activeOrigin.toLowerCase())) ||
          (activeOrigin === "عربي" && (genresStr.includes("عربي") || genresStr.includes("arabic") || itemCat === "arabic")) ||
          (activeOrigin === "أجنبي" && (genresStr.includes("أجنبي") || genresStr.includes("foreign") || genresStr.includes("hollywood"))) ||
          (activeOrigin === "تركي" && (genresStr.includes("تركي") || genresStr.includes("turkish"))) ||
          (activeOrigin === "كوري" && (genresStr.includes("كوري") || genresStr.includes("korean"))) ||
          (activeOrigin === "ياباني" && (genresStr.includes("ياباني") || genresStr.includes("japanese"))) ||
          (activeOrigin === "آسيوي" && (genresStr.includes("آسيوي") || genresStr.includes("asian")));
        if (!originMatch) return false;
      }

      if (activeGenre !== "all") {
        if (!genresStr.includes(activeGenre.toLowerCase())) return false;
      }

      if (activeType !== "all") {
        if (activeType === "movie" && item.type !== "movie") return false;
        if (activeType === "series" && item.type !== "series") return false;
      }

      if (activeFormat !== "all") {
        if (!genresStr.includes(activeFormat.toLowerCase()) && !itemCat.includes(activeFormat.toLowerCase())) {
          return false;
        }
      }

      if (activeStatus !== "all") {
        const s = (item.status || "").toLowerCase();
        if (activeStatus === "ended" && !s.includes("ended") && !s.includes("منتهي")) return false;
        if (activeStatus === "returning" && !s.includes("returning") && !s.includes("مستمر")) return false;
      }

      if (activeSeasons !== "all") {
        const count = item.seasonCount || 1;
        if (activeSeasons === "single" && count !== 1) return false;
        if (activeSeasons === "multi" && count < 2) return false;
        if (activeSeasons === "5+" && count < 5) return false;
      }

      if (activeStudio !== "all") {
        if (!genresStr.includes(activeStudio.toLowerCase()) && !item.plot.toLowerCase().includes(activeStudio.toLowerCase())) {
          return false;
        }
      }

      if (activeTopic !== "all") {
        if (!genresStr.includes(activeTopic.toLowerCase()) && !item.plot.toLowerCase().includes(activeTopic.toLowerCase())) {
          return false;
        }
      }

      if (activePromotion !== "all") {
        if (!genresStr.includes(activePromotion.toLowerCase()) && !item.plot.toLowerCase().includes(activePromotion.toLowerCase())) {
          return false;
        }
      }

      if (activeDuration !== "all") {
        const mins = item.runtimeMinutes || 0;
        if (activeDuration === "short" && mins > 30) return false;
        if (activeDuration === "medium" && (mins < 30 || mins > 90)) return false;
        if (activeDuration === "feature" && (mins < 80 || mins > 140)) return false;
        if (activeDuration === "epic" && mins < 140) return false;
      }

      if (activeAudioDub !== "all") {
        if (activeAudioDub === "spacetoon" && !genresStr.includes("سبيستون") && !item.plot.includes("سبيستون")) return false;
        if (activeAudioDub === "egyptian" && !genresStr.includes("مصري") && !item.plot.includes("مصري")) return false;
        if (activeAudioDub === "fusha" && !genresStr.includes("فصحى") && !item.hasArabicAudio) return false;
      }

      return true;
    });
  }, [
    items,
    selectedCategory,
    searchQuery,
    hasArabicAudio,
    hasArabicSubtitles,
    activeYear,
    activeRating,
    activeQuality,
    activeContentRating,
    activeOrigin,
    activeGenre,
    activeType,
    activeFormat,
    activeStatus,
    activeSeasons,
    activeStudio,
    activeTopic,
    activePromotion,
    activeDuration,
    activeAudioDub,
  ]);

  // Sorting
  const sortedItems = useMemo(() => {
    return [...filteredItems].sort((a, b) => {
      const titleA = a.titleAr || a.titleEn || "";
      const titleB = b.titleAr || b.titleEn || "";

      if (activeSort === "rating") return (b.rating || 0) - (a.rating || 0);
      if (activeSort === "year") return (b.year || 0) - (a.year || 0);
      if (activeSort === "files") return (b.fileCount || 0) - (a.fileCount || 0);
      if (activeSort === "runtime") return (b.runtimeMinutes || 0) - (a.runtimeMinutes || 0);
      if (activeSort === "title") return titleA.localeCompare(titleB, "ar");
      return 0; // newest / default
    });
  }, [filteredItems, activeSort]);

  // Fallback showcase items
  const heroItems = useMemo(() => items.slice(0, 5), [items]);

  const resetAllFilters = useCallback(() => {
    setActiveOrigin("all");
    setActiveGenre("all");
    setActiveType("all");
    setActiveFormat("all");
    setActiveStatus("all");
    setActiveSeasons("all");
    setActiveStudio("all");
    setActiveTopic("all");
    setActivePromotion("all");
    setActiveSeasonYear("all");
    setActiveEra("all");
    setActiveDuration("all");
    setActiveAudioDub("all");
    setActiveQuality("all");
    setActiveYear("all");
    setActiveRating("all");
    setActiveContentRating("all");
    setHasArabicAudio(false);
    setHasArabicSubtitles(false);
    setSearchQuery("");
  }, []);

  return (
    <div className="space-y-8 pb-16 text-right" dir="rtl">
      {/* 1. Unified database-backed showcase */}
      {heroItems.length > 0 && (
        <ShowcaseHero
          context="category"
          category={selectedCategory}
          fallbackItems={heroItems}
          onOpenMedia={onOpenMedia}
          onNavigate={(target) => {
            if (target?.category && target.category !== selectedCategory) {
              window.location.hash = `#/catalog/${target.category}`;
            }
          }}
        />
      )}

      {/* 2. Unified Official Database Smart Hubs Rail */}
      <SmartHubRail
        scope={selectedCategory}
        title={`مجموعات ومحاور ${categoryConfig.titleAr}`}
        description="تصنيفات ذكية ومحاور حقيقية مبنية تلقائيًا ومربوطة بلوحة التحكم."
        onViewAll={() => (window.location.hash = "#/directory/hubs")}
        onOpen={(hub) => (window.location.hash = `#/hub/${hub.slug}`)}
      />

      {/* 3. Multi-Dimensional Context-Aware Filter Toolbar */}
      <FilterToolbar
        activeOrigin={activeOrigin}
        onSelectOrigin={filterConfig.showOrigin ? setActiveOrigin : null}
        activeGenre={activeGenre}
        onSelectGenre={filterConfig.showGenre ? setActiveGenre : null}
        activeType={activeType}
        onSelectType={filterConfig.showType ? setActiveType : null}
        activeFormat={activeFormat}
        onSelectFormat={filterConfig.showFormat ? setActiveFormat : null}
        activeStatus={activeStatus}
        onSelectStatus={filterConfig.showStatus ? setActiveStatus : null}
        activeSeasons={activeSeasons}
        onSelectSeasons={filterConfig.showSeasons ? setActiveSeasons : null}
        activeStudio={activeStudio}
        onSelectStudio={filterConfig.showStudios ? setActiveStudio : null}
        activeTopic={activeTopic}
        onSelectTopic={filterConfig.showTopic ? setActiveTopic : null}
        activePromotion={activePromotion}
        onSelectPromotion={filterConfig.showPromotion ? setActivePromotion : null}
        activeSeasonYear={activeSeasonYear}
        onSelectSeasonYear={filterConfig.showSeasonYear ? setActiveSeasonYear : null}
        activeEra={activeEra}
        onSelectEra={filterConfig.showEra ? setActiveEra : null}
        activeDuration={activeDuration}
        onSelectDuration={filterConfig.showDuration ? setActiveDuration : null}
        activeAudioDub={activeAudioDub}
        onSelectAudioDub={filterConfig.showAudioDub ? setActiveAudioDub : null}
        activeQuality={activeQuality}
        onSelectQuality={filterConfig.showQuality ? setActiveQuality : null}
        activeYear={activeYear}
        onSelectYear={filterConfig.showYear ? setActiveYear : null}
        activeRating={activeRating}
        onSelectRating={filterConfig.showRating ? setActiveRating : null}
        activeContentRating={activeContentRating}
        onSelectContentRating={filterConfig.showContentRating ? setActiveContentRating : null}
        hasArabicAudio={hasArabicAudio}
        onToggleArabicAudio={filterConfig.showAudioSubtitles !== false ? setHasArabicAudio : null}
        hasArabicSubtitles={hasArabicSubtitles}
        onToggleArabicSubtitles={filterConfig.showAudioSubtitles !== false ? setHasArabicSubtitles : null}
        activeSort={activeSort}
        onSelectSort={setActiveSort}
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        origins={filterConfig.origins}
        genres={filterConfig.genres}
        types={filterConfig.types}
        formats={filterConfig.formats}
        statuses={filterConfig.statuses}
        seasons={filterConfig.seasons}
        studios={filterConfig.studios}
        topics={filterConfig.topics}
        promotions={filterConfig.promotions}
        seasonYears={filterConfig.seasonYears}
        eras={filterConfig.eras}
        durations={filterConfig.durations}
        audioDubs={filterConfig.audioDubs}
        contentRatings={filterConfig.contentRatings}
        showOriginFilter={Boolean(filterConfig.showOrigin)}
        showGenreFilter={Boolean(filterConfig.showGenre)}
        showTypeFilter={Boolean(filterConfig.showType)}
        resultCount={filteredItems.length}
        onResetFilters={resetAllFilters}
      />

      {/* 4. Items Grid Section with Chunked Infinite Scroll */}
      <div className="space-y-4">
        {loading ? (
          <div className="flex items-center justify-center py-20">
            <div className="text-center space-y-3">
              <div className="w-10 h-10 border-4 border-fuchsia-500/30 border-t-fuchsia-500 rounded-full animate-spin mx-auto" />
              <p className="text-xs text-[var(--text-secondary)]">جاري تحميل وتصنيف الأعمال الفنية...</p>
            </div>
          </div>
        ) : filteredItems.length === 0 ? (
          <div className="p-16 rounded-3xl border border-dashed border-[var(--border-default)] bg-[var(--bg-card)] text-center space-y-3">
            <span className="text-4xl">🎬</span>
            <h3 className="text-base font-bold text-[var(--text-primary)]">لا توجد أعمال تطابق الفلاتر المحددة</h3>
            <p className="text-xs text-[var(--text-secondary)] max-w-sm mx-auto">
              جرب تغيير خيارات البحث أو إعادة تعيين الفلاتر لعرض كافة محتويات هذا القسم.
            </p>
          </div>
        ) : (
          <>
            <MediaCollection items={sortedItems} onOpen={onOpenMedia} />

            {/* Infinite Scroll Sentinel */}
            {hasMore && (
              <div ref={observerTargetRef} className="py-8 flex justify-center items-center">
                {loadingMore ? (
                  <div className="flex items-center gap-3 px-5 py-2.5 rounded-2xl bg-[var(--bg-card)] border border-[var(--border-default)] text-xs text-[var(--text-secondary)] shadow-lg animate-pulse">
                    <div className="w-4 h-4 border-2 border-fuchsia-500/30 border-t-fuchsia-500 rounded-full animate-spin" />
                    <span>جاري تحميل المزيد من الأعمال...</span>
                  </div>
                ) : (
                  <div className="h-6 w-full" />
                )}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
