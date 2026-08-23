import React, { useState, useEffect } from "react";
import { Outlet, useNavigate, useLocation, Link } from "react-router-dom";
import Icon from "../components/Icon.jsx";
import { adminLogout, checkAdminSession } from "../lib/api.js";

const adminNavItems = [
  { id: "categories", path: "/admin/categories", label: "إدارة الأقسام والتصنيفات", icon: "mask", badge: "Core" },
  { id: "media", path: "/admin/media", label: "إدارة وتحرير الأعمال", icon: "film", badge: "CRUD" },
  { id: "indexer", path: "/admin/indexer", label: "فهرسة واختيار المجلدات", icon: "search", badge: "Scan" },
  { id: "tmdb", path: "/admin/tmdb", label: "إعدادات مزود TMDB والكوتا", icon: "spark", badge: "API" },
  { id: "quality", path: "/admin/quality", label: "صحة وجودة المكتبة", icon: "spark", badge: "Doctor" },
  { id: "migration", path: "/admin/migration", label: "معالج الترتيب والنقل", icon: "arrowLeft", badge: "Tools" },
  { id: "overview", path: "/admin/overview", label: "حالة النظام والخدمات", icon: "dashboard", badge: "Live" },
];

export default function AdminPortalLayout({ health, onSyncIndex }) {
  const navigate = useNavigate();
  const location = useLocation();
  const [adminUser, setAdminUser] = useState(null);
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  useEffect(() => {
    // Check local storage / session token
    const token = localStorage.getItem("nexora_admin_token");
    const userStr = localStorage.getItem("nexora_admin_user");
    if (!token) {
      navigate("/admin/login");
      return;
    }
    if (userStr) {
      try {
        setAdminUser(JSON.parse(userStr));
      } catch {
        setAdminUser({ name: "مدير النظام", role: "superadmin" });
      }
    }
  }, [navigate]);

  async function handleLogout() {
    try {
      await adminLogout();
    } catch {}
    localStorage.removeItem("nexora_admin_token");
    localStorage.removeItem("nexora_admin_user");
    navigate("/admin/login");
  }

  return (
    <div className="relative min-h-screen bg-[#07060D] text-white selection:bg-purple-600/40 selection:text-white font-sans text-right" dir="rtl">
      {/* Background Micro Glow */}
      <div className="pointer-events-none fixed inset-0 z-0 overflow-hidden">
        <div className="absolute top-0 right-1/4 h-[500px] w-[500px] rounded-full bg-purple-900/10 blur-[150px]" />
        <div className="absolute bottom-10 left-10 h-[400px] w-[400px] rounded-full bg-indigo-900/10 blur-[150px]" />
      </div>

      <div className="relative z-10 mx-auto flex min-h-screen max-w-[1920px] gap-4 p-3 sm:p-5 lg:gap-6 lg:p-6">
        {/* Isolated Admin Navigation Sidebar */}
        <aside className="fixed inset-y-0 right-0 z-50 flex w-72 flex-col justify-between border-l border-white/10 bg-[#0A0914] p-5 shadow-2xl transition-transform duration-300 lg:static lg:z-0 lg:w-64 xl:w-72 lg:translate-x-0 lg:border-l-0 lg:bg-[#0A0914]/90 lg:rounded-[2rem] lg:border lg:border-white/10">
          <div className="space-y-6">
            {/* Admin Brand Logo Header */}
            <div className="flex items-center justify-between border-b border-white/10 pb-4 text-right">
              <div className="flex items-center gap-3">
                <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-indigo-600 to-purple-800 text-white shadow-lg shadow-purple-900/50">
                  <Icon name="settings" className="h-6 w-6" />
                </div>
                <div>
                  <h1 className="text-lg font-black text-white flex items-center gap-1.5">
                    NEXORA <span className="text-[10px] px-2 py-0.5 rounded-full bg-purple-500/20 text-purple-300 border border-purple-500/30">ADMIN</span>
                  </h1>
                  <p className="text-[10px] text-white/50">لوحة الإدارة والتحكم المركزي</p>
                </div>
              </div>
            </div>

            {/* Admin Navigation Links */}
            <div className="space-y-1">
              <p className="px-3 text-[11px] font-bold text-white/40 uppercase tracking-wider mb-2">أدوات الإدارة والخدمات</p>
              <nav className="space-y-1">
                {adminNavItems.map((item) => {
                  const isActive = location.pathname.startsWith(item.path);
                  return (
                    <Link
                      key={item.id}
                      to={item.path}
                      className={`flex w-full items-center justify-between rounded-2xl px-3.5 py-2.5 text-right font-bold transition duration-200 ${
                        isActive
                          ? "bg-gradient-to-r from-purple-800 via-indigo-700 to-purple-900 text-white shadow-lg shadow-purple-900/50 border border-indigo-500/40"
                          : "text-white/65 hover:bg-white/[0.06] hover:text-white"
                      }`}
                    >
                      <div className="flex items-center gap-2.5">
                        <Icon name={item.icon} className={`h-4 w-4 shrink-0 ${isActive ? "text-white" : "text-white/40"}`} />
                        <span className="text-xs">{item.label}</span>
                      </div>
                      <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-white/10 text-white/60">
                        {item.badge}
                      </span>
                    </Link>
                  );
                })}
              </nav>
            </div>
          </div>

          {/* Admin Profile & Actions at Bottom */}
          <div className="space-y-3 pt-4 border-t border-white/10">
            {/* Active Admin User Pill */}
            <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-3 text-right flex items-center justify-between">
              <div className="flex items-center gap-2.5">
                <div className="grid h-8 w-8 place-items-center rounded-xl bg-gradient-to-tr from-purple-600 to-indigo-500 text-xs font-black text-white">
                  AD
                </div>
                <div>
                  <p className="text-xs font-bold text-white">{adminUser?.name || "مدير النظام"}</p>
                  <p className="text-[10px] text-emerald-400 font-mono">جلسة نشطة 🟢</p>
                </div>
              </div>

              <button
                onClick={handleLogout}
                className="p-1.5 rounded-lg hover:bg-rose-500/20 text-gray-400 hover:text-rose-300 transition"
                title="تسجيل الخروج"
              >
                <Icon name="arrowLeft" className="w-4 h-4" />
              </button>
            </div>

            {/* Quick Switch to Customer Lounge */}
            <Link
              to="/"
              className="w-full py-2.5 rounded-2xl bg-white/[0.06] hover:bg-white/10 border border-white/10 text-xs font-bold text-white flex items-center justify-center gap-2 transition"
            >
              <span>🍿 العودة لواجهة تصفح الاستراحة</span>
              <span>‹</span>
            </Link>
          </div>
        </aside>

        {/* Admin Content Area */}
        <div className="flex min-w-0 flex-1 flex-col gap-6">
          {/* Admin Header Bar */}
          <header className="flex items-center justify-between gap-4 p-4 rounded-2xl bg-[#0A0914]/80 border border-white/10 backdrop-blur-xl">
            <div className="flex items-center gap-3">
              <span className="text-xs font-bold text-gray-400">بوابة الإدارة</span>
              <span className="text-gray-600">/</span>
              <span className="text-xs font-bold text-purple-300">
                {adminNavItems.find((i) => location.pathname.startsWith(i.path))?.label || "لوحة التحكم"}
              </span>
            </div>

            <div className="flex items-center gap-3">
              {/* Server Health Status Pill */}
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-emerald-950/40 border border-emerald-500/30 text-xs text-emerald-300">
                <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                <span className="font-semibold">السيرفر متصل (Port 8080)</span>
              </div>

              <button
                onClick={onSyncIndex}
                className="px-3.5 py-1.5 rounded-xl bg-purple-950/60 hover:bg-purple-900/60 border border-purple-500/30 text-purple-300 text-xs font-bold transition flex items-center gap-1.5"
              >
                <Icon name="spark" className="w-3.5 h-3.5" />
                <span>مزامنة الفهرس</span>
              </button>
            </div>
          </header>

          {/* Admin Page Outlet */}
          <main className="min-w-0 flex-1">
            <Outlet />
          </main>
        </div>
      </div>
    </div>
  );
}
