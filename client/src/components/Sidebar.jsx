import React from "react";
import Icon from "./Icon.jsx";
import SideNavigation from "./layout/SideNavigation.jsx";

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
