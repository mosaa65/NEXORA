import { motion } from "framer-motion";
import Icon from "./Icon";
import { resolveAPIURL } from "../lib/api";

const typeLabels = { movie: "فيلم", series: "مسلسل", anime: "أنمي", documentary: "وثائقي", play: "مسرحية", season: "موسم" };
const typeIcons = { movie: "film", series: "tv", anime: "spark", documentary: "film", play: "mask", season: "tv" };
const typeStyles = {
  movie: "border-sky-300/25 bg-sky-950/70 text-sky-100",
  series: "border-violet-300/25 bg-violet-950/70 text-violet-100",
  anime: "border-fuchsia-300/25 bg-fuchsia-950/70 text-fuchsia-100",
  documentary: "border-cyan-300/25 bg-cyan-950/70 text-cyan-100",
  play: "border-amber-300/25 bg-amber-950/70 text-amber-100",
  season: "border-indigo-300/25 bg-indigo-950/70 text-indigo-100",
};
const statusLabels = { completed: "مكتمل", ongoing: "يعرض الآن", upcoming: "قادم", cancelled: "ملغي" };
const statusStyles = {
  completed: "border-emerald-300/35 bg-emerald-500/20 text-emerald-50 shadow-[0_5px_18px_rgba(16,185,129,.20)]",
  ongoing: "border-fuchsia-300/35 bg-fuchsia-500/20 text-fuchsia-50 shadow-[0_5px_18px_rgba(217,70,239,.22)]",
  upcoming: "border-sky-300/35 bg-sky-500/20 text-sky-50 shadow-[0_5px_18px_rgba(14,165,233,.20)]",
  cancelled: "border-slate-300/25 bg-slate-500/20 text-slate-100",
};
const statusDotStyles = { completed: "bg-emerald-300", ongoing: "bg-fuchsia-300", upcoming: "bg-sky-300", cancelled: "bg-slate-300" };

const contentRatingStyles = {
  G: "border-emerald-400/35 bg-emerald-950/80 text-emerald-200",
  PG: "border-sky-400/35 bg-sky-950/80 text-sky-200",
  "PG-13": "border-amber-400/35 bg-amber-950/80 text-amber-200",
  R: "border-rose-500/40 bg-rose-950/85 text-rose-200",
  "NC-17": "border-rose-600/50 bg-rose-950 text-rose-300",
  "TV-Y": "border-emerald-400/35 bg-emerald-950/80 text-emerald-200",
  "TV-Y7": "border-sky-400/35 bg-sky-950/80 text-sky-200",
  "TV-G": "border-emerald-400/35 bg-emerald-950/80 text-emerald-200",
  "TV-PG": "border-sky-400/35 bg-sky-950/80 text-sky-200",
  "TV-14": "border-amber-400/35 bg-amber-950/80 text-amber-200",
  "TV-MA": "border-rose-500/40 bg-rose-950/85 text-rose-200",
  "18+": "border-rose-500/40 bg-rose-950/85 text-rose-200",
  "16+": "border-amber-400/35 bg-amber-950/80 text-amber-200",
};

function getContentRatingStyle(rating) {
  if (!rating) return "";
  const key = String(rating).trim().toUpperCase();
  return contentRatingStyles[key] || "border-white/20 bg-black/65 text-white/90";
}

function formatSize(bytes) {
  const value = Number(bytes || 0);
  if (!Number.isFinite(value) || value <= 0) return "";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const unit = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  const amount = value / 1024 ** unit;
  return `${amount >= 10 || unit === 0 ? amount.toFixed(0) : amount.toFixed(1)} ${units[unit]}`;
}

function formatRuntime(minutes) {
  const value = Number(minutes || 0);
  if (!Number.isFinite(value) || value <= 0) return "";
  return value < 60 ? `${value} د` : `${Math.floor(value / 60)}س${value % 60 ? ` ${value % 60}د` : ""}`;
}

