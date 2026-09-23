/**
 * NexoraPlayer — the player module's public entry point.
 *
 * The implementation lives in `components/player/` (engine host, controls, scrub,
 * settings, overlays, progress hook). This file remains the import path the app
 * already uses, so no consumer changed when the component was split.
 */
export { default } from "./player/NexoraPlayer.jsx";
export { progressKeyFor, readSavedProgress } from "./player/usePlayerProgress.js";
