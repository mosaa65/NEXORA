import React, { useEffect, useState, useMemo } from "react";
import { useTransfer } from "../../context/TransferContext.jsx";
import {
  getTransferDevices,
  getTransferDeviceApps,
  browseTransferPath,
  createTransferFolder,
  openFileLocation,
  ejectTransferDevice
} from "../../lib/api.js";
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

export default function TransferModal() {
  const {
    selectedFiles,
    totalSelectedSize,
    removeFileFromSelection,
    isTransferModalOpen,
    closeTransferModal,
    startTransfer,
    devices: streamedDevices
  } = useTransfer();

  // 1. Devices State
  const [devices, setDevices] = useState([]);
  const [loadingDevices, setLoadingDevices] = useState(false);
  const [selectedDevice, setSelectedDevice] = useState(null);

  // 2. iOS Apps State
  const [deviceApps, setDeviceApps] = useState([]);
  const [loadingApps, setLoadingApps] = useState(false);
  const [selectedApp, setSelectedApp] = useState(null);
  const [customBundleId, setCustomBundleId] = useState("");

  // 3. Navigation & File Browser State (For iOS & USB Storage)
  const [browsingApp, setBrowsingApp] = useState(null);
  const [currentPath, setCurrentPath] = useState("");
  const [browserEntries, setBrowserEntries] = useState([]);
  const [loadingBrowser, setLoadingBrowser] = useState(false);
  const [newFolderName, setNewFolderName] = useState("");
  const [showNewFolderModal, setShowNewFolderModal] = useState(false);
  const [creatingFolder, setCreatingFolder] = useState(false);

  // 4. Android custom folders
  const [androidTargetFolder, setAndroidTargetFolder] = useState("Movies");
  const [androidSubFolder, setAndroidSubFolder] = useState("");

  // 5. Transfer execution & Eject state
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMsg, setErrorMsg] = useState("");
  const [successMsg, setSuccessMsg] = useState("");
  const [openingFolder, setOpeningFolder] = useState(false);
  const [ejectingDeviceId, setEjectingDeviceId] = useState(null);

  // Auto-detect default subfolder from selected media
  useEffect(() => {
    if (selectedFiles.length > 0 && !androidSubFolder) {
      const mediaTitle = selectedFiles[0]?.mediaTitle || "";
      if (mediaTitle) {
        setAndroidSubFolder(mediaTitle);
      }
    }
  }, [selectedFiles]);

  // Reset transient state whenever the modal is re-opened.
  useEffect(() => {
    if (isTransferModalOpen) {
      setErrorMsg("");
      setSuccessMsg("");
      setIsSubmitting(false);
    }
  }, [isTransferModalOpen]);

  // Live device feed from the shared SSE stream
  useEffect(() => {
    if (!Array.isArray(streamedDevices)) return;
    const list = normalizeTransferDevices(streamedDevices);
    setDevices(list);
    setSelectedDevice((prev) => {
      if (prev) {
        const match = list.find((d) => d.id === prev.id);
        if (match) return match;
      }
      const storageDevice = list.find((d) => d.type === "storage" || d.id?.startsWith("disk_"));
      return storageDevice || list[0] || null;
    });
  }, [streamedDevices]);

  // Manual refresh button
  const fetchDevices = async (showLoading = false) => {
    if (showLoading) setLoadingDevices(true);
    try {
      const res = await getTransferDevices();
      const list = normalizeTransferDevices(res.devices || []);
      setDevices(list);
      setSelectedDevice((prev) => {
        if (prev) {
          const match = list.find((d) => d.id === prev.id);
          if (match) return match;
        }
        const storageDevice = list.find((d) => d.type === "storage" || d.id?.startsWith("disk_"));
        return storageDevice || list[0] || null;
      });
    } catch (err) {
      if (showLoading) {
        setErrorMsg("تعذر جلب قائمة الأجهزة: " + (err.message || ""));
      }
    } finally {
      if (showLoading) setLoadingDevices(false);
    }
  };

  // Device classification
  const isIOS = useMemo(() => {
    if (!selectedDevice) return false;
    return selectedDevice.type === "ios" || selectedDevice.name?.toLowerCase().includes("iphone");
  }, [selectedDevice]);

  const isStorage = useMemo(() => {
    if (!selectedDevice) return false;
    return selectedDevice.type === "storage" || selectedDevice.id?.startsWith("disk_");
  }, [selectedDevice]);

  // Storage Capacity & Pre-checks
  const deviceTotalSpace = Number(selectedDevice?.total_space || 0);
  const deviceFreeSpace = Number(selectedDevice?.free_space || 0);
  const hasSpaceInfo = deviceTotalSpace > 0 && deviceFreeSpace > 0;

  const usedSpace = hasSpaceInfo ? Math.max(0, deviceTotalSpace - deviceFreeSpace) : 0;
  const usedPercent = hasSpaceInfo ? (usedSpace / deviceTotalSpace) * 100 : 0;
  const projectedPercent = hasSpaceInfo ? (Math.min(deviceFreeSpace, totalSelectedSize) / deviceTotalSpace) * 100 : 0;

  const isSpaceInsufficient = hasSpaceInfo && totalSelectedSize > deviceFreeSpace;

  const isFAT32 = String(selectedDevice?.file_system || "").toUpperCase() === "FAT32";
  const hasFileOver4GB = isFAT32 && selectedFiles.some((f) => Number(f.size || 0) >= 4294967295);

  // Load iOS Apps when iOS device is selected
  useEffect(() => {
    if (!selectedDevice || !isIOS) {
      setDeviceApps([]);
      setSelectedApp(null);
      setBrowsingApp(null);
      return;
    }

    setLoadingApps(true);
    setErrorMsg("");
    setBrowsingApp(null);
    getTransferDeviceApps(selectedDevice.id)
      .then((res) => {
        const apps = (res.apps || []).filter((a) => a.file_sharing_enabled);
        setDeviceApps(apps);
        const defaultApp = apps[0] || null;
        setSelectedApp(defaultApp);
        if (defaultApp) {
          setCustomBundleId(defaultApp.bundle_id || "");
        }
      })
      .catch((err) => {
        setDeviceApps([]);
        setErrorMsg(err.message || "لا يوجد تطبيق يدعم File Sharing على هذا الجهاز");
      })
      .finally(() => setLoadingApps(false));
  }, [selectedDevice?.id, isIOS]);

  // When a USB storage device is selected, automatically initialize its root directory
  useEffect(() => {
    if (!selectedDevice || !isStorage) return;
    const letter = selectedDevice.id.replace("disk_", "");
    const root = letter ? `${letter}:\\` : "";
    setCurrentPath(root);
    loadDirectory(selectedDevice, root);
  }, [selectedDevice?.id, isStorage]);

  // Load File Browser entries when browsing an app or path
  const loadDirectory = async (targetDevice, targetPath, appBundleId = "") => {
    if (!targetDevice) return;
    setLoadingBrowser(true);
    setErrorMsg("");
    try {
      const devType = targetDevice.type || (isIOS ? "ios" : isStorage ? "storage" : "android");
      const res = await browseTransferPath(targetDevice.id, targetPath, devType, appBundleId);
      setBrowserEntries(res.entries || []);
      setCurrentPath(res.path || targetPath);
    } catch (err) {
      setErrorMsg("تعذر فتح المجلد: " + (err.message || ""));
      setBrowserEntries([]);
    } finally {
      setLoadingBrowser(false);
    }
  };

  // Switch to browsing view for an iOS app
  const handleOpenAppBrowser = (app) => {
    setSelectedApp(app);
    setCustomBundleId(app?.bundle_id || "");
    setBrowsingApp(app);
    const bundleId = app?.bundle_id || customBundleId;
    setCurrentPath("/Documents");
    loadDirectory(selectedDevice, "/Documents", bundleId);
  };

  const handleNavigateFolder = (folderEntry) => {
    const nextPath = folderEntry.path;
    const bundleId = browsingApp?.bundle_id || customBundleId;
    loadDirectory(selectedDevice, nextPath, bundleId);
  };

  const handleNavigateUp = () => {
    if (isStorage) {
      const letter = selectedDevice.id.replace("disk_", "");
      const root = `${letter}:\\`;
      if (!currentPath || currentPath.toUpperCase() === root.toUpperCase() || currentPath.length <= 3) {
        return;
      }
      const trimmed = currentPath.replace(/[/\\]+$/, "");
      const lastSlash = Math.max(trimmed.lastIndexOf("\\"), trimmed.lastIndexOf("/"));
      if (lastSlash > 2) {
        loadDirectory(selectedDevice, trimmed.substring(0, lastSlash));
      } else {
        loadDirectory(selectedDevice, root);
      }
      return;
    }

    if (!currentPath || currentPath === "/Documents" || currentPath === "/" || currentPath.endsWith(":\\")) {
      setBrowsingApp(null);
      return;
    }
    const parts = currentPath.split("/").filter(Boolean);
    parts.pop();
    const parentPath = "/" + parts.join("/");
    const target = parentPath === "/" ? "/Documents" : parentPath;
    const bundleId = browsingApp?.bundle_id || customBundleId;
    loadDirectory(selectedDevice, target, bundleId);
  };

  // Create folder inside current directory
  const handleCreateFolder = async (e) => {
    e.preventDefault();
    const folderName = newFolderName.trim();
    if (!folderName) return;

    setCreatingFolder(true);
    setErrorMsg("");
    try {
      let newPath = "";
      if (isStorage) {
        newPath = currentPath.endsWith("\\") || currentPath.endsWith("/")
          ? `${currentPath}${folderName}`
          : `${currentPath}\\${folderName}`;
      } else {
        newPath = currentPath.endsWith("/") ? `${currentPath}${folderName}` : `${currentPath}/${folderName}`;
      }

      const bundleId = browsingApp?.bundle_id || customBundleId;
      const devType = selectedDevice?.type || (isIOS ? "ios" : isStorage ? "storage" : "android");

      await createTransferFolder({
        device_id: selectedDevice.id,
        path: newPath,
        device_type: devType,
        bundle_id: bundleId
      });

      setNewFolderName("");
      setShowNewFolderModal(false);
      await loadDirectory(selectedDevice, currentPath, bundleId);
    } catch (err) {
      setErrorMsg("تعذر إنشاء المجلد: " + (err.message || ""));
    } finally {
      setCreatingFolder(false);
    }
  };

  // Safe Eject Handler
  const handleEjectDevice = async (e, dev) => {
    e.stopPropagation();
    if (!dev?.id) return;
    setEjectingDeviceId(dev.id);
    setErrorMsg("");
    setSuccessMsg("");
    try {
      await ejectTransferDevice(dev.id);
      setSuccessMsg(`تم إخراج ${dev.name} بأمان. يمكنك الآن نزع القرص.`);
      fetchDevices();
    } catch (err) {
      setErrorMsg("تعذر إخراج القرص: " + (err.message || ""));
    } finally {
      setEjectingDeviceId(null);
    }
  };

  // Perform Transfer
  const handleStartCopy = async () => {
    if (!selectedDevice) {
      setErrorMsg("يرجى اختيار جهاز موصول أولاً");
      return;
    }
    if (selectedFiles.length === 0) {
      setErrorMsg("لا توجد ملفات محددة للنسخ");
      return;
    }
    if (isSpaceInsufficient) {
      setErrorMsg(`المساحة المتوفرة (${formatBytes(deviceFreeSpace)}) غير كافية لحجم الملفات (${formatBytes(totalSelectedSize)})`);
      return;
    }
    if (hasFileOver4GB) {
      setErrorMsg("نظام ملفات القرص (FAT32) لا يدعم ملفات أكبر من 4 جيجابايت.");
      return;
    }

    setIsSubmitting(true);
    setErrorMsg("");

    try {
      let targetApp = "";
      let targetFolder = "";
      let subFolder = "";

      if (isIOS) {
        targetApp = customBundleId.trim() || selectedApp?.bundle_id || "";
        if (!targetApp) {
          throw new Error("يرجى اختيار تطبيق iPhone أو إدخال المعرف");
        }
        subFolder = currentPath || "/Documents";
      } else if (isStorage) {
        const letter = selectedDevice.id.replace("disk_", "");
        targetFolder = currentPath || `${letter}:\\`;
      } else {
        targetFolder = androidTargetFolder;
        subFolder = androidSubFolder.trim();
      }

      await startTransfer({
        deviceId: selectedDevice.id,
        targetApp,
        targetFolder,
        subFolder,
        files: selectedFiles
      });
      setIsSubmitting(false);
    } catch (err) {
      setErrorMsg(err.message || "فشل بدء عملية النقل");
      setIsSubmitting(false);
    }
  };

  const handleOpenFileLocation = async () => {
    if (selectedFiles.length === 0) return;
    setOpeningFolder(true);
    try {
      const first = selectedFiles[0];
      await openFileLocation({
        file_id: first?.fileId || (Number(first?.id) > 0 ? Number(first?.id) : 0),
        path: first?.filePath || first?.file_path || first?.path || ""
      });
    } catch (err) {
      setErrorMsg("تعذر فتح المجلد: " + (err.message || ""));
    } finally {
      setOpeningFolder(false);
    }
  };

  if (!isTransferModalOpen) return null;

  const count = selectedFiles.length;
  const sizeLabel = formatBytes(totalSelectedSize);
  const filesCountLabel = count === 1 ? "ملف واحد" : count === 2 ? "ملفان" : count <= 10 ? `${count} ملفات` : `${count} ملفًا`;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/85 p-3 sm:p-6 backdrop-blur-xl animate-fadeIn"
      dir="rtl"
    >
      <div className="relative flex flex-col w-full max-w-5xl h-[90vh] max-h-[870px] rounded-3xl border border-purple-500/30 bg-[#0C0B18] shadow-2xl overflow-hidden">
        {/* Header */}
        <header className="flex items-center justify-between border-b border-white/10 px-6 py-4 bg-white/[0.02]">
          <div className="flex items-center gap-3">
            <span className="flex h-10 w-10 items-center justify-center rounded-2xl bg-gradient-to-tr from-purple-600 to-emerald-500 text-lg shadow-lg">
              ⚡
            </span>
            <div>
              <h2 className="text-base font-black text-white">النسخ إلى الأجهزة ووحدات USB</h2>
              <p className="text-xs text-purple-300/70">
                نقل مباشر وفائق السرعة إلى فلاشات USB والهواتف الذكية
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={closeTransferModal}
            className="flex h-8 w-8 items-center justify-center rounded-full border border-white/10 bg-white/5 text-white/70 transition hover:bg-white/15 hover:text-white"
          >
            ✕
          </button>
        </header>

        {/* Alerts Banner */}
        {errorMsg && (
          <aside aria-label="تنبيه الأخطاء" className="mx-6 mt-3 rounded-xl border border-red-500/40 bg-red-950/70 p-3 text-xs font-bold text-red-200 flex items-center justify-between">
            <span>⚠️ {errorMsg}</span>
            <button
              type="button"
              onClick={() => setErrorMsg("")}
              className="text-white/60 hover:text-white"
            >
              ✕
            </button>
          </aside>
        )}

        {successMsg && (
          <aside aria-label="تنبيه النجاح" className="mx-6 mt-3 rounded-xl border border-emerald-500/40 bg-emerald-950/70 p-3 text-xs font-bold text-emerald-200 flex items-center justify-between">
            <span>✓ {successMsg}</span>
            <button
              type="button"
              onClick={() => setSuccessMsg("")}
              className="text-white/60 hover:text-white"
            >
              ✕
            </button>
          </aside>
        )}

        {/* Main 2-Column Grid */}
        <div className="flex-1 grid grid-cols-1 md:grid-cols-12 gap-0 overflow-hidden">
          {/* ======================================================== */}
          {/* RIGHT SIDEBAR: Devices & Selected Items (4 cols) */}
          {/* ======================================================== */}
          <aside aria-label="الأجهزة المتصلة والعناصر المحددة" className="md:col-span-4 border-l border-white/10 flex flex-col bg-black/20 p-5 overflow-y-auto space-y-5">
            {/* 1. Connected Devices */}
            <div>
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-black text-white/90">الأجهزة المتصلة</span>
                  <span className="flex items-center gap-1 text-[10px] text-emerald-400 font-bold bg-emerald-500/10 border border-emerald-500/20 px-2 py-0.5 rounded-full">
                    <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
                    فحص تلقائي
                  </span>
                </div>
                <button
                  type="button"
                  onClick={() => fetchDevices(true)}
                  disabled={loadingDevices}
                  className="text-[11px] font-bold text-purple-400 hover:text-purple-300 transition"
                >
                  {loadingDevices ? "جاري الفحص..." : "🔄 تحديث"}
                </button>
              </div>

              {devices.length === 0 ? (
                <div className="rounded-2xl border border-dashed border-amber-500/30 bg-amber-950/15 p-4 text-center">
                  <p className="text-2xl mb-1">🔌</p>
                  <p className="text-xs font-bold text-amber-200">لا توجد فلاشة USB أو هاتف متصل</p>
                  <p className="text-[10px] text-white/50 mt-1">
                    قم بتوصيل فلاشة USB أو هاتف بالكمبيوتر وسيتم التعرف عليه فوراً
                  </p>
                </div>
              ) : (
                <div className="space-y-2">
                  {devices.map((dev) => {
                    const isSelected = selectedDevice?.id === dev.id;
                    const devIsIOS = dev.type === "ios" || dev.name?.toLowerCase().includes("iphone");
                    const devIsStorage = dev.type === "storage" || dev.id?.startsWith("disk_");

                    return (
                      <button
                        key={dev.id}
                        type="button"
                        onClick={() => {
                          setSelectedDevice(dev);
                          setBrowsingApp(null);
                        }}
                        className={`w-full flex items-center justify-between rounded-2xl p-3 text-right transition border ${
                          isSelected
                            ? "border-purple-500 bg-purple-950/70 text-white shadow-lg"
                            : "border-white/10 bg-white/[0.02] text-white/70 hover:bg-white/5 hover:text-white"
                        }`}
                      >
                        <div className="flex items-center gap-2.5 truncate">
                          <span className="text-2xl flex-shrink-0">
                            {devIsIOS ? "🍏" : devIsStorage ? "💾" : "🤖"}
                          </span>
                          <div className="truncate text-right">
                            <p className="text-xs font-black text-white truncate max-w-[160px]">
                              {dev.name}
                            </p>
                            <div className="flex items-center gap-1.5 text-[10px] text-purple-300/70 mt-0.5">
                              <span>
                                {devIsIOS ? "هاتف آيفون" : devIsStorage ? "ذاكرة USB" : "هاتف أندرويد"}
                              </span>
                              {dev.free_space > 0 && (
                                <span className="font-mono text-emerald-400 font-bold">
                                  • {formatBytes(dev.free_space)} متاح
                                </span>
                              )}
                            </div>
                          </div>
                        </div>

                        <div className="flex items-center gap-2 flex-shrink-0">
                          {devIsStorage && (
                            <span
                              role="button"
                              tabIndex={0}
                              onClick={(e) => handleEjectDevice(e, dev)}
                              className="text-[10px] rounded-lg border border-white/10 bg-white/5 px-2 py-1 text-white/60 hover:text-red-300 hover:border-red-500/30 transition flex items-center gap-1"
                              title="إخراج آمن للفلاشة"
                            >
                              <span>{ejectingDeviceId === dev.id ? "⏳" : "⏏️"}</span>
                            </span>
                          )}
                          <span
                            className={`h-2.5 w-2.5 rounded-full ${
                              isSelected ? "bg-emerald-400 shadow-[0_0_8px_#34d399]" : "bg-white/20"
                            }`}
                          />
                        </div>
                      </button>
                    );
                  })}
                </div>
              )}
            </div>

            <hr className="border-white/10" />

            {/* 2. Selected Files Summary */}
            <div className="flex-1 flex flex-col min-h-0">
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs font-black text-white/90">العناصر المحددة للنسخ</span>
                <span className="rounded-full bg-purple-600/30 px-2 py-0.5 text-[10px] font-bold text-purple-200">
                  {filesCountLabel}
                </span>
              </div>

              <div className="rounded-xl border border-white/10 bg-white/[0.02] p-3 mb-2 flex items-center justify-between text-xs">
                <span className="text-white/60">الحجم الإجمالي:</span>
                <span className="font-mono font-black text-emerald-400">{sizeLabel}</span>
              </div>

              {/* Scrollable list of selected items */}
              <div className="flex-1 overflow-y-auto space-y-1.5 pr-1 max-h-[180px]">
                {selectedFiles.map((file) => (
                  <div
                    key={file.id || file.filePath}
                    className="flex items-center justify-between rounded-xl border border-white/5 bg-white/[0.02] p-2 text-xs hover:border-white/10 transition"
                  >
                    <div className="truncate flex-1 pl-2 text-right">
                      <p className="text-[11px] font-bold text-white truncate">
                        {file.title}
                      </p>
                      <p className="text-[10px] text-white/40 font-mono">
                        {file.resolution} • {formatBytes(file.size)}
                      </p>
                    </div>
                    {selectedFiles.length > 1 && (
                      <button
                        type="button"
                        onClick={() => removeFileFromSelection(file.id || file.filePath)}
                        className="text-white/40 hover:text-red-400 p-1 text-xs"
                        title="إزالة من القائمة"
                      >
                        ✕
                      </button>
                    )}
                  </div>
                ))}
              </div>
            </div>
          </aside>

          {/* ======================================================== */}
          {/* CENTER CONTENT: Storage Bar + Browser (8 cols) */}
          {/* ======================================================== */}
          <main className="md:col-span-8 flex flex-col p-6 overflow-y-auto bg-black/40">
            {/* Storage Capacity Bar Component (Visible for all devices when space is known) */}
            {selectedDevice && hasSpaceInfo && (
              <div className="rounded-2xl border border-white/10 bg-white/[0.02] p-4 space-y-2.5 mb-4 shadow-sm">
                <div className="flex items-center justify-between text-xs">
                  <span className="font-black text-white flex items-center gap-1.5">
                    <span>📊</span>
                    <span>سعة التخزين في {selectedDevice.name}</span>
                    {selectedDevice.file_system && (
                      <span className="rounded-md bg-purple-500/20 text-purple-300 font-mono text-[10px] px-2 py-0.5 font-bold">
                        {selectedDevice.file_system}
                      </span>
                    )}
                  </span>
                  <span className="text-[11px] font-mono text-white/70">
                    <strong className="text-emerald-400">{formatBytes(deviceFreeSpace)}</strong> متوفر من {formatBytes(deviceTotalSpace)}
                  </span>
                </div>

                {/* Capacity Progress Bar with Used, Projected, and Free */}
                <div className="relative h-3 w-full overflow-hidden rounded-full bg-white/10 flex">
                  {/* Used portion */}
                  <div
                    className="h-full bg-purple-600/70 transition-all duration-300"
                    style={{ width: `${Math.min(100, usedPercent)}%` }}
                    title={`مستخدم: ${formatBytes(usedSpace)}`}
                  />
                  {/* Projected portion */}
                  <div
                    className={`h-full transition-all duration-300 ${
                      isSpaceInsufficient
                        ? "bg-red-500 animate-pulse"
                        : "bg-emerald-400 shadow-[0_0_10px_#34d399]"
                    }`}
                    style={{ width: `${Math.min(100 - usedPercent, projectedPercent)}%` }}
                    title={`الملفات المحددة: ${formatBytes(totalSelectedSize)}`}
                  />
                </div>

                {/* Legend */}
                <div className="flex flex-wrap items-center justify-between gap-2 text-[10px] font-mono text-white/50 pt-1">
                  <div className="flex items-center gap-3">
                    <span className="flex items-center gap-1">
                      <span className="h-2 w-2 rounded-full bg-purple-600"></span>
                      <span>مستخدم: {formatBytes(usedSpace)}</span>
                    </span>
                    <span className="flex items-center gap-1">
                      <span className={`h-2 w-2 rounded-full ${isSpaceInsufficient ? "bg-red-500" : "bg-emerald-400"}`}></span>
                      <span className={isSpaceInsufficient ? "text-red-400 font-bold" : "text-emerald-300 font-bold"}>
                        الملفات المحددة: +{formatBytes(totalSelectedSize)}
                      </span>
                    </span>
                  </div>
                  <span>
                    المتبقي بعد النسخ:{" "}
                    <strong className={isSpaceInsufficient ? "text-red-400 font-bold" : "text-white/80"}>
                      {isSpaceInsufficient ? "غير كافٍ" : formatBytes(Math.max(0, deviceFreeSpace - totalSelectedSize))}
                    </strong>
                  </span>
                </div>

                {isSpaceInsufficient && (
                  <div className="rounded-xl border border-red-500/40 bg-red-950/60 p-2.5 text-xs text-red-200 font-bold flex items-center gap-2">
                    <span>⚠️</span>
                    <span>المساحة المتبقية على القرص غير كافية لنسخ الملفات المحددة. يرجى حذف بعض الملفات أو تفريغ مساحة.</span>
                  </div>
                )}

                {hasFileOver4GB && (
                  <div className="rounded-xl border border-amber-500/40 bg-amber-950/60 p-2.5 text-xs text-amber-200 font-bold flex items-center gap-2">
                    <span>⚠️</span>
                    <span>القرص بنظام FAT32 ولا يقبل ملفات بحجم أكبر من 4 جيجابايت. يرجى تهيئة القرص بنظام NTFS أو exFAT.</span>
                  </div>
                )}
              </div>
            )}

            {/* View A: USB Storage Full Folder Browser */}
            {selectedDevice && isStorage ? (
              <div className="flex-1 flex flex-col space-y-4">
                {/* Browser Header & Breadcrumbs */}
                <div className="flex items-center justify-between rounded-2xl border border-purple-500/30 bg-purple-950/20 p-3">
                  <button
                    type="button"
                    onClick={handleNavigateUp}
                    className="flex items-center gap-1.5 rounded-xl border border-white/15 bg-white/5 px-3 py-1.5 text-xs font-bold text-white hover:bg-white/15 transition"
                  >
                    <span>←</span>
                    <span>المجلد السابق</span>
                  </button>

                  <div className="flex items-center gap-2 truncate text-xs font-mono font-bold text-purple-200">
                    <span className="text-base">💾</span>
                    <span className="text-white font-bold">{selectedDevice.name}:</span>
                    <span className="text-emerald-300 truncate max-w-[280px]" dir="ltr">
                      {currentPath || "الجذر الرئيسي"}
                    </span>
                  </div>

                  <button
                    type="button"
                    onClick={() => setShowNewFolderModal(true)}
                    className="flex items-center gap-1.5 rounded-xl bg-emerald-600/80 px-3 py-1.5 text-xs font-bold text-white hover:bg-emerald-500 transition shadow-md"
                  >
                    <span>+</span>
                    <span>مجلد جديد</span>
                  </button>
                </div>

                {/* Quick Shortcuts for Fast Workflow */}
                <div className="flex items-center gap-2 text-xs">
                  <span className="text-white/50 text-[11px]">اختصارات سريعة:</span>
                  <button
                    type="button"
                    onClick={() => {
                      const letter = selectedDevice.id.replace("disk_", "");
                      loadDirectory(selectedDevice, `${letter}:\\`);
                    }}
                    className="rounded-xl border border-white/10 bg-white/5 px-2.5 py-1 text-[11px] font-bold text-white/80 hover:bg-white/15 transition"
                  >
                    📁 الجذر الرئيسي
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      const letter = selectedDevice.id.replace("disk_", "");
                      loadDirectory(selectedDevice, `${letter}:\\Movies`);
                    }}
                    className="rounded-xl border border-white/10 bg-white/5 px-2.5 py-1 text-[11px] font-bold text-white/80 hover:bg-white/15 transition"
                  >
                    📁 مجلد Movies
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      const letter = selectedDevice.id.replace("disk_", "");
                      loadDirectory(selectedDevice, `${letter}:\\Series`);
                    }}
                    className="rounded-xl border border-white/10 bg-white/5 px-2.5 py-1 text-[11px] font-bold text-white/80 hover:bg-white/15 transition"
                  >
                    📁 مجلد Series
                  </button>
                </div>

                {/* Browser Entries List */}
                <div className="flex-1 rounded-2xl border border-white/10 bg-black/30 p-3 overflow-y-auto min-h-[240px]">
                  {loadingBrowser ? (
                    <div className="flex h-full items-center justify-center text-xs text-white/50 py-12">
                      جاري قراءة محتويات الفلاشة...
                    </div>
                  ) : browserEntries.length === 0 ? (
                    <div className="flex flex-col items-center justify-center h-full text-center text-xs text-white/50 py-10">
                      <span className="text-3xl mb-2">📁</span>
                      <p className="font-bold">المجلد فارغ حالياً</p>
                      <p className="text-[10px] text-white/40 mt-1">
                        سيتم نسخ الملفات المحددة مباشرة إلى هذا المجلد.
                      </p>
                    </div>
                  ) : (
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                      {browserEntries.map((entry) => (
                        <div
                          key={entry.path || entry.name}
                          onClick={() => entry.is_dir && handleNavigateFolder(entry)}
                          className={`flex items-center justify-between rounded-xl border p-3 text-xs transition ${
                            entry.is_dir
                              ? "border-purple-500/30 bg-purple-950/20 hover:bg-purple-900/40 cursor-pointer text-white font-bold"
                              : "border-white/5 bg-white/[0.02] text-white/70"
                          }`}
                        >
                          <div className="flex items-center gap-2.5 truncate">
                            <span className="text-lg">{entry.is_dir ? "📁" : "📄"}</span>
                            <span className="truncate">{entry.name}</span>
                          </div>
                          <span className="text-[10px] text-white/40 font-mono">
                            {entry.is_dir ? "فتح ➔" : formatBytes(entry.size)}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                <div className="rounded-xl border border-white/10 bg-white/[0.02] p-3 text-xs flex items-center justify-between">
                  <span className="text-white/60">مجلد الوجهة المعتمد حالياً:</span>
                  <span className="font-mono text-emerald-400 font-bold" dir="ltr">
                    {currentPath || "الجذر الرئيسي"}
                  </span>
                </div>
              </div>
            ) : selectedDevice && isIOS ? (
              /* View B: iPhone App List or File Browser */
              browsingApp ? (
                <div className="flex-1 flex flex-col space-y-4">
                  <div className="flex items-center justify-between rounded-2xl border border-purple-500/30 bg-purple-950/20 p-3">
                    <button
                      type="button"
                      onClick={handleNavigateUp}
                      className="flex items-center gap-1.5 rounded-xl border border-white/15 bg-white/5 px-3 py-1.5 text-xs font-bold text-white hover:bg-white/15 transition"
                    >
                      <span>←</span>
                      <span>{currentPath === "/Documents" ? "الرجوع للتطبيقات" : "المجلد السابق"}</span>
                    </button>

                    <div className="flex items-center gap-2 truncate text-xs font-mono font-bold text-purple-200">
                      <span className="text-base">{browsingApp.icon}</span>
                      <span className="text-white font-bold">{browsingApp.name}:</span>
                      <span className="text-emerald-300 truncate max-w-[280px]" dir="ltr">
                        {currentPath}
                      </span>
                    </div>

                    <button
                      type="button"
                      onClick={() => setShowNewFolderModal(true)}
                      className="flex items-center gap-1.5 rounded-xl bg-emerald-600/80 px-3 py-1.5 text-xs font-bold text-white hover:bg-emerald-500 transition shadow-md"
                    >
                      <span>+</span>
                      <span>مجلد جديد</span>
                    </button>
                  </div>

                  <div className="flex-1 rounded-2xl border border-white/10 bg-black/30 p-3 overflow-y-auto min-h-[220px]">
                    {loadingBrowser ? (
                      <div className="flex h-full items-center justify-center text-xs text-white/50 py-12">
                        جاري تصفح ملفات التطبيق...
                      </div>
                    ) : browserEntries.length === 0 ? (
                      <div className="flex flex-col items-center justify-center h-full text-center text-xs text-white/50 py-8">
                        <span className="text-3xl mb-2">📁</span>
                        <p className="font-bold">المجلد فارغ حالياً</p>
                        <p className="text-[10px] text-white/40 mt-1">
                          سيتم نسخ الملفات المحددة إلى هذا المسار مباشرة.
                        </p>
                      </div>
                    ) : (
                      <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                        {browserEntries.map((entry) => (
                          <div
                            key={entry.path || entry.name}
                            onClick={() => entry.is_dir && handleNavigateFolder(entry)}
                            className={`flex items-center justify-between rounded-xl border p-3 text-xs transition ${
                              entry.is_dir
                                ? "border-purple-500/30 bg-purple-950/20 hover:bg-purple-900/40 cursor-pointer text-white font-bold"
                                : "border-white/5 bg-white/[0.02] text-white/70"
                            }`}
                          >
                            <div className="flex items-center gap-2.5 truncate">
                              <span className="text-lg">{entry.is_dir ? "📁" : "📄"}</span>
                              <span className="truncate">{entry.name}</span>
                            </div>
                            <span className="text-[10px] text-white/40 font-mono">
                              {entry.is_dir ? "مجلد ➔" : formatBytes(entry.size)}
                            </span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>

                  <div className="rounded-xl border border-white/10 bg-white/[0.02] p-3 text-xs flex items-center justify-between">
                    <span className="text-white/60">الوجهة المستهدفة الحالية:</span>
                    <span className="font-mono text-emerald-400 font-bold" dir="ltr">
                      {currentPath}
                    </span>
                  </div>
                </div>
              ) : (
                <div className="space-y-4">
                  <div className="flex items-center justify-between border-b border-white/10 pb-3">
                    <h3 className="text-xs font-black text-white">تطبيقات iPhone الداعمة لمشاركة الملفات</h3>
                    <span className="rounded-full bg-emerald-500/20 px-2.5 py-0.5 text-[10px] font-bold text-emerald-300">
                      File Sharing جاهز
                    </span>
                  </div>

                  {loadingApps ? (
                    <div className="py-12 text-center text-xs text-white/50">
                      جاري فحص تطبيقات الميديا في iPhone...
                    </div>
                  ) : deviceApps.length === 0 ? (
                    <div className="rounded-2xl border border-yellow-500/30 bg-yellow-950/20 p-6 text-center text-xs">
                      <p className="text-3xl mb-2">📲</p>
                      <p className="font-bold text-yellow-200">
                        لم يتم العثور على تطبيق يدعم مشاركة مجلد Documents
                      </p>
                      <p className="text-[11px] text-white/60 mt-1">
                        يرجى تثبيت تطبيق مثل VLC أو Infuse أو Documents ثم إعادة الفحص.
                      </p>
                    </div>
                  ) : (
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                      {deviceApps.map((app) => {
                        const isSelected = selectedApp?.bundle_id === app.bundle_id;
                        return (
                          <div
                            key={app.bundle_id}
                            className={`rounded-2xl border p-4 transition flex flex-col justify-between ${
                              isSelected
                                ? "border-purple-500 bg-purple-950/50 shadow-lg"
                                : "border-white/10 bg-white/[0.02] hover:bg-white/5"
                            }`}
                          >
                            <div className="flex items-center gap-3 mb-3">
                              <span className="text-3xl">{app.icon}</span>
                              <div className="truncate">
                                <h4 className="text-xs font-black text-white">{app.name}</h4>
                                <p className="text-[10px] text-purple-300/60 font-mono truncate" dir="ltr">
                                  {app.bundle_id}
                                </p>
                              </div>
                            </div>

                            <div className="flex items-center gap-2 mt-2">
                              <button
                                type="button"
                                onClick={() => handleOpenAppBrowser(app)}
                                className="flex-1 rounded-xl bg-purple-600/80 py-2 text-xs font-black text-white hover:bg-purple-500 transition shadow-md"
                              >
                                📂 عرض المجلدات
                              </button>
                              <button
                                type="button"
                                onClick={() => {
                                  setSelectedApp(app);
                                  setCustomBundleId(app.bundle_id);
                                }}
                                className="rounded-xl border border-white/15 bg-white/5 px-3 py-2 text-xs font-bold text-white/80 hover:bg-white/15 transition"
                              >
                                اختيار
                              </button>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  )}

                  <div className="rounded-2xl border border-white/10 bg-white/[0.02] p-4 space-y-2">
                    <label className="block text-[11px] font-bold text-white/70">
                      تحديد Bundle ID يدويًا (اختياري):
                    </label>
                    <input
                      type="text"
                      value={customBundleId}
                      onChange={(e) => setCustomBundleId(e.target.value)}
                      placeholder="org.videolan.vlc-ios"
                      dir="ltr"
                      className="w-full rounded-xl border border-white/15 bg-black/40 px-3 py-2 text-xs font-mono font-bold text-white outline-none focus:border-purple-500"
                    />
                  </div>
                </div>
              )
            ) : selectedDevice ? (
              /* View C: Android MTP Options */
              <div className="space-y-4">
                <div className="flex items-center justify-between border-b border-white/10 pb-3">
                  <h3 className="text-xs font-black text-white">مجلدات التخزين في {selectedDevice.name}</h3>
                  <span className="rounded-full bg-blue-500/20 px-2.5 py-0.5 text-[10px] font-bold text-blue-300">
                    هاتف متصل
                  </span>
                </div>

                <div className="grid grid-cols-3 gap-2">
                  {["Movies", "Download", "DCIM"].map((folder) => (
                    <button
                      key={folder}
                      type="button"
                      onClick={() => setAndroidTargetFolder(folder)}
                      className={`rounded-2xl p-3.5 text-center text-xs font-bold transition border ${
                        androidTargetFolder === folder
                          ? "border-purple-500 bg-purple-900/60 text-white shadow-md"
                          : "border-white/10 bg-white/5 text-white/70 hover:bg-white/10"
                      }`}
                    >
                      📁 {folder}
                    </button>
                  ))}
                </div>

                <div className="rounded-2xl border border-white/10 bg-white/[0.02] p-4 space-y-2">
                  <label className="block text-[11px] font-bold text-white/70">
                    اسم المجلد الفرعي (يُنشأ تلقائيًا داخل {androidTargetFolder}):
                  </label>
                  <input
                    type="text"
                    value={androidSubFolder}
                    onChange={(e) => setAndroidSubFolder(e.target.value)}
                    placeholder="مثال: Game of Thrones Season 1"
                    className="w-full rounded-xl border border-white/15 bg-black/40 px-3 py-2.5 text-xs font-bold text-white outline-none focus:border-purple-500"
                  />
                </div>
              </div>
            ) : (
              <div className="flex h-full items-center justify-center text-xs text-white/40 py-16">
                يرجى اختيار جهاز أو فلاشة من القائمة الجانبية
              </div>
            )}
          </main>
        </div>

        {/* ======================================================== */}
        {/* MODAL FOOTER */}
        {/* ======================================================== */}
        <footer className="border-t border-white/10 p-4 bg-white/[0.02] flex flex-col sm:flex-row items-center justify-between gap-3">
          <button
            type="button"
            onClick={handleOpenFileLocation}
            disabled={openingFolder || selectedFiles.length === 0}
            className="text-xs font-bold text-purple-300 hover:text-purple-200 transition flex items-center gap-1.5"
          >
            <span>📂</span>
            <span>{openingFolder ? "جاري الفتح..." : "فتح موضع الملف في الكمبيوتر"}</span>
          </button>

          <button
            type="button"
            onClick={handleStartCopy}
            disabled={isSubmitting || !selectedDevice || selectedFiles.length === 0 || isSpaceInsufficient || hasFileOver4GB}
            className={`flex items-center justify-center gap-2 rounded-2xl px-8 py-3 text-xs font-black text-white shadow-xl transition active:scale-95 ${
              isSubmitting || !selectedDevice || selectedFiles.length === 0 || isSpaceInsufficient || hasFileOver4GB
                ? "bg-white/10 text-white/30 cursor-not-allowed"
                : "bg-gradient-to-r from-purple-600 via-fuchsia-600 to-emerald-500 hover:brightness-110 shadow-purple-900/40"
            }`}
          >
            <span className="text-base">⬇</span>
            <span>
              {isSubmitting
                ? "جاري إرسال الطلب..."
                : isSpaceInsufficient
                ? "المساحة غير كافية للنسخ"
                : hasFileOver4GB
                ? "يتجاوز حد 4GB لنظام FAT32"
                : `نسخ ${filesCountLabel} • ${sizeLabel}`}
            </span>
          </button>
        </footer>
      </div>

      {/* Inline New Folder Modal */}
      {showNewFolderModal && (
        <div className="fixed inset-0 z-60 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm">
          <form
            onSubmit={handleCreateFolder}
            className="w-full max-w-sm rounded-2xl border border-purple-500/40 bg-[#120F28] p-5 shadow-2xl space-y-4"
          >
            <h4 className="text-xs font-black text-white">إنشاء مجلد جديد على الجهاز</h4>
            <input
              type="text"
              autoFocus
              value={newFolderName}
              onChange={(e) => setNewFolderName(e.target.value)}
              placeholder="اسم المجلد الجديد"
              className="w-full rounded-xl border border-white/15 bg-black/40 px-3 py-2 text-xs font-bold text-white outline-none focus:border-purple-500"
            />
            <div className="flex items-center justify-end gap-2">
              <button
                type="button"
                onClick={() => setShowNewFolderModal(false)}
                className="rounded-xl border border-white/10 px-3 py-1.5 text-xs text-white/60 hover:bg-white/5"
              >
                إلغاء
              </button>
              <button
                type="submit"
                disabled={creatingFolder || !newFolderName.trim()}
                className="rounded-xl bg-purple-600 px-4 py-1.5 text-xs font-bold text-white hover:bg-purple-500 disabled:opacity-50"
              >
                {creatingFolder ? "جاري الإنشاء..." : "إنشاء"}
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
}
