import { useEffect, useMemo, useState } from "react";
import MediaCard from "../components/MediaCard.jsx";
import Icon from "../components/Icon.jsx";
import { getMediaList, resolveAPIURL } from "../lib/api.js";

const categoryTitles = {
  anime: { titleAr: "الأنمي", label: "أنمي", description: "اكتشف عوالم الأنمي، المواسم والحلقات المفضلة لديك." },
  movies: { titleAr: "الأفلام", label: "فيلم", description: "تصفح مكتبتك السينمائية واختر فيلم الليلة." },
  series: { titleAr: "المسلسلات", label: "مسلسل", description: "اختر طريقة الاستكشاف المناسبة لك، ثم انغمس في عالمك المفضل." },
  kids: { titleAr: "أطفال وكرتون", label: "عمل", description: "محتوى عائلي ممتع ومناسب للصغار." },
  documentaries: { titleAr: "وثائقيات", label: "وثائقي", description: "معرفة وقصص حقيقية من مكتبتك." },
  plays: { titleAr: "مسرحيات", label: "مسرحية", description: "عروض مسرحية وكوميدية جاهزة للمشاهدة." }
};

const regions = [
  { id: "turkish", title: "مسلسلات تركية", shortLabel: "تركية", description: "حكايات مشوّقة ورومانسية ودراما من تركيا.", terms: ["تركي", "تركية", "turkish", "turkey"], tint: "from-amber-500/75 via-orange-800/40 to-[#130b20]" },
  { id: "foreign", title: "مسلسلات أجنبية", shortLabel: "أجنبية", description: "أفضل المسلسلات العالمية، من الجريمة إلى الخيال العلمي.", terms: ["أجنبي", "أجنبية", "english", "american", "british", "hollywood", "أمريكي", "بريطاني"], tint: "from-cyan-500/70 via-blue-900/45 to-[#0b0d1e]" },
  { id: "arabic", title: "مسلسلات عربية", shortLabel: "عربية", description: "إنتاج عربي متنوع من الدراما والكوميديا والتشويق.", terms: ["عربي", "عربية", "arabic", "مصري", "خليجي"], tint: "from-rose-500/65 via-red-950/45 to-[#170b17]" },
  { id: "korean", title: "مسلسلات كورية", shortLabel: "كورية", description: "دراما كورية آسرة وقصص قريبة من القلب.", terms: ["كوري", "كورية", "korean", "korea"], tint: "from-pink-500/70 via-fuchsia-950/45 to-[#170918]" },
  { id: "indian", title: "مسلسلات هندية", shortLabel: "هندية", description: "دراما وحكايات طويلة من شبه القارة الهندية.", terms: ["هندي", "هندية", "indian", "india", "bollywood"], tint: "from-orange-500/75 via-red-950/45 to-[#1b0c12]" },
  { id: "spanish", title: "مسلسلات إسبانية", shortLabel: "إسبانية", description: "غموض وإثارة ودراما إسبانية مختارة.", terms: ["إسباني", "إسبانية", "spanish", "spain"], tint: "from-violet-500/75 via-indigo-950/45 to-[#100b1d]" }
];

const genres = ["أكشن", "مغامرة", "دراما", "كوميديا", "رومانسي", "جريمة", "غموض", "إثارة", "تاريخي", "خيال علمي", "رعب", "فانتازيا"];
const dateGroups = [
  { id: "new", title: "الأحدث إصداراً", shortLabel: "2024 فما بعد", description: "أحدث الإضافات والإصدارات في المكتبة.", matches: (item) => item.year >= 2024, tint: "from-fuchsia-500/75 via-purple-900/45 to-[#100b1e]" },
  { id: "2020s", title: "مسلسلات 2020 - 2023", shortLabel: "2020 - 2023", description: "أعمال حديثة تستحق المتابعة.", matches: (item) => item.year >= 2020 && item.year <= 2023, tint: "from-sky-500/75 via-blue-950/45 to-[#08111d]" },
  { id: "2010s", title: "مسلسلات العقد السابق", shortLabel: "2010 - 2019", description: "أعمال صنعت ذائقة عقد كامل.", matches: (item) => item.year >= 2010 && item.year < 2020, tint: "from-emerald-500/70 via-teal-950/45 to-[#081714]" },
  { id: "classics", title: "كلاسيكيات خالدة", shortLabel: "قبل 2010", description: "حكايات لا تفقد سحرها مع الوقت.", matches: (item) => item.year > 0 && item.year < 2010, tint: "from-yellow-500/70 via-amber-950/45 to-[#171207]" }
];

