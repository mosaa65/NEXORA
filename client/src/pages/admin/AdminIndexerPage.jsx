import { useEffect, useState } from "react";
import GlassCard from "../../components/GlassCard.jsx";
import Icon from "../../components/Icon.jsx";
import DirectoryPickerModal from "../../components/DirectoryPickerModal.jsx";
import { getDisks, scanDisks, indexLibrary, previewIndex } from "../../lib/api.js";

/**
 * AdminIndexerPage — فهرسة واختيار المجلدات والأقراص مع المعاينة الذكية (Dry-Run)
 * Route: /admin/indexer
 */
export default function AdminIndexerPage() {
  const [indexRoot, setIndexRoot] = useState("C:/Users/mousa/Desktop/مسلسل اجنبي");
  const [indexState, setIndexState] = useState("idle");
  const [indexResult, setIndexResult] = useState(null);
  const [indexError, setIndexError] = useState("");

  // Preview State (Dry-Run before insert)
  const [previewState, setPreviewState] = useState("idle");
  const [previewData, setPreviewData] = useState(null);
  const [previewError, setPreviewError] = useState("");

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

  async function handlePreview() {
    if (!indexRoot.trim()) return;
    setPreviewState("loading");
    setPreviewError("");
    setPreviewData(null);
    try {
      const res = await previewIndex([indexRoot.trim()]);
      setPreviewData(res);
      setPreviewState("success");
    } catch (err) {
      setPreviewError(err?.message || "تعذر إنشاء المعاينة.");
      setPreviewState("error");
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
      setPreviewData(null);
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
          <span>تحديد مسار المجلد والفهرسة الذكية</span>
        </h2>
        <p className="mt-1 text-sm text-white/60">
          اختر مسار المجلد لمعاينته بدقة واكتشاف البوسترات والمواسم والحلقات واسم العمل الحقيقي قبل الإدخال:
        </p>

        <div className="mt-4 flex flex-wrap gap-2">
          <span className="text-xs text-white/40 self-center ml-2">مسارات سريعة:</span>
          <button type="button" onClick={() => setIndexRoot("C:/Users/mousa/Desktop/مسلسل اجنبي")}
            className="px-3 py-1.5 rounded-xl bg-gradient-to-r from-purple-600/30 to-fuchsia-600/30 border border-fuchsia-500/40 text-xs font-semibold text-fuchsia-200 hover:brightness-110 transition">
            🎬 C:/Users/mousa/Desktop/مسلسل اجنبي
          </button>
          <button type="button" onClick={() => setIndexRoot("C:/Users/mousa/Desktop/project/NEXORA/server/testdata/test_media_root")}
            className="px-3 py-1.5 rounded-xl bg-white/[0.05] border border-white/10 text-xs font-semibold text-white/80 hover:bg-white/10 transition">
            📁 مجلد بيانات الاختبار (test_media_root)
          </button>
          <button type="button" onClick={() => setIndexRoot("D:/Media")}
            className="px-3 py-1.5 rounded-xl bg-white/[0.05] border border-white/10 text-xs font-semibold text-white/80 hover:bg-white/10 transition">
            💾 D:/Media
          </button>
        </div>

        <div className="mt-4 flex flex-col md:flex-row gap-3">
          <div className="flex-1 flex items-center gap-2">
            <input value={indexRoot} onChange={(e) => setIndexRoot(e.target.value)} placeholder="مثال: C:/Users/mousa/Desktop/مسلسل اجنبي" dir="ltr"
              className="flex-1 rounded-2xl border border-white/10 bg-black/40 px-4 py-3.5 text-sm text-white outline-none placeholder:text-white/30 focus:border-fuchsia-500 focus:ring-1 focus:ring-fuchsia-500 transition font-mono" />
            <button type="button" onClick={() => setIsDirPickerOpen(true)}
              className="px-4 py-3.5 rounded-2xl bg-white/[0.08] hover:bg-white/[0.15] text-white text-xs font-bold border border-white/15 transition flex items-center gap-1.5 shrink-0"
              title="استعراض مجلدات السيرفر">
              <span>📁 استعراض...</span>
            </button>
          </div>

          <div className="flex gap-2">
            <button type="button" onClick={handlePreview} disabled={previewState === "loading" || !indexRoot.trim()}
              className="inline-flex items-center justify-center gap-2 rounded-2xl border border-cyan-500/40 bg-cyan-500/15 hover:bg-cyan-500/25 px-5 py-3.5 text-sm font-bold text-cyan-200 shadow-lg shadow-cyan-900/30 transition disabled:opacity-50">
              <Icon name="spark" className={`h-4 w-4 text-cyan-400 ${previewState === "loading" ? "animate-spin" : ""}`} />
              <span>{previewState === "loading" ? "جارٍ إنشاء المعاينة..." : "🔍 معاينة الفهرسة (Dry-Run)"}</span>
            </button>

            <button type="button" onClick={handleIndex} disabled={indexState === "loading" || !indexRoot.trim()}
              className="inline-flex items-center justify-center gap-2 rounded-2xl bg-gradient-to-r from-purple-600 via-fuchsia-600 to-purple-700 px-6 py-3.5 text-sm font-bold text-white shadow-lg shadow-purple-900/50 hover:brightness-110 transition disabled:opacity-50">
              <Icon name="search" className={`h-4 w-4 ${indexState === "loading" ? "animate-spin" : ""}`} />
              <span>{indexState === "loading" ? "جارٍ الفهرسة..." : "🚀 بدء الفهرسة الشاملة"}</span>
            </button>
          </div>
        </div>

        <DirectoryPickerModal
          isOpen={isDirPickerOpen}
          onClose={() => setIsDirPickerOpen(false)}
          initialPath={indexRoot}
          onSelectDirectory={(selected) => setIndexRoot(selected)}
        />

        {/* --- DRY RUN PREVIEW ACCORDION --- */}
        {previewState === "success" && previewData && (
          <div className="mt-6 p-6 rounded-3xl bg-[#0B0916] border border-cyan-500/30 space-y-6 animate-fadeIn">
            <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 border-b border-white/10 pb-4">
              <div>
                <h3 className="text-lg font-black text-cyan-200 flex items-center gap-2">
                  <span>✨ نتيجة المعاينة الذكية قبل الفهرسة</span>
                </h3>
                <p className="text-xs text-white/50 mt-1">
                  تم اكتشاف {previewData.totalMedia} عمل و {previewData.totalFiles} ملف فيديو. لم يتم إدخال أي شيء لقاعدة البيانات بعد.
                </p>
              </div>
              <button type="button" onClick={handleIndex} disabled={indexState === "loading"}
                className="px-6 py-2.5 rounded-2xl bg-gradient-to-r from-emerald-600 via-teal-600 to-emerald-700 text-xs font-bold text-white shadow-lg shadow-emerald-900/40 hover:brightness-110 transition">
                {indexState === "loading" ? "جارٍ الإدخال..." : "🚀 اعتماد وإدخال الآن لقاعدة البيانات"}
              </button>
            </div>

            {previewData.mediaItems?.map((item, idx) => (
              <div key={idx} className="p-5 rounded-2xl bg-white/[0.03] border border-white/10 space-y-4">
                <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
                  <div className="flex items-center gap-4">
                    {/* Poster thumbnail */}
                    <div className="relative h-20 w-14 rounded-xl overflow-hidden bg-black/60 border border-white/10 shrink-0">
                      {item.poster_path ? (
                        <img src={item.poster_path} alt={item.title_en} className="h-full w-full object-cover" />
                      ) : (
                        <div className="h-full w-full flex items-center justify-center text-xs text-white/30">بدون غلاف</div>
                      )}
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <h4 className="text-lg font-black text-white">{item.title_ar || item.title_en}</h4>
                        {item.title_ar && item.title_en && item.title_ar !== item.title_en && (
                          <span className="text-xs font-mono text-white/50">({item.title_en})</span>
                        )}
                      </div>
                      <div className="mt-1 flex flex-wrap items-center gap-2">
                        <span className="px-2 py-0.5 rounded-md bg-purple-500/20 text-purple-300 text-[11px] font-bold">
                          قسم: {item.category_slug}
                        </span>
                        <span className="px-2 py-0.5 rounded-md bg-cyan-500/20 text-cyan-300 text-[11px] font-bold">
                          نوع: {item.type}
                        </span>
                        {item.origin_tags?.map((tag) => (
                          <span key={tag} className="px-2 py-0.5 rounded-md bg-amber-500/20 text-amber-300 text-[11px] font-bold">
                            🌍 {tag}
                          </span>
                        ))}
                        {item.poster_path && (
                          <span className="px-2 py-0.5 rounded-md bg-emerald-500/20 text-emerald-300 text-[11px] font-bold">
                            🖼️ تم العثور على بوستر محلي
                          </span>
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="text-left font-mono text-xs text-white/60">
                    <p>{item.seasons?.length || 0} موسم • {item.total_files} ملف</p>
                    <p className="text-[10px] text-white/40">{(item.total_size / 1048576).toFixed(1)} MB</p>
                  </div>
                </div>

                {/* Seasons and episodes list */}
                <div className="space-y-2 pt-2 border-t border-white/5">
                  {item.seasons?.map((season) => (
                    <div key={season.season_number} className="p-3 rounded-xl bg-black/40 border border-white/5 text-xs space-y-1.5">
                      <div className="flex items-center justify-between text-fuchsia-300 font-bold">
                        <span>الموسم {season.season_number} ({season.episode_count} حلقة)</span>
                      </div>
                      <div className="grid gap-1 sm:grid-cols-2">
                        {season.episodes?.map((ep, eIdx) => (
                          <div key={eIdx} className="flex items-center justify-between p-1.5 rounded-lg bg-white/[0.02] text-[11px] font-mono text-white/70" dir="ltr">
                            <span className="truncate max-w-[280px]" title={ep.path}>{ep.path.split(/[\\/]/).pop()}</span>
                            <span className="text-emerald-400 font-bold shrink-0 ml-2">الحلقة {ep.episode_number || "1"}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}

        {previewState === "error" && (
          <div className="mt-4 p-4 rounded-xl bg-rose-950/20 border border-rose-500/30 text-xs text-rose-300">
            ❌ {previewError}
          </div>
        )}

        {indexState === "success" && indexResult && (
          <div className="mt-6 p-5 rounded-2xl bg-emerald-950/20 border border-emerald-500/30 space-y-4 animate-fadeIn">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-emerald-400 font-bold text-sm">
                <span className="h-2.5 w-2.5 rounded-full bg-emerald-400" />
                <span>تمت الفهرسة ومعالجة التسميات والبوسترات بنجاح!</span>
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
