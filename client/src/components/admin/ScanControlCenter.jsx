import { useCallback, useEffect, useRef, useState } from "react";
import GlassCard from "../../components/GlassCard.jsx";
import Icon from "../../components/Icon.jsx";
import {
  getScanStatus,
  pauseScan,
  resumeScan,
  cancelScan
} from "../../lib/api.js";

/**
 * ScanControlCenter — لوحة التحكم الحيّة في الفحص
 *
 * تعرض تقدمًا حقيًا (لا نسبة تخمينية) وحالة كل worker على حدة، وتتيح
 * Pause / Resume / Cancel.
 *
 * ملاحظات تصميمية مهمة:
 *
 *  - Pause تعاوني: الرد غالبًا "pausing" لأن كل worker يُكمل الملف الذي
 *    يعمل عليه. الواجهة تُظهر هذه المرحلة الانتقالية ولا تدّعي توقفًا فوريًا.
 *  - Cancel عملية مختلفة عن Pause: يُنهي الفحص ويفقد counters. لذلك يُطلب
 *    تأكيد صريح قبل تنفيذه.
 *  - الاستطلاع (polling) يتوقف تلقائيًا عند انتهاء الفحص أو إلغاء المكوّن،
 *    حتى لا يستمر طلب شبكي بلا داعٍ.
 */

const POLL_INTERVAL_MS = 1500;

/** التسميات العربية لأسباب التوقف وحالات الفحص. */
const SCAN_STATE_LABELS = {
  running: "يعمل",
  pausing: "جارٍ الإيقاف المؤقت…",
  paused: "متوقف مؤقتًا",
  resuming: "جارٍ الاستئناف…",
  cancelled: "أُلغي",
  idle: "لا يوجد فحص"
};

const STATUS_LABELS = {
  running: "يعمل",
  completed: "اكتمل",
  partial: "اكتمل جزئيًا",
  failed: "فشل",
  cancelled: "أُلغي"
};

const ROOT_STATUS_LABELS = {
  pending: "بالانتظار",
  available: "متاح",
  scanning: "يُفحص",
  completed: "اكتمل",
  unavailable: "غير متاح",
  error: "خطأ",
  offline: "غير متصل"
};

const ROLE_LABELS = {
  discovery: "استكشاف",
  metadata: "تحليل",
  persistence: "كتابة",
  idle: "خامل"
};

const ROLE_TONES = {
  discovery: "bg-cyan-500/20 text-cyan-300 border-cyan-500/30",
  metadata: "bg-fuchsia-500/20 text-fuchsia-300 border-fuchsia-500/30",
  persistence: "bg-emerald-500/20 text-emerald-300 border-emerald-500/30",
  idle: "bg-white/5 text-white/40 border-white/10"
};

function formatNumber(value) {
  if (value === null || value === undefined) return "0";
  return Number(value).toLocaleString("en-US");
}

