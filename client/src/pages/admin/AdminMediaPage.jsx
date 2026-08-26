import React, { useEffect, useMemo, useState } from "react";
import Icon from "../../components/Icon.jsx";
import MediaCollection from "../../components/MediaCollection.jsx";
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
  classifyOriginsFromFolders,
  createMediaItem,
  deleteMediaItem,
  enrichMedia,
  getCategories,
  getMediaList,
  updateMediaItem,
} from "../../lib/api.js";
import {
  ALL_GENRES,
  COUNTRIES_LIST,
  COUNTRY_TAGS,
  DEFAULT_CATEGORIES,
  QUALITIES_LIST,
  RATINGS_LIST,
  YEARS_LIST,
} from "./adminConstants.js";

/**
 * AdminMediaPage — إدارة وتحرير الأعمال والوسائط (CMS)
 * Route: /admin/media
 */
export default function AdminMediaPage() {
  // Categories
  const [categories, setCategories] = useState(DEFAULT_CATEGORIES);
  const [selectedCategory, setSelectedCategory] = useState("all");
  const [selectedGenre, setSelectedGenre] = useState("all");
  const [selectedCountry, setSelectedCountry] = useState("all");
  const [selectedQuality, setSelectedQuality] = useState("all");
  const [selectedYear, setSelectedYear] = useState("all");
  const [selectedRating, setSelectedRating] = useState("all");
  const [managerSort, setManagerSort] = useState("latest");
  const [managerSearch, setManagerSearch] = useState("");

  // Media
  const [mediaItems, setMediaItems] = useState([]);
  const [mediaLoading, setMediaLoading] = useState(false);
  const [totalMediaCount, setTotalMediaCount] = useState(0);

  // Edit/Add Modal
  const [editingItem, setEditingItem] = useState(null);
  const [isNewItem, setIsNewItem] = useState(false);
  const [modalForm, setModalForm] = useState({
    title_ar: "",
    title_en: "",
    type: "movie",
    category_slug: "movies",
    plot_ar: "",
    plot_en: "",
    release_year: 2024,
    rating: 8.5,
    poster_path: "",
    banner_path: "",
    genres: [],
  });
  const [enrichLoading, setEnrichLoading] = useState(false);
  const [bulkEnrichStatus, setBulkEnrichStatus] = useState({
    state: "idle",
    completed: 0,
    failed: 0,
    total: 0,
  });
  const [originClassifyState, setOriginClassifyState] = useState("idle");
  const [saveLoading, setSaveLoading] = useState(false);
  const [customGenreInput, setCustomGenreInput] = useState("");

  // Delete Modal
  const [deletingItem, setDeletingItem] = useState(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  useEffect(() => {
    getCategories()
      .then((data) => {
        if (data.categories?.length > 0) setCategories(data.categories);
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    loadMediaItems();
  }, [selectedCategory, managerSort, managerSearch]);

  async function loadMediaItems() {
    setMediaLoading(true);
    try {
      const options = { limit: 100, sort: managerSort, q: managerSearch };
      if (selectedCategory !== "all") options.category = selectedCategory;
      const data = await getMediaList(options);
      if (data.items) {
        setMediaItems(data.items);
        setTotalMediaCount(data.total || data.items.length);
      } else {
        setMediaItems([]);
        setTotalMediaCount(0);
      }
    } catch {
      setMediaItems([]);
      setTotalMediaCount(0);
    } finally {
      setMediaLoading(false);
    }
  }

  const filteredMediaItems = useMemo(() => {
    return mediaItems.filter((item) => {
      if (selectedGenre !== "all") {
        const itemGenres = item.genres || [];
        if (!itemGenres.some((g) => g.includes(selectedGenre) || selectedGenre.includes(g)))
          return false;
      }
      if (selectedCountry !== "all") {
        const countryTerms = COUNTRY_TAGS[selectedCountry] || [];
        const searchable = (item.genres || []).join(" ").toLocaleLowerCase();
        if (!countryTerms.some((term) => searchable.includes(term.toLocaleLowerCase())))
          return false;
      }
      if (selectedYear !== "all") {
        const y = item.release_year || 0;
        if (selectedYear === "2024" && y !== 2024) return false;
        if (selectedYear === "2023" && y !== 2023) return false;
        if (selectedYear === "2020-2022" && (y < 2020 || y > 2022)) return false;
        if (selectedYear === "2010s" && (y < 2010 || y > 2019)) return false;
        if (selectedYear === "classic" && y >= 2010) return false;
      }
      if (selectedRating !== "all") {
        if ((item.rating || 0) < parseFloat(selectedRating)) return false;
      }
      return true;
    });
  }, [mediaItems, selectedGenre, selectedCountry, selectedYear, selectedRating]);

  // --- Handlers ---
  function handleOpenEdit(item) {
    setIsNewItem(false);
    setEditingItem(item);
    setModalForm({
      title_ar: item.title_ar || "",
      title_en: item.title_en || "",
      type: item.type || "movie",
      category_slug:
        item.category_slug || (selectedCategory !== "all" ? selectedCategory : "movies"),
      plot_ar: item.plot_ar || "",
      plot_en: item.plot_en || "",
      release_year: item.release_year || 2024,
      rating: item.rating || 8.0,
      poster_path: item.poster_path || "",
      banner_path: item.banner_path || "",
      genres: item.genres || [],
    });
  }

  function handleOpenAddNew() {
    setIsNewItem(true);
    setEditingItem({ id: "new" });
    setModalForm({
      title_ar: "",
      title_en: "",
      type:
        selectedCategory === "movies"
          ? "movie"
          : selectedCategory === "anime"
          ? "anime"
          : "series",
      category_slug: selectedCategory !== "all" ? selectedCategory : "movies",
      plot_ar: "",
      plot_en: "",
      release_year: 2024,
      rating: 8.5,
      poster_path: "",
      banner_path: "",
      genres: [],
    });
  }

  async function handleSaveMedia() {
    if (!modalForm.title_en.trim() && !modalForm.title_ar.trim()) {
      alert("يرجى إدخال اسم العمل (بالعربي أو الإنجليزي)");
      return;
    }
    setSaveLoading(true);
    try {
      const payload = {
        ...modalForm,
        category_slug: modalForm.type === "anime" ? "anime" : modalForm.type === "series" ? "series" : modalForm.category_slug,
      };
      if (isNewItem) {
        await createMediaItem(payload);
      } else {
        await updateMediaItem(editingItem.id, payload);
      }
      setEditingItem(null);
      loadMediaItems();
    } catch (err) {
      alert("تعذر الحفظ: " + (err?.message || ""));
    } finally {
      setSaveLoading(false);
    }
  }

  async function handleEnrichInModal() {
    if (!editingItem || isNewItem) return;
    setEnrichLoading(true);
    try {
      const res = await enrichMedia(editingItem.id);
      const metadata = Array.isArray(res.metadata) ? res.metadata : [res.metadata].filter(Boolean);
      const arabic = metadata.find((item) => String(item.locale || "").toLowerCase().startsWith("ar"));
      const english = metadata.find((item) => String(item.locale || "").toLowerCase().startsWith("en")) || metadata[0];
      if (english || arabic) {
        setModalForm((prev) => ({
          ...prev,
          title_ar: arabic?.title || prev.title_ar,
          title_en: english?.title || prev.title_en,
          plot_ar: arabic?.overview || prev.plot_ar,
          plot_en: english?.overview || prev.plot_en,
          release_year: english?.releaseYear || arabic?.releaseYear || prev.release_year,
          rating: english?.rating || arabic?.rating || prev.rating,
          poster_path: english?.cachedPosterPath || english?.posterPath || arabic?.cachedPosterPath || prev.poster_path,
          banner_path: english?.cachedBannerPath || english?.bannerPath || arabic?.cachedBannerPath || prev.banner_path,
          genres: english?.genres?.length > 0 ? english.genres : prev.genres,
        }));
      }
    } catch (err) {
      alert("تعذر الجلب: " + (err?.message || ""));
    } finally {
      setEnrichLoading(false);
    }
  }

  async function handleBulkEnrich() {
    const targets = filteredMediaItems.filter((item) => item?.id);
    if (targets.length === 0) {
      alert("لا توجد أعمال مطابقة للفلاتر الحالية لتحديثها.");
      return;
    }
    if (
      !window.confirm(
        `سيتم جلب وتوحيد بيانات ${targets.length} عملاً ظاهراً الآن من TMDB. قد يستغرق ذلك وقتاً. هل تريد المتابعة؟`
      )
    )
      return;
    let completed = 0,
      failed = 0;
    setBulkEnrichStatus({ state: "running", completed, failed, total: targets.length });
    for (const item of targets) {
      try {
        await enrichMedia(item.id);
        completed += 1;
      } catch {
        failed += 1;
      }
      setBulkEnrichStatus({ state: "running", completed, failed, total: targets.length });
    }
    setBulkEnrichStatus({ state: "done", completed, failed, total: targets.length });
    loadMediaItems();
  }

  async function handleClassifyOrigins() {
    if (
      !window.confirm(
        "سيقرأ النظام مجلدات مكتبتك ويضيف تلقائياً وسوماً مثل عربي، تركي، وكوري. لن يحذف أي تصنيف موجود. هل تريد المتابعة؟"
      )
    )
      return;
    setOriginClassifyState("running");
    try {
      const result = await classifyOriginsFromFolders();
      setOriginClassifyState(`done:${result.updated || 0}`);
      loadMediaItems();
    } catch (err) {
      setOriginClassifyState("error");
      alert("تعذر التصنيف من المجلدات: " + (err?.message || ""));
    }
  }

  function toggleGenreInModal(genre) {
    setModalForm((prev) => ({
      ...prev,
      genres: prev.genres.includes(genre)
        ? prev.genres.filter((g) => g !== genre)
        : [...prev.genres, genre],
    }));
  }

  function addCustomGenre() {
    if (!customGenreInput.trim()) return;
    const g = customGenreInput.trim();
    if (!modalForm.genres.includes(g)) {
      setModalForm((prev) => ({ ...prev, genres: [...prev.genres, g] }));
    }
    setCustomGenreInput("");
  }

  async function confirmDeleteMedia() {
    if (!deletingItem) return;
    setDeleteLoading(true);
    try {
      await deleteMediaItem(deletingItem.id);
      setDeletingItem(null);
      loadMediaItems();
    } catch (err) {
      alert("تعذر حذف العمل: " + (err?.message || ""));
    } finally {
      setDeleteLoading(false);
    }
  }

  return (
    <div className="space-y-6 text-right animate-fadeIn" dir="rtl">
      {/* Category Filter Pills */}
      <div className="flex items-center gap-2 overflow-x-auto pb-1 scrollbar-none">
        <button
          type="button"
          onClick={() => setSelectedCategory("all")}
          className={`px-4 py-2 rounded-2xl text-xs font-bold transition shrink-0 ${
            selectedCategory === "all"
              ? "bg-gradient-to-r from-fuchsia-600 to-purple-600 text-white shadow-lg shadow-fuchsia-950/40"
              : "bg-white/[0.05] text-white/70 hover:bg-white/10 hover:text-white"
          }`}
        >
          🌟 جميع الأقسام ({totalMediaCount})
        </button>
        {categories.map((cat) => (
          <button
            key={cat.slug}
            type="button"
            onClick={() => setSelectedCategory(cat.slug)}
            className={`px-4 py-2 rounded-2xl text-xs font-bold transition shrink-0 ${
              selectedCategory === cat.slug
                ? "bg-gradient-to-r from-fuchsia-600 to-purple-600 text-white shadow-lg shadow-fuchsia-950/40"
                : "bg-white/[0.05] text-white/70 hover:bg-white/10 hover:text-white"
            }`}
          >
            {cat.name_ar || cat.nameAr}
          </button>
        ))}
      </div>

      {/* Header & Bulk Operations Toolbar */}
      <div className="flex flex-col lg:flex-row items-start lg:items-center justify-between gap-4 p-5 rounded-3xl border border-white/10 bg-[#0C0A18]/80 backdrop-blur-2xl shadow-xl">
        <div>
          <div className="flex items-center gap-2">
            <span className="flex h-8 w-8 items-center justify-center rounded-xl bg-purple-500/20 text-purple-300">
              <Icon name="film" className="h-4 w-4" />
            </span>
            <h1 className="text-xl sm:text-2xl font-black text-white">
              إدارة وتحرير الأعمال الفنية (Media CMS)
            </h1>
          </div>
          <p className="text-xs text-white/50 mt-1">
            إضافة أعمال جديدة، تعديل البوسترات والبانرات، جلب الميتا من TMDB، وحذف الأعمال الفنية.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2.5">
          <Button
            variant="secondary"
            onClick={handleClassifyOrigins}
            disabled={originClassifyState === "running"}
            className="text-xs"
          >
            <span>
              {originClassifyState === "running" ? "جارٍ الفحص..." : "🏷️ تصنيف البلدان تلقائياً"}
            </span>
          </Button>

          <Button
            variant="secondary"
            onClick={handleBulkEnrich}
            disabled={bulkEnrichStatus.state === "running"}
            className="text-xs"
          >
            <span>
              {bulkEnrichStatus.state === "running"
                ? `تحديث (${bulkEnrichStatus.completed}/${bulkEnrichStatus.total})`
                : "✨ تحديث TMDB للكل"}
            </span>
          </Button>

          <Button
            variant="primary"
            onClick={handleOpenAddNew}
            className="rounded-2xl shadow-neon text-xs"
          >
            <span>➕ إضافة عمل جديد</span>
          </Button>
        </div>
      </div>

      {/* Filter Control Bar */}
      <Card className="p-4 space-y-3">
        <div className="grid gap-3 grid-cols-1 sm:grid-cols-2 lg:grid-cols-5">
          {/* Search Input */}
          <div className="relative">
            <input
              type="text"
              value={managerSearch}
              onChange={(e) => setManagerSearch(e.target.value)}
              placeholder="بحث بالاسم العربي أو الإنجليزي..."
              className="w-full rounded-xl border border-white/10 bg-black/40 pr-9 pl-3 py-2 text-xs text-white outline-none focus:border-fuchsia-500"
            />
            <span className="absolute right-3 top-2.5 text-white/40">🔍</span>
            {managerSearch && (
              <button
                type="button"
                onClick={() => setManagerSearch("")}
                className="absolute left-3 top-2 text-white/40 hover:text-white text-xs"
              >
                ✕
              </button>
            )}
          </div>

          {/* Sort Selector */}
          <select
            value={managerSort}
            onChange={(e) => setManagerSort(e.target.value)}
            className="rounded-xl border border-white/10 bg-black/40 px-3 py-2 text-xs text-white outline-none focus:border-fuchsia-500"
          >
            <option value="latest">الأحدث إضافة</option>
            <option value="rating">الأعلى تقييماً ⭐</option>
            <option value="year">سنة الإنتاج</option>
            <option value="title">أبجدياً (A - Z)</option>
          </select>

          {/* Country Selector */}
          <select
            value={selectedCountry}
            onChange={(e) => setSelectedCountry(e.target.value)}
            className="rounded-xl border border-white/10 bg-black/40 px-3 py-2 text-xs text-white outline-none focus:border-fuchsia-500"
          >
            {COUNTRIES_LIST.map((c) => (
              <option key={c.id} value={c.id}>
                {c.label}
              </option>
            ))}
          </select>

          {/* Year Selector */}
          <select
            value={selectedYear}
            onChange={(e) => setSelectedYear(e.target.value)}
            className="rounded-xl border border-white/10 bg-black/40 px-3 py-2 text-xs text-white outline-none focus:border-fuchsia-500"
          >
            {YEARS_LIST.map((y) => (
              <option key={y.id} value={y.id}>
                {y.label}
              </option>
            ))}
          </select>

          {/* Rating Selector */}
          <select
            value={selectedRating}
            onChange={(e) => setSelectedRating(e.target.value)}
            className="rounded-xl border border-white/10 bg-black/40 px-3 py-2 text-xs text-white outline-none focus:border-fuchsia-500"
          >
            {RATINGS_LIST.map((r) => (
              <option key={r.id} value={r.id}>
                {r.label}
              </option>
            ))}
          </select>
        </div>

        {/* Genre Badges Bar */}
        <div className="flex items-center gap-1.5 overflow-x-auto pt-2 border-t border-white/5 scrollbar-none">
          <span className="text-xs font-bold text-white/50 ml-1 shrink-0">التصنيف:</span>
          <button
            type="button"
            onClick={() => setSelectedGenre("all")}
            className={`px-2.5 py-1 rounded-xl text-xs font-bold transition shrink-0 ${
              selectedGenre === "all"
                ? "bg-fuchsia-600 text-white"
                : "bg-white/[0.04] text-white/60 hover:bg-white/10 hover:text-white"
            }`}
          >
            الكل
          </button>
          {ALL_GENRES.map((g) => (
            <button
              key={g}
              type="button"
              onClick={() => setSelectedGenre(g)}
              className={`px-2.5 py-1 rounded-xl text-xs font-medium transition shrink-0 ${
                selectedGenre === g
                  ? "bg-fuchsia-600 text-white shadow"
                  : "bg-white/[0.04] text-white/60 hover:bg-white/10 hover:text-white"
              }`}
            >
              {g}
            </button>
          ))}
        </div>
      </Card>

      {/* Media Collection Grid */}
      {mediaLoading ? (
        <div className="p-20 text-center text-white/60">
          <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-fuchsia-500 border-t-transparent" />
          <p className="mt-3 text-xs font-bold">جارٍ تحميل الأعمال...</p>
        </div>
      ) : filteredMediaItems.length === 0 ? (
        <Card className="p-16 text-center space-y-3">
          <p className="text-4xl">🎬</p>
          <h3 className="text-lg font-bold text-white">لا توجد أعمال تطابق الفلاتر المحددة</h3>
          <p className="text-xs text-white/50 max-w-md mx-auto">
            جرب تغيير خيارات الفلترة أو إضافة عمل فني جديد.
          </p>
          <Button variant="primary" onClick={handleOpenAddNew} className="mt-2">
            ➕ إضافة عمل جديد الآن
          </Button>
        </Card>
      ) : (
        <MediaCollection
          items={filteredMediaItems}
          onOpen={handleOpenEdit}
          className="admin-media-collection"
          cardActions={(item) => (
            <>
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  handleOpenEdit(item);
                }}
                className="flex h-9 w-9 items-center justify-center rounded-xl border border-white/15 bg-black/70 text-white shadow-sm backdrop-blur transition hover:bg-fuchsia-600 active:scale-95"
                title="تعديل العمل"
              >
                <Icon name="settings" className="h-4 w-4" />
              </button>
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  setDeletingItem(item);
                }}
                className="flex h-9 w-9 items-center justify-center rounded-xl border border-rose-500/20 bg-rose-950/80 text-rose-200 shadow-sm backdrop-blur transition hover:bg-rose-600 active:scale-95"
                title="حذف العمل"
              >
                <Icon name="close" className="h-4 w-4" />
              </button>
            </>
          )}
        />
      )}

      {/* Edit / Add Media Modal */}
      {editingItem && (
        <Modal
          isOpen={Boolean(editingItem)}
          onClose={() => setEditingItem(null)}
          title={isNewItem ? "➕ إضافة عمل سينمائي جديد" : "✏️ تعديل بيانات العمل والغلاف"}
          subtitle="تعديل العناوين، البوسترات والبانرات، القصة، والوسوم الفنية"
          size="xl"
          actions={
            <>
              {!isNewItem && (
                <Button
                  variant="secondary"
                  onClick={handleEnrichInModal}
                  loading={enrichLoading}
                  className="mr-auto"
                >
                  <span>{enrichLoading ? "جارٍ الجلب..." : "✨ جلب فوري من TMDB"}</span>
                </Button>
              )}
              <Button variant="ghost" onClick={() => setEditingItem(null)}>
                إلغاء
              </Button>
              <Button
                variant="primary"
                onClick={handleSaveMedia}
                loading={saveLoading}
                className="px-6"
              >
                💾 {saveLoading ? "جارٍ الحفظ..." : "حفظ التعديلات"}
              </Button>
            </>
          }
        >
          <div className="space-y-6 max-h-[75vh] overflow-y-auto pr-1 pl-1">
            {/* Section 1: Basic Information */}
            <div className="space-y-3">
              <h4 className="text-xs font-black text-fuchsia-400 uppercase tracking-wider flex items-center gap-1.5">
                <span>1. المعلومات الأساسية</span>
              </h4>
              <div className="grid gap-4 sm:grid-cols-2">
                <Input
                  label="العنوان باللغة العربية"
                  value={modalForm.title_ar}
                  onChange={(e) => setModalForm({ ...modalForm, title_ar: e.target.value })}
                  placeholder="مثال: صراع العروش أو إنسبشن"
                  required
                />
                <Input
                  label="العنوان باللغة الإنجليزية"
                  value={modalForm.title_en}
                  onChange={(e) => setModalForm({ ...modalForm, title_en: e.target.value })}
                  placeholder="e.g. Game of Thrones"
                  dir="ltr"
                  mono
                />
                <Select
                  label="القسم / التصنيف الرئيسي"
                  value={modalForm.category_slug}
                  onChange={(e) => setModalForm({ ...modalForm, category_slug: e.target.value })}
                  options={categories.map((c) => ({
                    value: c.slug,
                    label: c.name_ar || c.nameAr,
                  }))}
                />
                <Select
                  label="نوع العمل"
                  value={modalForm.type}
                  onChange={(e) => {
                    const type = e.target.value;
                    setModalForm({
                      ...modalForm,
                      type,
                      category_slug: type === "anime" ? "anime" : type === "series" ? "series" : modalForm.category_slug,
                    });
                  }}
                  options={[
                    { value: "movie", label: "فيلم (Movie)" },
                    { value: "series", label: "مسلسل (Series)" },
                    { value: "anime", label: "أنمي (Anime)" },
                  ]}
                />
                <Input
                  label="سنة الإنتاج"
                  type="number"
                  value={modalForm.release_year}
                  onChange={(e) =>
                    setModalForm({ ...modalForm, release_year: parseInt(e.target.value) || 0 })
                  }
                />
                <Input
                  label="التقييم (من 10)"
                  type="number"
                  step="0.1"
                  min="0"
                  max="10"
                  value={modalForm.rating}
                  onChange={(e) =>
                    setModalForm({ ...modalForm, rating: parseFloat(e.target.value) || 0 })
                  }
                />
              </div>
            </div>

            {/* Section 2: Dual Image Picker (Poster & Banner) */}
            <div className="space-y-3 pt-2 border-t border-white/10">
              <h4 className="text-xs font-black text-fuchsia-400 uppercase tracking-wider flex items-center gap-1.5">
                <span>2. الصور والأغلفة (رفع من الجهاز أو إدخال رابط)</span>
              </h4>
              <div className="grid gap-5 lg:grid-cols-2">
                <ImagePickerInput
                  label="بوستر العمل (Poster 2:3)"
                  value={modalForm.poster_path}
                  onChange={(val) => setModalForm({ ...modalForm, poster_path: val })}
                  aspectRatio="poster"
                  placeholder="https://... أو /posters/item.jpg"
                  helperText="الغلاف الرئيسي الرأسي للعمل الفني"
                />
                <ImagePickerInput
                  label="بانر العمل العريض (Banner 16:9)"
                  value={modalForm.banner_path}
                  onChange={(val) => setModalForm({ ...modalForm, banner_path: val })}
                  aspectRatio="banner"
                  placeholder="https://... أو /banners/item.jpg"
                  helperText="الخلفية العريضة في صفحة التفاصيل والهيدر"
                />
              </div>
            </div>

            {/* Section 3: Plot / Overview */}
            <div className="space-y-3 pt-2 border-t border-white/10">
              <h4 className="text-xs font-black text-fuchsia-400 uppercase tracking-wider flex items-center gap-1.5">
                <span>3. نبذة وقصة العمل</span>
              </h4>
              <div className="space-y-3">
                <Textarea
                  label="القصة باللغة العربية"
                  value={modalForm.plot_ar}
                  onChange={(e) => setModalForm({ ...modalForm, plot_ar: e.target.value })}
                  placeholder="نبذة مشوقة عن قصة وأحداث العمل الفني..."
                  rows={3}
                />
                <Textarea
                  label="Overview (English)"
                  value={modalForm.plot_en}
                  onChange={(e) => setModalForm({ ...modalForm, plot_en: e.target.value })}
                  placeholder="Synopsis of the movie or series in English..."
                  dir="ltr"
                  rows={2}
                />
              </div>
            </div>

            {/* Section 4: Genres & Tags */}
            <div className="space-y-3 pt-2 border-t border-white/10">
              <h4 className="text-xs font-black text-fuchsia-400 uppercase tracking-wider flex items-center gap-1.5">
                <span>4. التصنيفات والوسوم (Genres & Tags)</span>
              </h4>
              <div className="flex flex-wrap gap-1.5 max-h-36 overflow-y-auto p-2 rounded-2xl bg-black/40 border border-white/5">
                {ALL_GENRES.map((genre) => {
                  const isSelected = modalForm.genres.includes(genre);
                  return (
                    <button
                      key={genre}
                      type="button"
                      onClick={() => toggleGenreInModal(genre)}
                      className={`px-3 py-1 rounded-xl text-xs font-bold transition active:scale-95 ${
                        isSelected
                          ? "bg-gradient-to-r from-fuchsia-600 to-purple-600 text-white shadow"
                          : "bg-white/5 text-white/60 hover:bg-white/10 hover:text-white"
                      }`}
                    >
                      {genre} {isSelected ? "✓" : "+"}
                    </button>
                  );
                })}
              </div>

              {/* Custom Genre Adder */}
              <div className="flex items-center gap-2 pt-1">
                <input
                  type="text"
                  value={customGenreInput}
                  onChange={(e) => setCustomGenreInput(e.target.value)}
                  placeholder="إضافة وسم مخصص..."
                  className="flex-1 rounded-xl border border-white/10 bg-black/40 px-3 py-1.5 text-xs text-white outline-none focus:border-fuchsia-500"
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault();
                      addCustomGenre();
                    }
                  }}
                />
                <Button size="sm" variant="secondary" onClick={addCustomGenre}>
                  + إضافة وسم
                </Button>
              </div>
            </div>
          </div>
        </Modal>
      )}

      {/* Delete Confirmation Modal */}
      {deletingItem && (
        <ConfirmModal
          isOpen={Boolean(deletingItem)}
          onClose={() => setDeletingItem(null)}
          onConfirm={confirmDeleteMedia}
          loading={deleteLoading}
          title="تأكيد حذف العمل الفني"
          message={`هل أنت متأكد من رغبتك في حذف "${
            deletingItem.title_ar || deletingItem.title_en
          }"؟`}
          confirmText="نعم، حذف العمل"
          cancelText="إلغاء"
        />
      )}
    </div>
  );
}
