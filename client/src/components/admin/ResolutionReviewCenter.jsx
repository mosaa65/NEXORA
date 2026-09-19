import { useCallback, useEffect, useState } from "react";
import GlassCard from "../../components/GlassCard.jsx";
import Icon from "../../components/Icon.jsx";
import {
  getResolutionQueue,
  getResolutionStats,
  decideResolution,
  searchLibrary
} from "../../lib/api.js";

/**
 * ResolutionReviewCenter — شاشة "يحتاج مراجعة"
 *
 * تجمع الملفات التي رفض Entity Resolution الحسم فيها. هذه هي الملفات التي كانت
 * سابقًا ستُصبح أعمالًا باسم "01" أو باسم موقع، فصارت الآن بانتظار قرار بشري.
 *
 * لكل عنصر تعرض الشاشة:
 *   - المسار الأصلي والاسم المكتشف
 *   - سبب عدم الحسم بكود قابل للقراءة
 *   - المرشحون مرتّبون بدرجاتهم مع تفصيل الأدلة
 *
 * والقرار يُنفّذ فعليًا ويمكن ترقيته إلى alias دائم (learn_alias) فلا يتكرر
 * نفس الغموض.
 */

/** التسميات العربية لأكواد أسباب عدم الحسم. */
const REASON_LABELS = {
  ambiguous_work: "عنوانان متقاربان — لا يمكن الترجيح",
  unknown_media_type: "نوع غير محدد (فيلم أم مسلسل؟)",
  no_usable_title: "لا يوجد عنوان صالح",
  missing_title: "عنوان مفقود",
  low_confidence_attachment: "ارتباط بثقة منخفضة",
  creation_disabled: "الإنشاء التلقائي معطّل",
  new_work: "عمل جديد مقترح",
  episode_without_number: "حلقة بدون رقم",
  invalid_season: "رقم موسم غير منطقي",
  impossible_episode: "رقم حلقة غير منطقي",
  duplicate_episode_identity: "تعارض في هوية الحلقة",
  duplicate_path: "مسار مكرر",
  missing_category: "تصنيف مفقود"
};

const REASON_TONES = {
  ambiguous_work: "border-amber-500/30 bg-amber-500/10 text-amber-300",
  unknown_media_type: "border-orange-500/30 bg-orange-500/10 text-orange-300",
  no_usable_title: "border-rose-500/30 bg-rose-500/10 text-rose-300",
  duplicate_episode_identity: "border-rose-500/30 bg-rose-500/10 text-rose-300",
  duplicate_path: "border-rose-500/30 bg-rose-500/10 text-rose-300"
};

/** التسميات العربية لأزرار القرار. */
const ACTION_LABELS = {
  attach: "اربط بعمل موجود",
  create: "أنشئ عملًا جديدًا",
  ignore: "تجاهل",
  mark_movie: "حدّده كفيلم",
  move_season: "انقله لموسم آخر"
};

