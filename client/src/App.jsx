import { AnimatePresence, motion } from "framer-motion";
import { useDeferredValue, useEffect, useState, startTransition } from "react";
import Sidebar from "./components/Sidebar.jsx";
import TopBar from "./components/TopBar.jsx";
import DashboardPage from "./pages/DashboardPage.jsx";
import CategoryPage from "./pages/CategoryPage.jsx";
import MediaDetailsPage from "./pages/MediaDetailsPage.jsx";
import AdminPage from "./pages/AdminPage.jsx";
import { categorySeed, findMockMedia, getCategoryMeta, mockLibrary, navigationItems } from "./data/library.js";
import { getCategories, getHealth, searchLibrary, syncIndex } from "./lib/api.js";

const fallbackCategories = categorySeed;

export default function App() {
  const [activeView, setActiveView] = useState("dashboard");
  const [selectedCategory, setSelectedCategory] = useState("anime");
  const [selectedMedia, setSelectedMedia] = useState(mockLibrary[0]);
  const [searchQuery, setSearchQuery] = useState("");
  const deferredQuery = useDeferredValue(searchQuery);
  const [health, setHealth] = useState(null);
  const [categories, setCategories] = useState(fallbackCategories);
  const [searchResults, setSearchResults] = useState(mockLibrary);
  const [isSearching, setIsSearching] = useState(false);
  const [syncStatus, setSyncStatus] = useState("جاهز");

  useEffect(() => {
    let alive = true;
    getHealth()
      .then((payload) => {
        if (alive) {
          setHealth(payload);
        }
      })
      .catch(() => {
        if (alive) {
          setHealth({ ok: false, database: { databaseOk: false } });
        }
      });

    getCategories()
      .then((payload) => {
        if (!alive) {
          return;
        }
        const transformed = (payload.categories || []).map((category) => {
          const meta = getCategoryMeta(category.slug);
          return {
            slug: category.slug,
            titleEn: category.name_en || meta.titleEn,
            titleAr: category.name_ar || meta.titleAr,
            count: category.file_count ?? category.media_count ?? meta.count ?? 0,
            description: meta.description,
            accent: meta.accent
          };
        });
        if (transformed.length > 0) {
          setCategories(transformed);
        }
      })
      .catch(() => {
        if (alive) {
          setCategories(fallbackCategories);
        }
      });

    return () => {
      alive = false;
    };
  }, []);

  useEffect(() => {
    let alive = true;
    const filter = selectedCategory && activeView === "categories" ? selectedCategory : "";
    setIsSearching(true);

    searchLibrary(deferredQuery, {
      limit: activeView === "dashboard" ? 6 : 12,
      category: filter || undefined
    })
      .then((payload) => {
        if (!alive) {
          return;
        }
        const hits = payload.hits?.length ? payload.hits : mockLibrary.filter((item) => {
          const term = deferredQuery.trim().toLowerCase();
          if (!term) {
            return true;
          }
          return [item.titleEn, item.titleAr, item.type, item.categorySlug]
            .join(" ")
            .toLowerCase()
            .includes(term);
        });

        setSearchResults(
          hits.map((hit) =>
            normalizeResult(hit)
          )
        );
      })
      .catch(() => {
        if (alive) {
          const term = deferredQuery.trim().toLowerCase();
          const fallback = mockLibrary.filter((item) => {
            if (!term) {
              return true;
            }
            return [item.titleEn, item.titleAr, item.type, item.categorySlug]
              .join(" ")
              .toLowerCase()
              .includes(term);
          });
          setSearchResults(fallback);
        }
      })
      .finally(() => {
        if (alive) {
          setIsSearching(false);
        }
      });

    return () => {
      alive = false;
    };
  }, [activeView, deferredQuery, selectedCategory]);

  const featured = searchResults.slice(0, 4);
  const currentMedia = selectedMedia || findMockMedia(1);

  function navigate(view) {
    startTransition(() => {
      setActiveView(view);
    });
  }

  function openMedia(item) {
    setSelectedMedia(item);
    navigate("details");
  }

  function openCategory(slug) {
    setSelectedCategory(slug);
    navigate("categories");
  }

  async function handleSyncIndex() {
    setSyncStatus("جارٍ المزامنة");
    try {
      await syncIndex(1000);
      setSyncStatus("تمت المزامنة");
    } catch {
      setSyncStatus("غير متصل");
    }
    window.setTimeout(() => setSyncStatus("جاهز"), 3000);
  }

  return (
    <main className="relative min-h-screen overflow-hidden bg-charcoal text-white">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(90,50,244,0.25),transparent_28%),radial-gradient(circle_at_top_right,rgba(25,183,255,0.22),transparent_28%),radial-gradient(circle_at_bottom_right,rgba(236,72,153,0.15),transparent_25%)]" />
      <div className="pointer-events-none absolute inset-0 opacity-[0.12] [background-image:linear-gradient(rgba(255,255,255,0.12)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.12)_1px,transparent_1px)] [background-size:64px_64px]" />

      <div className="relative mx-auto flex min-h-screen max-w-[1700px] gap-6 p-4 md:p-6">
        <div className="hidden w-[19rem] xl:block">
          <Sidebar
            activeView={activeView}
            categories={categories}
            health={health}
            onNavigate={navigate}
            onOpenCategory={openCategory}
          />
        </div>

        <div className="flex min-h-screen flex-1 flex-col gap-6">
          <TopBar
            searchQuery={searchQuery}
            onSearchChange={setSearchQuery}
            onSyncIndex={handleSyncIndex}
            onOpenView={navigate}
            syncStatus={syncStatus}
            categoriesCount={categories.length}
            isSearching={isSearching}
            backendState={health === null ? "loading" : health?.ok ? "online" : "offline"}
          />

          <div className="grid gap-3 xl:hidden">
            <div className="rounded-[1.5rem] border border-white/10 bg-white/[0.04] p-3 shadow-panel backdrop-blur-2xl">
              <div className="flex gap-2 overflow-x-auto">
                {navigationItems.map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => navigate(item.id)}
                    className={`shrink-0 rounded-full border px-4 py-2 text-sm font-semibold transition ${
                      activeView === item.id
                        ? "border-electric/30 bg-electric/12 text-electric"
                        : "border-white/10 bg-black/20 text-white/72"
                    }`}
                  >
                    {item.label}
                  </button>
                ))}
              </div>
            </div>

            <div className="rounded-[1.5rem] border border-white/10 bg-white/[0.04] p-3 shadow-panel backdrop-blur-2xl">
              <div className="flex gap-2 overflow-x-auto">
                {categories.slice(0, 5).map((category) => (
                  <button
                    key={category.slug}
                    type="button"
                    onClick={() => openCategory(category.slug)}
                    className={`shrink-0 rounded-full border px-4 py-2 text-sm font-semibold transition ${
                      selectedCategory === category.slug
                        ? "border-electric/30 bg-electric/12 text-electric"
                        : "border-white/10 bg-black/20 text-white/72"
                    }`}
                    >
                    {category.titleAr}
                  </button>
                ))}
              </div>
            </div>
          </div>

          <div className="flex-1">
            <AnimatePresence mode="wait">
              <motion.div
                key={`${activeView}-${selectedCategory}`}
                initial={{ opacity: 0, y: 16 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -16 }}
                transition={{ duration: 0.28 }}
              >
                {activeView === "dashboard" && (
                  <DashboardPage
                    categories={categories}
                    featured={featured}
                    searchResults={searchResults}
                    health={health}
                    onOpenMedia={openMedia}
                    onOpenCategory={openCategory}
                    onSyncIndex={handleSyncIndex}
                    syncStatus={syncStatus}
                  />
                )}

                {activeView === "categories" && (
                  <CategoryPage
                    categories={categories}
                    selectedCategory={selectedCategory}
                    searchResults={searchResults}
                    onOpenCategory={openCategory}
                    onOpenMedia={openMedia}
                    searchQuery={deferredQuery}
                  />
                )}

                {activeView === "details" && (
                  <MediaDetailsPage media={currentMedia} onOpenCategory={openCategory} />
                )}

                {activeView === "admin" && (
                  <AdminPage health={health} onSyncIndex={handleSyncIndex} />
                )}
              </motion.div>
            </AnimatePresence>
          </div>
        </div>
      </div>
    </main>
  );
}

function normalizeResult(hit) {
  const base = mockLibrary.find((item) => String(item.id) === String(hit.id)) || mockLibrary[0];
  return {
    ...base,
    id: hit.id ?? base.id,
    titleEn: hit.title_en || hit.titleEn || base.titleEn,
    titleAr: hit.title_ar || hit.titleAr || base.titleAr,
    type: hit.type || base.type,
    plot: hit.plot_ar || hit.plot_en || base.plot,
    year: hit.release_year ?? base.year,
    rating: hit.rating ?? base.rating,
    resolution: hit.resolution || base.resolution,
    posterPath: hit.poster_path || hit.posterPath || base.posterPath,
    bannerPath: hit.banner_path || hit.bannerPath || base.bannerPath,
    categorySlug: hit.category_slug || base.categorySlug,
    fileCount: hit.file_count ?? base.fileCount,
    gradient: base.gradient,
    seasons: base.seasons,
    highlights: base.highlights,
    duration: base.duration,
    seasonLabel: base.seasonLabel,
    episodeLabel: base.episodeLabel
  };
}
