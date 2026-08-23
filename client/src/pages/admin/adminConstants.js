/**
 * Shared constants and utilities for Admin pages.
 * Extracted from the monolithic AdminPage.jsx to avoid duplication.
 */

export const DEFAULT_CATEGORIES = [
  { id: 1, slug: "movies", name_ar: "الأفلام", name_en: "Movies", icon: "film", color: "from-blue-600 to-indigo-700" },
  { id: 2, slug: "series", name_ar: "المسلسلات", name_en: "Series", icon: "tv", color: "from-emerald-600 to-teal-700" },
  { id: 3, slug: "anime", name_ar: "الأنمي", name_en: "Anime", icon: "spark", color: "from-purple-600 to-fuchsia-700" },
  { id: 4, slug: "kids", name_ar: "الأطفال والكرتون", name_en: "Kids & Cartoons", icon: "smile", color: "from-amber-500 to-orange-600" },
  { id: 5, slug: "documentaries", name_ar: "الوثائقيات", name_en: "Documentaries", icon: "book", color: "from-cyan-600 to-blue-700" },
  { id: 6, slug: "plays", name_ar: "المسرحيات", name_en: "Plays", icon: "mask", color: "from-rose-600 to-pink-700" },
];

export const ALL_GENRES = [
  "أكشن", "مغامرة", "دراما", "كوميديا", "خيال علمي", "غموض", "إثارة", "رعب", "رومانسي",
  "فانتازيا", "تاريخي", "سيرة ذاتية", "جريمة", "شياطين", "نينجا", "طبيعة", "عائلي", "رياضي", "حرب", "موسيقى",
  "تركي", "عربي", "كوري", "أجنبي", "هندي", "إسباني",
];

export const COUNTRIES_LIST = [
  { id: "all", label: "كافة الجنسيات" },
  { id: "hollywood", label: "أجنبي (هوليوود)" },
  { id: "arabic", label: "عربي" },
  { id: "turkish", label: "تركي" },
  { id: "korean", label: "كوري" },
  { id: "japanese", label: "ياباني / أنمي" },
  { id: "indian", label: "هندي (بوليوود)" },
];

export const COUNTRY_TAGS = {
  hollywood: ["أجنبي", "أجنبية", "english", "american", "british", "hollywood", "أمريكي", "بريطاني"],
  arabic: ["عربي", "عربية", "arabic", "مصري", "خليجي"],
  turkish: ["تركي", "تركية", "turkish", "turkey"],
  korean: ["كوري", "كورية", "korean", "korea"],
  japanese: ["ياباني", "يابانية", "japanese", "anime", "أنمي"],
  indian: ["هندي", "هندية", "indian", "india", "bollywood"],
};

export const QUALITIES_LIST = [
  { id: "all", label: "كافة الجودات" },
  { id: "4k", label: "4K UHD فائقة" },
  { id: "1080p", label: "1080p FHD عالية" },
  { id: "720p", label: "720p HD متوسطة" },
  { id: "480p", label: "480p SD عادية" },
];

export const YEARS_LIST = [
  { id: "all", label: "كافة السنوات" },
  { id: "2024", label: "2024 حديث" },
  { id: "2023", label: "2023" },
  { id: "2020-2022", label: "2020 - 2022" },
  { id: "2010s", label: "2010 - 2019" },
  { id: "classic", label: "أقدم من 2010" },
];

export const RATINGS_LIST = [
  { id: "all", label: "كافة التقييمات" },
  { id: "9", label: "★ 9.0 فما فوق (تحف فنية)" },
  { id: "8", label: "★ 8.0 فما فوق (ممتاز جداً)" },
  { id: "7", label: "★ 7.0 فما فوق (جيد)" },
];

/**
 * Formats byte values to human-readable string.
 */
export function formatBytes(bytes) {
  if (!bytes || bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB", "PB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}