function formatBytes(bytes) {
  const value = Number(bytes) || 0;
  if (value <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const exponent = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  return `${(value / Math.pow(1024, exponent)).toFixed(exponent === 0 ? 0 : 1)} ${units[exponent]}`;
}

function formatConfidence(value) {
  const numeric = Number(value) || 0;
  if (numeric <= 0) return "—";
  const percent = numeric > 1 ? numeric : numeric * 100;
  return `${percent.toFixed(0)}%`;
}

function confidenceTone(value) {
  const numeric = Number(value) || 0;
  const percent = numeric > 1 ? numeric : numeric * 100;
  if (percent >= 70) return "text-emerald-300";
  if (percent >= 45) return "text-amber-300";
  return "text-rose-300";
}

export default function ResolutionReviewCenter({ onCountChange }) {
  const [items, setItems] = useState([]);
  const [stats, setStats] = useState({ total: 0, byReason: [] });
  const [filter, setFilter] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // الجلسة: العنصر المفتوح للحسم
  const [active, setActive] = useState(null);
  const [decision, setDecision] = useState({
    action: "attach",
    workId: 0,
    season: 0,
    episode: 0,
    newTitle: "",
    learnAlias: true
  });
  const [submitting, setSubmitting] = useState(false);
  const [actionError, setActionError] = useState("");
  const [successMessage, setSuccessMessage] = useState("");

  // بحث العمل لربطه
  const [workQuery, setWorkQuery] = useState("");
  const [workOptions, setWorkOptions] = useState([]);
  const [searching, setSearching] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [queue, statistics] = await Promise.all([
        getResolutionQueue({ reason: filter || undefined, limit: 200 }),
        getResolutionStats()
      ]);
      setItems(queue?.items ?? []);
      setStats(statistics ?? { total: 0, byReason: [] });
      onCountChange?.(statistics?.total ?? 0);
      setError("");
    } catch (err) {
      setError(err?.message || "تعذّر تحميل قائمة المراجعة.");
    } finally {
      setLoading(false);
    }
  }, [filter, onCountChange]);

  useEffect(() => {
    load();
  }, [load]);

  /** يفتح عنصرًا ويُهيّئ النموذج من بياناته المكتشفة. */
  function openItem(item) {
    setActive(item);
    setActionError("");
    setSuccessMessage("");
    setWorkQuery("");
    setWorkOptions([]);
    setDecision({
      action: "attach",
      // إن وُجد مرشح واحد نظيف نعرضه كبداية مقترحة
      workId: item.candidates?.[0]?.work_id ?? 0,
      season: item.detected_season ?? 0,
      episode: item.detected_episode ?? 0,
      newTitle: item.detected_title ?? "",
      learnAlias: true
    });
  }

  /** يبحث عن عمل لربطه بالملف. */
  async function searchWorks(query) {
    setWorkQuery(query);
    if (!query || query.trim().length < 2) {
      setWorkOptions([]);
      return;
    }
    setSearching(true);
    try {
      const result = await searchLibrary(query.trim(), { limit: 8 });
      setWorkOptions((result?.hits ?? []).map((hit) => ({
        id: hit.id,
        title: hit.title_ar || hit.title_en,
        titleEn: hit.title_en,
        type: hit.type,
        year: hit.release_year
      })));
    } catch {
      setWorkOptions([]);
    } finally {
      setSearching(false);
    }
  }

  /** يطبّق القرار على العنصر النشط. */
  async function submitDecision() {
    if (!active) return;

    if (decision.action === "attach" && !decision.workId) {
      setActionError("اختر العمل الذي تريد ربط الملف به.");
      return;
    }
    if (decision.action === "create" && !decision.newTitle.trim()) {
      setActionError("اكتب اسم العمل الجديد.");
      return;
    }

    setSubmitting(true);
    setActionError("");
    try {
      await decideResolution(active.id, decision);
      setSuccessMessage(
        decision.learnAlias
          ? "تم تطبيق القرار وتعلّم الربط — لن يتكرر هذا الغموض."
          : "تم تطبيق القرار."
      );
      setActive(null);
      await load();
    } catch (err) {
      setActionError(err?.message || "تعذّر تطبيق القرار.");
    } finally {
      setSubmitting(false);
    }
  }

  const totalPending = stats.total || 0;

  return (
    <GlassCard className="p-6 border-amber-500/20">
      {/* ── الرأس ──────────────────────── */}
      <div className="flex flex-col gap-4 border-b border-white/10 pb-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h2 className="text-lg font-bold text-white flex items-center gap-2">
            <Icon name="spark" className="h-5 w-5 text-amber-400" />
            <span>مركز المراجعة — يحتاج قرارًا</span>
            {totalPending > 0 && (
              <span className="rounded-full border-amber-500/40 bg-amber-500/15 px-2.5 py-0.5 text-xs font-bold text-amber-300">
                {totalPending}
              </span>
            )}
          </h2>
          <p className="mt-1 text-xs text-white/50">
            ملفات لم يستطع النظام الحسم فيها بأمان. لم يُخمّن لها عنوان ولا موسم — القرار لك.
          </p>
        </div>

        <button
          type="button"
          onClick={load}
          disabled={loading}
          className="rounded-xl border-white/10 bg-white/[0.06] px-4 py-2 text-xs font-bold text-white transition hover:bg-white/10 disabled:opacity-50"
        >
          {loading ? "جارٍ التحديث…" : "🔄 تحديث"}
        </button>
      </div>

      {/* ── الأسباب مجمّعة ─────────────────────────────── */}
      {Array.isArray(stats.byReason) && stats.byReason.length > 0 && (
        <div className="mt-4 flex flex-wrap gap-2">
          <button
            type="button"
            onClick={() => setFilter("")}
            className={`rounded-lg border px-3 py-1.5 text-[11px] font-bold transition ${filter === ""
                ? "border-fuchsia-500/50 bg-fuchsia-500/20 text-fuchsia-200"
                : "border-white/10 bg-white/5 text-white/60 hover:bg-white/10"
              }`}
          >
            الكل ({totalPending})
          </button>
          {stats.byReason.map((entry) => (
            <button
              key={entry.reason}
              type="button"
              onClick={() => setFilter(entry.reason)}
              className={`rounded-lg border px-3 py-1.5 text-[11px] font-bold transition ${filter === entry.reason
                  ? "border-fuchsia-500/50 bg-fuchsia-500/20 text-fuchsia-200"
                  : "border-white/10 bg-white/5 text-white/60 hover:bg-white/10"
                }`}
            >
              {REASON_LABELS[entry.reason] || entry.reason} ({entry.count})
            </button>
          ))}
        </div>
      )}

      {successMessage && (
        <div className="mt-4 rounded-xl border-emerald-500/30 bg-emerald-950/20 p-3 text-xs text-emerald-300">
          ✓ {successMessage}
        </div>
      )}
      {error && (
        <div className="mt-4 rounded-xl border-rose-500/30 bg-rose-950/20 p-3 text-xs text-rose-300">
          ⚠️ {error}
        </div>
      )}

      {/* ── قائمة العناصر ──────────────────────────────── */}
      {!active && (
        <div className="mt-4">
          {loading && items.length === 0 ? (
            <p className="py-8 text-center text-xs text-white/40">جارٍ التحميل…</p>
          ) : items.length === 0 ? (
            <div className="rounded-2xl border-emerald-500/20 bg-emerald-950/10 p-8 text-center">
              <p className="text-sm font-bold text-emerald-300">✓ لا يوجد ما يحتاج مراجعة</p>
              <p className="mt-1 text-xs text-white/50">
                كل الملفات المفهرسة ارتبطت بعمل بثقة كافية.
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {items.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => openItem(item)}
                  className="w-full rounded-2xl border-white/10 bg-white/[0.03] p-4 text-right transition hover:border-amber-500/40 hover:bg-amber-950/10"
                >
                  <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span
                          className={`shrink-0 rounded-md border px-2 py-0.5 text-[11px] font-bold ${REASON_TONES[item.reason] ||
                            "border-white/10 bg-white/5 text-white/60"
                            }`}
                        >
                          {REASON_LABELS[item.reason] || item.reason}
                        </span>
                        {item.detected_media_type && (
                          <span className="shrink-0 rounded-md bg-white/5 px-2 py-0.5 text-[11px] text-white/50">
                            {item.detected_media_type}
                          </span>
                        )}
                      </div>

                      <p className="mt-1.5 truncate font-mono text-xs text-white/80" dir="ltr">
                        {item.original_filename || item.file_path}
                      </p>

                      <div className="mt-1 flex flex-wrap gap-3 text-[11px] text-white/40">
                        <span dir="ltr">
                          {item.detected_title ? `مكتشف: ${item.detected_title}` : "بلا عنوان مكتشف"}
                        </span>
                        {(item.detected_season > 0 || item.detected_episode > 0) && (
                          <span dir="ltr">
                            موسم {item.detected_season || "—"} · حلقة {item.detected_episode || "—"}
                          </span>
                        )}
                        <span dir="ltr">{formatBytes(item.size)}</span>
                        <span className={confidenceTone(item.parser_confidence)}>
                          ثقة التحليل {formatConfidence(item.parser_confidence)}
                        </span>
                      </div>
                    </div>

                    <span className="shrink-0 self-start rounded-xl border-fuchsia-500/30 bg-fuchsia-500/10 px-3 py-1.5 text-[11px] font-bold text-fuchsia-200 md:self-center">
                      اتخاذ قرار ↵
                    </span>
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>
      )}

      {/* ── لوحة الحسم ─────────────────── */}
      {active && (
        <div className="mt-4 space-y-4">
          {/* ملخص العنصر */}
          <div className="rounded-2xl border-cyan-500/30 bg-[#0B0916] p-5">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <h3 className="text-sm font-bold text-cyan-200">الملف الذي يحتاج قرارًا</h3>
                <p className="mt-1.5 truncate font-mono text-xs text-white/80" dir="ltr">
                  {active.file_path}
                </p>
                <p className="mt-1 text-[11px] text-white/50">
                  {REASON_LABELS[active.reason] || active.reason}
                  {active.reason_detail ? ` — ${active.reason_detail}` : ""}
                </p>
              </div>
              <button
                type="button"
                onClick={() => setActive(null)}
                className="shrink-0 rounded-lg bg-white/10 px-3 py-1.5 text-[11px] font-bold text-white/70 hover:bg-white/20"
              >
                إغلاق
              </button>
            </div>
          </div>

          {/* المرشحون بتفصيل الأدلة */}
          {Array.isArray(active.candidates) && active.candidates.length > 0 && (
            <div className="rounded-2xl border-white/10 bg-white/[0.02] p-4">
              <h4 className="text-xs font-bold text-white/70">
                المرشحون الذين حسبهم النظام (مرتّبون بالأولوية)
              </h4>
              <p className="mt-1 text-[11px] text-white/40">
                كل درجة مبنية على أدلة مسمّاة حتى ترى سبب الترشيح، لا مجرد رقم.
              </p>

              <div className="mt-3 space-y-2">
                {active.candidates.map((candidate) => (
                  <div
                    key={candidate.work_id}
                    className={`rounded-xl border p-3 transition ${decision.workId === candidate.work_id
                        ? "border-emerald-500/50 bg-emerald-950/15"
                        : "border-white/10 bg-black/20"
                      }`}
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div className="flex items-center gap-2">
                        <button
                          type="button"
                          onClick={() =>
                            setDecision((previous) => ({
                              ...previous,
                              action: "attach",
                              workId: candidate.work_id
                            }))
                          }
                          className={`rounded-lg px-3 py-1 text-[11px] font-bold transition ${decision.workId === candidate.work_id
                              ? "bg-emerald-600 text-white"
                              : "bg-white/10 text-white/70 hover:bg-white/20"
                            }`}
                        >
                          {decision.workId === candidate.work_id ? "✓ مُختار" : "اختر"}
                        </button>
                        <span className="text-sm font-bold text-white/90" dir="auto">
                          {candidate.title}
                        </span>
                      </div>
                      <span className="font-mono text-xs text-fuchsia-300">
                        {Number(candidate.score || 0).toFixed(0)} نقطة
                      </span>
                    </div>

                    {Array.isArray(candidate.breakdown) && candidate.breakdown.length > 0 && (
                      <div className="mt-2 flex flex-wrap gap-1.5">
                        {candidate.breakdown.map((evidence, index) => (
                          <span
                            key={`${evidence.label}-${index}`}
                            title={evidence.detail}
                            className={`rounded-md px-2 py-0.5 font-mono text-[10px] ${evidence.points >= 0
                                ? "bg-emerald-500/10 text-emerald-300"
                                : "bg-rose-500/10 text-rose-300"
                              }`}
                          >
                            {evidence.label} {evidence.points >= 0 ? "+" : ""}
                            {Number(evidence.points).toFixed(0)}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* أزرار القرار */}
          <div className="rounded-2xl border-fuchsia-500/30 bg-black/30 p-4">
            <h4 className="text-xs font-bold text-white/70">القرار</h4>
            <div className="mt-2 flex flex-wrap gap-2">
              {["attach", "create", "mark_movie", "ignore"].map((action) => (
                <button
                  key={action}
                  type="button"
                  onClick={() => setDecision((previous) => ({ ...previous, action }))}
                  className={`rounded-xl border px-3.5 py-2 text-xs font-bold transition ${decision.action === action
                      ? "border-fuchsia-500/50 bg-fuchsia-500/20 text-fuchsia-200"
                      : "border-white/10 bg-white/5 text-white/60 hover:bg-white/10"
                    }`}
                >
                  {ACTION_LABELS[action]}
                </button>
              ))}
            </div>

            {/* رابط بعمل: بحث */}
            {decision.action === "attach" && (
              <div className="mt-4">
                <label className="text-[11px] font-bold text-white/60">
                  ابحث عن العمل الذي تريد الربط به
                </label>
                <input
                  value={workQuery}
                  onChange={(event) => searchWorks(event.target.value)}
                  placeholder="اكتب اسم العمل…"
                  className="mt-1.5 w-full rounded-xl border-white/10 bg-black/40 px-3.5 py-2.5 text-sm text-white outline-none placeholder:text-white/30 focus:border-fuchsia-500"
                />

                {searching && (
                  <p className="mt-2 text-[11px] text-white/40">جارٍ البحث…</p>
                )}

                {workOptions.length > 0 && (
                  <div className="mt-2 space-y-1.5">
                    {workOptions.map((option) => (
                      <button
                        key={option.id}
                        type="button"
                        onClick={() =>
                          setDecision((previous) => ({
                            ...previous,
                            workId: option.id
                          }))
                        }
                        className={`w-full rounded-xl border px-3 py-2 text-right transition ${decision.workId === option.id
                            ? "border-emerald-500/50 bg-emerald-950/20"
                            : "border-white/10 bg-white/[0.03] hover:bg-white/[0.07]"
                          }`}
                      >
                        <span className="text-xs font-bold text-white/90">
                          {option.title}
                        </span>
                        <span className="ml-2 font-mono text-[10px] text-white/40" dir="ltr">
                          #{option.id} · {option.type}
                          {option.year ? ` · ${option.year}` : ""}
                        </span>
                      </button>
                    ))}
                  </div>
                )}

                {decision.workId > 0 && (
                  <p className="mt-2 text-[11px] text-emerald-300">
                    سيُربط بالعمل #{decision.workId}
                  </p>
                )}
              </div>
            )}

            {/* إنشاء عمل جديد: الاسم */}
            {decision.action === "create" && (
              <div className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-2">
                <div>
                  <label className="text-[11px] font-bold text-white/60">اسم العمل الجديد</label>
                  <input
                    value={decision.newTitle}
                    onChange={(event) =>
                      setDecision((previous) => ({ ...previous, newTitle: event.target.value }))
                    }
                    placeholder="مثال: The Dark Knight Rises"
                    className="mt-1.5 w-full rounded-xl border-white/10 bg-black/40 px-3.5 py-2.5 text-sm text-white outline-none placeholder:text-white/30 focus:border-fuchsia-500"
                  />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-white/60">
                    اسم العمل الأصلي المربوط (اختياري)
                  </label>
                  <input
                    value={decision.workId || ""}
                    onChange={(event) =>
                      setDecision((previous) => ({
                        ...previous,
                        workId: Number(event.target.value) || 0
                      }))
                    }
                    placeholder="معرّف عمل موجود لإعادة التسمية"
                    className="mt-1.5 w-full rounded-xl border-white/10 bg-black/40 px-3.5 py-2.5 font-mono text-sm text-white outline-none placeholder:text-white/30 focus:border-fuchsia-500"
                    dir="ltr"
                  />
                </div>
              </div>
            )}

            {/* الموسم والحلقة */}
            {(decision.action === "attach" || decision.action === "mark_movie") && (
              <div className="mt-4 grid grid-cols-2 gap-3">
                <div>
                  <label className="text-[11px] font-bold text-white/60">الموسم</label>
                  <input
                    type="number"
                    min="0"
                    value={decision.season}
                    onChange={(event) =>
                      setDecision((previous) => ({
                        ...previous,
                        season: Number(event.target.value) || 0
                      }))
                    }
                    className="mt-1.5 w-full rounded-xl border-white/10 bg-black/40 px-3.5 py-2.5 font-mono text-sm text-white outline-none focus:border-fuchsia-500"
                    dir="ltr"
                  />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-white/60">الحلقة</label>
                  <input
                    type="number"
                    min="0"
                    value={decision.episode}
                    onChange={(event) =>
                      setDecision((previous) => ({
                        ...previous,
                        episode: Number(event.target.value) || 0
                      }))
                    }
                    className="mt-1.5 w-full rounded-xl border-white/10 bg-black/40 px-3.5 py-2.5 font-mono text-sm text-white outline-none focus:border-fuchsia-500"
                    dir="ltr"
                  />
                </div>
              </div>
            )}

            {/* تعلّم الربط */}
            <label className="mt-4 flex cursor-pointer items-start gap-2.5 rounded-xl border-white/10 bg-white/[0.02] p-3">
              <input
                type="checkbox"
                checked={decision.learnAlias}
                onChange={(event) =>
                  setDecision((previous) => ({ ...previous, learnAlias: event.target.checked }))
                }
                className="mt-0.5 h-4 w-4 shrink-0 accent-fuchsia-500"
              />
              <span className="text-[11px] leading-relaxed text-white/60">
                <strong className="text-white/80">تعلّم هذا الربط</strong> — يُحفظ الاسم المكتشف
                كـ alias دائم لهذا العمل، فلا يتكرر نفس الغموض في ملفات قادمة. يُنصح بتفعيله عند
                التأكد من القرار.
              </span>
            </label>

            {actionError && (
              <div className="mt-3 rounded-xl border-rose-500/30 bg-rose-950/20 p-3 text-xs text-rose-300">
                ⚠️ {actionError}
              </div>
            )}

            <div className="mt-4 flex items-center gap-2">
              <button
                type="button"
                onClick={submitDecision}
                disabled={submitting}
                className="rounded-xl bg-gradient-to-r from-purple-600 via-fuchsia-600 to-purple-700 px-5 py-2.5 text-xs font-bold text-white transition hover:brightness-110 disabled:opacity-50"
              >
                {submitting ? "جارٍ التطبيق…" : "تطبيق القرار"}
              </button>
              <button
                type="button"
                onClick={() => setActive(null)}
                className="rounded-xl bg-white/10 px-4 py-2.5 text-xs font-bold text-white/70 hover:bg-white/20"
              >
                إلغاء
              </button>
            </div>
          </div>
        </div>
      )}
    </GlassCard>
  );
}
