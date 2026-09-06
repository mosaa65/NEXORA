import React, { createContext, useContext, useRef, useCallback, useEffect } from "react";

const NavigationStateContext = createContext(null);

const SESSION_PREFIX = "nexora:nav-state:";

export function NavigationStateProvider({ children }) {
  // In-memory fast cache for instant (0ms) state and scroll restoration
  const memoryCache = useRef(new Map());
  const activeKeyRef = useRef("");

  // Track scroll position continuously
  useEffect(() => {
    let ticking = false;
    const handleScroll = () => {
      if (!ticking) {
        window.requestAnimationFrame(() => {
          if (activeKeyRef.current) {
            const current = memoryCache.current.get(activeKeyRef.current);
            if (current) {
              current.scrollY = window.scrollY;
            }
          }
          ticking = false;
        });
        ticking = true;
      }
    };

    window.addEventListener("scroll", handleScroll, { passive: true });
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  const setActiveKey = useCallback((key) => {
    activeKeyRef.current = key || "";
  }, []);

  const savePageState = useCallback((key, state) => {
    if (!key) return;
    const existing = memoryCache.current.get(key) || {};
    const updated = {
      ...existing,
      ...state,
      scrollY: state.scrollY !== undefined ? state.scrollY : (window.scrollY || existing.scrollY || 0),
      savedAt: Date.now(),
    };
    memoryCache.current.set(key, updated);

    // Also persist non-massive metadata to sessionStorage for resilience
    try {
      const lightweight = {
        scrollY: updated.scrollY,
        filters: updated.filters,
        sort: updated.sort,
        page: updated.page,
        totalCount: updated.totalCount,
        savedAt: updated.savedAt,
      };
      sessionStorage.setItem(`${SESSION_PREFIX}${key}`, JSON.stringify(lightweight));
    } catch {}
  }, []);

  const getPageState = useCallback((key) => {
    if (!key) return null;
    const memory = memoryCache.current.get(key);
    if (memory) return memory;

    try {
      const saved = JSON.parse(sessionStorage.getItem(`${SESSION_PREFIX}${key}`));
      if (saved) return saved;
    } catch {}

    return null;
  }, []);

  const clearPageState = useCallback((key) => {
    if (key) {
      memoryCache.current.delete(key);
      try { sessionStorage.removeItem(`${SESSION_PREFIX}${key}`); } catch {}
    } else {
      memoryCache.current.clear();
      try {
        Object.keys(sessionStorage)
          .filter((k) => k.startsWith(SESSION_PREFIX))
          .forEach((k) => sessionStorage.removeItem(k));
      } catch {}
    }
  }, []);

  return (
    <NavigationStateContext.Provider
      value={{
        setActiveKey,
        savePageState,
        getPageState,
        clearPageState,
      }}
    >
      {children}
    </NavigationStateContext.Provider>
  );
}

export function useNavigationState() {
  const context = useContext(NavigationStateContext);
  if (!context) {
    throw new Error("useNavigationState must be used within a NavigationStateProvider");
  }
  return context;
}
