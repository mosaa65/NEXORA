const categoryProfiles = {
  movies: {
    titleEn: "Movies",
    titleAr: "أفلام",
    count: 4820,
    accent: "linear-gradient(135deg, rgba(25,183,255,0.34), rgba(90,50,244,0.24))",
    description: "أفلام سينمائية وعناوين جماهيرية مصفوفة لليالي المشاهدة الطويلة."
  },
  series: {
    titleEn: "Series",
    titleAr: "مسلسلات",
    count: 184,
    accent: "linear-gradient(135deg, rgba(16,185,129,0.3), rgba(59,130,246,0.22))",
    description: "مواسم متتابعة وحلقات جاهزة لرحلات المتابعة المستمرة."
  },
  anime: {
    titleEn: "Anime",
    titleAr: "أنمي",
    count: 211,
    accent: "linear-gradient(135deg, rgba(236,72,153,0.28), rgba(124,58,237,0.28))",
    description: "أرشيف أنمي عربي/إنجليزي بأسماء مختلطة وبحث سريع وفوري."
  },
  kids: {
    titleEn: "Kids",
    titleAr: "أطفال",
    count: 72,
    accent: "linear-gradient(135deg, rgba(245,158,11,0.32), rgba(239,68,68,0.18))",
    description: "منطقة عائلية هادئة وسهلة التصفح ومناسبة للصغار."
  },
  plays: {
    titleEn: "Plays",
    titleAr: "مسرحيات",
    count: 64,
    accent: "linear-gradient(135deg, rgba(234,179,8,0.28), rgba(168,85,247,0.22))",
    description: "عروض مسرحية وكوميدية محفوظة للعرض المحلي بدون انقطاع."
  },
  documentaries: {
    titleEn: "Documentaries",
    titleAr: "وثائقيات",
    count: 129,
    accent: "linear-gradient(135deg, rgba(14,165,233,0.3), rgba(20,184,166,0.22))",
    description: "معرفة، تاريخ، علوم، وغوص عميق في محتوى طويل المدى."
  }
};

const categoryTitleLookup = Object.fromEntries(
  Object.entries(categoryProfiles).map(([slug, profile]) => [slug, profile])
);

