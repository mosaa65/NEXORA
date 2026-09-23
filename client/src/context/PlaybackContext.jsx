import { createContext, useCallback, useContext, useMemo, useState } from "react";

const PlaybackContext = createContext(null);

/**
 * PlaybackProvider — owns the global "mini player" (floating picture-in-picture)
 * state so that a minimised video keeps playing across page navigation.
 *
 * The player element itself is rendered once by `MiniPlayerDock`, mounted ABOVE
 * the router's routes. Because it never unmounts on navigation, switching pages
 * does not tear the video down; the WatchPage hands its playback payload here
 * when the user minimises and stops rendering its own copy.
 */
export function PlaybackProvider({ children }) {
  const [payload, setPayload] = useState(null);
  const [expanded, setExpanded] = useState(false);

  // Enter the dock with a complete payload describing what to play.
  const minimize = useCallback((next) => {
    if (!next?.src) return;
    setPayload(next);
    setExpanded(false);
  }, []);

  // Leave the dock entirely (stop playback).
  const close = useCallback(() => {
    setPayload(null);
    setExpanded(false);
  }, []);

  const value = useMemo(
    () => ({
      payload,
      // A payload present means the dock is showing.
      isActive: Boolean(payload?.src),
      expanded,
      setExpanded,
      minimize,
      close,
    }),
    [payload, expanded, minimize, close]
  );

  return <PlaybackContext.Provider value={value}>{children}</PlaybackContext.Provider>;
}

export function usePlayback() {
  const ctx = useContext(PlaybackContext);
  if (!ctx) throw new Error("usePlayback must be used within a PlaybackProvider");
  return ctx;
}