function formatBytes(bytes) {
  const value = Number(bytes) || 0;
  if (value <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB", "PB"];
  const exponent = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  const scaled = value / Math.pow(1024, exponent);
  return `${scaled.toFixed(exponent === 0 ? 0 : 1)} ${units[exponent]}`;
}

/** يعرض مدة بالثواني بشكل مقروء. */
function formatDuration(seconds) {
  const total = Math.max(0, Math.floor(Number(seconds) || 0));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = total % 60;
  if (hours > 0) return `${hours}س ${minutes}د ${secs}ث`;
  if (minutes > 0) return `${minutes}د ${secs}ث`;
  return `${secs}ث`;
}

/** يقتصر المسار الطويل على آخر جزأين حتى لا يكسر التخطيط. */
function shortenPath(path) {
  if (!path) return "";
  const parts = String(path).split(/[\\/]/).filter(Boolean);
  if (parts.length <= 2) return String(path);
  return `…/${parts.slice(-2).join("/")}`;
}

export default function ScanControlCenter({ onScanFinished }) {
  const [state, setState] = useState({
    running: false,
    progress: null,
    control: null,
    workers: [],
    activeWorkers: 0,
    idleWorkers: 0
  });
  const [actionState, setActionState] = useState("idle"); // idle | pausing | resuming | cancelling
  const [error, setError] = useState("");
  const [confirmCancel, setConfirmCancel] = useState(false);

  // نتتبع حالة التشغيل السابقة لمعرفة متى انتهى الفحص بالضبط، فنُحدّث
  // لوحة الفهرسة مرة واحدة عند الانتهاء بدل كل استطلاع.
  const wasRunning = useRef(false);
  const onScanFinishedRef = useRef(onScanFinished);
  onScanFinishedRef.current = onScanFinished;

  const load = useCallback(async () => {
    try {
      const data = await getScanStatus();
      const running = Boolean(data?.running);
      setState({
        running,
        progress: data?.progress ?? null,
        control: data?.control ?? null,
        workers: data?.workers ?? [],
        activeWorkers: data?.activeWorkers ?? 0,
        idleWorkers: data?.idleWorkers ?? 0
      });
      setError("");

      // الإشعار بالانتهاء يحدث مرة واحدة عند الانتقال من "يعمل" إلى "متوقف".
      if (wasRunning.current && !running) {
        onScanFinishedRef.current?.();
      }
      wasRunning.current = running;
    } catch (err) {
      setError(err?.message || "تعذّر قراءة حالة الفحص.");
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    let timer = null;

    const tick = async () => {
      await load();
      if (!cancelled) {
        timer = setTimeout(tick, POLL_INTERVAL_MS);
      }
    };
    tick();

    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
    };
  }, [load]);

  const progress = state.progress;
  const scanState = state.control?.state || (state.running ? "running" : "idle");
  const isPaused = scanState === "paused" || scanState === "pausing";

  async function handlePause() {
    setActionState("pausing");
    try {
      const data = await pauseScan();
      setState((previous) => ({
        ...previous,
        control: data?.control ?? previous.control,
        workers: data?.workers ?? previous.workers
      }));
    } catch (err) {
      setError(err?.message || "تعذّر إيقاف الفحص مؤقتًا.");
    } finally {
      setActionState("idle");
    }
  }

  async function handleResume() {
    setActionState("resuming");
    try {
      await resumeScan();
    } catch (err) {
      setError(err?.message || "تعذّر استئناف الفحص.");
    } finally {
      setActionState("idle");
    }
  }

  async function handleCancel() {
    setActionState("cancelling");
    try {
      await cancelScan();
      setConfirmCancel(false);
    } catch (err) {
      setError(err?.message || "تعذّر إلغاء الفحص.");
    } finally {
      setActionState("idle");
    }
  }

  // ── الحالة الخاملة: لا فحص جارٍ ──────────────────────────
  if (!state.running) {
    return (
      <GlassCard className="p-6 border-white/10">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-bold text-white flex items-center gap-2">
              <Icon name="spark" className="h-5 w-5 text-white/40" />
              <span>مركز التحكم في الفحص</span>
            </h2>
            <p className="mt-1 text-xs text-white/50">
              لا يوجد فحص جارٍ حاليًا. ابدأ فهرسة من الأعلى وستظهر هنا حالة كل عامل ومراحل التقدم لحظيًا.
            </p>
          </div>
          <span className="rounded-full border-white/10 bg-white/5 px-3 py-1.5 text-xs font-bold text-white/50">
            خامل
          </span>
        </div>

        {progress && (
          <p className="mt-4 text-[11px] text-white/30">
            آخر فحص: {STATUS_LABELS[progress.status] || progress.status} · مدة {progress.duration?.human || "—"}
          </p>
        )}

        {error && (
          <div className="mt-4 rounded-xl border-amber-500/30 bg-amber-950/20 p-3 text-xs text-amber-300">
            ⚠️ {error}
          </div>
        )}
      </GlassCard>
    );
  }

  const totalProcessed =
    (progress?.newFiles || 0) +
    (progress?.modifiedFiles || 0) +
    (progress?.renamedFiles || 0) +
    (progress?.unchangedFiles || 0);

  // نسبة التقدم تقديرية لأن الحجم الكلي غير معروف قبل انتهاء الاستكشاف،
  // لذلك نُظهرها فقط عند معرفة الحجم. النسب الحقيقية أدناه هي المتقدمة
  // الملاحظة (processed / totalBytes / throughput).
  const candidates = progress?.videoCandidates || 0;
  const processedRatio = candidates > 0 ? Math.min(100, (totalProcessed / candidates) * 100) : null;

  return (
    <GlassCard className="p-6 border-fuchsia-500/30">
      {/* ── رأس اللوحة وأزرار التحكم ─────────────────────── */}
      <div className="flex flex-col gap-4 border-b border-white/10 pb-4 md:flex-row md:items-center:justify-between">
        <div>
          <h2 className="text-lg font-bold text-white flex items-center gap-2">
            <span className={`h-2.5 w-2.5 rounded-full ${isPaused ? "bg-amber-400" : "bg-emerald-400 animate-pulse"}`} />
            <span>مركز التحكم في الفحص</span>
            <span className="rounded-md border-white/10 bg-white/5 px-2 py-0.5 text-[11px] font-bold text-white/70">
              {SCAN_STATE_LABELS[scanState] || scanState}
            </span>
          </h2>
          <p className="mt-1 font-mono text-[11px] text-white/40" dir="ltr">
            {progress?.scanId || "—"} · {progress?.root || "كل المسارات"}
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {!isPaused ? (
            <button
              type="button"
              onClick={handlePause}
              disabled={actionState !== "idle"}
              className="rounded-xl border-amber-500/40 bg-amber-500/15 px-4 py-2 text-xs font-bold text-amber-200 transition hover:bg-amber-500/25 disabled:opacity-50"
            >
              {actionState === "pausing" ? "جارٍ الإيقاف…" : "⏸ إيقاف مؤقت"}
            </button>
          ) : (
            <button
              type="button"
              onClick={handleResume}
              disabled={actionState !== "idle"}
              className="rounded-xl border-emerald-500/40 bg-emerald-500/15 px-4 py-2 text-xs font-bold text-emerald-200 transition hover:bg-emerald-500/25 disabled:opacity-50"
            >
              {actionState === "resuming" ? "جارٍ الاستئناف…" : "▶ استئناف"}
            </button>
          )}

          {confirmCancel ? (
            <div className="flex items-center gap-2 rounded-xl border-rose-500/40 bg-rose-950/30 px-3 py-1.5">
              <span className="text-[11px] font-bold text-rose-200">إلغاء نهائي؟</span>
              <button
                type="button"
                onClick={handleCancel}
                disabled={actionState !== "idle"}
                className="rounded-lg bg-rose-600 px-3 py-1 text-[11px] font-bold text-white hover:bg-rose-500 disabled:opacity-50"
              >
                نعم، إلغاء
              </button>
              <button
                type="button"
                onClick={() => setConfirmCancel(false)}
                className="rounded-lg bg-white/10 px-3 py-1 text-[11px] font-bold text-white/70 hover:bg-white/20"
              >
                تراجع
              </button>
            </div>
          ) : (
            <button
              type="button"
              onClick={() => setConfirmCancel(true)}
              disabled={actionState !== "idle"}
              className="rounded-xl border-rose-500/30 bg-rose-950/20 px-4 py-2 text-xs font-bold text-rose-300 transition hover:bg-rose-950/40 disabled:opacity-50"
            >
              ⏹ إلغاء الفحص
            </button>
          )}
        </div>
      </div>

      {/* تنبيه الفرق بين الإيقاف المؤقت والإلغاء */}
      {isPaused && (
        <div className="mt-4 rounded-xl border-amber-500/30 bg-amber-950/20 p-3 text-xs text-amber-200">
          <strong>الفحص متوقف مؤقتًا.</strong> العدّادات والـ cursor وحالة كل قرص محفوظة — الاستئناف
          يكمل من نفس النقطة. الإلغاء وحده يُنهي الفحص ويفقد التقدم.
          {(state.control?.draining ?? 0) > 0 && (
            <span className="mt-1 block text-amber-300/80">
              {state.control.draining} عامل يُنهي العنصر الحالي قبل التوقف التام.
            </span>
          )}
        </div>
      )}

      {/* ── الأرقام الحقيقية ─────────────────────────────── */}
      <div className="mt-5 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Metric label="مجلدات" value={formatNumber(progress?.directoriesVisited)} />
        <Metric label="ملفات مفحوصة" value={formatNumber(progress?.filesSeen)} />
        <Metric label="مرشحة للوسائط" value={formatNumber(progress?.videoCandidates)} tone="text-cyan-300" />
        <Metric label="مقبولة" value={formatNumber(progress?.acceptedMedia)} tone="text-emerald-300" />

        <Metric label="جديدة" value={formatNumber(progress?.newFiles)} tone="text-emerald-300" />
        <Metric label="متغيرة" value={formatNumber(progress?.modifiedFiles)} tone="text-amber-300" />
        <Metric label="منقولة" value={formatNumber(progress?.renamedFiles)} tone="text-cyan-300" />
        <Metric label="غير متغيرة" value={formatNumber(progress?.unchangedFiles)} />

        <Metric label="حجم إجمالي" value={formatBytes(progress?.totalBytes)} />
        <Metric label="تم تجاوزه" value={formatBytes(progress?.skippedBytes)} />
        <Metric label="ملفات/ثانية" value={Number(progress?.throughput?.filesPerSecond || 0).toFixed(0)} tone="text-fuchsia-300" />
        <Metric label="MB/ثانية" value={Number(progress?.throughput?.mbPerSecond || 0).toFixed(1)} tone="text-fuchsia-300" />

        <Metric label="أخطاء صلاحيات" value={formatNumber(progress?.permissionErrors)} tone={progress?.permissionErrors ? "text-rose-300" : ""} />
        <Metric label="أخطاء نظام ملفات" value={formatNumber(progress?.filesystemErrors)} tone={progress?.filesystemErrors ? "text-rose-300" : ""} />
        <Metric label="ثقة منخفضة" value={formatNumber(progress?.lowConfidenceItems)} tone={progress?.lowConfidenceItems ? "text-amber-300" : ""} />
        <Metric label="مدة" value={progress?.duration?.human || "—"} />
      </div>

      {/* شريط تقدم حقي المعنى */}
      {processedRatio !== null && (
        <div className="mt-5">
          <div className="flex items-center justify-between text-[11px] text-white/50">
            <span>تمت معالجة {formatNumber(totalProcessed)} من {formatNumber(candidates)} مرشح</span>
            <span>{processedRatio.toFixed(1)}%</span>
          </div>
          <div className="mt-1.5 h-2 w-full overflow-hidden rounded-full bg-white/10">
            <div
              className={`h-full rounded-full transition-all duration-500 ${isPaused ? "bg-amber-500" : "bg-gradient-to-r from-purple-500 to-fuchsia-500"}`}
              style={{ width: `${processedRatio}%` }}
            />
          </div>
        </div>
      )}

      {/* ── حالة الأقراص ─────────────────── */}
      {progress?.rootStatus && Object.keys(progress.rootStatus).length > 0 && (
        <div className="mt-5">
          <h3 className="text-xs font-bold text-white/60">حالة الأقراص (كل قرص مستقل)</h3>
          <div className="mt-2 flex flex-wrap gap-2">
            {Object.entries(progress.rootStatus).map(([root, status]) => (
              <span
                key={root}
                className={`rounded-lg border px-2.5 py-1 font-mono text-[11px] ${status === "completed"
                    ? "border-emerald-500/30 bg-emerald-500/10 text-emerald-300"
                    : status === "unavailable" || status === "offline"
                      ? "border-rose-500/30 bg-rose-500/10 text-rose-300"
                      : "border-white/10 bg-white/5 text-white/60"
                  }`}
                dir="ltr"
              >
                {root} · {ROOT_STATUS_LABELS[status] || status}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* ── جدول العوامل ─────────────────── */}
      <div className="mt-5">
        <div className="flex items-center justify-between">
          <h3 className="text-xs font-bold text-white/60">
            العوامل (Workers) — {state.activeWorkers} نشط · {state.idleWorkers} خامل
          </h3>
          <span className="text-[11px] text-white/30">يُحدَّث كل {(POLL_INTERVAL_MS / 1000).toFixed(1)}ث</span>
        </div>

        <div className="mt-2 overflow-hidden rounded-xl border-white/10">
          <table className="w-full text-right text-xs" dir="rtl">
            <thead className="bg-white/[0.04] text-[11px] text-white/50">
              <tr>
                <th className="px-3 py-2 font-bold">العامل</th>
                <th className="px-3 py-2 font-bold">المهمة</th>
                <th className="px-3 py-2 font-bold">الحالة</th>
                <th className="px-3 py-2 font-bold">الملف الحالي</th>
                <th className="px-3 py-2 font-bold">مُعالج</th>
              </tr>
            </thead>
            <tbody>
              {state.workers.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-3 py-4 text-center text-white/30">
                    لا توجد بيانات عوامل بعد…
                  </td>
                </tr>
              ) : (
                state.workers.map((worker) => (
                  <tr key={worker.id} className="border-t border-white/5">
                    <td className="px-3 py-2 font-mono text-white/70">#{worker.id}</td>
                    <td className="px-3 py-2">
                      <span
                        className={`rounded-md border px-2 py-0.5 text-[11px] font-bold ${ROLE_TONES[worker.role] || ROLE_TONES.idle
                          }`}
                      >
                        {ROLE_LABELS[worker.role] || worker.role}
                      </span>
                    </td>
                    <td className="px-3 py-2">
                      {worker.active ? (
                        <span className="text-emerald-400">● نشط</span>
                      ) : (
                        <span className="text-white/30">○ خامل</span>
                      )}
                    </td>
                    <td className="px-3 py-2 font-mono text-[11px] text-white/50" dir="ltr">
                      {worker.currentPath ? shortenPath(worker.currentPath) : "—"}
                    </td>
                    <td className="px-3 py-2 font-mono text-white/60">{formatNumber(worker.processed)}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* ── المشاكل المكتشفة ─────────────────────────────── */}
      {Array.isArray(progress?.topProblems) && progress.topProblems.length > 0 && (
        <div className="mt-5">
          <h3 className="text-xs font-bold text-white/60">أهم المشاكل المكتشفة</h3>
          <div className="mt-2 space-y-2">
            {progress.topProblems.map((problem, index) => (
              <div
                key={`${problem.code || problem.title}-${index}`}
                className="rounded-xl border-white/10 bg-white/[0.03] p-3"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="text-xs font-bold text-white/90">{problem.title}</span>
                  <span className="rounded-md bg-white/10 px-2 py-0.5 font-mono text-[11px] text-white/60">
                    {formatNumber(problem.count)}
                  </span>
                </div>
                <p className="mt-1 text-[11px] leading-relaxed text-white/50">{problem.detail}</p>
                <div className="mt-1 flex flex-wrap items-center gap-3 text-[11px]">
                  <span className="text-cyan-300/80">تأثير البيانات: {problem.affectsDataQuality}</span>
                  <span className="text-amber-300/80">الإجراء: {problem.urgency}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* ── الأخطاء المصنّفة ─────────────────────────────── */}
      {Array.isArray(progress?.errors) && progress.errors.length > 0 && (
        <div className="mt-5">
          <h3 className="text-xs font-bold text-white/60">الأخطاء حسب النوع</h3>
          <div className="mt-2 flex flex-wrap gap-2">
            {progress.errors.map((item) => (
              <span
                key={item.code}
                className="rounded-lg border-rose-500/20 bg-rose-950/20 px-2.5 py-1 font-mono text-[11px] text-rose-300"
              >
                {item.code}: {formatNumber(item.count)}
              </span>
            ))}
          </div>
        </div>
      )}

      {error && (
        <div className="mt-4 rounded-xl border-amber-500/30 bg-amber-950/20 p-3 text-xs text-amber-300">
          ⚠️ {error}
        </div>
      )}
    </GlassCard>
  );
}

/** بطاقة رقم واحد. */
function Metric({ label, value, tone = "text-white" }) {
  return (
    <div className="rounded-xl border-white/5 bg-black/30 p-3">
      <p className="text-[11px] text-white/40">{label}</p>
      <p className={`mt-0.5 font-mono text-sm font-bold ${tone}`} dir="ltr">
        {value}
      </p>
    </div>
  );
}
