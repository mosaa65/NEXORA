import { useState } from "react";
import Icon from "./Icon.jsx";

export default function TopBar({
  searchQuery,
  onSearchChange,
  searchResults = [],
  onOpenMedia,
  onQuickPlay,
  onToggleSidebar
}) {
  const [showSearchDropdown, setShowSearchDropdown] = useState(false);

  return (
    <header className="sticky top-0 z-30 mb-6 flex items-center justify-between gap-4 rounded-2xl border border-white/10 bg-[#0A0914]/90 px-4 py-3 shadow-panel backdrop-blur-2xl">
      {/* Mobile Drawer Button */}
      <button
        type="button"
        onClick={onToggleSidebar}
        className="flex h-10 w-10 items-center justify-center rounded-xl border border-white/10 bg-white/5 text-white lg:hidden"
        title="القائمة"
      >
        ☰
      </button>

      {/* Search Input Box */}
      <div className="relative flex-1 max-w-lg">
        <div className="flex w-full items-center gap-3 rounded-full border border-white/10 bg-white/[0.06] px-4 py-2.5 transition focus-within:border-fuchsia-500/50 focus-within:bg-black/60">
          <Icon name="search" className="h-4 w-4 text-white/50 shrink-0" />
          <input
            value={searchQuery}
            onChange={(event) => {
              onSearchChange(event.target.value);
              setShowSearchDropdown(true);
            }}
            onFocus={() => setShowSearchDropdown(true)}
            placeholder="ابحث عن أنمي، فيلم، مسلسل..."
            className="w-full bg-transparent text-xs font-bold text-white outline-none placeholder:text-white/40 sm:text-sm text-right"
          />
          {searchQuery && (
            <button
              type="button"
              onClick={() => {
                onSearchChange("");
                setShowSearchDropdown(false);
              }}
              className="text-xs text-white/40 hover:text-white"
            >
              ✕
            </button>
          )}
        </div>

        {/* Live Search Dropdown */}
        {showSearchDropdown && searchQuery.trim() && (
          <div className="absolute right-0 top-full z-50 mt-2 max-h-96 w-full overflow-y-auto rounded-2xl border border-white/15 bg-[#0D0E18]/98 p-2 shadow-2xl backdrop-blur-2xl">
            <div className="flex items-center justify-between border-b border-white/10 p-2 text-xs text-white/60">
              <span>نتائج البحث الفوري ({searchResults.length})</span>
              <button
                type="button"
                onClick={() => setShowSearchDropdown(false)}
                className="hover:text-white"
              >
                إغلاق ✕
              </button>
            </div>
            {searchResults.length > 0 ? (
              <div className="mt-1 divide-y divide-white/5">
                {searchResults.slice(0, 6).map((item) => (
                  <div
                    key={item.id}
                    className="flex items-center justify-between p-2.5 transition hover:bg-white/5 rounded-xl"
                  >
                    <button
                      type="button"
                      onClick={() => {
                        onQuickPlay(item);
                        setShowSearchDropdown(false);
                      }}
                      className="rounded-xl bg-gradient-to-r from-fuchsia-600 to-purple-600 px-3 py-1.5 text-xs font-bold text-white shadow-neon shrink-0"
                    >
                      ▶ تشغيل
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        onOpenMedia(item);
                        setShowSearchDropdown(false);
                      }}
                      className="text-right flex-1 mx-3 min-w-0"
                    >
                      <p className="truncate text-xs font-bold text-white sm:text-sm">{item.titleAr}</p>
                      <p className="truncate text-[10px] text-white/50">{item.titleEn} · {item.year} · {item.resolution}</p>
                    </button>
                  </div>
                ))}
              </div>
            ) : (
              <div className="p-4 text-center text-xs text-white/50">
                لا توجد نتائج تطابق "{searchQuery}"
              </div>
            )}
          </div>
        )}
      </div>

      {/* Brand Name Emblem in Header (for Tablet & Desktop) */}
      <div className="flex items-center gap-3">
        <div className="text-right">
          <h2 className="text-base font-black text-white">مكتبتي</h2>
          <p className="text-[10px] font-bold text-white/40">نظام إدارة الوسائط</p>
        </div>
        <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-fuchsia-600 to-purple-800 text-white shadow-lg shadow-purple-900/50">
          <svg className="h-5 w-5 stroke-current" fill="none" viewBox="0 0 24 24" strokeWidth="2">
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 3L2 21h20L12 3z" />
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 9l-4 7h8l-4-7z" />
          </svg>
        </div>
      </div>
    </header>
  );
}
