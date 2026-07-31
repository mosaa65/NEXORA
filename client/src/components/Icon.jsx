const paths = {
  dashboard: "M4 6h7v7H4z M13 6h7v4h-7z M13 12h7v6h-7z M4 15h7v3H4z",
  library: "M5 5h6v14H5z M13 5h6v14h-6z M4 7h1v10H4z M19 7h1v10h-1z",
  play: "M9 7l8 5-8 5V7z M5 5h2v14H5z",
  settings:
    "M11.5 3.5h1l.7 2.2c.4.1.8.3 1.1.5l2.1-1 1.4 1.4-1 2.1c.2.3.4.7.5 1.1l2.2.7v2l-2.2.7c-.1.4-.3.8-.5 1.1l1 2.1-1.4 1.4-2.1-1c-.3.2-.7.4-1.1.5l-.7 2.2h-1l-.7-2.2c-.4-.1-.8-.3-1.1-.5l-2.1 1-1.4-1.4 1-2.1c-.2-.3-.4-.7-.5-1.1l-2.2-.7v-2l2.2-.7c.1-.4.3-.8.5-1.1l-1-2.1 1.4-1.4 2.1 1c.3-.2.7-.4 1.1-.5z M12 9.5A2.5 2.5 0 1 0 12 14a2.5 2.5 0 0 0 0-5z",
  search: "M11 4a7 7 0 1 0 4.3 12.5l4.1 4.1 1.4-1.4-4.1-4.1A7 7 0 0 0 11 4zm0 2a5 5 0 1 1 0 10 5 5 0 0 1 0-10z",
  server:
    "M4 7.5C4 6.1 6.7 5 10 5s6 .9 6 2.5S13.3 10 10 10 4 8.9 4 7.5zm0 5C4 11.1 6.7 10 10 10s6 .9 6 2.5S13.3 15 10 15s-6-1.1-6-2.5zm0 5C4 16.1 6.7 15 10 15s6 .9 6 2.5S13.3 20 10 20s-6-1.1-6-2.5z",
  spark:
    "M12 2l1.5 5.5L19 9l-5.5 1.5L12 16l-1.5-5.5L5 9l5.5-1.5L12 2z M4 15l.8 2.5L7 18.3 4.8 19l-.8 2.5L3.2 19 1 18.3l2.2-.8L4 15z",
  bell:
    "M10 19a2 2 0 0 0 4 0h-4z M12 4a5 5 0 0 0-5 5c0 4-2 5-2 5h14s-2-1-2-5a5 5 0 0 0-5-5z",
  disk:
    "M4 6h16v4H4z M4 12h16v6H4z M7 8a1 1 0 1 0 0-2 1 1 0 0 0 0 2zm0 6a1 1 0 1 0 0-2 1 1 0 0 0 0 2z",
  arrowLeft: "M19 12H7m5-5-5 5 5 5",
  arrowRight: "M5 12h12m-5-5 5 5-5 5"
};

export default function Icon({ name, className = "" }) {
  const d = paths[name] || paths.spark;
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.75"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden="true"
    >
      <path d={d} />
    </svg>
  );
}
