import { useMemo, useState } from "react";
import GlassCard from "../components/GlassCard.jsx";
import Icon from "../components/Icon.jsx";
import { copyMedia, previewMigration } from "../lib/api.js";

const GROUPS = ["all", "movies", "series", "anime"];
const ACTIONS = ["all", "keep", "move", "rename", "duplicate"];

export default function AdminPage({ health, onSyncIndex }) {
  const [previewRoot, setPreviewRoot] = useState("");
  const [previewState, setPreviewState] = useState("idle");
  const [previewResult, setPreviewResult] = useState(null);
  const [activeGroup, setActiveGroup] = useState("all");
  const [activeAction, setActiveAction] = useState("all");
  const [searchTerm, setSearchTerm] = useState("");
  const [expandedGroups, setExpandedGroups] = useState(() => new Set(["movies", "series", "anime"]));
  const [expandedFolders, setExpandedFolders] = useState(() => new Set());
  const [copyTarget, setCopyTarget] = useState("");
  const [copySources, setCopySources] = useState("");
  const [copyState, setCopyState] = useState("idle");
  const [copyResult, setCopyResult] = useState(null);

  const services = [
    { label: "الـ API", value: health?.ok ? "يعمل" : "متوقف", tone: health?.ok ? "text-emerald-300" : "text-rose-300" },
    { label: "قاعدة البيانات", value: health?.database?.databaseOk ? "متصلة" : "بانتظار", tone: health?.database?.databaseOk ? "text-emerald-300" : "text-rose-300" },
    { label: "مراقبة المجلدات", value: "نشطة", tone: "text-cyan-300" },
    { label: "البث", value: "جاهز للمدى", tone: "text-fuchsia-300" }
  ];

  const tasks = [
    "معاينة نقل المجلدات",
    "فحص التكرار",
    "صف التحقق من SHA-256",
    "تحويل الترجمة إلى WebVTT",
    "توليد المصغرات"
  ];

  const previewEntries = previewResult?.entries || [];
  const stats = previewResult
    ? {
        total: previewEntries.length,
        keep: previewResult.keepCount || 0,
        move: previewResult.moveCount || 0,
        rename: previewResult.renameCount || 0,
        duplicate: previewResult.duplicateCount || 0
      }
    : { total: 0, keep: 0, move: 0, rename: 0, duplicate: 0 };

  const filteredEntries = useMemo(() => {
    const term = searchTerm.trim().toLowerCase();
    return previewEntries.filter((entry) => {
      if (activeGroup !== "all" && normalizeKind(entry.kind) !== activeGroup) {
        return false;
      }
      if (activeAction !== "all" && (entry.action || "move") !== activeAction) {
        return false;
      }
      if (!term) {
        return true;
      }
      return [entry.title, entry.source, entry.target, entry.reason, entry.action, entry.kind]
        .filter(Boolean)
        .join(" ")
        .toLowerCase()
        .includes(term);
    });
  }, [activeAction, activeGroup, previewEntries, searchTerm]);

  const tree = useMemo(() => buildPreviewTree(filteredEntries), [filteredEntries]);

  async function handlePreview() {
    if (!previewRoot.trim()) {
      return;
    }
    setPreviewState("loading");
    try {
      const result = await previewMigration(previewRoot.trim());
      setPreviewResult(result);
      setPreviewState("ready");
    } catch {
      setPreviewState("error");
    }
  }

  async function handleCopy() {
    const sources = copySources
      .split("\n")
      .map((item) => item.trim())
      .filter(Boolean);
    if (sources.length === 0 || !copyTarget.trim()) {
      return;
    }
    setCopyState("loading");
    try {
      const result = await copyMedia({ sources, target: copyTarget.trim() });
      setCopyResult(result);
      setCopyState("ready");
    } catch {
      setCopyState("error");
    }
  }

  function toggleGroup(groupKey) {
    setExpandedGroups((current) => {
      const next = new Set(current);
      if (next.has(groupKey)) {
        next.delete(groupKey);
      } else {
        next.add(groupKey);
      }
      return next;
    });
  }

  function toggleFolder(folderKey) {
    setExpandedFolders((current) => {
      const next = new Set(current);
      if (next.has(folderKey)) {
        next.delete(folderKey);
      } else {
        next.add(folderKey);
      }
      return next;
    });
  }

  function expandAll() {
    setExpandedGroups(new Set(["movies", "series", "anime"]));
    const nextFolders = new Set();
    for (const group of tree.groups) {
      for (const folder of group.folders) {
        nextFolders.add(`${group.key}:${folder.path}`);
      }
    }
    setExpandedFolders(nextFolders);
  }

  function collapseAll() {
    setExpandedGroups(new Set());
    setExpandedFolders(new Set());
  }

  return (
    <div className="space-y-6">
      <GlassCard className="p-6">
        <p className="text-xs font-semibold text-electric/80">لوحة الإدارة</p>
        <h1 className="mt-4 text-4xl font-black text-white md:text-5xl">غرفة العمليات</h1>
        <p className="mt-3 max-w-3xl text-base leading-8 text-white/70">
          هنا تتم معاينة التنظيم قبل النقل الفعلي، ومراقبة حالة المنصة، وتنفيذ النسخ مع التحقق من سلامة الملف.
        </p>
        <div className="mt-6 flex flex-wrap gap-3">
          <button
            type="button"
            onClick={onSyncIndex}
            className="inline-flex items-center gap-2 rounded-2xl border border-electric/25 bg-electric/12 px-4 py-3 text-sm font-semibold text-electric"
          >
            <Icon name="spark" className="h-4 w-4" />
            مزامنة الفهرس
          </button>
          <button
            type="button"
            className="inline-flex items-center gap-2 rounded-2xl border border-white/10 bg-white/[0.05] px-4 py-3 text-sm font-semibold text-white/80"
          >
            <Icon name="disk" className="h-4 w-4" />
            فتح مدير الأقراص
          </button>
        </div>
      </GlassCard>

      <section className="grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
        <GlassCard className="p-6">
          <div className="flex items-center justify-between">
            <div className="text-right">
              <p className="text-xs font-semibold text-white/40">Migration Wizard</p>
              <h2 className="mt-2 text-2xl font-bold text-white">شجرة المعاينة</h2>
            </div>
            <Icon name="arrowRight" className="h-5 w-5 text-electric" />
          </div>

          <div className="mt-5 space-y-3 text-right">
            <label className="block">
              <span className="mb-2 block text-xs font-semibold text-white/45">مسار المصدر</span>
              <input
                value={previewRoot}
                onChange={(event) => setPreviewRoot(event.target.value)}
                placeholder="D:\\Media"
                className="w-full rounded-2xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white outline-none placeholder:text-white/30"
              />
            </label>
            <button
              type="button"
              onClick={handlePreview}
              className="inline-flex items-center gap-2 rounded-2xl border border-electric/25 bg-electric/12 px-4 py-3 text-sm font-semibold text-electric"
            >
              <Icon name="spark" className="h-4 w-4" />
              إنشاء المعاينة
            </button>
            <div className="rounded-2xl border border-white/10 bg-black/20 p-4 text-sm leading-7 text-white/65">
              الحالة: {previewState === "loading" ? "جارٍ الفحص" : previewState === "ready" ? `${stats.total} ملف` : previewState === "error" ? "تعذر الفحص" : "بانتظار إدخال المسار"}
            </div>
          </div>

          <div className="mt-5 grid gap-3 md:grid-cols-5">
            <StatCard label="إجمالي الملفات" value={stats.total} />
            <StatCard label="Keep" value={stats.keep} tone="text-emerald-300" />
            <StatCard label="Move" value={stats.move} tone="text-cyan-300" />
            <StatCard label="Rename" value={stats.rename} tone="text-fuchsia-300" />
            <StatCard label="Duplicate" value={stats.duplicate} tone="text-amber-300" />
          </div>

          <div className="mt-5 grid gap-3 md:grid-cols-2">
            <label className="block">
              <span className="mb-2 block text-xs font-semibold text-white/45">بحث داخل المعاينة</span>
              <input
                value={searchTerm}
                onChange={(event) => setSearchTerm(event.target.value)}
                placeholder="ابحث بالعنوان أو المسار أو السبب..."
                className="w-full rounded-2xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white outline-none placeholder:text-white/30"
              />
            </label>
            <div className="grid grid-cols-2 gap-2">
              <FilterPill label="Group" value={activeGroup} items={GROUPS} onChange={setActiveGroup} />
              <FilterPill label="Action" value={activeAction} items={ACTIONS} onChange={setActiveAction} />
            </div>
          </div>

          <div className="mt-5 flex flex-wrap gap-2">
            {GROUPS.map((group) => (
              <button
                key={group}
                type="button"
                onClick={() => setActiveGroup(group)}
                className={`rounded-full border px-4 py-2 text-xs font-semibold transition ${
                  activeGroup === group
                    ? "border-electric/25 bg-electric/12 text-electric"
                    : "border-white/10 bg-white/[0.04] text-white/65"
                }`}
              >
                {group === "all" ? "الكل" : group === "movies" ? "Movies" : group === "series" ? "Series" : "Anime"}
              </button>
            ))}
            {ACTIONS.map((action) => (
              <button
                key={action}
                type="button"
                onClick={() => setActiveAction(action)}
                className={`rounded-full border px-4 py-2 text-xs font-semibold transition ${
                  activeAction === action
                    ? "border-electric/25 bg-electric/12 text-electric"
                    : "border-white/10 bg-white/[0.04] text-white/65"
                }`}
              >
                {action}
              </button>
            ))}
            <button type="button" onClick={expandAll} className="rounded-full border border-white/10 bg-white/[0.04] px-4 py-2 text-xs font-semibold text-white/65">
              Expand all
            </button>
            <button type="button" onClick={collapseAll} className="rounded-full border border-white/10 bg-white/[0.04] px-4 py-2 text-xs font-semibold text-white/65">
              Collapse all
            </button>
          </div>

          <div className="mt-5 max-h-[34rem] overflow-auto pr-1">
            {previewState === "ready" && stats.total === 0 ? (
              <div className="rounded-2xl border border-white/10 bg-black/20 p-4 text-sm text-white/60">
                لم يتم العثور على ملفات فيديو في المسار المحدد.
              </div>
            ) : null}

            {tree.groups.map((group) => (
              <div key={group.key} className="mb-4 rounded-[1.4rem] border border-white/10 bg-black/18 p-4">
                <button
                  type="button"
                  onClick={() => toggleGroup(group.key)}
                  className="flex w-full items-center justify-between text-right"
                >
                  <div>
                    <h3 className="text-lg font-bold text-white">{group.label}</h3>
                    <p className="text-xs text-white/45">{group.count} ملف</p>
                  </div>
                  <span className="rounded-full border border-white/10 bg-white/[0.05] px-3 py-1 text-xs font-semibold text-white/70">
                    {expandedGroups.has(group.key) ? "Collapse" : "Expand"}
                  </span>
                </button>

                {expandedGroups.has(group.key) ? (
                  <div className="mt-4 space-y-2">
                    {group.folders.map((folder) => {
                      const folderKey = `${group.key}:${folder.path}`;
                      const isOpen = expandedFolders.has(folderKey) || group.count <= 6;
                      return (
                        <details key={folder.path} className="rounded-2xl border border-white/10 bg-black/18 p-3" open={isOpen}>
                          <summary
                            className="cursor-pointer list-none"
                            onClick={(event) => {
                              event.preventDefault();
                              toggleFolder(folderKey);
                            }}
                          >
                            <div className="flex items-center justify-between gap-3">
                              <div className="text-right">
                                <p className="text-sm font-semibold text-white">{folder.name}</p>
                                <p className="text-[11px] text-white/40">{folder.count} ملف</p>
                              </div>
                              <span className="rounded-full border border-white/10 bg-white/[0.05] px-3 py-1 text-[11px] font-semibold text-white/65">
                                {expandedFolders.has(folderKey) || group.count <= 6 ? "open" : "closed"}
                              </span>
                            </div>
                          </summary>

                          <div className="mt-3 space-y-2">
                            {folder.entries.map((entry) => (
                              <div key={`${entry.source}-${entry.target}`} className="rounded-2xl border border-white/10 bg-white/[0.04] p-3 text-right">
                                <div className="flex items-center justify-between gap-3">
                                  <span className={`rounded-full border px-3 py-1 text-[11px] font-semibold ${actionTone(entry.action)}`}>
                                    {entry.action}
                                  </span>
                                  <span className="text-[11px] text-white/40">{formatBytes(entry.size)}</span>
                                </div>
                                <p className="mt-2 break-all text-sm font-semibold text-white">{entry.title || "عنوان غير معروف"}</p>
                                <p className="mt-1 text-[11px] text-white/40">
                                  {entry.season ? `Season ${String(entry.season).padStart(2, "0")}` : "Movie"}
                                  {entry.episode ? ` · Episode ${String(entry.episode).padStart(2, "0")}` : ""}
                                  {entry.resolution ? ` · ${entry.resolution}` : ""}
                                </p>
                                <p className="mt-1 break-all text-xs text-white/45">{entry.source}</p>
                                <div className="mt-2 grid gap-2 text-[11px] text-white/60 md:grid-cols-2">
                                  <div className="rounded-xl border border-white/10 bg-black/24 px-3 py-2">
                                    <span className="text-white/35">From:</span> {entry.sourceName}
                                  </div>
                                  <div className="rounded-xl border border-white/10 bg-black/24 px-3 py-2">
                                    <span className="text-white/35">To:</span> {entry.targetName}
                                  </div>
                                </div>
                                <div className="mt-2 rounded-xl border border-electric/15 bg-black/25 px-3 py-2 text-[11px] text-white/65">
                                  <span className="text-white/35">→</span> {entry.target}
                                </div>
                                {entry.reason ? <p className="mt-2 text-[11px] text-white/45">{entry.reason}</p> : null}
                              </div>
                            ))}
                          </div>
                        </details>
                      );
                    })}
                  </div>
                ) : null}
              </div>
            ))}
          </div>
        </GlassCard>

        <div className="space-y-6">
          <GlassCard className="p-6">
            <div className="flex items-center justify-between">
              <div className="text-right">
                <p className="text-xs font-semibold text-white/40">Copy Engine</p>
                <h2 className="mt-2 text-2xl font-bold text-white">نسخ مع SHA-256</h2>
              </div>
              <Icon name="disk" className="h-5 w-5 text-electric" />
            </div>

            <div className="mt-5 space-y-3 text-right">
              <label className="block">
                <span className="mb-2 block text-xs font-semibold text-white/45">الوجهة</span>
                <input
                  value={copyTarget}
                  onChange={(event) => setCopyTarget(event.target.value)}
                  placeholder="D:\\Library\\Archive"
                  className="w-full rounded-2xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white outline-none placeholder:text-white/30"
                />
              </label>
              <label className="block">
                <span className="mb-2 block text-xs font-semibold text-white/45">المصادر، كل مسار في سطر</span>
                <textarea
                  value={copySources}
                  onChange={(event) => setCopySources(event.target.value)}
                  rows={5}
                  placeholder={"D:\\Media\\Movie.mkv\nD:\\Media\\Episode01.mkv"}
                  className="w-full rounded-2xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white outline-none placeholder:text-white/30"
                />
              </label>
              <button
                type="button"
                onClick={handleCopy}
                className="inline-flex items-center gap-2 rounded-2xl border border-electric/25 bg-electric/12 px-4 py-3 text-sm font-semibold text-electric"
              >
                <Icon name="settings" className="h-4 w-4" />
                بدء النسخ
              </button>
              <div className="rounded-2xl border border-white/10 bg-black/20 p-4 text-sm leading-7 text-white/65">
                الحالة: {copyState === "loading" ? "جارٍ النسخ" : copyState === "ready" ? `${copyResult?.items?.length || 0} ملف منسوخ` : copyState === "error" ? "تعذر النسخ" : "جاهز"}
              </div>
            </div>

            <div className="mt-5 max-h-[18rem] space-y-3 overflow-auto pr-1">
              {(copyResult?.items || []).slice(0, 6).map((item) => (
                <div key={`${item.source}-${item.target}`} className="rounded-2xl border border-white/10 bg-black/20 p-4 text-right">
                  <p className="text-xs font-semibold text-white/40">{item.checksum}</p>
                  <p className="mt-2 break-all text-sm font-semibold text-white">{item.target}</p>
                  <p className="mt-2 text-xs text-white/55">{formatBytes(item.bytes)}</p>
                </div>
              ))}
            </div>
          </GlassCard>

          <GlassCard className="p-6">
            <div className="flex items-center justify-between">
              <div className="text-right">
                <p className="text-xs font-semibold text-white/40">الخدمات</p>
                <h2 className="mt-2 text-2xl font-bold text-white">حالة المنصة</h2>
              </div>
              <div className="rounded-2xl border border-white/10 bg-black/20 p-3 text-electric">
                <Icon name="server" className="h-5 w-5" />
              </div>
            </div>

            <div className="mt-5 space-y-3">
              {services.map((service) => (
                <div key={service.label} className="flex items-center justify-between rounded-2xl border border-white/10 bg-black/20 px-4 py-3">
                  <span className={`text-sm font-semibold ${service.tone}`}>{service.value}</span>
                  <p className="font-semibold text-white">{service.label}</p>
                </div>
              ))}
            </div>
          </GlassCard>

          <GlassCard className="p-6">
            <div className="flex items-center justify-between">
              <div className="text-right">
                <p className="text-xs font-semibold text-white/40">الصف</p>
                <h2 className="mt-2 text-2xl font-bold text-white">مهام المرحلة الثالثة</h2>
              </div>
              <Icon name="settings" className="h-5 w-5 text-electric" />
            </div>

            <div className="mt-5 grid gap-3 md:grid-cols-2">
              {tasks.map((task, index) => (
                <div key={task} className="rounded-2xl border border-white/10 bg-white/[0.04] p-4 text-right">
                  <p className="text-xs font-semibold text-white/35">مهمة {index + 1}</p>
                  <p className="mt-2 text-sm font-semibold text-white">{task}</p>
                </div>
              ))}
            </div>
          </GlassCard>
        </div>
      </section>
    </div>
  );
}

