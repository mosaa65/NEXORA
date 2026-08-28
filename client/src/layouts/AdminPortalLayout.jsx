import { useEffect, useState } from "react";
import { Link, Outlet, useLocation, useNavigate } from "react-router-dom";
import Icon from "../components/Icon.jsx";
import SideNavigation from "../components/layout/SideNavigation.jsx";
import useTheme from "../hooks/useTheme.js";
import { adminLogout } from "../lib/api.js";

const adminNavItems = [
  { id: "overview", path: "/admin/overview", label: "نظرة عامة", icon: "dashboard" },
  { id: "categories", path: "/admin/categories", label: "الأقسام والتصنيفات", icon: "mask" },
  { id: "collections", path: "/admin/collections", label: "المجموعات والعروض", icon: "spark" },
  { id: "hubs", path: "/admin/hubs", label: "المحاور الذكية", icon: "grid" },
  { id: "media", path: "/admin/media", label: "إدارة الأعمال", icon: "film" },
  { id: "indexer", path: "/admin/indexer", label: "الفهرسة والمجلدات", icon: "search" },
  { id: "tmdb", path: "/admin/tmdb", label: "إعدادات TMDB", icon: "spark" },
  { id: "quality", path: "/admin/quality", label: "جودة المكتبة", icon: "book" },
  { id: "migration", path: "/admin/migration", label: "الترتيب والنقل", icon: "arrowLeft" },
];

