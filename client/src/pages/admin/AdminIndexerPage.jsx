import { useEffect, useState } from "react";
import GlassCard from "../../components/GlassCard.jsx";
import Icon from "../../components/Icon.jsx";
import DirectoryPickerModal from "../../components/DirectoryPickerModal.jsx";
import { getDisks, scanDisks, indexLibrary } from "../../lib/api.js";

/**
 * AdminIndexerPage — فهرسة واختيار المجلدات والأقراص
 * Route: /admin/indexer
 */
export default function AdminIndexerPage() {
  const [indexRoot, setIndexRoot] = useState("C:/Users/mousa/Desktop/project/NEXORA/server/testdata/test_media_root");
  const [indexState, setIndexState] = useState("idle");
  const [indexResult, setIndexResult] = useState(null);
  const [indexError, setIndexError] = useState("");
  const [disks, setDisks] = useState([]);
  const [disksLoading, setDisksLoading] = useState(false);
  const [isDirPickerOpen, setIsDirPickerOpen] = useState(false);

  useEffect(() => { loadDisks(); }, []);

  async function loadDisks() {
    setDisksLoading(true);
    try {
      const data = await getDisks();
      if (data.disks && data.disks.length > 0) setDisks(data.disks);
    } catch {
      setDisks([
        { disk_letter: "C", disk_label: "النظام (System)", total_space: 512000000000, free_space: 180000000000, used_space: 332000000000, used_percent: 64.8 },
        { disk_letter: "D", disk_label: "مكتبة الأفلام والمسلسلات (D:)", total_space: 8000000000000, free_space: 3200000000000, used_space: 4800000000000, used_percent: 60.0 },
        { disk_letter: "E", disk_label: "أرشيف الأنمي والميديا (E:)", total_space: 14000000000000, free_space: 2100000000000, used_space: 1190000000000, used_percent: 85.0 },
      ]);
    } finally {
      setDisksLoading(false);
    }
  }

  async function handleScanDisks() {
    setDisksLoading(true);
    try {
      const data = await scanDisks();
      if (data.disks) setDisks(data.disks);
    } finally {
      setDisksLoading(false);
    }
  }

  async function handleIndex() {
    if (!indexRoot.trim()) return;
    setIndexState("loading");
    setIndexError("");
    try {
      const res = await indexLibrary([indexRoot.trim()]);
      setIndexResult(res);
      setIndexState("success");
      loadDisks();
    } catch (err) {
      setIndexError(err?.message || "تعذر إكمال الفهرسة.");
      setIndexState("error");
    }
  }

  return (
    <div className="space-y-6 text-right animate-fadeIn" dir="rtl">
      {/* Disks Panel */}
      <GlassCard className="p-6">
        <div className="flex items-center justify-between border-b border-white/10 pb-4">
          <div>
            <h2 className="text-xl font-bold text-white flex items-center gap-2">
              <Icon name="disk" className="h-5 w-5 text-fuchsia-400" />
              <span>الأقراص ووحدات التخزين المتصلة (Storage Disks)</span>
            </h2>
            <p className="text-xs text-white/50 mt-1">انقر على أي قرص لاختياره تلقائياً في مسار الفهرسة</p>
          </div>
          <button type="button" onClick={handleScanDisks} disabled={disksLoading}
            className="flex items-center gap-2 px-3.5 py-2 rounded-xl bg-white/[0.06] border border-white/10 hover:bg-white/10 text-xs font-bold text-white transition disabled:opacity-50">
            <Icon name="spark" className={`h-3.5 w-3.5 text-fuchsia-400 ${disksLoading ? "animate-spin" : ""}`} />
            <span>{disksLoading ? "جارٍ الفحص..." : "تحديث الأقراص"}</span>
          </button>
        </div>

        <div className="mt-5 grid gap-4 md:grid-cols-3">
          {disks.map((disk) => {
            const usedGB = (disk.used_space / 1073741824).toFixed(1);
            const totalGB = (disk.total_space / 1073741824).toFixed(1);
            const freeGB = (disk.free_space / 1073741824).toFixed(1);
            const percent = disk.used_percent || (disk.total_space > 0 ? (disk.used_space / disk.total_space) * 100 : 0);

            return (
              <div key={disk.disk_letter} onClick={() => setIndexRoot(`${disk.disk_letter}:/`)}
                className="cursor-pointer group p-4 rounded-2xl bg-white/[0.03] border border-white/10 hover:border-fuchsia-500/50 hover:bg-fuchsia-950/20 transition duration-300">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br from-purple-800 to-fuchsia-900 text-white font-black text-lg shadow-md group-hover:scale-105 transition">
                      {disk.disk_letter}:
                    </div>
                    <div>
                      <p className="text-sm font-bold text-white group-hover:text-fuchsia-300 transition">{disk.disk_label || `القرص ${disk.disk_letter}:`}</p>
                      <p className="text-[11px] text-white/50">متاح: <span className="text-emerald-400 font-semibold">{freeGB} GB</span> من {totalGB} GB</p>
                    </div>
                  </div>
                  <span className="text-xs font-mono font-bold text-fuchsia-300 bg-fuchsia-500/10 px-2 py-1 rounded-lg">{percent.toFixed(0)}%</span>
                </div>

                <div className="mt-4 h-2 w-full rounded-full bg-white/10 overflow-hidden">
                  <div className={`h-full rounded-full transition-all duration-500 ${percent > 90 ? "bg-rose-500" : percent > 75 ? "bg-amber-500" : "bg-gradient-to-r from-purple-500 to-fuchsia-500"}`}
                    style={{ width: `${Math.min(percent, 100)}%` }} />
                </div>

                <div className="mt-3 flex items-center justify-between text-[11px] text-white/40">
                  <span>مستهلك: {usedGB} GB</span>
                  <span className="text-fuchsia-400 group-hover:underline">اختر هذا القرص ↵</span>
                </div>
              </div>
            );
          })}
        </div>
      </GlassCard>

      {/* Indexing Panel */}
      <GlassCard className="p-6 border-fuchsia-500/30">
        <h2 className="text-xl font-bold text-white flex items-center gap-2">
          <Icon name="search" className="h-5 w-5 text-fuchsia-400" />
          <span>تحديد مسار المجلد وبدء الفهرسة الذكية</span>
        </h2>
        <p className="mt-1 text-sm text-white/60">اختر مساراً جاهزاً بالأسفل أو اكتب مسار أي مجلد أو قرص لبدء تنظيمه وفهرسته في ثوانٍ:</p>

        <div className="mt-4 flex flex-wrap gap-2">
          <span className="text-xs text-white/40 self-center ml-2">مسارات سريعة:</span>
          <button type="button" onClick={() => setIndexRoot("C:/Users/mousa/Desktop/project/NEXORA/server/testdata/test_media_root")}
            className="px-3 py-1.5 rounded-xl bg-fuchsia-600/20 border border-fuchsia-500/30 text-xs font-semibold text-fuchsia-300 hover:bg-fuchsia-600/30 transition">
            📁 مجلد بيانات الاختبار التجريبية (test_media_root)
          </button>
          <button type="button" onClick={() => setIndexRoot("D:/Media")}
            className="px-3 py-1.5 rounded-xl bg-white/[0.05] border border-white/10 text-xs font-semibold text-white/80 hover:bg-white/10 transition">
            💾 D:/Media
          </button>
          <button type="button" onClick={() => setIndexRoot("E:/Anime")}
            className="px-3 py-1.5 rounded-xl bg-white/[0.05] border border-white/10 text-xs font-semibold text-white/80 hover:bg-white/10 transition">
            💾 E:/Anime
          </button>
        </div>

        <div className="mt-4 flex flex-col md:flex-row gap-3">
          <div className="flex-1 flex items-center gap-2">
            <input value={indexRoot} onChange={(e) => setIndexRoot(e.target.value)} placeholder="مثال: D:/Media أو C:/Downloads/Anime" dir="ltr"
              className="flex-1 rounded-2xl border border-white/10 bg-black/40 px-4 py-3.5 text-sm text-white outline-none placeholder:text-white/30 focus:border-fuchsia-500 focus:ring-1 focus:ring-fuchsia-500 transition font-mono" />
            <button type="button" onClick={() => setIsDirPickerOpen(true)}
              className="px-4 py-3.5 rounded-2xl bg-white/[0.08] hover:bg-white/[0.15] text-white text-xs font-bold border border-white/15 transition flex items-center gap-1.5 shrink-0"
              title="استعراض مجلدات السيرفر">
              <span>📁 استعراض الأقراص...</span>
            </button>
          </div>

          <button type="button" onClick={handleIndex} disabled={indexState === "loading" || !indexRoot.trim()}
            className="inline-flex items-center justify-center gap-2 rounded-2xl bg-gradient-to-r from-purple-600 via-fuchsia-600 to-purple-700 px-7 py-3.5 text-sm font-bold text-white shadow-lg shadow-purple-900/50 hover:brightness-110 transition disabled:opacity-50">
            <Icon name="search" className={`h-4 w-4 ${indexState === "loading" ? "animate-spin" : ""}`} />
            <span>{indexState === "loading" ? "جارٍ الفهرسة الذكية..." : "🚀 بدء الفهرسة الشاملة"}</span>
          </button>
        </div>

        <DirectoryPickerModal
          isOpen={isDirPickerOpen}
          onClose={() => setIsDirPickerOpen(false)}
          initialPath={indexRoot}
          onSelectDirectory={(selected) => setIndexRoot(selected)}
        />

        {indexState === "success" && indexResult && (
          <div className="mt-6 p-5 rounded-2xl bg-emerald-950/20 border border-emerald-500/30 space-y-4 animate-fadeIn">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-emerald-400 font-bold text-sm">
                <span className="h-2.5 w-2.5 rounded-full bg-emerald-400" />
                <span>تمت الفهرسة ومعالجة التسميات بنجاح!</span>
              </div>
            </div>
            <div className="grid gap-3 sm:grid-cols-4">
              <div className="p-3.5 rounded-xl bg-black/30 border border-white/5 text-center">
                <p className="text-2xl font-black text-white">{indexResult.scanned || 0}</p>
                <p className="text-xs text-white/50 mt-1">ملف تم مسحه</p>
              </div>
              <div className="p-3.5 rounded-xl bg-black/30 border border-white/5 text-center">
                <p className="text-2xl font-black text-emerald-400">{indexResult.imported || 0}</p>
                <p className="text-xs text-white/50 mt-1">أُدخلت لقاعدة البيانات</p>
              </div>
              <div className="p-3.5 rounded-xl bg-black/30 border border-white/5 text-center">
                <p className="text-2xl font-black text-cyan-400">{indexResult.searchSync?.indexed || 0}</p>
                <p className="text-xs text-white/50 mt-1">مفهرسة بالبحث الفوري</p>
              </div>
              <div className="p-3.5 rounded-xl bg-black/30 border border-white/5 text-center">
                <p className="text-2xl font-black text-fuchsia-400">{indexResult.inspected || 0}</p>
                <p className="text-xs text-white/50 mt-1">مفحوصة تقنياً (ffprobe)</p>
              </div>
            </div>
          </div>
        )}

        {indexState === "error" && (
          <div className="mt-4 p-4 rounded-xl bg-rose-950/20 border border-rose-500/30 text-xs text-rose-300">
            ❌ {indexError}
          </div>
        )}
      </GlassCard>
    </div>
  );
}
