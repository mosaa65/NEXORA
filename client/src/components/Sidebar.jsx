import Icon from "./Icon.jsx";

const cinemaNavItems = [
  { id: "dashboard", label: "الرئيسية", icon: "dashboard" },
  { id: "movies", label: "الأفلام", icon: "film" },
  { id: "series", label: "المسلسلات", icon: "tv" },
  { id: "anime", label: "الأنمي", icon: "spark" },
  { id: "kids", label: "الأطفال والكرتون", icon: "smile" },
  { id: "documentaries", label: "الوثائقيات", icon: "book" },
  { id: "plays", label: "المسرحيات", icon: "mask" },
  { id: "favorites", label: "المفضلة", icon: "star" }
];

const adminSubItems = [
  { id: "admin-categories", label: "إدارة الأقسام والتصنيفات", icon: "mask" },
  { id: "admin-manager", label: "إدارة وتحرير الأعمال", icon: "film" },
  { id: "admin-indexing", label: "فهرسة واختيار المجلدات", icon: "search" },
  { id: "admin-tmdb", label: "إعدادات مزود TMDB", icon: "spark" },
  { id: "admin-quality", label: "صحة وجودة المكتبة", icon: "spark" },
  { id: "admin-migration", label: "معالج الترتيب والنقل", icon: "arrowLeft" },
  { id: "admin-overview", label: "حالة النظام والخدمات", icon: "dashboard" }
];

export default function Sidebar({ isOpen, onClose, activeView, onNavigate, onAdminNavigate, activeAdminAnchor }) {
  const isAdminActive = activeView === "admin";

  return (
    <>
      {/* Mobile Backdrop */}
      {isOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/80 backdrop-blur-md lg:hidden"
          onClick={onClose}
        />
      )}

      <aside
        className={`fixed inset-y-0 right-0 z-50 flex w-72 flex-col justify-between border-l border-white/10 bg-[#0A0914] p-5 shadow-2xl transition-transform duration-300 lg:static lg:z-0 lg:w-64 xl:w-72 lg:translate-x-0 lg:border-l-0 lg:bg-[#0A0914]/90 lg:rounded-[2rem] lg:border lg:border-white/10 lg:shadow-panel ${
          isOpen ? "translate-x-0" : "translate-x-full"
        }`}
      >
        <div className="overflow-y-auto max-h-[calc(100vh-140px)] pr-1">
          {/* Brand Logo Section */}
          <div className="mb-6 flex items-center justify-between border-b border-white/10 pb-5 text-right">
            <div className="flex items-center gap-3">
              {/* Triangular Logo Emblem */}
              <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-fuchsia-600 to-purple-800 text-white shadow-lg shadow-purple-900/50">
                <svg className="h-6 w-6 stroke-current" fill="none" viewBox="0 0 24 24" strokeWidth="2">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 3L2 21h20L12 3z" />
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 9l-4 7h8l-4-7z" />
                </svg>
              </div>
              <div>
                <h1 className="text-xl font-black text-white tracking-wide">NEXORA</h1>
                <p className="text-[11px] font-bold text-white/40">نظام إدارة وبث الوسائط</p>
              </div>
            </div>

            <button
              type="button"
              onClick={onClose}
              className="rounded-xl border border-white/10 p-2 text-white/60 hover:text-white lg:hidden"
            >
              ✕
            </button>
          </div>

          {/* Section 1: Cinema Library */}
          <div className="mb-4">
            <p className="px-3 text-[11px] font-bold text-white/40 uppercase tracking-wider mb-2">المكتبة السينمائية</p>
            <nav className="space-y-1">
              {cinemaNavItems.map((item) => {
                const active = activeView === item.id;
                return (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => onNavigate(item.id)}
                    className={`flex w-full items-center gap-3 rounded-2xl px-3.5 py-2.5 text-right font-bold transition duration-200 ${
                      active
                        ? "bg-gradient-to-r from-purple-800 via-fuchsia-700 to-purple-900 text-white shadow-lg shadow-purple-900/50 border border-fuchsia-500/30"
                        : "text-white/65 hover:bg-white/[0.06] hover:text-white"
                    }`}
                  >
                    <Icon name={item.icon} className={`h-4 w-4 shrink-0 ${active ? "text-white" : "text-white/50"}`} />
                    <span className="flex-1 text-xs">{item.label}</span>
                  </button>
                );
              })}
            </nav>
          </div>

          {/* Section 2: Admin & Management Suite */}
          <div className="pt-3 border-t border-white/10">
            <button
              type="button"
              onClick={() => onNavigate("admin")}
              className={`flex w-full items-center justify-between rounded-2xl px-3.5 py-2.5 text-right font-bold transition duration-200 mb-2 ${
                isAdminActive
                  ? "bg-gradient-to-r from-fuchsia-800 to-purple-900 text-white border border-fuchsia-500/30 shadow-md"
                  : "text-white/80 hover:bg-white/[0.06] hover:text-white"
              }`}
            >
              <div className="flex items-center gap-3">
                <Icon name="settings" className="h-4 w-4 text-fuchsia-400" />
                <span className="text-xs font-black">لوحة التحكم والإدارة</span>
              </div>
              <span className="text-[10px] px-1.5 py-0.5 rounded bg-fuchsia-500/20 text-fuchsia-300 font-mono">CMS</span>
            </button>

            {/* Admin Sub-navigation */}
            <div className="mr-2 pr-2 border-r border-fuchsia-500/20 space-y-1">
              {adminSubItems.map((adminItem) => {
                const isSubActive = isAdminActive && activeAdminAnchor === adminItem.id;
                return (
                  <button
                    key={adminItem.id}
                    type="button"
                    onClick={() => {
                      onAdminNavigate?.(adminItem.id);
                    }}
                    className={`flex w-full items-center gap-2.5 rounded-xl px-3 py-2 text-right text-xs font-semibold transition ${
                      isSubActive
                        ? "bg-fuchsia-600/30 text-white border border-fuchsia-500/40 shadow-sm"
                        : "text-white/55 hover:bg-white/[0.05] hover:text-white"
                    }`}
                  >
                    <Icon name={adminItem.icon} className={`h-3.5 w-3.5 ${isSubActive ? "text-fuchsia-300" : "text-white/40"}`} />
                    <span className="truncate">{adminItem.label}</span>
                  </button>
                );
              })}
            </div>
          </div>
        </div>

        {/* User Profile Badge at Bottom */}
        <div className="mt-4 rounded-2xl border border-white/10 bg-white/[0.03] p-3 text-right">
          <div className="flex items-center gap-3">
            <div className="grid h-9 w-9 place-items-center rounded-xl bg-gradient-to-tr from-purple-600 to-fuchsia-500 text-xs font-black text-white">
              NX
            </div>
            <div>
              <p className="text-xs font-bold text-white">مكتبة الاستراحة</p>
              <p className="text-[10px] text-white/40">بث شبكي فوري LAN 100+ TB</p>
            </div>
          </div>
        </div>
      </aside>
    </>
  );
}
