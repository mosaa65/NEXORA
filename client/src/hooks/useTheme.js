import { useContext } from "react";
import { ThemeContext } from "../context/ThemeContext.jsx";

/**
 * Custom hook for accessing theme state.
 *
 * Usage:
 *   const { theme, toggleTheme } = useTheme();
 */
export default function useTheme() {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be used within a ThemeProvider");
  }
  return ctx;
}
