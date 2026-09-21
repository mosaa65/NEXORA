import React, { useEffect, useLayoutEffect, useState, useDeferredValue, Suspense } from "react";
import { HashRouter, Routes, Route, Navigate, useNavigate, useParams, useLocation, useNavigationType } from "react-router-dom";
import CustomerCinemaLayout from "./layouts/CustomerCinemaLayout.jsx";
import AdminPortalLayout from "./layouts/AdminPortalLayout.jsx";
import DashboardPage from "./pages/DashboardPage.jsx";
import CategoryPage from "./pages/CategoryPage.jsx";
import SmartHubPage from "./pages/SmartHubPage.jsx";
const FranchisePage = React.lazy(() => import("./pages/FranchisePage.jsx"));
const PersonPage = React.lazy(() => import("./pages/PersonPage.jsx"));
const DirectoryPage = React.lazy(() => import("./pages/DirectoryPage.jsx"));
const MediaDetailsPage = React.lazy(() => import("./pages/MediaDetailsPage.jsx"));
const AdminCategoriesPage = React.lazy(() => import("./pages/admin/AdminCategoriesPage.jsx"));
const AdminCollectionsPage = React.lazy(() => import("./pages/admin/AdminCollectionsPage.jsx"));
const AdminSmartHubsPage = React.lazy(() => import("./pages/admin/AdminSmartHubsPage.jsx"));
const AdminMediaPage = React.lazy(() => import("./pages/admin/AdminMediaPage.jsx"));
const AdminIndexerPage = React.lazy(() => import("./pages/admin/AdminIndexerPage.jsx"));
const AdminReviewPage = React.lazy(() => import("./pages/admin/AdminReviewPage.jsx"));
const AdminQualityPage = React.lazy(() => import("./pages/admin/AdminQualityPage.jsx"));
const AdminMigrationPage = React.lazy(() => import("./pages/admin/AdminMigrationPage.jsx"));
const AdminOverviewPage = React.lazy(() => import("./pages/admin/AdminOverviewPage.jsx"));
const AdminTransferPage = React.lazy(() => import("./pages/admin/AdminTransferPage.jsx"));
const TMDBSettingsPage = React.lazy(() => import("./pages/TMDBSettingsPage.jsx"));
const AdminLoginPage = React.lazy(() => import("./pages/AdminLoginPage.jsx"));
// Dev-only harness for the Video.js player. Not linked anywhere in the UI.
const DevPlayerPage = React.lazy(() => import("./pages/DevPlayerPage.jsx"));
const WatchPage = React.lazy(() => import("./pages/WatchPage.jsx"));

import { TransferProvider } from "./context/TransferContext.jsx";
import { PlaybackProvider, usePlayback } from "./context/PlaybackContext.jsx";
import MiniPlayerDock from "./components/MiniPlayerDock.jsx";
import TransferModal from "./components/transfer/TransferModal.jsx";
import MiniTransferCenter from "./components/transfer/MiniTransferCenter.jsx";
import { categorySeed, getCategoryMeta } from "./data/library.js";
import { getCategories, getHealth, getMediaDetail, getFileSubtitles, getMediaList, syncIndex, resolveAPIURL } from "./lib/api.js";

/**
 * Handles scroll-to-top ONLY on fresh navigations (PUSH/REPLACE).
 * POP (Back/Forward) is intentionally ignored — each page's useScrollRestoration
 * hook is responsible for restoring the correct scroll position after data loads.
 */
function ScrollManager() {
  const location = useLocation();
  const navType = useNavigationType();

  useEffect(() => {
    if (typeof window !== "undefined" && "scrollRestoration" in window.history) {
      window.history.scrollRestoration = "manual";
    }
  }, []);

  useLayoutEffect(() => {
    // Never interfere with POP (Back/Forward button) — let each page restore itself
    if (navType === "POP") return;
    window.scrollTo({ top: 0, left: 0, behavior: "instant" });
  }, [location.pathname, location.search, navType]);

  return null;
}

