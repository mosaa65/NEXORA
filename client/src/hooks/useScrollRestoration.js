import { useEffect, useLayoutEffect, useRef } from "react";
import { useNavigationType } from "react-router-dom";
import { useNavigationState } from "../context/NavigationStateContext.jsx";

/**
 * Custom hook to manage scroll restoration and view state persistence for catalogue pages.
 * Ensures instant (0ms) scroll restoration when returning via Back button (POP navigation),
 * while ensuring fresh visits (PUSH/REPLACE) always start cleanly at the top (0, 0).
 */
export function useScrollRestoration(pageKey, isDataReady = true, customData = {}) {
  const { setActiveKey, savePageState, getPageState } = useNavigationState();
  const navType = useNavigationType();
  const hasRestoredRef = useRef(false);
  const customDataRef = useRef(customData);
  customDataRef.current = customData;

  // Set active key for continuous scroll listener
  useEffect(() => {
    setActiveKey(pageKey);
    return () => {
      setActiveKey("");
    };
  }, [pageKey, setActiveKey]);

  // Restore scroll position ONLY when returning back (POP navigation)
  useLayoutEffect(() => {
    if (!pageKey || !isDataReady || hasRestoredRef.current) return;

    if (navType === "POP") {
      const cached = getPageState(pageKey);
      if (cached && typeof cached.scrollY === "number" && cached.scrollY > 0) {
        hasRestoredRef.current = true;
        window.requestAnimationFrame(() => {
          window.scrollTo({ top: cached.scrollY, behavior: "instant" });
          window.requestAnimationFrame(() => {
            window.scrollTo({ top: cached.scrollY, behavior: "instant" });
          });
        });
        return;
      }
    }

    // On fresh visits (PUSH or REPLACE) or when no saved scroll: ensure top of page
    hasRestoredRef.current = true;
    window.scrollTo({ top: 0, left: 0, behavior: "instant" });
  }, [pageKey, isDataReady, getPageState, navType]);

  // Save state on unmount
  useEffect(() => {
    return () => {
      if (pageKey) {
        savePageState(pageKey, {
          scrollY: window.scrollY,
          ...customDataRef.current,
        });
      }
    };
  }, [pageKey, savePageState]);
}
