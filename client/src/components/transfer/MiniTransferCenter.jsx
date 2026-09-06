import React from "react";
import { useTransfer } from "../../context/TransferContext.jsx";

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

function formatSpeed(speedMbps, speedBps) {
  if (speedBps && speedBps > 0) {
    return `${formatBytes(speedBps)}/s`;
  }
  if (speedMbps && speedMbps > 0) {
    return `${speedMbps.toFixed(1)} MB/s`;
  }
  return "";
}

function formatETA(etaSeconds) {
  if (!etaSeconds || etaSeconds <= 0) return "";
  const mins = Math.floor(etaSeconds / 60);
  const secs = etaSeconds % 60;
  if (mins > 0) {
    return `${mins}:${secs < 10 ? "0" : ""}${secs} متبقي`;
  }
  return `${secs} ثانية متبقية`;
}

function getPhaseBadge(job) {
  if (!job) return { label: "جاري التحضير", color: "bg-purple-500/20 text-purple-300" };
  if (job.status === "completed") return { label: "✓ مكتمل", color: "bg-emerald-500/20 text-emerald-300" };
  if (job.status === "failed") return { label: "✕ فشل", color: "bg-red-500/20 text-red-300" };
  if (job.status === "cancelled") return { label: "ملغى", color: "bg-white/10 text-white/60" };
  if (job.status === "retrying" || job.phase === "retrying") return { label: "🔄 إعادة محاولة", color: "bg-amber-500/20 text-amber-300" };
  if (job.status === "waiting_device") return { label: "⏳ بانتظار توصيل الجهاز", color: "bg-amber-500/20 text-amber-300" };

  switch (job.phase) {
    case "queued":
      return { label: "في قائمة الانتظار", color: "bg-blue-500/20 text-blue-300" };
    case "preparing":
      return { label: "جاري التحضير", color: "bg-purple-500/20 text-purple-300" };
    case "connecting":
      return { label: "جاري الاتصال بالجهاز", color: "bg-purple-500/20 text-purple-300" };
    case "checking_destination":
      return { label: "فحص مجلد الوجهة", color: "bg-indigo-500/20 text-indigo-300" };
    case "copying":
      return { label: "جاري النسخ", color: "bg-fuchsia-500/20 text-fuchsia-300" };
    case "verifying":
      return { label: "جاري حفظ وتأكيد الملف بأمان", color: "bg-emerald-500/20 text-emerald-300" };
    default:
      return { label: "جاري النقل", color: "bg-purple-500/20 text-purple-300" };
  }
}

