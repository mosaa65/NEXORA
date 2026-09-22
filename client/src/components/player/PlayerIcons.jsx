/**
 * PlayerIcons — the icon set the player surface draws with.
 *
 * These are inline SVG rather than the shared `Icon` component because the player
 * needs filled transport glyphs (play/pause/skip) at several sizes, which the
 * stroke-only catalogue icon set does not provide. Everything here uses the same
 * 24×24 viewBox, 2px stroke and round caps as `Icon.jsx` so the two families read
 * as one system.
 */
function Base({ className = "h-5 w-5", children, filled = false }) {
  return (
    <svg
      viewBox="0 0 24 24"
      className={className}
      aria-hidden="true"
      fill={filled ? "currentColor" : "none"}
      stroke={filled ? "none" : "currentColor"}
      strokeWidth={filled ? undefined : 2}
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      {children}
    </svg>
  );
}

const SHAPES = {
  play: (className) => (
    <Base className={className} filled>
      <path d="M8 5.4v13.2c0 .78.86 1.26 1.53.86l10.2-6.6a1 1 0 000-1.72L9.53 4.54A1 1 0 008 5.4z" />
    </Base>
  ),
  pause: (className) => (
    <Base className={className} filled>
      <rect x="7" y="5" width="3.5" height="14" rx="1" />
      <rect x="13.5" y="5" width="3.5" height="14" rx="1" />
    </Base>
  ),
  rewind: (className) => (
    <Base className={className}>
      <path d="M4 10a8.5 8.5 0 1 1 1.35 7.1" />
      <path d="M4 4.5V10h5.5" />
    </Base>
  ),
  forward: (className) => (
    <Base className={className}>
      <path d="M20 10a8.5 8.5 0 1 0-1.35 7.1" />
      <path d="M20 4.5V10h-5.5" />
    </Base>
  ),
  skipNext: (className) => (
    <Base className={className} filled>
      <path d="M6 5.4v13.2c0 .78.86 1.26 1.53.86l8.2-6.6a1 1 0 000-1.72L7.53 4.54A1 1 0 006 5.4z" />
      <rect x="17" y="5" width="2.2" height="14" rx="1" />
    </Base>
  ),
  skipPrev: (className) => (
    <Base className={className} filled>
      <path d="M18 5.4v13.2c0 .78-.86 1.26-1.53.86l-8.2-6.6a1 1 0 010-1.72l8.2-6.6A1 1 0 0118 5.4z" />
      <rect x="4.8" y="5" width="2.2" height="14" rx="1" />
    </Base>
  ),
  volume: (className) => (
    <Base className={className}>
      <path d="M4 10v4h4l5 4V6L8 10H4z" />
      <path d="M16 9a4 4 0 010 6M18.5 6.5a7.5 7.5 0 010 11" />
    </Base>
  ),
  mute: (className) => (
    <Base className={className}>
      <path d="M4 10v4h4l5 4V6L8 10H4zM17 10l4 4m0-4l-4" />
    </Base>
  ),
  pip: (className) => (
    <Base className={className}>
      <rect x="3.5" y="5" width="17" height="14" rx="2" />
      <rect x="12.5" y="12" width="5" height="4" rx=".7" fill="currentColor" stroke="none" />
    </Base>
  ),
  fullscreen: (className) => (
    <Base className={className}>
      <path d="M8 3H3v5m13-5h5v5M8 21H3v-5m13 5h5v-5" />
    </Base>
  ),
  fullscreenExit: (className) => (
    <Base className={className}>
      <path d="M3 8h5V3m8 5h5V3M3 16h5v5m8-5h5v5" />
    </Base>
  ),
  captions: (className) => (
    <Base className={className}>
      <rect x="3" y="5" width="18" height="14" rx="2.5" />
      <path d="M10 10.5a2.5 2.5 0 100 3M17 10.5a2.5 2.5 0 100 3" />
    </Base>
  ),
  settings: (className) => (
    <Base className={className}>
      <circle cx="12" cy="12" r="3" />
      <path d="M12 3.5v2M12 18.5v2M4.9 7.5l1.7 1M17.4 15.5l1.7 1M4.9 16.5l1.7-1M17.4 8.5l1.7-1" />
    </Base>
  ),
  playlist: (className) => (
    <Base className={className}>
      <path d="M5 6h14M5 12h14M5 18h9" />
      <path d="M18 16v5m-2.5-2.5h5" />
    </Base>
  ),
  check: (className) => (
    <Base className={className}>
      <path d="M5 12.5l4.5 4.5L19 7" />
    </Base>
  ),
  close: (className) => (
    <Base className={className}>
      <path d="M6 6l12 12M18 6L6 18" />
    </Base>
  ),
  minimize: (className) => (
    <Base className={className}>
      <path d="M5 12h14" />
    </Base>
  ),
  quality: (className) => (
    <Base className={className}>
      <rect x="3" y="6" width="18" height="12" rx="2" />
      <path d="M8 10v4M16 10v4M11 12h2" />
    </Base>
  ),
  audio: (className) => (
    <Base className={className}>
      <path d="M9 18V5l11-2v13" />
      <path d="M9 18a3 3 0 11-6 0 3 3 0 016 0z" />
    </Base>
  ),
  next: (className) => (
    <Base className={className} filled>
      <path d="M6 5.4v13.2c0 .78.86 1.26 1.53.86l8.2-6.6a1 1 0 000-1.72L7.53 4.54A1 1 0 006 5.4z" />
      <rect x="17" y="5" width="2.2" height="14" rx="1" />
    </Base>
  ),
  info: (className) => (
    <Base className={className}>
      <circle cx="12" cy="12" r="9" />
      <path d="M12 11v6M12 7.5h.01" />
    </Base>
  ),
  alert: (className) => (
    <Base className={className}>
      <path d="M12 3l9.5 16.5H2.5L12 3z" />
      <path d="M12 9.5v4.5M12 17h.01" />
    </Base>
  ),
};

/**
 * Render one player icon by name. Unknown names render nothing rather than a
 * fallback glyph, so a typo shows up as a missing icon instead of the wrong one.
 */
export default function PlayerIcon({ name, className = "h-5 w-5" }) {
  const shape = SHAPES[name];
  return shape ? shape(className) : null;
}