export default function AdminPortalLayout({ health, onSyncIndex }) {
  const navigate = useNavigate();
  const location = useLocation();
  const { theme, toggleTheme, font, setFont } = useTheme();
  const [adminUser, setAdminUser] = useState(null);
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const activeItem = adminNavItems.find((item) => location.pathname.startsWith(item.path));
  const isOnline = health?.ok !== false;

  useEffect(() => {
    const token = localStorage.getItem("nexora_admin_token");
    const user = localStorage.getItem("nexora_admin_user");
    if (!token) {
      navigate("/admin/login", { replace: true });
      return;
    }
    try {
      setAdminUser(user ? JSON.parse(user) : { name: "مدير النظام" });
    } catch {
      setAdminUser({ name: "مدير النظام" });
    }
  }, [navigate]);

  async function handleLogout() {
    try { await adminLogout(); } catch {}
    localStorage.removeItem("nexora_admin_token");
    localStorage.removeItem("nexora_admin_user");
    navigate("/admin/login", { replace: true });
  }

  return (
    <div className="relative min-h-screen bg-[var(--bg-base)] text-[var(--text-primary)] font-sans text-right" dir="rtl">
      <div className="pointer-events-none fixed inset-0 z-0 overflow-hidden" aria-hidden="true">
        <div className="absolute top-0 right-1/4 h-[500px] w-[500px] rounded-full bg-purple-900/10 blur-[150px]" />
        <div className="absolute bottom-10 left-10 h-[400px] w-[400px] rounded-full bg-indigo-900/10 blur-[150px]" />
      </div>

      <div className="relative z-10 mx-auto flex min-h-screen max-w-[1920px] gap-4 p-3 sm:p-5 lg:gap-6 lg:p-6">
        <SideNavigation
          isOpen={isSidebarOpen}
          onClose={() => setIsSidebarOpen(false)}
          label="لوحة الإدارة"
          subtitle="تحكم مركزي للمكتبة"
          emblem="settings"
          tone="admin"
          footer={
            <>
              <div className="navigation-footer-card">
                <div className="navigation-footer-card__copy">
                  <strong>{adminUser?.name || "مدير النظام"}</strong>
                  <small>جلسة إدارة محمية</small>
                </div>
                <span className="navigation-status">نشطة</span>
              </div>
              <div className="navigation-footer-actions">
                <button className="navigation-icon-button" type="button" aria-label="تسجيل الخروج" title="تسجيل الخروج" onClick={handleLogout}><Icon name="logout" className="h-4 w-4" /></button>
                <button className="navigation-theme-button" type="button" onClick={toggleTheme} aria-label={theme === "dark" ? "تفعيل المظهر الفاتح" : "تفعيل المظهر الداكن"}>
                  <Icon name={theme === "dark" ? "sun" : "moon"} className="h-4 w-4" />{theme === "dark" ? "فاتح" : "داكن"}
                </button>
                <button className="navigation-theme-button" type="button" onClick={() => setFont(font === "plex" ? "cairo" : "plex")} aria-label="تبديل خط الواجهة"><span className="text-sm font-black">ع</span>{font === "plex" ? "القاهرة" : "Plex"}</button>
                <Link className="navigation-icon-button" to="/" aria-label="العودة لواجهة العميل" title="واجهة العميل"><Icon name="play" className="h-4 w-4" /></Link>
              </div>
            </>
          }
        >
          <section className="navigation-group" aria-labelledby="admin-navigation-label">
            <span id="admin-navigation-label" className="navigation-group__label">إدارة النظام</span>
            <nav>
              <ul className="navigation-list">
                {adminNavItems.map((item) => {
                  const isActive = location.pathname.startsWith(item.path);
                  return (
                    <li key={item.id}>
                      <Link to={item.path} onClick={() => setIsSidebarOpen(false)} className={`navigation-item navigation-item--${item.id} ${isActive ? "is-active" : ""}`} aria-current={isActive ? "page" : undefined}>
                        <span className="navigation-item__icon"><Icon name={item.icon} /></span><span>{item.label}</span>
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </nav>
          </section>
        </SideNavigation>

        <div className="flex min-w-0 flex-1 flex-col gap-4 sm:gap-6 lg:pr-[18.5rem] xl:pr-[19rem]">
          <header className="sticky top-3 z-30 flex min-h-14 items-center justify-between gap-3 rounded-full border border-[var(--border-default)] bg-[var(--bg-card)]/95 px-4 py-2 sm:px-6 shadow-[var(--shadow-md)] backdrop-blur-2xl transition-all" dir="rtl">
            <div className="flex items-center gap-3 min-w-0">
              <button className="flex h-10 w-10 lg:hidden shrink-0 items-center justify-center rounded-full border border-[var(--border-default)] bg-[var(--bg-elevated)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition" type="button" aria-label="فتح قائمة الإدارة" onClick={() => setIsSidebarOpen(true)}>
                <Icon name="menu" className="h-5 w-5" />
              </button>
              <div className="min-w-0">
                <p className="text-[10px] font-bold text-[var(--text-muted)]">بوابة الإدارة المركزية</p>
                <h1 className="truncate text-xs sm:text-sm font-black text-[var(--text-primary)]">{activeItem?.label || "لوحة التحكم"}</h1>
              </div>
            </div>

            <div className="mr-auto flex items-center gap-2.5 shrink-0">
              <span className={`hidden items-center gap-2 rounded-full border px-3 py-1.5 text-xs font-bold sm:flex ${isOnline ? "border-emerald-500/30 bg-emerald-500/10 text-emerald-400" : "border-rose-500/30 bg-rose-500/10 text-rose-400"}`}>
                <span className={`h-2 w-2 rounded-full ${isOnline ? "bg-emerald-400" : "bg-rose-400"}`} />{isOnline ? "الخادم متصل" : "الخادم غير متصل"}
              </span>
              <button onClick={onSyncIndex} className="inline-flex items-center gap-2 rounded-full border border-[var(--border-default)] bg-[var(--bg-elevated)] px-3.5 py-1.5 text-xs font-bold text-[var(--text-primary)] hover:border-[var(--color-accent)] hover:text-[var(--color-accent)] transition" type="button">
                <Icon name="spark" className="h-4 w-4 text-fuchsia-400" />
                <span className="hidden sm:inline">مزامنة الفهرس</span>
                <span className="sm:hidden">مزامنة</span>
              </button>
              <img
                src="/nexora-brand-logo.PNG"
                alt="NEXORA"
                className="h-7 sm:h-8 w-auto object-contain cursor-pointer transition-transform duration-200 hover:scale-105 select-none"
                onClick={() => { navigate("/"); }}
                title="NEXORA الرئيسية"
              />
            </div>
          </header>
          <main className="min-w-0 flex-1"><Outlet /></main>
        </div>
      </div>
    </div>
  );
}
