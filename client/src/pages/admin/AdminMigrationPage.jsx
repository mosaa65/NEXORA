import { useState } from "react";
import GlassCard from "../../components/GlassCard.jsx";
import { previewMigration, copyMedia } from "../../lib/api.js";

/**
 * AdminMigrationPage — معالج الترتيب والنقل
 * Route: /admin/migration
 */
export default function AdminMigrationPage() {
  const [previewRoot, setPreviewRoot] = useState("C:/Users/mousa/Desktop/project/NEXORA/server/testdata/test_media_root");
  const [previewState, setPreviewState] = useState("idle");
  const [previewResult, setPreviewResult] = useState(null);
  const [copyTarget, setCopyTarget] = useState("D:/Media_Organized");
  const [copySources, setCopySources] = useState("");
  const [copyState, setCopyState] = useState("idle");
  const [copyResult, setCopyResult] = useState(null);

  async function handlePreview() {
    if (!previewRoot.trim()) return;
    setPreviewState("loading");
    try {
      const result = await previewMigration(previewRoot.trim());
      setPreviewResult(result);
      setPreviewState("ready");
    } catch { setPreviewState("error"); }
  }

  async function handleCopy() {
    const sources = copySources.split("\n").map((s) => s.trim()).filter(Boolean);
    if (sources.length === 0 || !copyTarget.trim()) return;
    setCopyState("loading");
    try {
      const result = await copyMedia({ sources, target: copyTarget.trim() });
      setCopyResult(result);
      setCopyState("ready");
    } catch { setCopyState("error"); }
  }

  return (
    <section className="grid gap-6 xl:grid-cols-2 text-right animate-fadeIn" dir="rtl">
      <GlassCard className="p-6">
        <h2 className="text-xl font-bold text-white mb-2">معاينة خطة التنظيم الفيزيائي</h2>
        <div className="space-y-3">
          <input value={previewRoot} onChange={(e) => setPreviewRoot(e.target.value)} placeholder="D:/Downloads/Unorganized" dir="ltr"
            className="w-full rounded-xl border border-white/10 bg-black/40 p-3 text-xs text-white font-mono" />
          <button type="button" onClick={handlePreview} disabled={previewState === "loading"}
            className="px-5 py-2.5 rounded-xl bg-purple-600 text-xs font-bold text-white disabled:opacity-50">
            {previewState === "loading" ? "جارٍ التحليل..." : "إنشاء المعاينة"}
          </button>
        </div>
        {previewResult && (
          <div className="mt-4 p-4 rounded-xl bg-black/40 text-xs space-y-2">
            <p>إجمالي الملفات: {previewResult.entries?.length || 0}</p>
            <p className="text-emerald-400">سيتم نقلها: {previewResult.moveCount || 0}</p>
            <p className="text-amber-400">مكررة: {previewResult.duplicateCount || 0}</p>
          </div>
        )}
      </GlassCard>

      <GlassCard className="p-6">
        <h2 className="text-xl font-bold text-white mb-2">تنفيذ النقل الآمن مع البصمة</h2>
        <div className="space-y-3">
          <input value={copyTarget} onChange={(e) => setCopyTarget(e.target.value)} placeholder="D:/Media_Organized" dir="ltr"
            className="w-full rounded-xl border border-white/10 bg-black/40 p-3 text-xs text-white font-mono" />
          <textarea rows={4} value={copySources} onChange={(e) => setCopySources(e.target.value)} placeholder="مسار كل ملف في سطر..." dir="ltr"
            className="w-full rounded-xl border border-white/10 bg-black/40 p-3 text-xs text-white font-mono" />
          <button type="button" onClick={handleCopy} disabled={copyState === "loading"}
            className="w-full py-3 rounded-xl bg-emerald-600 text-xs font-bold text-white disabled:opacity-50">
            {copyState === "loading" ? "جارٍ النقل..." : "بدء النقل الآمن"}
          </button>
        </div>
      </GlassCard>
    </section>
  );
}
