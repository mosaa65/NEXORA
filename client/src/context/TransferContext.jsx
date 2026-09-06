import React, { createContext, useContext, useState, useEffect, useRef, useCallback } from "react";
import {
  getTransferJobs,
  startDeviceTransfer,
  cancelTransferJob,
  resolveAPIURL
} from "../lib/api.js";

const TransferContext = createContext(null);

export function TransferProvider({ children }) {
  // 1. Selection State
  const [selectedFiles, setSelectedFiles] = useState([]);
  const [isTransferModalOpen, setIsTransferModalOpen] = useState(false);

  // 2. Active Jobs & Mini Transfer Center State
  const [activeJobs, setActiveJobs] = useState([]);
  const [devices, setDevices] = useState([]);
  const [isCenterExpanded, setIsCenterExpanded] = useState(false);
  const [centerDismissed, setCenterDismissed] = useState(false);

  // Live updates come over a single Server-Sent Events stream: job progress
  // (throttled to ~2 Hz server-side) and instant device hot-plug changes.
  // HTTP is only used for the explicit start/cancel commands below.
  const applyJobEvent = useCallback((job) => {
    if (!job || !job.id) return;
    setActiveJobs((prev) => {
      const index = prev.findIndex((j) => j.id === job.id);
      if (index === -1) return [job, ...prev];
      const next = prev.slice();
      next[index] = job;
      return next;
    });
  }, []);

  // A brand-new job means a new copy started (e.g. the events stream rebuilt
  // after a reconnect, or a job added from another tab): bring the center
  // back if the user had dismissed it. Pure effect, runs after the render.
  const knownJobIdsRef = useRef(new Set());
  useEffect(() => {
    const ids = new Set(activeJobs.map((j) => j.id));
    const hasNewJob = activeJobs.some((j) => !knownJobIdsRef.current.has(j.id));
    knownJobIdsRef.current = ids;
    if (hasNewJob) {
      setCenterDismissed(false);
    }
  }, [activeJobs]);

  const fetchJobs = useCallback(async () => {
    try {
      const res = await getTransferJobs();
      const jobs = Array.isArray(res?.jobs) ? res.jobs : [];
      setActiveJobs(jobs);
    } catch {
      // ignore transient network errors
    }
  }, []);

  useEffect(() => {
    const source = new EventSource(resolveAPIURL("/api/transfer/events"));
    source.addEventListener("jobs", (event) => {
      try {
        const payload = JSON.parse(event.data);
        if (Array.isArray(payload.jobs)) setActiveJobs(payload.jobs);
      } catch {}
    });
    source.addEventListener("job", (event) => {
      try {
        const payload = JSON.parse(event.data);
        applyJobEvent(payload.job);
      } catch {}
    });
    source.addEventListener("devices", (event) => {
      try {
        const payload = JSON.parse(event.data);
        if (Array.isArray(payload.devices)) setDevices(payload.devices);
      } catch {}
    });
    source.onerror = () => {
      // EventSource reconnects automatically; the server re-seeds snapshots.
    };
    return () => source.close();
  }, [applyJobEvent]);

  // File Selection Helpers
  const normalizeFile = (file, mediaTitle = "", poster = "") => {
    const id = file?.id || file?.file_id || file?.file_path || file?.path || Math.random().toString();
    const path = file?.filePath || file?.file_path || file?.path || file?.source_path || "";
    const name = file?.title_ar || file?.title_en || (file?.episode_number ? `الحلقة ${file.episode_number}` : file?.file_name || "ملف فيديو");
    const size = Number(file?.file_size || file?.size || 0);
    const resolution = file?.resolution || "1080p";
    return {
      id: String(id),
      filePath: path,
      title: name,
      episodeNumber: file?.episode_number || null,
      size,
      resolution,
      mediaTitle: mediaTitle || file?.media_title || "",
      poster: poster || file?.poster || ""
    };
  };

  const isFileSelected = useCallback(
    (file) => {
      if (!file) return false;
      const targetPath = file.filePath || file.file_path || file.path || file.source_path || "";
      const targetId = String(file.id || file.file_id || "");
      return selectedFiles.some(
        (f) => (targetPath && f.filePath === targetPath) || (targetId && f.id === targetId)
      );
    },
    [selectedFiles]
  );

  const toggleFileSelection = useCallback((file, mediaTitle = "", poster = "") => {
    if (!file) return;
    const norm = normalizeFile(file, mediaTitle, poster);
    setSelectedFiles((prev) => {
      const exists = prev.some((f) => (norm.filePath && f.filePath === norm.filePath) || f.id === norm.id);
      if (exists) {
        return prev.filter((f) => (norm.filePath ? f.filePath !== norm.filePath : f.id !== norm.id));
      }
      return [...prev, norm];
    });
  }, []);

  const selectMultipleFiles = useCallback((filesArray = [], mediaTitle = "", poster = "") => {
    if (!Array.isArray(filesArray) || filesArray.length === 0) return;
    const newItems = filesArray.map((f) => normalizeFile(f, mediaTitle, poster));
    setSelectedFiles((prev) => {
      const map = new Map();
      prev.forEach((item) => map.set(item.filePath || item.id, item));
      newItems.forEach((item) => map.set(item.filePath || item.id, item));
      return Array.from(map.values());
    });
  }, []);

  const removeFileFromSelection = useCallback((fileIdOrPath) => {
    setSelectedFiles((prev) =>
      prev.filter((f) => f.id !== String(fileIdOrPath) && f.filePath !== String(fileIdOrPath))
    );
  }, []);

  const clearSelection = useCallback(() => {
    setSelectedFiles([]);
  }, []);

  // Modal Controls
  const openTransferModal = useCallback((singleFile = null, mediaTitle = "", poster = "") => {
    if (singleFile) {
      const norm = normalizeFile(singleFile, mediaTitle, poster);
      setSelectedFiles([norm]);
    }
    setIsTransferModalOpen(true);
  }, []);

  const closeTransferModal = useCallback(() => {
    setIsTransferModalOpen(false);
  }, []);

  // Starting a Transfer
  const startTransfer = useCallback(
    async ({ deviceId, targetApp = "", targetFolder = "", subFolder = "", files = [] }) => {
      const filesToTransfer = files.length > 0 ? files : selectedFiles;
      if (!deviceId || filesToTransfer.length === 0) {
        throw new Error("يرجى تحديد الجهاز والملفات المراد نسخها");
      }

      const sourcePaths = filesToTransfer
        .map((f) => f.filePath || f.file_path || f.path || f.source_path)
        .filter((p) => Boolean(p && String(p).trim()));

      if (sourcePaths.length === 0) {
        throw new Error("لم يتم العثور على مسارات صالحة للملفات المحددة");
      }

      const payload = {
        device_id: deviceId,
        source_path: sourcePaths[0],
        source_paths: sourcePaths,
        target_app: targetApp,
        target_folder: targetFolder,
        sub_folder: subFolder
      };

      const res = await startDeviceTransfer(payload);
      if (!res.ok && !res.job) {
        throw new Error(res.error || "فشل بدء مهمة النسخ");
      }

      // Refresh jobs immediately
      await fetchJobs();

      // Clear selection and close modal, show mini transfer center
      clearSelection();
      closeTransferModal();
      setIsCenterExpanded(true);
      setCenterDismissed(false);

      return res.job;
    },
    [selectedFiles, fetchJobs, clearSelection, closeTransferModal]
  );

  const cancelJob = useCallback(
    async (jobId) => {
      if (!jobId) return;
      try {
        await cancelTransferJob(jobId);
        await fetchJobs();
      } catch (err) {
        console.error("Failed to cancel job", err);
      }
    },
    [fetchJobs]
  );

  const totalSelectedSize = selectedFiles.reduce((acc, f) => acc + (f.size || 0), 0);

  const hasRunningJobs = activeJobs.some(
    (j) =>
      j.status === "processing" ||
      j.status === "pending" ||
      j.status === "queued" ||
      j.status === "retrying" ||
      j.status === "waiting_device"
  );

  return (
    <TransferContext.Provider
      value={{
        selectedFiles,
        totalSelectedSize,
        isFileSelected,
        toggleFileSelection,
        selectMultipleFiles,
        removeFileFromSelection,
        clearSelection,

        isTransferModalOpen,
        openTransferModal,
        closeTransferModal,

        activeJobs,
        devices,
        hasRunningJobs,
        isCenterExpanded,
        setIsCenterExpanded,
        toggleCenterExpanded: () => setIsCenterExpanded((prev) => !prev),
        centerDismissed,
        setCenterDismissed,
        startTransfer,
        cancelJob,
        refreshJobs: fetchJobs
      }}
    >
      {children}
    </TransferContext.Provider>
  );
}

export function useTransfer() {
  const context = useContext(TransferContext);
  if (!context) {
    throw new Error("useTransfer must be used within a TransferProvider");
  }
  return context;
}
