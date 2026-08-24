import Icon from "./Icon.jsx";
import { resolveAPIURL } from "../lib/api.js";

const accent = { violet: "from-violet-600/90 to-fuchsia-600/50", amber: "from-amber-600/90 to-orange-600/45", cyan: "from-cyan-600/90 to-blue-600/45", rose: "from-rose-600/90 to-pink-600/45", emerald: "from-emerald-600/90 to-teal-600/45" };

export default function SmartHubCard({ hub, onOpen }) {
  const image = resolveAPIURL(hub.artwork_path) || "/nexora-library-backdrop.PNG";
  const tone = accent[hub.accent] || accent.violet;
  return <button type="button" onClick={() => onOpen?.(hub)} className="group relative min-h-[204px] w-full overflow-hidden rounded-2xl border border-[color:var(--border)] bg-[color:var(--surface)] text-right shadow-lg transition duration-200 hover:-translate-y-1 hover:border-white/30 hover:shadow-2xl focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fuchsia-400" dir="rtl">
    <img src={image} alt="" className="absolute inset-0 h-full w-full object-cover transition duration-300 group-hover:scale-[1.04]" loading="lazy" />
    <div className={`absolute inset-0 bg-gradient-to-t ${tone} via-black/45 to-black/10`} />
    <div className="relative flex min-h-[204px] flex-col justify-end gap-2 p-4 text-white">
      <span className="flex h-9 w-9 items-center justify-center rounded-xl border border-white/25 bg-black/25 backdrop-blur"><Icon name={hub.icon || "spark"} className="h-4 w-4" /></span>
      <div><h3 className="text-lg font-black leading-tight">{hub.title_ar || hub.title_en}</h3>{hub.title_ar && hub.title_en && <p className="mt-0.5 text-[11px] font-semibold text-white/65" dir="ltr">{hub.title_en}</p>}</div>
      <p className="line-clamp-2 text-xs leading-5 text-white/75">{hub.description_ar || hub.description_en || "استكشف مختارات هذا المحور."}</p>
      <div className="mt-1 flex items-center justify-between border-t border-white/15 pt-2 text-xs font-bold"><span>{hub.item_count} عمل</span><span className="inline-flex items-center gap-1">استكشف <Icon name="arrowLeft" className="h-3.5 w-3.5" /></span></div>
    </div>
  </button>;
}
