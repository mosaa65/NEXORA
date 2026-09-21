/**
 * watchContent — the single source of truth for how the watch screen adapts to
 * the kind of work being played.
 *
 * It keeps the type-driven decisions (labels, sections, fallbacks) and the
 * small formatters out of the components, so WatchPage and the episode/related
 * cards stay consistent with the catalogue's `UnifiedMediaCard` vocabulary.
 */

/** Known media kinds in the NEXORA library. */
export const MEDIA_KINDS = ["movie", "series", "anime", "documentary", "play"];

const KIND_LABELS = {
  movie: "فيلم",
  series: "مسلسل",
  anime: "أنمي",
  documentary: "وثائقي",
  play: "مسرحية",
  season: "موسم",
};

/** Human label for a media kind. */
export function kindLabel(type) {
  return KIND_LABELS[type] || "عمل";
}

/** True when the work is consumed as a sequence of episodes. */
export function isEpisodic(type) {
  return type === "series" || type === "anime" || type === "documentary";
}

/** Title of the side list, by kind. */
export function listTitle(type) {
  if (type === "movie") return "ملفات الفيلم";
  if (type === "anime") return "الحلقات";
  if (type === "series") return "الحلقات";
  if (type === "documentary") return "الحلقات";
  return "الملفات";
}

/** Title of the "watch next" (related) section, by kind. */
export function relatedTitle(type) {
  if (type === "movie") return "أفلام مشابهة";
  if (type === "anime") return "أنمي مشابه";
  if (type === "series") return "مسلسلات مشابهة";
  if (type === "documentary") return "وثائقيات مشابهة";
  return "أعمال ذات صلة";
}

/** Word used for a single playable item, by kind. */
export function itemNoun(type) {
  if (type === "movie") return "ملف";
  return "حلقة";
}

/** Format bytes into a compact human size. Empty string when unknown. */
export function formatSize(bytes) {
  const value = Number(bytes || 0);
  if (!Number.isFinite(value) || value <= 0) return "";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const unit = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  const amount = value / 1024 ** unit;
  return `${amount >= 10 || unit === 0 ? amount.toFixed(0) : amount.toFixed(1)} ${units[unit]}`;
}

/** Format minutes (or seconds when `isSeconds`) into "س/د" text. Empty when 0. */
export function formatRuntime(value, isSeconds = false) {
  let minutes = Number(value || 0);
  if (!Number.isFinite(minutes) || minutes <= 0) return "";
  if (isSeconds) minutes = Math.round(minutes / 60);
  if (minutes < 60) return `${minutes} د`;
  return `${Math.floor(minutes / 60)}س${minutes % 60 ? ` ${minutes % 60}د` : ""}`;
}

/** Clock for durations in seconds → "1:05:06" / "5:45". */
export function clock(seconds = 0) {
  const total = Math.max(0, Math.floor(Number.isFinite(seconds) ? seconds : 0));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  return h > 0
    ? `${h}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`
    : `${m}:${String(s).padStart(2, "0")}`;
}

/** Label that best describes one playable item. */
export function episodeLabel(item, index, type) {
  const ar = item?.title_ar || item?.episode_title_ar;
  const en = item?.title_en || item?.episode_title_en;
  if (ar) return ar;
  if (en) return en;
  if (item?.episode_number) return `الحلقة ${item.episode_number}`;
  return `${itemNoun(type)} ${index + 1}`;
}

/** The smart one-line meta string for an episode card (per card_plan rules). */
export function episodeFacts(item) {
  const facts = [];
  if (item?.air_date) facts.push(item.air_date);
  const runtime = formatRuntime(item?.runtime || item?.duration, Boolean(item?.duration && !item?.runtime));
  if (runtime) facts.push(runtime);
  if (item?.resolution) facts.push(item.resolution);
  const size = formatSize(item?.file_size);
  if (size) facts.push(size);
  return facts;
}

/** Read the locally saved playback progress for a file id. Returns null or a ratio. */
export function readProgress(fileId) {
  try {
    const raw = localStorage.getItem(`nexora:playback:${fileId}`);
    if (!raw) return null;
    const saved = JSON.parse(raw);
    if (!saved?.duration) return null;
    if (saved.completed) return { ratio: 1, position: saved.duration, completed: true };
    return { ratio: Math.min(1, Math.max(0, saved.position / saved.duration)), position: saved.position, completed: false };
  } catch {
    return null;
  }
}
