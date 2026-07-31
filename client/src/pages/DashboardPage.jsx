import GlassCard from "../components/GlassCard.jsx";
import MediaCard from "../components/MediaCard.jsx";
import MetricCard from "../components/MetricCard.jsx";
import Icon from "../components/Icon.jsx";
import { buildHeroCopy, dashboardMetrics, mockActivity, serviceItems } from "../data/library.js";

export default function DashboardPage({
  categories,
  featured,
  searchResults,
  health,
  onOpenMedia,
  onOpenCategory,
  onSyncIndex,
  syncStatus
}) {
  const hero = buildHeroCopy();
  const online = health === null ? null : Boolean(health?.ok);
  const healthTone =
    online === null ? "bg-white/20" : online ? "bg-emerald-400 shadow-[0_0_20px_rgba(52,211,153,.8)]" : "bg-rose-400";
  const healthLabel = online === null ? "جارٍ التحقق" : online ? "متصل" : "غير متصل";
  const primaryPick = featured[0] || searchResults[0];
  const secondaryPick = featured[1] || searchResults[1] || primaryPick;

  return (
    <div className="space-y-6">
      <section className="grid gap-6 xl:grid-cols-[1.25fr_0.9fr]">
        <GlassCard className="relative overflow-hidden p-0">
          <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(90,50,244,0.38),transparent_28%),radial-gradient(circle_at_bottom_right,rgba(25,183,255,0.22),transparent_32%),linear-gradient(135deg,rgba(5,8,18,0.96),rgba(12,14,28,0.82))]" />
          <div className="absolute -right-20 top-8 h-64 w-64 rounded-full bg-electric/18 blur-3xl" />
          <div className="absolute bottom-0 left-0 h-56 w-56 rounded-full bg-royal/20 blur-3xl" />

          <div className="relative grid gap-6 p-6 lg:grid-cols-[1.05fr_0.95fr] lg:p-8">
            <div className="flex flex-col justify-between gap-6 text-right">
              <div className="max-w-3xl">
                <p className="text-xs font-semibold text-electric/90">{hero.eyebrow}</p>
                <h1 className="mt-4 text-4xl font-black leading-tight text-white md:text-6xl">
                  {hero.title}
                </h1>
                <p className="mt-5 max-w-2xl text-base leading-8 text-white/70 md:text-lg">
                  {hero.subtitle}
                </p>
              </div>

              <div className="flex flex-wrap justify-end gap-3">
                <button
                  type="button"
                  onClick={onSyncIndex}
                  className="inline-flex items-center justify-center gap-2 rounded-2xl border border-electric/25 bg-electric/12 px-4 py-3 font-semibold text-electric transition hover:bg-electric/18"
                >
                  <Icon name="spark" className="h-4 w-4" />
                  مزامنة الفهرس
                </button>
                <span className="inline-flex items-center rounded-2xl border border-white/10 bg-black/25 px-4 py-3 text-sm font-semibold text-white/70">
                  {syncStatus || "جاهز"}
                </span>
                <button
                  type="button"
                  onClick={() => onOpenCategory("anime")}
                  className="inline-flex items-center justify-center gap-2 rounded-2xl border border-white/10 bg-white/[0.05] px-4 py-3 font-semibold text-white/80 transition hover:bg-white/[0.08]"
                >
                  <Icon name="library" className="h-4 w-4" />
                  افتح قسم الأنمي
                </button>
              </div>

              <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                {dashboardMetrics.map((metric) => (
                  <MetricCard key={metric.label} {...metric} />
                ))}
              </div>
            </div>

            <button
              type="button"
              onClick={() => primaryPick && onOpenMedia(primaryPick)}
              className="group relative min-h-[26rem] overflow-hidden rounded-[2rem] border border-white/10 bg-white/[0.04] text-left shadow-panel transition hover:-translate-y-1 hover:border-electric/30 hover:shadow-neon"
            >
              <div
                className="absolute inset-0 transition duration-700 group-hover:scale-[1.03]"
                style={{ backgroundImage: primaryPick?.gradient || "linear-gradient(135deg, rgba(90,50,244,0.7), rgba(25,183,255,0.45))" }}
              />
              <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(255,255,255,0.18),transparent_30%),radial-gradient(circle_at_bottom_left,rgba(25,183,255,0.16),transparent_32%),linear-gradient(180deg,rgba(4,7,18,0.08),rgba(4,7,18,0.84))]" />
              <div className="absolute inset-0 ring-1 ring-inset ring-white/10" />

              <div className="relative flex h-full flex-col justify-between p-5">
                <div className="flex items-start justify-between gap-3">
                  <div className="rounded-full border border-white/12 bg-black/28 px-3 py-2 text-[11px] font-semibold text-white/75">
                    العمل الأبرز
                  </div>
                  <div className="rounded-full border border-white/12 bg-black/28 px-3 py-2 text-xs text-white/75">
                    {primaryPick?.resolution || "4K"}
                  </div>
                </div>

                <div className="rounded-[1.75rem] border border-white/10 bg-black/22 p-5 text-right backdrop-blur-sm">
                  <p className="text-xs font-semibold text-white/45">{primaryPick?.titleEn || "Featured Title"}</p>
                  <h3 className="mt-2 text-3xl font-black leading-tight text-white">{primaryPick?.titleAr || "عنوان بارز"}</h3>
                  <p className="mt-3 line-clamp-3 text-sm leading-6 text-white/72">
                    {primaryPick?.plot || "بطاقة سينمائية مميزة تعرض العمل الأكثر جذبًا داخل المكتبة."}
                  </p>

                  <div className="mt-4 flex flex-wrap justify-end gap-2">
                    {(primaryPick?.highlights || ["سينمائي", "مميز", "فوري"]).slice(0, 3).map((highlight) => (
                      <span
                        key={highlight}
                        className="rounded-full border border-white/10 bg-white/[0.05] px-3 py-1 text-[11px] font-semibold text-white/70"
                      >
                        {highlight}
                      </span>
                    ))}
                  </div>
                </div>

                <div className="grid gap-3 sm:grid-cols-2">
                  <div className="rounded-2xl border border-white/10 bg-black/28 px-4 py-3 text-right">
                    <p className="text-xs font-semibold text-white/35">المتابعة التالية</p>
                    <p className="mt-2 text-sm font-semibold text-white">
                      {secondaryPick?.titleAr || "عنوان إضافي"}
                    </p>
                    <p className="text-xs font-semibold text-white/45">{secondaryPick?.titleEn || "Next pick"}</p>
                  </div>
                  <div className="rounded-2xl border border-white/10 bg-black/28 px-4 py-3 text-right">
                    <p className="text-xs font-semibold text-white/35">الحالة الحية</p>
                    <p className="mt-2 text-sm font-semibold text-white">{healthLabel}</p>
                    <p className="text-xs text-white/45">قناة التشغيل المحلية</p>
                  </div>
                </div>
              </div>
            </button>
          </div>
        </GlassCard>

        <div className="space-y-6">
          <GlassCard className="p-6">
            <div className="flex items-center justify-between">
              <div className="text-right">
                <p className="text-xs font-semibold text-white/40">الأنظمة الحية</p>
                <h2 className="mt-2 text-2xl font-bold text-white">نبض التشغيل</h2>
              </div>
              <div className={`h-3 w-3 rounded-full ${healthTone}`} />
            </div>
            <div className="mt-5 space-y-3">
              {serviceItems.map((service) => (
                <div key={service.label} className="flex items-center justify-between rounded-2xl border border-white/10 bg-black/20 px-4 py-3">
                  <div className="text-right">
                    <p className="text-sm font-semibold text-white">{service.label}</p>
                    <p className="text-xs text-white/45">مكوّن جاهز داخل الشبكة المحلية</p>
                  </div>
                  <span className={`text-sm font-semibold ${service.tone}`}>{service.value}</span>
                </div>
              ))}
            </div>
          </GlassCard>

          <GlassCard className="p-6">
            <div className="flex items-center justify-between">
              <div className="text-right">
                <p className="text-xs font-semibold text-white/40">الأقسام</p>
                <h2 className="mt-2 text-2xl font-bold text-white">{categories.length} قسم متاح</h2>
              </div>
              <Icon name="library" className="h-5 w-5 text-electric" />
            </div>
            <div className="mt-5 space-y-3">
              {categories.slice(0, 4).map((category) => (
                <button
                  key={category.slug}
                  type="button"
                  onClick={() => onOpenCategory(category.slug)}
                  className="flex w-full items-center justify-between rounded-2xl border border-white/10 bg-black/20 px-4 py-3 text-left transition hover:border-electric/20 hover:bg-white/[0.05]"
                >
                  <div className="text-right">
                    <p className="font-semibold text-white">{category.titleAr}</p>
                    <p className="text-xs text-white/45">{category.titleEn}</p>
                  </div>
                  <div className="text-right">
                    <p className="font-bold text-white">{category.count}</p>
                    <p className="text-[11px] font-semibold text-white/35">عنوان</p>
                  </div>
                </button>
              ))}
            </div>
          </GlassCard>
        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.35fr_0.95fr]">
        <GlassCard className="p-6">
          <div className="flex items-end justify-between gap-4">
            <div className="text-right">
              <p className="text-xs font-semibold text-white/40">الرف المميز</p>
              <h2 className="mt-2 text-2xl font-bold text-white">أقوى العناوين على الخادم المحلي</h2>
            </div>
            <span className="rounded-full border border-electric/20 bg-electric/12 px-3 py-1 text-xs font-semibold text-electric">
              {searchResults.length} نتيجة
            </span>
          </div>
          <div className="mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-2">
            {featured.map((item, index) => (
              <MediaCard key={item.id} item={item} onOpen={onOpenMedia} index={index} />
            ))}
          </div>
        </GlassCard>

        <GlassCard className="p-6">
          <div className="flex items-center justify-between">
            <div className="text-right">
              <p className="text-xs font-semibold text-white/40">آخر التحركات</p>
              <h2 className="mt-2 text-2xl font-bold text-white">ما يجري خلف الزجاج</h2>
            </div>
            <Icon name="bell" className="h-5 w-5 text-electric" />
          </div>
          <div className="mt-6 space-y-4">
            {mockActivity.map((item) => (
              <div key={item.title} className="rounded-2xl border border-white/10 bg-black/20 p-4 text-right">
                <div className="flex items-center justify-between">
                  <span className={`text-xs font-semibold ${
                    item.tone === "emerald" ? "text-emerald-300" : item.tone === "cyan" ? "text-cyan-300" : "text-violet-300"
                  }`}>
                    {item.label}
                  </span>
                  <span className="text-xs text-white/35">الآن</span>
                </div>
                <h3 className="mt-3 text-lg font-semibold text-white">{item.title}</h3>
                <p className="mt-2 text-sm leading-6 text-white/65">{item.body}</p>
              </div>
            ))}
          </div>
        </GlassCard>
      </section>
    </div>
  );
}
