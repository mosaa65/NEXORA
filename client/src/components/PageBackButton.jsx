import Icon from "./Icon.jsx";

/** A visible, reliable return action for pages reached from catalogue cards. */
export default function PageBackButton({ fallback = "/" }) {
  const goBack = () => {
    if (window.history.length > 1) {
      window.history.back();
    } else {
      window.location.hash = `#${fallback}`;
    }
  };

  return (
    <button
      type="button"
      onClick={goBack}
      className="inline-flex min-h-10 items-center gap-1.5 rounded-xl border border-white/20 bg-black/25 px-3.5 text-xs font-bold text-white backdrop-blur-md transition hover:border-white/40 hover:bg-black/40 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
      aria-label="العودة للصفحة السابقة"
    >
      <Icon name="arrowRight" className="h-4 w-4" />
      <span>العودة</span>
    </button>
  );
}
