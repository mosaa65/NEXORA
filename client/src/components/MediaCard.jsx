import { motion } from "framer-motion";

export default function MediaCard({ item, onOpen, index = 0, compact = false }) {
  const imageSrc = item.posterPath || (
    item.id === 1 ? "/images/tokyo_ghoul_hero.png" :
    item.id === 2 ? "/images/attack_on_titan_poster.png" :
    item.id === 3 ? "/images/demon_slayer_poster.png" :
    item.id === 4 ? "/images/jujutsu_kaisen_poster.png" :
    "/images/attack_on_titan_poster.png"
  );

  return (
    <motion.button
      type="button"
      onClick={() => onOpen(item)}
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: index * 0.03, duration: 0.25 }}
      className={`group relative flex flex-col overflow-hidden rounded-xl border border-white/10 bg-[#0D0E18] text-right transition-all duration-300 hover:scale-[1.03] hover:border-fuchsia-500/50 ${
        compact ? "w-32 sm:w-36" : "w-full"
      }`}
    >
      {/* Poster Image - Pure, Crisp & Untinted (No colored overlay on top) */}
      <div className={`relative w-full overflow-hidden bg-black/60 ${compact ? "aspect-[3/4]" : "aspect-[2/3]"}`}>
        <img
          src={imageSrc}
          alt={item.titleAr}
          className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-105"
        />

        {/* Rating Badge at Bottom Right */}
        <div className="absolute bottom-1.5 right-1.5 z-10 rounded-md bg-black/80 px-1.5 py-0.5 text-[10px] font-extrabold text-yellow-400">
          ★ {item.rating?.toFixed?.(1) || "9.0"}
        </div>
      </div>

      {/* Title Footer */}
      <div className="p-2 text-right">
        <h3 className="truncate text-xs font-bold text-white group-hover:text-fuchsia-400 transition-colors">
          {item.titleAr}
        </h3>
        <p className="mt-0.5 truncate text-[10px] font-medium text-white/50">{item.year || "2023"}</p>
      </div>
    </motion.button>
  );
}
