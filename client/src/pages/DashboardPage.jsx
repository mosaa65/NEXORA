import { useRef } from "react";
import MediaCard from "../components/MediaCard.jsx";
import Icon from "../components/Icon.jsx";
import { buildHeroCopy, dashboardMetrics, mockLibrary } from "../data/library.js";

export default function DashboardPage({
  searchQuery,
  onSearchChange,
  searchResults,
  onOpenMedia,
  onQuickPlay,
  onToggleSidebar
}) {
  const hero = buildHeroCopy();
  const primaryPick = mockLibrary.find((item) => item.id === 1) || mockLibrary[0];
  const mostWatched = mockLibrary;
  const carouselRef = useRef(null);

  function scrollCarousel(direction) {
    if (carouselRef.current) {
      const scrollAmount = direction === "left" ? -280 : 280;
      carouselRef.current.scrollBy({ left: scrollAmount, behavior: "smooth" });
    }
  }

  return (
    <div className="space-y-6">
      {/* Hero Showcase Container with Header Integrated Overlay */}
      <section className="relative overflow-hidden rounded-2xl border border-white/10 bg-[#0B0A16]">
        {/* Full Image Background extending to top header */}
        <div
          className="absolute inset-0 bg-cover bg-right sm:bg-center opacity-85 transition-all duration-1000"
          style={{ backgroundImage: `url('${primaryPick?.posterPath || "/images/tokyo_ghoul_hero.png"}')` }}
        />
        <div className="absolute inset-0 bg-gradient-to-t from-[#0B0A16] via-[#0B0A16]/40 to-transparent" />
        <div className="absolute inset-0 bg-gradient-to-l from-[#0B0A16]/95 via-[#0B0A16]/50 to-transparent" />

        {/* Integrated Top Bar Header (Transparent Over Hero Image, NO border, NO blur) */}
        <div className="relative z-20 flex items-center justify-between gap-4 p-4 sm:p-6 bg-transparent border-none">
          {/* Left: Shorter & Fully Rounded Search Bar */}
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

          {/* Right: Brand Emblem "مكتبتي / نظام إدارة الوسائط" (Pure Icon without BG color fill) */}
          <div className="flex items-center gap-3">
            <div className="text-right">
              <h2 className="text-lg font-black text-white">مكتبتي</h2>
              <p className="text-[10px] font-bold text-white/50">نظام إدارة الوسائط</p>
            </div>
            {/* Pure Triangle SVG Icon with NO background fill */}
            <div className="flex items-center justify-center text-fuchsia-400">
              <svg className="h-7 w-7 stroke-current" fill="none" viewBox="0 0 24 24" strokeWidth="2.2">
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 3L2 21h20L12 3z" />
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 9l-4 7h8l-4-7z" />
              </svg>
            </div>
          </div>
        </div>

        {/* Hero Banner Content (LEFT ALIGNED for Title, Text & Buttons) */}
        <div className="relative z-10 flex flex-col justify-end p-6 sm:p-10 lg:p-12 min-h-[22rem] sm:min-h-[26rem]">
          <div className="max-w-xl text-left">
            <h1 className="text-4xl font-black text-white sm:text-5xl lg:text-6xl tracking-tight leading-none">
              طوكيو غول
            </h1>
            <p className="mt-2 text-base font-bold text-white/70 sm:text-lg">Tokyo Ghoul</p>
            <p className="mt-4 max-w-lg text-xs leading-relaxed text-white/80 sm:text-sm text-left">
              في طوكيو حيث تعيش غيلان بين البشر بالتخفي، تنقلب حياة الشاب (كانيكي) عندما تلتهمه إحدى الغيلان بدلاً من أن تصبح عشاءه، فيتحول إلى نصف بشري ونصف غول محاصر بين عالمين.
            </p>

            {/* Action Buttons (LEFT ALIGNED & NOT 50% ROUNDED, SVG Play Icon) */}
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

      {/* "الأكثر مشاهدة" Carousel Section with Left & Right Scroll Buttons */}
      <section className="space-y-3">
        <div className="flex items-center justify-between border-b border-white/10 pb-2 text-right">
          <h2 className="text-lg font-black text-white sm:text-xl">الأكثر مشاهدة</h2>

          {/* Carousel Left & Right Arrows */}
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

        {/* Carousel Scroll Container (Controlled Compact Cards, Pure Images) */}
        <div
          ref={carouselRef}
          className="flex items-center gap-3 overflow-x-auto pb-2 scrollbar-none scroll-smooth"
        >
          {mostWatched.map((item, index) => (
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

      {/* Bottom Statistics Bar (Flat dark rounded panel) */}
      <section className="rounded-2xl border border-white/10 bg-[#0A0914] p-4 text-center">
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6 divide-x-0 sm:divide-x sm:divide-x-reverse sm:divide-white/10">
          {dashboardMetrics.map((stat) => (
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
