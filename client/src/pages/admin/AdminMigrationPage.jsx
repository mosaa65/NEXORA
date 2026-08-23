import { useState } from "react";
import GlassCard from "../../components/GlassCard.jsx";
import Icon from "../../components/Icon.jsx";
import DirectoryPickerModal from "../../components/DirectoryPickerModal.jsx";
import { previewMigration, copyMedia } from "../../lib/api.js";

/**
 * AdminMigrationPage — معالج التنظيم والنقل الفيزيائي الذكي
 * Route: /admin/migration
 */
export default function AdminMigrationPage() {
  const [previewRoot, setPreviewRoot] = useState("C:/Users/mousa/Desktop/مسلسل اجنبي");
  const [previewState, setPreviewState] = useState("idle");
  const [previewResult, setPreviewResult] = useState(null);
  const [previewError, setPreviewError] = useState("");

  const [copyTarget, setCopyTarget] = useState("D:/Media_Organized");
  const [copyState, setCopyState] = useState("idle");
  const [copyResult, setCopyResult] = useState(null);
  const [copyError, setCopyError] = useState("");

  // Directory pickers
  const [pickerTarget, setPickerTarget] = useState(null); // "source" | "dest" | null

  async function handlePreview() {
    if (!previewRoot.trim()) return;
    setPreviewState("loading");
    setPreviewError("");
    setPreviewResult(null);
    try {
      const result = await previewMigration(previewRoot.trim());
      setPreviewResult(result);
      setPreviewState("ready");
    } catch (err) {
      setPreviewError(err?.message || "تعذر إنشاء خطة المعاينة.");
      setPreviewState("error");
    }
  }

  async function handleExecuteAll() {
    if (!previewResult || !previewResult.entries || previewResult.entries.length === 0) return;
    if (!copyTarget.trim()) {
      alert("يرجى تحديد المجلد الهدف لنقل الملفات إليه.");
      return;
    }

    const sources = previewResult.entries
      .filter((e) => e.action !== "keep" && e.source)
      .map((e) => e.source);

    if (sources.length === 0) {
      alert("كافة الملفات منظمة بالفعل ولا تحتاج لنقل.");
      return;
    }

    if (!window.confirm(`هل أنت متأكد من نقل ${sources.length} ملفاً إلى المجلد الهدف: ${copyTarget} مع التحقق من البصمات؟`)) {
      return;
    }

    setCopyState("loading");
    setCopyError("");
    try {
      const result = await copyMedia({ sources, target: copyTarget.trim() });
      setCopyResult(result);
      setCopyState("success");
      // Refresh preview
      handlePreview();
    } catch (err) {
      setCopyError(err?.message || "تعذر إكمال عملية النقل.");
      setCopyState("error");
    }
  }

  return (
    <div className="space-y-6 text-right animate-fadeIn" dir="rtl">
      {/* Header Info */}
      <GlassCard className="p-6">
        <h2 className="text-xl font-bold text-white flex items-center gap-2">
          <Icon name="spark" className="h-5 w-5 text-fuchsia-400" />
          <span>معالج التنظيم الفيزيائي وإعادة هيكلة الملفات (Smart File Migration)</span>
        </h2>
        <p className="mt-1 text-xs text-white/60 leading-relaxed">
          يقوم هذا المعالج بفحص مجلداتك غير المنظمة، واقتراح هيكلية قياسية متوافقة (مثل <code>Series/Title/Season 01/Title S01E01.mp4</code>)، وتوفير معاينة تفصيلية قبل نقل أي بايت، مع التحقق من بصمات SHA-256 لضمان عدم تلف البيانات.
        </p>

        {/* Source & Target Configuration */}
        <div className="mt-5 grid gap-4 lg:grid-cols-2">
          <div className="space-y-2">
            <label className="block text-xs font-bold text-white/70">📁 مجلد المصدر (المجلد المراد فحصه وترتيبه):</label>
            <div className="flex items-center gap-2">
              <input
                value={previewRoot}
                onChange={(e) => setPreviewRoot(e.target.value)}
                placeholder="C:/Downloads/Unorganized"
                dir="ltr"
                className="flex-1 rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-xs text-white outline-none focus:border-fuchsia-500 font-mono"
              />
              <button
                type="button"
                onClick={() => setPickerTarget("source")}
                className="px-3 py-2.5 rounded-xl bg-white/[0.08] hover:bg-white/15 text-xs font-bold text-white border border-white/10 shrink-0"
              >
                استعراض...
              </button>
            </div>
          </div>

          <div className="space-y-2">
            <label className="block text-xs font-bold text-white/70">💾 مجلد الوجهة (المكان المنظم الجديد):</label>
            <div className="flex items-center gap-2">
              <input
                value={copyTarget}
                onChange={(e) => setCopyTarget(e.target.value)}
                placeholder="D:/Media_Organized"
                dir="ltr"
                className="flex-1 rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-xs text-white outline-none focus:border-fuchsia-500 font-mono"
              />
              <button
                type="button"
                onClick={() => setPickerTarget("dest")}
                className="px-3 py-2.5 rounded-xl bg-white/[0.08] hover:bg-white/15 text-xs font-bold text-white border border-white/10 shrink-0"
              >
                استعراض...
              </button>
            </div>
          </div>
        </div>

        {/* Action Button */}
        <div className="mt-5 flex items-center justify-between border-t border-white/10 pt-4">
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setPreviewRoot("C:/Users/mousa/Desktop/مسلسل اجنبي")}
              className="px-3 py-1.5 rounded-xl bg-fuchsia-600/20 border border-fuchsia-500/30 text-xs font-bold text-fuchsia-200"
            >
              🎬 C:/Users/mousa/Desktop/مسلسل اجنبي
            </button>
            <button
              type="button"
              onClick={() => setPreviewRoot("C:/Users/mousa/Desktop/project/NEXORA/server/testdata/test_media_root")}
              className="px-3 py-1.5 rounded-xl bg-white/[0.05] border border-white/10 text-xs font-bold text-white/70"
            >
              📁 test_media_root
            </button>
          </div>

          <button
            type="button"
            onClick={handlePreview}
            disabled={previewState === "loading" || !previewRoot.trim()}
            className="flex items-center gap-2 px-6 py-3 rounded-2xl bg-gradient-to-r from-purple-600 to-fuchsia-600 text-xs font-bold text-white shadow-lg shadow-purple-900/50 hover:brightness-110 disabled:opacity-50 transition"
          >
            <span>{previewState === "loading" ? "جارٍ تحليل الملفات والمجلدات..." : "🔍 إنشاء خطة المعاينة الذكية"}</span>
          </button>
        </div>
      </GlassCard>

      <DirectoryPickerModal
        isOpen={pickerTarget !== null}
        onClose={() => setPickerTarget(null)}
        initialPath={pickerTarget === "source" ? previewRoot : copyTarget}
        onSelectDirectory={(selected) => {
          if (pickerTarget === "source") setPreviewRoot(selected);
          if (pickerTarget === "dest") setCopyTarget(selected);
          setPickerTarget(null);
        }}
      />

      {/* Preview Result Table */}
      {previewState === "ready" && previewResult && (
        <GlassCard className="p-6 space-y-5">
          <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 border-b border-white/10 pb-4">
            <div>
              <h3 className="text-lg font-bold text-white">خطة التنظيم المقترحة</h3>
              <p className="text-xs text-white/50 mt-1">
                إجمالي الملفات: {previewResult.entries?.length || 0} • سيتم نقلها: {previewResult.moveCount || 0} • مكررة: {previewResult.duplicateCount || 0}
              </p>
            </div>

            <button
              type="button"
              onClick={handleExecuteAll}
              disabled={copyState === "loading" || (previewResult.moveCount || 0) === 0}
              className="px-6 py-3 rounded-2xl bg-gradient-to-r from-emerald-600 via-teal-600 to-emerald-700 text-xs font-bold text-white shadow-lg shadow-emerald-900/40 hover:brightness-110 disabled:opacity-50 transition"
            >
              {copyState === "loading" ? "جارٍ النقل والتحقق من البصمات..." : "🚀 تنفيذ خطة النقل والترتيب الآمن"}
            </button>
          </div>

          {copyState === "success" && copyResult && (
            <div className="p-4 rounded-2xl bg-emerald-950/30 border border-emerald-500/30 text-xs text-emerald-300 space-y-1">
              <p className="font-bold">✅ تم إكمال النقل الآمن بنجاح!</p>
              <p>تم نقل {copyResult.items?.length || 0} ملفاً بحجم إجمالي {((copyResult.completedBytes || 0) / 1048576).toFixed(1)} MB.</p>
            </div>
          )}

          {copyState === "error" && (
            <div className="p-4 rounded-2xl bg-rose-950/30 border border-rose-500/30 text-xs text-rose-300">
              ❌ {copyError}
            </div>
          )}

          {/* Entries list */}
          <div className="space-y-2 max-h-[500px] overflow-y-auto">
            {previewResult.entries?.map((entry, idx) => {
              const actionBg =
                entry.action === "move"
                  ? "bg-purple-500/20 text-purple-300 border-purple-500/30"
                  : entry.action === "duplicate"
                  ? "bg-amber-500/20 text-amber-300 border-amber-500/30"
                  : "bg-white/10 text-white/60 border-white/10";

              return (
                <div
                  key={idx}
                  className="p-3.5 rounded-2xl bg-white/[0.02] border border-white/5 space-y-2 text-xs"
                >
                  <div className="flex items-center justify-between">
                    <span className="font-bold text-white text-sm">
                      {entry.title || entry.source.split(/[\\/]/).pop()}
                      {entry.season > 0 && ` • الموسم ${entry.season}`}
                      {entry.episode > 0 && ` • الحلقة ${entry.episode}`}
                    </span>
                    <span className={`px-2 py-0.5 rounded-md border text-[10px] font-bold ${actionBg}`}>
                      {entry.action === "move" ? "نقل وإعادة تسمية" : entry.action === "duplicate" ? "ملف مكرر" : "موجود ومنظم"}
                    </span>
                  </div>

                  <div className="grid gap-1 font-mono text-[11px] text-white/50" dir="ltr">
                    <p className="truncate">
                      <span className="text-white/30">المسار الحالي: </span>
                      {entry.source}
                    </p>
                    <p className="truncate text-emerald-400">
                      <span className="text-white/30">المسار المنظم: </span>
                      {entry.target}
                    </p>
                  </div>
                </div>
              );
            })}
          </div>
        </GlassCard>
      )}

      {previewState === "error" && (
        <div className="p-4 rounded-2xl bg-rose-950/20 border border-rose-500/30 text-xs text-rose-300">
          ❌ {previewError}
        </div>
      )}
    </div>
  );
}
