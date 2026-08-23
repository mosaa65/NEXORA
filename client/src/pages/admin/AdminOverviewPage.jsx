import GlassCard from "../../components/GlassCard.jsx";
import { scanDisks, syncIndex } from "../../lib/api.js";

/**
 * AdminOverviewPage — حالة النظام والخدمات
 * Route: /admin/overview
 */
export default function AdminOverviewPage({ health, onSyncIndex }) {
  return (
    <div className="space-y-6 text-right animate-fadeIn" dir="rtl">
      <div className="grid gap-4 md:grid-cols-4">
        <GlassCard className="p-5">
          <p className="text-xs text-white/50">خادم الـ API</p>
          <p className="mt-2 text-2xl font-bold text-emerald-400">● يعمل</p>
        </GlassCard>
        <GlassCard className="p-5">
          <p className="text-xs text-white/50">قاعدة البيانات</p>
          <p className="mt-2 text-2xl font-bold text-emerald-400">● متصلة</p>
        </GlassCard>
        <GlassCard className="p-5">
          <p className="text-xs text-white/50">محرك البحث</p>
          <p className="mt-2 text-2xl font-bold text-cyan-400">● Meilisearch</p>
        </GlassCard>
        <GlassCard className="p-5">
          <p className="text-xs text-white/50">مراقب المجلدات</p>
          <p className="mt-2 text-2xl font-bold text-fuchsia-400">● نشط</p>
        </GlassCard>
      </div>

      <GlassCard className="p-6">
        <h2 className="text-lg font-bold text-white mb-3">إجراءات سريعة</h2>
        <div className="flex gap-3">
          <button type="button" onClick={onSyncIndex}
            className="px-4 py-2.5 rounded-xl bg-fuchsia-600/30 text-xs font-bold text-fuchsia-200 hover:bg-fuchsia-600/50">
            مزامنة محرك البحث الفوري
          </button>
          <button type="button" onClick={() => scanDisks().catch(() => {})}
            className="px-4 py-2.5 rounded-xl bg-white/10 text-xs font-bold text-white hover:bg-white/20">
            تحديث الأقراص
          </button>
        </div>
      </GlassCard>
    </div>
  );
}
