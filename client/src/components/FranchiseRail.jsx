import { useEffect, useState } from "react";
import Icon from "./Icon.jsx";
import { getFranchises, resolveAPIURL } from "../lib/api.js";

function FranchiseCard({ item, onOpen }) {
  const title = item.title_ar || item.title_en;
  const artwork = resolveAPIURL(item.backdrop_path || item.poster_path) || "/nexora-library-backdrop.PNG";
  return <button type="button" onClick={() => onOpen?.(item)} className="group relative h-48 w-[280px] shrink-0 overflow-hidden rounded-xl border border-[var(--border-default)] bg-[var(--bg-card)] text-right shadow-[var(--shadow-sm)] transition duration-300 hover:-translate-y-1 hover:border-fuchsia-400/55 hover:shadow-[var(--shadow-lg)] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--color-accent)] sm:h-56 sm:w-[320px]" dir="rtl" aria-label={`فتح سلسلة ${title}`}>
    <img src={artwork} alt="" loading="lazy" onError={(event) => { event.currentTarget.src = "/nexora-library-backdrop.PNG"; }} className="absolute inset-0 h-full w-full object-cover transition duration-700 group-hover:scale-105" />
    <span className="absolute inset-0 bg-gradient-to-l from-[#0b0915]/95 via-[#0b0915]/55 to-[#0b0915]/10" />
    <span className="absolute inset-0 bg-gradient-to-t from-[#0b0915]/90 via-transparent" />
    <span className="relative flex h-full flex-col justify-end gap-2 p-4">
      <span className="flex h-9 w-9 items-center justify-center rounded-lg border border-white/15 bg-black/35 text-fuchsia-100 backdrop-blur"><Icon name="film" className="h-4 w-4" /></span>
      <span><strong className="block line-clamp-1 text-base font-black text-white">{title}</strong>{item.title_ar && item.title_en && <small dir="ltr" className="mt-0.5 block truncate text-left text-[11px] font-semibold text-white/60">{item.title_en}</small>}</span>
      <span className="inline-flex w-fit items-center gap-1.5 rounded-md border border-white/10 bg-black/30 px-2 py-1 text-[11px] font-bold text-white/85 backdrop-blur"><Icon name="film" className="h-3.5 w-3.5 text-fuchsia-200" />{item.local_item_count} أفلام متاحة</span>
    </span>
  </button>;
}

export default function FranchiseRail({ onOpen }) {
  const [items, setItems] = useState([]);
  useEffect(() => { let alive = true; getFranchises().then((data) => { if (alive) setItems(data.franchises || []); }).catch(() => { if (alive) setItems([]); }); return () => { alive = false; }; }, []);
  if (!items.length) return null;
  return <section className="space-y-4" dir="rtl" aria-label="سلاسل الأفلام"><div><p className="text-xs font-bold text-fuchsia-300">مكتبتك المترابطة</p><h2 className="mt-1 flex items-center gap-2 text-xl font-black text-[var(--text-primary)]"><Icon name="film" className="h-5 w-5 text-fuchsia-300" />سلاسل الأفلام</h2><p className="mt-1 text-xs text-[var(--text-muted)]">تجميع تلقائي محفوظ محلياً من بيانات الأفلام المتاحة لديك.</p></div><div className="flex gap-4 overflow-x-auto px-1 pb-5 pt-3 scrollbar-none snap-x">{items.map((item) => <div key={item.slug} className="snap-start"><FranchiseCard item={item} onOpen={onOpen} /></div>)}</div></section>;
}
