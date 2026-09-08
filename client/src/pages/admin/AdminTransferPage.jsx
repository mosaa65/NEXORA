import React, { useEffect, useState } from "react";
import Icon from "../../components/Icon.jsx";
import { getTransferDevices, getTransferJobs, cancelTransferJob, getMediaList, getMediaDetail, ejectTransferDevice } from "../../lib/api.js";
import { useTransfer } from "../../context/TransferContext.jsx";
import { normalizeTransferDevices } from "../../lib/transferDevices.js";

function formatBytes(bytes) {
  const size = Number(bytes || 0);
  if (!Number.isFinite(size) || size <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let unitIndex = 0;
  let current = size;
  while (current >= 1024 && unitIndex < units.length - 1) {
    current /= 1024;
    unitIndex += 1;
  }
  return `${current.toFixed(unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
}

export default function AdminTransferPage() {
  const [devices, setDevices] = useState([]);
  const [jobs, setJobs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [mediaItems, setMediaItems] = useState([]);
  const [catalogError, setCatalogError] = useState("");
  const { openTransferModal } = useTransfer();

  const refreshData = () => {
    Promise.allSettled([getTransferDevices(), getTransferJobs(), getMediaList({ limit: 20 })])
      .then(([devRes, jobsRes, mediaRes]) => {
        if (devRes.status === "fulfilled") setDevices(normalizeTransferDevices(devRes.value?.devices || []));
        if (jobsRes.status === "fulfilled") setJobs(jobsRes.value?.jobs || []);
        if (mediaRes.status === "fulfilled") setMediaItems(mediaRes.value?.items || []);
      })
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    refreshData();
    const timer = setInterval(refreshData, 3000);
    return () => clearInterval(timer);
  }, []);

  const handleCancelJob = async (jobId) => {
    try {
      await cancelTransferJob(jobId);
      refreshData();
    } catch {}
  };

  const handleOpenCopyModal = async (media, file) => {
    setCatalogError("");
    const title = media.title_ar || media.title_en || media.titleAr || media.titleEn || "";

    if (file?.id || file?.file_path || file?.path) {
      openTransferModal(file, title);
      return;
    }

    try {
      const detail = await getMediaDetail(media.id);
      const directFiles = detail?.files || [];
      const seasonFiles = (detail?.seasons || []).flatMap((season) => season.episodes || []);
      const firstVideo = [...directFiles, ...seasonFiles].find((item) => item?.id || item?.file_path || item?.path);

      if (!firstVideo) {
        setCatalogError("لا يوجد ملف فيديو فعلي لهذا العمل. افتح تفاصيل العمل أو أعد فهرسة المجلدات ثم حاول مرة أخرى.");
        return;
      }

      openTransferModal(firstVideo, title);
    } catch (err) {
      setCatalogError("تعذر جلب ملفات الفيديو لهذا العمل: " + (err.message || "تأكد من تشغيل السيرفر"));
    }
  };

  return (
    <div className="space-y-6 text-right" dir="rtl">
      {/* Header Banner */}
      <div className="relative overflow-hidden rounded-3xl border border-purple-500/20 bg-gradient-to-r from-[#120F2A] via-[#1A1238] to-[#0D0B1C] p-6 shadow-2xl">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <button
            type="button"
            onClick={refreshData}
            className="flex items-center justify-center gap-2 rounded-2xl border border-purple-500/30 bg-purple-950/40 px-4 py-2.5 text-xs font-bold text-purple-200 transition hover:bg-purple-900/60"
          >
            <span>🔄 تحديث حالة الأجهزة والمنافذ</span>
          </button>
          <div className="flex items-center gap-3">
            <div>
              <h2 className="text-xl font-black text-white">📱 إدارة النسخ عبر USB للزبائن</h2>
              <p className="text-xs text-purple-300/80">
                مراقبة أجهزة الأندرويد والآيفون الموصولة ونقل الأفلام والمسلسلات فوريًا
              </p>
            </div>
            <div className="flex h-12 w-12 items-center justify-center rounded-2xl border border-purple-500/30 bg-purple-900/30 text-2xl">
              📲
            </div>
          </div>
        </div>
      </div>

      {/* Connected Devices Grid */}
      <div className="space-y-3">
        <h3 className="text-sm font-black text-white flex items-center justify-between">
          <span className="rounded-full bg-emerald-500/10 px-3 py-1 font-mono text-xs text-emerald-400 border border-emerald-500/30">
            {devices.length} جهاز متصل حالياً
          </span>
          <span>🔌 الأجهزة المتصلة بالمنافذ</span>
        </h3>

        {loading ? (
          <div className="rounded-2xl border border-white/10 bg-white/[0.02] p-8 text-center text-xs text-white/50">
            جاري مسح منافذ USB والأجهزة المتصلة...
          </div>
        ) : devices.length === 0 ? (
          <div className="rounded-2xl border border-yellow-500/20 bg-yellow-950/10 p-6 text-center">
            <p className="text-3xl mb-2">🔌</p>
            <p className="text-xs font-bold text-yellow-200">لا توجد فلاشات USB أو أجهزة موصولة حالياً</p>
            <p className="mt-1 text-[11px] text-white/50">
              وصل فلاشة USB أو هاتف ذكي بكابل USB المربوط بالكمبيوتر وسيتم التعرف عليه تلقائياً.
            </p>
          </div>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {devices.map((device) => {
              const isAndroid = device.type === "android";
              const isIOS = device.type === "ios";
              const isStorage = device.type === "storage" || device.id?.startsWith("disk_");
              const hasSpace = Number(device.total_space || 0) > 0;
              const used = Math.max(0, Number(device.total_space || 0) - Number(device.free_space || 0));
              const usedPct = hasSpace ? (used / Number(device.total_space)) * 100 : 0;

              return (
                <div
                  key={device.id}
                  className="relative overflow-hidden rounded-2xl border border-purple-500/30 bg-gradient-to-b from-white/[0.04] to-white/[0.01] p-4 text-right backdrop-blur-md shadow-lg space-y-2.5"
                >
                  <div className="flex items-center justify-between">
                    <span className="rounded-full border border-emerald-500/30 bg-emerald-950/60 px-2.5 py-0.5 font-mono text-[10px] font-bold text-emerald-300">
                      {device.status || "متصل"}
                    </span>
                    <span className="text-2xl">{isAndroid ? "🤖" : isIOS ? "🍏" : "💾"}</span>
                  </div>
                  <div>
                    <h4 className="text-sm font-bold text-white truncate">{device.name}</h4>
                    <p className="mt-0.5 text-[11px] text-white/50">
                      {isIOS ? "هاتف آيفون" : isStorage ? "ذاكرة فلاش USB" : "هاتف أندرويد"}
                      {device.file_system ? ` (${device.file_system})` : ""}
                    </p>
                  </div>

                  {hasSpace && (
                    <div className="space-y-1">
                      <div className="flex items-center justify-between text-[10px] font-mono text-white/60">
                        <span>المتوفر: {formatBytes(device.free_space)}</span>
                        <span>{formatBytes(device.total_space)}</span>
                      </div>
                      <div className="h-1.5 w-full overflow-hidden rounded-full bg-white/10">
                        <div
                          className="h-full bg-gradient-to-r from-purple-500 to-emerald-400"
                          style={{ width: `${Math.min(100, usedPct)}%` }}
                        />
                      </div>
                    </div>
                  )}

                  {isStorage && (
                    <button
                      type="button"
                      onClick={async () => {
                        try {
                          await ejectTransferDevice(device.id);
                          refreshData();
                        } catch {}
                      }}
                      className="w-full text-center rounded-xl border border-white/10 bg-white/5 py-1 text-[11px] font-bold text-white/70 hover:text-red-300 hover:bg-white/10 transition"
                    >
                      ⏏️ إخراج آمن للفلاشة
                    </button>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Transfer Queue & History */}
      <div className="space-y-3">
        <h3 className="text-sm font-black text-white">📋 سجل وقائمة عمليات النسخ</h3>
        <div className="overflow-hidden rounded-2xl border border-white/10 bg-[#0C0B18]">
          {jobs.length === 0 ? (
            <div className="p-8 text-center text-xs text-white/40">
              لا توجد عمليات نسخ نشطة أو سابقة حالياً.
            </div>
          ) : (
            <div className="divide-y divide-white/5">
              {jobs.map((job) => {
                const isRunning = job.status === "processing" || job.status === "pending";
                const isCompleted = job.status === "completed";
                const isFailed = job.status === "failed";
                const isCancelled = job.status === "cancelled";

                return (
                  <div key={job.id} className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between hover:bg-white/[0.02]">
                    <div className="flex items-center gap-3">
                      {isRunning && (
                        <button
                          type="button"
                          onClick={() => handleCancelJob(job.id)}
                          className="rounded-xl border border-red-500/30 bg-red-950/40 px-3 py-1.5 text-[11px] font-bold text-red-300 transition hover:bg-red-900/60"
                        >
                          إلغاء
                        </button>
                      )}
                      <span className={`rounded-full px-2.5 py-0.5 text-[10px] font-bold ${
                        isCompleted ? "bg-emerald-950 text-emerald-300 border border-emerald-500/30" :
                        isRunning ? "bg-purple-950 text-purple-300 border border-purple-500/30 animate-pulse" :
                        isCancelled ? "bg-gray-800 text-gray-400" :
                        "bg-red-950 text-red-300 border border-red-500/30"
                      }`}>
                        {isCompleted ? "مكتمل" : isRunning ? "جاري النسخ..." : isCancelled ? "ملغاة" : "فشل"}
                      </span>
                    </div>

                    <div className="flex-1 text-right">
                      <p className="text-xs font-bold text-white truncate">{job.file_name}</p>
                      <div className="mt-1 flex items-center gap-3 text-[11px] text-white/50">
                        <span>الهاتف: {job.device_name}</span>
                        {job.speed_mbps > 0 && <span>السرعة: {job.speed_mbps.toFixed(1)} MB/s</span>}
                        {job.file_size > 0 && <span>الحجم: {(job.file_size / (1024 * 1024)).toFixed(0)} MB</span>}
                      </div>

                      {/* Progress bar for running job */}
                      {isRunning && (
                        <div className="mt-2 h-1.5 w-full overflow-hidden rounded-full bg-white/10">
                          <div
                            className="h-full rounded-full bg-gradient-to-r from-purple-500 to-fuchsia-400 transition-all duration-300"
                            style={{ width: `${job.progress || 0}%` }}
                          />
                        </div>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>

      {/* Media Catalog Quick Copy Trigger */}
      <div className="space-y-3">
        <h3 className="text-sm font-black text-white">🎬 اختيار وسائط ونقلها فورياً</h3>
        {catalogError && (
          <div className="rounded-2xl border border-red-500/30 bg-red-950/30 p-3 text-xs font-bold text-red-200">
            {catalogError}
          </div>
        )}
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {mediaItems.map((item) => (
            <div
              key={item.id}
              className="flex items-center justify-between rounded-2xl border border-white/10 bg-white/[0.02] p-3 transition hover:bg-white/5"
            >
              <button
                type="button"
                onClick={() => handleOpenCopyModal(item, null)}
                className="flex items-center gap-1.5 rounded-xl border border-purple-500/30 bg-purple-950/40 px-3 py-1.5 text-xs font-bold text-purple-200 transition hover:bg-purple-900/60"
              >
                <span>📱 نسخ للهاتف</span>
              </button>
              <div className="text-right truncate pr-2">
                <p className="text-xs font-bold text-white truncate">{item.title_ar || item.title_en || item.titleAr}</p>
                <p className="text-[10px] text-white/40">{item.type === "series" ? "مسلسل" : item.type === "anime" ? "أنمي" : "فيلم"}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