function compactDetails(media, isSeason) {
  const isEpisodic = isSeason || media.type === "series" || media.type === "anime";
  const facts = [];
  if (isSeason) {
    if (media.seasonNumber) facts.push(`الموسم ${media.seasonNumber}`);
    if (media.fileCount) facts.push(`${media.fileCount} حلقة`);
  } else if (isEpisodic) {
    if (media.seasonCount) facts.push(`${media.seasonCount} مواسم`);
    if (media.tmdbSeasonCount && media.tmdbSeasonCount !== media.seasonCount) facts.push(`TMDB ${media.tmdbSeasonCount} مواسم`);
  } else {
    if (media.year) facts.push(String(media.year));
    const runtime = formatRuntime(media.runtimeMinutes);
    if (runtime) facts.push(runtime);
  }
  const size = formatSize(media.totalSize);
  if (size) facts.push(size);
  return facts.slice(0, 5);
}

/** The single, offline-first card for catalogue grids, carousels and seasons. */
export default function UnifiedMediaCard({ media, onOpen, variant = "standard", layout = "grid" }) {
  if (!media) return null;
  const item = {
    titleAr: media.titleAr ?? media.title_ar ?? "", titleEn: media.titleEn ?? media.title_en ?? "", type: media.type,
    year: media.year ?? media.releaseYear ?? media.release_year, rating: Number(media.rating || 0), posterPath: media.posterPath ?? media.poster_path,
    contentRating: media.contentRating ?? media.content_rating ?? "",
    status: media.status, seasonCount: Number(media.seasonCount ?? media.season_count ?? 0), seasonNumber: Number(media.seasonNumber ?? media.season_number ?? 0),
    tmdbSeasonCount: Number(media.tmdbSeasonCount ?? media.tmdb_season_count ?? 0), tmdbEpisodeCount: Number(media.tmdbEpisodeCount ?? media.tmdb_episode_count ?? 0),
    fileCount: Number(media.fileCount ?? media.file_count ?? 0), totalSize: media.totalSize ?? media.total_size, bestResolution: media.bestResolution ?? media.best_resolution,
    runtimeMinutes: media.runtimeMinutes ?? media.runtime_minutes, hasArabicAudio: media.hasArabicAudio ?? media.has_arabic_audio,
    hasArabicSubtitles: media.hasArabicSubtitles ?? media.has_arabic_subtitles, isSeason: Boolean(media.isSeason || media.type === "season"),
  };
  const title = item.titleAr || item.titleEn || "عنوان غير متوفر";
  const englishTitle = item.titleEn && item.titleEn !== title ? item.titleEn : "";
  const titleIsArabic = /[\u0600-\u06FF]/.test(title);
  const mediaKind = item.isSeason ? "season" : item.type;
  const posterURL = resolveAPIURL(item.posterPath) || "/nexora-poster-placeholder.PNG";
  const facts = compactDetails(item, item.isSeason);
  const status = statusLabels[item.status];
  const statusStyle = statusStyles[item.status] || "border-white/15 bg-black/50 text-white";
  const statusDotStyle = statusDotStyles[item.status] || "bg-white/70";
  const compact = variant === "compact";

  if (layout === "list") {
    return (
      <motion.button type="button" onClick={() => onOpen?.(media)} whileHover={{ x: -3 }} whileTap={{ scale: 0.99 }} transition={{ duration: 0.2 }}
        className="group flex w-full items-stretch overflow-hidden rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] text-right shadow-[var(--shadow-sm)] transition-[border-color,box-shadow] hover:border-fuchsia-400/55 hover:shadow-[var(--shadow-md)] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--color-accent)]" dir="rtl" aria-label={`فتح تفاصيل ${title}`}>
        <div className="relative w-24 shrink-0 overflow-hidden bg-black sm:w-32"><img src={posterURL} alt="" loading="lazy" onError={(event) => { event.currentTarget.src = "/nexora-poster-placeholder.PNG"; }} className="h-full min-h-[132px] w-full object-cover transition-transform duration-500 group-hover:scale-105" /><div className="absolute inset-0 bg-gradient-to-l from-black/35 to-transparent" />{item.bestResolution && <span className="absolute bottom-2 right-2 rounded-md border border-cyan-300/30 bg-cyan-950/75 px-1.5 py-0.5 text-[9px] font-black text-cyan-100 backdrop-blur">{item.bestResolution}</span>}</div>
        <div className="flex min-w-0 flex-1 flex-col justify-center p-3.5 sm:p-4">
          <div className="flex items-center justify-between gap-3"><div className="flex flex-wrap items-center gap-1.5"><span className={`inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 text-[9px] font-extrabold ${typeStyles[mediaKind] || "border-white/15 bg-black/55 text-white"}`}><Icon name={typeIcons[mediaKind] || "film"} className="h-3 w-3" />{typeLabels[mediaKind] || "مكتبة"}</span>{item.contentRating && <span className={`inline-flex items-center rounded-md border px-1.5 py-0.5 text-[9px] font-black tracking-wider ${getContentRatingStyle(item.contentRating)}`}>{item.contentRating}</span>}{status && <span className={`inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 text-[9px] font-extrabold ${statusStyle}`}><i className={`h-1.5 w-1.5 rounded-full ${statusDotStyle}`} />{status}</span>}</div>{item.rating > 0 && <span className="inline-flex items-center gap-1 text-[11px] font-black tabular-nums text-amber-500"><Icon name="star" className="h-3.5 w-3.5 fill-current stroke-0" />{item.rating.toFixed(1)}</span>}</div>
          <h3 dir={titleIsArabic ? "rtl" : "ltr"} className={`mt-2 truncate text-sm font-extrabold text-[var(--text-primary)] ${titleIsArabic ? "text-right" : "text-left"}`}>{title}</h3>{englishTitle && <p dir="ltr" className="mt-0.5 truncate text-left text-[11px] text-[var(--text-muted)]">{englishTitle}</p>}
          <div className="mt-2 flex flex-wrap items-center gap-1.5 text-[10px] font-semibold text-[var(--text-secondary)]">{facts.map((fact) => <span key={fact} className="rounded-md bg-[var(--bg-surface)] px-1.5 py-0.5">{fact}</span>)}{item.hasArabicAudio && <span className="rounded-md bg-emerald-500/10 px-1.5 py-0.5 text-[var(--color-success)] font-bold">صوت عربي</span>}{!item.hasArabicAudio && item.hasArabicSubtitles && <span className="rounded-md bg-cyan-500/10 px-1.5 py-0.5 text-[var(--color-info)] font-bold">ترجمة عربية</span>}</div>
        </div>
        <div className="flex w-10 items-center justify-center border-r border-[var(--border-subtle)] text-[var(--text-muted)] transition-colors group-hover:text-[var(--color-accent)]"><span aria-hidden="true">‹</span></div>
      </motion.button>
    );
  }

  return (
    <motion.button type="button" onClick={() => onOpen?.(media)} whileHover={{ y: -5 }} whileTap={{ scale: 0.985 }} transition={{ duration: 0.22, ease: [0.16, 1, 0.3, 1] }}
      className="group relative flex w-full flex-col overflow-hidden rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] text-right shadow-[var(--shadow-md)] transition-[border-color,box-shadow] duration-300 hover:border-fuchsia-400/55 hover:shadow-[var(--shadow-lg)] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--color-accent)]"
      aria-label={`فتح تفاصيل ${title}`} dir="rtl">
      <div className="relative aspect-[4/5] w-full overflow-hidden bg-[#151225]">
        <img src={posterURL} alt={`بوستر ${title}`} loading="lazy" onError={(event) => { event.currentTarget.src = "/nexora-poster-placeholder.PNG"; }} className="h-full w-full object-cover transition-transform duration-700 motion-reduce:transition-none group-hover:scale-[1.055]" />
        <div className="absolute inset-0 bg-[linear-gradient(180deg,rgba(5,4,12,.34)_0%,transparent_36%,rgba(8,7,14,.08)_52%,transparent_100%)]" />
        <div className="absolute inset-x-2 top-2 flex items-start justify-between gap-1">
          <div className="flex min-w-0 flex-nowrap items-center justify-end gap-1 whitespace-nowrap">
            <span className={`inline-flex shrink-0 items-center gap-1 rounded-md border px-1.5 py-1 text-[9px] font-extrabold shadow-sm backdrop-blur-md sm:px-2 sm:text-[10px] ${typeStyles[mediaKind] || "border-white/15 bg-black/55 text-white/95"}`}><Icon name={typeIcons[mediaKind] || "film"} className="h-2.5 w-2.5 shrink-0 sm:h-3 sm:w-3" />{typeLabels[mediaKind] || "مكتبة"}</span>
            {item.contentRating && <span className={`inline-flex shrink-0 items-center rounded-md border px-1.5 py-1 text-[9px] font-black tracking-wider shadow-sm backdrop-blur-md sm:px-2 sm:text-[10px] ${getContentRatingStyle(item.contentRating)}`}>{item.contentRating}</span>}
            {status && <span className={`inline-flex shrink-0 items-center gap-1 rounded-md border px-1.5 py-1 text-[9px] font-extrabold backdrop-blur-md sm:px-2 sm:text-[10px] ${statusStyle}`}><i className={`h-1.5 w-1.5 rounded-full ${statusDotStyle} ${item.status === "ongoing" ? "animate-pulse" : ""}`} />{status}</span>}
          </div>
          {item.rating > 0 && <span className="inline-flex shrink-0 items-center gap-1 rounded-md border border-amber-300/25 bg-black/55 px-1.5 py-1 text-[9px] font-black tabular-nums text-amber-100 shadow-sm backdrop-blur-md sm:px-2 sm:text-[10px]"><Icon name="star" className="h-2.5 w-2.5 fill-current stroke-0 text-amber-300 sm:h-3 sm:w-3" />{item.rating.toFixed(1)}</span>}
        </div>
        <div className="absolute inset-x-2.5 bottom-2.5 flex items-end justify-between gap-2">
          {item.bestResolution ? <span className="rounded-md border border-cyan-300/30 bg-cyan-950/70 px-2 py-1 text-[10px] font-black tracking-wide text-cyan-100 shadow-sm backdrop-blur-md">{item.bestResolution}</span> : <span />}
          <span className="translate-y-1 rounded-full border border-white/10 bg-black/35 px-2.5 py-1 text-[10px] font-bold text-white/90 opacity-0 backdrop-blur-md transition duration-200 group-hover:translate-y-0 group-hover:opacity-100 motion-reduce:transition-none">التفاصيل <span aria-hidden="true">‹</span></span>
        </div>
      </div>
      <div className={`relative min-h-[98px] ${compact ? "p-2.5" : "p-3 sm:p-4"}`}>
        <div className="absolute inset-x-3 top-0 h-px bg-[var(--border-subtle)]" />
        <h3 dir={titleIsArabic ? "rtl" : "ltr"} className={`line-clamp-2 text-[13px] font-extrabold leading-5 text-[var(--text-primary)] transition-colors group-hover:text-[var(--color-accent-hover)] sm:text-[15px] sm:leading-6 ${titleIsArabic ? "text-right" : "text-left tracking-[0.01em]"}`}>{title}</h3>
        {englishTitle && <p dir="ltr" className="mt-0.5 truncate text-left text-[10px] font-medium tracking-wide text-[var(--text-muted)] sm:text-[11px]">{englishTitle}</p>}
        <div className="mt-2 flex min-h-5 flex-nowrap items-center gap-1 overflow-x-auto whitespace-nowrap scrollbar-none touch-pan-x text-[9px] font-semibold leading-5 text-[var(--text-secondary)] sm:text-[10px]" aria-label="معلومات العمل">{facts.map((fact, index) => <span key={`${fact}-${index}`} className="inline-flex shrink-0 items-center gap-1 rounded-md bg-[var(--bg-surface)] px-1.5 py-px sm:px-2">{index > 0 && <i className="h-1 w-1 rounded-full bg-[var(--text-muted)]" />}{fact}</span>)}{item.hasArabicAudio && <span className="shrink-0 rounded-md border border-emerald-500/20 bg-emerald-500/10 px-1.5 py-px text-[9px] font-bold text-[var(--color-success)] sm:px-2 sm:text-[10px]">صوت عربي</span>}{!item.hasArabicAudio && item.hasArabicSubtitles && <span className="shrink-0 rounded-md border border-cyan-500/20 bg-cyan-500/10 px-1.5 py-px text-[9px] font-bold text-[var(--color-info)] sm:px-2 sm:text-[10px]">ترجمة عربية</span>}</div>
      </div>
    </motion.button>
  );
}
