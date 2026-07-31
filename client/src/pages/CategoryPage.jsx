import GlassCard from "../components/GlassCard.jsx";
import MediaCard from "../components/MediaCard.jsx";

export default function CategoryPage({
  categories,
  selectedCategory,
  searchResults,
  onOpenCategory,
  onOpenMedia,
  searchQuery
}) {
  const active = categories.find((category) => category.slug === selectedCategory) || categories[0];

  return (
    <div className="space-y-6">
      <GlassCard className="p-6">
        <p className="text-xs font-semibold text-electric/80">عرض التصنيفات</p>
        <div className="mt-4 flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div className="text-right">
            <h1 className="text-3xl font-black text-white md:text-5xl">{active?.titleAr || "التصنيفات"}</h1>
            <p className="mt-2 text-sm font-semibold text-white/45">{active?.titleEn || ""}</p>
            <p className="mt-3 max-w-3xl text-base leading-8 text-white/70">
              تصفّح الأقسام المنسّقة، ومرّر البحث الحي إلى Meilisearch، وانتقل مباشرة إلى أي عنوان تريده.
            </p>
          </div>
          <div className="rounded-2xl border border-electric/20 bg-electric/12 px-4 py-3 text-sm font-semibold text-electric">
            {searchQuery ? `جارٍ البحث عن "${searchQuery}"` : "التصفح المباشر مفعّل"}
          </div>
        </div>

        <div className="mt-6 flex flex-wrap gap-3">
          {categories.map((category) => {
            const isActive = active?.slug === category.slug;
            return (
              <button
                key={category.slug}
                type="button"
                onClick={() => onOpenCategory(category.slug)}
                className={`rounded-full border px-4 py-2 text-sm font-medium transition ${
                  isActive
                    ? "border-electric/30 bg-electric/14 text-electric"
                    : "border-white/10 bg-white/[0.05] text-white/70 hover:border-white/20 hover:bg-white/[0.08]"
                }`}
              >
                {category.titleAr}
              </button>
            );
          })}
        </div>
      </GlassCard>

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {categories.map((category) => (
          <GlassCard key={category.slug} className="overflow-hidden p-0">
            <button type="button" onClick={() => onOpenCategory(category.slug)} className="block h-full w-full text-left">
              <div className="h-2" style={{ backgroundImage: category.accent }} />
              <div className="p-5 text-right">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="text-xs font-semibold text-white/35">{category.titleEn}</p>
                    <h3 className="mt-2 text-2xl font-bold text-white">{category.titleAr}</h3>
                  </div>
                  <div className="rounded-2xl border border-white/10 bg-black/20 px-3 py-2 text-right">
                    <p className="text-xl font-bold text-white">{category.count}</p>
                    <p className="text-[11px] font-semibold text-white/35">عنوان</p>
                  </div>
                </div>
                <p className="mt-4 text-sm leading-7 text-white/65">{category.description}</p>
              </div>
            </button>
          </GlassCard>
        ))}
      </section>

      <GlassCard className="p-6">
        <div className="flex items-end justify-between gap-4">
          <div className="text-right">
            <p className="text-xs font-semibold text-white/40">نتائج Meilisearch</p>
            <h2 className="mt-2 text-2xl font-bold text-white">نتائج البحث الممرّرة</h2>
          </div>
          <p className="text-sm text-white/50">{searchResults.length} عنوان</p>
        </div>

        <div className="mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {searchResults.length > 0 ? (
            searchResults.map((item, index) => (
              <MediaCard key={`${item.id}-${index}`} item={item} onOpen={onOpenMedia} index={index} compact />
            ))
          ) : (
            <div className="rounded-3xl border border-white/10 bg-black/20 p-6 text-right text-white/60">
              لا توجد نتائج تطابق البحث الحالي بعد. جرّب كلمات أوسع بالعربية أو الإنجليزية.
            </div>
          )}
        </div>
      </GlassCard>
    </div>
  );
}
