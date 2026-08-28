import { useState, useRef, useEffect } from "react";
import Icon from "./Icon.jsx";
import useTheme from "../hooks/useTheme.js";
import { resolveAPIURL } from "../lib/api.js";

export default function TopBar({
  searchQuery = "",
  onSearchChange = () => {},
  searchResults = [],
  onOpenMedia = (item) => { window.location.hash = `#/media/${item.id}`; },
  onQuickPlay = () => {},
  onToggleSidebar,
  isCollapsed = false,
}) {
  const { theme, toggleTheme } = useTheme();
  const [showSearchDropdown, setShowSearchDropdown] = useState(false);
  const searchContainerRef = useRef(null);

  // Close dropdown on outside click
  useEffect(() => {
    function handleClickOutside(event) {
      if (searchContainerRef.current && !searchContainerRef.current.contains(event.target)) {
        setShowSearchDropdown(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const handleKeyDown = (e) => {
    if (e.key === "Escape") {
      setShowSearchDropdown(false);
    } else if (e.key === "Enter" && searchResults.length > 0) {
      onOpenMedia(searchResults[0]);
      setShowSearchDropdown(false);
    }
  };

  return (
    <header className="luminous-container sticky top-0 z-40 mb-6 flex items-center justify-between gap-2.5 sm:gap-4 rounded-full bg-[var(--bg-card)]/95 px-3 sm:px-6 py-2 sm:py-2.5 border border-[var(--border-default)] shadow-[var(--shadow-md)] backdrop-blur-2xl transition-all" dir="rtl">
      {/* Actions: Sidebar Toggle & Theme Toggle */}
      <div className="flex items-center gap-2 shrink-0">
        {/* Sidebar Toggle */}
        <button
          type="button"
          onClick={onToggleSidebar}
          className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-[var(--border-default)] bg-[var(--bg-elevated)] text-[var(--text-secondary)] transition-all duration-200 hover:bg-[var(--bg-surface)] hover:text-[var(--text-primary)] hover:scale-105 active:scale-95 shadow-sm"
          title={isCollapsed ? "توسيع القائمة الجانبية" : "تصغير القائمة لأيقونات"}
          aria-label="تبديل القائمة الجانبية"
        >
          <Icon name="menu" className="h-5 w-5" />
        </button>

        {/* Theme Toggle Icon Button */}
        <button
          type="button"
          onClick={toggleTheme}
          className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-[var(--border-default)] bg-[var(--bg-elevated)] text-[var(--text-secondary)] transition-all duration-200 hover:bg-[var(--bg-surface)] hover:text-[var(--text-primary)] hover:scale-105 active:scale-95 shadow-sm"
          title={theme === "dark" ? "تفعيل المظهر الفاتح" : "تفعيل المظهر الداكن"}
          aria-label="تبديل مظهر التطبيق"
        >
          <Icon
            name={theme === "dark" ? "sun" : "moon"}
            className={`h-4 w-4 transition-colors ${
              theme === "dark" ? "text-amber-300" : "text-purple-300"
            }`}
          />
        </button>
      </div>

      {/* Main Search Input & Live Results (Expands to fill all available space) */}
      <div ref={searchContainerRef} className="relative flex-1 min-w-0 mx-1.5 sm:mx-3">
        <div className="flex w-full items-center gap-2 sm:gap-3 rounded-full border border-[var(--border-default)] bg-[var(--bg-input)] px-3.5 sm:px-4 py-2 text-[var(--text-primary)] shadow-inner transition focus-within:border-[var(--color-accent)] focus-within:ring-2 focus-within:ring-[var(--color-accent-light)]">
          <Icon name="search" className="h-4 w-4 text-fuchsia-400 shrink-0" />
          <input
            type="text"
            value={searchQuery}
            onChange={(event) => {
              onSearchChange(event.target.value);
              setShowSearchDropdown(true);
            }}
            onFocus={() => setShowSearchDropdown(true)}
            onKeyDown={handleKeyDown}
            placeholder="ابحث عن فيلم، كرتون، مسلسل، أو شخصية..."
            className="w-full min-w-0 bg-transparent text-xs sm:text-sm font-semibold text-[var(--text-primary)] outline-none placeholder:text-[var(--text-muted)] text-right selection:bg-[var(--color-accent-light)]"
          />
          {searchQuery && (
            <button
              type="button"
              onClick={() => {
                onSearchChange("");
                setShowSearchDropdown(false);
              }}
              className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-[var(--bg-elevated)] text-xs text-[var(--text-muted)] hover:bg-[var(--bg-surface)] hover:text-[var(--text-primary)] transition"
              title="مسح البحث"
            >
              ✕
            </button>
          )}
        </div>

        {/* Live Search Dropdown */}
        {showSearchDropdown && searchQuery.trim() && (
          <div className="absolute right-0 top-full z-50 mt-2 max-h-96 w-full overflow-y-auto rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-2 shadow-[var(--shadow-xl)] backdrop-blur-2xl ring-1 ring-[var(--border-subtle)] scrollbar-thin">
            <div className="flex items-center justify-between border-b border-[var(--border-subtle)] px-3 py-2 text-xs font-bold text-[var(--text-muted)]">
              <span className="flex items-center gap-1.5">
                <span className="h-2 w-2 rounded-full bg-fuchsia-500 animate-pulse" />
                نتائج البحث ({searchResults.length})
              </span>
              <button
                type="button"
                onClick={() => setShowSearchDropdown(false)}
                className="text-[var(--text-muted)] hover:text-[var(--text-primary)] transition text-xs"
              >
                إغلاق ✕
              </button>
            </div>

            {searchResults.length > 0 ? (
              <div className="mt-1 space-y-1">
                {searchResults.slice(0, 8).map((item) => {
                  const poster = resolveAPIURL(item.poster_path || item.posterPath) || "/nexora-poster-placeholder.PNG";
                  return (
                    <div
                      key={item.id}
                      className="group flex items-center justify-between gap-3 p-2 transition hover:bg-[var(--bg-elevated)] rounded-xl border border-transparent hover:border-[var(--border-subtle)]"
                    >
                      {/* Quick Play Button */}
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation();
                          onQuickPlay(item);
                          setShowSearchDropdown(false);
                        }}
                        className="rounded-lg bg-gradient-to-r from-fuchsia-600 to-purple-600 px-3 py-1.5 text-xs font-bold text-white shadow-md shadow-fuchsia-900/40 hover:opacity-90 active:scale-95 shrink-0 transition"
                      >
                        ▶ تشغيل
                      </button>

                      {/* Item Content Click */}
                      <button
                        type="button"
                        onClick={() => {
                          onOpenMedia(item);
                          setShowSearchDropdown(false);
                        }}
                        className="flex flex-1 items-center gap-3 text-right min-w-0 text-[var(--text-primary)]"
                      >
                        <div className="flex flex-1 flex-col min-w-0">
                          <p className="truncate text-xs sm:text-sm font-bold text-[var(--text-primary)] group-hover:text-[var(--color-accent-hover)] transition-colors">
                            {item.title_ar || item.titleAr || item.title_en || item.titleEn}
                          </p>
                          <div className="flex items-center gap-2 text-[11px] text-[var(--text-muted)] truncate mt-0.5">
                            {(item.title_en || item.titleEn) && (
                              <span className="truncate" dir="ltr">{item.title_en || item.titleEn}</span>
                            )}
                            {(item.release_year || item.year) && (
                              <span>• {item.release_year || item.year}</span>
                            )}
                          </div>
                        </div>
                        <img
                          src={poster}
                          alt=""
                          className="h-12 w-9 rounded-lg object-cover bg-black/20 shrink-0 border border-[var(--border-default)]"
                          onError={(e) => {
                            e.target.src = "/nexora-poster-placeholder.PNG";
                          }}
                        />
                      </button>
                    </div>
                  );
                })}
              </div>
            ) : (
              <div className="p-6 text-center text-xs text-[var(--text-muted)]">
                لا توجد نتائج مطابقة لـ "{searchQuery}"
              </div>
            )}
          </div>
        )}
      </div>

      {/* Official Brand Logo */}
      <div className="flex items-center gap-2 shrink-0">
        <img
          src="/nexora-brand-logo.PNG"
          alt="NEXORA"
          className="h-7 sm:h-9 md:h-10 w-auto object-contain cursor-pointer transition-transform duration-200 hover:scale-105 select-none"
          onClick={() => { window.location.hash = "#/"; }}
          title="NEXORA الرئيسية"
        />
      </div>
    </header>
  );
}
