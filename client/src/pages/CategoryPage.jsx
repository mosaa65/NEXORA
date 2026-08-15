import { useEffect, useState } from "react";
import MediaCard from "../components/MediaCard.jsx";
import { getMediaList } from "../lib/api.js";
import { mockLibrary } from "../data/library.js";

const categoryTitles = {
  anime: { titleAr: "الأنمي", label: "أنمي" },
  movies: { titleAr: "الأفلام", label: "فيلم" },
  series: { titleAr: "المسلسلات", label: "مسلسل" },
  kids: { titleAr: "أطفال وكرتون", label: "عمل" },
  documentaries: { titleAr: "وثائقيات", label: "وثائقي" },
  plays: { titleAr: "مسرحيات", label: "مسرحية" }
};

export default function CategoryPage({
  selectedCategory,
  onOpenMedia,
  onQuickPlay
}) {
  const [items, setItems] = useState([]);
  const [totalCount, setTotalCount] = useState(0);
  const [activeSort, setActiveSort] = useState("latest");
  const [isLoading, setIsLoading] = useState(true);

  const meta = categoryTitles[selectedCategory] || {
    titleAr: selectedCategory,
    label: "عمل"
  };

  useEffect(() => {
    let alive = true;
    setIsLoading(true);

    getMediaList({
      category: selectedCategory,
      sort: activeSort,
      limit: 50
    })
      .then((data) => {
        if (!alive) return;
        if (data.items && data.items.length > 0) {
          const transformed = data.items.map((item) => ({
            id: item.id,
            titleAr: item.title_ar || item.title_en,
            titleEn: item.title_en,
            type: item.type,
            plot: item.plot_ar || item.plot_en || "عمل سينمائي مميز متاح في مكتبة NEXORA المحلية.",
            year: item.release_year || 2023,
            rating: item.rating || 8.5,
            posterPath: item.poster_path,
            bannerPath: item.banner_path,
            categorySlug: item.category_slug || selectedCategory,
            fileCount: item.file_count || 1
          }));
          setItems(transformed);
          setTotalCount(data.total || transformed.length);
        } else {
          setItems([]);
          setTotalCount(0);
        }
      })
      .catch(() => {
        if (!alive) return;
        setItems([]);
        setTotalCount(0);
      })
      .finally(() => {
        if (alive) setIsLoading(false);
      });

    return () => {
      alive = false;
    };
  }, [selectedCategory, activeSort]);

  return (
    <div className="space-y-6 text-right">
      {/* Category Header & Filters Bar */}
      <div className="flex flex-col gap-4 rounded-2xl border border-white/10 bg-[#0A0914] p-4 shadow-panel sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3 text-right">
          <h1 className="text-2xl font-black text-white sm:text-3xl">
            {meta.titleAr}
          </h1>
          <span className="rounded-full border border-purple-500/30 bg-purple-900/30 px-3 py-1 text-xs font-bold text-fuchsia-300">
            {totalCount} {meta.label}
          </span>
        </div>

        {/* Filter Pill Buttons */}
        <div className="flex flex-wrap items-center justify-end gap-2 text-xs font-bold">
          <button
            type="button"
            onClick={() => setActiveSort("latest")}
            className={`rounded-xl px-4 py-2 text-white transition ${
              activeSort === "latest"
                ? "bg-gradient-to-r from-purple-800 to-fuchsia-700 shadow-md shadow-purple-900/40"
                : "border border-white/10 bg-white/5 text-white/70 hover:bg-white/10 hover:text-white"
            }`}
          >
            الأحدث
          </button>
          <button
            type="button"
            onClick={() => setActiveSort("rating")}
            className={`rounded-xl px-4 py-2 text-white transition ${
              activeSort === "rating"
                ? "bg-gradient-to-r from-purple-800 to-fuchsia-700 shadow-md shadow-purple-900/40"
                : "border border-white/10 bg-white/5 text-white/70 hover:bg-white/10 hover:text-white"
            }`}
          >
            الأعلى تقييماً ★
          </button>
          <button
            type="button"
            onClick={() => setActiveSort("year")}
            className={`rounded-xl px-4 py-2 text-white transition ${
              activeSort === "year"
                ? "bg-gradient-to-r from-purple-800 to-fuchsia-700 shadow-md shadow-purple-900/40"
                : "border border-white/10 bg-white/5 text-white/70 hover:bg-white/10 hover:text-white"
            }`}
          >
            السنة 📅
          </button>
          <button
            type="button"
            onClick={() => setActiveSort("title")}
            className={`rounded-xl px-4 py-2 text-white transition ${
              activeSort === "title"
                ? "bg-gradient-to-r from-purple-800 to-fuchsia-700 shadow-md shadow-purple-900/40"
                : "border border-white/10 bg-white/5 text-white/70 hover:bg-white/10 hover:text-white"
            }`}
          >
            أبجدي أ-ي
          </button>
        </div>
      </div>

      {/* Grid of Poster Cards */}
      {isLoading ? (
        <div className="flex h-64 items-center justify-center">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-fuchsia-500 border-t-transparent" />
        </div>
      ) : items.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-2xl border border-white/10 bg-black/40 p-12 text-center text-white/60">
          <span className="text-4xl">🎬</span>
          <p className="mt-3 text-sm font-bold text-white">لا توجد أعمال في هذا القسم بعد</p>
          <p className="mt-1 text-xs text-white/40">قم بإضافة ملفات الوسائط ثم أعد الفهرسة من لوحة الإدارة</p>
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
          {items.map((item, index) => (
            <MediaCard
              key={item.id}
              item={item}
              onOpen={onOpenMedia}
              onQuickPlay={onQuickPlay}
              index={index}
            />
          ))}
        </div>
      )}
    </div>
  );
}
