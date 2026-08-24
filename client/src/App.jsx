import React, { useEffect, useState, useDeferredValue } from "react";
import { HashRouter, Routes, Route, Navigate, useNavigate, useParams } from "react-router-dom";
import CustomerCinemaLayout from "./layouts/CustomerCinemaLayout.jsx";
import AdminPortalLayout from "./layouts/AdminPortalLayout.jsx";
import DashboardPage from "./pages/DashboardPage.jsx";
import CategoryPage from "./pages/CategoryPage.jsx";
import MediaDetailsPage from "./pages/MediaDetailsPage.jsx";
import AdminCategoriesPage from "./pages/admin/AdminCategoriesPage.jsx";
import AdminMediaPage from "./pages/admin/AdminMediaPage.jsx";
import AdminIndexerPage from "./pages/admin/AdminIndexerPage.jsx";
import AdminQualityPage from "./pages/admin/AdminQualityPage.jsx";
import AdminMigrationPage from "./pages/admin/AdminMigrationPage.jsx";
import AdminOverviewPage from "./pages/admin/AdminOverviewPage.jsx";
import TMDBSettingsPage from "./pages/TMDBSettingsPage.jsx";
import AdminLoginPage from "./pages/AdminLoginPage.jsx";
import VideoPlayer from "./components/VideoPlayer.jsx";
import { categorySeed, findMockMedia, getCategoryMeta, mockLibrary } from "./data/library.js";
import { getCategories, getHealth, getMediaDetail, getFileSubtitles, getMediaList, syncIndex, resolveAPIURL } from "./lib/api.js";

// Helper Wrapper for Category View
function CategoryRouteWrapper({ onOpenMedia, onQuickPlay }) {
  const { category = "series" } = useParams();
  return (
    <CategoryPage
      selectedCategory={category}
      onOpenMedia={onOpenMedia}
      onQuickPlay={onQuickPlay}
    />
  );
}

// Helper Wrapper for Media Details View
function MediaDetailsRouteWrapper({ onOpenCategory, onQuickPlay }) {
  const { id } = useParams();
  const [mediaItem, setMediaItem] = useState(null);

  useEffect(() => {
    if (id) {
      const found = mockLibrary.find((m) => String(m.id) === String(id)) || {
        id: parseInt(id),
        titleAr: "هجوم العمالقة: الموسم الأخير",
        titleEn: "Attack on Titan: The Final Season",
        type: "anime",
        plot: "ملحمة إيرين ييغر وفيلق الاستكشاف في صراع البقاء الأخير.",
        year: 2023,
        rating: 9.1,
        posterPath: "/nexora-poster-placeholder.PNG",
        bannerPath: "/nexora-library-backdrop.PNG",
        categorySlug: "anime",
        fileCount: 28,
        genres: ["أكشن", "دراما", "أنمي", "فانتازيا"],
      };
      setMediaItem(found);
    }
  }, [id]);

  if (!mediaItem) return null;

  return (
    <MediaDetailsPage
      media={mediaItem}
      onOpenCategory={onOpenCategory}
      onQuickPlay={onQuickPlay}
    />
  );
}

