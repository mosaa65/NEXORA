import React, { useEffect, useState } from "react";
import Icon from "../../components/Icon.jsx";
import {
  Card,
  Badge,
  Button,
  Modal,
  ConfirmModal,
  Input,
} from "../../components/ui";
import {
  getCategories,
  createCategory,
  updateCategory,
  deleteCategory,
} from "../../lib/api.js";
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
  const [deleteLoading, setDeleteLoading] = useState(false);

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
      form: {
        id: cat.id,
        name_ar: cat.name_ar || cat.nameAr || "",
        name_en: cat.name_en || cat.nameEn || "",
        slug: cat.slug || "",
      },
    });
  }

  async function handleSaveCategory() {
    if (!categoryModal?.form?.name_ar?.trim() || !categoryModal?.form?.slug?.trim()) {
      alert("يرجى إدخال اسم التصنيف بالعربي والاسم اللطيف (Slug)");
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
    setDeleteLoading(true);
    try {
      await deleteCategory(deletingCategory.id || deletingCategory.slug);
      setDeletingCategory(null);
      loadCategoriesData();
    } catch (err) {
      alert("تعذر حذف التصنيف: " + (err?.message || ""));
    } finally {
      setDeleteLoading(false);
    }
  }

  return (
    <div className="space-y-6 text-right animate-fadeIn" dir="rtl">
      {/* Header & Add Button */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-5 rounded-3xl border border-white/10 bg-[#0C0A18]/80 backdrop-blur-2xl shadow-xl">
        <div>
          <div className="flex items-center gap-2">
            <span className="flex h-8 w-8 items-center justify-center rounded-xl bg-teal-500/20 text-teal-400">
              <Icon name="mask" className="h-4 w-4" />
            </span>
            <h1 className="text-xl sm:text-2xl font-black text-white">
              الأقسام والتصنيفات (Categories)
            </h1>
          </div>
          <p className="text-xs text-white/55 mt-1 max-w-xl">
            إدارة الأقسام الرئيسية في المكتبة السينمائية مع متابعة إجمالي الأعمال والملفات المفهرسة في كل قسم.
          </p>
        </div>

        <Button
          variant="primary"
          onClick={handleOpenAddCategory}
          className="rounded-2xl shrink-0 shadow-neon"
        >
          <span>➕ إضافة تصنيف جديد</span>
        </Button>
      </div>

      {/* Categories Grid */}
      {categoriesLoading ? (
        <div className="p-20 text-center text-white/60">
          <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-fuchsia-500 border-t-transparent" />
          <p className="mt-3 text-xs font-bold">جارٍ تحميل التصنيفات...</p>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {categories.map((cat) => {
            const count = cat.media_count || cat.mediaCount || 0;
            const fileCount = cat.file_count || cat.fileCount || 0;

            return (
              <div
                key={cat.id || cat.slug}
                className="group relative flex flex-col justify-between p-5 rounded-3xl border border-white/10 bg-[#0E0C1A] hover:border-fuchsia-500/50 hover:shadow-2xl hover:shadow-purple-950/50 transition-all duration-300"
              >
                <div>
                  {/* Top Row: Icon & Slug */}
                  <div className="flex items-center justify-between">
                    <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-purple-800 to-fuchsia-900 text-white shadow-md">
                      <Icon name={cat.icon || "film"} className="h-5 w-5 text-fuchsia-300" />
                    </div>
                    <span className="text-[10px] font-mono font-bold text-fuchsia-400 bg-fuchsia-500/10 px-2.5 py-1 rounded-full border border-fuchsia-500/20">
                      slug: {cat.slug}
                    </span>
                  </div>

                  {/* Name & English */}
                  <h3 className="mt-4 text-lg font-black text-white group-hover:text-fuchsia-300 transition-colors">
                    {cat.name_ar || cat.nameAr}
                  </h3>
                  <p className="text-xs text-white/40 font-mono mt-0.5" dir="ltr">
                    {cat.name_en || cat.nameEn || cat.slug}
                  </p>

                  {/* Statistics Counters */}
                  <div className="mt-4 grid grid-cols-2 gap-2 text-center">
                    <div className="p-2.5 rounded-2xl bg-black/40 border border-white/5">
                      <p className="text-base font-black text-white">{count}</p>
                      <p className="text-[10px] text-white/50">عمل مفهرس</p>
                    </div>
                    <div className="p-2.5 rounded-2xl bg-black/40 border border-white/5">
                      <p className="text-base font-black text-emerald-400">{fileCount}</p>
                      <p className="text-[10px] text-white/50">ملف فيديو</p>
                    </div>
                  </div>
                </div>

                {/* Actions Footer */}
                <div className="mt-5 flex items-center justify-between border-t border-white/10 pt-3 text-xs">
                  <button
                    type="button"
                    onClick={() => onNavigateToMedia?.(cat.slug)}
                    className="text-fuchsia-400 font-bold hover:text-fuchsia-300 hover:underline transition"
                  >
                    تصفح الأعمال ↵
                  </button>

                  <div className="flex items-center gap-1.5">
                    <Button
                      size="sm"
                      variant="secondary"
                      onClick={() => handleOpenEditCategory(cat)}
                      title="تعديل التصنيف"
                    >
                      <Icon name="settings" className="h-3.5 w-3.5" />
                    </Button>
                    <Button
                      size="sm"
                      variant="danger"
                      onClick={() => setDeletingCategory(cat)}
                      title="حذف التصنيف"
                    >
                      <Icon name="close" className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Category Add/Edit Modal */}
      {categoryModal && (
        <Modal
          isOpen={Boolean(categoryModal)}
          onClose={() => setCategoryModal(null)}
          title={
            categoryModal.isNew
              ? "➕ إضافة تصنيف أو قسم جديد"
              : "✏️ تعديل بيانات التصنيف"
          }
          subtitle="تحديد الاسم العربي والإنجليزي والمعرّف اللطيف للقسم"
          size="md"
          actions={
            <>
              <Button variant="ghost" onClick={() => setCategoryModal(null)}>
                إلغاء
              </Button>
              <Button
                variant="primary"
                onClick={handleSaveCategory}
                loading={categorySaveLoading}
                className="px-6"
              >
                💾 {categorySaveLoading ? "جارٍ الحفظ..." : "حفظ التصنيف"}
              </Button>
            </>
          }
        >
          <div className="space-y-4">
            <Input
              label="الاسم باللغة العربية"
              value={categoryModal.form.name_ar}
              onChange={(e) =>
                setCategoryModal({
                  ...categoryModal,
                  form: { ...categoryModal.form, name_ar: e.target.value },
                })
              }
              placeholder="مثال: مسلسلات تركية أو وثائقيات علمية"
              required
            />

            <Input
              label="الاسم باللغة الإنجليزية"
              value={categoryModal.form.name_en}
              onChange={(e) =>
                setCategoryModal({
                  ...categoryModal,
                  form: { ...categoryModal.form, name_en: e.target.value },
                })
              }
              placeholder="e.g. Turkish Series"
              dir="ltr"
              mono
            />

            <Input
              label="الاسم اللطيف (Slug - إنجليزي فقط بدون مسافات)"
              value={categoryModal.form.slug}
              onChange={(e) =>
                setCategoryModal({
                  ...categoryModal,
                  form: { ...categoryModal.form, slug: e.target.value },
                })
              }
              placeholder="e.g. turkish-series"
              dir="ltr"
              mono
              required
            />
          </div>
        </Modal>
      )}

      {/* Category Delete Confirmation Modal */}
      {deletingCategory && (
        <ConfirmModal
          isOpen={Boolean(deletingCategory)}
          onClose={() => setDeletingCategory(null)}
          onConfirm={confirmDeleteCategory}
          loading={deleteLoading}
          title="تأكيد حذف التصنيف"
          message={`هل أنت متأكد من رغبتك في حذف تصنيف "${
            deletingCategory.name_ar || deletingCategory.nameAr
          }"؟`}
          confirmText="نعم، حذف التصنيف"
          cancelText="إلغاء"
        />
      )}
    </div>
  );
}
