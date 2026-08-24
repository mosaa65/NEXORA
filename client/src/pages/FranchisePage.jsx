import { useEffect, useState } from "react";
import Icon from "../components/Icon.jsx";
import MediaCollection from "../components/MediaCollection.jsx";
import { getFranchiseMedia, resolveAPIURL } from "../lib/api.js";

export default function FranchisePage({ slug, onOpenMedia }) {
  const [data, setData] = useState(null);
  const [error, setError] = useState("");
  useEffect(() => { let alive = true; setData(null); setError(""); getFranchiseMedia(slug, { limit: 100 }).then((payload) => { if (alive) setData(payload); }).catch((err) => { if (alive) setError(err.message || "تعذر تحميل السلسلة"); }); return () => { alive = false; }; }, [slug]);
  if (error) return <div className="rounded-xl border border-rose-400/30 bg-rose-950/15 p-7 text-center text-sm font-bold text-rose-200" dir="rtl">{error}</div>;
  if (!data) return <div className="h-72 animate-pulse rounded-xl border border-white/10 bg-white/[0.03]" aria-label="جارٍ تحميل السلسلة" />;
  const franchise = data.franchise || {};
  const title = franchise.title_ar || franchise.title_en || "سلسلة أفلام";
  const artwork = resolveAPIURL(franchise.backdrop_path || franchise.poster_path) || "/nexora-library-backdrop.PNG";
  const items = (data.items || []).map((item) => ({ id:item.id, titleAr:item.title_ar, titleEn:item.title_en, type:item.type, year:item.release_year, rating:item.rating, posterPath:item.poster_path, status:item.status, seasonCount:item.season_count, tmdbSeasonCount:item.tmdb_season_count, totalSize:item.total_size, bestResolution:item.best_resolution, runtimeMinutes:item.runtime_minutes, hasArabicAudio:item.has_arabic_audio, hasArabicSubtitles:item.has_arabic_subtitles }));
  return <div className="space-y-7 pb-16" dir="rtl"><section className="relative min-h-[310px] overflow-hidden rounded-xl border border-white/10 bg-[#12101d] sm:min-h-[370px]"><img src={artwork} alt="" className="absolute inset-0 h-full w-full object-cover" onError={(event) => { event.currentTarget.src = "/nexora-library-backdrop.PNG"; }}/><div className="absolute inset-0 bg-gradient-to-l from-black/95 via-black/65 to-black/20"/><div className="relative flex min-h-[310px] max-w-2xl flex-col justify-end gap-4 p-6 sm:min-h-[370px] sm:p-9"><span className="flex h-11 w-11 items-center justify-center rounded-xl border border-white/20 bg-white/10 text-fuchsia-100 backdrop-blur"><Icon name="film" /></span><div><h1 className="text-3xl font-black text-white sm:text-4xl">{title}</h1>{franchise.title_ar && franchise.title_en && <p className="mt-1 text-sm font-semibold text-white/60" dir="ltr">{franchise.title_en}</p>}<p className="mt-3 max-w-xl text-sm leading-7 text-white/80">{franchise.overview_ar || franchise.overview_en || "أجزاء هذه السلسلة المتوفرة في مكتبتك المحلية."}</p></div><span className="inline-flex w-fit items-center gap-1.5 rounded-lg border border-white/15 bg-black/25 px-3 py-1.5 text-xs font-bold text-white backdrop-blur"><Icon name="film" className="h-3.5 w-3.5" />{data.total || items.length} أفلام متاحة محلياً</span></div></section><MediaCollection items={items} onOpen={onOpenMedia} /></div>;
}