export default function App() {
  const [health, setHealth] = useState(null);
  const [categories, setCategories] = useState(categorySeed);
  const [searchQuery, setSearchQuery] = useState("");
  const deferredQuery = useDeferredValue(searchQuery);
  const [searchResults, setSearchResults] = useState([]);
  const [isSearching, setIsSearching] = useState(false);

  // Global Real Video Player Modal State
  const [playingMediaState, setPlayingMediaState] = useState(null);

  function handleQuickPlay(item, episodeOrFile) {
    setPlayingMediaState({ media: item, initialFile: episodeOrFile || null });
  }

  useEffect(() => {
    getHealth()
      .then(setHealth)
      .catch(() => setHealth({ ok: false, database: { databaseOk: false } }));

    getCategories()
      .then((payload) => {
        const transformed = (payload.categories || []).map((category) => {
          const meta = getCategoryMeta(category.slug);
          return {
            slug: category.slug,
            titleEn: category.name_en || meta.titleEn,
            titleAr: category.name_ar || meta.titleAr,
            count: category.file_count ?? category.media_count ?? meta.count ?? 0,
            description: meta.description,
            accent: meta.accent,
          };
        });
        if (transformed.length > 0) setCategories(transformed);
      })
      .catch(() => setCategories(categorySeed));
  }, []);

  useEffect(() => {
    if (!deferredQuery.trim()) {
      setSearchResults([]);
      return;
    }

    setIsSearching(true);
    // Catalogue search deliberately goes through PostgreSQL-backed /api/media.
    // Meilisearch remains available for administration, but card data must have
    // the same local offline-first contract as every other catalogue surface.
    getMediaList({ q: deferredQuery, limit: 30 })
      .then((payload) => {
        setSearchResults(payload?.items || []);
      })
      .catch(() => setSearchResults([]))
      .finally(() => setIsSearching(false));
  }, [deferredQuery]);

  async function handleSyncIndex() {
    try {
      await syncIndex(1000);
    } catch {}
  }

  return (
    <HashRouter>
      <Routes>
        {/* ========================================================================= */}
        {/* 1. CUSTOMER CINEMA LOUNGE LAYOUT ROUTES                                    */}
        {/* ========================================================================= */}
        <Route
          path="/"
          element={
            <CustomerCinemaLayout
              health={health}
              categories={categories}
              searchQuery={searchQuery}
              onSearchChange={setSearchQuery}
            />
          }
        >
          {/* Main Dashboard / Home */}
          <Route
            index
            element={
              <DashboardPage
                searchQuery={searchQuery}
                onSearchChange={setSearchQuery}
                searchResults={searchResults}
                onOpenMedia={(item) => (window.location.hash = `#/media/${item.id}`)}
                onQuickPlay={handleQuickPlay}
                onNavigateCategory={(slug) => (window.location.hash = `#/catalog/${slug}`)}
              />
            }
          />

          {/* Category Catalog Route with Origin Hubs */}
          <Route
            path="catalog/:category"
            element={
              <CategoryRouteWrapper
                onOpenMedia={(item) => (window.location.hash = `#/media/${item.id}`)}
                onQuickPlay={handleQuickPlay}
              />
            }
          />

          {/* Favorites Route */}
          <Route
            path="favorites"
            element={
              <CategoryRouteWrapper
                onOpenMedia={(item) => (window.location.hash = `#/media/${item.id}`)}
                onQuickPlay={handleQuickPlay}
              />
            }
          />

          {/* Media Details Route */}
          <Route
            path="media/:id"
            element={
              <MediaDetailsRouteWrapper
                onOpenCategory={(slug) => (window.location.hash = `#/catalog/${slug}`)}
                onQuickPlay={handleQuickPlay}
              />
            }
          />
        </Route>

        {/* ========================================================================= */}
        {/* 2. ADMIN AUTHENTICATION                                                  */}
        {/* ========================================================================= */}
        <Route
          path="/admin/login"
          element={
            <AdminLoginPage
              onLoginSuccess={() => (window.location.hash = "#/admin/categories")}
            />
          }
        />

        {/* ========================================================================= */}
        {/* 3. ISOLATED ADMIN PORTAL ROUTES                                          */}
        {/* ========================================================================= */}
        <Route
          path="/admin"
          element={<AdminPortalLayout health={health} onSyncIndex={handleSyncIndex} />}
        >
          <Route index element={<Navigate to="/admin/categories" replace />} />
          <Route path="categories" element={<AdminCategoriesPage onNavigateToMedia={(slug) => (window.location.hash = `#/admin/media`)} />} />
          <Route path="media" element={<AdminMediaPage />} />
          <Route path="indexer" element={<AdminIndexerPage />} />
          <Route path="tmdb" element={<TMDBSettingsPage />} />
          <Route path="quality" element={<AdminQualityPage />} />
          <Route path="migration" element={<AdminMigrationPage />} />
          <Route path="overview" element={<AdminOverviewPage health={health} onSyncIndex={handleSyncIndex} />} />
        </Route>

        {/* Fallback */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>

      {/* Global Real Video Player Modal */}
      {playingMediaState && (
        <RealVideoPlayerModal
          media={playingMediaState.media}
          initialFile={playingMediaState.initialFile}
          onClose={() => setPlayingMediaState(null)}
        />
      )}
    </HashRouter>
  );
}

function RealVideoPlayerModal({ media, initialFile, onClose }) {
  const [mediaDetail, setMediaDetail] = useState(null);
  const [activeFile, setActiveFile] = useState(initialFile || null);
  const [subtitles, setSubtitles] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let alive = true;
    if (media?.id) {
      setLoading(true);
      getMediaDetail(media.id)
        .then((data) => {
          if (!alive) return;
          setMediaDetail(data);
          if (!activeFile) {
            const firstEp = data.seasons?.[0]?.episodes?.[0] || data.files?.[0];
            if (firstEp) setActiveFile(firstEp);
          }
        })
        .catch(() => {})
        .finally(() => {
          if (alive) setLoading(false);
        });
    } else {
      setLoading(false);
    }
    return () => {
      alive = false;
    };
  }, [media?.id]);

  useEffect(() => {
    let alive = true;
    if (activeFile?.id) {
      getFileSubtitles(activeFile.id)
        .then((res) => {
          if (!alive) return;
          const subs = (res.subtitles || []).map((sub) => ({
            kind: "captions",
            label: sub.label || (sub.language === "ar" ? "العربية" : sub.language),
            src: resolveAPIURL(`/api/stream/file/${activeFile.id}/subtitles/${sub.index}`),
            srcLang: sub.language || "ar",
            default: sub.language === "ar",
          }));
          setSubtitles(subs);
        })
        .catch(() => {
          if (alive) setSubtitles([]);
        });
    } else {
      setSubtitles([]);
    }
    return () => {
      alive = false;
    };
  }, [activeFile?.id]);

  const seasons = mediaDetail?.seasons || [];
  const directFiles = mediaDetail?.files || [];
  const hasSeasons = seasons.length > 0;

  const allPlayableItems = hasSeasons
    ? seasons.flatMap((s) =>
        (s.episodes || []).map((ep) => ({
          ...ep,
          seasonNumber: s.season_number,
        }))
      )
    : directFiles;

  const currentFile = activeFile || allPlayableItems[0] || null;

  const streamSrc = currentFile?.id
    ? resolveAPIURL(`/api/stream/file/${currentFile.id}`)
    : currentFile?.file_path
    ? resolveAPIURL(`/api/stream?path=${encodeURIComponent(currentFile.file_path)}`)
    : "";

  const title = media.titleAr || media.titleEn || "تشغيل الوسائط";
  const poster = resolveAPIURL(media.bannerPath || media.posterPath) || "";

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/95 p-3 sm:p-6 backdrop-blur-2xl text-right" dir="rtl">
      <div className="relative flex h-[92vh] w-full max-w-7xl flex-col overflow-hidden rounded-[2rem] border border-white/15 bg-[#0A0914] p-4 sm:p-6 shadow-2xl">
        {/* Player Header */}
        <div className="mb-4 flex items-center justify-between border-b border-white/10 pb-3">
          <button
            type="button"
            onClick={onClose}
            className="flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-xs font-bold text-white transition hover:bg-white/20"
          >
            <span>العودة ‹</span>
          </button>

          <div className="text-right">
            <p className="text-xs sm:text-sm font-bold text-fuchsia-300">
              {title} {currentFile ? `· ${currentFile.title_ar || currentFile.title_en || (currentFile.episode_number ? `الحلقة ${currentFile.episode_number}` : "ملف التشغيل")}` : ""}
            </p>
          </div>

          <div className="flex items-center gap-2 text-xs font-mono font-bold text-emerald-400 bg-emerald-950/60 px-3 py-1 rounded-full border border-emerald-500/30">
            <span>{currentFile?.resolution || "1080p"} · بث شبكي فوري LAN</span>
          </div>
        </div>

        {/* Video & Real Episodes Grid */}
        <div className="grid flex-1 gap-6 overflow-hidden lg:grid-cols-[1.35fr_0.65fr]">
          {/* Real Video Player Component */}
          <div className="relative flex flex-col justify-center overflow-hidden rounded-2xl border border-white/10 bg-black">
            {streamSrc ? (
              <VideoPlayer
                key={streamSrc}
                src={streamSrc}
                title={title}
                poster={poster}
                tracks={subtitles}
              />
            ) : (
              <div className="flex flex-col items-center justify-center p-8 text-center text-white/60">
                <p className="text-sm font-bold">
                  {loading ? "جارٍ فحص ملفات الفيديو المتاحة..." : "لا يتوفر ملف فيديو صالح للتشغيل في قاعدة البيانات لهذا العمل."}
                </p>
                <p className="mt-2 text-xs text-white/40">
                  قم بفهرسة مجلد الميديا لتفعيل التشغيل الفوري.
                </p>
              </div>
            )}
          </div>

          {/* Real Episodes List Sidebar */}
          <div className="flex flex-col overflow-hidden rounded-2xl border border-white/10 bg-[#0D0E18] p-4 text-right">
            <div className="mb-3 flex items-center justify-between border-b border-white/10 pb-3">
              <span className="rounded-md border border-white/10 bg-white/5 px-2.5 py-1 text-xs font-bold text-white/70">
                {allPlayableItems.length} حلقات / ملفات
              </span>
              <h4 className="text-sm font-black text-white">قائمة الحلقات الفعلية</h4>
            </div>

            <div className="flex-1 space-y-2 overflow-y-auto pr-1">
              {allPlayableItems.map((item, idx) => {
                const isSelected = currentFile && (currentFile.id === item.id || currentFile.file_path === item.file_path);
                const epTitle = item.title_ar || item.title_en || (item.episode_number ? `الحلقة ${item.episode_number}` : `الملف #${idx + 1}`);
                return (
                  <button
                    key={item.id || idx}
                    type="button"
                    onClick={() => setActiveFile(item)}
                    className={`flex w-full items-center justify-between rounded-xl p-3 text-right transition ${
                      isSelected
                        ? "bg-gradient-to-r from-purple-900/90 to-fuchsia-900/90 border border-fuchsia-500/50 text-white shadow-lg"
                        : "border border-white/5 bg-white/[0.02] text-white/70 hover:bg-white/5 hover:text-white"
                    }`}
                  >
                    <span className="text-[11px] font-mono text-white/50">{item.resolution || "1080p"}</span>
                    <div>
                      <p className="text-xs font-bold text-white">{epTitle}</p>
                      <p className="text-[10px] text-white/40">
                        {item.seasonNumber ? `الموسم ${item.seasonNumber} · ` : ""}
                        {item.file_size ? `${(Number(item.file_size) / (1024 * 1024)).toFixed(0)} MB` : "فيديو محلي"}
                      </p>
                    </div>
                  </button>
                );
              })}
              {allPlayableItems.length === 0 && !loading && (
                <div className="p-8 text-center text-xs text-white/40">
                  لا توجد حلقات فعلية مسجلة لهذا العمل في قاعدة البيانات.
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