export default function MiniTransferCenter() {
  const {
    activeJobs,
    isCenterExpanded,
    setIsCenterExpanded,
    toggleCenterExpanded,
    cancelJob,
    centerDismissed,
    setCenterDismissed
  } = useTransfer();

  // Filter jobs to show active or recent jobs
  const relevantJobs = activeJobs.filter(
    (j) =>
      j.status === "processing" ||
      j.status === "pending" ||
      j.status === "queued" ||
      j.status === "retrying" ||
      j.status === "waiting_device" ||
      j.status === "completed" ||
      j.status === "failed"
  );

  if (relevantJobs.length === 0 || centerDismissed) {
    return null;
  }

  // Primary active job to highlight in collapsed view
  const primaryJob =
    relevantJobs.find(
      (j) =>
        j.status === "processing" ||
        j.status === "retrying" ||
        j.status === "queued" ||
        j.status === "pending"
    ) || relevantJobs[0];

  const badge = getPhaseBadge(primaryJob);
  const progress = Math.min(100, Math.max(0, primaryJob?.progress || 0));
  const speed = formatSpeed(primaryJob?.speed_mbps, primaryJob?.speed_bps);
  const eta = formatETA(primaryJob?.eta_seconds);
  const transferred = formatBytes(primaryJob?.transferred || primaryJob?.transferred_bytes || 0);
  const totalSize = formatBytes(primaryJob?.file_size || primaryJob?.total_bytes || 0);
  const isAllComplete = relevantJobs.every((j) => j.status === "completed");

  return (
    <div
      className="fixed bottom-4 left-4 right-4 sm:left-auto sm:right-6 z-50 sm:w-[420px] transition-all duration-300"
      dir="rtl"
    >
      <div className="overflow-hidden rounded-3xl border border-purple-500/40 bg-[#0F0E20]/95 shadow-[0_16px_50px_rgba(0,0,0,0.8)] backdrop-blur-2xl">
        {/* ======================================================== */}
        {/* COLLAPSED BAR HEADER */}
        {/* ======================================================== */}
        <div
          onClick={toggleCenterExpanded}
          className="flex cursor-pointer items-center justify-between p-3.5 transition hover:bg-white/[0.03]"
        >
          <div className="flex items-center gap-2.5 truncate">
            <span className="flex h-7 w-7 items-center justify-center rounded-xl bg-purple-600/30 text-sm animate-pulse">
              {isAllComplete ? "🎉" : "⚡"}
            </span>
            <div className="truncate">
              <div className="flex items-center gap-1.5">
                <span className={`rounded-md px-1.5 py-0.2 text-[9px] font-bold ${badge.color}`}>
                  {badge.label}
                </span>
                <p className="text-xs font-black text-white truncate max-w-[180px]">
                  {primaryJob.current_file || primaryJob.file_name || primaryJob.device_name || "مهمة نقل"}
                </p>
              </div>
              <p className="text-[10px] text-white/50 font-mono mt-0.5">
                {progress.toFixed(0)}% • {transferred} / {totalSize}
                {speed ? ` • ${speed}` : ""}
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {eta && <span className="text-[10px] font-mono text-purple-300/80">{eta}</span>}
            <button
              type="button"
              className="flex h-7 w-7 items-center justify-center rounded-lg bg-white/5 text-xs text-white/70 hover:bg-white/10"
              title={isCenterExpanded ? "تصغير" : "توسيع"}
            >
              {isCenterExpanded ? "▼" : "▲"}
            </button>
            {isAllComplete && (
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  setCenterDismissed(true);
                }}
                className="flex h-7 w-7 items-center justify-center rounded-lg bg-white/5 text-xs text-white/70 hover:bg-white/10 hover:text-white"
                title="إغلاق"
              >
                ✕
              </button>
            )}
          </div>
        </div>

        {/* Global mini progress bar along bottom of collapsed header */}
        <div className="h-1 w-full bg-white/5">
          <div
            className={`h-full transition-all duration-300 ${
              isAllComplete
                ? "bg-emerald-400"
                : "bg-gradient-to-r from-purple-500 via-fuchsia-500 to-emerald-400"
            }`}
            style={{ width: `${progress}%` }}
          />
        </div>

        {/* ======================================================== */}
        {/* EXPANDED CONTENT VIEW */}
        {/* ======================================================== */}
        {isCenterExpanded && (
          <div className="border-t border-white/10 p-4 space-y-3 max-h-[360px] overflow-y-auto">
            <div className="flex items-center justify-between text-xs text-white/70 pb-1 border-b border-white/5">
              <span className="font-bold">قائمة مهام النقل الحالية ({relevantJobs.length})</span>
              <button
                type="button"
                onClick={() => setIsCenterExpanded(false)}
                className="text-[10px] text-purple-400 hover:underline"
              >
                تصغير
              </button>
            </div>

            {relevantJobs.map((job) => {
              const jobBadge = getPhaseBadge(job);
              const jobProgress = Math.min(100, Math.max(0, job.progress || 0));
              const jobSpeed = formatSpeed(job.speed_mbps, job.speed_bps);
              const jobEta = formatETA(job.eta_seconds);
              const isRunning =
                job.status === "processing" ||
                job.status === "retrying" ||
                job.status === "queued" ||
                job.status === "pending";

              return (
                <div
                  key={job.id}
                  className="rounded-2xl border border-white/10 bg-black/40 p-3 space-y-2 text-xs"
                >
                  {/* Job Header */}
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-1.5 truncate flex-1 pl-2">
                      <span className={`rounded-md px-1.5 py-0.5 text-[9px] font-bold ${jobBadge.color}`}>
                        {jobBadge.label}
                      </span>
                      <p className="font-bold text-white truncate">
                        {job.current_file || job.file_name}
                      </p>
                    </div>

                    {isRunning && (
                      <button
                        type="button"
                        onClick={() => cancelJob(job.id)}
                        className="rounded-lg border border-red-500/30 bg-red-950/30 px-2 py-1 text-[10px] font-bold text-red-300 hover:bg-red-900/50 transition"
                      >
                        إلغاء
                      </button>
                    )}
                  </div>

                  {/* Multi-file status subtitle */}
                  {job.total_files > 1 && (
                    <div className="flex items-center justify-between text-[10px] text-purple-300/80">
                      <span>
                        الملف {job.finished_files || job.completed_files || 0} من {job.total_files}
                      </span>
                      {job.destination_path && (
                        <span className="truncate max-w-[200px] text-white/40" dir="ltr">
                          {job.destination_path}
                        </span>
                      )}
                    </div>
                  )}

                  {/* Progress Bar */}
                  <div className="h-2 w-full overflow-hidden rounded-full bg-white/10">
                    <div
                      className="h-full rounded-full bg-gradient-to-r from-purple-500 via-fuchsia-500 to-emerald-400 transition-all duration-300"
                      style={{ width: `${jobProgress}%` }}
                    />
                  </div>

                  {/* Stats line */}
                  <div className="flex items-center justify-between text-[10px] font-mono text-white/50">
                    <span>
                      {formatBytes(job.transferred || job.transferred_bytes || 0)} /{" "}
                      {formatBytes(job.file_size || job.total_bytes || 0)} ({jobProgress.toFixed(0)}%)
                    </span>
                    <span>
                      {jobSpeed} {jobEta ? `• ${jobEta}` : ""}
                    </span>
                  </div>

                  {/* Error display if failed */}
                  {job.error && (
                    <div className="rounded-lg border border-red-500/30 bg-red-950/40 p-2 text-[10px] text-red-200">
                      {job.error}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
