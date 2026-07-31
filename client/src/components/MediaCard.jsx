import { motion } from "framer-motion";
import Icon from "./Icon.jsx";
import { getMediaTypeLabel } from "../data/library.js";

export default function MediaCard({ item, onOpen, compact = false, index = 0 }) {
  return (
    <motion.button
      type="button"
      onClick={() => onOpen(item)}
      initial={{ opacity: 0, y: 18 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: index * 0.04, duration: 0.35 }}
      className={`group relative overflow-hidden rounded-[2rem] border border-white/10 bg-white/[0.04] text-left shadow-panel transition hover:-translate-y-1 hover:border-electric/30 hover:shadow-neon ${
        compact ? "min-h-[24rem]" : "min-h-[28rem]"
      }`}
    >
      <div
        className="absolute inset-0 opacity-90 transition duration-700 group-hover:scale-[1.03] group-hover:opacity-100"
        style={{ backgroundImage: item.gradient }}
      />
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(255,255,255,0.18),transparent_28%),radial-gradient(circle_at_bottom_right,rgba(25,183,255,0.18),transparent_35%),linear-gradient(180deg,rgba(4,7,18,0.02),rgba(4,7,18,0.84))]" />
      <div className="absolute inset-0 ring-1 ring-inset ring-white/10" />

      <div className="relative flex h-full flex-col justify-between p-4 sm:p-5">
        <div className="flex items-start justify-between gap-3">
          <div className="rounded-full border border-white/12 bg-black/28 px-3 py-2 text-[11px] font-semibold text-white/70">
            {getMediaTypeLabel(item.type)}
          </div>
          <div className="rounded-full border border-white/12 bg-black/28 px-3 py-2 text-xs text-white/75">
            {item.resolution}
          </div>
        </div>

        <div className="flex flex-1 flex-col justify-end gap-4">
          <div className="min-h-[8rem] rounded-[1.5rem] border border-white/8 bg-[linear-gradient(180deg,transparent,rgba(0,0,0,0.12))] p-4 backdrop-blur-sm">
            <p className="text-xs font-semibold text-white/45">{item.titleEn}</p>
            <h3 className="mt-2 text-2xl font-black leading-tight text-white">{item.titleAr}</h3>
            <p className="mt-2 line-clamp-3 text-sm leading-6 text-white/72">{item.plot}</p>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <div className="rounded-2xl border border-white/12 bg-black/28 px-3 py-3">
              <div className="flex items-center gap-2">
                <Icon name="spark" className="h-4 w-4 text-electric" />
                <span className="text-xs font-semibold text-white/45">الموسم / المدة</span>
              </div>
              <p className="mt-2 text-sm font-semibold text-white">{item.seasonLabel || item.duration}</p>
            </div>
            <div className="rounded-2xl border border-white/12 bg-black/28 px-3 py-3">
              <div className="flex items-center gap-2">
                <Icon name="play" className="h-4 w-4 text-fuchsia-300" />
                <span className="text-xs font-semibold text-white/45">الحلقات / الملفات</span>
              </div>
              <p className="mt-2 text-sm font-semibold text-white">{item.episodeLabel || `${item.fileCount} ملف`}</p>
            </div>
          </div>

          <div className="flex items-end justify-between gap-4">
            <div className="space-y-1 text-sm text-white/70">
              <div className="flex items-center gap-2">
                <Icon name="spark" className="h-4 w-4 text-electric" />
                <span>{item.highlights?.[0] || "متميز"}</span>
              </div>
              <div className="flex items-center gap-2">
                <Icon name="bell" className="h-4 w-4 text-fuchsia-300" />
                <span>{item.highlights?.[1] || "محتوى مختار"}</span>
              </div>
            </div>
            <div className="rounded-2xl border border-white/12 bg-black/30 px-3 py-2 text-right">
              <p className="text-xs font-semibold text-white/40">التقييم</p>
              <p className="mt-1 text-lg font-bold text-white">{item.rating?.toFixed?.(1) || "N/A"}</p>
            </div>
          </div>
        </div>
      </div>
    </motion.button>
  );
}
