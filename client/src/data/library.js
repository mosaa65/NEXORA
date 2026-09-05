const categoryProfiles = {
  movies: {
    titleEn: "Movies",
    titleAr: "الأفلام",
    count: 5430,
    accent: "linear-gradient(135deg, rgba(25,183,255,0.34), rgba(90,50,244,0.24))",
    description: "أفلام سينمائية وعناوين عالمية مصفوفة لليالي المشاهدة الطويلة."
  },
  series: {
    titleEn: "Series",
    titleAr: "المسلسلات",
    count: 2650,
    accent: "linear-gradient(135deg, rgba(16,185,129,0.3), rgba(59,130,246,0.22))",
    description: "مواسم متتابعة وحلقات جاهزة لرحلات المتابعة المستمرة."
  },
  anime: {
    titleEn: "Anime",
    titleAr: "الأنمي",
    count: 8350,
    accent: "linear-gradient(135deg, rgba(236,72,153,0.28), rgba(124,58,237,0.28))",
    description: "أرشيف أنمي عربي/إنجليزي بأسماء مختلطة وبحث سريع وفوري."
  },
  kids: {
    titleEn: "Kids",
    titleAr: "الأطفال",
    count: 420,
    accent: "linear-gradient(135deg, rgba(245,158,11,0.32), rgba(239,68,68,0.18))",
    description: "منطقة عائلية هادئة وسهلة التصفح ومناسبة للصغار."
  },
  plays: {
    titleEn: "Plays",
    titleAr: "المسرحيات",
    count: 180,
    accent: "linear-gradient(135deg, rgba(234,179,8,0.28), rgba(168,85,247,0.22))",
    description: "عروض مسرحية وكوميدية محفوظة للعرض المحلي بدون انقطاع."
  },
  documentaries: {
    titleEn: "Documentaries",
    titleAr: "الوثائقيات",
    count: 310,
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

export const mockLibrary = [
  {
    id: 1,
    titleEn: "Tokyo Ghoul",
    titleAr: "طوكيو غول",
    type: "anime",
    categorySlug: "anime",
    year: 2014,
    resolution: "1080p",
    seasonLabel: "الموسم 1",
    episodeLabel: "الحلقة 12",
    duration: "24 دقيقة",
    rating: 8.7,
    fileCount: 48,
    plot: "في طوكيو حيث تعيش غيلان بين البشر بالتخفي، تنقلب حياة الشاب (كانيكي) عندما تلتهمه إحدى الغيلان بدلاً من أن تصبح عشاءه، فيتحول إلى نصف بشري ونصف غول محاصر بين عالمين.",
    posterPath: "/nexora-poster-placeholder.PNG",
    bannerPath: "/nexora-library-backdrop.PNG",
    highlights: ["أنمي", "فانتازيا مظلمة", "رعب"],
    seasons: [1, 2, 3, 4]
  },
  {
    id: 2,
    titleEn: "Attack on Titan",
    titleAr: "هجوم العمالقة",
    type: "anime",
    categorySlug: "anime",
    year: 2013,
    resolution: "1080p",
    seasonLabel: "الموسم 4",
    episodeLabel: "الحلقة 28",
    duration: "24 دقيقة",
    rating: 9.0,
    fileCount: 88,
    plot: "منذ مائة عام، ظهرت العمالقة فجأة ودمرت معظم البشرية. يعيش الباقون في عالم محاط بأسوار ضخمة لحمايتهم من العمالقة... عندما يُخترق السور الأول، يبدأ إيرين غيغار رحلة الانتقام والبحث عن الحقيقة.",
    posterPath: "/nexora-poster-placeholder.PNG",
    bannerPath: "/nexora-library-backdrop.PNG",
    highlights: ["خيال مظلم", "دراما", "أكشن"],
    views: "2.3M",
    seasons: [
      { number: 1, title: "الموسم الأول", episodeCount: 25 },
      { number: 2, title: "الموسم الثاني", episodeCount: 12 },
      { number: 3, title: "الموسم الثالث", episodeCount: 22 },
      { number: 4, title: "الموسم الرابع", episodeCount: 28 },
      { number: 5, title: "الموسم الأخير", episodeCount: 16 }
    ]
  },
  {
    id: 3,
    titleEn: "Demon Slayer",
    titleAr: "ديمون سلاير",
    type: "anime",
    categorySlug: "anime",
    year: 2023,
    resolution: "1080p",
    seasonLabel: "الموسم 3",
    episodeLabel: "الحلقة 11",
    duration: "24 دقيقة",
    rating: 9.1,
    fileCount: 55,
    plot: "يتعهد تانجيرو كاماتو بالانتقام لعائلته وإعادة أخته نيزوكو إلى هيئتها البشرية بعد تحولها إلى شيطان، منضماً إلى فيلق قتلة الشياطين.",
    posterPath: "/nexora-poster-placeholder.PNG",
    bannerPath: "/nexora-library-backdrop.PNG",
    highlights: ["أكشن", "شياطين", "سيوف"],
    seasons: [1, 2, 3]
  },
  {
    id: 4,
    titleEn: "Jujutsu Kaisen",
    titleAr: "جوجوتسو كايسن",
    type: "anime",
    categorySlug: "anime",
    year: 2023,
    resolution: "1080p",
    seasonLabel: "الموسم 2",
    episodeLabel: "الحلقة 23",
    duration: "24 دقيقة",
    rating: 9.0,
    fileCount: 47,
    plot: "ينتلع الفتى إيتادوري يوجي إصبع الساحر الأسطوري ريومن سوكونا، فيصبح وعاءً له وينضم لمدرسة جوجوتسو لمكافحة اللعنات.",
    posterPath: "/nexora-poster-placeholder.PNG",
    bannerPath: "/nexora-library-backdrop.PNG",
    highlights: ["لعنات", "خوارق", "معارك"],
    seasons: [1, 2]
  },
  {
    id: 5,
    titleEn: "One Piece",
    titleAr: "ون بيس",
    type: "anime",
    categorySlug: "anime",
    year: 2023,
    resolution: "1080p",
    seasonLabel: "الموسم 20",
    episodeLabel: "الحلقة 1086",
    duration: "24 دقيقة",
    rating: 9.0,
    fileCount: 1086,
    plot: "ينطلق مونكي دي لوفي وطاقم قبعة القش في رحلة أسطورية عبر البحار للبحث عن الكنز الأعظم ون بيس وليصبح ملك القراصنة.",
    posterPath: "/nexora-poster-placeholder.PNG",
    bannerPath: "/nexora-library-backdrop.PNG",
    highlights: ["مغامرة", "قراصنة", "كوميديا"],
    seasons: [1, 2, 3, 4, 5]
  },
  {
    id: 6,
    titleEn: "One Punch Man",
    titleAr: "ون بنش مان",
    type: "anime",
    categorySlug: "anime",
    year: 2015,
    resolution: "1080p",
    seasonLabel: "الموسم 2",
    episodeLabel: "الحلقة 12",
    duration: "24 دقيقة",
    rating: 8.3,
    fileCount: 24,
    plot: "سايتاما بطل خارق يقضي على أي خصم بلكمة واحدة فقط، ويبحث عن مواجهة حقيقية تعيد له حماس القتال.",
    posterPath: "/nexora-poster-placeholder.PNG",
    highlights: ["أبطال خارقون", "كوميديا", "قتال"],
    seasons: [1, 2]
  },
  {
    id: 7,
    titleEn: "Naruto Shippuden",
    titleAr: "ناروتو شيبودن",
    type: "anime",
    categorySlug: "anime",
    year: 2017,
    resolution: "1080p",
    seasonLabel: "الموسم 21",
    episodeLabel: "الحلقة 500",
    duration: "24 دقيقة",
    rating: 8.6,
    fileCount: 500,
    plot: "يعود ناروتو بعد سنوات التدريب يسعى لحماية قريته واستعادة صديقه ساسكي وتحقيق حلمه بلقب الهوكاجي.",
    posterPath: "/nexora-poster-placeholder.PNG",
    bannerPath: "/nexora-library-backdrop.PNG",
    highlights: ["نينجا", "صداقة", "معارك"],
    seasons: [1, 2, 3]
  },
  {
    id: 8,
    titleEn: "Bleach",
    titleAr: "بليتش",
    type: "anime",
    categorySlug: "anime",
    year: 2012,
    resolution: "1080p",
    seasonLabel: "الموسم 16",
    episodeLabel: "الحلقة 366",
    duration: "24 دقيقة",
    rating: 8.5,
    fileCount: 366,
    plot: "يحصل إيتشيغو كوروساكي على قوى الشينيغامي لحماية البشر والأرواح من الوحوش الجائعة.",
    posterPath: "/nexora-poster-placeholder.PNG",
    highlights: ["شينيغامي", "سيوف", "أرواح"],
    seasons: [1, 2]
  },
  {
    id: 9,
    titleEn: "Detective Conan",
    titleAr: "المحقّق كونان",
    type: "anime",
    categorySlug: "anime",
    year: 2023,
    resolution: "1080p",
    seasonLabel: "الموسم 31",
    episodeLabel: "الحلقة 1100",
    duration: "24 دقيقة",
    rating: 8.4,
    fileCount: 1100,
    plot: "يتناول سينشي كودو عقاراً يقلص جسده إلى طفل فيتخذ اسم كونان إيدوجاوا ويحل أعقد القضايا أثناء تتبع العصابة السوداء.",
    posterPath: "/nexora-poster-placeholder.PNG",
    highlights: ["غموض", "تحقيق", "ذكاء"],
    seasons: [1, 2, 3]
  },
  {
    id: 10,
    titleEn: "Dragon Ball Super",
    titleAr: "دراغون بول سوبر",
    type: "anime",
    categorySlug: "anime",
    year: 2018,
    resolution: "1080p",
    seasonLabel: "الموسم 5",
    episodeLabel: "الحلقة 131",
    duration: "24 دقيقة",
    rating: 8.4,
    fileCount: 131,
    plot: "يخوض غوكو ومحاربو الزد مواجهات أسطورية ضد آلهة الدمار ومحاربي الأكوان في بطولات القوة.",
    posterPath: "/nexora-poster-placeholder.PNG",
    highlights: ["تحولات", "معارك خارقة", "أكوان"],
    seasons: [1, 2, 3, 4, 5]
  }
];

export const mockActivity = [
  {
    label: "المفهرس",
    title: "اكتملت مزامنة Meilisearch",
    body: "تمت إضافة عناوين أنمي جديدة وأصبحت متاحة للبحث الفوري.",
    tone: "emerald"
  },
  {
    label: "الماسح",
    title: "مراقبة المجلدات نشطة",
    body: "الخادم يراقب مسارات الوسائط المحددة ويضيف الملفات فور وصولها.",
    tone: "cyan"
  }
];

export const dashboardMetrics = [
  { label: "إجمالي المحتوى", value: "18,430", hint: "عنوان مفهرس" },
  { label: "إجمالي الحجم", value: "98.7 TB", hint: "سعة الأقراص" },
  { label: "الأفلام", value: "5,430", hint: "فيلم سينمائي" },
  { label: "المسلسلات", value: "2,650", hint: "مسلسل تلفزيوني" },
  { label: "الأنمي", value: "8,350", hint: "أنمي عربي/إنجليزي" },
  { label: "الحلقات", value: "112,000", hint: "حلقة محلية" }
];

export const detailEpisodes = [
  { number: 1, title: "الحلقة 01 - مكان إشارة المعركة", duration: "24:10" },
  { number: 2, title: "الحلقة 02 - عودة إلى الشكل", duration: "24:10" },
  { number: 3, title: "الحلقة 03 - ضوء الأمل", duration: "24:10" },
  { number: 4, title: "الحلقة 04 - الليل يتقدم", duration: "24:10" },
  { number: 5, title: "الحلقة 05 - رد فعل", duration: "24:10" },
  { number: 6, title: "الحلقة 06 - العملاق والشاحن", duration: "24:10" },
  { number: 7, title: "الحلقة 07 - ما وراء الجدار", duration: "24:10" },
  { number: 8, title: "الحلقة 08 - القتال النهائي", duration: "24:10" }
];

export function buildHeroCopy() {
  return {
    eyebrow: "طوكيو غول",
    title: "طوكيو غول",
    subtitle:
      "في طوكيو حيث تعيش غيلان بين البشر بالتخفي، تنقلب حياة الشاب (كانيكي) عندما تلتهمه إحدى الغيلان بدلاً من أن تصبح عشاءه، فيتحول إلى نصف بشري ونصف غول محاصر بين عالمين."
  };
}

export function findMockMedia(id) {
  return mockLibrary.find((item) => String(item.id) === String(id)) || mockLibrary[0];
}
