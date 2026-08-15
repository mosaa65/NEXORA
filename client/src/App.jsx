import { AnimatePresence, motion } from "framer-motion";
import { useDeferredValue, useEffect, useState, startTransition } from "react";
import Sidebar from "./components/Sidebar.jsx";
import TopBar from "./components/TopBar.jsx";
import DashboardPage from "./pages/DashboardPage.jsx";
import CategoryPage from "./pages/CategoryPage.jsx";
import MediaDetailsPage from "./pages/MediaDetailsPage.jsx";
import AdminPage from "./pages/AdminPage.jsx";
import VideoPlayer from "./components/VideoPlayer.jsx";
import Icon from "./components/Icon.jsx";
import { categorySeed, detailEpisodes, findMockMedia, getCategoryMeta, mockLibrary, navigationItems } from "./data/library.js";
import { getCategories, getHealth, getMediaFiles, searchLibrary, syncIndex, resolveAPIURL } from "./lib/api.js";

const fallbackCategories = categorySeed;

export default function App() {
  const [activeView, setActiveView] = useState("dashboard");
  const [selectedCategory, setSelectedCategory] = useState("anime");
  const [selectedMedia, setSelectedMedia] = useState(mockLibrary[0]);
  const [playingMedia, setPlayingMedia] = useState(null);
  const [playingEpisode, setPlayingEpisode] = useState(detailEpisodes[0]);
  const [playingVideoFile, setPlayingVideoFile] = useState(null);
  const [playingVideoFiles, setPlayingVideoFiles] = useState([]);
  const [isPlaying, setIsPlaying] = useState(true);
  const [currentTimeSec, setCurrentTimeSec] = useState(765); // 12:45
  const [searchQuery, setSearchQuery] = useState("");
  const deferredQuery = useDeferredValue(searchQuery);
  const [health, setHealth] = useState(null);
  const [categories, setCategories] = useState(fallbackCategories);
  const [searchResults, setSearchResults] = useState(mockLibrary);
  const [isSearching, setIsSearching] = useState(false);
  const [syncStatus, setSyncStatus] = useState("جاهز");
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const [adminAnchor, setAdminAnchor] = useState("admin-overview");

  useEffect(() => {
    let interval;
    if (playingMedia && isPlaying && !playingVideoFile) {
      interval = setInterval(() => {
        setCurrentTimeSec((prev) => (prev >= 1450 ? 0 : prev + 1));
      }, 1000);
    }
    return () => clearInterval(interval);
  }, [playingMedia, isPlaying, playingVideoFile]);

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
    const filter = (activeView === "anime" || activeView === "movies" || activeView === "series" || activeView === "categories")
      ? (activeView === "movies" ? "movies" : activeView === "series" ? "series" : selectedCategory)
      : "";

    setIsSearching(true);

    searchLibrary(deferredQuery, {
      limit: 24,
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

        setSearchResults(hits.map((hit) => normalizeResult(hit)));
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

  const currentMedia = selectedMedia || findMockMedia(1);

  function navigate(view) {
    startTransition(() => {
      if (view === "admin") {
        setAdminAnchor("admin-overview");
      } else if (["anime", "movies", "series", "kids", "documentaries", "plays"].includes(view)) {
        setSelectedCategory(view);
      }
      setActiveView(view);
      setIsSidebarOpen(false);
    });
  }

  function openMedia(item) {
    setSelectedMedia(item);
    navigate("details");
  }

  async function quickPlayMedia(item, episode) {
    setSelectedMedia(item);
    setIsPlaying(true);
    setCurrentTimeSec(0);
    try {
      const payload = await getMediaFiles(item.id);
      const files = payload.files || [];
      setPlayingVideoFiles(files);
      if (episode && episode.id) {
        setPlayingVideoFile(episode);
      } else {
        setPlayingVideoFile(files[0] || null);
      }
      setPlayingMedia(item);
    } catch {
      setPlayingVideoFiles([]);
      setPlayingVideoFile(episode || null);
      setPlayingMedia(item);
    }
  }

  function openCategory(slug) {
    setSelectedCategory(slug);
    navigate(slug);
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

  function formatTime(seconds) {
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  }

  return (
    <main className="relative min-h-screen bg-[#08070E] text-white selection:bg-fuchsia-600/40 selection:text-white font-sans">
      {/* Background Micro Glow */}
      <div className="pointer-events-none fixed inset-0 z-0 bg-[radial-gradient(circle_at_20%_0%,rgba(147,51,234,0.18),transparent_40%),radial-gradient(circle_at_80%_0%,rgba(217,70,239,0.14),transparent_35%)]" />

      <div className="relative z-10 mx-auto max-w-[1920px] p-3 sm:p-5 lg:p-8">
        {/* Render TopBar only on views that do not have their own integrated header */}
        {activeView !== "dashboard" && activeView !== "details" && (
          <TopBar
            searchQuery={searchQuery}
            onSearchChange={setSearchQuery}
            searchResults={searchResults}
            onOpenMedia={openMedia}
            onQuickPlay={quickPlayMedia}
            onToggleSidebar={() => setIsSidebarOpen(!isSidebarOpen)}
          />
        )}

        {/* Main Content Layout */}
        <div className="flex flex-col gap-6 lg:flex-row">
          {/* Sidebar */}
          <Sidebar
            isOpen={isSidebarOpen}
            onClose={() => setIsSidebarOpen(false)}
            activeView={activeView}
            onNavigate={navigate}
            activeAdminAnchor={adminAnchor}
            onAdminNavigate={(anchor) => {
              setAdminAnchor(anchor);
              startTransition(() => {
                setActiveView("admin");
                setIsSidebarOpen(false);
              });
            }}
          />

          {/* Active View Container */}
          <div className="min-w-0 flex-1">
            <AnimatePresence mode="wait">
              <motion.div
                key={`${activeView}-${selectedCategory}`}
                initial={{ opacity: 0, y: 12 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -12 }}
                transition={{ duration: 0.25 }}
              >
                {activeView === "dashboard" && (
                  <DashboardPage
                    searchQuery={searchQuery}
                    onSearchChange={setSearchQuery}
                    searchResults={searchResults}
                    onOpenMedia={openMedia}
                    onQuickPlay={quickPlayMedia}
                    onToggleSidebar={() => setIsSidebarOpen(!isSidebarOpen)}
                  />
                )}

                {(activeView === "categories" || ["anime", "movies", "series", "kids", "documentaries", "plays"].includes(activeView)) && (
                  <CategoryPage
                    selectedCategory={selectedCategory}
                    searchResults={searchResults}
                    onOpenMedia={openMedia}
                    onQuickPlay={quickPlayMedia}
                  />
                )}

                {activeView === "details" && (
                  <MediaDetailsPage
                    media={currentMedia}
                    searchQuery={searchQuery}
                    onSearchChange={setSearchQuery}
                    onOpenCategory={openCategory}
                    onQuickPlay={quickPlayMedia}
                  />
                )}

                {/* Completely Isolated Admin Console View */}
                {activeView === "admin" && (
                  <AdminPage health={health} onSyncIndex={handleSyncIndex} activeAnchor={adminAnchor} />
                )}
              </motion.div>
            </AnimatePresence>
          </div>
        </div>
      </div>

      {/* Video Player & Episodes Overlay (Exact Match to Bottom-Right Screenshot View) */}
      {playingMedia && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/95 p-3 sm:p-6 backdrop-blur-2xl text-right">
          <motion.div
            initial={{ opacity: 0, scale: 0.96 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.96 }}
            className="relative flex h-[92vh] w-full max-w-7xl flex-col overflow-hidden rounded-[2rem] border border-white/15 bg-[#0A0914] p-4 sm:p-6 shadow-2xl"
          >
            {/* Header Row */}
            <div className="mb-4 flex items-center justify-between border-b border-white/10 pb-3">
              <button
                type="button"
                onClick={() => setPlayingMedia(null)}
                className="flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-xs font-bold text-white transition hover:bg-white/20"
              >
                العودة ‹
              </button>

              <div className="text-right">
                <p className="text-xs sm:text-sm font-bold text-fuchsia-300">
                  {playingMedia.titleAr} - {playingMedia.seasonLabel || "الموسم الثالث"} - {playingEpisode.title}
                </p>
              </div>

              <div className="flex items-center gap-3">
                <div className="text-right hidden sm:block">
                  <h3 className="text-sm font-black text-white">مكتبتي</h3>
                  <p className="text-[10px] text-white/40">نظام إدارة الوسائط</p>
                </div>
                <div className="flex h-9 w-9 items-center justify-center text-fuchsia-400">
                  <svg className="h-6 w-6 stroke-current" fill="none" viewBox="0 0 24 24" strokeWidth="2.2">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 3L2 21h20L12 3z" />
                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 9l-4 7h8l-4-7z" />
                  </svg>
                </div>
              </div>
            </div>

            {/* Content Body: Player (Left) + Episodes Sidebar (Right) */}
            <div className="grid flex-1 gap-6 overflow-hidden lg:grid-cols-[1.35fr_0.65fr]">
              {/* Left Video Player Container */}
              <div className="relative flex flex-col overflow-hidden rounded-2xl border border-white/10 bg-black shadow-2xl">
                {playingVideoFile?.stream_url ? (
                  <VideoPlayer
                    src={resolveAPIURL(playingVideoFile.stream_url)}
                    title={playingMedia.titleAr}
                    poster={playingMedia.posterPath ? resolveAPIURL(playingMedia.posterPath) : undefined}
                  />
                ) : (
                  <div className="relative flex h-full w-full flex-col justify-between overflow-hidden bg-black/90 p-5">
                    {/* Dark Character Artwork Backdrop */}
                    <div
                      className="absolute inset-0 bg-cover bg-center opacity-70 transition-all duration-700"
                      style={{ backgroundImage: `url('${playingMedia.posterPath || "/images/jujutsu_kaisen_poster.png"}')` }}
                    />
                    <div className="absolute inset-0 bg-gradient-to-t from-black via-black/30 to-black/20" />

                    {/* Centered Interactive Play/Pause Button */}
                    <div className="relative z-10 my-auto flex flex-col items-center justify-center text-center">
                      <button
                        type="button"
                        onClick={() => setIsPlaying(!isPlaying)}
                        className="flex h-20 w-20 items-center justify-center rounded-full bg-gradient-to-tr from-fuchsia-600 to-purple-600 text-white shadow-2xl shadow-purple-900/80 transition hover:scale-110"
                      >
                        <Icon name={isPlaying ? "play" : "play"} className="h-8 w-8 text-white fill-current" />
                      </button>
                      <p className="mt-4 text-lg font-black text-white">{playingMedia.titleAr} · {playingEpisode.title}</p>
                      <p className="mt-1 text-xs font-bold text-white/60">بث شبكي محلي LAN · دقة 1080p · صوت محيطي</p>
                    </div>

                    {/* Bottom Custom Player Controls Bar (Matching Reference Image) */}
                    <div className="relative z-10 space-y-2 rounded-2xl border border-white/10 bg-black/70 p-3.5 backdrop-blur-md">
                      {/* Scrubber Line */}
                      <div className="relative h-1.5 w-full cursor-pointer rounded-full bg-white/20">
                        <div
                          className="h-1.5 rounded-full bg-gradient-to-r from-purple-600 to-fuchsia-500"
                          style={{ width: `${(currentTimeSec / 1450) * 100}%` }}
                        />
                        <div
                          className="absolute top-1/2 h-3.5 w-3.5 -translate-x-1/2 -translate-y-1/2 rounded-full bg-white shadow-md shadow-fuchsia-500/50"
                          style={{ left: `${(currentTimeSec / 1450) * 100}%` }}
                        />
                      </div>

                      {/* Timestamps & Control Action Buttons */}
                      <div className="flex items-center justify-between pt-1 text-xs text-white/80">
                        {/* Time Indicator */}
                        <div className="flex items-center gap-2 font-mono text-xs font-bold text-white/70">
                          <span>{formatTime(currentTimeSec)}</span>
                          <span>/</span>
                          <span>24:10</span>
                        </div>

                        {/* Control Buttons */}
                        <div className="flex items-center gap-4">
                          <button type="button" className="hover:text-white" title="كتم الصوت">
                            <Icon name="smile" className="h-4 w-4" />
                          </button>
                          <button
                            type="button"
                            onClick={() => setCurrentTimeSec((prev) => Math.max(0, prev - 10))}
                            className="hover:text-white"
                            title="تراجع 10 ثوان"
                          >
                            ⏮
                          </button>

                          <button
                            type="button"
                            onClick={() => setIsPlaying(!isPlaying)}
                            className="flex h-9 w-9 items-center justify-center rounded-full bg-gradient-to-r from-fuchsia-600 to-purple-600 text-white shadow-md hover:scale-105"
                          >
                            {isPlaying ? "⏸" : "▶"}
                          </button>

                          <button
                            type="button"
                            onClick={() => setCurrentTimeSec((prev) => Math.min(1450, prev + 10))}
                            className="hover:text-white"
                            title="تقديم 10 ثوان"
                          >
                            ⏭
                          </button>

                          <button type="button" className="hover:text-white" title="الإعدادات">
                            <Icon name="settings" className="h-4 w-4" />
                          </button>
                          <button type="button" className="hover:text-white" title="ملء الشاشة">
                            ⛶
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                )}
              </div>

              {/* Right Episode List Sidebar */}
              <div className="flex flex-col overflow-hidden rounded-2xl border border-white/10 bg-[#0D0E18] p-4 text-right">
                {/* Season Header */}
                <div className="mb-3 flex items-center justify-between border-b border-white/10 pb-3">
                  <span className="rounded-md border border-white/10 bg-white/5 px-2.5 py-1 text-xs font-bold text-white/70">
                    22 حلقة
                  </span>
                  <div className="flex items-center gap-1.5 cursor-pointer">
                    <span className="text-xs text-white/40">▼</span>
                    <h4 className="text-sm font-black text-white">الموسم الثالث</h4>
                  </div>
                </div>

                {/* Episodes Scroll List */}
                <div className="flex-1 space-y-2.5 overflow-y-auto pr-1">
                  {detailEpisodes.map((ep) => {
                    const isSelected = ep.number === playingEpisode.number;
                    return (
                      <button
                        key={ep.number}
                        type="button"
                        onClick={() => {
                          setPlayingEpisode(ep);
                          setIsPlaying(true);
                          setCurrentTimeSec(0);
                        }}
                        className={`flex w-full items-center justify-between rounded-xl p-3 text-right transition ${
                          isSelected
                            ? "bg-gradient-to-r from-purple-900/90 via-fuchsia-900/90 to-purple-900/90 border border-fuchsia-500/50 text-white shadow-lg shadow-purple-900/40"
                            : "border border-white/5 bg-white/[0.02] text-white/70 hover:border-white/15 hover:bg-white/5"
                        }`}
                      >
                        <span className="text-xs font-bold text-white/50">{ep.duration}</span>
                        <div className="flex items-center gap-3">
                          <div className="text-right">
                            <p className="text-xs font-bold text-white">{ep.title}</p>
                            <p className="text-[10px] text-white/40">مكان إشارة المعركة</p>
                          </div>
                          <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${
                            isSelected
                              ? "bg-fuchsia-600 text-white shadow-md shadow-fuchsia-900/50"
                              : "bg-white/10 text-white/40"
                          }`}>
                            ▶
                          </div>
                        </div>
                      </button>
                    );
                  })}
                </div>
              </div>
            </div>
          </motion.div>
        </div>
      )}
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
