const categoryProfiles = {
  movies: {
    titleEn: "Movies",
    titleAr: "الأفلام",
    count: 0,
    accent: "linear-gradient(135deg, rgba(25,183,255,0.34), rgba(90,50,244,0.24))",
    description: "أفلام سينمائية وعناوين عالمية مصفوفة لليالي المشاهدة الطويلة."
  },
  series: {
    titleEn: "Series",
    titleAr: "المسلسلات",
    count: 0,
    accent: "linear-gradient(135deg, rgba(16,185,129,0.3), rgba(59,130,246,0.22))",
    description: "مواسم متتابعة وحلقات جاهزة لرحلات المتابعة المستمرة."
  },
  anime: {
    titleEn: "Anime",
    titleAr: "الأنمي",
    count: 0,
    accent: "linear-gradient(135deg, rgba(236,72,153,0.28), rgba(124,58,237,0.28))",
    description: "أرشيف أنمي عربي/إنجليزي بأسماء مختلطة وبحث سريع وفوري."
  },
  kids: {
    titleEn: "Kids",
    titleAr: "الأطفال",
    count: 0,
    accent: "linear-gradient(135deg, rgba(245,158,11,0.32), rgba(239,68,68,0.18))",
    description: "منطقة عائلية هادئة وسهلة التصفح ومناسبة للصغار."
  },
  plays: {
    titleEn: "Plays",
    titleAr: "المسرحيات",
    count: 0,
    accent: "linear-gradient(135deg, rgba(234,179,8,0.28), rgba(168,85,247,0.22))",
    description: "عروض مسرحية وكوميدية محفوظة للعرض المحلي بدون انقطاع."
  },
  documentaries: {
    titleEn: "Documentaries",
    titleAr: "الوثائقيات",
    count: 0,
    accent: "linear-gradient(135deg, rgba(14,165,233,0.3), rgba(20,184,166,0.22))",
    description: "معرفة، تاريخ، علوم، وغوص عميق في محتوى طويل المدى."
  }
};

const categoryTitleLookup = Object.fromEntries(
  Object.entries(categoryProfiles).map(([slug, profile]) => [slug, profile])
);

export const navigationItems = [
  { id: "dashboard", label: "الرئيسية", icon: "dashboard" },
  { id: "movies", label: "الأفلام", icon: "film" },
  { id: "series", label: "المسلسلات", icon: "tv" },
  { id: "anime", label: "الأنمي", icon: "spark" },
  { id: "kids", label: "الأطفال", icon: "smile" },
  { id: "plays", label: "المسرحيات", icon: "mask" },
  { id: "documentaries", label: "الوثائقيات", icon: "book" },
  { id: "favorites", label: "المفضلة", icon: "star" },
  { id: "downloads", label: "التنزيلات", icon: "download" },
  { id: "admin", label: "الإدارة والتخزين", icon: "settings" }
];

export const serviceItems = [
  { label: "واجهة Go", value: "متصل", tone: "text-emerald-300" },
  { label: "Postgres", value: "سليم", tone: "text-cyan-300" },
  { label: "Meilisearch", value: "متزامن", tone: "text-fuchsia-300" },
  { label: "Redis", value: "جاهز", tone: "text-orange-200" }
];

export const categorySeed = Object.entries(categoryProfiles).map(([slug, profile]) => ({
  slug,
  ...profile
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
    documentary: "وثائقي"
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
    tone: "emerald"
  },
  {
    label: "الفهرسة",
    title: "مزامنة قاعدة البيانات",
    body: "يتم تحديث الفهارس وبيانات الأعمال تلقائياً عند إضافة أي محتوى جديد.",
    tone: "purple"
  },
  {
    label: "الماسح",
    title: "مراقبة المجلدات نشطة",
    body: "الخادم يراقب مسارات الوسائط المحددة ويضيف الملفات فور وصولها.",
    tone: "cyan"
  }
];

export const dashboardMetrics = [
  { label: "إجمالي المحتوى", value: "0", hint: "عنوان مفهرس" },
  { label: "إجمالي الحجم", value: "0 GB", hint: "سعة الأقراص" },
  { label: "الأفلام", value: "0", hint: "فيلم سينمائي" },
  { label: "المسلسلات", value: "0", hint: "مسلسل تلفزيوني" },
  { label: "الأنمي", value: "0", hint: "أنمي عربي/إنجليزي" },
  { label: "الحلقات", value: "0", hint: "حلقة محلية" }
];

export const detailEpisodes = [];

export function buildHeroCopy() {
  return {
    eyebrow: "NEXORA",
    title: "مكتبة الوسائط الرقمية",
    subtitle: "تجربة تصفح وبث سينمائية منزلية متكاملة لجميع أعمالك الفنية."
  };
}

export function findMockMedia(id) {
  return null;
}