function FilterPill({ label, value, items, onChange }) {
  return (
    <div className="rounded-2xl border border-white/10 bg-black/20 p-3 text-right">
      <p className="text-[11px] font-semibold text-white/40">{label}</p>
      <div className="mt-2 flex flex-wrap gap-2">
        {items.map((item) => (
          <button
            key={item}
            type="button"
            onClick={() => onChange(item)}
            className={`rounded-full border px-3 py-1 text-[11px] font-semibold transition ${
              value === item
                ? "border-electric/25 bg-electric/12 text-electric"
                : "border-white/10 bg-white/[0.04] text-white/65"
            }`}
          >
            {item}
          </button>
        ))}
      </div>
    </div>
  );
}

function StatCard({ label, value, tone = "text-white" }) {
  return (
    <div className="rounded-2xl border border-white/10 bg-black/20 p-4 text-right">
      <p className="text-xs font-semibold text-white/40">{label}</p>
      <p className={`mt-2 text-2xl font-black ${tone}`}>{value}</p>
    </div>
  );
}

function buildPreviewTree(entries) {
  const groups = new Map();

  for (const entry of entries) {
    const normalized = normalizeEntry(entry);
    const groupKey = normalizeKind(normalized.kind);
    const groupLabel = groupKey === "anime" ? "Anime" : groupKey === "series" ? "Series" : "Movies";
    const group = groups.get(groupKey) || { key: groupKey, label: groupLabel, kind: groupLabel, count: 0, folders: new Map() };
    group.count += 1;

    const folderPath = normalized.folderPath || "/";
    const folder = group.folders.get(folderPath) || {
      path: folderPath,
      name: normalized.folderName || folderPath,
      count: 0,
      entries: []
    };
    folder.count += 1;
    folder.entries.push(normalized);
    group.folders.set(folderPath, folder);
    groups.set(groupKey, group);
  }

  const orderedGroups = Array.from(groups.values()).map((group) => ({
    ...group,
    folders: Array.from(group.folders.values()).sort((a, b) => a.path.localeCompare(b.path))
  }));

  return {
    groups: orderedGroups.sort((a, b) => a.label.localeCompare(b.label))
  };
}

