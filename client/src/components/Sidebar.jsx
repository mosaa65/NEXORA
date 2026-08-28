import React from "react";
import Icon from "./Icon.jsx";
import SideNavigation from "./layout/SideNavigation.jsx";

const cinemaNavItems = [
  { id: "dashboard", label: "الرئيسية", icon: "dashboard" },
  { id: "movies", label: "الأفلام السينمائية", icon: "film" },
  { id: "series", label: "المسلسلات والدراما", icon: "tv" },
  { id: "anime", label: "الأنمي والرسوم اليابانية", icon: "spark" },
  { id: "kids", label: "الأطفال والكرتون العائلي", icon: "smile" },
  { id: "family", label: "العائلة والسينما العائلية", icon: "smile" },
  { id: "documentaries", label: "الوثائقيات والمعرفة", icon: "book" },
  { id: "plays", label: "المسرحيات والكوميديا", icon: "mask" },
  { id: "ramadan", label: "الرمضانيات والإنتاج الرمضاني", icon: "spark" },
  { id: "wrestling", label: "المصارعة الحرة والرياضة", icon: "shield" },
  { id: "music", label: "المكتبة الصوتية والموسيقى", icon: "music" },
  { id: "apps", label: "البرامج والتطبيقات", icon: "grid" },
  { id: "favorites", label: "المفضلة", icon: "star" },
];

export default function Sidebar({
  isOpen,
  onClose,
  isCollapsed = false,
  onToggleCollapse,
  activeView,
  onNavigate,
}) {
  return (
    <SideNavigation
      isOpen={isOpen}
      onClose={onClose}
      isCollapsed={isCollapsed}
      onToggleCollapse={onToggleCollapse}
      label="NEXORA"
      tone="cinema"
    >
      <section className="navigation-group" aria-labelledby="cinema-navigation-label">
        <span id="cinema-navigation-label" className="navigation-group__label">
          استكشف المكتبة
        </span>
        <nav>
          <ul className="navigation-list">
            {cinemaNavItems.map((item) => {
              const isActive = activeView === item.id;
              return (
                <li key={item.id}>
                  <button
                    type="button"
                    className={`navigation-item navigation-item--${item.id} ${
                      isActive ? "is-active" : ""
                    }`}
                    aria-current={isActive ? "page" : undefined}
                    onClick={() => onNavigate(item.id)}
                    data-tooltip={item.label}
                    title={item.label}
                  >
                    <span className="navigation-item__icon">
                      <Icon name={item.icon} />
                    </span>
                    <span className="truncate">{item.label}</span>
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
