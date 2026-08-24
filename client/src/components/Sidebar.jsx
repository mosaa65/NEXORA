import Icon from "./Icon.jsx";
import SideNavigation from "./layout/SideNavigation.jsx";
import useTheme from "../hooks/useTheme.js";

const cinemaNavItems = [
  { id: "dashboard", label: "الرئيسية", icon: "dashboard" },
  { id: "movies", label: "الأفلام", icon: "film" },
  { id: "series", label: "المسلسلات", icon: "tv" },
  { id: "anime", label: "الأنمي", icon: "spark" },
  { id: "kids", label: "الأطفال والكرتون", icon: "smile" },
  { id: "documentaries", label: "الوثائقيات", icon: "book" },
  { id: "plays", label: "المسرحيات", icon: "mask" },
  { id: "favorites", label: "المفضلة", icon: "star" },
];

export default function Sidebar({ isOpen, onClose, activeView, onNavigate }) {
  const { theme, toggleTheme, font, setFont } = useTheme();

  return (
    <SideNavigation
      isOpen={isOpen}
      onClose={onClose}
      label="مكتبتي"
      subtitle="تجربة مشاهدة خاصة"
      emblem="play"
      footer={
        <>
          <div className="navigation-footer-card">
            <div className="navigation-footer-card__copy">
              <strong>مكتبة الاستراحة</strong>
              <small>محتوى محلي وبث سريع عبر الشبكة</small>
            </div>
            <span className="navigation-status">متصل</span>
          </div>
          <div className="navigation-footer-actions">
            <button className="navigation-theme-button" type="button" onClick={toggleTheme} aria-label={theme === "dark" ? "تفعيل المظهر الفاتح" : "تفعيل المظهر الداكن"}>
              <Icon name={theme === "dark" ? "sun" : "moon"} className="h-4 w-4" />
              {theme === "dark" ? "مظهر فاتح" : "مظهر داكن"}
            </button>
            <button className="navigation-theme-button" type="button" onClick={() => setFont(font === "plex" ? "cairo" : "plex")} aria-label="تبديل خط الواجهة">
              <span className="text-sm font-black">ع</span>
              {font === "plex" ? "خط القاهرة" : "خط Plex"}
            </button>
          </div>
        </>
      }
    >
      <section className="navigation-group" aria-labelledby="cinema-navigation-label">
        <span id="cinema-navigation-label" className="navigation-group__label">استكشف المكتبة</span>
        <nav>
          <ul className="navigation-list">
            {cinemaNavItems.map((item) => {
              const isActive = activeView === item.id;
              return (
                <li key={item.id}>
                  <button type="button" className={`navigation-item navigation-item--${item.id} ${isActive ? "is-active" : ""}`} aria-current={isActive ? "page" : undefined} onClick={() => onNavigate(item.id)}>
                    <span className="navigation-item__icon"><Icon name={item.icon} /></span>
                    <span>{item.label}</span>
                  </button>
                </li>
              );
            })}
          </ul>
        </nav>
      </section>
    </SideNavigation>
  );
}
