import { useEffect, useRef, useState } from "react";
import MediaCard from "../components/MediaCard.jsx";
import Icon from "../components/Icon.jsx";
import { getDashboardStats, getMediaList, resolveAPIURL } from "../lib/api.js";
import { mockLibrary } from "../data/library.js";
import { horizontalWheel } from "../lib/horizontalScroll.js";

export default function DashboardPage({
  searchQuery,
  onSearchChange,
  searchResults,
  onOpenMedia,
  onQuickPlay,
  onToggleSidebar
}) {
  const [items, setItems] = useState([]);
  const [stats, setStats] = useState(null);
  const carouselRef = useRef(null);

  useEffect(() => {
    let alive = true;

    getMediaList({ limit: 30, sort: "rating" })
      .then((data) => {
        if (!alive) return;
        if (data.items && data.items.length > 0) {
          const transformed = data.items.map((item) => ({
            id: item.id,
            titleAr: item.title_ar || item.title_en,
            titleEn: item.title_en,
            type: item.type,
            plot: item.plot_ar || item.plot_en || "عمل سينمائي مميز متاح في مكتبة NEXORA المحلية.",
            year: item.release_year || 2023,
            rating: item.rating || 8.5,
            posterPath: item.poster_path,
            bannerPath: item.banner_path,
            categorySlug: item.category_slug,
            fileCount: item.file_count || 1
          }));
          setItems(transformed);
        } else {
          setItems(mockLibrary);
        }
      })
      .catch(() => {
        if (alive) setItems(mockLibrary);
      });

    getDashboardStats()
      .then((data) => {
        if (!alive) return;
        if (data) setStats(data);
      })
      .catch(() => {});

    return () => {
      alive = false;
    };
  }, []);

  const displayItems = searchResults?.length > 0 && searchQuery ? searchResults : items.length > 0 ? items : mockLibrary;
  const primaryPick = displayItems[0] || mockLibrary[0];
  const carouselItems = displayItems.slice(0, 15);
  const seriesItems = displayItems.filter((item) => item.type === "series" || item.type === "anime").slice(0, 12);

  function scrollCarousel(direction) {
    if (carouselRef.current) {
      const scrollAmount = direction === "left" ? -280 : 280;
      carouselRef.current.scrollBy({ left: scrollAmount, behavior: "smooth" });
    }
  }

  const metrics = [
    { label: "إجمالي الأعمال", value: stats?.total_media ? `${stats.total_media}` : `${displayItems.length}` },
    { label: "ملفات الفيديو", value: stats?.total_files ? `${stats.total_files}` : "133" },
    { label: "مساحة التخزين", value: stats?.total_storage_bytes ? `${(stats.total_storage_bytes / (1024 * 1024 * 1024)).toFixed(1)} GB` : "249 KB" },
    { label: "الأقسام النشطة", value: `${stats?.categories?.length || 6}` },
    { label: "الحلقات المفقودة", value: `${stats?.missing_episodes_count ?? 1}` },
    { label: "الملفات المكررة", value: `${stats?.duplicates_count ?? 2}` }
  ];

  return (
    <div className="space-y-6">
      {/* Hero Showcase Container with Header Integrated Overlay */}
      <section className="relative overflow-hidden rounded-2xl border border-white/10 bg-[#0B0A16]">
        {/* Full Image Background */}
        <div
          className="absolute inset-0 bg-cover bg-right sm:bg-center opacity-85 transition-all duration-1000"
          style={{ backgroundImage: `url('${resolveAPIURL(primaryPick?.bannerPath || primaryPick?.posterPath) || "/images/tokyo_ghoul_hero.png"}')` }}
        />
        <div className="absolute inset-0 bg-gradient-to-t from-[#0B0A16] via-[#0B0A16]/40 to-transparent" />
        <div className="absolute inset-0 bg-gradient-to-l from-[#0B0A16]/95 via-[#0B0A16]/50 to-transparent" />

        {/* Integrated Top Bar Header */}
        <div className="relative z-20 flex items-center justify-between gap-4 p-4 sm:p-6 bg-transparent border-none">
          {/* Left: Search Bar */}
          <div className="relative flex-1 max-w-xs sm:max-w-sm">
            <div className="flex w-full items-center gap-2.5 rounded-full border border-white/20 bg-black/40 px-4 py-2 backdrop-blur-sm focus-within:border-fuchsia-500/60">
              <Icon name="search" className="h-4 w-4 text-white/60 shrink-0" />
              <input
                value={searchQuery}
                onChange={(event) => onSearchChange(event.target.value)}
                placeholder="ابحث عن أنمي، فيلم، مسلسل..."
                className="w-full bg-transparent text-xs font-bold text-white outline-none placeholder:text-white/40 text-right"
              />
            </div>
          </div>

          {/* Right: Brand Emblem */}
          <div className="flex items-center gap-3">
            <div className="text-right">
              <h2 className="text-lg font-black text-white">مكتبتي</h2>
              <p className="text-[10px] font-bold text-white/50">نظام إدارة الوسائط</p>
            </div>
            <div className="flex items-center justify-center text-fuchsia-400">
              <svg className="h-7 w-7 stroke-current" fill="none" viewBox="0 0 24 24" strokeWidth="2.2">
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 3L2 21h20L12 3z" />
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 9l-4 7h8l-4-7z" />
              </svg>
            </div>
          </div>
        </div>

        {/* Hero Banner Content */}
        <div className="relative z-10 flex flex-col justify-end p-6 sm:p-10 lg:p-12 min-h-[22rem] sm:min-h-[26rem]">
          <div className="max-w-xl text-right md:text-left">
            <h1 className="text-4xl font-black text-white sm:text-5xl lg:text-6xl tracking-tight leading-none">
              {primaryPick.titleAr}
            </h1>
            <p className="mt-2 text-base font-bold text-white/70 sm:text-lg">{primaryPick.titleEn}</p>
            <p className="mt-4 max-w-lg text-xs leading-relaxed text-white/80 sm:text-sm text-right md:text-left">
              {primaryPick.plot}
            </p>

            {/* Action Buttons */}
            <div className="mt-6 flex flex-wrap items-center justify-start gap-3">
              <button
                type="button"
                onClick={() => onQuickPlay(primaryPick)}
                className="inline-flex items-center gap-2 rounded-xl bg-gradient-to-r from-fuchsia-600 via-purple-600 to-fuchsia-700 px-6 py-2.5 text-xs font-black text-white shadow-lg shadow-purple-900/50 transition hover:scale-105 sm:text-sm"
              >
                <Icon name="play" className="h-4 w-4 text-white fill-current shrink-0" />
                <span>تشغيل الآن</span>
              </button>

              <button
                type="button"
                onClick={() => onOpenMedia(primaryPick)}
                className="inline-flex items-center gap-2 rounded-xl border border-white/20 bg-white/10 px-5 py-2.5 text-xs font-bold text-white backdrop-blur-md transition hover:bg-white/20 sm:text-sm"
              >
                المزيد من المعلومات
              </button>
            </div>
          </div>
        </div>
      </section>

      {/* "الأكثر مشاهدة / الأحدث" Carousel Section */}
      <section className="space-y-3">
        <div className="flex items-center justify-between border-b border-white/10 pb-2 text-right">
          <h2 className="text-lg font-black text-white sm:text-xl">أحدث الأعمال المفهرسة</h2>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => scrollCarousel("right")}
              className="flex h-8 w-8 items-center justify-center rounded-lg border border-white/15 bg-white/5 text-white/80 transition hover:bg-white/15"
              title="التالي"
            >
              ›
            </button>
            <button
              type="button"
              onClick={() => scrollCarousel("left")}
              className="flex h-8 w-8 items-center justify-center rounded-lg border border-white/15 bg-white/5 text-white/80 transition hover:bg-white/15"
              title="السابق"
            >
              ‹
            </button>
          </div>
        </div>

        {/* Carousel Scroll Container */}
        <div
          ref={carouselRef}
          onWheel={horizontalWheel}
          className="flex items-center gap-3 overflow-x-auto pb-2 scrollbar-none scroll-smooth"
        >
          {carouselItems.map((item, index) => (
            <div key={item.id} className="shrink-0">
              <MediaCard
                item={item}
                onOpen={onOpenMedia}
                index={index}
                compact
              />
            </div>
          ))}
        </div>
      </section>

      {seriesItems.length > 0 && <section className="space-y-3">
        <div className="flex items-center justify-between border-b border-white/10 pb-2 text-right"><h2 className="text-lg font-black text-white sm:text-xl">مواسم وحلقات متاحة</h2><p className="text-xs text-white/45">اختر العمل لاستعراض مواسمه وحلقاته</p></div>
        <div onWheel={horizontalWheel} className="flex items-center gap-3 overflow-x-auto pb-2 scrollbar-none scroll-smooth">{seriesItems.map((item, index) => <div key={item.id} className="shrink-0"><MediaCard item={item} onOpen={onOpenMedia} index={index} compact /></div>)}</div>
      </section>}

      {/* Bottom Statistics Bar */}
      <section className="rounded-2xl border border-white/10 bg-[#0A0914] p-4 text-center">
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6 divide-x-0 sm:divide-x sm:divide-x-reverse sm:divide-white/10">
          {metrics.map((stat) => (
            <div key={stat.label} className="p-1">
              <p className="text-[11px] font-bold text-white/40">{stat.label}</p>
              <p className="mt-1 text-lg font-black text-white">{stat.value}</p>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
