import Icon from "./Icon.jsx";

export default function TopBar({
  searchQuery,
  onSearchChange,
  onSyncIndex,
  onOpenView,
  syncStatus,
  categoriesCount,
  isSearching,
  backendState
}) {
  const stateTone =
    backendState === "online"
      ? "bg-emerald-400/10 text-emerald-200"
      : backendState === "offline"
        ? "bg-rose-400/10 text-rose-200"
        : "bg-white/10 text-white/70";
  const statusLabel =
    backendState === "online"
      ? "متصل بخادم NEXORA"
      : backendState === "offline"
        ? "الخادم غير متاح"
        : "جارٍ فحص الخادم...";

  return (
    <header className="flex flex-col gap-4 rounded-[2rem] border border-white/10 bg-white/5 p-4 shadow-panel backdrop-blur-2xl xl:flex-row xl:items-center xl:justify-between">
      <div className="flex items-center gap-4">
        <div className={`flex h-12 w-12 items-center justify-center rounded-2xl ${stateTone}`}>
          <Icon name="server" className="h-6 w-6" />
        </div>
        <div className="text-right">
          <p className="text-xs font-semibold text-white/45">طبقة التحكم</p>
          <h2 className="mt-1 text-xl font-semibold text-white">
            {statusLabel}
          </h2>
        </div>
      </div>

      <div className="flex flex-1 flex-col gap-3 xl:flex-row xl:items-center xl:justify-end">
        <div className="flex w-full items-center gap-3 rounded-2xl border border-white/10 bg-black/25 px-4 py-3 xl:max-w-xl">
          <Icon name="search" className="h-4 w-4 text-electric" />
          <input
            value={searchQuery}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder="ابحث عن فيلم أو مسلسل أو اسم..."
            className="w-full bg-transparent text-sm text-white outline-none placeholder:text-white/35"
          />
          {isSearching ? (
            <span className="text-xs font-semibold text-white/40">بحث مباشر</span>
          ) : (
            <span className="text-xs font-semibold text-white/25">{categoriesCount} أقسام</span>
          )}
        </div>

        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={onSyncIndex}
            className="inline-flex items-center gap-2 rounded-2xl border border-electric/30 bg-electric/12 px-4 py-3 text-sm font-semibold text-electric transition hover:bg-electric/18"
          >
            <Icon name="spark" className="h-4 w-4" />
            مزامنة الفهرس
          </button>
          <button
            type="button"
            onClick={() => onOpenView("admin")}
            className="inline-flex items-center gap-2 rounded-2xl border border-white/10 bg-white/[0.05] px-4 py-3 text-sm font-semibold text-white/80 transition hover:bg-white/[0.08]"
          >
            <Icon name="settings" className="h-4 w-4" />
            لوحة الإدارة
          </button>
          <span className="rounded-full border border-white/10 bg-black/25 px-3 py-2 text-xs font-semibold text-white/55">
            {syncStatus || "جاهز"}
          </span>
        </div>
      </div>
    </header>
  );
}
