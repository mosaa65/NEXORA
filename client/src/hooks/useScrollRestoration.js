import { useEffect, useLayoutEffect, useRef } from "react";
import { useNavigationState } from "../context/NavigationStateContext.jsx";

/**
 * Custom hook to manage scroll restoration and view state persistence for catalogue pages.
 * Ensures instant (0ms) scroll restoration without jumping or flicker when navigating back.
 */
export function useScrollRestoration(pageKey, isDataReady = true, customData = {}) {
  const { setActiveKey, savePageState, getPageState } = useNavigationState();
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

  // Restore scroll position as soon as data is ready in the DOM
  useLayoutEffect(() => {
    if (!pageKey || !isDataReady) return;

    const cached = getPageState(pageKey);
    if (cached && typeof cached.scrollY === "number" && cached.scrollY > 0 && !hasRestoredRef.current) {
      hasRestoredRef.current = true;
      // Use double RAF to ensure DOM layout and images have computed heights
      window.requestAnimationFrame(() => {
        window.scrollTo({ top: cached.scrollY, behavior: "instant" });
        window.requestAnimationFrame(() => {
          window.scrollTo({ top: cached.scrollY, behavior: "instant" });
        });
      });
    }
  }, [pageKey, isDataReady, getPageState]);

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
