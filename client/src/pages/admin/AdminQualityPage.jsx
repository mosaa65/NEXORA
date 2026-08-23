import { useEffect, useState } from "react";
import GlassCard from "../../components/GlassCard.jsx";
import { getQualityReport, calculateChecksums } from "../../lib/api.js";
import { formatBytes } from "./adminConstants.js";

/**
 * AdminQualityPage — صحة وجودة المكتبة
 * Route: /admin/quality
 */
export default function AdminQualityPage() {
  const [qualityReport, setQualityReport] = useState(null);
  const [qualityLoading, setQualityLoading] = useState(false);
  const [qualityTab, setQualityTab] = useState("missing");
  const [checksumState, setChecksumState] = useState("idle");

  useEffect(() => { loadQualityReport(); }, []);

  async function loadQualityReport() {
    setQualityLoading(true);
    try {
      const report = await getQualityReport();
      setQualityReport(report);
    } catch {}
    finally { setQualityLoading(false); }
  }

  async function handleTriggerChecksums() {
    setChecksumState("loading");
    try {
      await calculateChecksums();
      setChecksumState("success");
      loadQualityReport();
    } catch { setChecksumState("error"); }
  }

  return (
    <div className="space-y-6 text-right animate-fadeIn" dir="rtl">
      {/* Stats Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <GlassCard className="p-5 border-purple-500/20">
          <p className="text-xs font-bold text-white/50">إجمالي الأعمال والوسائط</p>
          <p className="mt-2 text-3xl font-black text-white">{qualityReport?.total_media || 0}</p>
          <p className="text-xs text-purple-400 mt-1">{qualityReport?.total_files || 0} ملف فيديو مسجل</p>
        </GlassCard>

        <GlassCard className="p-5 border-cyan-500/20">
          <p className="text-xs font-bold text-white/50">إجمالي الحجم المخزن</p>
          <p className="mt-2 text-3xl font-black text-cyan-300">{formatBytes(qualityReport?.total_size_bytes)}</p>
          <p className="text-xs text-white/40 mt-1">عبر كافة الأقراص</p>
        </GlassCard>

        <GlassCard className="p-5 border-amber-500/20">
          <p className="text-xs font-bold text-white/50">الحلقات المفقودة (ثغرات)</p>
          <p className="mt-2 text-3xl font-black text-amber-400">{qualityReport?.missing_episodes_count || 0}</p>
          <p className="text-xs text-amber-300/70 mt-1">حلقات تحتاج تحميل أو نقل</p>
        </GlassCard>

        <GlassCard className="p-5 border-rose-500/20">
          <p className="text-xs font-bold text-white/50">المساحة المهدرة (تكرارات)</p>
          <p className="mt-2 text-3xl font-black text-rose-400">{formatBytes(qualityReport?.total_wasted_bytes)}</p>
          <p className="text-xs text-rose-300/70 mt-1">{qualityReport?.duplicate_groups_count || 0} مجموعة مكررة</p>
        </GlassCard>
      </div>

      {/* Detail Tabs */}
      <GlassCard className="p-6">
        <div className="flex flex-wrap items-center justify-between gap-4 border-b border-white/10 pb-4">
          <div className="flex gap-2">
            <button onClick={() => setQualityTab("missing")}
              className={`px-4 py-2 rounded-xl text-xs font-bold transition ${qualityTab === "missing" ? "bg-purple-600 text-white" : "bg-white/[0.05] text-white/60 hover:text-white"}`}>
              ⚠️ الحلقات المفقودة ({qualityReport?.missing_episodes_count || 0})
            </button>
            <button onClick={() => setQualityTab("duplicates")}
              className={`px-4 py-2 rounded-xl text-xs font-bold transition ${qualityTab === "duplicates" ? "bg-purple-600 text-white" : "bg-white/[0.05] text-white/60 hover:text-white"}`}>
              🧹 كاشف التكرارات ({qualityReport?.duplicate_groups_count || 0})
            </button>
            <button onClick={() => setQualityTab("corrupted")}
              className={`px-4 py-2 rounded-xl text-xs font-bold transition ${qualityTab === "corrupted" ? "bg-purple-600 text-white" : "bg-white/[0.05] text-white/60 hover:text-white"}`}>
              🛡️ الملفات المعطوبة ({qualityReport?.corrupted_files_count || 0})
            </button>
          </div>

          <button type="button" onClick={handleTriggerChecksums} disabled={checksumState === "loading"}
            className="flex items-center gap-2 px-4 py-2 rounded-xl bg-gradient-to-r from-purple-700 to-fuchsia-700 text-xs font-bold text-white disabled:opacity-50">
            <span>{checksumState === "loading" ? "جارٍ الحساب..." : "حساب بصمات SHA-256 للمكتبة"}</span>
          </button>
        </div>

        {qualityTab === "missing" && (
          <div className="mt-5 space-y-3">
            {qualityReport?.missing_episodes?.length > 0 ? (
              qualityReport.missing_episodes.map((item, idx) => (
                <div key={idx} className="flex items-center justify-between p-4 rounded-2xl bg-white/[0.02] border border-amber-500/20">
                  <div className="flex items-center gap-3">
                    <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-amber-500/10 text-amber-400 font-bold text-xs">E{item.missing_episode_number}</div>
                    <div>
                      <p className="text-sm font-bold text-white">{item.show_title_ar || item.show_title_en} — الموسم {item.season_number}</p>
                      <p className="text-xs text-amber-300/80">حلقة ناقصة: {item.missing_episode_number}</p>
                    </div>
                  </div>
                  <div className="font-mono text-xs text-white/40 bg-black/40 px-3 py-1.5 rounded-xl" dir="ltr">{item.suggested_filename}</div>
                </div>
              ))
            ) : (
              <div className="p-8 text-center text-white/50">✅ تسلسل الحلقات مكتمل ولا توجد فجوات!</div>
            )}
          </div>
        )}

        {qualityTab === "duplicates" && (
          <div className="mt-5 space-y-4">
            {qualityReport?.duplicate_groups?.length > 0 ? (
              qualityReport.duplicate_groups.map((group, idx) => (
                <div key={idx} className="p-4 rounded-2xl bg-white/[0.02] border border-white/10 space-y-2">
                  <div className="flex items-center justify-between text-xs">
                    <span className="font-mono text-fuchsia-400">SHA-256: {group.checksum?.substring(0, 16)}...</span>
                    <span className="text-rose-400 font-bold">الحجم: {formatBytes(group.file_size)}</span>
                  </div>
                  <div className="space-y-1">
                    {group.files?.map((f, fIdx) => (
                      <div key={fIdx} className="p-2.5 rounded-lg bg-black/40 text-xs font-mono text-white/70" dir="ltr">{f.file_path}</div>
                    ))}
                  </div>
                </div>
              ))
            ) : (
              <div className="p-8 text-center text-white/50">✅ لا توجد ملفات مكررة مطابقة في المكتبة!</div>
            )}
          </div>
        )}

        {qualityTab === "corrupted" && (
          <div className="mt-5 space-y-3">
            {qualityReport?.corrupted_files?.length > 0 ? (
              qualityReport.corrupted_files.map((file, idx) => (
                <div key={idx} className="p-4 rounded-2xl bg-rose-500/10 border border-rose-500/30 space-y-1 text-xs">
                  <p className="font-bold text-rose-300">{file.show_title}</p>
                  <p className="font-mono text-white/60" dir="ltr">{file.file_path}</p>
                  <p className="text-rose-400">{file.error_output}</p>
                </div>
              ))
            ) : (
              <div className="p-8 text-center text-white/50">✅ تم فحص تدفقات الفيديو ولا توجد ملفات معطوبة!</div>
            )}
          </div>
        )}
      </GlassCard>
    </div>
  );
}
