import React, { useEffect, useMemo, useState } from "react";
import ShowcaseHero from "../components/ShowcaseHero.jsx";
import FilterToolbar from "../components/FilterToolbar.jsx";
import MediaCollection from "../components/MediaCollection.jsx";
import SmartHubRail from "../components/SmartHubRail.jsx";
import Icon from "../components/Icon.jsx";
import { getMediaList } from "../lib/api.js";
import { getCategoryConfig } from "../data/categoryConfig.js";

export default function CategoryPage({ selectedCategory = "series", onOpenMedia, onQuickPlay }) {
  const [items, setItems] = useState([]);
  const [totalCount, setTotalCount] = useState(0);
  const [loading, setLoading] = useState(true);

  // Context-Aware Filter States
  const [activeOrigin, setActiveOrigin] = useState("all");
  const [activeGenre, setActiveGenre] = useState("all");
  const [activeType, setActiveType] = useState("all");
  const [activeFormat, setActiveFormat] = useState("all");
  const [activeStatus, setActiveStatus] = useState("all");
  const [activeSeasons, setActiveSeasons] = useState("all");
  const [activeStudio, setActiveStudio] = useState("all");
  const [activeTopic, setActiveTopic] = useState("all");
  const [activePromotion, setActivePromotion] = useState("all");
  const [activeSeasonYear, setActiveSeasonYear] = useState("all");
  const [activeEra, setActiveEra] = useState("all");
  const [activeDuration, setActiveDuration] = useState("all");
  const [activeAudioDub, setActiveAudioDub] = useState("all");
  const [activeQuality, setActiveQuality] = useState("all");
  const [activeYear, setActiveYear] = useState("all");
  const [activeRating, setActiveRating] = useState("all");
  const [activeContentRating, setActiveContentRating] = useState("all");
  const [hasArabicAudio, setHasArabicAudio] = useState(false);
  const [hasArabicSubtitles, setHasArabicSubtitles] = useState(false);
  const [activeSort, setActiveSort] = useState("newest");
  const [searchQuery, setSearchQuery] = useState("");

  // Category Configuration
  const categoryConfig = useMemo(() => getCategoryConfig(selectedCategory), [selectedCategory]);
  const filterConfig = categoryConfig.filterConfig || {};

  // Reset filters when changing category
  useEffect(() => {
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
  }, [selectedCategory]);

  useEffect(() => {
    loadCategoryItems();
  }, [selectedCategory, activeSort]);

  async function loadCategoryItems() {
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
        limit: 1000,
      };

      // Multi-Indexing Strategy:
      // Movies: fetch all movies (live action + cartoons)
      // Series: fetch all series (live drama + cartoons)
      // Kids / Family / Anime: fetch full library so client categorizes and isolates with 100% precision
      if (selectedCategory === "movies") {
        fetchParams.type = "movie";
      } else if (selectedCategory === "series") {
        fetchParams.type = "series";
      } else if (["family", "kids", "anime"].includes(selectedCategory)) {
        // Full cross-category indexing
      } else {
        fetchParams.category = selectedCategory;
      }

      const res = await getMediaList(fetchParams);

      const transformed = (res?.items || []).map((item) => ({
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
      }));

      setItems(transformed);
      setTotalCount(res?.total || transformed.length);
    } catch {
      setItems([]);
      setTotalCount(0);
    } finally {
      setLoading(false);
    }
  }

  // 100% Context-Aware Multi-Dimensional Filtering Logic
  const filteredItems = useMemo(() => {
    return items.filter((item) => {
      const itemCat = (item.categorySlug || "").toLowerCase();
      const genresStr = (item.genres || []).join(" ").toLowerCase();
      const cr = String(item.contentRating || "").toUpperCase().trim();

      // 0. Multi-Indexing & Isolation Rules
      if (selectedCategory === "movies") {
        // ONLY EXCLUDE ANIME - Cartoons/Animation (Disney/Pixar/Dreamworks) ARE FULLY INCLUDED in Movies!
        if (itemCat === "anime" || genresStr.includes("أنمي") || genresStr.includes("anime")) return false;
        if (item.type && item.type !== "movie") return false;
      } else if (selectedCategory === "series") {
        // ONLY EXCLUDE ANIME SERIES
        if (itemCat === "anime" || genresStr.includes("أنمي") || genresStr.includes("anime")) return false;
        if (item.type && item.type !== "series") return false;
      } else if (selectedCategory === "kids") {
        // Kids & Cartoons: animation, cartoons, family, spacetoon, disney, pixar, dreamworks
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
        // Exclude mature content from kids
        if (["R", "NC-17", "TV-MA", "18+", "18"].includes(cr)) return false;
      } else if (selectedCategory === "family") {
        // Family Cinema: any family-safe movie or series across all genres
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
        // Anime: anime movies, series, OVAs
        const isAnime = itemCat === "anime" || genresStr.includes("أنمي") || genresStr.includes("anime");
        if (!isAnime) return false;
      }

      // 1. Search Query
      if (searchQuery.trim()) {
        const query = searchQuery.trim().toLowerCase();
        const matchesTitle =
          (item.titleAr && item.titleAr.toLowerCase().includes(query)) ||
          (item.titleEn && item.titleEn.toLowerCase().includes(query)) ||
          (item.plot && item.plot.toLowerCase().includes(query)) ||
          genresStr.includes(query);
        if (!matchesTitle) return false;
      }

      // 2. Origin Filter
      if (activeOrigin !== "all") {
        const hasOrigin = item.genres.some((g) => g.includes(activeOrigin));
        if (!hasOrigin) return false;
      }

      // 3. Genre Filter
      if (activeGenre !== "all") {
        const hasGenre = item.genres.some((g) => g.includes(activeGenre));
        if (!hasGenre) return false;
      }

      // 4. Type / Format Filter
      if (activeType !== "all" && item.type !== activeType) return false;

      if (activeFormat !== "all") {
        if (activeFormat === "series" && item.type !== "series") return false;
        if (activeFormat === "movie" && item.type !== "movie") return false;
        if (activeFormat === "ova" && !genresStr.includes("ova") && !genresStr.includes("أوفا") && !genresStr.includes("خاص")) return false;
      }

      // 5. Status Filter
      if (activeStatus !== "all") {
        const s = (item.status || "completed").toLowerCase();
        if (activeStatus === "completed" && s !== "completed") return false;
        if (activeStatus === "ongoing" && s !== "ongoing" && s !== "returning series") return false;
      }

      // 6. Seasons Count Filter
      if (activeSeasons !== "all") {
        const sc = item.seasonCount || item.tmdbSeasonCount || 1;
        if (activeSeasons === "mini" && sc > 1) return false;
        if (activeSeasons === "medium" && (sc < 2 || sc > 3)) return false;
        if (activeSeasons === "long" && sc < 4) return false;
      }

      // 7. Studio Filter (Kids / Cartoons)
      if (activeStudio !== "all") {
        const studioKeywords = {
          disney: ["ديزني", "disney"],
          pixar: ["بيكسار", "pixar"],
          dreamworks: ["دريم وركس", "dreamworks"],
          spacetoon: ["سبيستون", "spacetoon", "الزهرة"],
          cartoon_network: ["كرتون نتورك", "cartoon network", "cn"],
          nickelodeon: ["نيكلوديون", "nickelodeon"],
          illumination: ["إلومينيشن", "illumination", "minions"],
          ghibli: ["غيبلي", "ghibli"],
        };
        const keywords = studioKeywords[activeStudio] || [activeStudio];
        const matchesStudio = keywords.some(
          (k) =>
            genresStr.includes(k) ||
            (item.titleAr && item.titleAr.toLowerCase().includes(k)) ||
            (item.titleEn && item.titleEn.toLowerCase().includes(k)) ||
            (item.plot && item.plot.toLowerCase().includes(k))
        );
        if (!matchesStudio) return false;
      }

      // 8. Topic Filter (Documentaries)
      if (activeTopic !== "all") {
        const topicKeywords = {
          nature: ["طبيعة", "حيوان", "nature", "wildlife", "earth", "bbc", "natgeo"],
          space: ["فضاء", "فلك", "space", "cosmos", "universe", "planet"],
          history: ["تاريخ", "حضارة", "history", "ancient", "war", "حضارات"],
          tech: ["تكنولوجيا", "علوم", "science", "technology", "future", "ai"],
          crime: ["جريمة", "تحقيق", "true crime", "crime", "murder"],
          war: ["حرب", "عسكري", "war", "military"],
          social: ["مجتمع", "ثقافة", "society", "culture"],
        };
        const keywords = topicKeywords[activeTopic] || [activeTopic];
        const matchesTopic = keywords.some(
          (k) =>
            genresStr.includes(k) ||
            (item.titleAr && item.titleAr.toLowerCase().includes(k)) ||
            (item.titleEn && item.titleEn.toLowerCase().includes(k)) ||
            (item.plot && item.plot.toLowerCase().includes(k))
        );
        if (!matchesTopic) return false;
      }

      // 9. Promotion Filter (Wrestling / Sports)
      if (activePromotion !== "all") {
        const promoKeywords = {
          wwe: ["wwe", "raw", "smackdown", "wrestlemania", "royal rumble", "مصارعة"],
          aew: ["aew", "dynamite", "rampage"],
          ufc: ["ufc", "mma", "قتال"],
          boxing: ["boxing", "ملاكمة"],
        };
        const keywords = promoKeywords[activePromotion] || [activePromotion];
        const matchesPromo = keywords.some(
          (k) =>
            genresStr.includes(k) ||
            (item.titleAr && item.titleAr.toLowerCase().includes(k)) ||
            (item.titleEn && item.titleEn.toLowerCase().includes(k))
        );
        if (!matchesPromo) return false;
      }

      // 10. Season Year (Ramadan)
      if (activeSeasonYear !== "all") {
        const year = Number(item.year || 0);
        if (activeSeasonYear === "2025" && year !== 2025) return false;
        if (activeSeasonYear === "2024" && year !== 2024) return false;
        if (activeSeasonYear === "2023" && year !== 2023) return false;
        if (activeSeasonYear === "2020-2022" && (year < 2020 || year > 2022)) return false;
        if (activeSeasonYear === "classic" && year >= 2020) return false;
      }

      // 11. Era Filter (Plays)
      if (activeEra !== "all") {
        const year = Number(item.year || 0);
        if (activeEra === "golden" && year > 1989) return false;
        if (activeEra === "nineties" && (year < 1990 || year > 2009)) return false;
        if (activeEra === "modern" && year < 2010) return false;
      }

      // 12. Duration Filter (Movies)
      if (activeDuration !== "all") {
        const rt = Number(item.runtimeMinutes || 0);
        if (rt > 0) {
          if (activeDuration === "short" && rt >= 90) return false;
          if (activeDuration === "standard" && (rt < 90 || rt > 135)) return false;
          if (activeDuration === "epic" && rt <= 135) return false;
        }
      }

      // 13. Audio Dubs Filter
      if (activeAudioDub !== "all") {
        if (activeAudioDub === "arabic_classic" && !genresStr.includes("سبيستون") && !genresStr.includes("الزهرة") && !item.hasArabicAudio) return false;
        if (activeAudioDub === "arabic_modern" && !item.hasArabicAudio) return false;
        if (activeAudioDub === "subbed" && !item.hasArabicSubtitles) return false;
        if (activeAudioDub === "egyptian" && !genresStr.includes("مصري") && !item.hasArabicAudio) return false;
        if (activeAudioDub === "msa" && !item.hasArabicAudio) return false;
      }

      // 14. Content Rating
      if (activeContentRating !== "all") {
        if (activeContentRating === "family") {
          if (!["G", "PG", "TV-G", "TV-Y", "TV-Y7", "ALL"].includes(cr)) return false;
        } else if (activeContentRating === "teen") {
          if (!["PG-13", "TV-14", "13+", "12"].includes(cr)) return false;
        } else if (activeContentRating === "mature") {
          if (!["R", "NC-17", "TV-MA", "18+", "18", "MA"].includes(cr)) return false;
        } else {
          if (cr !== activeContentRating) return false;
        }
      }

      // 15. Quality & Year & Rating
      if (activeQuality !== "all" && item.bestResolution !== activeQuality) return false;
      if (activeRating !== "all" && (item.rating || 0) < Number(activeRating)) return false;
      if (hasArabicAudio && !item.hasArabicAudio) return false;
      if (hasArabicSubtitles && !item.hasArabicSubtitles) return false;
      if (activeYear !== "all") {
        const year = Number(item.year || 0);
        if (activeYear === "2025+" && year < 2025) return false;
        if (activeYear === "2024" && year !== 2024) return false;
        if (activeYear === "2023" && year !== 2023) return false;
        if (activeYear === "2020-2024" && (year < 2020 || year > 2024)) return false;
        if (activeYear === "2010-2019" && (year < 2010 || year > 2019)) return false;
        if (activeYear === "2000-2009" && (year < 2000 || year > 2009)) return false;
        if (activeYear === "classic" && year >= 2000) return false;
      }

      return true;
    });
  }, [
    items,
    selectedCategory,
    searchQuery,
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
  ]);

  const sortedItems = useMemo(() => {
    return [...filteredItems].sort((a, b) => {
      const titleA = (a.titleAr || a.titleEn || "").toLocaleLowerCase();
      const titleB = (b.titleAr || b.titleEn || "").toLocaleLowerCase();
      if (activeSort === "oldest") return (a.id || 0) - (b.id || 0);
      if (activeSort === "rating") return (b.rating || 0) - (a.rating || 0);
      if (activeSort === "rating_low") return (a.rating || 0) - (b.rating || 0);
      if (activeSort === "year") return (b.year || 0) - (a.year || 0);
      if (activeSort === "year_old") return (a.year || 0) - (b.year || 0);
      if (activeSort === "title_desc") return titleB.localeCompare(titleA, "ar");
      if (activeSort === "files") return (b.fileCount || 0) - (a.fileCount || 0);
      if (activeSort === "runtime") return (b.runtimeMinutes || 0) - (a.runtimeMinutes || 0);
      if (activeSort === "title") return titleA.localeCompare(titleB, "ar");
      return 0; // newest / default
    });
  }, [filteredItems, activeSort]);

  // Fallback showcase items
  const heroItems = useMemo(() => items.slice(0, 5), [items]);

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

      {/* 2. Unified Official Database Smart Hubs Rail (SmartHubCard with artwork & badges) */}
      <SmartHubRail
        scope={selectedCategory}
        title={`مجموعات ومحاور ${categoryConfig.titleAr}`}
        description="تصنيفات ذكية ومحاور حقيقية مبنية تلقائيًا ومربوطة بلوحة التحكم."
        onViewAll={() => (window.location.hash = "#/directory/hubs")}
        onOpen={(hub) => (window.location.hash = `#/hub/${hub.slug}`)}
      />

      {/* 3. Multi-Dimensional Context-Aware Filter Toolbar */}
      <FilterToolbar
        // State values
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
        // Custom Options from Category Config
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
        onResetFilters={() => {
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
        }}
      />

      {/* 4. Items Grid Section */}
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
          <MediaCollection items={sortedItems} onOpen={onOpenMedia} />
        )}
      </div>
    </div>
  );
}
