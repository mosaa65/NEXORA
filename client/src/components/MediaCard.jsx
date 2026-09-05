import { motion } from "framer-motion";
import { resolveAPIURL } from "../lib/api.js";

const typeLabels = { movie: "فيلم", series: "مسلسل", anime: "أنمي", documentary: "وثائقي", play: "مسرحية" };

export default function MediaCard({ item, onOpen, index = 0, compact = false }) {
  const imageSrc = item.posterPath ? resolveAPIURL(item.posterPath) : "";
  const displayTitle = /[\u0600-\u06FF]/.test(item.titleAr || "") ? item.titleAr : (item.titleEn || item.titleAr);
  const tags = (item.genres || item.highlights || []).slice(0, 2);

  return (
    <motion.button
      type="button"
      onClick={() => onOpen(item)}
      initial={{ opacity: 0, y: 16 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: Math.min(index, 10) * 0.035, duration: 0.3 }}
      className={`group relative flex flex-col overflow-hidden rounded-2xl border border-white/10 bg-[#0d0c18] text-right shadow-[0_12px_28px_rgba(0,0,0,.28)] transition-all duration-300 hover:-translate-y-1 hover:border-fuchsia-400/65 hover:shadow-[0_20px_38px_rgba(112,26,117,.30)] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fuchsia-400 ${compact ? "w-32 sm:w-36" : "w-full"}`}
    >
      <div className={`relative w-full overflow-hidden bg-[radial-gradient(circle_at_70%_15%,rgba(217,70,239,.5),transparent_32%),linear-gradient(145deg,#1d1741,#100f1c_60%,#082034)] ${compact ? "aspect-[3/4]" : "aspect-[2/3]"}`}>
        <img src={imageSrc || "/nexora-poster-placeholder.PNG"} alt={displayTitle} onError={(event) => { event.currentTarget.src = "/nexora-poster-placeholder.PNG"; }} className="h-full w-full object-cover transition duration-700 group-hover:scale-110" />
        <div className="absolute inset-0 bg-gradient-to-t from-[#08070e] via-[#08070e]/10 to-transparent" />
        <div className="absolute right-2.5 top-2.5 rounded-lg border border-white/15 bg-black/45 px-2 py-1 text-[10px] font-black text-white/90 backdrop-blur-md">{typeLabels[item.type] || "مكتبة"}</div>
        <div className="absolute bottom-2.5 left-2.5 flex items-center gap-1 rounded-lg border border-yellow-300/20 bg-black/55 px-2 py-1 text-[10px] font-black text-yellow-300 backdrop-blur-md"><span>★</span>{item.rating?.toFixed?.(1) || "—"}</div>
        {item.fileCount > 1 && <div className="absolute bottom-2.5 right-2.5 rounded-lg bg-fuchsia-500/20 px-2 py-1 text-[10px] font-bold text-fuchsia-100 backdrop-blur-md">{item.fileCount} حلقة</div>}
      </div>

      <div className="min-h-[82px] p-3">
        <h3 className="line-clamp-2 text-sm font-black leading-5 text-white transition-colors group-hover:text-fuchsia-300">{displayTitle}</h3>
        <div className="mt-2 flex items-center justify-between gap-2"><span className="text-[10px] font-bold text-white/40">{item.year || "—"}</span><div className="flex min-w-0 gap-1">{tags.map((tag) => <span key={tag} className="max-w-[62px] truncate rounded-md bg-white/[.06] px-1.5 py-0.5 text-[9px] font-bold text-white/55">{tag}</span>)}</div></div>
      </div>
    </motion.button>
  );
}
