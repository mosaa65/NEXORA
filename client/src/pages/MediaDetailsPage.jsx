import { useEffect, useState } from "react";
import Icon from "../components/Icon.jsx";
import { getMediaDetail, enrichMedia } from "../lib/api.js";

export default function MediaDetailsPage({
  media,
  searchQuery = "",
  onSearchChange = () => {},
  onOpenCategory,
  onQuickPlay
}) {
  const [detail, setDetail] = useState(null);
  const [selectedSeasonIdx, setSelectedSeasonIdx] = useState(0);
  const [isEnriching, setIsEnriching] = useState(false);
  const [enrichMsg, setEnrichMsg] = useState("");

  const current = detail || media || {
    id: 1,
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
    seasons: []
  };

  useEffect(() => {
    let alive = true;
    if (media?.id) {
      getMediaDetail(media.id)
        .then((data) => {
          if (!alive) return;
          if (data && data.id) {
            setDetail({
              id: data.id,
              titleAr: data.title_ar || data.title_en || media.titleAr,
              titleEn: data.title_en || media.titleEn,
              type: data.type || media.type,
              year: data.release_year || media.year || 2023,
              rating: data.rating || media.rating || 8.5,
              plot: data.plot_ar || data.plot_en || media.plot || "عمل سينمائي مميز متاح في مكتبة NEXORA المحلية.",
              posterPath: data.poster_path || media.posterPath,
              bannerPath: data.banner_path || media.bannerPath,
              categorySlug: data.category_slug || media.categorySlug,
              highlights: data.genres?.length > 0 ? data.genres : ["دراما", "أكشن", "إثارة"],
              seasons: data.seasons || [],
              files: data.files || []
            });
          }
        })
        .catch(() => {});
    }

    return () => {
      alive = false;
    };
  }, [media]);

  async function handleEnrichMetadata() {
    if (!current?.id) return;
    setIsEnriching(true);
    setEnrichMsg("جارٍ جلب البيانات والبوستر...");
    try {
      const res = await enrichMedia(current.id);
      if (res.ok && res.metadata) {
        setEnrichMsg("✅ تم تحديث البيانات والبوستر بنجاح!");
        setDetail((prev) => ({
          ...prev,
          titleAr: res.metadata.title || prev.titleAr,
          plot: res.metadata.overview || prev.plot,
          rating: res.metadata.rating || prev.rating,
          year: res.metadata.releaseYear || prev.year,
          posterPath: res.metadata.cachedPosterPath || res.metadata.posterPath || prev.posterPath,
          bannerPath: res.metadata.cachedBannerPath || res.metadata.bannerPath || prev.bannerPath,
          highlights: res.metadata.genres?.length > 0 ? res.metadata.genres : prev.highlights
        }));
      } else {
        setEnrichMsg("لم يتم العثور على تطابق.");
      }
    } catch (err) {
      setEnrichMsg("تعذر الاتصال بمزود البيانات.");
    } finally {
      setIsEnriching(false);
      setTimeout(() => setEnrichMsg(""), 4000);
    }
  }

  const seasonsList = current.seasons && current.seasons.length > 0 ? current.seasons : [];
  const currentSeason = seasonsList[selectedSeasonIdx] || seasonsList[0];
  const activeEpisodes = currentSeason?.episodes || current.files || [];

  return (
    <div className="relative space-y-6 text-right">
      {/* Top Header Bar */}
      <div className="flex items-center justify-between gap-4 border-b border-white/10 pb-4">
        {/* Left: Back Button */}
        <button
          type="button"
          onClick={() => onOpenCategory(current.categorySlug || "anime")}
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

      {/* Main Details Showcase Banner */}
      <div className="relative overflow-hidden rounded-2xl border border-white/10 bg-[#0B0A16] p-6 sm:p-8 lg:p-10 shadow-panel">
        {/* Background Artwork */}
        <div
          className="absolute inset-0 bg-cover bg-center opacity-40 transition-all duration-1000"
          style={{ backgroundImage: `url('${current.bannerPath || current.posterPath || "/images/aot_banner_detail.png"}')` }}
        />
        <div className="absolute inset-0 bg-gradient-to-t from-[#0B0A16] via-[#0B0A16]/80 to-transparent" />
        <div className="absolute inset-0 bg-gradient-to-r from-[#0B0A16]/95 via-[#0B0A16]/60 to-transparent" />

        <div className="relative z-10 grid gap-8 md:grid-cols-[auto_1fr] items-start">
          {/* Fixed Poster Card */}
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
              <span className="rounded-md bg-white/10 px-2.5 py-1">{current.year || 2023}</span>
              <span className="rounded-md bg-white/10 px-2.5 py-1">HD</span>
              {(current.highlights || ["دراما", "أكشن"]).map((h) => (
                <span key={h} className="rounded-md border border-purple-500/30 bg-purple-950/40 px-2.5 py-1 text-fuchsia-300">
                  {h}
                </span>
              ))}
              <span className="flex items-center gap-1 text-yellow-400 font-extrabold mr-2">
                ★ {Number(current.rating || 8.5).toFixed(1)}
              </span>
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
                onClick={handleEnrichMetadata}
                disabled={isEnriching}
                className="inline-flex items-center gap-2 rounded-xl border border-fuchsia-500/40 bg-fuchsia-950/40 px-5 py-3 text-xs font-bold text-fuchsia-300 backdrop-blur-md transition hover:bg-fuchsia-900/50 sm:text-sm disabled:opacity-50"
              >
                <span>✨ جلب الغلاف والقصة (TMDB/MAL)</span>
              </button>
            </div>

            {enrichMsg && (
              <p className="text-xs font-bold text-fuchsia-300 animate-pulse">{enrichMsg}</p>
            )}
          </div>
        </div>
      </div>

      {/* Seasons Tab & Episode List */}
      {seasonsList.length > 0 && (
        <section className="space-y-4">
          <div className="border-b border-white/10 pb-2">
            <h2 className="text-lg font-black text-white sm:text-xl">المواسم والحلقات</h2>
          </div>

          {/* Season Selector Tabs */}
          <div className="flex flex-wrap gap-2">
            {seasonsList.map((season, idx) => (
              <button
                key={season.id || idx}
                type="button"
                onClick={() => setSelectedSeasonIdx(idx)}
                className={`rounded-xl px-4 py-2 text-xs font-bold transition ${
                  selectedSeasonIdx === idx
                    ? "bg-gradient-to-r from-purple-800 to-fuchsia-700 text-white shadow-md shadow-purple-900/40"
                    : "border border-white/10 bg-white/5 text-white/70 hover:bg-white/10 hover:text-white"
                }`}
              >
                {season.title_ar || season.title_en || `الموسم ${season.season_number || idx + 1}`}
                <span className="mr-1.5 opacity-60">({season.episodes?.length || 0} حلقة)</span>
              </button>
            ))}
          </div>

          {/* Episodes Grid */}
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {activeEpisodes.map((ep, epIdx) => (
              <button
                key={ep.id || epIdx}
                type="button"
                onClick={() => onQuickPlay(current, ep)}
                className="group flex items-center justify-between gap-3 rounded-xl border border-white/10 bg-black/40 p-3 text-right transition hover:border-fuchsia-500/50 hover:bg-black/60 shadow-sm"
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-fuchsia-600/20 text-fuchsia-400 group-hover:bg-fuchsia-600 group-hover:text-white transition">
                  ▶
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-xs font-bold text-white group-hover:text-fuchsia-300">
                    {ep.title_ar || ep.title_en || `الحلقة ${ep.episode_number || epIdx + 1}`}
                  </p>
                  <p className="mt-0.5 text-[10px] font-medium text-white/50">
                    {ep.resolution || "1080p"} • {ep.video_codec || "HEVC/H264"}
                  </p>
                </div>
              </button>
            ))}
          </div>
        </section>
      )}

      {/* Direct Video Files list if no seasons */}
      {seasonsList.length === 0 && current.files && current.files.length > 0 && (
        <section className="space-y-4">
          <div className="border-b border-white/10 pb-2">
            <h2 className="text-lg font-black text-white sm:text-xl">ملفات الفيديو المتاحة</h2>
          </div>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {current.files.map((file, idx) => (
              <button
                key={file.id || idx}
                type="button"
                onClick={() => onQuickPlay(current, file)}
                className="group flex items-center justify-between gap-3 rounded-xl border border-white/10 bg-black/40 p-3 text-right transition hover:border-fuchsia-500/50 hover:bg-black/60 shadow-sm"
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-fuchsia-600/20 text-fuchsia-400 group-hover:bg-fuchsia-600 group-hover:text-white transition">
                  ▶
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-xs font-bold text-white group-hover:text-fuchsia-300">
                    {file.title_ar || file.title_en || `ملف التشغيل #${idx + 1}`}
                  </p>
                  <p className="mt-0.5 text-[10px] font-medium text-white/50">
                    {file.resolution || "1080p"} • {(Number(file.file_size || 0) / (1024 * 1024)).toFixed(1)} MB
                  </p>
                </div>
              </button>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
