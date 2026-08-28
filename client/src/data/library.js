import { MASTER_CATEGORIES, MASTER_CATEGORY_MAP } from "./categoryConfig.js";

const categoryProfiles = Object.fromEntries(
  MASTER_CATEGORIES.map((cat) => [
    cat.slug,
    {
      titleEn: cat.titleEn,
      titleAr: cat.titleAr,
      count: 0,
      accent: cat.accent,
      description: cat.description,
    },
  ])
);

const categoryTitleLookup = Object.fromEntries(
  Object.entries(categoryProfiles).map(([slug, profile]) => [slug, profile])
);

export const navigationItems = [
  { id: "dashboard", label: "الرئيسية", icon: "dashboard" },
  { id: "movies", label: "الأفلام السينمائية", icon: "film" },
  { id: "series", label: "المسلسلات والدراما", icon: "tv" },
  { id: "anime", label: "الأنمي والرسوم اليابانية", icon: "spark" },
  { id: "kids", label: "الأطفال والكرتون العائلي", icon: "smile" },
  { id: "family", label: "العائلة والسينما العائلية", icon: "smile" },
  { id: "documentaries", label: "الوثائقيات والمعرفة", icon: "book" },
  { id: "plays", label: "المسرحيات والكوميديا", icon: "mask" },
  { id: "ramadan", label: "الرمضانيات والإنتاج الرمضاني", icon: "spark" },
  { id: "wrestling", label: "المصارعة الحرة والرياضة", icon: "shield" },
  { id: "music", label: "المكتبة الصوتية والموسيقى", icon: "music" },
  { id: "apps", label: "البرامج والتطبيقات", icon: "grid" },
  { id: "favorites", label: "المفضلة", icon: "star" },
  { id: "downloads", label: "التنزيلات", icon: "download" },
  { id: "admin", label: "الإدارة والتخزين", icon: "settings" },
];

export const serviceItems = [
  { label: "واجهة Go", value: "متصل", tone: "text-emerald-300" },
  { label: "Postgres", value: "سليم", tone: "text-cyan-300" },
  { label: "Meilisearch", value: "متزامن", tone: "text-fuchsia-300" },
  { label: "Redis", value: "جاهز", tone: "text-orange-200" },
];

export const categorySeed = Object.entries(categoryProfiles).map(([slug, profile]) => ({
  slug,
  ...profile,
}));

export function getCategoryMeta(slug) {
  return categoryTitleLookup[slug] || categoryProfiles.movies;
}

export function getCategoryTitleEn(slug) {
  return getCategoryMeta(slug).titleEn;
}

export function getCategoryTitleAr(slug) {
  return getCategoryMeta(slug).titleAr;
}

export function getMediaTypeLabel(type) {
  const lookup = {
    movie: "فيلم",
    series: "مسلسل",
    anime: "أنمي",
    play: "مسرحية",
    documentary: "وثائقي",
    ramadan: "عمل رمضاني",
    wrestling: "عرض مصارعة",
    music: "محتوى صوتي",
    app: "برنامج / تطبيق",
    family: "عمل عائلي",
  };
  return lookup[type] || type;
}

// Clean mock library placeholder
export const mockLibrary = [];

export const streamStatusFeed = [
  {
    label: "البث المباشر",
    title: "خادم الوسائط المحلي جاهز",
    body: "جاهز لدفق ملفات الفيديو مباشرة وبأعلى جودة عبر الشبكة المحلية.",
    tone: "emerald",
  },
  {
    label: "الفهرسة",
    title: "مزامنة قاعدة البيانات",
    body: "يتم تحديث الفهارس وبيانات الأعمال تلقائياً عند إضافة أي محتوى جديد.",
    tone: "purple",
  },
  {
    label: "الماسح",
    title: "مراقبة المجلدات نشطة",
    body: "الخادم يراقب مسارات الوسائط المحددة ويضيف الملفات فور وصولها.",
    tone: "cyan",
  },
];

export const dashboardMetrics = [
  { label: "إجمالي المحتوى", value: "0", hint: "عنوان مفهرس" },
  { label: "إجمالي الحجم", value: "0 GB", hint: "سعة الأقراص" },
  { label: "الأفلام", value: "0", hint: "فيلم سينمائي" },
  { label: "المسلسلات", value: "0", hint: "مسلسل تلفزيوني" },
  { label: "الأنمي", value: "0", hint: "أنمي عربي/إنجليزي" },
  { label: "الحلقات", value: "0", hint: "حلقة محلية" },
];

export const detailEpisodes = [];

export function buildHeroCopy() {
  return {
    eyebrow: "NEXORA",
    title: "مكتبة الوسائط الرقمية",
    subtitle: "تجربة تصفح وبث سينمائية منزلية متكاملة لجميع أعمالك الفنية.",
  };
}

export function findMockMedia(id) {
  return null;
}
