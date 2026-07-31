import Icon from "./Icon.jsx";
import { navigationItems } from "../data/library.js";

export default function Sidebar({ activeView, categories, health, onNavigate, onOpenCategory }) {
  return (
    <aside className="flex h-full flex-col gap-6 rounded-[2rem] border border-white/10 bg-white/[0.06] p-4 shadow-panel backdrop-blur-2xl">
      <div className="rounded-[1.5rem] border border-white/10 bg-[linear-gradient(145deg,rgba(90,50,244,0.36),rgba(25,183,255,0.12),rgba(255,255,255,0.03))] p-4">
        <div className="flex items-center gap-3">
          <div className="grid h-11 w-11 place-items-center rounded-2xl bg-black/35 text-electric shadow-neon">
            <Icon name="spark" className="h-5 w-5" />
          </div>
          <div className="text-right">
            <p className="text-xs uppercase tracking-[0.45em] text-white/55">NEXORA</p>
            <p className="mt-1 text-lg font-bold">لوحة الوسائط</p>
          </div>
        </div>
        <p className="mt-4 max-w-xs text-sm leading-6 text-white/70">
          غرفة تحكم فخمة لشبكة وسائط محلية، مصممة للسرعة والهدوء البصري وسهولة التشغيل.
        </p>
      </div>

      <nav className="space-y-2">
        {navigationItems.map((item) => {
          const active = activeView === item.id;
          return (
            <button
              key={item.id}
              type="button"
              onClick={() => onNavigate(item.id)}
              className={`flex w-full items-center gap-3 rounded-2xl border px-4 py-3 text-left transition ${
                active
                  ? "border-electric/30 bg-electric/12 text-white shadow-neon"
                  : "border-white/[0.08] bg-white/[0.03] text-white/72 hover:border-white/[0.15] hover:bg-white/[0.06]"
              }`}
            >
              <Icon name={item.icon} className="h-5 w-5 shrink-0" />
              <span className="flex-1 text-right font-medium">{item.label}</span>
            </button>
          );
        })}
      </nav>

      <div className="rounded-[1.5rem] border border-white/10 bg-black/20 p-4">
        <div className="flex items-center justify-between">
          <div className="text-right">
            <p className="text-xs font-semibold text-white/40">الحالة</p>
            <p className="mt-2 text-base font-semibold">الخادم متصل</p>
          </div>
          <div className={`h-3 w-3 rounded-full ${health?.ok ? "bg-emerald-400 shadow-[0_0_18px_rgba(52,211,153,.75)]" : "bg-rose-400"}`} />
        </div>
        <div className="mt-3 space-y-2 text-sm text-white/65">
          <div className="flex items-center justify-between">
            <span>قاعدة البيانات</span>
            <span className={health?.database?.databaseOk ? "text-emerald-300" : "text-rose-300"}>
              {health?.database?.databaseOk ? "سليمة" : "بانتظار الاتصال"}
            </span>
          </div>
          <div className="flex items-center justify-between">
            <span>المفهرس</span>
            <span className="text-cyan-300">متزامن</span>
          </div>
        </div>
      </div>

      <div className="rounded-[1.5rem] border border-white/10 bg-white/[0.03] p-4">
        <div className="flex items-center justify-between">
          <div className="text-right">
            <p className="text-xs font-semibold text-white/40">الأقسام</p>
            <p className="mt-2 text-base font-semibold">{categories.length} قسم</p>
          </div>
          <Icon name="library" className="h-5 w-5 text-electric" />
        </div>
        <div className="mt-4 space-y-2">
          {categories.slice(0, 4).map((category) => (
            <button
              key={category.slug}
              type="button"
              onClick={() => onOpenCategory(category.slug)}
              className="flex w-full items-center justify-between rounded-2xl border border-white/[0.08] bg-black/20 px-4 py-3 text-left transition hover:border-electric/20 hover:bg-white/[0.05]"
            >
              <div className="text-right">
                <p className="font-medium text-white">{category.titleAr}</p>
                <p className="text-xs text-white/45">{category.titleEn}</p>
              </div>
              <div className="text-right">
                <p className="text-sm font-semibold text-white">{category.count}</p>
                <p className="text-[11px] font-semibold text-white/35">عنصر</p>
              </div>
            </button>
          ))}
        </div>
      </div>

      <div className="mt-auto rounded-[1.5rem] border border-white/10 bg-[linear-gradient(145deg,rgba(10,12,22,0.9),rgba(90,50,244,0.12))] p-4">
        <div className="flex items-center gap-3">
          <div className="grid h-12 w-12 place-items-center rounded-2xl border border-white/10 bg-black/35 text-sm font-black text-electric shadow-neon">
            MO
          </div>
          <div className="text-right">
            <p className="text-sm font-semibold text-white">مشرف النظام</p>
            <p className="text-xs text-white/50">إدارة المحتوى والنسخ والبحث</p>
          </div>
        </div>
      </div>
    </aside>
  );
}
