import React from "react";
import Icon from "../Icon.jsx";
import "./navigation.css";

/** Shared responsive navigation frame for the customer lounge and admin portal. */
export default function SideNavigation({
  isOpen,
  onClose,
  isCollapsed = false,
  onToggleCollapse,
  label,
  subtitle,
  emblem = "spark",
  tone = "cinema",
  children,
  footer,
}) {
  const isAdmin = tone === "admin";

  return (
    <>
      {isOpen && (
        <button
          className="navigation-backdrop"
          type="button"
          aria-label="إغلاق القائمة"
          onClick={onClose}
        />
      )}
      <aside
        className={`side-navigation side-navigation--${tone} ${isOpen ? "is-open" : ""} ${
          isCollapsed ? "is-collapsed" : ""
        }`}
        aria-label={label || "NEXORA"}
      >
        <div className="side-navigation__inner">
          <header className="side-navigation__brand">
            <div className="side-navigation__brand-copy">
              <img
                className="side-navigation__logo"
                src="/nexora-brand-logo.PNG"
                alt="NEXORA"
                title="NEXORA"
              />
              {isAdmin && (
                <span className="side-navigation__brand-text">
                  <small className="side-navigation__eyebrow">NEXORA NETWORK</small>
                  <strong>{label}</strong>
                  <small>{subtitle}</small>
                </span>
              )}
            </div>

            {/* Close Button on Mobile */}
            <button
              className="navigation-icon-button navigation-close-button"
              type="button"
              aria-label="إغلاق القائمة"
              onClick={onClose}
            >
              <Icon name="close" className="h-5 w-5" />
            </button>
          </header>

          <div className="side-navigation__scroll">{children}</div>

          {footer && !isCollapsed && (
            <footer className="side-navigation__footer">{footer}</footer>
          )}
        </div>
      </aside>
    </>
  );
}