function genreGroup(genre) {
  return { id: `genre-${genre}`, title: `مسلسلات ${genre}`, shortLabel: genre, description: `مجموعة مختارة من مسلسلات ${genre}.`, terms: [genre], tint: "from-purple-500/75 via-fuchsia-950/45 to-[#12091a]" };
}

function containsTerms(item, terms = []) {
  const searchable = (item.genres || []).join(" ").toLocaleLowerCase();
  return terms.some((term) => searchable.includes(term.toLocaleLowerCase()));
}

function collectionMatches(item, collection) {
  if (!collection) return true;
  return collection.matches ? collection.matches(item) : containsTerms(item, collection.terms);
}

export default function CategoryPage({ selectedCategory, onOpenMedia }) {
  const [items, setItems] = useState([]);
  const [totalCount, setTotalCount] = useState(0);
  const [activeSort, setActiveSort] = useState("latest");
  const [browseMode, setBrowseMode] = useState("region");
  const [activeCollection, setActiveCollection] = useState(null);
  const [isLoading, setIsLoading] = useState(true);

  const meta = categoryTitles[selectedCategory] || { titleAr: selectedCategory, label: "عمل", description: "تصفح محتوى مكتبتك المحلية." };
  const isSeriesHub = selectedCategory === "series" && !activeCollection;

  useEffect(() => {
    setActiveCollection(null);
    setBrowseMode("region");
  }, [selectedCategory]);

  useEffect(() => {
    let alive = true;
    setIsLoading(true);
    getMediaList({ category: selectedCategory, sort: activeSort, limit: 100 })
      .then((data) => {
        if (!alive) return;
        const transformed = (data.items || []).map((item) => ({
          id: item.id, titleAr: item.title_ar || item.title_en, titleEn: item.title_en, type: item.type,
          plot: item.plot_ar || item.plot_en || "عمل مميز متاح في مكتبة NEXORA المحلية.", year: item.release_year || 2023,
          rating: item.rating || 8.5, posterPath: item.poster_path, bannerPath: item.banner_path,
          categorySlug: item.category_slug || selectedCategory, fileCount: item.file_count || 1, genres: item.genres || []
        }));
        setItems(transformed);
        setTotalCount(data.total || transformed.length);
      })
      .catch(() => alive && (setItems([]), setTotalCount(0)))
      .finally(() => alive && setIsLoading(false));
    return () => { alive = false; };
  }, [selectedCategory, activeSort]);

  const collections = useMemo(() => {
    if (browseMode === "genre") return genres.map(genreGroup);
    if (browseMode === "date") return dateGroups;
    return regions;
  }, [browseMode]);

  const visibleItems = useMemo(() => items.filter((item) => collectionMatches(item, activeCollection)), [items, activeCollection]);
  const heroPath = visibleItems.find((item) => item.bannerPath)?.bannerPath || visibleItems[0]?.posterPath;
  const heroImage = heroPath ? resolveAPIURL(heroPath) : "/images/aot_banner_detail.png";

  function openCollection(collection) {
    setActiveCollection(collection);
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  if (isLoading) return <div className="flex h-64 items-center justify-center"><div className="h-8 w-8 animate-spin rounded-full border-2 border-fuchsia-500 border-t-transparent" /></div>;

  if (isSeriesHub) {
    return (
      <div className="space-y-6 text-right" dir="rtl">
        <section className="relative overflow-hidden rounded-3xl border border-white/10 bg-[radial-gradient(circle_at_12%_0%,rgba(168,85,247,.27),transparent_30%),linear-gradient(120deg,#140d25,#090a16_56%,#0a172b)] p-6 shadow-panel sm:p-9">
          <div className="absolute -left-20 top-0 h-64 w-64 rounded-full bg-cyan-500/10 blur-3xl" />
          <div className="relative flex flex-col gap-7">
            <div>
              <p className="mb-2 text-xs font-bold text-fuchsia-300">NEXORA · مساحة المسلسلات</p>
              <h1 className="text-3xl font-black text-white sm:text-4xl">اكتشف المسلسلات بطريقتك</h1>
              <p className="mt-2 max-w-xl text-sm leading-7 text-white/55">ابدأ بالبلد، بالنوع، أو بالفترة الزمنية. كل مجموعة تفتح لك صفحة مشاهدة متكاملة بتصميم سينمائي.</p>
            </div>
            <div className="flex flex-wrap gap-2">
              {[["region", "حسب البلد"], ["genre", "حسب التصنيف"], ["date", "حسب التاريخ"]].map(([id, label]) => (
                <button key={id} type="button" onClick={() => setBrowseMode(id)} className={`rounded-xl px-4 py-2.5 text-xs font-black transition ${browseMode === id ? "bg-gradient-to-l from-fuchsia-600 to-purple-700 text-white shadow-lg shadow-purple-950/45" : "border border-white/10 bg-black/25 text-white/60 hover:bg-white/10 hover:text-white"}`}>{label}</button>
              ))}
            </div>
          </div>
        </section>

        <section>
          <div className="mb-4 flex items-center justify-between"><div><h2 className="text-xl font-black text-white">{browseMode === "region" ? "المسلسلات حسب البلد" : browseMode === "genre" ? "المسلسلات حسب التصنيف" : "المسلسلات حسب التاريخ"}</h2><p className="mt-1 text-xs text-white/45">اختر مجموعة للانتقال إلى صفحة المسلسلات الخاصة بها.</p></div><span className="rounded-full bg-white/5 px-3 py-1 text-xs font-bold text-white/50">{totalCount} مسلسل</span></div>
          <div className="grid grid-cols-2 gap-4 md:grid-cols-3 xl:grid-cols-4">
            {collections.map((collection, index) => {
              const count = items.filter((item) => collectionMatches(item, collection)).length;
              const cardPath = items.find((item) => collectionMatches(item, collection))?.bannerPath || items.find((item) => collectionMatches(item, collection))?.posterPath;
              const cardImage = cardPath ? resolveAPIURL(cardPath) : "/images/aot_banner_detail.png";
              return <button key={collection.id} type="button" onClick={() => openCollection(collection)} className="group relative aspect-square overflow-hidden rounded-2xl border border-white/10 bg-[#0b0a15] text-right shadow-lg transition duration-300 hover:-translate-y-1 hover:border-fuchsia-400/60 hover:shadow-fuchsia-950/40">
                <img src={cardImage} alt="" className="absolute inset-0 h-full w-full object-cover opacity-45 transition duration-500 group-hover:scale-110 group-hover:opacity-65" />
                <div className={`absolute inset-0 bg-gradient-to-t ${collection.tint}`} /><div className="absolute inset-0 bg-gradient-to-b from-black/5 via-transparent to-black/75" />
                <div className="relative flex h-full flex-col justify-between p-4 sm:p-5"><span className="flex h-9 w-9 items-center justify-center rounded-xl border border-white/15 bg-black/25 text-fuchsia-100"><Icon name={browseMode === "genre" ? "spark" : browseMode === "date" ? "book" : "tv"} className="h-4 w-4" /></span><div><p className="text-base font-black text-white sm:text-lg">{collection.title}</p><p className="mt-1 text-[11px] leading-5 text-white/65">{collection.description}</p><div className="mt-3 flex items-center justify-between text-xs font-bold text-white/80"><span>{count} مسلسل</span><span className="transition-transform group-hover:-translate-x-1">‹</span></div></div></div>
              </button>;
            })}
          </div>
        </section>
      </div>
    );
  }

  const title = activeCollection?.title || meta.titleAr;
  return (
    <div className="space-y-6 text-right" dir="rtl">
      <section className="relative min-h-[270px] overflow-hidden rounded-3xl border border-white/10 bg-[#0b0914] shadow-panel sm:min-h-[320px]">
        <img src={heroImage} alt="" className="absolute inset-0 h-full w-full object-cover opacity-55" />
        <div className={`absolute inset-0 bg-gradient-to-l ${activeCollection?.tint || "from-fuchsia-700/65 via-[#0b0914]/90 to-[#0b0914]/75"}`} /><div className="absolute inset-0 bg-gradient-to-t from-[#090811] via-[#090811]/35 to-transparent" />
        <div className="relative flex min-h-[270px] flex-col justify-end p-5 sm:min-h-[320px] sm:p-8">
          {selectedCategory === "series" && <button type="button" onClick={() => setActiveCollection(null)} className="absolute right-5 top-5 flex items-center gap-1 rounded-full border border-white/15 bg-black/30 px-3 py-2 text-xs font-bold text-white/80 backdrop-blur-md transition hover:bg-black/55 sm:right-8 sm:top-7">العودة للمجموعات <Icon name="arrowRight" className="h-3.5 w-3.5" /></button>}
          <p className="mb-2 text-xs font-black text-fuchsia-200">NEXORA · مجموعة مختارة</p><h1 className="text-3xl font-black text-white sm:text-5xl">{title}</h1><p className="mt-2 max-w-xl text-sm leading-7 text-white/70">{activeCollection?.description || meta.description}</p>
          <div className="mt-5 flex flex-wrap gap-2"><span className="rounded-xl border border-white/15 bg-black/25 px-3 py-2 text-xs font-bold text-white">{visibleItems.length} {meta.label}</span><span className="rounded-xl border border-white/15 bg-black/25 px-3 py-2 text-xs font-bold text-white/75">مكتبة محلية · جودة عالية</span></div>
        </div>
      </section>

      <div className="flex flex-col gap-4 rounded-2xl border border-white/10 bg-[#0b0a15]/80 p-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2 text-sm font-black text-white"><span className="h-2 w-2 rounded-full bg-fuchsia-400 shadow-[0_0_12px_#e879f9]" />كل الأعمال</div>
        <div className="flex flex-wrap gap-2 text-xs font-bold">{[["latest", "الأحدث"], ["rating", "الأعلى تقييماً ★"], ["year", "السنة"], ["title", "أبجدي أ-ي"]].map(([id, label]) => <button key={id} type="button" onClick={() => setActiveSort(id)} className={`rounded-xl px-3.5 py-2 transition ${activeSort === id ? "bg-gradient-to-l from-fuchsia-600 to-purple-700 text-white" : "border border-white/10 bg-white/[.03] text-white/55 hover:bg-white/10 hover:text-white"}`}>{label}</button>)}</div>
      </div>

      {visibleItems.length === 0 ? <div className="rounded-3xl border border-dashed border-white/15 bg-black/25 p-12 text-center"><span className="text-4xl">🎬</span><p className="mt-3 text-sm font-bold text-white">لا توجد أعمال في {title} بعد</p><p className="mt-1 text-xs text-white/45">أضف وسم البلد أو التصنيف للعمل من لوحة الإدارة ليظهر هنا.</p></div> : <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">{visibleItems.map((item, index) => <MediaCard key={item.id} item={item} onOpen={onOpenMedia} index={index} />)}</div>}
    </div>
  );
}
