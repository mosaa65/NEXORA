import Icon from "../components/Icon.jsx";

const defaultSeasons = [
  { number: 1, title: "الموسم الأول", episodeCount: 25, thumb: "/images/attack_on_titan_poster.png" },
  { number: 2, title: "الموسم الثاني", episodeCount: 12, thumb: "/images/demon_slayer_poster.png" },
  { number: 3, title: "الموسم الثالث", episodeCount: 22, thumb: "/images/jujutsu_kaisen_poster.png" },
  { number: 4, title: "الموسم الرابع", episodeCount: 28, thumb: "/images/one_piece_poster.png" },
  { number: 5, title: "الموسم الأخير", episodeCount: 16, thumb: "/images/tokyo_ghoul_hero.png" }
];

export default function MediaDetailsPage({
  media,
  searchQuery = "",
  onSearchChange = () => {},
  onOpenCategory,
  onQuickPlay
}) {
  const current = media || {
    titleAr: "هجوم العمالقة",
    titleEn: "Attack on Titan",
    year: 2013,
    resolution: "1080p",
    duration: "24 دقيقة",
    rating: 9.0,
    views: "2.3M",
    plot: "منذ مائة عام، ظهرت العمالقة فجأة ودمرت معظم البشرية. يعيش الباقون في عالم محاط بأسوار ضخمة لحمايتهم من العمالقة... عندما يُخترق السور الأول، يبدأ إيرين غيغار رحلة الانتقام والبحث عن الحقيقة.",
    posterPath: "/images/attack_on_titan_poster.png",
    bannerPath: "/images/aot_banner_detail.png",
    highlights: ["خيال مظلم", "دراما", "أكشن"],
    seasons: defaultSeasons
  };

  const seasonsList = Array.isArray(current.seasons) && current.seasons.length > 0
    ? current.seasons.map((s, idx) => {
        if (typeof s === "object" && s !== null && s.title) return s;
        const numNames = ["الأول", "الثاني", "الثالث", "الرابع", "الأخير"];
        return {
          number: Number(s) || idx + 1,
          title: `الموسم ${numNames[idx] || (idx + 1)}`,
          episodeCount: 12 + idx * 4,
          thumb: current.posterPath || "/images/attack_on_titan_poster.png"
        };
      })
    : defaultSeasons;

  return (
    <div className="relative space-y-6 text-right">
      {/* Top Header Bar Integrated Overlay (Matching Top-Right Reference View) */}
      <div className="flex items-center justify-between gap-4 border-b border-white/10 pb-4">
        {/* Left: Back Button */}
        <button
          type="button"
          onClick={() => onOpenCategory("anime")}
          className="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-xs font-bold text-white backdrop-blur-md transition hover:bg-white/20"
        >
          العودة ‹
        </button>

        {/* Center: Search Input Bar */}
        <div className="relative flex-1 max-w-sm">
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

        {/* Right: Brand Emblem "مكتبتي / نظام إدارة الوسائط" */}
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

      {/* Main Details Showcase Banner */}
      <div className="relative overflow-hidden rounded-2xl border border-white/10 bg-[#0B0A16] p-6 sm:p-8 lg:p-10 shadow-panel">
        {/* Background Artwork */}
        <div
          className="absolute inset-0 bg-cover bg-center opacity-40 transition-all duration-1000"
          style={{ backgroundImage: `url('${current.bannerPath || "/images/aot_banner_detail.png"}')` }}
        />
        <div className="absolute inset-0 bg-gradient-to-t from-[#0B0A16] via-[#0B0A16]/80 to-transparent" />
        <div className="absolute inset-0 bg-gradient-to-r from-[#0B0A16]/95 via-[#0B0A16]/60 to-transparent" />

        <div className="relative z-10 grid gap-8 md:grid-cols-[auto_1fr] items-start">
          {/* Fixed Poster Card Width (Max 240px - No Stretching) */}
          <div className="w-48 sm:w-56 shrink-0 overflow-hidden rounded-2xl border-2 border-white/20 shadow-2xl shadow-purple-950/60 mx-auto md:mx-0">
            <img
              src={current.posterPath || "/images/attack_on_titan_poster.png"}
              alt={current.titleAr}
              className="h-full w-full object-cover aspect-[2/3]"
              onError={(e) => {
                e.target.src = "/images/attack_on_titan_poster.png";
              }}
            />
          </div>

          {/* Media Info & Metadata Text */}
          <div className="flex flex-col gap-4 text-right">
            <div>
              <h1 className="text-3xl font-black text-white sm:text-4xl lg:text-5xl tracking-tight">
                {current.titleAr}
              </h1>
              <p className="mt-1 text-sm font-bold text-white/60 sm:text-base">{current.titleEn}</p>
            </div>

            {/* Metadata Badges Row */}
            <div className="flex flex-wrap items-center justify-end gap-2 text-xs font-bold text-white/80">
              <span className="rounded-md bg-white/10 px-2.5 py-1">{current.year || 2013}</span>
              <span className="rounded-md bg-white/10 px-2.5 py-1">TV-MA</span>
              <span className="rounded-md bg-white/10 px-2.5 py-1">{current.duration || "24 دقيقة"}</span>
              {(current.highlights || ["خيال مظلم", "دراما", "أكشن"]).map((h) => (
                <span key={h} className="rounded-md border border-purple-500/30 bg-purple-950/40 px-2.5 py-1 text-fuchsia-300">
                  {h}
                </span>
              ))}
              <span className="flex items-center gap-1 text-yellow-400 font-extrabold mr-2">
                ★ {current.rating?.toFixed?.(1) || "9.0"}
              </span>
              <span className="text-white/40">👁 {current.views || "2.3M"}</span>
            </div>

            {/* Plot Synopsis Paragraph */}
            <p className="text-xs leading-relaxed text-white/75 sm:text-sm max-w-2xl">
              {current.plot}
            </p>

            {/* Action Buttons */}
            <div className="mt-2 flex flex-wrap items-center justify-end gap-3">
              <button
                type="button"
                onClick={() => onQuickPlay(current)}
                className="inline-flex items-center gap-2 rounded-xl bg-gradient-to-r from-fuchsia-600 via-purple-600 to-fuchsia-700 px-7 py-3 text-xs font-black text-white shadow-lg shadow-purple-900/60 transition hover:scale-105 sm:text-sm"
              >
                <Icon name="play" className="h-4 w-4 text-white fill-current shrink-0" />
                <span>تشغيل الآن</span>
              </button>

              <button
                type="button"
                className="inline-flex items-center gap-2 rounded-xl border border-white/20 bg-white/10 px-5 py-3 text-xs font-bold text-white backdrop-blur-md transition hover:bg-white/20 sm:text-sm"
              >
                المفضلة ♡
              </button>

              <button
                type="button"
                className="inline-flex items-center gap-2 rounded-xl border border-white/20 bg-white/10 px-5 py-3 text-xs font-bold text-white backdrop-blur-md transition hover:bg-white/20 sm:text-sm"
              >
                المشاركة 🔗
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Seasons Section ("المواسم") - Matching Reference Image Grid Layout */}
      <section className="space-y-3">
        <div className="border-b border-white/10 pb-2">
          <h2 className="text-lg font-black text-white sm:text-xl">المواسم</h2>
        </div>

        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-5">
          {seasonsList.map((season) => (
            <button
              key={season.number}
              type="button"
              onClick={() => onQuickPlay(current)}
              className="group flex items-center gap-3 rounded-2xl border border-white/10 bg-black/40 p-3 text-right transition hover:-translate-y-0.5 hover:border-fuchsia-500/50 hover:bg-black/60 shadow-md"
            >
              <div className="h-12 w-12 shrink-0 rounded-xl overflow-hidden bg-black/60 border border-white/10">
                <img
                  src={season.thumb || current.posterPath || "/images/attack_on_titan_poster.png"}
                  alt={season.title}
                  className="h-full w-full object-cover group-hover:scale-105 transition-transform duration-300"
                />
              </div>
              <div className="min-w-0 flex-1">
                <p className="truncate text-xs font-bold text-white group-hover:text-fuchsia-300 transition-colors">
                  {season.title}
                </p>
                <p className="mt-0.5 text-[10px] font-medium text-white/50">{season.episodeCount} حلقة</p>
              </div>
            </button>
          ))}
        </div>
      </section>
    </div>
  );
}
