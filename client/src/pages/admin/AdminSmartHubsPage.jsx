import React, { useEffect, useState } from "react";
import Icon from "../../components/Icon.jsx";
import {
  Card,
  Badge,
  Button,
  Modal,
  ConfirmModal,
  Input,
  Select,
  Textarea,
  ImagePickerInput,
} from "../../components/ui";
import {
  getAdminSmartHubs,
  createSmartHub,
  saveSmartHub,
  resolveAPIURL,
} from "../../lib/api.js";

const ACCENT_TONES = [
  { id: "violet", label: "بنفسجي ملكي (Violet)", color: "#8B5CF6" },
  { id: "amber", label: "كهرماني دافئ (Amber)", color: "#F59E0B" },
  { id: "cyan", label: "سماوي مضيء (Cyan)", color: "#06B6D4" },
  { id: "rose", label: "وردي نيون (Rose)", color: "#F43F5E" },
  { id: "emerald", label: "أخضر زمردي (Emerald)", color: "#10B981" },
  { id: "fuchsia", label: "فوشيا متوهج (Fuchsia)", color: "#D946EF" },
];

const SCOPE_OPTIONS = [
  { id: "all", label: "كامل المكتبة (الكل)" },
  { id: "movies", label: "الأفلام فقط" },
  { id: "series", label: "المسلسلات فقط" },
  { id: "anime", label: "الأنمي فقط" },
];

const INITIAL_HUB_FORM = {
  isNew: true,
  slug: "",
  scope: "all",
  title_ar: "",
  title_en: "",
  description_ar: "",
  artwork_path: "",
  artwork_position: "center center",
  accent: "violet",
  icon: "spark",
  priority: 0,
  min_item_count: 3,
  is_active: true,
  rule: {
    types: [],
    categories: [],
    tags_any: [],
    year_from: 0,
    rating_gte: 0,
  },
};

