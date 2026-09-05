import React, { useEffect, useMemo, useState } from "react";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import Icon from "./Icon.jsx";
import { getShowcases, resolveAPIURL } from "../lib/api.js";

const accents = {
  violet: "from-violet-500 to-fuchsia-400",
  amber: "from-amber-400 to-orange-500",
  cyan: "from-cyan-400 to-blue-500",
  rose: "from-rose-400 to-pink-500",
  emerald: "from-emerald-400 to-teal-500",
};

const typeLabel = { movie: "فيلم", series: "مسلسل", anime: "أنمي", documentary: "وثائقي", play: "مسرحية" };
const statusLabel = { completed: "مكتمل", ongoing: "مستمر", returning: "مستمر", cancelled: "متوقف", ended: "مكتمل" };

const DEFAULT_FALLBACK_SLIDES = [
  { id: "fallback-1", kind: "fallback", title_ar: "اكتشف مكتبة NEXORA", description_ar: "تصفح أفلامك ومسلسلاتك وأنميك المفضل من مكان واحد.", artwork_path: "/nexora-library-backdrop.PNG", accent: "violet", type: "movie" },
  { id: "fallback-2", kind: "fallback", title_ar: "ليلة سينمائية تبدأ هنا", description_ar: "اختيارات متنوعة وتجربة مشاهدة سلسة داخل مكتبتك المحلية.", artwork_path: "/nexora-library-backdrop.PNG", accent: "cyan", type: "movie" },
  { id: "fallback-3", kind: "fallback", title_ar: "مسلسلات ومواسم كاملة", description_ar: "احتفظ بمتعتك قريبة، وانتقل بين الحلقات بسهولة.", artwork_path: "/nexora-library-backdrop.PNG", accent: "amber", type: "series" },
  { id: "fallback-4", kind: "fallback", title_ar: "عوالم الأنمي", description_ar: "مغامرات وحكايات لا تنتهي في مكتبتك الخاصة.", artwork_path: "/nexora-library-backdrop.PNG", accent: "rose", type: "anime" },
  { id: "fallback-5", kind: "fallback", title_ar: "مختارات تستحق المشاهدة", description_ar: "يتم تحديث العرض تلقائيًا عند جاهزية بيانات مكتبتك.", artwork_path: "/nexora-library-backdrop.PNG", accent: "emerald", type: "movie" },
  { id: "fallback-6", kind: "fallback", title_ar: "مكتبتك، بطريقتك", description_ar: "محاور ذكية وبطاقات مرتبة للوصول السريع إلى المحتوى.", artwork_path: "/nexora-library-backdrop.PNG", accent: "violet", type: "series" },
];

function fallbackSlides(items) {
  const itemSlides = (items || []).slice(0, 6).map((item) => ({
    id: `media-${item.id}`,
    kind: "featured",
    media_id: item.id,
    title_ar: item.titleAr || "",
    title_en: item.titleEn || "",
    description_ar: item.plot || "",
    artwork_path: item.bannerPath || item.posterPath || "",
    artwork_position: "center center",
    accent: "violet",
    type: item.type,
    status: item.status,
    release_year: item.year,
    rating: item.rating,
    best_resolution: item.bestResolution,
    genres: item.genres || [],
  }));
  return itemSlides.length ? itemSlides : DEFAULT_FALLBACK_SLIDES;
}

function accentForSlide(slide) {
  if (slide.kind === "fallback" || slide.kind === "collection") {
    return slide.accent || "violet";
  }

  const genres = (slide.genres || []).join(" ").toLowerCase();
  if (genres.includes("تركي") || genres.includes("turkish")) return "amber";
  if (genres.includes("أنمي") || genres.includes("anime") || genres.includes("ياباني") || genres.includes("كوري")) return "rose";
  if (genres.includes("عربي") || genres.includes("مصري") || genres.includes("arabic")) return "emerald";
  if (genres.includes("كرتون") || genres.includes("ديزني") || genres.includes("بيكسار") || genres.includes("animation")) return "cyan";

  switch (slide.type) {
    case "series": return "emerald";
    case "anime": return "rose";
    case "documentary": return "cyan";
    case "play": return "violet";
    case "movie": return "cyan";
    default: return slide.accent || "violet";
  }
}

