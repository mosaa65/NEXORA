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

export default function FloatingCopyButton() {
  const { selectedFiles, totalSelectedSize, openTransferModal, clearSelection, isTransferModalOpen } = useTransfer();

  if (!selectedFiles || selectedFiles.length === 0 || isTransferModalOpen) {
    return null;
  }

  const count = selectedFiles.length;
  const sizeLabel = formatBytes(totalSelectedSize);
  const filesLabel = count === 1 ? "ملف واحد" : count === 2 ? "ملفان" : count <= 10 ? `${count} ملفات` : `${count} ملفًا`;

  return (
    <aside
      className="fixed bottom-6 right-6 z-40 flex items-center gap-2 rounded-2xl border border-purple-500/40 bg-[#120F28]/95 p-1.5 shadow-[0_12px_40px_rgba(147,51,234,0.35)] backdrop-blur-2xl transition-all duration-300 hover:scale-105 hover:border-purple-400"
      dir="rtl"
      aria-label="شريط النسخ السريع"
    >
      <button
        type="button"
        onClick={() => openTransferModal()}
        className="flex items-center gap-3 rounded-xl bg-gradient-to-r from-purple-600 via-fuchsia-600 to-emerald-500 px-4 py-2.5 text-xs font-black text-white shadow-lg transition hover:brightness-110 active:scale-95"
      >
        <span className="flex h-6 w-6 items-center justify-center rounded-full bg-white/20 text-sm animate-pulse">
          ⚡
        </span>
        <span>
          نسخ {filesLabel} <span className="text-white/80">• {sizeLabel}</span>
        </span>
      </button>

      <button
        type="button"
        onClick={clearSelection}
        title="إلغاء التحديد"
        className="flex h-9 w-9 items-center justify-center rounded-xl border border-white/10 bg-white/5 text-xs text-white/60 transition hover:bg-red-500/20 hover:text-red-300 hover:border-red-500/40"
      >
        ✕
      </button>
    </aside>
  );
}
