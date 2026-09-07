import { useCallback, useEffect, useState } from "react";

/**
 * View mode hook for the catalogue grid/list surfaces.
 *
 * Allows a small set of visual modes (mirroring Windows 11 File Explorer):
 * "extralarge", "large", "medium" (all grid densities) plus "list" and "details".
 *
 * Persists the user's last choice in localStorage so it survives navigation
 * and page reloads (matching the existing `nexora_*` persistence pattern).
 *
 * @param {string} [storageKey="nexora_view_mode"] localStorage key.
 * @param {string} [defaultMode="medium"] Fallback when nothing is stored.
 */
export default function useViewMode(storageKey = "nexora_view_mode", defaultMode = "medium") {
  const VALID = ["extralarge", "large", "medium", "list", "details"];

  const [mode, setMode] = useState(() => {
    if (typeof window === "undefined") return defaultMode;
    try {
      const stored = window.localStorage.getItem(storageKey);
      if (stored && VALID.includes(stored)) return stored;
    } catch {}
    return defaultMode;
  });

  useEffect(() => {
    try {
      window.localStorage.setItem(storageKey, mode);
    } catch {}
  }, [mode, storageKey]);

  const isGrid = mode === "extralarge" || mode === "large" || mode === "medium";
  const isList = mode === "list";
  const isDetails = mode === "details";

  const setModeSafe = useCallback(
    (next) => {
      if (VALID.includes(next)) setMode(next);
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    []
  );

  return { mode, setMode: setModeSafe, isGrid, isList, isDetails };
}
