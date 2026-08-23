import React, { useEffect, useRef, useState } from "react";
import HeroSlider from "../components/HeroSlider.jsx";
import UnifiedMediaCard from "../components/UnifiedMediaCard.jsx";
import HubBannerCard from "../components/HubBannerCard.jsx";
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
            titleAr: item.title_ar || item.title_en,
            titleEn: item.title_en,
            type: item.type,
            plot: item.plot_ar || item.plot_en || "عمل سينمائي متاح على شبكة NEXORA المحلية.",
            year: item.release_year || 2024,
            rating: item.rating || 8.5,
            resolution: "1080p",
            posterPath: item.poster_path,
            bannerPath: item.banner_path,
            categorySlug: item.category_slug,
            fileCount: item.file_count || 1,
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
      {/* 1. Grand Hero Showcase Slider */}
      {heroItems.length > 0 && (
        <HeroSlider
          items={heroItems}
          onOpenMedia={onOpenMedia}
          onQuickPlay={onQuickPlay}
        />
      )}

      {/* 2. Featured Hub Banners Grid */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-black text-white flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full bg-fuchsia-500 shadow-[0_0_12px_#e879f9]" />
            مجموعات الاستراحة الحصرية
          </h2>
          <span className="text-xs font-bold text-gray-400">تصفح سريع</span>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <HubBannerCard
            title="👑 روائع الدراما التركية"
            subtitle="الحكايات العثمانية والدراما العائلية الرومانسية الكاملة"
            count={seriesList.length}
            backdrop="/images/aot_banner_detail.png"
            accentColor="from-amber-800/90 via-orange-950/80 to-[#0C0B17]"
            borderColor="border-amber-500/40"
            tag="الأكثر طلباً"
            onClick={() => onNavigateCategory && onNavigateCategory("series")}
          />
          <HubBannerCard
            title="🎬 أفلام هوليوود والعالم"
            subtitle="أحدث أفلام الأكشن والخيال العلمي والإثارة بدقة 4K"
            count={moviesList.length}
            backdrop="/images/demon_slayer_poster.png"
            accentColor="from-cyan-800/90 via-blue-950/80 to-[#0C0B17]"
            borderColor="border-cyan-500/40"
            tag="سينما 4K"
            onClick={() => onNavigateCategory && onNavigateCategory("movies")}
          />
          <HubBannerCard
            title="⚔️ عوالم الأنمي الأسطورية"
            subtitle="ون بيس، هجوم العمالقة، وجوجوتسو كايسن بمواسمها الكاملة"
            count={seriesList.length}
            backdrop="/images/jujutsu_kaisen_poster.png"
            accentColor="from-purple-800/90 via-fuchsia-950/80 to-[#0C0B17]"
            borderColor="border-purple-500/40"
            tag="Anime Top"
            onClick={() => onNavigateCategory && onNavigateCategory("anime")}
          />
          <HubBannerCard
            title="🏰 ديزني والأطفال العائلي"
            subtitle="أفلام الرسوم المتحركة وسلاسل الأبطال الخارقين والكرتون"
            count={moviesList.length}
            backdrop="/images/naruto_poster.png"
            accentColor="from-sky-800/90 via-indigo-950/80 to-[#0C0B17]"
            borderColor="border-sky-500/40"
            tag="عائلي وأطفال"
            onClick={() => onNavigateCategory && onNavigateCategory("kids")}
          />
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
            <div key={media.id} className="w-40 sm:w-48 shrink-0 snap-start">
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
            <div key={media.id} className="w-40 sm:w-48 shrink-0 snap-start">
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
            <div key={media.id} className="w-40 sm:w-48 shrink-0 snap-start">
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
