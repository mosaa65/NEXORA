import React, { useState } from "react";
import { Outlet, useNavigate, useLocation } from "react-router-dom";
import Sidebar from "../components/Sidebar.jsx";
import TopBar from "../components/TopBar.jsx";

export default function CustomerCinemaLayout({
  health,
  categories = [],
  searchQuery = "",
  onSearchChange,
  searchResults = [],
  onOpenMedia = (item) => { window.location.hash = `#/media/${item.id}`; },
  onQuickPlay = () => {},
}) {
  const navigate = useNavigate();
  const location = useLocation();
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  // Derive active view from pathname
  const path = location.pathname;
  let activeView = "dashboard";
  if (path.startsWith("/catalog/")) {
    activeView = path.replace("/catalog/", "");
  } else if (path === "/favorites") {
    activeView = "favorites";
  }

  function handleNavigate(viewId) {
    setIsSidebarOpen(false);
    if (viewId === "dashboard") {
      navigate("/");
    } else if (viewId === "favorites") {
      navigate("/favorites");
    } else {
      navigate(`/catalog/${viewId}`);
    }
  }

  return (
    <div className="relative min-h-screen bg-[var(--bg-base)] text-[var(--text-primary)] selection:bg-[var(--color-accent-light)] font-sans text-right" dir="rtl">
      {/* Background Ambient Glow */}
      <div className="pointer-events-none fixed inset-0 z-0 overflow-hidden">
        <div className="absolute -right-40 -top-40 h-[500px] w-[500px] rounded-full bg-purple-900/15 blur-[120px]" />
        <div className="absolute -left-40 top-1/3 h-[500px] w-[500px] rounded-full bg-fuchsia-900/10 blur-[140px]" />
        <div className="absolute bottom-0 right-1/4 h-[400px] w-[400px] rounded-full bg-cyan-900/10 blur-[130px]" />
      </div>

      <div className="relative z-10 mx-auto flex min-h-screen max-w-[1920px] gap-4 p-3 sm:p-5 lg:gap-6 lg:p-6">
        {/* Cinema Navigation Sidebar */}
        <Sidebar
          isOpen={isSidebarOpen}
          onClose={() => setIsSidebarOpen(false)}
          activeView={activeView}
          onNavigate={handleNavigate}
        />

        {/* Main Content Area */}
        <div className="flex min-w-0 flex-1 flex-col gap-6 lg:pr-[18.5rem] xl:pr-[19rem]">
          {/* Top Bar Header */}
          <TopBar
            health={health}
            categories={categories}
            searchQuery={searchQuery}
            onSearchChange={onSearchChange}
            searchResults={searchResults}
            onOpenMedia={onOpenMedia}
            onQuickPlay={onQuickPlay}
            onToggleSidebar={() => setIsSidebarOpen(!isSidebarOpen)}
          />

          {/* Page Outlet */}
          <main className="min-w-0 flex-1">
            <Outlet />
          </main>
        </div>
      </div>
    </div>
  );
}