// Helper Wrapper for Category View with unique key per category
function CategoryRouteWrapper({ onOpenMedia, onQuickPlay }) {
  const { category = "series" } = useParams();
  return (
    <CategoryPage
      key={category}
      selectedCategory={category}
      onOpenMedia={onOpenMedia}
      onQuickPlay={onQuickPlay}
    />
  );
}

function SmartHubRouteWrapper({ onOpenMedia }) {
  const { slug } = useParams();
  return <SmartHubPage key={slug} slug={slug} onOpenMedia={onOpenMedia} />;
}
function FranchiseRouteWrapper({ onOpenMedia }) {
  const { slug } = useParams();
  return <FranchisePage key={slug} slug={slug} onOpenMedia={onOpenMedia} />;
}
function PersonRouteWrapper({ onOpenMedia }) {
  const { slug } = useParams();
  return <PersonPage key={slug} slug={slug} onOpenMedia={onOpenMedia} />;
}

// Helper Wrapper for Media Details View with unique key per media id
function MediaDetailsRouteWrapper({ onOpenCategory, onQuickPlay }) {
  const { id } = useParams();

  if (!id) return null;

  return (
    <MediaDetailsPage
      key={id}
      media={{ id: parseInt(id, 10) }}
      onOpenCategory={onOpenCategory}
      onQuickPlay={onQuickPlay}
    />
  );
}

