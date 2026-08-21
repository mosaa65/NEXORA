import React, { useState, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { getSystemDrives, browseSystemDirectory } from "../lib/api";
import Icon from "./Icon";

export default function DirectoryPickerModal({ isOpen, onClose, onSelectDirectory, initialPath = "" }) {
  const [currentPath, setCurrentPath] = useState(initialPath);
  const [parentPath, setParentPath] = useState("");
  const [directories, setDirectories] = useState([]);
  const [drives, setDrives] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  useEffect(() => {
    if (isOpen) {
      loadDrivesAndPath(initialPath);
    }
  }, [isOpen, initialPath]);

  async function loadDrivesAndPath(path) {
    setLoading(true);
    setError(null);
    try {
      const drivesRes = await getSystemDrives();
      setDrives(drivesRes.drives || []);

      if (path) {
        await navigateTo(path);
      } else if (drivesRes.drives?.length > 0) {
        const defaultDrive = drivesRes.drives.find((d) => d.disk_letter !== "C") || drivesRes.drives[0];
        await navigateTo(`${defaultDrive.disk_letter}:\\`);
      }
    } catch (err) {
      setError("فشل تحميل مسارات الأقراص: " + err.message);
    } finally {
      setLoading(false);
    }
  }

  async function navigateTo(path) {
    setLoading(true);
    setError(null);
    try {
      const res = await browseSystemDirectory(path);
      setCurrentPath(res.current_path || path);
      setParentPath(res.parent_path || "");
      setDirectories(res.directories || []);
    } catch (err) {
      setError("تعذر قراءة المجلد: " + err.message);
    } finally {
      setLoading(false);
    }
  }

  function handleSelect() {
    if (currentPath) {
      onSelectDirectory(currentPath);
      onClose();
    }
  }

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-md" dir="rtl">
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        exit={{ opacity: 0, scale: 0.95 }}
        className="bg-slate-900 border border-slate-700/80 rounded-3xl p-6 max-w-2xl w-full space-y-4 shadow-2xl text-right max-h-[85vh] flex flex-col"
      >
        {/* Modal Header */}
        <div className="flex items-center justify-between border-b border-slate-800 pb-3">
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-xl bg-indigo-600/20 text-indigo-400 border border-indigo-500/30">
              <Icon name="search" className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-lg font-bold text-white">مستعرض مجلدات وأقراص السيرفر</h3>
              <p className="text-xs text-gray-400">تصفح واختيار المجلد المراد فهرسته مباشرة من القرص الصلب</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-2 rounded-xl hover:bg-slate-800 text-gray-400 hover:text-white transition"
          >
            ✕
          </button>
        </div>

        {/* Drives Quick Switch Bar */}
        <div className="space-y-1.5">
          <span className="text-xs font-semibold text-gray-400">الأقراص المتصلة بالسيرفر:</span>
          <div className="flex items-center gap-2 flex-wrap">
            {drives.map((d) => {
              const drivePath = `${d.disk_letter}:\\`;
              const isSelected = currentPath.toLowerCase().startsWith(d.disk_letter.toLowerCase());
              return (
                <button
                  key={d.disk_letter}
                  onClick={() => navigateTo(drivePath)}
                  className={`px-3 py-1.5 rounded-xl text-xs font-bold transition flex items-center gap-1.5 border ${
                    isSelected
                      ? "bg-indigo-600 text-white border-indigo-400 shadow-md shadow-indigo-600/30"
                      : "bg-slate-800 text-gray-300 border-slate-700 hover:bg-slate-700"
                  }`}
                >
                  <span>💾 القرص ({d.disk_letter}:)</span>
                  {d.disk_label && <span className="text-[10px] text-white/60">[{d.disk_label}]</span>}
                </button>
              );
            })}
          </div>
        </div>

        {/* Current Path Bar & Up Navigation */}
        <div className="flex items-center gap-2 bg-slate-800/80 border border-slate-700/80 p-2.5 rounded-2xl">
          <button
            onClick={() => parentPath && navigateTo(parentPath)}
            disabled={!parentPath}
            className="p-2 rounded-xl bg-slate-700/60 hover:bg-slate-600 text-white disabled:opacity-30 transition"
            title="المجلد الأعلى"
          >
            <Icon name="arrowLeft" className="w-4 h-4 rotate-180" />
          </button>
          <div className="flex-1 overflow-x-auto text-xs font-mono text-cyan-300 px-2 py-1 bg-slate-950/60 rounded-xl border border-slate-800">
            {currentPath || "جاري تحديد المسار..."}
          </div>
        </div>

        {/* Error Alert */}
        {error && (
          <div className="p-3 rounded-xl bg-rose-950/60 border border-rose-500/30 text-rose-300 text-xs">
            {error}
          </div>
        )}

        {/* Directories List */}
        <div className="flex-1 overflow-y-auto min-h-[220px] max-h-[300px] border border-slate-800 rounded-2xl p-2 bg-slate-950/40 space-y-1">
          {loading ? (
            <div className="flex items-center justify-center h-full py-12 text-gray-400 gap-2 text-xs">
              <div className="w-4 h-4 border-2 border-indigo-500/30 border-t-indigo-500 rounded-full animate-spin" />
              <span>جاري قراءة المجلدات...</span>
            </div>
          ) : directories.length === 0 ? (
            <div className="text-center py-12 text-gray-500 text-xs">
              لا توجد مجلدات فرعية داخل هذا المسار.
            </div>
          ) : (
            directories.map((dir) => (
              <button
                key={dir.path}
                onClick={() => navigateTo(dir.path)}
                className="w-full flex items-center justify-between p-2.5 rounded-xl hover:bg-slate-800/80 transition text-right group border border-transparent hover:border-slate-700"
              >
                <div className="flex items-center gap-2.5">
                  <span className="text-base group-hover:scale-110 transition-transform">📁</span>
                  <span className="text-sm font-semibold text-white group-hover:text-indigo-300">
                    {dir.name}
                  </span>
                </div>
                <span className="text-[11px] text-gray-500 group-hover:text-gray-400 font-mono">
                  تصفح ‹
                </span>
              </button>
            ))
          )}
        </div>

        {/* Modal Actions */}
        <div className="flex items-center justify-between pt-3 border-t border-slate-800">
          <span className="text-xs text-gray-400">
            المسار المحدد: <strong className="text-white font-mono">{currentPath}</strong>
          </span>
          <div className="flex items-center gap-2">
            <button
              onClick={onClose}
              className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-gray-300 text-xs font-bold transition"
            >
              إلغاء
            </button>
            <button
              onClick={handleSelect}
              className="px-5 py-2 rounded-xl bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white text-xs font-bold shadow-lg shadow-indigo-500/20 transition flex items-center gap-2"
            >
              <Icon name="spark" className="w-3.5 h-3.5" />
              <span>اختيار هذا المجلد</span>
            </button>
          </div>
        </div>
      </motion.div>
    </div>
  );
}