export default function AdminSmartHubsPage() {
  const [hubs, setHubs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [editModal, setEditModal] = useState(null);
  const [saving, setSaving] = useState(false);
  const [deletingHub, setDeletingHub] = useState(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  useEffect(() => {
    loadHubs();
  }, []);

  async function loadHubs() {
    setLoading(true);
    try {
      const data = await getAdminSmartHubs();
      setHubs(data.hubs || []);
    } catch {
      setHubs([]);
    } finally {
      setLoading(false);
    }
  }

  function handleOpenAdd() {
    setEditModal({ ...INITIAL_HUB_FORM });
  }

  function handleOpenEdit(hub) {
    setEditModal({
      ...hub,
      isNew: false,
      rule: {
        types: hub.rule?.types || [],
        categories: hub.rule?.categories || [],
        tags_any: hub.rule?.tags_any || [],
        year_from: hub.rule?.year_from || 0,
        rating_gte: hub.rule?.rating_gte || 0,
      },
    });
  }

  function updateField(key, value) {
    setEditModal((prev) => ({ ...prev, [key]: value }));
  }

  function updateRuleField(key, value) {
    setEditModal((prev) => ({
      ...prev,
      rule: { ...(prev.rule || {}), [key]: value },
    }));
  }

  const parseList = (str) =>
    str
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean);

  async function handleSaveHub() {
    if (!editModal.title_ar?.trim() || !editModal.slug?.trim()) {
      alert("يرجى إدخال عنوان المحور بالعربي والمعرّف اللطيف (Slug)");
      return;
    }

    setSaving(true);
    try {
      const payload = {
        ...editModal,
        priority: Number(editModal.priority) || 0,
        min_item_count: Number(editModal.min_item_count) || 1,
        rule: {
          types: editModal.rule?.types || [],
          categories: editModal.rule?.categories || [],
          tags_any: editModal.rule?.tags_any || [],
          year_from: Number(editModal.rule?.year_from) || 0,
          rating_gte: Number(editModal.rule?.rating_gte) || 0,
        },
      };

      if (editModal.isNew) {
        await createSmartHub(payload);
      } else {
        await saveSmartHub(editModal.slug, payload);
      }
      setEditModal(null);
      loadHubs();
    } catch (err) {
      alert(`تعذر حفظ المحور: ${err.message || ""}`);
    } finally {
      setSaving(false);
    }
  }

  async function handleConfirmDelete() {
    if (!deletingHub) return;
    setDeleteLoading(true);
    try {
      await saveSmartHub(deletingHub.slug, { ...deletingHub, is_active: false });
      setDeletingHub(null);
      loadHubs();
    } catch (err) {
      alert(`تعذر حذف المحور: ${err.message || ""}`);
    } finally {
      setDeleteLoading(false);
    }
  }

  return (
    <div className="space-y-6 text-right animate-fadeIn" dir="rtl">
      {/* Header Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-5 rounded-3xl border border-white/10 bg-[#0C0A18]/80 backdrop-blur-2xl shadow-xl">
        <div>
          <div className="flex items-center gap-2">
            <span className="flex h-8 w-8 items-center justify-center rounded-xl bg-fuchsia-500/20 text-fuchsia-400">
              <Icon name="grid" className="h-4 w-4" />
            </span>
            <h1 className="text-xl sm:text-2xl font-black text-white">
              المحاور الذكية (Smart Hubs)
            </h1>
          </div>
          <p className="text-xs text-white/55 mt-1 max-w-xl">
            مجموعات سينمائية ديناميكية مبنية وفق قواعد تلقائية من بيانات مكتبتك المحلية، مع تحكم كامل بالهوية والأغلفة.
          </p>
        </div>

        <Button
          variant="primary"
          onClick={handleOpenAdd}
          icon={<Icon name="plus" className="h-4 w-4" />}
          className="rounded-2xl shrink-0 shadow-neon"
        >
          <span>إضافة محور ذكي جديد</span>
        </Button>
      </div>

      {/* Hubs Grid Cards */}
      {loading ? (
        <div className="p-20 text-center text-white/60">
          <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-fuchsia-500 border-t-transparent" />
          <p className="mt-3 text-xs font-bold">جارٍ تحميل المحاور الذكية...</p>
        </div>
      ) : hubs.length === 0 ? (
        <Card className="p-16 text-center space-y-3">
          <p className="text-4xl">✨</p>
          <h3 className="text-lg font-bold text-white">لا توجد محاور ذكية مضافة حالياً</h3>
          <p className="text-xs text-white/50 max-w-md mx-auto">
            قم بإنشاء محاور جديدة مثل (روائع الدراما التركية، كلاسيكيات ديزني، أو سينما الجريمة والغموض).
          </p>
          <Button variant="primary" onClick={handleOpenAdd} className="mt-2">
            ➕ إضافة أول محور ذكي
          </Button>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {hubs.map((hub) => {
            const artwork = resolveAPIURL(hub.artwork_path) || "/nexora-library-backdrop.PNG";
            return (
              <div
                key={hub.slug}
                className="group relative flex flex-col justify-between overflow-hidden rounded-3xl border border-white/10 bg-[#0E0C1A] hover:border-fuchsia-500/50 hover:shadow-2xl hover:shadow-purple-950/60 transition-all duration-300"
              >
                {/* Artwork Thumbnail & Badges Header */}
                <div className="relative h-32 w-full overflow-hidden bg-black/60">
                  <img
                    src={artwork}
                    alt=""
                    className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-105 opacity-60"
                    style={{ objectPosition: hub.artwork_position || "center center" }}
                    onError={(e) => {
                      e.target.src = "/nexora-library-backdrop.PNG";
                    }}
                  />
                  <div className="absolute inset-0 bg-gradient-to-t from-[#0E0C1A] via-[#0E0C1A]/60 to-transparent" />

                  {/* Top Floating Badges */}
                  <div className="absolute top-3 left-3 right-3 flex items-center justify-between gap-2">
                    <span
                      className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[10px] font-bold backdrop-blur-md border ${
                        hub.is_active
                          ? "bg-emerald-500/20 text-emerald-300 border-emerald-500/30"
                          : "bg-white/10 text-white/50 border-white/10"
                      }`}
                    >
                      <span
                        className={`h-1.5 w-1.5 rounded-full ${
                          hub.is_active ? "bg-emerald-400 animate-pulse" : "bg-white/40"
                        }`}
                      />
                      {hub.is_active ? "ظاهر في الواجهة" : "مخفي"}
                    </span>

                    <span className="rounded-full bg-black/60 px-2.5 py-0.5 text-[10px] font-mono font-bold text-fuchsia-300 border border-white/10 backdrop-blur-md">
                      slug: {hub.slug}
                    </span>
                  </div>
                </div>

                {/* Card Content Body */}
                <div className="p-5 space-y-3 flex-1 flex flex-col justify-between">
                  <div>
                    <div className="flex items-center justify-between">
                      <h3 className="text-lg font-black text-white group-hover:text-fuchsia-300 transition-colors">
                        {hub.title_ar}
                      </h3>
                      <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-white/5 text-white/70">
                        <Icon name={hub.icon || "spark"} className="h-3.5 w-3.5" />
                      </span>
                    </div>

                    {hub.title_en && (
                      <p className="text-xs text-white/40 font-mono mt-0.5" dir="ltr">
                        {hub.title_en}
                      </p>
                    )}

                    {hub.description_ar && (
                      <p className="text-xs text-white/60 line-clamp-2 mt-2 leading-relaxed">
                        {hub.description_ar}
                      </p>
                    )}
                  </div>

                  {/* Meta Specs & Rules */}
                  <div className="grid grid-cols-3 gap-2 text-center text-[10px] pt-3 border-t border-white/5">
                    <div className="p-2 rounded-xl bg-black/40 border border-white/5">
                      <p className="font-mono font-bold text-white">{hub.scope || "الكل"}</p>
                      <p className="text-white/40 mt-0.5">النطاق</p>
                    </div>
                    <div className="p-2 rounded-xl bg-black/40 border border-white/5">
                      <p className="font-mono font-bold text-fuchsia-300">
                        {hub.min_item_count || 3}
                      </p>
                      <p className="text-white/40 mt-0.5">حد أدنى</p>
                    </div>
                    <div className="p-2 rounded-xl bg-black/40 border border-white/5">
                      <p className="font-mono font-bold text-amber-300">{hub.priority || 0}</p>
                      <p className="text-white/40 mt-0.5">الأولوية</p>
                    </div>
                  </div>

                  {/* Actions Footer */}
                  <div className="flex items-center justify-between pt-3 border-t border-white/10">
                    <button
                      type="button"
                      onClick={() => (window.location.hash = `#/hub/${hub.slug}`)}
                      className="text-xs font-bold text-fuchsia-400 hover:text-fuchsia-300 hover:underline transition"
                    >
                      معاينة المحور ↵
                    </button>

                    <div className="flex items-center gap-1.5">
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => handleOpenEdit(hub)}
                        title="تعديل المحور"
                      >
                        <Icon name="settings" className="h-3.5 w-3.5" />
                        <span>تعديل</span>
                      </Button>
                      <Button
                        size="sm"
                        variant="danger"
                        onClick={() => setDeletingHub(hub)}
                        title="حذف المحور"
                      >
                        <Icon name="close" className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Add / Edit Smart Hub Modal */}
      {editModal && (
        <Modal
          isOpen={Boolean(editModal)}
          onClose={() => setEditModal(null)}
          title={editModal.isNew ? "إنشاء محور ذكي جديد" : `تعديل محور: ${editModal.title_ar}`}
          subtitle="تحديد قواعد المطابقة، النطاق، وهوية العرض التلقائي"
          size="xl"
          className="admin-form-modal sm:max-w-5xl"
          actions={
            <>
              <Button variant="ghost" onClick={() => setEditModal(null)}>
                إلغاء
              </Button>
              <Button
                variant="primary"
                onClick={handleSaveHub}
                loading={saving}
                className="px-6"
              >
                💾 {saving ? "جارٍ الحفظ..." : "حفظ بيانات المحور"}
              </Button>
            </>
          }
        >
          <div className="space-y-6">
            {/* Section 1: Basic Identity */}
            <div className="space-y-3">
              <h4 className="text-xs font-black text-fuchsia-400 uppercase tracking-wider flex items-center gap-1.5">
                <Icon name="info" className="h-4 w-4" /><span>1. هوية المحور الأساسية</span>
              </h4>
              <div className="grid gap-4 sm:grid-cols-2">
                <Input
                  label="العنوان باللغة العربية"
                  value={editModal.title_ar || ""}
                  onChange={(e) => updateField("title_ar", e.target.value)}
                  placeholder="مثال: روائع الدراما التركية"
                  required
                />
                <Input
                  label="العنوان باللغة الإنجليزية"
                  value={editModal.title_en || ""}
                  onChange={(e) => updateField("title_en", e.target.value)}
                  placeholder="e.g. Turkish Drama Classics"
                  dir="ltr"
                  mono
                />
                <Input
                  label="المعرّف اللطيف (Slug - إنجليزي بدون مسافات)"
                  value={editModal.slug || ""}
                  onChange={(e) => updateField("slug", e.target.value)}
                  placeholder="e.g. turkish-drama"
                  disabled={!editModal.isNew}
                  dir="ltr"
                  mono
                  required
                />
                <Select
                  label="نطاق البحث والفهرسة"
                  value={editModal.scope || "all"}
                  onChange={(e) => updateField("scope", e.target.value)}
                  options={SCOPE_OPTIONS}
                />
              </div>
            </div>

            {/* Section 2: Artwork & Image (With Dual Upload + Path Support) */}
            <div className="space-y-3 pt-2 border-t border-white/10">
              <h4 className="text-xs font-black text-fuchsia-400 uppercase tracking-wider flex items-center gap-1.5">
                <Icon name="image" className="h-4 w-4" /><span>2. صورة وغلاف المحور (Artwork)</span>
              </h4>
              <div className="grid gap-4 sm:grid-cols-2 items-start">
                <ImagePickerInput
                  label="صورة البانر والغلاف"
                  value={editModal.artwork_path || ""}
                  onChange={(val) => updateField("artwork_path", val)}
                  aspectRatio="banner"
                  placeholder="https://... أو /artworks/hub.jpg"
                  helperText="اختر صورة من جهازك أو اكتب مسار الصورة مباشرة"
                />
                <div className="space-y-4">
                  <Select
                    label="لون التمييز (Accent Glow)"
                    value={editModal.accent || "violet"}
                    onChange={(e) => updateField("accent", e.target.value)}
                    options={ACCENT_TONES.map((t) => ({ value: t.id, label: t.label }))}
                  />
                  <Input
                    label="موضع تركيز الصورة (Position)"
                    value={editModal.artwork_position || "center center"}
                    onChange={(e) => updateField("artwork_position", e.target.value)}
                    placeholder="center center أو top center"
                    dir="ltr"
                    mono
                  />
                  <div className="grid grid-cols-2 gap-3">
                    <Input
                      label="أولوية الترتيب"
                      type="number"
                      value={editModal.priority ?? 0}
                      onChange={(e) => updateField("priority", e.target.value)}
                    />
                    <Input
                      label="الحد الأدنى لظهور المحور"
                      type="number"
                      value={editModal.min_item_count ?? 3}
                      onChange={(e) => updateField("min_item_count", e.target.value)}
                    />
                  </div>
                </div>
              </div>
            </div>

            {/* Section 3: Description */}
            <div className="space-y-3 pt-2 border-t border-white/10">
              <Textarea
                label="الوصف العربي للمحور"
                value={editModal.description_ar || ""}
                onChange={(e) => updateField("description_ar", e.target.value)}
                placeholder="نبذة تعريفية مشوقة عن نوعية الأعمال المختارة في هذا المحور..."
                rows={2}
              />
            </div>

            {/* Section 4: Smart Matching Rules */}
            <div className="space-y-3 pt-2 border-t border-white/10">
              <h4 className="text-xs font-black text-fuchsia-400 uppercase tracking-wider flex items-center gap-1.5">
                <Icon name="database" className="h-4 w-4" /><span>3. قواعد التجميع والتصنيف التلقائي</span>
              </h4>
              <div className="grid gap-4 sm:grid-cols-2">
                <Input
                  label="أنواع الأعمال (مفصولة بفاصلة: movie, series, anime)"
                  value={(editModal.rule?.types || []).join(", ")}
                  onChange={(e) => updateRuleField("types", parseList(e.target.value))}
                  placeholder="movie, series, anime"
                  dir="ltr"
                  mono
                />
                <Input
                  label="الوسوم والتصنيفات المطلوبة (مفصولة بفاصلة)"
                  value={(editModal.rule?.tags_any || []).join(", ")}
                  onChange={(e) => updateRuleField("tags_any", parseList(e.target.value))}
                  placeholder="أكشن, تركي, عائلي, إثارة"
                />
                <Input
                  label="من سنة إنتاج (سنة البداية)"
                  type="number"
                  value={editModal.rule?.year_from || ""}
                  onChange={(e) => updateRuleField("year_from", e.target.value)}
                  placeholder="2020"
                />
                <Input
                  label="الحد الأدنى للتقييم (من 10)"
                  type="number"
                  step="0.1"
                  min="0"
                  max="10"
                  value={editModal.rule?.rating_gte || ""}
                  onChange={(e) => updateRuleField("rating_gte", e.target.value)}
                  placeholder="7.5"
                />
              </div>
            </div>

            {/* Section 5: Active Status Switch */}
            <div className="flex items-center gap-3 p-3 rounded-2xl bg-white/5 border border-white/10">
              <input
                type="checkbox"
                id="hub_is_active"
                checked={editModal.is_active}
                onChange={(e) => updateField("is_active", e.target.checked)}
                className="h-5 w-5 rounded-lg border-white/20 text-fuchsia-600 focus:ring-fuchsia-500 cursor-pointer"
              />
              <label htmlFor="hub_is_active" className="text-xs font-bold text-white cursor-pointer select-none">
                تفعيل وإظهار المحور الذكي للمشاهدين في الصفحة الرئيسية والأقسام
              </label>
            </div>
          </div>
        </Modal>
      )}

      {/* Delete Confirmation Modal */}
      {deletingHub && (
        <ConfirmModal
          isOpen={Boolean(deletingHub)}
          onClose={() => setDeletingHub(null)}
          onConfirm={handleConfirmDelete}
          loading={deleteLoading}
          title="تأكيد حذف المحور الذكي"
          message={`هل أنت متأكد من رغبتك في إزالة المحور الذكي "${deletingHub.title_ar}"؟ لن يتم حذف أي ملفات من المكتبة.`}
          confirmText="نعم، حذف المحور"
          cancelText="إلغاء"
        />
      )}
    </div>
  );
}
