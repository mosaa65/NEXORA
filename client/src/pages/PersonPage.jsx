import { useEffect, useState } from "react";
import Icon from "../components/Icon.jsx";
import MediaCollection from "../components/MediaCollection.jsx";
import { getPersonMedia, resolveAPIURL } from "../lib/api.js";

export default function PersonPage({ slug, onOpenMedia }) {
  const [data, setData] = useState(null);
  const [error, setError] = useState("");
  useEffect(() => { let alive = true; setData(null); setError(""); getPersonMedia(slug, { limit: 100 }).then((payload) => { if (alive) setData(payload); }).catch((err) => { if (alive) setError(err.message || "تعذر تحميل أعمال الشخص"); }); return () => { alive = false; }; }, [slug]);
  if (error) return <div className="rounded-xl border border-rose-400/30 bg-rose-950/15 p-7 text-center text-sm font-bold text-rose-200" dir="rtl">{error}</div>;
  if (!data) return <div className="h-72 animate-pulse rounded-xl border border-white/10 bg-white/[0.03]" aria-label="جارٍ تحميل الأعمال" />;
  const person = data.person || {};
  const name = person.name_ar || person.name_en || "شخص من المكتبة";
  const image = resolveAPIURL(person.profile_path);
  const items = (data.items || []).map((item) => ({ id:item.id, titleAr:item.title_ar, titleEn:item.title_en, type:item.type, year:item.release_year, rating:item.rating, posterPath:item.poster_path, status:item.status, seasonCount:item.season_count, tmdbSeasonCount:item.tmdb_season_count, totalSize:item.total_size, bestResolution:item.best_resolution, runtimeMinutes:item.runtime_minutes, hasArabicAudio:item.has_arabic_audio, hasArabicSubtitles:item.has_arabic_subtitles }));
  return <div className="space-y-7 pb-16" dir="rtl"><section className="relative overflow-hidden rounded-xl border border-white/10 bg-gradient-to-l from-cyan-950/40 via-[#12101d] to-fuchsia-950/30 p-6 sm:p-9"><div className="flex flex-col items-center gap-6 text-center sm:flex-row sm:text-right">{image ? <img src={image} alt="" onError={(event) => { event.currentTarget.style.display = "none"; }} className="h-32 w-28 rounded-xl border border-white/15 object-cover shadow-xl sm:h-40 sm:w-32" /> : <span className="flex h-32 w-28 items-center justify-center rounded-xl border border-white/10 bg-white/5 text-cyan-100 sm:h-40 sm:w-32"><Icon name="user" className="h-12 w-12" /></span>}<div className="space-y-3"><span className="inline-flex items-center gap-1.5 rounded-md border border-cyan-300/20 bg-cyan-300/10 px-2.5 py-1 text-xs font-bold text-cyan-100"><Icon name="user" className="h-3.5 w-3.5" />شخصية مرتبطة محلياً</span><h1 className="text-3xl font-black text-white sm:text-4xl">{name}</h1>{person.name_ar && person.name_en && <p className="text-sm font-semibold text-white/60" dir="ltr">{person.name_en}</p>}<p className="text-sm text-white/70">{person.known_for_department || "أعمال مرتبطة داخل مكتبتك"}</p><span className="inline-flex items-center gap-1.5 text-xs font-bold text-cyan-100"><Icon name="film" className="h-4 w-4" />{data.total || items.length} أعمال متاحة محلياً</span></div></div></section><MediaCollection items={items} onOpen={onOpenMedia} /></div>;
}
