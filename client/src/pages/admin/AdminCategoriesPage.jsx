import { useEffect, useState } from "react";
import GlassCard from "../../components/GlassCard.jsx";
import Icon from "../../components/Icon.jsx";
import { getCategories, createCategory, updateCategory, deleteCategory } from "../../lib/api.js";
import { DEFAULT_CATEGORIES } from "./adminConstants.js";

/**
 * AdminCategoriesPage — إدارة الأقسام والتصنيفات
 * Route: /admin/categories
 */
export default function AdminCategoriesPage({ onNavigateToMedia }) {
  const [categories, setCategories] = useState(DEFAULT_CATEGORIES);
  const [categoriesLoading, setCategoriesLoading] = useState(false);
  const [categoryModal, setCategoryModal] = useState(null);
  const [categorySaveLoading, setCategorySaveLoading] = useState(false);
  const [deletingCategory, setDeletingCategory] = useState(null);

  useEffect(() => {
    loadCategoriesData();
  }, []);

  async function loadCategoriesData() {
    setCategoriesLoading(true);
    try {
      const data = await getCategories();
      if (data.categories && data.categories.length > 0) {
        setCategories(data.categories);
      }
    } catch {
      // Use defaults
    } finally {
      setCategoriesLoading(false);
    }
  }

  function handleOpenAddCategory() {
    setCategoryModal({
      isNew: true,
      form: { name_ar: "", name_en: "", slug: "" },
    });
  }

  function handleOpenEditCategory(cat) {
    setCategoryModal({
      isNew: false,
      form: { id: cat.id, name_ar: cat.name_ar, name_en: cat.name_en, slug: cat.slug },
    });
  }

  async function handleSaveCategory() {
    if (!categoryModal?.form?.name_ar?.trim() || !categoryModal?.form?.slug?.trim()) {
      alert("يرجى إدخال اسم التصنيف بالعربي والاسم اللطيف (slug)");
      return;
    }
    setCategorySaveLoading(true);
    try {
      if (categoryModal.isNew) {
        await createCategory(categoryModal.form);
      } else {
        await updateCategory(categoryModal.form.id, categoryModal.form);
      }
      setCategoryModal(null);
      loadCategoriesData();
    } catch (err) {
      alert("تعذر حفظ التصنيف: " + (err?.message || ""));
    } finally {
      setCategorySaveLoading(false);
    }
  }

  async function confirmDeleteCategory() {
    if (!deletingCategory) return;
    try {
      await deleteCategory(deletingCategory.id);
      setDeletingCategory(null);
      loadCategoriesData();
    } catch (err) {
      alert("تعذر حذف التصنيف: " + (err?.message || ""));
    }
  }

  return (
    <div className="space-y-6 text-right animate-fadeIn" dir="rtl">
      {/* Header & Add Button */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-black text-white flex items-center gap-2">
            <span>الأقسام والتصنيفات المتاحة في النظام</span>
          </h2>
          <p className="text-xs text-white/50 mt-1">
            يمكنك إضافة تصنيفات جديدة (مثل مسلسلات تركية، كورية، وثائقيات علمية، برامج تلفزيونية) أو تعديلها.
          </p>
        </div>

        <button
          type="button"
          onClick={handleOpenAddCategory}
          className="flex items-center gap-2 rounded-2xl bg-gradient-to-r from-emerald-600 via-teal-600 to-emerald-700 px-5 py-3 text-xs font-bold text-white shadow-lg shadow-emerald-900/40 hover:brightness-110 transition"
        >
          <span>➕ إضافة تصنيف أو قسم جديد</span>
        </button>
      </div>

      {/* Categories Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {categories.map((cat) => {
          const count = cat.media_count || cat.mediaCount || 0;
          const fileCount = cat.file_count || cat.fileCount || 0;

          return (
            <div
              key={cat.id || cat.slug}
              className="group relative flex flex-col justify-between p-5 rounded-3xl border border-white/10 bg-[#0E0C1A] hover:border-fuchsia-500/50 hover:shadow-xl hover:shadow-purple-950/50 transition duration-300"
            >
              <div>
                {/* Top Row */}
                <div className="flex items-center justify-between">
                  <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br from-purple-800 to-fuchsia-900 text-white shadow-md">
                    <Icon name={cat.icon || "film"} className="h-6 w-6 text-fuchsia-300" />
                  </div>
                  <span className="text-[11px] font-mono font-bold text-fuchsia-400 bg-fuchsia-500/10 px-2.5 py-1 rounded-xl">
                    slug: {cat.slug}
                  </span>
                </div>

                {/* Name & English */}
                <h3 className="mt-4 text-xl font-black text-white group-hover:text-fuchsia-300 transition">
                  {cat.name_ar || cat.nameAr}
                </h3>
                <p className="text-xs text-white/40 font-mono mt-0.5">
                  {cat.name_en || cat.nameEn || cat.slug}
                </p>

                {/* Statistics */}
                <div className="mt-4 grid grid-cols-2 gap-2 text-center">
                  <div className="p-2.5 rounded-xl bg-black/30 border border-white/5">
                    <p className="text-base font-black text-white">{count}</p>
                    <p className="text-[10px] text-white/50">عمل / فيلم</p>
                  </div>
                  <div className="p-2.5 rounded-xl bg-black/30 border border-white/5">
                    <p className="text-base font-black text-emerald-400">{fileCount}</p>
                    <p className="text-[10px] text-white/50">ملف فيديو</p>
                  </div>
                </div>
              </div>

              {/* Actions */}
              <div className="mt-5 flex items-center justify-between border-t border-white/10 pt-3 text-xs">
                <button
                  type="button"
                  onClick={() => onNavigateToMedia?.(cat.slug)}
                  className="text-fuchsia-400 font-bold hover:underline"
                >
                  تصفح أعمال القسم ↵
                </button>

                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => handleOpenEditCategory(cat)}
                    className="px-2.5 py-1 rounded-lg bg-white/10 text-white/80 hover:bg-white/20 hover:text-white"
                    title="تعديل التصنيف"
                  >
                    ✏️
                  </button>
                  <button
                    type="button"
                    onClick={() => setDeletingCategory(cat)}
                    className="px-2.5 py-1 rounded-lg bg-rose-500/20 text-rose-300 hover:bg-rose-500/40"
                    title="حذف التصنيف"
                  >
                    🗑️
                  </button>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {/* Category Add/Edit Modal */}
      {categoryModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/85 p-4 backdrop-blur-md animate-fadeIn">
          <div className="w-full max-w-lg rounded-3xl border border-fuchsia-500/30 bg-[#0D0B18] p-6 sm:p-8 space-y-5 shadow-2xl">
            <div className="flex items-center justify-between border-b border-white/10 pb-3">
              <h3 className="text-xl font-bold text-white">
                {categoryModal.isNew ? "➕ إضافة تصنيف أو قسم جديد" : "✏️ تعديل بيانات التصنيف"}
              </h3>
              <button
                type="button"
                onClick={() => setCategoryModal(null)}
                className="rounded-xl border border-white/10 p-1.5 text-white/60 hover:text-white"
              >
                ✕
              </button>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-xs font-bold text-white/70 mb-1">الاسم باللغة العربية</label>
                <input
                  type="text"
                  value={categoryModal.form.name_ar}
                  onChange={(e) => setCategoryModal({ ...categoryModal, form: { ...categoryModal.form, name_ar: e.target.value } })}
                  placeholder="مثال: مسلسلات تركية أو وثائقيات علمية"
                  className="w-full rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-sm text-white outline-none focus:border-fuchsia-500"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-white/70 mb-1">الاسم باللغة الإنجليزية</label>
                <input
                  type="text"
                  value={categoryModal.form.name_en}
                  onChange={(e) => setCategoryModal({ ...categoryModal, form: { ...categoryModal.form, name_en: e.target.value } })}
                  placeholder="e.g. Turkish Series"
                  dir="ltr"
                  className="w-full rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-sm text-white outline-none focus:border-fuchsia-500 font-mono"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-white/70 mb-1">الاسم اللطيف (Slug - إنجليزي فقط بدون مسافات)</label>
                <input
                  type="text"
                  value={categoryModal.form.slug}
                  onChange={(e) => setCategoryModal({ ...categoryModal, form: { ...categoryModal.form, slug: e.target.value } })}
                  placeholder="e.g. turkish-series"
                  dir="ltr"
                  className="w-full rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-sm text-white outline-none focus:border-fuchsia-500 font-mono"
                />
              </div>
            </div>

            <div className="flex items-center justify-end gap-3 border-t border-white/10 pt-4">
              <button
                type="button"
                onClick={() => setCategoryModal(null)}
                className="px-4 py-2 rounded-xl border border-white/10 bg-white/[0.05] text-xs font-bold text-white"
              >
                إلغاء
              </button>
              <button
                type="button"
                onClick={handleSaveCategory}
                disabled={categorySaveLoading}
                className="px-6 py-2 rounded-xl bg-gradient-to-r from-purple-600 to-fuchsia-600 text-xs font-bold text-white shadow-lg shadow-purple-900/50 hover:brightness-110 disabled:opacity-50"
              >
                {categorySaveLoading ? "جارٍ الحفظ..." : "💾 حفظ التصنيف"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Category Delete Confirmation */}
      {deletingCategory && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/85 p-4 backdrop-blur-md animate-fadeIn">
          <div className="w-full max-w-md rounded-3xl border border-rose-500/30 bg-[#0D0B18] p-6 text-center space-y-4 shadow-2xl">
            <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-rose-500/20 text-2xl text-rose-400">
              🗑️
            </div>
            <h3 className="text-xl font-bold text-white">تأكيد حذف التصنيف</h3>
            <p className="text-xs text-white/70">
              هل أنت متأكد من حذف تصنيف <span className="font-bold text-rose-300">"{deletingCategory.name_ar || deletingCategory.nameAr}"</span>؟
            </p>
            <div className="flex items-center justify-center gap-3 pt-2">
              <button
                type="button"
                onClick={() => setDeletingCategory(null)}
                className="px-5 py-2.5 rounded-xl border border-white/10 bg-white/[0.05] text-xs font-bold text-white"
              >
                إلغاء
              </button>
              <button
                type="button"
                onClick={confirmDeleteCategory}
                className="px-6 py-2.5 rounded-xl bg-rose-600 hover:bg-rose-700 text-xs font-bold text-white shadow-lg shadow-rose-900/50"
              >
                نعم، احذف
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
