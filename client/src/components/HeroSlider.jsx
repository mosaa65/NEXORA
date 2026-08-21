import React, { useState, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import Icon from "./Icon";
import { resolveAPIURL } from "../lib/api";

export default function HeroSlider({ items = [], onOpenMedia, onQuickPlay }) {
  const [currentIndex, setCurrentIndex] = useState(0);

  useEffect(() => {
    if (!items || items.length <= 1) return;
    const interval = setInterval(() => {
      setCurrentIndex((prev) => (prev + 1) % items.length);
    }, 8000);
    return () => clearInterval(interval);
  }, [items]);

  if (!items || items.length === 0) return null;

  const current = items[currentIndex] || items[0];
  const backdropURL = resolveAPIURL(current.bannerPath || current.posterPath) || "/images/aot_banner_detail.png";

  return (
    <div className="relative w-full h-[400px] sm:h-[480px] lg:h-[540px] rounded-[2.5rem] overflow-hidden border border-white/10 shadow-2xl bg-[#0B0A16] group" dir="rtl">
      {/* Background Media Backdrop with Ken-Burns Subtle Scale */}
      <AnimatePresence mode="wait">
        <motion.div
          key={current.id || currentIndex}
          initial={{ opacity: 0, scale: 1.05 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 1 }}
          className="absolute inset-0 bg-cover bg-center"
          style={{ backgroundImage: `url('${backdropURL}')` }}
        >
          {/* Dual Gradient Overlay: Left-To-Right & Bottom-Up */}
          <div className="absolute inset-0 bg-gradient-to-r from-[#08070E] via-[#08070E]/80 to-transparent" />
          <div className="absolute inset-0 bg-gradient-to-t from-[#08070E] via-transparent to-black/30" />
        </motion.div>
      </AnimatePresence>

      {/* Hero Content Information */}
      <div className="relative z-10 h-full flex flex-col justify-end p-6 sm:p-10 lg:p-14 max-w-2xl space-y-4">
        {/* Badges Bar */}
        <div className="flex items-center gap-2.5 flex-wrap">
          <span className="px-3 py-1 rounded-full text-xs font-black bg-gradient-to-r from-fuchsia-600 to-purple-600 text-white shadow-lg shadow-purple-900/50">
            ★ متصدر المشاهدات
          </span>
          {current.resolution && (
            <span className="px-2.5 py-0.5 rounded-lg text-xs font-mono font-bold bg-cyan-500/20 text-cyan-300 border border-cyan-500/30">
              {current.resolution}
            </span>
          )}
          {current.year && (
            <span className="px-2.5 py-0.5 rounded-lg text-xs font-bold bg-white/10 text-white/80 backdrop-blur-md">
              {current.year}
            </span>
          )}
          {current.rating > 0 && (
            <span className="px-2.5 py-0.5 rounded-lg text-xs font-bold bg-amber-500/20 text-amber-300 border border-amber-500/30 flex items-center gap-1">
              ⭐ {current.rating}
            </span>
          )}
        </div>

        {/* Title */}
        <h1 className="text-2xl sm:text-4xl lg:text-5xl font-black text-white leading-tight drop-shadow-md">
          {current.titleAr || current.titleEn}
        </h1>

        {/* Plot Synopsis */}
        <p className="text-xs sm:text-sm text-gray-300 line-clamp-3 leading-relaxed drop-shadow">
          {current.plot || "استمتع بمشاهدة هذا العمل الفني الرائع بجودة فائقة وصوت محيطي عبر شبكة الاستراحة المحلية."}
        </p>

        {/* Action Buttons */}
        <div className="flex items-center gap-3 pt-2">
          <button
            onClick={() => onQuickPlay && onQuickPlay(current)}
            className="px-6 py-3 rounded-2xl bg-gradient-to-r from-fuchsia-600 via-purple-600 to-indigo-600 hover:from-fuchsia-500 hover:to-indigo-500 text-white font-bold text-sm shadow-xl shadow-purple-900/40 hover:scale-105 transition-all flex items-center gap-2"
          >
            <Icon name="play" className="w-4 h-4 fill-current" />
            <span>تشغيل فوري (Play)</span>
          </button>

          <button
            onClick={() => onOpenMedia && onOpenMedia(current)}
            className="px-5 py-3 rounded-2xl bg-white/10 hover:bg-white/20 text-white font-bold text-sm backdrop-blur-md border border-white/15 hover:border-white/30 transition-all flex items-center gap-2"
          >
            <Icon name="spark" className="w-4 h-4 text-fuchsia-300" />
            <span>تفاصيل العمل</span>
          </button>
        </div>
      </div>

      {/* Slide Indicators Navigation (Bottom Left) */}
      {items.length > 1 && (
        <div className="absolute bottom-6 left-6 z-20 flex items-center gap-1.5 bg-black/40 backdrop-blur-md px-3 py-1.5 rounded-full border border-white/10">
          {items.map((_, idx) => (
            <button
              key={idx}
              onClick={() => setCurrentIndex(idx)}
              className={`h-2 rounded-full transition-all duration-300 ${
                idx === currentIndex ? "w-6 bg-gradient-to-r from-fuchsia-500 to-purple-500" : "w-2 bg-white/30 hover:bg-white/60"
              }`}
            />
          ))}
        </div>
      )}
    </div>
  );
}
