import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import NexoraPlayer from "../components/NexoraPlayer.jsx";
import { getMediaDetail, getFileSubtitles, getMediaList, resolveAPIURL } from "../lib/api.js";

/**
 * DevPlayerPage — migration harness, NOT part of the customer experience.
 *
 * Route: /dev/player (reachable only by typing the URL, not linked anywhere).
 * Purpose: render the Video.js-based NexoraPlayer against a real catalogued
 * file so the migration can be compared side by side with the legacy player
 * before any user-facing switch happens.
 *
 * It deliberately reuses the same API calls the real modal uses
 * (getMediaDetail / getFileSubtitles / resolveAPIURL) so the source URL, the
 * subtitle tracks and the file identity come from the live catalogue rather
 * than from fixtures. If this page works, the same data can drive the real
 * player later.
 */
export default function DevPlayerPage() {
  const navigate = useNavigate();
  const [params] = useSearchParams();

  const [library, setLibrary] = useState([]);
  const [mediaId, setMediaId] = useState(params.get("media") || "");
  const [detail, setDetail] = useState(null);
  const [activeFile, setActiveFile] = useState(null);
  const [tracks, setTracks] = useState([]);
  const [status, setStatus] = useState("");

  // Step 1: offer a few real titles so a file can be picked without guessing ids.
  useEffect(() => {
    getMediaList({ limit: 30 })
      .then((payload) => setLibrary(payload?.items || []))
      .catch(() => setLibrary([]));
  }, []);

  // Step 2: load the chosen work and select its first playable file.
  useEffect(() => {
    if (!mediaId) {
      setDetail(null);
      setActiveFile(null);
      return;
    }
    let alive = true;
    setStatus("جارٍ تحميل تفاصيل العمل…");
    getMediaDetail(mediaId)
      .then((data) => {
        if (!alive) return;
        setDetail(data);
        const seasons = data?.seasons || [];
        const playable = seasons.length > 0
          ? seasons.flatMap((s) => (s.episodes || []).map((ep) => ({ ...ep, seasonNumber: s.season_number })))
          : data?.files || [];
        setActiveFile(playable[0] || null);
        setStatus(playable.length === 0 ? "لا يوجد ملف فيديو مفهرس لهذا العمل." : "");
      })
      .catch((err) => alive && setStatus(`فشل التحميل: ${err.message}`));
    return () => { alive = false; };
  }, [mediaId]);

  // Step 3: resolve the subtitle tracks for the selected file.
  useEffect(() => {
    if (!activeFile?.id) { setTracks([]); return; }
    let alive = true;
    getFileSubtitles(activeFile.id)
      .then((res) => {
        if (!alive) return;
        setTracks((res.subtitles || []).map((sub) => ({
          kind: "captions",
          label: sub.label || (sub.language === "ar" ? "العربية" : sub.language),
          src: resolveAPIURL(`/api/stream/file/${activeFile.id}/subtitles/${sub.index}`),
          srcLang: sub.language || "ar",
          default: sub.language === "ar",
        })));
      })
      .catch(() => alive && setTracks([]));
    return () => { alive = false; };
  }, [activeFile?.id]);

  const streamSrc = useMemo(() => {
    if (!activeFile) return "";
    if (activeFile.id) return resolveAPIURL(`/api/stream/file/${activeFile.id}`);
    if (activeFile.file_path) return resolveAPIURL(`/api/stream?path=${encodeURIComponent(activeFile.file_path)}`);
    return "";
  }, [activeFile]);

  const title = detail?.title_ar || detail?.title_en || "اختبار المشغل";

  return (
    <div dir="rtl" className="min-h-screen bg-[var(--bg-base)] p-4 text-[var(--text-primary)] sm:p-8">
      <div className="mx-auto max-w-6xl space-y-4">
        <header className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border-[var(--border-default)] bg-[var(--bg-card)] p-4">
          <div>
            <h1 className="text-lg font-black">اختبار Video.js — مشغل NEXORA الجديد</h1>
            <p className="text-xs text-white/50">صفحة تطوير غير مرتبطة بأي رابط في الواجهة.</p>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => navigate("/")}
              className="rounded-xl border-[var(--border-default)] px-4 py-2 text-xs font-bold"
            >
              رجوع للموقع
            </button>
          </div>
        </header>

        <section className="flex flex-wrap items-center gap-3 rounded-2xl border-[var(--border-default)] bg-[var(--bg-card)] p-4">
          <label className="text-xs font-bold">
            رقم العمل
            <input
              value={mediaId}
              onChange={(e) => setMediaId(e.target.value)}
              className="ms-2 w-24 rounded-lg bg-black/30 px-2 py-1 text-xs"
              placeholder="مثال 12"
            />
          </label>
          <label className="text-xs font-bold">
            أو اختر من الكتالوج
            <select
              value={mediaId}
              onChange={(e) => setMediaId(e.target.value)}
              className="ms-2 max-w-[22rem] rounded-lg bg-black/30 px-2 py-1 text-xs"
            >
              <option value="">— اختر عملًا —</option>
              {library.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.title_ar || item.title_en} ({item.file_count ?? 0} ملف)
                </option>
              ))}
            </select>
          </label>
          {status && <span className="text-xs text-amber-300">{status}</span>}
        </section>

        <section className="overflow-hidden rounded-2xl border-[var(--border-default)] bg-black">
          {streamSrc ? (
            <NexoraPlayer src={streamSrc} poster={resolveAPIURL(detail?.banner_path || detail?.poster_path) || undefined} title={title} />
          ) : (
            <div className="flex aspect-video items-center justify-center text-sm text-white/50">
              {status || "اختر عملًا لعرض المشغل."}
            </div>
          )}
        </section>

        <section className="grid gap-3 sm:grid-cols-2">
          <div className="rounded-2xl border-[var(--border-default)] bg-[var(--bg-card)] p-4 text-xs">
            <p className="mb-2 font-black">معلومات التشخيص</p>
            <p>العنوان: {title}</p>
            <p className="break-all">المصدر: {streamSrc || "—"}</p>
            <p>رقم الملف: {activeFile?.id ?? "—"}</p>
            <p>الدقة: {activeFile?.resolution || "—"}</p>
            <p>الترجمات المكتشفة: {tracks.length}</p>
          </div>
          <div className="rounded-2xl border-[var(--border-default)] bg-[var(--bg-card)] p-4 text-xs">
            <p className="mb-2 font-black">ملاحظات</p>
            <p className="text-white/60">
              المشغل الجديد (Video.js داخل شل NEXORA) أصبح المشغل النهائي في التطبيق: تشغيل،
              ±10 ثوانٍ، ترجمة، استئناف، حفظ تقدم، انتقال تلقائي للحلقة التالية، معاينة الشريط
              الزمني، PiP، ملء الشاشة، وRTL. هذه الصفحة تبقى أداة تطوير للاختبار المعزول.
            </p>
          </div>
        </section>
      </div>
    </div>
  );
}