function normalizeEntry(entry) {
  const sourceName = basename(entry.source);
  const targetName = basename(entry.target);
  return {
    ...entry,
    kind: normalizeKind(entry.kind),
    sourceName,
    targetName,
    targetDir: dirname(entry.target),
    folderPath: dirname(entry.target),
    folderName: folderLabel(entry.target),
    action: entry.action || "move",
    reason: entry.reason || ""
  };
}

function normalizeKind(kind) {
  const lower = String(kind || "").toLowerCase();
  if (lower === "anime" || lower === "series" || lower === "movies") {
    return lower === "movies" ? "movies" : lower;
  }
  return "movies";
}

function actionTone(action) {
  if (action === "keep") {
    return "border-emerald-300/20 bg-emerald-300/10 text-emerald-200";
  }
  if (action === "rename") {
    return "border-fuchsia-300/20 bg-fuchsia-300/10 text-fuchsia-200";
  }
  if (action === "duplicate") {
    return "border-amber-300/20 bg-amber-300/10 text-amber-200";
  }
  return "border-cyan-300/20 bg-cyan-300/10 text-cyan-200";
}

function dirname(path) {
  return String(path || "").split(/[\\/]/).slice(0, -1).join("/") || "/";
}

function basename(path) {
  const parts = String(path || "").split(/[\\/]/);
  return parts[parts.length - 1] || path;
}

function folderLabel(path) {
  const parts = String(path || "").split(/[\\/]/).filter(Boolean);
  return parts.slice(-1)[0] || "/";
}

function formatBytes(bytes = 0) {
  if (!bytes) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = Number(bytes);
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  return `${value.toFixed(value >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
}
