import Icon from "../components/Icon.jsx";

export default function MediaDetailsPage({ media, onOpenCategory, onQuickPlay }) {
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
    bannerPath: "/images/attack_on_titan_poster.png",
    highlights: ["خيال مظلم", "دراما", "أكشن"],
    seasons: [
      { number: 1, title: "الموسم الأول", episodeCount: 25 },
      { number: 2, title: "الموسم الثاني", episodeCount: 12 },
      { number: 3, title: "الموسم الثالث", episodeCount: 22 },
      { number: 4, title: "الموسم الرابع", episodeCount: 28 },
      { number: 5, title: "الموسم الأخير", episodeCount: 16 }
    ]
  };

  const seasonsList = Array.isArray(current.seasons) && typeof current.seasons[0] === "object"
    ? current.seasons
    : [
        { number: 1, title: "الموسم الأول", episodeCount: 25 },
        { number: 2, title: "الموسم الثاني", episodeCount: 12 },
        { number: 3, title: "الموسم الثالث", episodeCount: 22 },
        { number: 4, title: "الموسم الرابع", episodeCount: 28 },
        { number: 5, title: "الموسم الأخير", episodeCount: 16 }
      ];

  return (
    <div className="space-y-8 text-right">
      {/* Top Controls Row */}
      <div className="flex items-center justify-between">
        <button
          type="button"
          onClick={() => onOpenCategory("anime")}
          className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-2 text-xs font-bold text-white/80 transition hover:bg-white/10"
        >
          العودة ‹
        </button>
      </div>

      {/* Main Showcase Hero Section */}
      <div className="relative overflow-hidden rounded-[2rem] border border-white/10 bg-[#0B0A16] p-6 sm:p-8 lg:p-10 shadow-panel">
        <div
          className="absolute inset-0 bg-cover bg-center opacity-30 blur-xl"
          style={{ backgroundImage: `url('${current.posterPath || "/images/attack_on_titan_poster.png"}')` }}
        />
        <div className="absolute inset-0 bg-gradient-to-t from-[#0B0A16] via-[#0B0A16]/80 to-transparent" />

        <div className="relative z-10 grid gap-8 lg:grid-cols-[0.8fr_1.2fr] items-center">
          {/* Left Poster Card Artwork */}
          <div className="mx-auto max-w-xs overflow-hidden rounded-2xl border-2 border-fuchsia-500/30 shadow-2xl shadow-purple-900/50 lg:mx-0">
            <img
              src={current.posterPath || "/images/attack_on_titan_poster.png"}
              alt={current.titleAr}
              className="h-full w-full object-cover"
              onError={(e) => {
                e.target.src = "/images/attack_on_titan_poster.png";
              }}
            />
          </div>

          {/* Right Media Details Text & Actions */}
          <div className="flex flex-col gap-4 text-right">
            <div>
              <h1 className="text-4xl font-black text-white sm:text-5xl lg:text-6xl tracking-tight">
                {current.titleAr}
              </h1>
              <p className="mt-2 text-lg font-bold text-white/60">{current.titleEn}</p>
            </div>

            {/* Chips & Badges Row */}
            <div className="flex flex-wrap items-center justify-end gap-2.5 text-xs font-bold text-white/80">
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

            {/* Synopsis Paragraph */}
            <p className="text-sm leading-relaxed text-white/75 sm:text-base max-w-2xl">
              {current.plot}
            </p>

            {/* Action Buttons */}
            <div className="mt-2 flex flex-wrap items-center justify-end gap-3.5">
              <button
                type="button"
                onClick={() => onQuickPlay(current)}
                className="inline-flex items-center gap-2.5 rounded-full bg-gradient-to-r from-fuchsia-600 via-purple-600 to-fuchsia-700 px-8 py-3.5 text-sm font-black text-white shadow-lg shadow-purple-900/60 transition hover:scale-105"
              >
                تشغيل الآن ▶
              </button>
              <button
                type="button"
                className="inline-flex items-center gap-2 rounded-full border border-white/20 bg-white/10 px-6 py-3.5 text-sm font-bold text-white backdrop-blur-md transition hover:bg-white/20"
              >
                المفضلة ♡
              </button>
              <button
                type="button"
                className="inline-flex items-center gap-2 rounded-full border border-white/20 bg-white/10 px-6 py-3.5 text-sm font-bold text-white backdrop-blur-md transition hover:bg-white/20"
              >
                المشاركة 🔗
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Seasons Section ("المواسم") */}
      <section className="space-y-4">
        <div className="border-b border-white/10 pb-3">
          <h2 className="text-xl font-black text-white sm:text-2xl">المواسم</h2>
        </div>

        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
          {seasonsList.map((season) => (
            <button
              key={season.number}
              type="button"
              onClick={() => onQuickPlay(current)}
              className="group flex flex-col justify-between rounded-2xl border border-white/10 bg-[#0D0E18] p-4 text-right transition hover:-translate-y-1 hover:border-fuchsia-500/50 hover:shadow-lg hover:shadow-purple-900/30"
            >
              <div className="aspect-video w-full rounded-xl overflow-hidden bg-black/40 mb-3 relative">
                <img
                  src={current.posterPath || "/images/attack_on_titan_poster.png"}
                  alt={season.title}
                  className="h-full w-full object-cover opacity-70 group-hover:scale-105 transition-transform duration-500"
                />
                <div className="absolute inset-0 grid place-items-center bg-black/30 group-hover:bg-purple-900/40 transition-colors">
                  <span className="text-xs font-bold text-white">▶</span>
                </div>
              </div>
              <p className="font-bold text-white text-sm group-hover:text-fuchsia-300">{season.title}</p>
              <p className="text-xs text-white/50 mt-1">{season.episodeCount} حلقة</p>
            </button>
          ))}
        </div>
      </section>
    </div>
  );
}
