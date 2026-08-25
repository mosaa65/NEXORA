import React, { useEffect, useMemo, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import ShowcaseHero from "../components/ShowcaseHero.jsx";
import HubBannerCard from "../components/HubBannerCard.jsx";
import FilterToolbar from "../components/FilterToolbar.jsx";
import MediaCollection from "../components/MediaCollection.jsx";
import SmartHubRail from "../components/SmartHubRail.jsx";
import Icon from "../components/Icon.jsx";
import { getMediaList, resolveAPIURL } from "../lib/api.js";

const categoryMetas = {
  anime: {
    titleAr: "الأنمي والرسوم اليابانية",
    label: "أنمي",
    description: "عوالم الأنمي الأسطورية، المواسم الكاملة والحلقات بأعلى جودة وصوت ياباني مع الترجمة العربية.",
  },
  movies: {
    titleAr: "الأفلام السينمائية",
    label: "فيلم",
    description: "مكتبة سينمائية متكاملة من أقوى أفلام هوليوود، السينما العربية، الكورية، والهندية بدقة 4K.",
  },
  series: {
    titleAr: "المسلسلات والدراما",
    label: "مسلسل",
    description: "أقوى إنتاجات الدراما التركية، العربية، والمسلسلات العالمية الحصرية مع كامل الحلقات والمواسم.",
  },
  kids: {
    titleAr: "الأطفال والكرتون العائلي",
    label: "عمل",
    description: "أفلام ديزني وبيكسار، سلاسل الأبطال الخارقين، ومسلسلات الكرتون وسبيستون المدبلجة الممتعة.",
  },
  documentaries: {
    titleAr: "الوثائقيات والمعرفة",
    label: "وثائقي",
    description: "سلاسل الطبيعة، التاريخ، والتكنولوجيا من كبرى استوديوهات الإنتاج العالمية.",
  },
  plays: {
    titleAr: "المسرحيات والكوميديا",
    label: "مسرحية",
    description: "كلاسيكيات المسرح الكوميدي الخليجي والمصري بجودة ممتازة.",
  },
};

const seriesHubs = [
  {
    id: "turkish",
    title: "👑 روائع الدراما التركية",
    subtitle: "حكايات ملحمية وعاطفية من قمة الإنتاج التركي المدبلج والمترجم",
    tag: "الأعلى طلباً",
    originTerm: "تركي",
    backdrop: "/nexora-library-backdrop.PNG",
    accentColor: "from-amber-700/80 via-orange-950/80 to-[#0F0E1A]",
    borderColor: "border-amber-500/40",
  },
  {
    id: "arabic",
    title: "🌟 الدراما العربية والخليجية",
    subtitle: "إنتاجات مصرية، خليجية، وشامية حصرية ومتنوعة لجميع الأذواق",
    tag: "إنتاج عربي",
    originTerm: "عربي",
    backdrop: "/nexora-library-backdrop.PNG",
    accentColor: "from-rose-700/80 via-purple-950/80 to-[#0F0E1A]",
    borderColor: "border-rose-500/40",
  },
  {
    id: "foreign",
    title: "🎬 المسلسلات الأجنبية والعالمية",
    subtitle: "أضخم أعمال HBO و Netflix و Apple من الجريمة والخيال العلمي",
    tag: "Global Hits",
    originTerm: "أجنبي",
    backdrop: "/nexora-library-backdrop.PNG",
    accentColor: "from-cyan-700/80 via-blue-950/80 to-[#0F0E1A]",
    borderColor: "border-cyan-500/40",
  },
  {
    id: "korean",
    title: "🌸 الدراما الكورية (K-Drama)",
    subtitle: "قصص مشوقة ورومانسية وإثارة آسيوية متصدرة التريند العالمي",
    tag: "K-Drama",
    originTerm: "كوري",
    backdrop: "/nexora-library-backdrop.PNG",
    accentColor: "from-fuchsia-700/80 via-pink-950/80 to-[#0F0E1A]",
    borderColor: "border-fuchsia-500/40",
  },
];

const kidsHubs = [
  {
    id: "disney",
    title: "🏰 كلاسيكيات ديزني وبيكسار",
    subtitle: "ملكة الثلج، حكاية لعبة، والأسد الملك بجودة فائقة ودبلجة احترافية",
    tag: "Disney / Pixar",
    originTerm: "",
    genreTerm: "عائلي",
    backdrop: "/nexora-library-backdrop.PNG",
    accentColor: "from-sky-700/80 via-indigo-950/80 to-[#0F0E1A]",
    borderColor: "border-sky-500/40",
  },
  {
    id: "spiderman",
    title: "🕷️ سلاسل الأبطال الخارقين",
    subtitle: "مغامرات سبايدرمان، باتمان، وأفلام الرسوم المتحركة الحماسية",
    tag: "Superheroes",
    originTerm: "",
    genreTerm: "أكشن",
    backdrop: "/nexora-library-backdrop.PNG",
    accentColor: "from-red-700/80 via-rose-950/80 to-[#0F0E1A]",
    borderColor: "border-red-500/40",
  },
  {
    id: "spacetoon",
    title: "⭐ ذكريات سبيستون المدبلجة",
    subtitle: "المحقق كونان، أبطال الديجيتال، وعهد الأصدقاء بحنين الزمن الجميل",
    tag: "سبيستون",
    originTerm: "ياباني",
    genreTerm: "كرتون",
    backdrop: "/nexora-library-backdrop.PNG",
    accentColor: "from-amber-600/80 via-orange-950/80 to-[#0F0E1A]",
    borderColor: "border-amber-500/40",
  },
];

export default function CategoryPage({ selectedCategory = "series", onOpenMedia, onQuickPlay }) {
  const [items, setItems] = useState([]);
  const [totalCount, setTotalCount] = useState(0);
  const [loading, setLoading] = useState(true);

  // Filter States
  const [activeOrigin, setActiveOrigin] = useState("all");
  const [activeGenre, setActiveGenre] = useState("all");
  const [activeSort, setActiveSort] = useState("newest");
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedHub, setSelectedHub] = useState(null);

  const meta = categoryMetas[selectedCategory] || categoryMetas.series;

  useEffect(() => {
    // Reset filters on category switch
    setActiveOrigin("all");
    setActiveGenre("all");
    setSearchQuery("");
    setSelectedHub(null);
    loadCategoryItems();
  }, [selectedCategory]);

  useEffect(() => {
    loadCategoryItems();
  }, [selectedCategory, activeSort]);

  async function loadCategoryItems() {
    setLoading(true);
    try {
      const res = await getMediaList({
        category: selectedCategory,
        sort: activeSort === "rating" ? "rating" : activeSort === "year" ? "year" : activeSort === "title" ? "title" : "",
        limit: 1000,
      });

      const transformed = (res?.items || []).map((item) => ({
        id: item.id,
        titleAr: item.title_ar,
        titleEn: item.title_en,
        type: item.type,
        plot: item.plot_ar || item.plot_en || "",
        year: item.release_year,
        rating: item.rating,
        posterPath: item.poster_path,
        bannerPath: item.banner_path,
        categorySlug: item.category_slug || selectedCategory,
        fileCount: item.file_count,
        status: item.status,
        seasonCount: item.season_count,
        tmdbSeasonCount: item.tmdb_season_count,
        tmdbEpisodeCount: item.tmdb_episode_count,
        totalSize: item.total_size,
        bestResolution: item.best_resolution,
        runtimeMinutes: item.runtime_minutes,
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

  // Filter Logic
  const filteredItems = useMemo(() => {
    return items.filter((item) => {
      // 1. Search Query
      if (searchQuery.trim()) {
        const query = searchQuery.trim().toLowerCase();
        const matchesTitle =
          (item.titleAr && item.titleAr.toLowerCase().includes(query)) ||
          (item.titleEn && item.titleEn.toLowerCase().includes(query)) ||
          (item.plot && item.plot.toLowerCase().includes(query));
        if (!matchesTitle) return false;
      }

      // 2. Hub Filter (if a hub was selected)
      if (selectedHub) {
        if (selectedHub.originTerm && !item.genres.some((g) => g.includes(selectedHub.originTerm))) {
          return false;
        }
        if (selectedHub.genreTerm && !item.genres.some((g) => g.includes(selectedHub.genreTerm))) {
          return false;
        }
      }

      // 3. Origin Filter
      if (activeOrigin !== "all") {
        const hasOrigin = item.genres.some((g) => g.includes(activeOrigin));
        if (!hasOrigin) return false;
      }

      // 4. Genre Filter
      if (activeGenre !== "all") {
        const hasGenre = item.genres.some((g) => g.includes(activeGenre));
        if (!hasGenre) return false;
      }

      return true;
    });
  }, [items, searchQuery, selectedHub, activeOrigin, activeGenre]);

  // Fallback content while the database-backed showcase is loading.
  const heroItems = useMemo(() => {
    return items.slice(0, 5);
  }, [items]);

  // Legacy hand-authored banners stay dormant while the database-driven hubs
  // are rolled out. They are kept only as a reversible migration fallback.
  const activeHubs = [];

  return (
    <div className="space-y-8 pb-16 text-right" dir="rtl">
      {/* 1. Unified database-backed showcase (If not in a sub-hub view) */}
      {!selectedHub && heroItems.length > 0 && (
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

      {/* 2. Sub-Hub Banner Header (If browsing a specific collection) */}
      {selectedHub && (
        <div
          className="relative min-h-[220px] rounded-3xl overflow-hidden border border-fuchsia-500/30 p-6 sm:p-8 flex flex-col justify-end text-right shadow-2xl bg-[#0E0D1B]"
          style={{ backgroundImage: `url('${selectedHub.backdrop}')`, backgroundSize: "cover", backgroundPosition: "center" }}
        >
          <div className="absolute inset-0 bg-gradient-to-t from-[#090812] via-[#090812]/80 to-transparent" />

          <div className="relative z-10 space-y-2">
            <button
              onClick={() => setSelectedHub(null)}
              className="px-3.5 py-1.5 rounded-full bg-white/10 hover:bg-white/20 text-white font-bold text-xs backdrop-blur-md border border-white/20 transition flex items-center gap-1.5 w-fit"
            >
              <span>‹ العودة لجميع المجموعات</span>
            </button>
            <h1 className="text-2xl sm:text-4xl font-black text-white">{selectedHub.title}</h1>
            <p className="text-xs sm:text-sm text-gray-300 max-w-xl">{selectedHub.subtitle}</p>
          </div>
        </div>
      )}

      {/* 3. Grand Origin Hubs / Collections Section */}
      {!selectedHub && <SmartHubRail scope={selectedCategory === "movies" ? "movies" : selectedCategory} title={`مجموعات ومحاور ${meta.titleAr}`} description="تصنيفات ذكية مبنية تلقائيًا من بيانات مكتبتك." onOpen={(hub) => (window.location.hash = `#/hub/${hub.slug}`)} />}

      {!selectedHub && activeHubs.length > 0 && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-xl font-black text-white flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-fuchsia-500 shadow-[0_0_10px_#e879f9]" />
                مجموعات ومحاور {meta.titleAr}
              </h2>
              <p className="text-xs text-gray-400">تصفح الإنتاجات المجمعة حسب المنشأ والسلاسل السينمائية</p>
            </div>
            <span className="text-xs font-mono font-bold text-gray-400 bg-white/5 px-3 py-1 rounded-full border border-white/10">
              {totalCount} {meta.label}
            </span>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            {activeHubs.map((hub) => {
              const hubCount = items.filter((it) => {
                if (hub.originTerm && it.genres.some((g) => g.includes(hub.originTerm))) return true;
                if (hub.genreTerm && it.genres.some((g) => g.includes(hub.genreTerm))) return true;
                return false;
              }).length;

              return (
                <HubBannerCard
                  key={hub.id}
                  title={hub.title}
                  subtitle={hub.subtitle}
                  count={hubCount}
                  backdrop={hub.backdrop}
                  accentColor={hub.accentColor}
                  borderColor={hub.borderColor}
                  tag={hub.tag}
                  onClick={() => {
                    setSelectedHub(hub);
                    window.scrollTo({ top: 400, behavior: "smooth" });
                  }}
                />
              );
            })}
          </div>
        </div>
      )}

      {/* 4. Multi-Dimensional Filter Toolbar */}
      <FilterToolbar
        activeOrigin={activeOrigin}
        onSelectOrigin={setActiveOrigin}
        activeGenre={activeGenre}
        onSelectGenre={setActiveGenre}
        activeSort={activeSort}
        onSelectSort={setActiveSort}
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        showOriginFilter={selectedCategory === "series" || selectedCategory === "movies"}
        resultCount={filteredItems.length}
        onResetFilters={() => {
          setActiveOrigin("all");
          setActiveGenre("all");
          setSearchQuery("");
        }}
      />

      {/* 5. Items Grid Section */}
      <div className="space-y-4">
        {loading ? (
          <div className="flex items-center justify-center py-20">
            <div className="text-center space-y-3">
              <div className="w-10 h-10 border-4 border-fuchsia-500/30 border-t-fuchsia-500 rounded-full animate-spin mx-auto" />
              <p className="text-xs text-gray-400">جاري تحميل الأعمال الفنية...</p>
            </div>
          </div>
        ) : filteredItems.length === 0 ? (
          <div className="p-16 rounded-3xl border border-dashed border-white/10 bg-black/30 text-center space-y-3">
            <span className="text-4xl">🎬</span>
            <h3 className="text-base font-bold text-white">لا توجد أعمال تطابق الفلاتر المحددة</h3>
            <p className="text-xs text-gray-400 max-w-sm mx-auto">
              جرب تغيير خيارات البحث أو إعادة ضبط فلاتر الدولة والتصنيف.
            </p>
          </div>
        ) : (
          <MediaCollection items={filteredItems} onOpen={onOpenMedia} />
        )}
      </div>
    </div>
  );
}
