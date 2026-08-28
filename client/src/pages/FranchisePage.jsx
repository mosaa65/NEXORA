import { useEffect, useState } from "react";
import Icon from "../components/Icon.jsx";
import MediaCollection from "../components/MediaCollection.jsx";
import PageBackButton from "../components/PageBackButton.jsx";
import { getFranchiseMedia, resolveAPIURL } from "../lib/api.js";

export default function FranchisePage({ slug, onOpenMedia }) {
  const [data, setData] = useState(null);
  const [error, setError] = useState("");
  useEffect(() => { let alive = true; setData(null); setError(""); getFranchiseMedia(slug, { limit: 100 }).then((payload) => { if (alive) setData(payload); }).catch((err) => { if (alive) setError(err.message || "تعذر تحميل السلسلة"); }); return () => { alive = false; }; }, [slug]);
  if (error) return <div className="rounded-xl border border-rose-400/30 bg-rose-950/15 p-7 text-center text-sm font-bold text-rose-200" dir="rtl">{error}</div>;
  if (!data) return <div className="h-72 animate-pulse rounded-xl border border-white/10 bg-white/[0.03]" aria-label="جارٍ تحميل السلسلة" />;
  const franchise = data.franchise || {};
  const title = `${franchise.title_ar || franchise.title_en || "سلسلة أفلام"}${Number(franchise.rating || 0) > 0 ? `  (★ ${Number(franchise.rating).toFixed(1)})` : ""}`;
  const artwork = resolveAPIURL(franchise.backdrop_path || franchise.poster_path) || "/nexora-library-backdrop.PNG";
  const items = (data.items || []).map((item) => ({ id:item.id, titleAr:item.title_ar, titleEn:item.title_en, type:item.type, year:item.release_year, rating:item.rating, posterPath:item.poster_path, status:item.status, seasonCount:item.season_count, tmdbSeasonCount:item.tmdb_season_count, totalSize:item.total_size, bestResolution:item.best_resolution, runtimeMinutes:item.runtime_minutes, hasArabicAudio:item.has_arabic_audio, hasArabicSubtitles:item.has_arabic_subtitles }));
  const parts = data.parts || [];
  const pendingParts = parts.filter((part) => !part.local);
  return (
    <div className="space-y-7 pb-16" dir="rtl">
      <section className="relative min-h-[310px] overflow-hidden rounded-xl border border-white/10 bg-[#12101d] sm:min-h-[370px]">
        <img src={artwork} alt="" className="absolute inset-0 h-full w-full object-cover" onError={(event) => { event.currentTarget.src = "/nexora-library-backdrop.PNG"; }}/>
        <div className="absolute inset-0 bg-gradient-to-l from-black/95 via-black/65 to-black/20"/>
        <div className="absolute right-4 top-4 z-10 sm:right-6 sm:top-6">
          <PageBackButton fallback="/" />
        </div>
        <div className="relative flex min-h-[310px] max-w-2xl flex-col justify-end gap-4 p-6 sm:min-h-[370px] sm:p-9">
          <span className="flex h-11 w-11 items-center justify-center rounded-xl border border-white/20 bg-white/10 text-fuchsia-100 backdrop-blur"><Icon name="film" /></span>
          <div>
            <h1 className="text-3xl font-black text-white sm:text-4xl">{title}</h1>
            {franchise.title_ar && franchise.title_en && <p className="mt-1 text-sm font-semibold text-white/60" dir="ltr">{franchise.title_en}</p>}
            <p className="mt-3 max-w-xl text-sm leading-7 text-white/80">{franchise.overview_ar || franchise.overview_en || "أجزاء هذه السلسلة المتوفرة في مكتبتك المحلية."}</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <span className="inline-flex w-fit items-center gap-1.5 rounded-lg border border-white/15 bg-black/25 px-3 py-1.5 text-xs font-bold text-white backdrop-blur"><Icon name="film" className="h-3.5 w-3.5" />{data.total || items.length} متاح محلياً</span>
            <span className="inline-flex w-fit items-center gap-1.5 rounded-lg border border-white/15 bg-black/25 px-3 py-1.5 text-xs font-bold text-white backdrop-blur"><Icon name="list" className="h-3.5 w-3.5" />{franchise.parts_count || parts.length || 1} أجزاء رسمية</span>
          </div>
        </div>
      </section>
      <section className="space-y-3">
        <div>
          <h2 className="text-xl font-black text-[var(--text-primary)]">المتاح في الاستراحة</h2>
          <p className="mt-1 text-xs text-[var(--text-muted)]">أجزاء هذه السلسلة الموجودة على أقراص مكتبتك الآن.</p>
        </div>
        <MediaCollection items={items} onOpen={onOpenMedia} />
      </section>
      {pendingParts.length > 0 && (
        <section className="space-y-3">
          <div>
            <h2 className="text-xl font-black text-[var(--text-primary)]">أجزاء السلسلة الأخرى</h2>
            <p className="mt-1 text-xs text-[var(--text-muted)]">أجزاء رسمية معروفة من السلسلة، لكنها قيد الإضافة إلى الاستراحة.</p>
          </div>
          <div className="flex gap-3 overflow-x-auto pb-3 scrollbar-none snap-x touch-pan-x">
            {pendingParts.map((part) => (
              <article key={part.external_id} className="group relative h-44 w-[min(74vw,280px)] shrink-0 snap-start overflow-hidden rounded-xl border border-dashed border-fuchsia-300/25 bg-[var(--bg-card)]">
                {part.poster_path && <img src={resolveAPIURL(part.poster_path)} alt="" loading="lazy" onError={(event) => { event.currentTarget.style.display = "none"; }} className="absolute inset-0 h-full w-full object-cover opacity-45 transition duration-300 group-hover:scale-105"/>}
                <div className="absolute inset-0 bg-gradient-to-l from-[#120d1d]/95 via-[#120d1d]/70 to-[#120d1d]/25"/>
                <div className="relative flex h-full flex-col justify-between p-4">
                  <div className="flex items-start justify-between gap-2">
                    <span className="inline-flex w-fit items-center gap-1 rounded-md bg-fuchsia-400/15 px-2 py-1 text-[10px] font-bold text-fuchsia-100"><Icon name="download" className="h-3 w-3" />قيد الإضافة</span>
                    {part.rating > 0 && <span className="inline-flex items-center gap-1 rounded-md bg-black/35 px-2 py-1 text-[10px] font-black text-amber-200"><Icon name="star" className="h-3 w-3 fill-current stroke-0" />{Number(part.rating).toFixed(1)}</span>}
                  </div>
                  <div>
                    <h3 className="line-clamp-2 text-sm font-black text-white">{part.title_ar || part.title || part.title_en || "جزء من السلسلة"}</h3>
                    {part.title_ar && part.title_en && <p dir="ltr" className="mt-0.5 truncate text-left text-[10px] font-semibold text-white/60">{part.title_en}</p>}
                    <p className="mt-1 line-clamp-1 text-[11px] leading-5 text-white/70">{part.overview_ar || part.overview || part.overview_en || "جزء رسمي من السلسلة."}</p>
                    <span className="mt-1 block text-xs font-semibold text-white/65">{part.year || "سنة غير متوفرة"}</span>
                  </div>
                </div>
              </article>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}

