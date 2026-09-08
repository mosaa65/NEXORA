import React, { useEffect, useState } from "react";
import {
  getTransferDevices,
  getTransferDeviceApps,
  getTransferAppFolders,
  startDeviceTransfer,
  getTransferJob,
  cancelTransferJob,
  openFileLocation
} from "../lib/api.js";
import { normalizeTransferDevices } from "../lib/transferDevices.js";

function formatBytes(value) {
  const size = Number(value || 0);
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

function getTransferPhaseLabel(job) {
  if (!job) return "جاري التحضير";
  if (job.status === "failed") return "فشل النسخ";
  if (job.status === "cancelled") return "تم الإلغاء";
  if (job.status === "completed") return "اكتمل النسخ";
  switch (job.phase) {
    case "queued":
      return "في قائمة الانتظار";
    case "preparing":
      return "جاري التحضير";
    case "copying":
      return "جاري النسخ";
    case "verifying":
      return "جاري التحقق من الملف";
    default:
      return "جاري النقل عبر USB";
  }
}

export default function CopyToPhoneModal({ file, mediaTitle, onClose }) {
  const [devices, setDevices] = useState([]);
  const [loadingDevices, setLoadingDevices] = useState(true);
  const [selectedDevice, setSelectedDevice] = useState(null);
  
  const [deviceApps, setDeviceApps] = useState([]);
  const [loadingApps, setLoadingApps] = useState(false);
  const [selectedApp, setSelectedApp] = useState(null);
  const [customBundleId, setCustomBundleId] = useState("");
  const [appFolders, setAppFolders] = useState([]);
  const [selectedFolder, setSelectedFolder] = useState("");
  const [loadingFolders, setLoadingFolders] = useState(false);

  const [targetFolder, setTargetFolder] = useState("Movies");
  const [subFolder, setSubFolder] = useState(mediaTitle || "");
  
  const [activeJob, setActiveJob] = useState(null);
  const [isCopying, setIsCopying] = useState(false);
  const [errorMsg, setErrorMsg] = useState("");
  const [openingFolder, setOpeningFolder] = useState(false);

  const fetchDevices = () => {
    if (isCopying) return;

    setLoadingDevices(true);
    setErrorMsg("");
    getTransferDevices()
      .then((res) => {
        const devList = normalizeTransferDevices(res.devices || []);
        setDevices(devList);
        setSelectedDevice((prev) => {
          if (prev) {
            const found = devList.find((d) => d.id === prev.id);
            return found || prev;
          }
          return devList[0] || null;
        });
      })
      .catch((err) => {
        setErrorMsg("تعذر جلب الأجهزة الموصولة: " + (err.message || "تأكد من تشغيل السيرفر"));
      })
      .finally(() => setLoadingDevices(false));
  };

  useEffect(() => {
    fetchDevices();
  }, []);

  useEffect(() => {
    if (isCopying || devices.length > 0) return undefined;

    const timer = setInterval(fetchDevices, 2500);
    return () => clearInterval(timer);
  }, [devices.length, isCopying]);

  useEffect(() => {
    if (mediaTitle && !subFolder) {
      setSubFolder(mediaTitle);
    }
  }, [mediaTitle]);

  useEffect(() => {
    if (!selectedDevice) return;

    setDeviceApps([]);
    setSelectedApp(null);
    setAppFolders([]);
    setErrorMsg("");

    if (selectedDevice.type === "ios" || selectedDevice.name?.toLowerCase().includes("iphone")) {
      setLoadingApps(true);
      getTransferDeviceApps(selectedDevice.id)
        .then((res) => {
          const apps = (res.apps || []).filter((app) => app.file_sharing_enabled);
          setDeviceApps(apps);
          const fallbackApp = apps[0] || null;
          setSelectedApp(fallbackApp);
          if (fallbackApp) {
            setCustomBundleId(fallbackApp.bundle_id || "");
          }
        })
        .catch((err) => {
          setDeviceApps([]);
          setSelectedApp(null);
          setErrorMsg(err.message || "لا يوجد تطبيق يدعم مشاركة الملفات على هذا الآيفون");
        })
        .finally(() => setLoadingApps(false));
    }
  }, [selectedDevice?.id]);

  useEffect(() => {
    if (!selectedDevice || !(selectedDevice.type === "ios" || selectedDevice.name?.toLowerCase().includes("iphone"))) return;
    const bundleId = customBundleId.trim() || selectedApp?.bundle_id || "";
    if (!bundleId) {
      setAppFolders([]);
      setSelectedFolder("");
      return;
    }

    setLoadingFolders(true);
    getTransferAppFolders(selectedDevice.id, bundleId)
      .then((res) => {
        const folders = (res.folders || []).filter((folder) => String(folder.name || "").trim() !== "");
        setAppFolders(folders);
        setSelectedFolder((prev) => prev || "");
      })
      .catch(() => {
        setAppFolders([]);
        setSelectedFolder("");
      })
      .finally(() => setLoadingFolders(false));
  }, [selectedDevice?.id, selectedApp?.bundle_id, customBundleId]);

  useEffect(() => {
    if (!activeJob?.id || !isCopying) return;

    const timer = setInterval(() => {
      getTransferJob(activeJob.id)
        .then((updatedJob) => {
          setActiveJob(updatedJob);
          if (updatedJob.status === "completed") {
            setIsCopying(false);
          } else if (updatedJob.status === "failed" || updatedJob.status === "cancelled") {
            setIsCopying(false);
            if (updatedJob.status === "failed") {
              setErrorMsg(updatedJob.error || "فشلت عملية النسخ");
            }
          }
        })
        .catch(() => {});
    }, 500);

    return () => clearInterval(timer);
  }, [activeJob?.id, isCopying]);

  const handleStartCopy = async (customApp, chosenFolder = "") => {
    const appToUse = customApp || selectedApp;
    const targetBundleId = appToUse?.bundle_id || customBundleId.trim() || "";
    const selectedDeviceIsIOS = selectedDevice?.type === "ios" || selectedDevice?.name?.toLowerCase().includes("iphone");

    if (!selectedDevice) {
      setErrorMsg("يرجى اختيار الهاتف الموصول أولاً");
      return;
    }
    if (selectedDeviceIsIOS && !targetBundleId) {
      setErrorMsg("اختر تطبيق iPhone أو أدخل Bundle ID يدوي يسمح بالنسخ عبر Documents");
      return;
    }
    if (selectedDeviceIsIOS && !customBundleId.trim() && appToUse && !appToUse.file_sharing_enabled) {
      setErrorMsg("هذا التطبيق لا يتيح مشاركة الملفات عبر Documents. اختر تطبيقاً عليه علامة جاهز أو أدخل Bundle ID يدوي تعرف أنه يدعم File Sharing.");
      return;
    }
    setErrorMsg("");
    setIsCopying(true);

    const destinationFolder = chosenFolder || selectedFolder || "";
    setSelectedFolder(destinationFolder);

    try {
      const payload = {
        device_id: selectedDevice.id,
        file_id: file?.id || 0,
        source_path: file?.filePath || file?.file_path || file?.path || file?.source_path || "",
        target_app: targetBundleId,
        target_folder: targetFolder,
        sub_folder: destinationFolder
      };

      const res = await startDeviceTransfer(payload);
      if (res.ok && res.job) {
        setActiveJob(res.job);
      } else {
        throw new Error(res.error || "فشل بدء النسخ");
      }
    } catch (err) {
      setIsCopying(false);
      setErrorMsg(err.message || "حدث خطأ أثناء إرسال طلب النسخ");
    }
  };

  const handleOpenFileFolder = async () => {
    setOpeningFolder(true);
    try {
      await openFileLocation({
        file_id: file?.id || 0,
        path: file?.filePath || file?.file_path || file?.path || file?.source_path || ""
      });
    } catch (err) {
      setErrorMsg("تعذر فتح مجلد الفيديو في الكمبيوتر: " + (err.message || ""));
    } finally {
      setOpeningFolder(false);
    }
  };

  const handleCancelCopy = async () => {
    if (activeJob?.id) {
      try {
        await cancelTransferJob(activeJob.id);
        setIsCopying(false);
      } catch {}
    }
  };

  const fileName = file?.title_ar || file?.title_en || (file?.episode_number ? `الحلقة ${file.episode_number}` : "ملف الفيديو");
  const displayTitle = mediaTitle ? `${mediaTitle} - ${fileName}` : fileName;
  const isIOS = selectedDevice?.type === "ios" || selectedDevice?.name?.toLowerCase().includes("iphone");
  const canStartCopy = selectedDevice && devices.length > 0 && (!isIOS || customBundleId.trim() || selectedApp?.file_sharing_enabled);
  const progressValue = Math.max(0, Math.min(100, Number(activeJob?.progress || 0)));
  const transferredLabel = formatBytes(activeJob?.transferred || 0);
  const fileSizeLabel = formatBytes(activeJob?.file_size || file?.file_size || 0);
  const phaseLabel = getTransferPhaseLabel(activeJob);

  const renderFolderTree = (folders = [], app, depth = 0) => (
    <div className="space-y-1">
      {folders.map((folder) => {
        const folderPath = folder.path || folder.name;
        const children = Array.isArray(folder.children) ? folder.children : [];
        return (
          <div key={`${app.bundle_id}-${folderPath}`} className="space-y-1">
            <div className="flex items-center justify-between rounded-lg border border-white/10 bg-white/5 px-2 py-2" style={{ marginLeft: `${depth * 12}px` }}>
              <button
                type="button"
                onClick={() => {
                  setSelectedFolder(folderPath);
                  handleStartCopy(app, folderPath);
                }}
                className="flex flex-1 items-center gap-2 text-right text-[11px] font-bold text-white hover:text-emerald-300"
              >
                <span>📁</span>
                <span>{folder.name}</span>
              </button>
              {children.length > 0 && (
                <span className="text-[9px] text-purple-300">{children.length}</span>
              )}
            </div>
            {children.length > 0 && (
              <div className="border-r border-white/10 pr-2">
                {renderFolderTree(children, app, depth + 1)}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/90 p-4 backdrop-blur-xl text-right" dir="rtl">
      <div className="relative w-full max-w-2xl max-h-[90vh] overflow-y-auto rounded-3xl border border-purple-500/30 bg-[#0F0E1E] p-6 shadow-2xl">
        {/* Header */}
        <div className="mb-5 flex items-center justify-between border-b border-white/10 pb-4">
          <button
            type="button"
            onClick={onClose}
            className="flex h-8 w-8 items-center justify-center rounded-full border border-white/15 bg-white/5 text-white/70 transition hover:bg-white/15 hover:text-white"
          >
            ✕
          </button>
          <div className="flex items-center gap-3">
            <span className="text-3xl">{isIOS ? "🍏" : "📱"}</span>
            <div>
              <h3 className="text-lg font-black text-white">
                {isIOS ? "نسخ الفيديو إلى جهاز iPhone" : "نسخ الفيديو إلى الهاتف"}
              </h3>
              <p className="text-xs text-purple-300/80">نقل ميديا مباشر وسريع عبر كابل USB</p>
            </div>
          </div>
        </div>

        {/* Media Preview Badge */}
        <div className="mb-4 rounded-2xl border border-purple-500/20 bg-purple-950/30 p-3 text-xs">
          <div className="flex items-center justify-between">
            <span className="rounded-md bg-purple-600/40 px-2 py-0.5 font-mono font-bold text-purple-300">{file?.resolution || "1080p"}</span>
            <p className="font-bold text-white truncate max-w-[400px]">{displayTitle}</p>
          </div>
        </div>

        {/* Alerts */}
        {errorMsg && (
          <div className="mb-4 rounded-xl border border-red-500/40 bg-red-950/60 p-3 text-xs font-bold text-red-300">
            ⚠️ {errorMsg}
          </div>
        )}

        {/* Success View */}
        {activeJob?.status === "completed" ? (
          <div className="my-4 space-y-4 rounded-2xl border border-emerald-500/40 bg-emerald-950/30 p-5 text-right">
            <div className="flex items-center gap-3 text-emerald-300">
              <span className="text-4xl">🎉</span>
              <div>
                <h4 className="text-base font-black text-white">تم نقل الفيديو إلى الهاتف بنجاح 100%!</h4>
                <p className="text-xs text-emerald-200">الملف جاهز للتشغيل مباشرة على جهاز المستخدم.</p>
              </div>
            </div>

            <div className="rounded-xl border border-emerald-500/30 bg-black/50 p-4 text-xs space-y-2">
              <p className="font-bold text-emerald-400">📍 المسار النهائي في الهاتف:</p>
              <div className="rounded-lg bg-white/5 p-2.5 font-mono text-[11px] text-white/90 dir-ltr text-right break-all leading-relaxed">
                {activeJob.destination_path || `الهاتف > Documents > ${fileName}`}
              </div>
            </div>

            <button
              type="button"
              onClick={onClose}
              className="mt-3 w-full rounded-xl bg-emerald-600 py-3 text-xs font-black text-white transition hover:bg-emerald-500 shadow-lg"
            >
              تم، إغلاق النافذة
            </button>
          </div>
        ) : isCopying ? (
          /* Progress View */
          <div className="my-6 space-y-4 rounded-2xl border border-purple-500/30 bg-purple-950/20 p-5">
            <div className="flex items-center justify-between text-xs font-bold">
              <span className="font-mono text-purple-300">
                {activeJob?.speed_mbps ? `${activeJob.speed_mbps.toFixed(1)} MB/s` : phaseLabel}
              </span>
              <span className="text-white">
                {phaseLabel} ({progressValue.toFixed(0)}%)
              </span>
            </div>

            <div className="h-3 w-full overflow-hidden rounded-full bg-white/10 p-0.5">
              <div
                className="h-full rounded-full bg-gradient-to-r from-purple-500 via-fuchsia-500 to-emerald-400 transition-all duration-300"
                style={{ width: `${progressValue}%` }}
              />
            </div>
            <div className="flex items-center justify-between text-[11px] font-bold text-white/55">
              <span>{transferredLabel} / {fileSizeLabel}</span>
              <span>{activeJob?.destination_path || "جاري تحديد وجهة النسخ..."}</span>
            </div>

            <button
              type="button"
              onClick={handleCancelCopy}
              className="mt-2 w-full rounded-xl border border-red-500/30 bg-red-950/40 py-2 text-xs font-bold text-red-300 transition hover:bg-red-900/60"
            >
              إلغاء العملية
            </button>
          </div>
        ) : (
          /* Main Form View */
          <div className="space-y-5">
            {/* Step 1: Device Selection */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <button
                  type="button"
                  onClick={fetchDevices}
                  className="text-[11px] font-bold text-purple-400 hover:underline"
                >
                  🔄 إعادة فحص الأجهزة المتصلة
                </button>
                <h4 className="text-xs font-bold text-white/90">1. اختر الهاتف الموصول بـ USB:</h4>
              </div>

              {loadingDevices ? (
                <div className="py-4 text-center text-xs text-white/50">
                  جاري فحص منافذ USB والأجهزة الموصولة...
                </div>
              ) : devices.length === 0 ? (
                <div className="rounded-2xl border border-yellow-500/30 bg-yellow-950/20 p-4 text-center">
                  <p className="text-xl mb-1">🔌</p>
                  <p className="text-xs font-bold text-yellow-200">قم بتوصيل الهاتف عبر كابل USB واضغط "إعادة فحص"</p>
                </div>
              ) : (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                  {devices.map((device) => {
                    const isSelected = selectedDevice?.id === device.id;
                    const isDeviceIOS = device.type === "ios" || device.name?.toLowerCase().includes("iphone");
                    return (
                      <button
                        key={device.id}
                        type="button"
                        onClick={() => setSelectedDevice(device)}
                        className={`flex items-center justify-between rounded-xl p-3 text-right transition border ${
                          isSelected
                            ? "border-purple-500 bg-purple-950/60 text-white shadow-lg"
                            : "border-white/10 bg-white/[0.02] text-white/70 hover:bg-white/5 hover:text-white"
                        }`}
                      >
                        <span className="text-xl">{isDeviceIOS ? "🍏" : "🤖"}</span>
                        <div className="text-right">
                          <p className="text-xs font-bold text-white">{device.name}</p>
                          <p className="text-[10px] text-white/40">{isDeviceIOS ? "آيفون iOS" : "أندرويد (MTP)"}</p>
                        </div>
                      </button>
                    );
                  })}
                </div>
              )}
            </div>

            {/* Step 2: iOS Target App or Android Folders */}
            {selectedDevice && isIOS ? (
              /* iOS Apps Selection */
              <div className="space-y-3 rounded-2xl border border-purple-500/30 bg-purple-950/10 p-4">
                <div className="flex items-center justify-between border-b border-white/10 pb-2">
                  <span className="rounded-full bg-emerald-500/20 px-2.5 py-0.5 text-[10px] font-bold text-emerald-300">
                    تطبيقات الميديا المتاحة
                  </span>
                  <h4 className="text-xs font-bold text-white">
                    2. اختر التطبيق المراد نقل الفيديو إليه بالآيفون:
                  </h4>
                </div>

                {loadingApps ? (
                  <div className="rounded-xl border border-white/10 bg-white/[0.02] p-4 text-center text-xs text-white/60">
                    جاري فحص تطبيقات الميديا التي تسمح بالنسخ...
                  </div>
                ) : deviceApps.length === 0 ? (
                  <div className="rounded-xl border border-yellow-500/30 bg-yellow-950/20 p-4 text-center text-xs text-yellow-200">
                    لم يتم العثور على تطبيق مثبت يتيح مشاركة مجلد Documents. ثبّت VLC أو Documents ثم أعد الفحص.
                  </div>
                ) : (
                  <div className="space-y-3">
                    {deviceApps.map((app) => {
                      const isAppSelected = selectedApp?.bundle_id === app.bundle_id;
                      const isFolderSelected = (customBundleId.trim() || selectedApp?.bundle_id) === app.bundle_id;
                      const appFoldersList = isFolderSelected
                        ? appFolders.filter((folder) => String(folder.name || "").trim() !== "" && String(folder.name || "").trim() !== "المجلد الرئيسي")
                        : [];

                      return (
                        <div
                          key={app.bundle_id}
                          className={`rounded-xl border p-3 transition ${
                            isAppSelected
                              ? "border-purple-500 bg-purple-900/40 text-white shadow-md"
                              : "border-white/10 bg-white/[0.02] text-white/80 hover:bg-white/5"
                          }`}
                        >
                          <div
                            className="flex cursor-pointer items-center justify-between"
                            onClick={() => {
                              setSelectedApp(app);
                              setCustomBundleId(app.bundle_id || "");
                            }}
                          >
                            <div className="flex items-center gap-3 text-right">
                              <div>
                                <div className="flex items-center gap-1.5 justify-end">
                                  <span className="rounded-full bg-emerald-500/15 px-2 py-0.5 text-[9px] font-black text-emerald-300">جاهز</span>
                                  <span className="text-xs font-bold text-white">{app.name}</span>
                                  <span className="text-base">{app.icon}</span>
                                </div>
                                <p className="text-[10px] text-purple-300/70" dir="ltr">{app.bundle_id}</p>
                              </div>
                            </div>
                            <button
                              type="button"
                              onClick={(e) => {
                                e.stopPropagation();
                                handleStartCopy(app, selectedFolder || "");
                              }}
                              className="inline-flex items-center justify-center gap-1.5 rounded-xl bg-gradient-to-r from-emerald-600 to-teal-600 px-4 py-2 text-xs font-black text-white shadow-lg hover:brightness-110"
                            >
                              ⚡ نسخ مباشر
                            </button>
                          </div>

                          <div className="mt-3 flex flex-wrap items-center gap-2">
                            {(loadingFolders && isFolderSelected) ? (
                              <span className="text-[10px] text-purple-300">جارٍ جلب مجلدات التطبيق...</span>
                            ) : (
                              <>
                                <button
                                  type="button"
                                  onClick={() => {
                                    setCustomBundleId(app.bundle_id);
                                    setSelectedApp(app);
                                    setSelectedFolder("");
                                  }}
                                  className="rounded-lg border border-white/10 bg-white/5 px-2 py-1 text-[10px] font-bold text-white/80 hover:bg-white/10"
                                >
                                  ✓ اختيار هذا التطبيق
                                </button>
                                <button
                                  type="button"
                                  onClick={() => {
                                    setCustomBundleId(app.bundle_id);
                                    setSelectedApp(app);
                                    setSelectedFolder("");
                                  }}
                                  className="rounded-lg border border-blue-500/30 bg-blue-500/10 px-2 py-1 text-[10px] font-bold text-blue-200 hover:bg-blue-500/20"
                                >
                                  📂 فتح مجلدات التطبيق
                                </button>
                              </>
                            )}
                          </div>

                          {isFolderSelected && (
                            <div className="mt-3 rounded-xl border border-white/10 bg-black/20 p-2">
                              <div className="mb-2 flex items-center justify-between">
                                <span className="text-[10px] font-bold text-purple-200">/Documents</span>
                                <span className="text-[9px] text-white/60">{appFoldersList.length} مجلد</span>
                              </div>

                              {appFoldersList.length === 0 ? (
                                <div className="rounded-lg border border-dashed border-white/10 bg-white/5 px-2 py-3 text-center text-[10px] font-bold text-white/60">
                                  لا توجد مجلدات داخل التطبيق، سيتم النسخ إلى المجلد الرئيسي
                                </div>
                              ) : (
                                renderFolderTree(appFoldersList, app)
                              )}
                            </div>
                          )}
                        </div>
                      );
                    })}
                  </div>
                )}
                <div className="grid gap-2 pt-2 sm:grid-cols-2">
                  <label className="block text-[11px] font-bold text-white/70">
                    Bundle ID يدوي
                    <input
                      type="text"
                      value={customBundleId}
                      onChange={(e) => setCustomBundleId(e.target.value)}
                      placeholder="com.example.player"
                      dir="ltr"
                      className="mt-1 w-full rounded-xl border border-white/15 bg-black/40 px-3 py-2 text-xs font-bold text-white outline-none focus:border-purple-500"
                    />
                  </label>
                  <label className="block text-[11px] font-bold text-white/70">
                    المسار داخل التطبيق
                    <input
                      type="text"
                      value={subFolder}
                      onChange={(e) => setSubFolder(e.target.value)}
                      placeholder="Movies/Series"
                      className="mt-1 w-full rounded-xl border border-white/15 bg-black/40 px-3 py-2 text-xs font-bold text-white outline-none focus:border-purple-500"
                    />
                  </label>
                </div>
              </div>
            ) : selectedDevice && (
              /* Android Target Folder Selection */
              <div className="space-y-3 rounded-2xl border border-white/10 bg-white/[0.02] p-4">
                <h4 className="text-xs font-bold text-white/90">2. تخصيص مجلد الوجهة بالهاتف:</h4>
                <div className="grid grid-cols-3 gap-2">
                  {["Movies", "Download", "DCIM"].map((f) => (
                    <button
                      key={f}
                      type="button"
                      onClick={() => setTargetFolder(f)}
                      className={`rounded-xl p-2.5 text-center text-xs font-bold transition border ${
                        targetFolder === f
                          ? "border-purple-500 bg-purple-900/60 text-white"
                          : "border-white/10 bg-white/5 text-white/60 hover:bg-white/10"
                      }`}
                    >
                      📁 {f}
                    </button>
                  ))}
                </div>

                <div>
                  <label className="block text-[11px] font-bold text-white/70 mb-1">
                    اسم المجلد الفرعي (سيتم إنشاؤه تلقائياً):
                  </label>
                  <input
                    type="text"
                    value={subFolder}
                    onChange={(e) => setSubFolder(e.target.value)}
                    placeholder="مثال: سند فيصل الجماعي"
                    className="w-full rounded-xl border border-white/15 bg-black/40 px-3 py-2 text-xs font-bold text-white outline-none focus:border-purple-500"
                  />
                </div>
              </div>
            )}

            {/* Quick Open File Location in Windows Explorer */}
            <div className="flex items-center justify-between rounded-xl border border-blue-500/20 bg-blue-950/20 p-3 text-xs">
              <button
                type="button"
                onClick={handleOpenFileFolder}
                disabled={openingFolder}
                className="inline-flex items-center justify-center gap-1.5 rounded-lg bg-blue-600/80 px-3 py-1.5 text-[11px] font-bold text-white hover:bg-blue-500 disabled:opacity-50"
              >
                <span>📂 {openingFolder ? "جاري الفتح..." : "فتح موضع الملف في الكمبيوتر"}</span>
              </button>
              <p className="text-[11px] text-blue-200/80">💡 يمكنك فتح مجلد الفيديو مباشرة وسحبه إلى أي مكان على الكمبيوتر.</p>
            </div>

            {/* Big Action Button */}
            <button
              type="button"
              disabled={!canStartCopy}
              onClick={() => handleStartCopy(selectedApp)}
              className={`w-full rounded-2xl py-3.5 text-xs font-black transition shadow-lg ${
                canStartCopy
                  ? "bg-gradient-to-r from-purple-600 via-fuchsia-600 to-emerald-600 text-white hover:brightness-110 shadow-purple-900/50"
                  : "bg-white/10 text-white/30 cursor-not-allowed"
              }`}
            >
              🚀 {isIOS ? `ابدأ النسخ المباشر إلى ${customBundleId.trim() || selectedApp?.name || "الآيفون"}` : "ابدأ النسخ المباشر إلى الهاتف"}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
