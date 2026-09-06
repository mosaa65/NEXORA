import { resolveAPIURL } from "../lib/api.js";
import { horizontalWheel } from "../lib/horizontalScroll.js";
import Icon from "./Icon.jsx";

const isArabic = (value) => /[\u0600-\u06FF]/.test(value || "");

function RelatedMediaCard({ item, onOpen }) {
  const title = item.title_ar || item.title_en || item.original_title || "عنوان غير متوفر";
  const englishTitle = item.title_en || item.original_title || "";
  const poster = resolveAPIURL(item.poster_path) || "/nexora-poster-placeholder.PNG";
  const available = Boolean(item.local && item.local_media_id);
  const Tag = available ? "button" : "article";
  return <Tag {...(available ? { type: "button", onClick: () => onOpen?.(item.local_media_id), "aria-label": `فتح ${title}` } : {})} className={`group relative w-[132px] shrink-0 overflow-hidden rounded-2xl border bg-[var(--bg-card)] text-right shadow-[var(--shadow-sm)] transition duration-300 sm:w-[158px] md:w-[174px] ${available ? "cursor-pointer border-emerald-400/20 hover:-translate-y-1 hover:border-emerald-400/65 hover:shadow-[0_18px_38px_rgba(16,185,129,.18)] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400" : "border-dashed border-fuchsia-400/30"}`}>
    <div className="relative aspect-[2/3] overflow-hidden bg-[var(--bg-elevated)]"><img src={poster} alt={title} loading="lazy" onError={(event) => { event.currentTarget.src = "/nexora-poster-placeholder.PNG"; }} className={`h-full w-full object-cover transition duration-700 ${available ? "group-hover:scale-110" : "opacity-70"}`} /><span className="absolute inset-0 bg-gradient-to-t from-[#090712] via-transparent to-black/20" /><span className={`absolute right-2 top-2 inline-flex items-center gap-1 rounded-lg border px-2 py-1 text-[9px] font-black backdrop-blur-md ${available ? "border-emerald-300/30 bg-emerald-950/75 text-emerald-100" : "border-fuchsia-300/25 bg-[#251432]/85 text-fuchsia-100"}`}><Icon name={available ? "play" : "download"} className="h-3 w-3" />{available ? "متاح الآن" : "قيد الإضافة"}</span>{item.rating > 0 && <span className="absolute bottom-2 left-2 inline-flex items-center gap-1 rounded-md border border-amber-300/25 bg-black/60 px-1.5 py-1 text-[9px] font-black text-amber-200 backdrop-blur"><Icon name="star" className="h-3 w-3 fill-current stroke-0" />{Number(item.rating).toFixed(1)}</span>}</div>
    <div className="min-h-[92px] space-y-1.5 p-2.5 sm:p-3"><h3 dir={isArabic(title) ? "rtl" : "ltr"} className={`line-clamp-2 text-[12px] font-black leading-5 text-[var(--text-primary)] sm:text-[13px] ${isArabic(title) ? "text-right" : "text-left"}`}>{title}</h3>{englishTitle && englishTitle !== title && <p dir="ltr" className="truncate text-left text-[10px] font-semibold text-[var(--text-muted)]">{englishTitle}</p>}<div className="flex items-center justify-between gap-1 text-[10px] font-bold text-[var(--text-muted)]"><span>{item.release_year || "—"}</span><span>{item.kind === "tv" ? "مسلسل" : "فيلم"}</span></div></div>
  </Tag>;
}

export default function RelatedMediaRail({ items = [], onOpen }) {
  if (!items.length) return null;
  const availableCount = items.filter((item) => item.local).length;
  return <section className="overflow-hidden rounded-3xl border border-[var(--border-default)] bg-[var(--bg-card)] p-4 shadow-[var(--shadow-sm)] sm:p-6" dir="rtl" aria-label="أعمال ذات صلة"><div className="mb-4 flex items-start justify-between gap-3 border-b border-[var(--border-subtle)] pb-3"><div><h2 className="flex items-center gap-2 text-base font-black text-[var(--text-primary)] sm:text-lg"><span className="flex h-7 w-7 items-center justify-center rounded-xl bg-fuchsia-500/15 text-fuchsia-300"><Icon name="spark" className="h-4 w-4" /></span>قد يعجبك أيضًا</h2><p className="mt-1 text-[11px] font-medium text-[var(--text-muted)] sm:text-xs">اقتراحات موثقة من TMDB ومطابقة بمكتبتك عبر المعرّف الرسمي.</p></div><span className="shrink-0 rounded-full border border-emerald-400/20 bg-emerald-500/10 px-2.5 py-1 text-[10px] font-black text-emerald-300">{availableCount} متاح</span></div><div onWheel={horizontalWheel} className="flex gap-3 overflow-x-auto pb-3 pt-1 scrollbar-none snap-x touch-pan-x sm:gap-4">{items.map((item) => <div key={`${item.provider}-${item.kind}-${item.external_id}`} className="snap-start"><RelatedMediaCard item={item} onOpen={onOpen} /></div>)}</div></section>;
}
