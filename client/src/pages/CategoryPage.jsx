import MediaCard from "../components/MediaCard.jsx";
import { mockLibrary } from "../data/library.js";

export default function CategoryPage({
  selectedCategory,
  onOpenMedia,
  onQuickPlay
}) {
  const animeList = mockLibrary.filter((item) => item.categorySlug === "anime" || item.type === "anime");
  const displayList = animeList.length >= 10 ? animeList : mockLibrary;

  return (
    <div className="space-y-6">
      {/* Anime Catalog Header & Filters Bar */}
      <div className="flex flex-col gap-4 rounded-2xl border border-white/10 bg-[#0A0914] p-4 shadow-panel sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3 text-right">
          <h1 className="text-2xl font-black text-white sm:text-3xl">الأنمي</h1>
          <span className="rounded-full border border-purple-500/30 bg-purple-900/30 px-3 py-1 text-xs font-bold text-fuchsia-300">
            8,350 أنمي
          </span>
        </div>

        {/* Filter Pill Buttons */}
        <div className="flex flex-wrap items-center justify-end gap-2 text-xs font-bold">
          <button
            type="button"
            className="rounded-xl bg-gradient-to-r from-purple-800 to-fuchsia-700 px-4 py-2 text-white shadow-md shadow-purple-900/40"
          >
            الكل
          </button>
          <button
            type="button"
            className="rounded-xl border border-white/10 bg-white/5 px-3.5 py-2 text-white/70 hover:bg-white/10 hover:text-white"
          >
            السنة ∨
          </button>
          <button
            type="button"
            className="rounded-xl border border-white/10 bg-white/5 px-3.5 py-2 text-white/70 hover:bg-white/10 hover:text-white"
          >
            الجودة ∨
          </button>
          <button
            type="button"
            className="rounded-xl border border-white/10 bg-white/5 px-3.5 py-2 text-white/70 hover:bg-white/10 hover:text-white"
          >
            ترتيب حسب ∨
          </button>

          <div className="mr-2 flex items-center rounded-xl border border-white/10 bg-black/40 p-1">
            <button type="button" className="rounded-lg bg-white/10 px-2 py-1 text-white">
              ⊞
            </button>
            <button type="button" className="px-2 py-1 text-white/40 hover:text-white">
              ☰
            </button>
          </div>
        </div>
      </div>

      {/* 5-Column Grid of 10 Anime Poster Cards */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
        {displayList.slice(0, 10).map((item, index) => (
          <MediaCard
            key={item.id}
            item={item}
            onOpen={onOpenMedia}
            onQuickPlay={onQuickPlay}
            index={index}
          />
        ))}
      </div>
    </div>
  );
}