function AppRoutes() {
  const navigate = useNavigate();
  const [health, setHealth] = useState(null);
  const [categories, setCategories] = useState(categorySeed);
  const [searchQuery, setSearchQuery] = useState("");
  const deferredQuery = useDeferredValue(searchQuery);
  const [searchResults, setSearchResults] = useState([]);
  const [isSearching, setIsSearching] = useState(false);

  // The floating dock keeps a video alive across pages; opening a new watch
  // screen should replace it, so gameplay never has two players at once.
  const { close: closeDock, isActive: dockActive } = usePlayback();

  // Playback now lives on its own screen (/watch/:id) instead of a floating modal.
  function handleQuickPlay(item, episodeOrFile) {
    if (!item?.id) return;
    if (dockActive) closeDock();
    const query = episodeOrFile?.id ? `?file=${episodeOrFile.id}` : "";
    navigate(`/watch/${item.id}${query}`);
  }

  // Expanding the floating dock returns to the watch screen and hands playback
  // back to the page (closing the dock so only one player exists).
  function handleExpandDock(payload) {
    closeDock();
    if (payload?.mediaId) navigate(`/watch/${payload.mediaId}${payload.fileId ? `?file=${payload.fileId}` : ""}`);
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

    // Debounce typing: a single search request is made after the user pauses,
    // rather than once for every character entered.
    const query = deferredQuery.trim();
    const timer = window.setTimeout(() => {
      setIsSearching(true);
      // Catalogue search deliberately goes through PostgreSQL-backed /api/media.
      // Meilisearch remains available for administration, but card data must have
      // the same local offline-first contract as every other catalogue surface.
      getMediaList({ q: query, limit: 30 })
        .then((payload) => {
          setSearchResults(payload?.items || []);
        })
        .catch(() => setSearchResults([]))
        .finally(() => setIsSearching(false));
    }, 300);

    return () => window.clearTimeout(timer);
  }, [deferredQuery]);

  async function handleSyncIndex() {
    try {
      await syncIndex(1000);
    } catch {}
  }

  return (
<>
      <ScrollManager />
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
              searchResults={searchResults}
              onOpenMedia={(item) => navigate(`/media/${item.id}`)}
              onQuickPlay={handleQuickPlay}
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
                onOpenMedia={(item) => navigate(`/media/${item.id}`)}
                onQuickPlay={handleQuickPlay}
                onNavigateCategory={(slug) => navigate(`/catalog/${slug}`)}
              />
            }
          />

          {/* Category Catalog Route with Origin Hubs */}
          <Route
            path="catalog/:category"
            element={
              <CategoryRouteWrapper
                onOpenMedia={(item) => navigate(`/media/${item.id}`)}
                onQuickPlay={handleQuickPlay}
              />
            }
          />
          <Route path="hub/:slug" element={<SmartHubRouteWrapper onOpenMedia={(item) => navigate(`/media/${item.id}`)} />} />
          <Route path="franchise/:slug" element={<FranchiseRouteWrapper onOpenMedia={(item) => navigate(`/media/${item.id}`)} />} />
          <Route path="person/:slug" element={<PersonRouteWrapper onOpenMedia={(item) => navigate(`/media/${item.id}`)} />} />
          <Route path="directory/hubs" element={<DirectoryPage kind="hubs" onOpen={(hub) => navigate(`/hub/${hub.slug}`)} />} />
          <Route path="directory/people" element={<DirectoryPage kind="people" onOpen={(person) => navigate(`/person/${person.slug}`)} />} />
          <Route path="directory/franchises" element={<DirectoryPage kind="franchises" onOpen={(franchise) => navigate(`/franchise/${franchise.slug}`)} />} />

          {/* Favorites Route */}
          <Route
            path="favorites"
            element={
              <CategoryRouteWrapper
                onOpenMedia={(item) => navigate(`/media/${item.id}`)}
                onQuickPlay={handleQuickPlay}
              />
            }
          />

          {/* Media Details Route */}
          <Route
            path="media/:id"
            element={
              <MediaDetailsRouteWrapper
                onOpenCategory={(slug) => navigate(`/catalog/${slug}`)}
                onQuickPlay={handleQuickPlay}
              />
            }
          />

          {/* Watch Screen — playback page inside the customer shell (keeps the
              search bar and navigation, like every other page). */}
          <Route
            path="watch/:id"
            element={
              <Suspense fallback={<div className="min-h-[60vh]" />}>
                <WatchPage />
              </Suspense>
            }
          />
        </Route>

        {/* ========================================================================= */}
        {/* 2. ADMIN AUTHENTICATION                                                  */}
        {/* ========================================================================= */}
        <Route
          path="/admin/login"
          element={
            <Suspense fallback={<div className="min-h-screen bg-[var(--bg-base)]" />}>
              <AdminLoginPage
                onLoginSuccess={() => navigate("/admin/categories")}
              />
            </Suspense>
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
          <Route path="categories" element={<AdminCategoriesPage onNavigateToMedia={() => navigate("/admin/media")} />} />
          <Route path="collections" element={<AdminCollectionsPage />} />
          <Route path="hubs" element={<AdminSmartHubsPage />} />
          <Route path="media" element={<AdminMediaPage />} />
          <Route path="indexer" element={<AdminIndexerPage />} />
          <Route path="review" element={<AdminReviewPage />} />
          <Route path="tmdb" element={<TMDBSettingsPage />} />
          <Route path="quality" element={<AdminQualityPage />} />
          <Route path="migration" element={<AdminMigrationPage />} />
          <Route path="transfer" element={<AdminTransferPage />} />
          <Route path="overview" element={<AdminOverviewPage health={health} onSyncIndex={handleSyncIndex} />} />
        </Route>

        {/* ========================================================================= */}
        {/* DEV HARNESS — not linked in the UI, reachable by URL only.                */}
        {/* Renders the Video.js-based NexoraPlayer against a real catalogue file     */}
        {/* in isolation, without opening the full playback modal.                    */}
        {/* ========================================================================= */}
        <Route
          path="/dev/player"
          element={
            <Suspense fallback={<div className="min-h-screen bg-[var(--bg-base)]" />}>
              <DevPlayerPage />
            </Suspense>
          }
        />

        {/* Fallback */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>

      {/* Global mini player — lives above the routes so a minimised video keeps
          playing while the user navigates to other pages. */}
      <MiniPlayerDock onExpand={handleExpandDock} />

{/* Global USB Transfer Experience (Modern Modal, Mini Transfer Center) */}
      <TransferModal />
      <MiniTransferCenter />
    </>
  );
}

export default function App() {
  return (
    <TransferProvider>
    <PlaybackProvider>
    <HashRouter>
      <AppRoutes />
    </HashRouter>
    </PlaybackProvider>
    </TransferProvider>
  );
}

