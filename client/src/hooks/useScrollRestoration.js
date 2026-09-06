import { useEffect, useLayoutEffect, useRef } from "react";
import { useNavigationType } from "react-router-dom";
import { useNavigationState } from "../context/NavigationStateContext.jsx";

/**
 * Custom hook to manage scroll restoration and view state persistence for catalogue pages.
 *
 * Strategy:
 * - POP (Back/Forward): Restore saved scroll position once data is ready.
 * - PUSH/REPLACE (fresh navigation): Always scroll to top (0,0).
 *
 * Key fix: hasRestoredRef resets on mount so each page entry gets one restoration attempt.
 * We defer POP restoration until isDataReady=true so infinite-scroll pages
 * with lazy-loaded content are scrolled to the right position AFTER data loads.
 */
export function useScrollRestoration(pageKey, isDataReady = true, customData = {}) {
  const { setActiveKey, savePageState, getPageState } = useNavigationState();
  const navType = useNavigationType();
  // Reset to false on every mount (new page entry)
  const hasRestoredRef = useRef(false);
  const customDataRef = useRef(customData);
  customDataRef.current = customData;

  const lastPositiveScrollYRef = useRef(0);

  // Continuously track positive user scroll position
  useEffect(() => {
    const onScroll = () => {
      if (window.scrollY > 0) {
        lastPositiveScrollYRef.current = window.scrollY;
      }
    };
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  // Set active key for continuous scroll listener
  useEffect(() => {
    setActiveKey(pageKey);
    return () => {
      setActiveKey("");
    };
  }, [pageKey, setActiveKey]);

  // Save scroll position immediately before the component unmounts
  useEffect(() => {
    return () => {
      if (pageKey) {
        const scrollY = window.scrollY > 0 ? window.scrollY : lastPositiveScrollYRef.current;
        savePageState(pageKey, {
          scrollY,
          ...customDataRef.current,
        });
      }
    };
  }, [pageKey, savePageState]);

  // Restore scroll on POP, or go to top on PUSH/REPLACE.
  // Uses useEffect (after paint) so that outgoing views (like MediaDetailsPage)
  // are never scrolled down before unmounting.
  useEffect(() => {
    if (!pageKey || hasRestoredRef.current) return;

    if (navType === "POP") {
      // Defer until data is available (avoids scrolling into empty skeleton)
      if (!isDataReady) return;

      const cached = getPageState(pageKey);
      if (cached && typeof cached.scrollY === "number" && cached.scrollY > 0) {
        hasRestoredRef.current = true;
        const targetY = cached.scrollY;
        // Defer to animation frame after outgoing page is removed from paint tree
        window.requestAnimationFrame(() => {
          window.scrollTo({ top: targetY, behavior: "instant" });
        });
        return;
      }
    }

    // PUSH/REPLACE or no saved position → scroll to top immediately (no data wait)
    hasRestoredRef.current = true;
    window.scrollTo({ top: 0, left: 0, behavior: "instant" });
  }, [pageKey, isDataReady, getPageState, navType]);
}