function factsFor(slide) {
  if (slide.kind === "collection") {
    return slide.item_count ? [`${slide.item_count} عمل` ] : [];
  }
  return [
    typeLabel[slide.type] || slide.type,
    statusLabel[slide.status] || "",
    slide.release_year ? String(slide.release_year) : "",
    slide.best_resolution || "",
  ].filter(Boolean).slice(0, 4);
}

export default function ShowcaseHero({ context = "home", category, fallbackItems = [], onOpenMedia, onNavigate }) {
  const [remoteSlides, setRemoteSlides] = useState([]);
  const [active, setActive] = useState(0);
  const [paused, setPaused] = useState(false);
  const reduceMotion = useReducedMotion();

  useEffect(() => {
    let alive = true;
    setActive(0);
    getShowcases({ context, category, limit: 6 })
      .then((payload) => { if (alive) setRemoteSlides(payload?.slides || []); })
      .catch(() => { if (alive) setRemoteSlides([]); });
    return () => { alive = false; };
  }, [context, category]);

  const slides = useMemo(() => remoteSlides.length ? remoteSlides : fallbackSlides(fallbackItems), [remoteSlides, fallbackItems]);
  const index = Math.min(active, Math.max(0, slides.length - 1));
  const current = slides[index];

  useEffect(() => {
    if (slides.length < 2 || paused || reduceMotion) return undefined;
    const timer = window.setInterval(() => setActive((value) => (value + 1) % slides.length), 6000);
    return () => window.clearInterval(timer);
  }, [slides.length, paused, reduceMotion]);

  if (!current) return null;

  const title = current.title_ar || current.title_en || "مكتبة NEXORA";
  const englishOnly = !current.title_ar && current.title_en;
  const description = current.description_ar || current.description_en || "مختارات من مكتبتك المحلية، جاهزة للاستكشاف.";
  const artwork = resolveAPIURL(current.artwork_path) || "/nexora-library-backdrop.PNG";
  const accent = accents[accentForSlide(current)] || accents.violet;
  const facts = factsFor(current);
  const isCollection = current.kind === "collection";

  const isFallback = current.kind === "fallback";
  const open = () => {
    if (isFallback || !current.media_id) return;
    setPaused(true);
    if (isCollection) onNavigate?.(current.target || {});
    else onOpenMedia?.({ id: current.media_id, titleAr: current.title_ar, titleEn: current.title_en });
  };

  return (
    <section
      className="luminous-hero relative isolate min-h-[360px] overflow-hidden rounded-2xl bg-[#0E0C1A] sm:min-h-[420px] lg:min-h-[480px]"
      dir="rtl"
      aria-label="العرض المميز"
      onMouseEnter={() => setPaused(true)}
      onFocusCapture={() => setPaused(true)}
      onPointerDown={() => setPaused(true)}
    >
      <AnimatePresence mode="wait" initial={false}>
        <motion.img
          key={current.id}
          src={artwork}
          alt=""
          className="absolute inset-0 -z-30 h-full w-full object-cover"
          style={{ objectPosition: current.artwork_position || "center center" }}
          initial={reduceMotion ? false : { opacity: 0, scale: 1.025 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: reduceMotion ? 0 : 0.32, ease: "easeOut" }}
        />
      </AnimatePresence>
      <div className="absolute inset-0 -z-20 bg-gradient-to-l from-black/90 via-black/65 to-black/10" />
      <div className="absolute inset-0 -z-10 bg-gradient-to-t from-black/70 via-transparent to-black/20" />
      <div className={`absolute inset-y-0 right-0 -z-10 w-2 bg-gradient-to-b ${accent}`} />

      <div className="flex min-h-[360px] max-w-2xl flex-col justify-end gap-4 p-5 pb-24 sm:min-h-[420px] sm:p-8 sm:pb-20 lg:min-h-[480px] lg:p-10">
        <div className="flex flex-wrap items-center gap-2">
          <span className={`inline-flex items-center gap-1.5 rounded-full bg-gradient-to-l ${accent} px-3 py-1.5 text-xs font-extrabold text-white shadow-lg`}>
            <Icon name={isCollection ? "spark" : current.type === "movie" ? "film" : "tv"} className="h-3.5 w-3.5" />
            {isCollection ? "مجموعة مختارة" : typeLabel[current.type] || "عمل مميز"}
          </span>
          {!isCollection && current.rating > 0 && <span className="rounded-full border border-amber-300/30 bg-black/30 px-3 py-1.5 text-xs font-bold text-amber-100 backdrop-blur">★ {Number(current.rating).toFixed(1)}</span>}
        </div>

        <div className="space-y-2">
          <h1 className={`max-w-xl text-3xl font-black leading-[1.15] tracking-tight text-white drop-shadow sm:text-4xl lg:text-5xl ${englishOnly ? "text-left" : ""}`} dir={englishOnly ? "ltr" : "rtl"}>{title}</h1>
          {current.title_ar && current.title_en && <p className="text-sm font-semibold tracking-wide text-white drop-shadow-sm" dir="ltr">{current.title_en}</p>}
          <p className="line-clamp-2 max-w-xl text-sm leading-7 text-white drop-shadow-sm sm:text-base">{description}</p>
        </div>

        {facts.length > 0 && <div className="flex flex-wrap gap-2">{facts.map((fact) => <span key={fact} className="rounded-lg border border-white/20 bg-black/35 px-2.5 py-1.5 text-xs font-bold text-white backdrop-blur">{fact}</span>)}</div>}

        <div className="flex flex-wrap items-center gap-3 pt-1">
          <button type="button" onClick={open} disabled={isFallback} className={`min-h-11 rounded-xl bg-gradient-to-l ${accent} px-5 text-sm font-extrabold text-white shadow-lg transition hover:brightness-110 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white disabled:cursor-default disabled:opacity-90`}>
            {isFallback ? "جارٍ تجهيز مكتبتك" : isCollection ? "استكشف المجموعة" : "تفاصيل العمل"}
          </button>
          {isCollection && current.target?.category && <button type="button" onClick={() => onNavigate?.({ category: current.target.category })} className="min-h-11 rounded-xl border border-white/25 bg-black/20 px-5 text-sm font-bold text-white backdrop-blur transition hover:bg-white/15 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white">عرض الكل</button>}
        </div>
      </div>

      {slides.length > 1 && <div className="absolute bottom-4 left-1/2 flex max-w-[calc(100%-2rem)] -translate-x-1/2 items-center gap-1.5 rounded-2xl border border-white/15 bg-black/35 p-1.5 backdrop-blur sm:bottom-7 sm:left-7 sm:max-w-none sm:translate-x-0 sm:gap-2" aria-label="شرائح العرض">
        <button type="button" onClick={() => { setPaused(true); setActive((value) => (value - 1 + slides.length) % slides.length); }} aria-label="المحتوى السابق" title="المحتوى السابق" className="flex h-8 w-8 items-center justify-center rounded-xl text-lg text-white/80 transition hover:bg-white/15 hover:text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-white">‹</button>
        <div className="flex items-center gap-1.5 px-1">{slides.map((slide, slideIndex) => <button key={slide.id} type="button" onClick={() => { setPaused(true); setActive(slideIndex); }} aria-label={`عرض الشريحة ${slideIndex + 1}`} aria-current={slideIndex === index} className={`h-2.5 rounded-full transition-all ${slideIndex === index ? "w-7 bg-white" : "w-2.5 bg-white/40 hover:bg-white/70"}`} />)}</div>
        <button type="button" onClick={() => { setPaused(true); setActive((value) => (value + 1) % slides.length); }} aria-label="المحتوى التالي" title="المحتوى التالي" className="flex h-8 w-8 items-center justify-center rounded-xl text-lg text-white/80 transition hover:bg-white/15 hover:text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-white">›</button>
      </div>}
    </section>
  );
}
