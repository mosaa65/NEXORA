import React from "react";
import { motion } from "framer-motion";
import Icon from "./Icon";

export default function HubBannerCard({
  title,
  subtitle,
  count,
  backdrop,
  accentColor = "from-purple-600/80 to-fuchsia-900/80",
  borderColor = "border-fuchsia-500/30",
  onClick,
  tag = "مجموعة مميزة",
}) {
  return (
    <motion.button
      type="button"
      whileHover={{ scale: 1.02, y: -4 }}
      transition={{ duration: 0.25 }}
      onClick={onClick}
      className="luminous-card relative h-44 w-full overflow-hidden rounded-2xl bg-[#0F0E1A] text-right group sm:h-52"
      dir="rtl"
    >
      {/* Background Image / Collage */}
      <div
        className="absolute inset-0 bg-cover bg-center transition-transform duration-700 group-hover:scale-108 opacity-70 group-hover:opacity-85"
        style={{ backgroundImage: `url('${backdrop}')` }}
      />

      {/* Dark Ambient Overlay Gradient */}
      <div className={`absolute inset-0 bg-gradient-to-t ${accentColor} via-black/40 to-transparent`} />
      <div className="absolute inset-0 bg-gradient-to-r from-black/80 via-black/40 to-transparent" />

      {/* Card Content Information */}
      <div className="relative z-10 h-full p-5 flex flex-col justify-between text-right">
        {/* Top Tag & Count */}
        <div className="flex items-center justify-between">
          <span className="text-[11px] font-bold px-2.5 py-0.5 rounded-full bg-white/10 text-white/90 backdrop-blur-md border border-white/15">
            {tag}
          </span>
          {count !== undefined && (
            <span className="text-xs font-mono font-bold text-fuchsia-300 bg-fuchsia-950/60 px-2.5 py-0.5 rounded-full border border-fuchsia-500/30">
              {count} عمل
            </span>
          )}
        </div>

        {/* Bottom Title & Action Trigger */}
        <div className="space-y-1">
          <h3 className="text-lg sm:text-xl font-black text-white group-hover:text-fuchsia-200 transition-colors drop-shadow-md">
            {title}
          </h3>
          <p className="text-xs text-gray-300 line-clamp-1">{subtitle}</p>

          <div className="pt-2 flex items-center gap-1.5 text-xs font-bold text-fuchsia-300 group-hover:translate-x-1 transition-transform">
            <span>تصفح المجموعة بالكامل</span>
            <span>‹</span>
          </div>
        </div>
      </div>
    </motion.button>
  );
}
