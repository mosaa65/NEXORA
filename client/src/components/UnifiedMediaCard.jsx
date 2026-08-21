import React from "react";
import { motion } from "framer-motion";
import Icon from "./Icon";
import { resolveAPIURL } from "../lib/api";

export default function UnifiedMediaCard({ media, onOpen, onQuickPlay }) {
  if (!media) return null;

  const posterURL = resolveAPIURL(media.posterPath) || "/images/jujutsu_kaisen_poster.png";

  return (
    <motion.div
      whileHover={{ scale: 1.04, y: -6 }}
      transition={{ duration: 0.22 }}
      className="group relative flex flex-col overflow-hidden rounded-2xl border border-white/10 bg-[#0C0B17] shadow-xl hover:shadow-2xl hover:shadow-purple-900/40 hover:border-fuchsia-500/50 transition-all cursor-pointer text-right"
      onClick={() => onOpen && onOpen(media)}
      dir="rtl"
    >
      {/* Poster Image Container */}
      <div className="relative aspect-[2/3] w-full overflow-hidden bg-slate-900">
        <img
          src={posterURL}
          alt={media.titleAr || media.titleEn}
          loading="lazy"
          className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-108"
        />

        {/* Ambient Dark Bottom Fade */}
        <div className="absolute inset-0 bg-gradient-to-t from-[#0C0B17] via-transparent to-transparent opacity-80 group-hover:opacity-60 transition-opacity" />

        {/* Top Badges (Resolution & Type) */}
        <div className="absolute top-2.5 right-2.5 flex items-center gap-1.5 flex-wrap z-10">
          {media.resolution && (
            <span className="px-2 py-0.5 rounded-md text-[10px] font-mono font-black bg-black/70 text-cyan-300 backdrop-blur-md border border-cyan-500/30 shadow">
              {media.resolution}
            </span>
          )}
          {media.fileCount > 1 && (
            <span className="px-2 py-0.5 rounded-md text-[10px] font-bold bg-purple-950/80 text-purple-300 backdrop-blur-md border border-purple-500/30">
              {media.fileCount} حلقات
            </span>
          )}
        </div>

        {/* Rating Badge (Top Left) */}
        {media.rating > 0 && (
          <div className="absolute top-2.5 left-2.5 px-2 py-0.5 rounded-md text-[10px] font-bold bg-black/70 text-amber-300 backdrop-blur-md border border-amber-500/30 shadow flex items-center gap-0.5 z-10">
            <span>⭐</span>
            <span>{media.rating.toFixed(1)}</span>
          </div>
        )}

        {/* Quick Play Floating Button (On Hover) */}
        <div className="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity duration-300 bg-black/30 backdrop-blur-[2px]">
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onQuickPlay && onQuickPlay(media);
            }}
            className="flex h-12 w-12 items-center justify-center rounded-full bg-gradient-to-tr from-fuchsia-600 to-purple-600 text-white shadow-xl shadow-purple-900/80 hover:scale-110 transition-transform"
            title="تشغيل فوري"
          >
            <Icon name="play" className="h-5 w-5 fill-current" />
          </button>
        </div>
      </div>

      {/* Media Details Footer */}
      <div className="p-3 flex flex-col justify-between flex-1 space-y-1 bg-[#0C0B17]">
        <h4 className="text-xs sm:text-sm font-bold text-white group-hover:text-fuchsia-300 transition-colors line-clamp-1">
          {media.titleAr || media.titleEn}
        </h4>

        <div className="flex items-center justify-between text-[11px] text-gray-400 font-medium pt-0.5">
          <span>{media.year || (media.releaseYear ? media.releaseYear : "—")}</span>
          <span className="text-[10px] text-purple-300 bg-purple-500/10 px-1.5 py-0.2 rounded border border-purple-500/20">
            {media.type === "movie" ? "فيلم" : media.type === "series" ? "مسلسل" : "أنمي"}
          </span>
        </div>
      </div>
    </motion.div>
  );
}
