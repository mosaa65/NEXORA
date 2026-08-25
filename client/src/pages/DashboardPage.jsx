import React, { useEffect, useRef, useState } from "react";
import ShowcaseHero from "../components/ShowcaseHero.jsx";
import UnifiedMediaCard from "../components/UnifiedMediaCard.jsx";
import HubBannerCard from "../components/HubBannerCard.jsx";
import SmartHubRail from "../components/SmartHubRail.jsx";
import FranchiseRail from "../components/FranchiseRail.jsx";
import PeopleRail from "../components/PeopleRail.jsx";
import Icon from "../components/Icon.jsx";
import { getDashboardStats, getMediaList } from "../lib/api.js";
import { mockLibrary } from "../data/library.js";

export default function DashboardPage({
  searchQuery,
  onSearchChange,
  searchResults,
  onOpenMedia,
  onQuickPlay,
  onNavigateCategory,
}) {
  const [items, setItems] = useState([]);
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);

  const topRatedRef = useRef(null);
  const seriesRef = useRef(null);
  const moviesRef = useRef(null);

  useEffect(() => {
    let alive = true;
    setLoading(true);

    getMediaList({ limit: 1000, sort: "rating" })
      .then((data) => {
        if (!alive) return;
        if (data?.items && data.items.length > 0) {
          const transformed = data.items.map((item) => ({
            id: item.id,
            titleAr: item.title_ar,
            titleEn: item.title_en,
            type: item.type,
            plot: item.plot_ar || item.plot_en || "",
            year: item.release_year,
            rating: item.rating,
            posterPath: item.poster_path,
            bannerPath: item.banner_path,
            categorySlug: item.category_slug,
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
        } else {
          setItems(mockLibrary);
        }
      })
      .catch(() => {
        if (alive) setItems(mockLibrary);
      })
      .finally(() => {
        if (alive) setLoading(false);
      });

    getDashboardStats()
      .then((data) => {
        if (alive && data) setStats(data);
      })
      .catch(() => {});

    return () => {
      alive = false;
    };
  }, []);

  const displayItems = searchResults?.length > 0 && searchQuery ? searchResults : items.length > 0 ? items : mockLibrary;
  const heroItems = displayItems.slice(0, 6);
  const seriesList = displayItems.filter((i) => i.type === "series" || i.type === "anime").slice(0, 15);
  const moviesList = displayItems.filter((i) => i.type === "movie").slice(0, 15);
  const topRatedList = [...displayItems].sort((a, b) => (b.rating || 0) - (a.rating || 0)).slice(0, 15);

  function scroll(ref, direction) {
    if (ref.current) {
      const scrollAmount = direction === "left" ? -350 : 350;
      ref.current.scrollBy({ left: scrollAmount, behavior: "smooth" });
    }
  }

  return (
    <div className="space-y-10 pb-16 text-right" dir="rtl">
      {/* 1. Database-backed editorial showcase */}
      {heroItems.length > 0 && (
        <ShowcaseHero
          context="home"
          fallbackItems={heroItems}
          onOpenMedia={onOpenMedia}
          onNavigate={(target) => target?.category && onNavigateCategory?.(target.category)}
        />
      )}

      <SmartHubRail
        title="استكشف حسب ذوقك"
        description="محاور ذكية تتكوّن تلقائيًا من بيانات مكتبتك المحلية."
        onOpen={(hub) => (window.location.hash = `#/hub/${hub.slug}`)}
      />

      <FranchiseRail onOpen={(franchise) => { window.location.hash = `#/franchise/${franchise.slug}`; }} />

      <PeopleRail onOpen={(person) => { window.location.hash = `#/person/${person.slug}`; }} />

      {/* 2. Featured Hub Banners Grid */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-black text-white flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full bg-fuchsia-500 shadow-[0_0_12px_#e879f9]" />
            مجموعات الاستراحة الحصرية
          </h2>
          <span className="text-xs font-bold text-gray-400">تصفح سريع</span>
        </div>

        <div className="flex gap-4 overflow-x-auto pb-3 scrollbar-none snap-x touch-pan-x">
          <div className="w-[min(84vw,340px)] shrink-0 snap-start sm:w-80"><HubBannerCard
            title="👑 روائع الدراما التركية"
            subtitle="الحكايات العثمانية والدراما العائلية الرومانسية الكاملة"
            count={seriesList.length}
            backdrop="/nexora-library-backdrop.PNG"
            accentColor="from-amber-800/90 via-orange-950/80 to-[#0C0B17]"
            borderColor="border-amber-500/40"
            tag="الأكثر طلباً"
            onClick={() => onNavigateCategory && onNavigateCategory("series")}
          /></div>
          <div className="w-[min(84vw,340px)] shrink-0 snap-start sm:w-80"><HubBannerCard
            title="🎬 أفلام هوليوود والعالم"
            subtitle="أحدث أفلام الأكشن والخيال العلمي والإثارة بدقة 4K"
            count={moviesList.length}
            backdrop="/nexora-library-backdrop.PNG"
            accentColor="from-cyan-800/90 via-blue-950/80 to-[#0C0B17]"
            borderColor="border-cyan-500/40"
            tag="سينما 4K"
            onClick={() => onNavigateCategory && onNavigateCategory("movies")}
          /></div>
          <div className="w-[min(84vw,340px)] shrink-0 snap-start sm:w-80"><HubBannerCard
            title="⚔️ عوالم الأنمي الأسطورية"
            subtitle="ون بيس، هجوم العمالقة، وجوجوتسو كايسن بمواسمها الكاملة"
            count={seriesList.length}
            backdrop="/nexora-library-backdrop.PNG"
            accentColor="from-purple-800/90 via-fuchsia-950/80 to-[#0C0B17]"
            borderColor="border-purple-500/40"
            tag="Anime Top"
            onClick={() => onNavigateCategory && onNavigateCategory("anime")}
          /></div>
          <div className="w-[min(84vw,340px)] shrink-0 snap-start sm:w-80"><HubBannerCard
            title="🏰 ديزني والأطفال العائلي"
            subtitle="أفلام الرسوم المتحركة وسلاسل الأبطال الخارقين والكرتون"
            count={moviesList.length}
            backdrop="/nexora-library-backdrop.PNG"
            accentColor="from-sky-800/90 via-indigo-950/80 to-[#0C0B17]"
            borderColor="border-sky-500/40"
            tag="عائلي وأطفال"
            onClick={() => onNavigateCategory && onNavigateCategory("kids")}
          /></div>
        </div>
      </div>

      {/* 3. Carousel: Top Rated Showcase */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg sm:text-xl font-black text-white flex items-center gap-2">
              <span className="text-amber-400">⭐</span>
              الأعلى تقييماً في المكتبة (Top Rated)
            </h2>
            <p className="text-xs text-gray-400">أقوى الأعمال الفنية المعتمدة من IMDB و TMDB</p>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => scroll(topRatedRef, "right")}
              className="p-2 rounded-xl bg-slate-800/80 hover:bg-slate-700 text-white transition border border-slate-700"
              title="السابق"
            >
              ›
            </button>
            <button
              onClick={() => scroll(topRatedRef, "left")}
              className="p-2 rounded-xl bg-slate-800/80 hover:bg-slate-700 text-white transition border border-slate-700"
              title="التالي"
            >
              ‹
            </button>
          </div>
        </div>

        <div
          ref={topRatedRef}
          className="flex items-stretch gap-4 overflow-x-auto pb-4 scrollbar-none snap-x"
        >
          {topRatedList.map((media) => (
            <div key={media.id} className="w-[calc((100vw-3.75rem)/2)] shrink-0 snap-start sm:w-56 lg:w-60">
              <UnifiedMediaCard
                media={media}
                onOpen={onOpenMedia}
                onQuickPlay={onQuickPlay}
              />
            </div>
          ))}
        </div>
      </div>

      {/* 4. Carousel: Trending Series & Drama */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg sm:text-xl font-black text-white flex items-center gap-2">
              <span className="text-fuchsia-400">📺</span>
              المسلسلات والدراما الأكثر إقبالاً
            </h2>
            <p className="text-xs text-gray-400">حلقات ومواسم كاملة جاهزة للمشاهدة المباشرة</p>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => scroll(seriesRef, "right")}
              className="p-2 rounded-xl bg-slate-800/80 hover:bg-slate-700 text-white transition border border-slate-700"
            >
              ›
            </button>
            <button
              onClick={() => scroll(seriesRef, "left")}
              className="p-2 rounded-xl bg-slate-800/80 hover:bg-slate-700 text-white transition border border-slate-700"
            >
              ‹
            </button>
          </div>
        </div>

        <div
          ref={seriesRef}
          className="flex items-stretch gap-4 overflow-x-auto pb-4 scrollbar-none snap-x"
        >
          {seriesList.map((media) => (
            <div key={media.id} className="w-[calc((100vw-3.75rem)/2)] shrink-0 snap-start sm:w-56 lg:w-60">
              <UnifiedMediaCard
                media={media}
                onOpen={onOpenMedia}
                onQuickPlay={onQuickPlay}
              />
            </div>
          ))}
        </div>
      </div>

      {/* 5. Carousel: Blockbuster Movies */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg sm:text-xl font-black text-white flex items-center gap-2">
              <span className="text-cyan-400">🎬</span>
              الأفلام والسينما (Blockbuster Cinema)
            </h2>
            <p className="text-xs text-gray-400">أحدث الإصدارات العالمية والأجنبية</p>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => scroll(moviesRef, "right")}
              className="p-2 rounded-xl bg-slate-800/80 hover:bg-slate-700 text-white transition border border-slate-700"
            >
              ›
            </button>
            <button
              onClick={() => scroll(moviesRef, "left")}
              className="p-2 rounded-xl bg-slate-800/80 hover:bg-slate-700 text-white transition border border-slate-700"
            >
              ‹
            </button>
          </div>
        </div>

        <div
          ref={moviesRef}
          className="flex items-stretch gap-4 overflow-x-auto pb-4 scrollbar-none snap-x"
        >
          {moviesList.map((media) => (
            <div key={media.id} className="w-[calc((100vw-3.75rem)/2)] shrink-0 snap-start sm:w-56 lg:w-60">
              <UnifiedMediaCard
                media={media}
                onOpen={onOpenMedia}
                onQuickPlay={onQuickPlay}
              />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
