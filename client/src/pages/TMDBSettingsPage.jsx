import React, { useState, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  getTMDBSettings,
  updateTMDBSettings,
  getTMDBStats,
  getTMDBModules,
  updateTMDBModules,
  testTMDBConnection,
  getTMDBRemoteConfiguration,
  searchLibrary,
  getTMDBPreview,
  getFranchises,
  refreshFranchise,
  refreshMissingFranchises,
} from "../lib/api";
import Icon from "../components/Icon";

export default function TMDBSettingsPage() {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [settings, setSettings] = useState(null);
  const [modules, setModules] = useState([]);
  const [stats, setStats] = useState(null);
  const [testResult, setTestResult] = useState(null);
  const [remoteConfig, setRemoteConfig] = useState(null);
  const [showConfigModal, setShowConfigModal] = useState(false);
  const [statusMessage, setStatusMessage] = useState(null);
  const [previewMediaId, setPreviewMediaId] = useState("");
  const [previewResult, setPreviewResult] = useState(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [searchSampleList, setSearchSampleList] = useState([]);
  const [franchises, setFranchises] = useState([]);
  const [franchiseRefreshing, setFranchiseRefreshing] = useState(null);

  // Filter category in modules view
  const [activeCategory, setActiveCategory] = useState("all");

  useEffect(() => {
    loadAllData();
  }, []);

  async function loadAllData() {
    setLoading(true);
    try {
      const [settingsRes, modulesRes, statsRes] = await Promise.allSettled([
        getTMDBSettings(),
        getTMDBModules(),
        getTMDBStats(),
      ]);

      if (settingsRes.status === "fulfilled") setSettings(settingsRes.value);
      if (modulesRes.status === "fulfilled") {
        setModules(modulesRes.value?.modules || []);
      }
      if (statsRes.status === "fulfilled") setStats(statsRes.value);
      getFranchises(24).then((res) => setFranchises(res?.franchises || [])).catch(() => {});

      // Load search sample items for preview
      searchLibrary("", { limit: 10 })
        .then((res) => {
          if (res?.items) setSearchSampleList(res.items);
        })
        .catch(() => {});
    } catch (err) {
      showMessage("فشل تحميل إعدادات TMDB: " + err.message, "error");
    } finally {
      setLoading(false);
    }
  }

  async function handleRefreshFranchise(id) {
    setFranchiseRefreshing(id);
    try {
      await refreshFranchise(id);
      showMessage("تم تحديث بيانات السلسلة باللغتين بنجاح", "success");
      const res = await getFranchises(24);
      setFranchises(res?.franchises || []);
    } catch (err) {
      showMessage("تعذر تحديث السلسلة: " + err.message, "error");
    } finally {
      setFranchiseRefreshing(null);
    }
  }

  async function handleRefreshMissingFranchises() {
    setFranchiseRefreshing("all");
    try {
      const res = await refreshMissingFranchises();
      showMessage(`تم تحديث ${res?.refreshed?.length || 0} سلسلة قديمة`, "success");
      const list = await getFranchises(24);
      setFranchises(list?.franchises || []);
    } catch (err) {
      showMessage("تعذر تحديث السلاسل القديمة: " + err.message, "error");
    } finally {
      setFranchiseRefreshing(null);
    }
  }

  function showMessage(msg, type = "success") {
    setStatusMessage({ text: msg, type });
    setTimeout(() => setStatusMessage(null), 4000);
  }

  async function handleTestConnection() {
    setTesting(true);
    setTestResult(null);
    try {
      const result = await testTMDBConnection();
      setTestResult(result);
      if (result.connected) {
        showMessage(`✅ الاتصال ناجح بـ TMDB API! (زمن الاستجابة: ${result.latencyMs}ms)`, "success");
        if (result.configuration) {
          setRemoteConfig(result.configuration);
        }
      } else {
        showMessage(`❌ تعذر الاتصال: ${result.error}`, "error");
      }
    } catch (err) {
      setTestResult({ connected: false, error: err.message });
      showMessage(`❌ خطأ: ${err.message}`, "error");
    } finally {
      setTesting(false);
    }
  }

  async function handleFetchRemoteConfig() {
    try {
      const config = await getTMDBRemoteConfiguration();
      setRemoteConfig(config);
      setShowConfigModal(true);
    } catch (err) {
      showMessage("فشل جلب إعدادات الأحجام من TMDB: " + err.message, "error");
    }
  }

  async function handleSelectFetchMode(mode) {
    if (!settings) return;
    setSaving(true);
    try {
      const res = await updateTMDBModules({ fetch_mode: mode });
      if (res?.settings) {
        setSettings(res.settings);
        setModules(res.modules || []);
        showMessage(`تم تفعيل وضع الجلب: ${getFetchModeTitle(mode)}`, "success");
      }
    } catch (err) {
      showMessage("فشل تغيير وضع الجلب: " + err.message, "error");
    } finally {
      setSaving(false);
    }
  }

  async function handleSelectImageMode(mode) {
    if (!settings) return;
    setSaving(true);
    try {
      const res = await updateTMDBModules({ image_mode: mode });
      if (res?.settings) {
        setSettings(res.settings);
        showMessage(`تم تغيير وضع الصور إلى: ${getImageModeTitle(mode)}`, "success");
      }
    } catch (err) {
      showMessage("فشل تغيير وضع الصور: " + err.message, "error");
    } finally {
      setSaving(false);
    }
  }

  async function handleToggleModule(moduleId) {
    if (!settings?.modules) return;
    const updatedModules = {
      ...settings.modules,
      [moduleId]: !settings.modules[moduleId],
    };

    setSaving(true);
    try {
      const res = await updateTMDBModules({ modules: updatedModules });
      if (res?.settings) {
        setSettings(res.settings);
        setModules(res.modules || []);
        showMessage("تم تحديث خيارات البيانات بنجاح", "success");
      }
    } catch (err) {
      showMessage("فشل تحديث الوحدة: " + err.message, "error");
    } finally {
      setSaving(false);
    }
  }

  async function handleSaveSettings(e) {
    if (e) e.preventDefault();
    if (!settings) return;
    setSaving(true);
    try {
      const res = await updateTMDBSettings(settings);
      if (res?.settings) {
        setSettings(res.settings);
      }
      showMessage("تم حفظ جميع إعدادات TMDB بنجاح ✅", "success");
      loadAllData();
    } catch (err) {
      showMessage("فشل حفظ الإعدادات: " + err.message, "error");
    } finally {
      setSaving(false);
    }
  }

  async function handleRunPreview(mediaId) {
    if (!mediaId) return;
    setPreviewLoading(true);
    setPreviewResult(null);
    try {
      const res = await getTMDBPreview(mediaId);
      setPreviewResult(res);
    } catch (err) {
      showMessage("فشل المعاينة: " + err.message, "error");
    } finally {
      setPreviewLoading(false);
    }
  }

  function getFetchModeTitle(mode) {
    switch (mode) {
      case "essential":
        return "الأساسي (Essential) — فائق السرعة وأقل استهلاك";
      case "standard":
        return "القياسي الموصى به (Standard) — متوازن وشامل";
      case "full":
        return "الكامل (Full Catalog) — تفاصيل وصور شاملة";
      default:
        return "مخصص (Custom)";
    }
  }

  function getImageModeTitle(mode) {
    switch (mode) {
      case "hybrid":
        return "هجين (Hybrid) — البوستر والخلفية محلياً، والباقي عبر الإنترنت (توفير 90%)";
      case "local":
        return "أوفلاين كامل (Local Storage) — حفظ كل الصور محلياً بالسيرفر";
      case "remote":
        return "إنترنت فقط (Remote CDN) — عرض كل الصور عبر الروابط دون تحميل للقرص";
      default:
        return mode;
    }
  }

  const filteredModules =
    activeCategory === "all"
      ? modules
      : modules.filter((m) => m.category === activeCategory);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="text-center space-y-4">
          <div className="w-12 h-12 border-4 border-indigo-500/30 border-t-indigo-500 rounded-full animate-spin mx-auto" />
          <p className="text-gray-400 font-medium">جاري تحميل إعدادات ولوحة تحكم TMDB...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-8 pb-16 text-right max-w-7xl mx-auto" dir="rtl">
      {/* Toast Notification */}
      <AnimatePresence>
        {statusMessage && (
          <motion.div
            initial={{ opacity: 0, y: -20, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -20, scale: 0.95 }}
            className={`fixed top-6 left-1/2 -translate-x-1/2 z-50 px-6 py-3 rounded-2xl shadow-2xl backdrop-blur-xl border flex items-center gap-3 ${
              statusMessage.type === "success"
                ? "bg-emerald-950/80 border-emerald-500/40 text-emerald-200"
                : "bg-rose-950/80 border-rose-500/40 text-rose-200"
            }`}
          >
            <Icon
              name={statusMessage.type === "success" ? "CheckCircle" : "AlertTriangle"}
              className="w-5 h-5"
            />
            <span className="font-medium text-sm">{statusMessage.text}</span>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Header Section */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 p-6 rounded-3xl bg-gradient-to-l from-indigo-950/40 via-purple-950/20 to-slate-900/40 border border-indigo-500/20 backdrop-blur-xl">
        <div className="space-y-1">
          <div className="flex items-center gap-3">
            <div className="p-3 rounded-2xl bg-indigo-600/20 border border-indigo-500/30 text-indigo-400">
              <Icon name="Database" className="w-7 h-7" />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-white flex items-center gap-2">
                لوحة تحكم وإدارة بيانات TMDB
                <span className="text-xs px-2.5 py-0.5 rounded-full bg-indigo-500/20 text-indigo-300 border border-indigo-500/30">
                  v3 Engine
                </span>
              </h1>
              <p className="text-sm text-gray-400">
                التحكم الشامل في جلب البيانات، أوضاع التخزين، ترشيد استهلاك الإنترنت، وإدارة وحدات الفهرسة الذكية
              </p>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-3 flex-wrap">
          <button
            onClick={handleTestConnection}
            disabled={testing}
            className="px-4 py-2.5 rounded-xl bg-slate-800/80 hover:bg-slate-700/80 text-gray-200 border border-slate-700 text-sm font-medium transition-all flex items-center gap-2 disabled:opacity-50"
          >
            <Icon name={testing ? "Loader" : "Activity"} className={`w-4 h-4 ${testing ? "animate-spin" : "text-emerald-400"}`} />
            <span>{testing ? "جاري الفحص..." : "فحص الاتصال والـ Ping"}</span>
          </button>

          <button
            onClick={handleFetchRemoteConfig}
            className="px-4 py-2.5 rounded-xl bg-indigo-950/60 hover:bg-indigo-900/60 text-indigo-300 border border-indigo-500/30 text-sm font-medium transition-all flex items-center gap-2"
          >
            <Icon name="Image" className="w-4 h-4" />
            <span>أحجام الصور المتاحة (TMDB Config)</span>
          </button>

          <button
            onClick={handleSaveSettings}
            disabled={saving}
            className="px-5 py-2.5 rounded-xl bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white font-medium text-sm shadow-lg shadow-indigo-500/20 transition-all flex items-center gap-2 disabled:opacity-50"
          >
            <Icon name={saving ? "Loader" : "Save"} className={`w-4 h-4 ${saving ? "animate-spin" : ""}`} />
            <span>{saving ? "جاري الحفظ..." : "حفظ التغييرات"}</span>
          </button>
        </div>
      </div>

      <section className="space-y-4 rounded-3xl border border-amber-500/20 bg-amber-950/10 p-5 backdrop-blur-xl">
        <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
          <div>
            <h2 className="text-lg font-bold text-white">مزامنة سلاسل TMDB</h2>
            <p className="mt-1 text-xs text-gray-400">يتم جلب التفاصيل بالإنجليزية والعربية مرة واحدة ثم تخزينها 30 يومًا.</p>
          </div>
          <button onClick={handleRefreshMissingFranchises} disabled={franchiseRefreshing !== null} className="rounded-xl border border-amber-400/30 bg-amber-500/10 px-4 py-2 text-xs font-bold text-amber-200 disabled:opacity-50">
            {franchiseRefreshing === "all" ? "جارٍ تحديث السلاسل..." : "تحديث السلاسل القديمة"}
          </button>
        </div>
        {franchises.length > 0 && <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {franchises.map((franchise) => <div key={franchise.id} className="flex items-center justify-between gap-3 rounded-2xl border border-white/10 bg-black/20 p-3">
            <div className="min-w-0"><p className="truncate text-sm font-bold text-white">{franchise.title_ar || franchise.title_en}</p><p className="text-[11px] text-gray-500">{franchise.parts_count || 0} أجزاء · {franchise.rating ? Number(franchise.rating).toFixed(1) : "بدون تقييم"}</p></div>
            <button onClick={() => handleRefreshFranchise(franchise.id)} disabled={franchiseRefreshing !== null} className="shrink-0 rounded-lg border border-white/15 px-2.5 py-1.5 text-[11px] font-bold text-gray-200 disabled:opacity-50">{franchiseRefreshing === franchise.id ? "جارٍ..." : "تحديث"}</button>
          </div>)}
        </div>}
      </section>

      {/* 1. Live Stats & Bandwidth Monitoring Bar */}
      {stats && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="p-5 rounded-2xl bg-slate-900/60 border border-slate-800 backdrop-blur-md space-y-2">
            <div className="flex items-center justify-between text-gray-400 text-xs">
              <span>استهلاك الباندويث اليومي</span>
              <Icon name="Wifi" className="w-4 h-4 text-cyan-400" />
            </div>
            <div className="flex items-baseline justify-between">
              <span className="text-2xl font-bold text-white">
                {stats.mb_today ? stats.mb_today.toFixed(2) : "0.00"} <span className="text-sm font-normal text-gray-400">MB</span>
              </span>
              <span className="text-xs text-gray-400">
                من أصل {stats.daily_quota_mb ? `${stats.daily_quota_mb} MB` : "غير محدود"}
              </span>
            </div>
            {stats.daily_quota_mb > 0 && (
              <div className="w-full bg-slate-800 h-2 rounded-full overflow-hidden">
                <div
                  className={`h-full transition-all duration-500 ${
                    stats.daily_quota_used_percent > 85
                      ? "bg-rose-500"
                      : stats.daily_quota_used_percent > 60
                      ? "bg-amber-500"
                      : "bg-emerald-500"
                  }`}
                  style={{ width: `${Math.min(stats.daily_quota_used_percent, 100)}%` }}
                />
              </div>
            )}
            <p className="text-[11px] text-gray-500 flex justify-between">
              <span>نسبة الاستهلاك: {stats.daily_quota_used_percent?.toFixed(1) || 0}%</span>
              <span>{stats.images_today || 0} صورة اليوم</span>
            </p>
          </div>

          <div className="p-5 rounded-2xl bg-slate-900/60 border border-slate-800 backdrop-blur-md space-y-2">
            <div className="flex items-center justify-between text-gray-400 text-xs">
              <span>طلبات الـ API (Requests)</span>
              <Icon name="Server" className="w-4 h-4 text-purple-400" />
            </div>
            <div className="flex items-baseline justify-between">
              <span className="text-2xl font-bold text-white">{stats.requests_today || 0}</span>
              <span className="text-xs text-purple-300 bg-purple-500/10 px-2 py-0.5 rounded-full border border-purple-500/20">
                {stats.requests_this_month || 0} هذا الشهر
              </span>
            </div>
            <p className="text-[11px] text-gray-500">إجمالي الطلبات التراكمية: {stats.total_requests || 0} طلب</p>
          </div>

          <div className="p-5 rounded-2xl bg-slate-900/60 border border-slate-800 backdrop-blur-md space-y-2">
            <div className="flex items-center justify-between text-gray-400 text-xs">
              <span>تغطية المكتبة والبيانات</span>
              <Icon name="Film" className="w-4 h-4 text-emerald-400" />
            </div>
            <div className="flex items-baseline justify-between">
              <span className="text-2xl font-bold text-emerald-400">{stats.enriched_media_count || 0}</span>
              <span className="text-xs text-amber-300 bg-amber-500/10 px-2 py-0.5 rounded-full border border-amber-500/20">
                {stats.pending_media_count || 0} بحاجة لإثراء
              </span>
            </div>
            <p className="text-[11px] text-gray-500">إجمالي الصور المخزنة: {stats.total_images_downloaded || 0}</p>
          </div>

          <div className="p-5 rounded-2xl bg-slate-900/60 border border-slate-800 backdrop-blur-md space-y-2">
            <div className="flex items-center justify-between text-gray-400 text-xs">
              <span>حالة مزود TMDB</span>
              <div className="flex items-center gap-1.5">
                <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                <span className="text-emerald-400 text-xs">جاهز</span>
              </div>
            </div>
            <div className="text-sm font-semibold text-white truncate">
              خطة TMDB المجانية
            </div>
            <p className="text-[11px] text-gray-400">
              حد السرعة: ~35-40 طلب/ثانية مع تفعيل التراجع الآلي (Backoff)
            </p>
          </div>
        </div>
      )}

      {/* 2. Mode Profiles & Image Strategy */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Fetch Mode Selector */}
        <div className="p-6 rounded-3xl bg-slate-900/50 border border-slate-800/80 backdrop-blur-xl space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-bold text-white flex items-center gap-2">
                <Icon name="Layers" className="w-5 h-5 text-indigo-400" />
                أوضاع جلب البيانات (Fetch Mode Profiles)
              </h2>
              <p className="text-xs text-gray-400">اختر البروفايل المناسب لسرعة وسعة الإنترنت لديك</p>
            </div>
            <span className="text-xs font-mono px-3 py-1 rounded-full bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
              الوضع الحالي: {settings?.fetch_mode}
            </span>
          </div>

          <div className="space-y-3">
            {/* Essential */}
            <div
              onClick={() => handleSelectFetchMode("essential")}
              className={`p-4 rounded-2xl border cursor-pointer transition-all ${
                settings?.fetch_mode === "essential"
                  ? "bg-indigo-950/40 border-indigo-500/60 shadow-lg shadow-indigo-500/10 ring-1 ring-indigo-500/40"
                  : "bg-slate-800/40 border-slate-800 hover:border-slate-700"
              }`}
            >
              <div className="flex items-center justify-between mb-1">
                <span className="font-bold text-white flex items-center gap-2">
                  <span className="w-2.5 h-2.5 rounded-full bg-emerald-400" />
                  الوضع الأساسي (Essential)
                </span>
                <span className="text-xs font-medium text-emerald-400 bg-emerald-500/10 px-2.5 py-0.5 rounded-full">
                  ~150 - 300 KB / عنصر
                </span>
              </div>
              <p className="text-xs text-gray-400 leading-relaxed">
                يجلب فقط: العناوين بالعربية والإنجليزية، القصة والملخص، التصنيفات، التقييم، الكلمات المفتاحية، البوستر والخلفية الأساسية. <strong>مثالي للشبكات المحدودة وسرعة الفهرسة الفائقة.</strong>
              </p>
            </div>

            {/* Standard (Recommended) */}
            <div
              onClick={() => handleSelectFetchMode("standard")}
              className={`p-4 rounded-2xl border cursor-pointer transition-all ${
                settings?.fetch_mode === "standard"
                  ? "bg-indigo-950/40 border-indigo-500/60 shadow-lg shadow-indigo-500/10 ring-1 ring-indigo-500/40"
                  : "bg-slate-800/40 border-slate-800 hover:border-slate-700"
              }`}
            >
              <div className="flex items-center justify-between mb-1">
                <span className="font-bold text-white flex items-center gap-2">
                  <span className="w-2.5 h-2.5 rounded-full bg-indigo-400" />
                  الوضع القياسي الموصى به (Standard)
                </span>
                <span className="text-xs font-medium text-indigo-300 bg-indigo-500/10 px-2.5 py-0.5 rounded-full">
                  ~350 - 450 KB / عنصر
                </span>
              </div>
              <p className="text-xs text-gray-400 leading-relaxed">
                الوضع الأساسي + أسماء الممثلين (نصياً بدون صور)، روابط التريلرات من YouTube، المعرفات الخارجية IMDb، الترجمات، وتصنيف المحتوى العمري. <strong>أفضل توازن للمقاهي والصالات.</strong>
              </p>
            </div>

            {/* Full */}
            <div
              onClick={() => handleSelectFetchMode("full")}
              className={`p-4 rounded-2xl border cursor-pointer transition-all ${
                settings?.fetch_mode === "full"
                  ? "bg-indigo-950/40 border-indigo-500/60 shadow-lg shadow-indigo-500/10 ring-1 ring-indigo-500/40"
                  : "bg-slate-800/40 border-slate-800 hover:border-slate-700"
              }`}
            >
              <div className="flex items-center justify-between mb-1">
                <span className="font-bold text-white flex items-center gap-2">
                  <span className="w-2.5 h-2.5 rounded-full bg-purple-400" />
                  الوضع الكامل (Full Catalog)
                </span>
                <span className="text-xs font-medium text-purple-300 bg-purple-500/10 px-2.5 py-0.5 rounded-full">
                  ~3 - 5 MB / عنصر
                </span>
              </div>
              <p className="text-xs text-gray-400 leading-relaxed">
                يجلب كل شيء بما فيها صور الممثلين الـ 20، معارض الصور المتعددة، بوسترات الأعمال المقترحة، والمراجعات. <strong>يستهلك نت ومساحة أكبر.</strong>
              </p>
            </div>
          </div>
        </div>

        {/* Image Handling Strategy */}
        <div className="p-6 rounded-3xl bg-slate-900/50 border border-slate-800/80 backdrop-blur-xl space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-bold text-white flex items-center gap-2">
                <Icon name="Image" className="w-5 h-5 text-purple-400" />
                استراتيجية معالجة وتخزين الصور (Image Mode)
              </h2>
              <p className="text-xs text-gray-400">التحكم في مكان حفظ وعرض الملصقات والخلفيات</p>
            </div>
          </div>

          <div className="space-y-3">
            {/* Hybrid (Default) */}
            <div
              onClick={() => handleSelectImageMode("hybrid")}
              className={`p-4 rounded-2xl border cursor-pointer transition-all ${
                settings?.image_mode === "hybrid"
                  ? "bg-purple-950/40 border-purple-500/60 shadow-lg shadow-purple-500/10 ring-1 ring-purple-500/40"
                  : "bg-slate-800/40 border-slate-800 hover:border-slate-700"
              }`}
            >
              <div className="flex items-center justify-between mb-1">
                <span className="font-bold text-white flex items-center gap-2">
                  <Icon name="Shuffle" className="w-4 h-4 text-indigo-400" />
                  الوضع الهجين (Hybrid) — الافتراضي الموصى به
                </span>
                <span className="text-xs font-medium text-emerald-300 bg-emerald-500/10 px-2 py-0.5 rounded-full">
                  توفير 90%
                </span>
              </div>
              <p className="text-xs text-gray-400 leading-relaxed">
                تحميل البوستر والخلفية الأساسية فقط محلياً بالسيرفر للعمل أوفلاين في التصفح اليومي، بينما تعرض صور الممثلين والمعارض مباشرة عبر روابط TMDB CDN دون كتابتها على القرص.
              </p>
            </div>

            {/* Local */}
            <div
              onClick={() => handleSelectImageMode("local")}
              className={`p-4 rounded-2xl border cursor-pointer transition-all ${
                settings?.image_mode === "local"
                  ? "bg-purple-950/40 border-purple-500/60 shadow-lg shadow-purple-500/10 ring-1 ring-purple-500/40"
                  : "bg-slate-800/40 border-slate-800 hover:border-slate-700"
              }`}
            >
              <div className="flex items-center justify-between mb-1">
                <span className="font-bold text-white flex items-center gap-2">
                  <Icon name="HardDrive" className="w-4 h-4 text-emerald-400" />
                  أوفلاين كامل (Local Storage)
                </span>
                <span className="text-xs font-medium text-purple-300 bg-purple-500/10 px-2 py-0.5 rounded-full">
                  100% Offline
                </span>
              </div>
              <p className="text-xs text-gray-400 leading-relaxed">
                تنزيل وحفظ جميع الصور المفعلة محلياً على القرص الصلب. مناسب للشبكات المعزولة تماماً عن الإنترنت بعد إتمام الفهرسة.
              </p>
            </div>

            {/* Remote */}
            <div
              onClick={() => handleSelectImageMode("remote")}
              className={`p-4 rounded-2xl border cursor-pointer transition-all ${
                settings?.image_mode === "remote"
                  ? "bg-purple-950/40 border-purple-500/60 shadow-lg shadow-purple-500/10 ring-1 ring-purple-500/40"
                  : "bg-slate-800/40 border-slate-800 hover:border-slate-700"
              }`}
            >
              <div className="flex items-center justify-between mb-1">
                <span className="font-bold text-white flex items-center gap-2">
                  <Icon name="Globe" className="w-4 h-4 text-cyan-400" />
                  إنترنت فقط (Remote CDN)
                </span>
                <span className="text-xs font-medium text-cyan-300 bg-cyan-500/10 px-2 py-0.5 rounded-full">
                  Zero Disk
                </span>
              </div>
              <p className="text-xs text-gray-400 leading-relaxed">
                لا يتم تحميل أي صور على السيرفر، بل يتم عرض الصور من سيرفرات TMDB العالمية مباشرة عند تصفح العملاء. (يتطلب اتصال دائم بالإنترنت).
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* 3. Global Network & Quota Preferences */}
      <div className="p-6 rounded-3xl bg-slate-900/50 border border-slate-800/80 backdrop-blur-xl space-y-6">
        <h2 className="text-lg font-bold text-white flex items-center gap-2">
          <Icon name="Sliders" className="w-5 h-5 text-indigo-400" />
          إعدادات الشبكة وميزانية الإنترنت واللغات
        </h2>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {/* Daily Quota */}
          <div className="space-y-2">
            <label className="text-xs font-semibold text-gray-300 flex items-center gap-1.5">
              <span>حد التحميل اليومي (Daily Quota)</span>
            </label>
            <div className="relative">
              <input
                type="number"
                value={settings?.daily_bandwidth_mb || 0}
                onChange={(e) =>
                  setSettings({ ...settings, daily_bandwidth_mb: parseInt(e.target.value) || 0 })
                }
                className="w-full bg-slate-800/80 border border-slate-700 rounded-xl px-4 py-2.5 text-white text-sm focus:outline-none focus:border-indigo-500"
                placeholder="500"
              />
              <span className="absolute left-3 top-2.5 text-xs text-gray-400">ميجابايت / يوم</span>
            </div>
            <p className="text-[11px] text-gray-500">أدخل 0 للتحميل غير المحدود. (الافتراضي: 500 MB)</p>
          </div>

          {/* Preferred Language */}
          <div className="space-y-2">
            <label className="text-xs font-semibold text-gray-300">اللغة المفضلة للبيانات</label>
            <select
              value={settings?.preferred_language || "ar-SA"}
              onChange={(e) => setSettings({ ...settings, preferred_language: e.target.value })}
              className="w-full bg-slate-800/80 border border-slate-700 rounded-xl px-4 py-2.5 text-white text-sm focus:outline-none focus:border-indigo-500"
            >
              <option value="ar-SA">العربية (السعودية) — ar-SA</option>
              <option value="ar-EG">العربية (مصر) — ar-EG</option>
              <option value="en-US">الإنجليزية (الولايات المتحدة) — en-US</option>
            </select>
            <p className="text-[11px] text-gray-500">اللغة الأساسية لطلب العناوين والقصص</p>
          </div>

          {/* Poster Size */}
          <div className="space-y-2">
            <label className="text-xs font-semibold text-gray-300">دقة البوستر الافتراضية</label>
            <select
              value={settings?.poster_size || "w500"}
              onChange={(e) => setSettings({ ...settings, poster_size: e.target.value })}
              className="w-full bg-slate-800/80 border border-slate-700 rounded-xl px-4 py-2.5 text-white text-sm focus:outline-none focus:border-indigo-500"
            >
              <option value="w342">w342 (خفيف وسريع)</option>
              <option value="w500">w500 (متوازن عالي الجودة - موصى به)</option>
              <option value="w780">w780 (دقة فائقة)</option>
              <option value="original">original (الحجم الأصلي)</option>
            </select>
            <p className="text-[11px] text-gray-500">دقة صورة الغلاف عند التحميل أو العرض</p>
          </div>
        </div>
      </div>

      {/* 4. Granular Module Switches / Toggles */}
      <div className="p-6 rounded-3xl bg-slate-900/50 border border-slate-800/80 backdrop-blur-xl space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h2 className="text-lg font-bold text-white flex items-center gap-2">
              <Icon name="ToggleLeft" className="w-5 h-5 text-indigo-400" />
              وحدات وعناصر البيانات التفصيلية (Data Modules)
            </h2>
            <p className="text-xs text-gray-400">تفعيل أو إيقاف كل حقل بيانات بشكل مستقل حسب رغبة المقهى</p>
          </div>

          {/* Filter Categories */}
          <div className="flex items-center gap-1.5 p-1 bg-slate-800/80 rounded-xl border border-slate-700/60 text-xs">
            <button
              onClick={() => setActiveCategory("all")}
              className={`px-3 py-1.5 rounded-lg transition-all ${
                activeCategory === "all" ? "bg-indigo-600 text-white font-medium shadow" : "text-gray-400 hover:text-white"
              }`}
            >
              الكل ({modules.length})
            </button>
            <button
              onClick={() => setActiveCategory("text")}
              className={`px-3 py-1.5 rounded-lg transition-all ${
                activeCategory === "text" ? "bg-indigo-600 text-white font-medium shadow" : "text-gray-400 hover:text-white"
              }`}
            >
              نصوص ومعلومات
            </button>
            <button
              onClick={() => setActiveCategory("media")}
              className={`px-3 py-1.5 rounded-lg transition-all ${
                activeCategory === "media" ? "bg-indigo-600 text-white font-medium shadow" : "text-gray-400 hover:text-white"
              }`}
            >
              صور وفيديو
            </button>
            <button
              onClick={() => setActiveCategory("tv")}
              className={`px-3 py-1.5 rounded-lg transition-all ${
                activeCategory === "tv" ? "bg-indigo-600 text-white font-medium shadow" : "text-gray-400 hover:text-white"
              }`}
            >
              المواسم والحلقات
            </button>
            <button
              onClick={() => setActiveCategory("extra")}
              className={`px-3 py-1.5 rounded-lg transition-all ${
                activeCategory === "extra" ? "bg-indigo-600 text-white font-medium shadow" : "text-gray-400 hover:text-white"
              }`}
            >
              إضافات وتوصيات
            </button>
          </div>
        </div>

        {/* Modules Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filteredModules.map((item) => {
            const isEnabled = settings?.modules?.[item.id] ?? item.enabled;
            return (
              <div
                key={item.id}
                onClick={() => handleToggleModule(item.id)}
                className={`p-4 rounded-2xl border cursor-pointer transition-all flex items-start justify-between gap-3 select-none ${
                  isEnabled
                    ? "bg-indigo-950/20 border-indigo-500/40 hover:border-indigo-500/60"
                    : "bg-slate-900/30 border-slate-800/80 opacity-60 hover:opacity-100"
                }`}
              >
                <div className="space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="font-semibold text-sm text-white">{item.name_ar}</span>
                    {item.is_essential && (
                      <span className="text-[10px] px-2 py-0.2 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                        أساسي
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-gray-400 line-clamp-2">{item.description_ar}</p>
                  <span className="inline-block text-[11px] font-mono text-cyan-400 bg-cyan-950/30 px-2 py-0.5 rounded-md border border-cyan-500/20">
                    {item.estimated_bandwidth}
                  </span>
                </div>

                {/* Custom Toggle Switch */}
                <div
                  className={`w-11 h-6 flex items-center rounded-full p-1 transition-colors duration-300 shrink-0 ${
                    isEnabled ? "bg-indigo-600" : "bg-slate-700"
                  }`}
                >
                  <div
                    className={`bg-white w-4 h-4 rounded-full shadow-md transform transition-transform duration-300 ${
                      isEnabled ? "-translate-x-5" : "translate-x-0"
                    }`}
                  />
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* 5. Live Media Enrichment Preview Tester */}
      <div className="p-6 rounded-3xl bg-slate-900/50 border border-slate-800/80 backdrop-blur-xl space-y-4">
        <div>
          <h2 className="text-lg font-bold text-white flex items-center gap-2">
            <Icon name="Eye" className="w-5 h-5 text-indigo-400" />
            معاينة واختبار استهلاك العنصر قبل الإثراء (Enrichment Simulator)
          </h2>
          <p className="text-xs text-gray-400">
            اختر عنصراً من مكتبتك لمعاينة حجم الباندويث وعدد الطلبات التي ستُستهلك بالضبط وفق الإعدادات المحددة
          </p>
        </div>

        <div className="flex items-center gap-3 flex-wrap">
          <select
            value={previewMediaId}
            onChange={(e) => {
              setPreviewMediaId(e.target.value);
              handleRunPreview(e.target.value);
            }}
            className="bg-slate-800 border border-slate-700 rounded-xl px-4 py-2.5 text-white text-sm focus:outline-none focus:border-indigo-500 min-w-[280px]"
          >
            <option value="">-- اختر عنصراً من المكتبة للمعاينة --</option>
            {searchSampleList.map((item) => (
              <option key={item.id} value={item.id}>
                {item.title_ar || item.title_en} ({item.release_year || "غير محدد"}) - [{item.type}]
              </option>
            ))}
          </select>

          {previewLoading && (
            <div className="flex items-center gap-2 text-xs text-indigo-400">
              <Icon name="Loader" className="w-4 h-4 animate-spin" />
              <span>جاري محاكاة الطلب...</span>
            </div>
          )}
        </div>

        {previewResult && (
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            className="p-5 rounded-2xl bg-indigo-950/20 border border-indigo-500/30 grid grid-cols-1 sm:grid-cols-3 gap-4"
          >
            <div>
              <span className="text-xs text-gray-400">العنصر المحدد:</span>
              <p className="font-bold text-white text-sm">{previewResult.title_ar || previewResult.title_en}</p>
              <span className="text-xs text-indigo-300">النوع: {previewResult.type}</span>
            </div>
            <div>
              <span className="text-xs text-gray-400">عدد طلبات TMDB المتوقعة:</span>
              <p className="font-bold text-emerald-400 text-lg">{previewResult.estimatedRequests} طلبات فقط</p>
              <span className="text-xs text-gray-500">تشمل البحث والتفاصيل والترجمة</span>
            </div>
            <div>
              <span className="text-xs text-gray-400">حجم الاستهلاك المتوقع للنت:</span>
              <p className="font-bold text-cyan-400 text-lg">{previewResult.estimatedSizeFormatted}</p>
              <span className="text-xs text-gray-500">حسب استراتيجية: {previewResult.image_mode}</span>
            </div>
          </motion.div>
        )}
      </div>

      {/* Configuration Sizes Modal */}
      <AnimatePresence>
        {showConfigModal && remoteConfig && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-md">
            <motion.div
              initial={{ scale: 0.9, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.9, opacity: 0 }}
              className="bg-slate-900 border border-slate-700 rounded-3xl p-6 max-w-2xl w-full space-y-4 shadow-2xl text-right max-h-[85vh] overflow-y-auto"
            >
              <div className="flex items-center justify-between border-b border-slate-800 pb-3">
                <h3 className="text-lg font-bold text-white flex items-center gap-2">
                  <Icon name="CheckCircle" className="w-5 h-5 text-emerald-400" />
                  أحجام الصور الرسمية المعتمدة من TMDB
                </h3>
                <button
                  onClick={() => setShowConfigModal(false)}
                  className="p-2 rounded-xl hover:bg-slate-800 text-gray-400 hover:text-white"
                >
                  <Icon name="X" className="w-5 h-5" />
                </button>
              </div>

              <div className="space-y-4 text-xs">
                <div>
                  <span className="font-semibold text-gray-300 block mb-1">الرابط الآمن المعتمد (Base URL):</span>
                  <code className="text-cyan-400 bg-slate-800 px-3 py-1.5 rounded-lg block font-mono">
                    {remoteConfig.images?.secure_base_url || "https://image.tmdb.org/t/p/"}
                  </code>
                </div>

                <div>
                  <span className="font-semibold text-gray-300 block mb-1">أحجام البوسترات (Poster Sizes):</span>
                  <div className="flex flex-wrap gap-2">
                    {remoteConfig.images?.poster_sizes?.map((size) => (
                      <span key={size} className="px-2.5 py-1 rounded-lg bg-indigo-950/60 border border-indigo-500/30 text-indigo-300 font-mono">
                        {size}
                      </span>
                    ))}
                  </div>
                </div>

                <div>
                  <span className="font-semibold text-gray-300 block mb-1">أحجام الخلفيات (Backdrop Sizes):</span>
                  <div className="flex flex-wrap gap-2">
                    {remoteConfig.images?.backdrop_sizes?.map((size) => (
                      <span key={size} className="px-2.5 py-1 rounded-lg bg-purple-950/60 border border-purple-500/30 text-purple-300 font-mono">
                        {size}
                      </span>
                    ))}
                  </div>
                </div>

                <div>
                  <span className="font-semibold text-gray-300 block mb-1">أحجام صور الممثلين (Profile Sizes):</span>
                  <div className="flex flex-wrap gap-2">
                    {remoteConfig.images?.profile_sizes?.map((size) => (
                      <span key={size} className="px-2.5 py-1 rounded-lg bg-emerald-950/60 border border-emerald-500/30 text-emerald-300 font-mono">
                        {size}
                      </span>
                    ))}
                  </div>
                </div>
              </div>

              <div className="pt-4 border-t border-slate-800 flex justify-end">
                <button
                  onClick={() => setShowConfigModal(false)}
                  className="px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium"
                >
                  إغلاق
                </button>
              </div>
            </motion.div>
          </div>
        )}
      </AnimatePresence>
    </div>
  );
}
