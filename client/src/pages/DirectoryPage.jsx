import { useEffect, useState } from "react";
import Icon from "../components/Icon.jsx";
import { getFranchises, getPeople, getSmartHubs, resolveAPIURL } from "../lib/api.js";

const directoryConfig = {
  hubs: { title: "كل المحاور الذكية", subtitle: "استكشف جميع المحاور المحفوظة في مكتبتك.", icon: "grid", tone: "fuchsia" },
  people: { title: "كل الشخصيات", subtitle: "الممثلون وصنّاع الأفلام المرتبطون بأعمال مكتبتك.", icon: "user", tone: "cyan" },
  franchises: { title: "كل سلاسل الأفلام", subtitle: "السلاسل المرتبطة بالأعمال الموجودة في مكتبتك.", icon: "film", tone: "amber" },
};

function DirectoryCard({ kind, item, onOpen }) {
  const isPerson = kind === "people";
  const [imgError, setImgError] = useState(false);

  if (kind === "hubs") {
    const image = resolveAPIURL(item.artwork_path) || "/nexora-library-backdrop.PNG";
    return <button type="button" onClick={() => onOpen(item)} className="luminous-card group relative min-h-56 overflow-hidden rounded-2xl bg-[#0E0C1A] text-right" dir="rtl"><img src={image} alt="" className="absolute inset-0 h-full w-full object-cover transition duration-500 group-hover:scale-105" onError={(event) => { event.currentTarget.src = "/nexora-library-backdrop.PNG"; }} /><span className="absolute inset-0 bg-gradient-to-t from-black/90 via-black/45 to-transparent" /><span className="relative flex min-h-56 flex-col justify-end gap-2 p-5 text-white"><Icon name={item.icon || "spark"} className="h-7 w-7 text-fuchsia-300" /><strong className="text-lg font-black">{item.title_ar || item.title_en}</strong><small className="text-xs text-white/65">{item.item_count || 0} عمل</small></span></button>;
  }

  const rawImage = isPerson ? item.profile_path : (item.backdrop_path || item.poster_path);
  const image = resolveAPIURL(rawImage);
  return (
    <button
      type="button"
      onClick={() => onOpen(item)}
      className={`group overflow-hidden rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] text-right shadow-[var(--shadow-sm)] transition duration-300 hover:-translate-y-1 hover:border-[var(--color-info)] hover:shadow-[var(--shadow-lg)] ${isPerson ? "" : "min-h-48"}`}
      dir="rtl"
    >
      <div className={`relative overflow-hidden ${isPerson ? "aspect-[4/5] bg-gradient-to-br from-cyan-950/40 via-purple-950/20 to-fuchsia-950/30" : "h-32 bg-black/20"}`}>
        {!imgError && image ? (
          <img
            src={image}
            alt=""
            className="h-full w-full object-cover transition duration-500 group-hover:scale-105"
            onError={() => setImgError(true)}
          />
        ) : (
          <div className="absolute inset-0 flex items-center justify-center p-3 text-center">
            <span className="flex h-14 w-14 sm:h-16 sm:w-16 items-center justify-center rounded-2xl border border-[var(--border-default)] bg-[var(--bg-elevated)] text-[var(--color-info)] shadow-inner">
              <Icon name={isPerson ? "user" : "film"} className="h-7 w-7 sm:h-8 sm:w-8" />
            </span>
          </div>
        )}
      </div>
      <span className="block space-y-1 p-3.5 sm:p-4">
        <strong className="block truncate text-xs sm:text-sm font-black text-[var(--text-primary)]">
          {isPerson ? (item.name_ar || item.name_en) : (item.title_ar || item.title_en)}
        </strong>
        {isPerson && item.name_ar && item.name_en && (
          <small className="block truncate text-left text-[10px] sm:text-[11px] font-semibold text-[var(--text-muted)]" dir="ltr">
            {item.name_en}
          </small>
        )}
        <small className="inline-flex items-center gap-1 rounded-md bg-[var(--color-info-light)] px-1.5 py-0.5 text-[10px] font-bold text-[var(--color-info)]">
          <Icon name="film" className="h-3 w-3" />
          {isPerson ? `${item.local_media_count || 0} أعمال محلية` : `${item.local_item_count || 0} أفلام متاحة`}
        </small>
      </span>
    </button>
  );
}

export default function DirectoryPage({ kind, onOpen }) {
  const [items, setItems] = useState(null);
  const config = directoryConfig[kind] || directoryConfig.hubs;

  useEffect(() => {
    const load = kind === "people" ? getPeople(100) : kind === "franchises" ? getFranchises(100) : getSmartHubs();
    load.then((data) => setItems(data?.[kind === "people" ? "people" : kind === "franchises" ? "franchises" : "hubs"] || [])).catch(() => setItems([]));
  }, [kind]);

  if (!items) return <div className="min-h-72 animate-pulse rounded-2xl bg-[var(--bg-surface)]" />;
  return <div className="space-y-7 pb-16" dir="rtl"><header className="flex flex-wrap items-end justify-between gap-4 border-b border-[var(--border-subtle)] pb-5"><div><p className="text-xs font-bold text-[var(--color-accent)]">دليل المكتبة</p><h1 className="mt-1 flex items-center gap-2 text-2xl font-black text-[var(--text-primary)]"><Icon name={config.icon} className="h-6 w-6" />{config.title}</h1><p className="mt-2 text-sm text-[var(--text-muted)]">{config.subtitle}</p></div><button type="button" onClick={() => window.history.back()} className="inline-flex items-center gap-2 rounded-xl border border-[var(--border-default)] bg-[var(--bg-surface)] px-4 py-2.5 text-xs font-bold text-[var(--text-primary)] transition hover:bg-[var(--bg-elevated)]"><Icon name="arrowRight" className="h-4 w-4" />العودة</button></header>{items.length ? <div className={`grid grid-cols-2 gap-3 sm:gap-4 ${kind === "people" ? "sm:grid-cols-3 lg:grid-cols-5" : "sm:grid-cols-2 lg:grid-cols-3"}`}>{items.map((item) => <DirectoryCard key={item.slug || item.id} kind={kind} item={item} onOpen={onOpen} />)}</div> : <div className="rounded-2xl border border-dashed border-[var(--border-default)] p-16 text-center text-sm text-[var(--text-muted)]">لا توجد بيانات متاحة حاليًا.</div>}</div>;
}
