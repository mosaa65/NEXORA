import React, { createContext, useCallback, useEffect, useState } from "react";

const STORAGE_KEY = "nexora_theme";
const FONT_STORAGE_KEY = "nexora_font";
const VALID_THEMES = ["dark", "light"];
const VALID_FONTS = ["plex", "cairo"];

export const ThemeContext = createContext({
  theme: "dark",
  font: "plex",
  setTheme: () => {},
  setFont: () => {},
  toggleTheme: () => {},
});

/**
 * ThemeProvider — wraps the app and manages dark/light mode.
 * Persists choice in localStorage and applies `data-theme` to <html>.
 */
export function ThemeProvider({ children }) {
  const [theme, setThemeState] = useState(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored && VALID_THEMES.includes(stored)) return stored;
    } catch {}
    return "dark";
  });
  const [font, setFontState] = useState(() => {
    try {
      const stored = localStorage.getItem(FONT_STORAGE_KEY);
      if (stored && VALID_FONTS.includes(stored)) return stored;
    } catch {}
    return "plex";
  });

  // Apply theme to document
  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
    try {
      localStorage.setItem(STORAGE_KEY, theme);
    } catch {}
  }, [theme]);

  useEffect(() => {
    document.documentElement.setAttribute("data-font", font);
    try { localStorage.setItem(FONT_STORAGE_KEY, font); } catch {}
  }, [font]);

  const setTheme = useCallback((t) => {
    if (VALID_THEMES.includes(t)) setThemeState(t);
  }, []);
  const setFont = useCallback((value) => {
    if (VALID_FONTS.includes(value)) setFontState(value);
  }, []);

  const toggleTheme = useCallback(() => {
    setThemeState((prev) => (prev === "dark" ? "light" : "dark"));
  }, []);

  return (
    <ThemeContext.Provider value={{ theme, setTheme, toggleTheme, font, setFont }}>
      {children}
    </ThemeContext.Provider>
  );
}