export const navigationItems = [
  { id: "dashboard", label: "الرئيسية", icon: "dashboard" },
  { id: "categories", label: "التصنيفات", icon: "library" },
  { id: "details", label: "تفاصيل العمل", icon: "play" },
  { id: "admin", label: "الإدارة", icon: "settings" }
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

export const mockLibrary = [
  {
    id: 1,
    titleEn: "Attack on Titan",
    titleAr: "هجوم العمالقة",
    type: "anime",
    categorySlug: "anime",
    year: 2013,
    resolution: "1080p",
    seasonLabel: "الموسم 4",
    episodeLabel: "الحلقة 5",
    duration: "24 دقيقة",
    rating: 9.1,
    fileCount: 88,
    plot: "تقف البشرية خلف أسوار هائلة بينما يتكشف سر مرعب عن العمالقة ويقلب موازين العالم.",
    gradient:
      "linear-gradient(135deg, rgba(11,15,33,0.98), rgba(88,28,135,0.75), rgba(25,183,255,0.55))",
    highlights: ["أكشن", "فانتازيا مظلمة", "ترجمة عربية"],
    seasons: [1, 2, 3, 4]
  },
  {
    id: 2,
    titleEn: "The Last of Us",
    titleAr: "آخرنا",
    type: "series",
    categorySlug: "series",
    year: 2023,
    resolution: "4K",
    seasonLabel: "الموسم 1",
    episodeLabel: "الحلقة 2",
    duration: "58 دقيقة",
    rating: 8.7,
    fileCount: 9,
    plot: "يرافق ناجٍ قاسٍ فتاة صغيرة عبر عالم ينهار، وكل ميل يترك أثرًا شخصيًا جديدًا.",
    gradient:
      "linear-gradient(135deg, rgba(8,15,31,0.98), rgba(15,118,110,0.55), rgba(250,204,21,0.25))",
    highlights: ["دراما", "بقاء", "HDR"],
    seasons: [1]
  },
  {
    id: 3,
    titleEn: "Dune",
    titleAr: "كثيب",
    type: "movie",
    categorySlug: "movies",
    year: 2021,
    resolution: "4K",
    duration: "ساعتان و35 دقيقة",
    rating: 8.0,
    fileCount: 1,
    plot: "تنخرط عائلة نبيلة في سياسة كوكب صحراوي يسيطر على أثمن مورد في الكون.",
    gradient:
      "linear-gradient(135deg, rgba(24,24,27,0.96), rgba(146,64,14,0.66), rgba(251,191,36,0.22))",
    highlights: ["خيال علمي", "ملحمي", "Dolby Vision"],
    seasons: []
  },
  {
    id: 4,
    titleEn: "Blue Lock",
    titleAr: "بلو لوك",
    type: "anime",
    categorySlug: "anime",
    year: 2022,
    resolution: "1080p",
    seasonLabel: "الموسم 1",
    episodeLabel: "الحلقة 11",
    duration: "23 دقيقة",
    rating: 8.3,
    fileCount: 24,
    plot: "يدفع مشروع رياضي قاسٍ المهاجمين إلى منافسة لا ينجو منها إلا أكثرهم أنانية.",
    gradient:
      "linear-gradient(135deg, rgba(8,47,73,0.95), rgba(14,165,233,0.5), rgba(37,99,235,0.24))",
    highlights: ["رياضة", "إيقاع سريع", "مدبلج"],
    seasons: [1]
  },
  {
    id: 5,
    titleEn: "Planet Earth",
    titleAr: "كوكب الأرض",
    type: "movie",
    categorySlug: "documentaries",
    year: 2006,
    resolution: "1080p",
    duration: "1 ساعة و39 دقيقة",
    rating: 9.4,
    fileCount: 1,
    plot: "تُلتقط أروع بيئات الكوكب وكائناته بعدسة سينمائية هادئة وصبورة ومبهرة.",
    gradient:
      "linear-gradient(135deg, rgba(4,47,46,0.96), rgba(20,184,166,0.4), rgba(59,130,246,0.18))",
    highlights: ["طبيعة", "حائز على جوائز", "نسخة محسنة"],
    seasons: []
  },
  {
    id: 6,
    titleEn: "The Bear",
    titleAr: "ذا بير",
    type: "series",
    categorySlug: "series",
    year: 2022,
    resolution: "1080p",
    seasonLabel: "الموسم 2",
    episodeLabel: "الحلقة 4",
    duration: "34 دقيقة",
    rating: 8.9,
    fileCount: 18,
    plot: "يعود طاهٍ إلى منزله ليدير متجر الساندويتش العائلي ويكتشف أن المطبخ ساحة معركة كاملة.",
    gradient:
      "linear-gradient(135deg, rgba(69,10,10,0.96), rgba(220,38,38,0.56), rgba(249,115,22,0.18))",
    highlights: ["دراما", "إيقاع عالي", "صوت متعدد"],
    seasons: [1, 2, 3]
  }
];

export const mockActivity = [
  {
    label: "المفهرس",
    title: "اكتملت مزامنة Meilisearch",
    body: "تمت إضافة عنوانين جديدين من PostgreSQL وأصبحا متاحين للبحث مباشرة.",
    tone: "emerald"
  },
  {
    label: "الماسح",
    title: "مراقبة المجلدات نشطة",
    body: "الخادم يراقب مسارات الوسائط المحددة ويضيف الملفات الجديدة فور وصولها.",
    tone: "cyan"
  },
  {
    label: "قاعدة البيانات",
    title: "المخطط سليم",
    body: "تم تثبيت الترحيلات، وجداول الكتالوج جاهزة للتوسع القادم.",
    tone: "violet"
  }
];

export const dashboardMetrics = [
  { label: "حجم المكتبة", value: "5.4 تيرابايت", hint: "داخل الأقسام النشطة" },
  { label: "العناصر المفهرسة", value: "5,416", hint: "أفلام، مسلسلات، أنمي" },
  { label: "الخدمات الحية", value: "3/3", hint: "Postgres و Meili و Redis" },
  { label: "الوافدون الجدد", value: "18 اليوم", hint: "وصلوا للتو من الماسح" }
];

export const detailEpisodes = [
  { number: 1, title: "الإشارة في الضجيج" },
  { number: 2, title: "الجدران والتحذيرات" },
  { number: 3, title: "مدينة تحت المراقبة" },
  { number: 4, title: "وردية الليل" },
  { number: 5, title: "الاختراق" },
  { number: 6, title: "رماد وارتدادات" }
];

export function buildHeroCopy() {
  return {
    eyebrow: "لوحة السينما",
    title: "مرحبًا بك في مكتبة NEXORA العربية",
    subtitle:
      "واجهة فخمة وسريعة للبحث في العربية والإنجليزية، ومراقبة الخادم، والتنقل بين الأقسام كأنك داخل غرفة تشغيل سينمائية."
  };
}

export function findMockMedia(id) {
  return mockLibrary.find((item) => String(item.id) === String(id)) || mockLibrary[0];
}
