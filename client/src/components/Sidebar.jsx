import Icon from "./Icon.jsx";
import { navigationItems } from "../data/library.js";

export default function Sidebar({ isOpen, onClose, activeView, onNavigate }) {
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
        <div>
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
                <h1 className="text-xl font-black text-white tracking-wide">مكتبتي</h1>
                <p className="text-[11px] font-bold text-white/40">نظام إدارة الوسائط</p>
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

          {/* Navigation Links with REAL SVG Icons */}
          <nav className="space-y-1.5">
            {navigationItems.map((item) => {
              const active = activeView === item.id;
              return (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => onNavigate(item.id)}
                  className={`flex w-full items-center gap-3.5 rounded-2xl px-4 py-3 text-right font-bold transition duration-200 ${
                    active
                      ? "bg-gradient-to-r from-purple-800 via-fuchsia-700 to-purple-900 text-white shadow-lg shadow-purple-900/50 border border-fuchsia-500/30"
                      : "text-white/65 hover:bg-white/[0.06] hover:text-white"
                  }`}
                >
                  <Icon name={item.icon} className={`h-4 w-4 shrink-0 ${active ? "text-white" : "text-white/50"}`} />
                  <span className="flex-1 text-sm">{item.label}</span>
                </button>
              );
            })}
          </nav>
        </div>

        {/* User Profile Badge at Bottom */}
        <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-3 text-right">
          <div className="flex items-center gap-3">
            <div className="grid h-9 w-9 place-items-center rounded-xl bg-gradient-to-tr from-purple-600 to-fuchsia-500 text-xs font-black text-white">
              MO
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
