import { useEffect, useMemo, useState } from "react";
import GlassCard from "../components/GlassCard.jsx";
import Icon from "../components/Icon.jsx";
import {
  calculateChecksums,
  classifyOriginsFromFolders,
  copyMedia,
  createCategory,
  createMediaItem,
  deleteCategory,
  deleteMediaItem,
  enrichMedia,
  getCategories,
  getDisks,
  getDuplicateGroups,
  getMediaDetail,
  getMediaList,
  getMissingEpisodes,
  getQualityReport,
  indexLibrary,
  previewMigration,
  scanDisks,
  syncIndex,
  updateCategory,
  updateMediaItem
} from "../lib/api.js";

const DEFAULT_CATEGORIES = [
  { id: 1, slug: "movies", name_ar: "الأفلام", name_en: "Movies", icon: "film", color: "from-blue-600 to-indigo-700" },
  { id: 2, slug: "series", name_ar: "المسلسلات", name_en: "Series", icon: "tv", color: "from-emerald-600 to-teal-700" },
  { id: 3, slug: "anime", name_ar: "الأنمي", name_en: "Anime", icon: "spark", color: "from-purple-600 to-fuchsia-700" },
  { id: 4, slug: "kids", name_ar: "الأطفال والكرتون", name_en: "Kids & Cartoons", icon: "smile", color: "from-amber-500 to-orange-600" },
  { id: 5, slug: "documentaries", name_ar: "الوثائقيات", name_en: "Documentaries", icon: "book", color: "from-cyan-600 to-blue-700" },
  { id: 6, slug: "plays", name_ar: "المسرحيات", name_en: "Plays", icon: "mask", color: "from-rose-600 to-pink-700" }
];

const ALL_GENRES = [
  "أكشن", "مغامرة", "دراما", "كوميديا", "خيال علمي", "غموض", "إثارة", "رعب", "رومانسي",
  "فانتازيا", "تاريخي", "سيرة ذاتية", "جريمة", "شياطين", "نينجا", "طبيعة", "عائلي", "رياضي", "حرب", "موسيقى",
  "تركي", "عربي", "كوري", "أجنبي", "هندي", "إسباني"
];

const COUNTRIES_LIST = [
  { id: "all", label: "كافة الجنسيات" },
  { id: "hollywood", label: "أجنبي (هوليوود)" },
  { id: "arabic", label: "عربي" },
  { id: "turkish", label: "تركي" },
  { id: "korean", label: "كوري" },
  { id: "japanese", label: "ياباني / أنمي" },
  { id: "indian", label: "هندي (بوليوود)" }
];

const COUNTRY_TAGS = {
  hollywood: ["أجنبي", "أجنبية", "english", "american", "british", "hollywood", "أمريكي", "بريطاني"],
  arabic: ["عربي", "عربية", "arabic", "مصري", "خليجي"],
  turkish: ["تركي", "تركية", "turkish", "turkey"],
  korean: ["كوري", "كورية", "korean", "korea"],
  japanese: ["ياباني", "يابانية", "japanese", "anime", "أنمي"],
  indian: ["هندي", "هندية", "indian", "india", "bollywood"]
};

const QUALITIES_LIST = [
  { id: "all", label: "كافة الجودات" },
  { id: "4k", label: "4K UHD فائقة" },
  { id: "1080p", label: "1080p FHD عالية" },
  { id: "720p", label: "720p HD متوسطة" },
  { id: "480p", label: "480p SD عادية" }
];

const YEARS_LIST = [
  { id: "all", label: "كافة السنوات" },
  { id: "2024", label: "2024 حديث" },
  { id: "2023", label: "2023" },
  { id: "2020-2022", label: "2020 - 2022" },
  { id: "2010s", label: "2010 - 2019" },
  { id: "classic", label: "أقدم من 2010" }
];

const RATINGS_LIST = [
  { id: "all", label: "كافة التقييمات" },
  { id: "9", label: "★ 9.0 فما فوق (تحف فنية)" },
  { id: "8", label: "★ 8.0 فما فوق (ممتاز جداً)" },
  { id: "7", label: "★ 7.0 فما فوق (جيد)" }
];

export default function AdminPage({ health, onSyncIndex, activeAnchor, onNavigateCategory }) {
  // Main Tab Selection
  const [activeTab, setActiveTab] = useState(activeAnchor || "admin-categories");

  // =========================================================================
  // 1. Categories Management State
  // =========================================================================
  const [categories, setCategories] = useState(DEFAULT_CATEGORIES);
  const [categoriesLoading, setCategoriesLoading] = useState(false);
  const [categoryModal, setCategoryModal] = useState(null); // null | { isNew, form: { id, name_ar, name_en, slug } }
  const [categorySaveLoading, setCategorySaveLoading] = useState(false);
  const [deletingCategory, setDeletingCategory] = useState(null);

  // =========================================================================
  // 2. Media Catalog & Comprehensive Filters State
  // =========================================================================
  const [selectedCategory, setSelectedCategory] = useState("all");
  const [selectedGenre, setSelectedGenre] = useState("all");
  const [selectedCountry, setSelectedCountry] = useState("all");
  const [selectedQuality, setSelectedQuality] = useState("all");
  const [selectedYear, setSelectedYear] = useState("all");
  const [selectedRating, setSelectedRating] = useState("all");
  const [managerSort, setManagerSort] = useState("latest");
  const [managerSearch, setManagerSearch] = useState("");

  const [mediaItems, setMediaItems] = useState([]);
  const [mediaLoading, setMediaLoading] = useState(false);
  const [totalMediaCount, setTotalMediaCount] = useState(0);

  // Edit / Add Modal State
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
    genres: []
  });
  const [enrichLoading, setEnrichLoading] = useState(false);
  const [bulkEnrichStatus, setBulkEnrichStatus] = useState({ state: "idle", completed: 0, failed: 0, total: 0 });
  const [originClassifyState, setOriginClassifyState] = useState("idle");
  const [saveLoading, setSaveLoading] = useState(false);
  const [customGenreInput, setCustomGenreInput] = useState("");

  // Delete Media State
  const [deletingItem, setDeletingItem] = useState(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  // =========================================================================
  // 3. Indexing & Disks State
  // =========================================================================
  const [indexRoot, setIndexRoot] = useState("C:/Users/mousa/Desktop/project/NEXORA/server/testdata/test_media_root");
  const [indexState, setIndexState] = useState("idle");
  const [indexResult, setIndexResult] = useState(null);
  const [indexError, setIndexError] = useState("");
  const [disks, setDisks] = useState([]);
  const [disksLoading, setDisksLoading] = useState(false);

  // =========================================================================
  // 4. Quality & Health Report State
  // =========================================================================
  const [qualityReport, setQualityReport] = useState(null);
  const [qualityLoading, setQualityLoading] = useState(false);
  const [qualityTab, setQualityTab] = useState("missing"); // missing | duplicates | corrupted
  const [checksumState, setChecksumState] = useState("idle");

  // =========================================================================
  // 5. Migration State
  // =========================================================================
  const [previewRoot, setPreviewRoot] = useState("C:/Users/mousa/Desktop/project/NEXORA/server/testdata/test_media_root");
  const [previewState, setPreviewState] = useState("idle");
  const [previewResult, setPreviewResult] = useState(null);
  const [copyTarget, setCopyTarget] = useState("D:/Media_Organized");
  const [copySources, setCopySources] = useState("");
  const [copyState, setCopyState] = useState("idle");
  const [copyResult, setCopyResult] = useState(null);

  // Sync with activeAnchor prop from Sidebar
  useEffect(() => {
    if (activeAnchor) {
      setActiveTab(activeAnchor);
    }
  }, [activeAnchor]);

  // Load initial data
  useEffect(() => {
    loadCategoriesData();
    loadDisks();
    loadQualityReport();
  }, []);

  // Fetch Media Items whenever filter states change
  useEffect(() => {
    loadMediaItems();
  }, [selectedCategory, managerSort, managerSearch]);

  // Fetch categories from API
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

  // Fetch Media items from API
  async function loadMediaItems() {
    setMediaLoading(true);
    try {
      const options = {
        // The API accepts up to 100 entries; requesting 150 silently falls
        // back to its default page size, which hid most works from bulk repair.
        limit: 100,
        sort: managerSort,
        q: managerSearch
      };
      if (selectedCategory !== "all") {
        options.category = selectedCategory;
      }
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

  // Filter items in memory for advanced sub-facets (Genre, Quality, Country, Year, Rating)
  const filteredMediaItems = useMemo(() => {
    return mediaItems.filter((item) => {
      // 1. Genre filter
      if (selectedGenre !== "all") {
        const itemGenres = item.genres || [];
        const matches = itemGenres.some((g) => g.includes(selectedGenre) || selectedGenre.includes(g));
        if (!matches) return false;
      }

      // 2. Country / production origin is stored as an editable tag on the item.
      if (selectedCountry !== "all") {
        const countryTerms = COUNTRY_TAGS[selectedCountry] || [];
        const searchable = (item.genres || []).join(" ").toLocaleLowerCase();
        if (!countryTerms.some((term) => searchable.includes(term.toLocaleLowerCase()))) return false;
      }

      // 3. Year filter
      if (selectedYear !== "all") {
        const y = item.release_year || 0;
        if (selectedYear === "2024" && y !== 2024) return false;
        if (selectedYear === "2023" && y !== 2023) return false;
        if (selectedYear === "2020-2022" && (y < 2020 || y > 2022)) return false;
        if (selectedYear === "2010s" && (y < 2010 || y > 2019)) return false;
        if (selectedYear === "classic" && y >= 2010) return false;
      }

      // 4. Rating filter
      if (selectedRating !== "all") {
        const r = item.rating || 0;
        const minRating = parseFloat(selectedRating);
        if (r < minRating) return false;
      }

      return true;
    });
  }, [mediaItems, selectedGenre, selectedCountry, selectedYear, selectedRating]);

  // =========================================================================
  // Category CRUD Handlers
  // =========================================================================
  function handleOpenAddCategory() {
    setCategoryModal({
      isNew: true,
      form: { name_ar: "", name_en: "", slug: "" }
    });
  }

  function handleOpenEditCategory(cat) {
    setCategoryModal({
      isNew: false,
      form: { id: cat.id, name_ar: cat.name_ar, name_en: cat.name_en, slug: cat.slug }
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

  // =========================================================================
  // Media Item CRUD Handlers
  // =========================================================================
  function handleOpenEdit(item) {
    setIsNewItem(false);
    setEditingItem(item);
    setModalForm({
      title_ar: item.title_ar || "",
      title_en: item.title_en || "",
      type: item.type || "movie",
      category_slug: item.category_slug || (selectedCategory !== "all" ? selectedCategory : "movies"),
      plot_ar: item.plot_ar || "",
      plot_en: item.plot_en || "",
      release_year: item.release_year || 2023,
      rating: item.rating || 8.0,
      poster_path: item.poster_path || "",
      banner_path: item.banner_path || "",
      genres: item.genres || []
    });
  }

  function handleOpenAddNew() {
    setIsNewItem(true);
    setEditingItem({ id: "new" });
    setModalForm({
      title_ar: "",
      title_en: "",
      type: selectedCategory === "movies" ? "movie" : selectedCategory === "anime" ? "anime" : "series",
      category_slug: selectedCategory !== "all" ? selectedCategory : "movies",
      plot_ar: "",
      plot_en: "",
      release_year: 2024,
      rating: 8.5,
      poster_path: "",
      banner_path: "",
      genres: []
    });
  }

  async function handleSaveMedia() {
    if (!modalForm.title_en.trim() && !modalForm.title_ar.trim()) {
      alert("يرجى إدخال اسم العمل");
      return;
    }

    setSaveLoading(true);
    try {
      if (isNewItem) {
        await createMediaItem(modalForm);
      } else {
        await updateMediaItem(editingItem.id, modalForm);
      }
      setEditingItem(null);
      loadMediaItems();
      loadQualityReport();
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
      const enriched = Array.isArray(res.metadata) ? res.metadata[0] : res.metadata;
      if (enriched) {
        setModalForm((prev) => ({
          ...prev,
          title_ar: enriched.originalTitle || enriched.title || prev.title_ar,
          title_en: enriched.title || prev.title_en,
          plot_ar: enriched.overview || prev.plot_ar,
          plot_en: enriched.overview || prev.plot_en,
          release_year: enriched.releaseYear || prev.release_year,
          rating: enriched.rating || prev.rating,
          poster_path: enriched.cachedPosterPath || enriched.posterPath || prev.poster_path,
          banner_path: enriched.cachedBannerPath || enriched.bannerPath || prev.banner_path,
          genres: enriched.genres?.length > 0 ? enriched.genres : prev.genres
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
    if (!window.confirm(`سيتم جلب وتوحيد بيانات ${targets.length} عملاً ظاهراً الآن من TMDB/MAL. قد يستغرق ذلك عدة دقائق. هل تريد المتابعة؟`)) return;

    let completed = 0;
    let failed = 0;
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
    await Promise.all([loadMediaItems(), loadQualityReport()]);
  }

  async function handleClassifyOrigins() {
    if (!window.confirm("سيقرأ النظام مجلدات مكتبتك ويضيف تلقائياً وسوماً مثل عربي، تركي، وكوري. لن يحذف أي تصنيف موجود. هل تريد المتابعة؟")) return;
    setOriginClassifyState("running");
    try {
      const result = await classifyOriginsFromFolders();
      setOriginClassifyState(`done:${result.updated || 0}`);
      await loadMediaItems();
    } catch (err) {
      setOriginClassifyState("error");
      alert("تعذر التصنيف من المجلدات: " + (err?.message || ""));
    }
  }

  function toggleGenreInModal(genre) {
    setModalForm((prev) => {
      const exists = prev.genres.includes(genre);
      return {
        ...prev,
        genres: exists ? prev.genres.filter((g) => g !== genre) : [...prev.genres, genre]
      };
    });
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
      loadQualityReport();
    } catch (err) {
      alert("تعذر حذف العمل: " + (err?.message || ""));
    } finally {
      setDeleteLoading(false);
    }
  }

  // =========================================================================
  // Indexing, Disks, Quality
  // =========================================================================
  async function loadDisks() {
    setDisksLoading(true);
    try {
      const data = await getDisks();
      if (data.disks && data.disks.length > 0) setDisks(data.disks);
    } catch {
      setDisks([
        { disk_letter: "C", disk_label: "النظام (System)", total_space: 512000000000, free_space: 180000000000, used_space: 332000000000, used_percent: 64.8 },
        { disk_letter: "D", disk_label: "مكتبة الأفلام والمسلسلات (D:)", total_space: 8000000000000, free_space: 3200000000000, used_space: 4800000000000, used_percent: 60.0 },
        { disk_letter: "E", disk_label: "أرشيف الأنمي والميديا (E:)", total_space: 14000000000000, free_space: 2100000000000, used_space: 1190000000000, used_percent: 85.0 }
      ]);
    } finally {
      setDisksLoading(false);
    }
  }

  async function handleScanDisks() {
    setDisksLoading(true);
    try {
      const data = await scanDisks();
      if (data.disks) setDisks(data.disks);
    } finally {
      setDisksLoading(false);
    }
  }

  async function loadQualityReport() {
    setQualityLoading(true);
    try {
      const report = await getQualityReport();
      setQualityReport(report);
    } catch {
      // fallback
    } finally {
      setQualityLoading(false);
    }
  }

  async function handleIndex() {
    if (!indexRoot.trim()) return;
    setIndexState("loading");
    setIndexError("");
    try {
      const res = await indexLibrary([indexRoot.trim()]);
      setIndexResult(res);
      setIndexState("success");
      loadMediaItems();
      loadQualityReport();
      loadDisks();
    } catch (err) {
      setIndexError(err?.message || "تعذر إكمال الفهرسة.");
      setIndexState("error");
    }
  }

  async function handleTriggerChecksums() {
    setChecksumState("loading");
    try {
      await calculateChecksums();
      setChecksumState("success");
      loadQualityReport();
    } catch {
      setChecksumState("error");
    }
  }

  async function handlePreview() {
    if (!previewRoot.trim()) return;
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
    const sources = copySources.split("\n").map((s) => s.trim()).filter(Boolean);
    if (sources.length === 0 || !copyTarget.trim()) return;
    setCopyState("loading");
    try {
      const result = await copyMedia({ sources, target: copyTarget.trim() });
      setCopyResult(result);
      setCopyState("ready");
    } catch {
      setCopyState("error");
    }
  }

  function formatBytes(bytes) {
    if (!bytes || bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB", "PB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  }

  return (
    <div className="space-y-6 text-right" dir="rtl">
      {/* Top Banner Navigation */}
      <GlassCard className="p-6 border-fuchsia-500/25 bg-gradient-to-r from-purple-950/40 via-black/60 to-fuchsia-950/30">
        <div className="flex flex-col xl:flex-row items-start xl:items-center justify-between gap-5">
          <div>
            <div className="flex items-center gap-2 text-xs font-bold text-fuchsia-400">
              <span className="h-2 w-2 rounded-full bg-emerald-400 animate-pulse"></span>
              <span>لوحة التحكم وإدارة الوسائط الموحدة — NEXORA ADMIN SUITE</span>
            </div>
            <h1 className="mt-2 text-3xl font-black text-white tracking-wide md:text-4xl">
              إدارة الأقسام والمكتبة والأقراص
            </h1>
            <p className="mt-1 text-sm text-white/70">
              إدارة مرنة للأقسام والتصنيفات، تحرير وتخصيص الأعمال، فلاتر شاملة، وفهرسة لحظية للأقراص.
            </p>
          </div>

          {/* Primary Admin Tabs Switcher */}
          <div className="flex flex-wrap gap-2 p-1.5 rounded-2xl bg-black/60 border border-white/10 shadow-inner">
            <button
              onClick={() => setActiveTab("admin-categories")}
              className={`flex items-center gap-2 px-4 py-2.5 rounded-xl text-xs font-bold transition ${
                activeTab === "admin-categories"
                  ? "bg-gradient-to-r from-purple-600 to-fuchsia-600 text-white shadow-lg shadow-purple-900/50"
                  : "text-white/60 hover:text-white hover:bg-white/[0.05]"
              }`}
            >
              <Icon name="mask" className="h-4 w-4" />
              <span>🗂️ إدارة الأقسام والتصنيفات</span>
            </button>

            <button
              onClick={() => setActiveTab("admin-manager")}
              className={`flex items-center gap-2 px-4 py-2.5 rounded-xl text-xs font-bold transition ${
                activeTab === "admin-manager"
                  ? "bg-gradient-to-r from-purple-600 to-fuchsia-600 text-white shadow-lg shadow-purple-900/50"
                  : "text-white/60 hover:text-white hover:bg-white/[0.05]"
              }`}
            >
              <Icon name="film" className="h-4 w-4" />
              <span>🎬 إدارة وتحرير الأعمال (CMS)</span>
            </button>

            <button
              onClick={() => setActiveTab("admin-indexing")}
              className={`flex items-center gap-2 px-4 py-2.5 rounded-xl text-xs font-bold transition ${
                activeTab === "admin-indexing"
                  ? "bg-gradient-to-r from-purple-600 to-fuchsia-600 text-white shadow-lg shadow-purple-900/50"
                  : "text-white/60 hover:text-white hover:bg-white/[0.05]"
              }`}
            >
              <Icon name="search" className="h-4 w-4" />
              <span>💾 فهرسة واختيار المجلدات</span>
            </button>

            <button
              onClick={() => setActiveTab("admin-quality")}
              className={`flex items-center gap-2 px-4 py-2.5 rounded-xl text-xs font-bold transition ${
                activeTab === "admin-quality"
                  ? "bg-gradient-to-r from-purple-600 to-fuchsia-600 text-white shadow-lg shadow-purple-900/50"
                  : "text-white/60 hover:text-white hover:bg-white/[0.05]"
              }`}
            >
              <Icon name="spark" className="h-4 w-4" />
              <span>🛡️ صحة وجودة المكتبة</span>
              {qualityReport?.missing_episodes_count > 0 && (
                <span className="px-1.5 py-0.5 rounded-full bg-amber-500/30 text-amber-300 text-[10px]">
                  {qualityReport.missing_episodes_count}
                </span>
              )}
            </button>

            <button
              onClick={() => setActiveTab("admin-migration")}
              className={`flex items-center gap-2 px-4 py-2.5 rounded-xl text-xs font-bold transition ${
                activeTab === "admin-migration"
                  ? "bg-gradient-to-r from-purple-600 to-fuchsia-600 text-white shadow-lg shadow-purple-900/50"
                  : "text-white/60 hover:text-white hover:bg-white/[0.05]"
              }`}
            >
              <Icon name="arrowLeft" className="h-4 w-4" />
              <span>🔄 معالج الترتيب والنقل</span>
            </button>

            <button
              onClick={() => setActiveTab("admin-overview")}
              className={`flex items-center gap-2 px-4 py-2.5 rounded-xl text-xs font-bold transition ${
                activeTab === "admin-overview"
                  ? "bg-gradient-to-r from-purple-600 to-fuchsia-600 text-white shadow-lg shadow-purple-900/50"
                  : "text-white/60 hover:text-white hover:bg-white/[0.05]"
              }`}
            >
              <Icon name="dashboard" className="h-4 w-4" />
              <span>📊 حالة النظام</span>
            </button>
          </div>
        </div>
      </GlassCard>

      {/* ========================================================================= */}
      {/* 1. TAB: CATEGORIES & SECTIONS MANAGER (إدارة الأقسام والتصنيفات)          */}
      {/* ========================================================================= */}
      {activeTab === "admin-categories" && (
        <div className="space-y-6 animate-fadeIn">
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
                      onClick={() => {
                        setSelectedCategory(cat.slug);
                        setActiveTab("admin-manager");
                      }}
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
        </div>
      )}

      {/* ========================================================================= */}
      {/* 2. TAB: MEDIA CMS & COMPREHENSIVE FILTERS (إدارة وتحرير الأعمال والوسائط)  */}
      {/* ========================================================================= */}
      {activeTab === "admin-manager" && (
        <div className="space-y-6 animate-fadeIn">
          {/* Main Category Filter Pills */}
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={() => setSelectedCategory("all")}
              className={`px-4 py-2.5 rounded-2xl text-xs font-bold transition ${
                selectedCategory === "all"
                  ? "bg-gradient-to-r from-purple-600 to-fuchsia-600 text-white shadow-lg shadow-purple-900/50"
                  : "bg-white/[0.04] text-white/70 hover:bg-white/10 hover:text-white"
              }`}
            >
              🌟 كافة الأقسام ({totalMediaCount})
            </button>
            {categories.map((cat) => {
              const active = selectedCategory === cat.slug;
              return (
                <button
                  key={cat.slug}
                  type="button"
                  onClick={() => setSelectedCategory(cat.slug)}
                  className={`px-4 py-2.5 rounded-2xl text-xs font-bold transition ${
                    active
                      ? "bg-gradient-to-r from-purple-600 to-fuchsia-600 text-white shadow-lg shadow-purple-900/50"
                      : "bg-white/[0.04] text-white/70 hover:bg-white/10 hover:text-white"
                  }`}
                >
                  {cat.name_ar || cat.nameAr}
                </button>
              );
            })}
          </div>

          {/* Multi-Faceted Filter Control Center */}
          <GlassCard className="p-5 space-y-4">
            {/* Row 1: Search, Sort, Add New */}
            <div className="flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3">
              <div className="relative flex-1">
                <input
                  type="text"
                  value={managerSearch}
                  onChange={(e) => setManagerSearch(e.target.value)}
                  placeholder="بحث سريع بالعنوان العربي أو الإنجليزي أو الكلمات المفتاحية..."
                  className="w-full rounded-2xl border border-white/10 bg-black/40 px-4 py-3 pl-10 text-sm text-white outline-none placeholder:text-white/30 focus:border-fuchsia-500 transition"
                />
                <Icon name="search" className="absolute left-3.5 top-3.5 h-4 w-4 text-white/40" />
              </div>

              {/* Sorting */}
              <div className="flex items-center gap-2">
                <span className="text-xs text-white/50 whitespace-nowrap">الترتيب:</span>
                <select
                  value={managerSort}
                  onChange={(e) => setManagerSort(e.target.value)}
                  className="rounded-2xl border border-white/10 bg-black/50 px-3.5 py-3 text-xs font-bold text-white outline-none focus:border-fuchsia-500"
                >
                  <option value="latest">الأحدث إضافة</option>
                  <option value="rating">الأعلى تقييماً ★</option>
                  <option value="year">سنة الإنتاج 📅</option>
                  <option value="title">أبجدي أ-ي</option>
                </select>
              </div>

              {/* Add New Button */}
              <button
                type="button"
                onClick={handleOpenAddNew}
                className="flex items-center justify-center gap-2 rounded-2xl bg-gradient-to-r from-emerald-600 via-teal-600 to-emerald-700 px-6 py-3 text-xs font-bold text-white shadow-lg shadow-emerald-900/40 hover:brightness-110 transition whitespace-nowrap"
              >
                <span>➕ إضافة عمل جديد</span>
              </button>
              <button
                type="button"
                onClick={handleBulkEnrich}
                disabled={bulkEnrichStatus.state === "running" || filteredMediaItems.length === 0}
                className="flex items-center justify-center gap-2 rounded-2xl border border-fuchsia-400/30 bg-fuchsia-500/10 px-5 py-3 text-xs font-bold text-fuchsia-100 transition hover:bg-fuchsia-500/20 disabled:cursor-not-allowed disabled:opacity-50 whitespace-nowrap"
                title="يجلب النوع وبلد الإنتاج والصور والتقييمات للأعمال الظاهرة وفق الفلاتر الحالية"
              >
                <span>{bulkEnrichStatus.state === "running" ? `جارٍ التحديث ${bulkEnrichStatus.completed}/${bulkEnrichStatus.total}` : "✨ إصلاح بيانات الأعمال الظاهرة"}</span>
              </button>
              <button
                type="button"
                onClick={handleClassifyOrigins}
                disabled={originClassifyState === "running"}
                className="flex items-center justify-center gap-2 rounded-2xl border border-cyan-400/30 bg-cyan-500/10 px-5 py-3 text-xs font-bold text-cyan-100 transition hover:bg-cyan-500/20 disabled:cursor-not-allowed disabled:opacity-50 whitespace-nowrap"
                title="يعتمد على أسماء مجلداتك مثل مسلسلات/عربي أو مسلسلات تركية، ولا يحتاج TMDB"
              >
                <span>{originClassifyState === "running" ? "جارٍ قراءة المجلدات..." : "🗂️ تصنيف البلد من المجلدات"}</span>
              </button>
            </div>

            {bulkEnrichStatus.state !== "idle" && (
              <div className={`rounded-xl border px-3 py-2 text-xs ${bulkEnrichStatus.state === "done" ? "border-emerald-400/20 bg-emerald-500/10 text-emerald-200" : "border-fuchsia-400/20 bg-fuchsia-500/10 text-fuchsia-100"}`}>
                {bulkEnrichStatus.state === "running" ? `يتم الآن توحيد البيانات: ${bulkEnrichStatus.completed} مكتمل، ${bulkEnrichStatus.failed} تعذّر.` : `اكتمل تحديث البيانات: ${bulkEnrichStatus.completed} عمل، وتعذّر ${bulkEnrichStatus.failed}. الآن ستعمل فلاتر البلد والنوع تلقائياً.`}
              </div>
            )}
            {originClassifyState.startsWith("done:") && <p className="text-xs text-cyan-200">تمت قراءة مجلدات المكتبة وتصنيف {originClassifyState.split(":")[1]} عملاً. افتح صفحة المسلسلات الآن لتظهر المجموعات الصحيحة.</p>}

            {/* Row 2: Secondary Dropdown Filters (Country, Quality, Year, Rating) */}
            <div className="grid gap-3 sm:grid-cols-2 md:grid-cols-4 pt-3 border-t border-white/10">
              {/* Year Filter */}
              <div>
                <label className="block text-[11px] font-bold text-white/50 mb-1">📅 سنة الإنتاج</label>
                <select
                  value={selectedYear}
                  onChange={(e) => setSelectedYear(e.target.value)}
                  className="w-full rounded-xl border border-white/10 bg-black/40 px-3 py-2 text-xs text-white outline-none focus:border-fuchsia-500"
                >
                  {YEARS_LIST.map((y) => (
                    <option key={y.id} value={y.id}>{y.label}</option>
                  ))}
                </select>
              </div>

              {/* Rating Filter */}
              <div>
                <label className="block text-[11px] font-bold text-white/50 mb-1">★ التقييم</label>
                <select
                  value={selectedRating}
                  onChange={(e) => setSelectedRating(e.target.value)}
                  className="w-full rounded-xl border border-white/10 bg-black/40 px-3 py-2 text-xs text-white outline-none focus:border-fuchsia-500"
                >
                  {RATINGS_LIST.map((r) => (
                    <option key={r.id} value={r.id}>{r.label}</option>
                  ))}
                </select>
              </div>

              {/* Country / Nationality */}
              <div>
                <label className="block text-[11px] font-bold text-white/50 mb-1">🌍 جهة الإنتاج / البلد</label>
                <select
                  value={selectedCountry}
                  onChange={(e) => setSelectedCountry(e.target.value)}
                  className="w-full rounded-xl border border-white/10 bg-black/40 px-3 py-2 text-xs text-white outline-none focus:border-fuchsia-500"
                >
                  {COUNTRIES_LIST.map((c) => (
                    <option key={c.id} value={c.id}>{c.label}</option>
                  ))}
                </select>
              </div>

              {/* Quality Filter */}
              <div>
                <label className="block text-[11px] font-bold text-white/50 mb-1">📺 الجودة والدقة</label>
                <select
                  value={selectedQuality}
                  onChange={(e) => setSelectedQuality(e.target.value)}
                  className="w-full rounded-xl border border-white/10 bg-black/40 px-3 py-2 text-xs text-white outline-none focus:border-fuchsia-500"
                >
                  {QUALITIES_LIST.map((q) => (
                    <option key={q.id} value={q.id}>{q.label}</option>
                  ))}
                </select>
              </div>
            </div>

            {/* Row 3: Genres Pills Selector */}
            <div className="flex flex-wrap items-center gap-1.5 pt-3 border-t border-white/10">
              <span className="text-[11px] text-white/40 ml-2">نوع وتصنيف العمل (Genre):</span>
              <button
                type="button"
                onClick={() => setSelectedGenre("all")}
                className={`px-3 py-1 rounded-xl text-xs font-bold transition ${
                  selectedGenre === "all" ? "bg-fuchsia-600 text-white shadow" : "bg-white/[0.05] text-white/60 hover:text-white"
                }`}
              >
                الكل
              </button>
              {ALL_GENRES.map((g) => {
                const active = selectedGenre === g;
                return (
                  <button
                    key={g}
                    type="button"
                    onClick={() => setSelectedGenre(g)}
                    className={`px-2.5 py-1 rounded-xl text-xs font-medium transition ${
                      active ? "bg-fuchsia-600 text-white shadow" : "bg-white/[0.04] text-white/60 hover:bg-white/10 hover:text-white"
                    }`}
                  >
                    {g}
                  </button>
                );
              })}
            </div>
          </GlassCard>

          {/* Media Items Display Grid */}
          {mediaLoading ? (
            <div className="p-16 text-center text-white/60">
              <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-fuchsia-500 border-t-transparent"></div>
              <p className="mt-3 text-sm font-bold">جارٍ تحميل الأعمال...</p>
            </div>
          ) : filteredMediaItems.length === 0 ? (
            <GlassCard className="p-16 text-center">
              <p className="text-4xl">🎬</p>
              <h3 className="mt-3 text-lg font-bold text-white">لا توجد أعمال تطابق الفلاتر المحددة</h3>
              <p className="mt-1 text-xs text-white/50 max-w-md mx-auto">
                جرب تغيير خيارات الفلترة أو إضافة عمل جديد بنقرة زر.
              </p>
              <button
                type="button"
                onClick={handleOpenAddNew}
                className="mt-5 inline-flex items-center gap-2 rounded-2xl bg-fuchsia-600 px-5 py-2.5 text-xs font-bold text-white hover:bg-fuchsia-500 transition"
              >
                <span>➕ إضافة عمل الآن</span>
              </button>
            </GlassCard>
          ) : (
            <div className="grid gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
              {filteredMediaItems.map((item) => {
                const poster = item.poster_path || "/images/tokyo_ghoul_hero.png";
                return (
                  <div
                    key={item.id}
                    className="group relative flex flex-col justify-between overflow-hidden rounded-2xl border border-white/10 bg-[#0E0C1A] hover:border-fuchsia-500/50 hover:shadow-xl hover:shadow-purple-950/50 transition duration-300"
                  >
                    {/* Poster */}
                    <div className="relative aspect-[2/3] w-full overflow-hidden bg-black/60">
                      <img
                        src={poster}
                        alt={item.title_en}
                        className="h-full w-full object-cover group-hover:scale-105 transition duration-500"
                        onError={(e) => {
                          e.target.src = "/images/tokyo_ghoul_hero.png";
                        }}
                      />
                      <div className="absolute inset-0 bg-gradient-to-t from-[#0E0C1A] via-transparent to-black/40"></div>

                      {/* Top Badges */}
                      <div className="absolute top-2.5 right-2.5 left-2.5 flex items-center justify-between">
                        <span className="flex items-center gap-1 rounded-lg bg-black/70 px-2 py-1 text-[11px] font-bold text-amber-300 backdrop-blur-md">
                          ★ {item.rating ? item.rating.toFixed(1) : "8.0"}
                        </span>
                        <span className="rounded-lg bg-black/70 px-2 py-1 text-[11px] font-mono font-bold text-white/80 backdrop-blur-md">
                          {item.release_year || "2023"}
                        </span>
                      </div>

                      {/* Quick Action Overlay on Hover */}
                      <div className="absolute inset-0 flex items-center justify-center gap-2 bg-black/75 opacity-0 group-hover:opacity-100 transition-opacity backdrop-blur-sm">
                        <button
                          type="button"
                          onClick={() => handleOpenEdit(item)}
                          className="flex h-10 w-10 items-center justify-center rounded-xl bg-fuchsia-600 text-white shadow-lg hover:scale-110 transition"
                          title="تعديل العمل"
                        >
                          ✏️
                        </button>
                        <button
                          type="button"
                          onClick={() => setDeletingItem(item)}
                          className="flex h-10 w-10 items-center justify-center rounded-xl bg-rose-600 text-white shadow-lg hover:scale-110 transition"
                          title="حذف العمل"
                        >
                          🗑️
                        </button>
                      </div>
                    </div>

                    {/* Meta Info */}
                    <div className="p-3.5 space-y-2 flex-1 flex flex-col justify-between">
                      <div>
                        <h4 className="font-bold text-white text-sm line-clamp-1 group-hover:text-fuchsia-300 transition">
                          {item.title_ar || item.title_en}
                        </h4>
                        {item.title_ar && item.title_en && item.title_ar !== item.title_en && (
                          <p className="text-[11px] text-white/40 line-clamp-1 font-mono">{item.title_en}</p>
                        )}
                      </div>

                      {/* Genres tags */}
                      <div className="flex flex-wrap gap-1">
                        {item.genres?.slice(0, 2).map((genre, idx) => (
                          <span
                            key={idx}
                            className="rounded-md bg-white/[0.06] border border-white/5 px-1.5 py-0.5 text-[10px] text-white/70"
                          >
                            {genre}
                          </span>
                        ))}
                      </div>

                      {/* Bottom Footer */}
                      <div className="flex items-center justify-between border-t border-white/5 pt-2 text-[11px] text-white/40">
                        <span>{item.file_count || 1} ملف / حلقة</span>
                        <button
                          type="button"
                          onClick={() => handleOpenEdit(item)}
                          className="text-fuchsia-400 font-bold hover:underline"
                        >
                          تعديل ↵
                        </button>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* ========================================================================= */}
      {/* CATEGORY ADD/EDIT MODAL                                                   */}
      {/* ========================================================================= */}
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

      {/* ========================================================================= */}
      {/* CATEGORY DELETE MODAL                                                     */}
      {/* ========================================================================= */}
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

      {/* ========================================================================= */}
      {/* MEDIA ITEM EDIT / ADD MODAL                                               */}
      {/* ========================================================================= */}
      {editingItem && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/85 p-3 sm:p-6 backdrop-blur-md overflow-y-auto animate-fadeIn">
          <div className="relative w-full max-w-4xl rounded-3xl border border-fuchsia-500/30 bg-[#0D0B18] p-6 sm:p-8 shadow-2xl space-y-6 max-h-[92vh] overflow-y-auto">
            <div className="flex items-center justify-between border-b border-white/10 pb-4">
              <div>
                <h3 className="text-2xl font-black text-white">
                  {isNewItem ? "➕ إضافة عمل سينمائي جديد" : "✏️ تعديل بيانات العمل والغلاف"}
                </h3>
                <p className="text-xs text-white/50 mt-1">
                  تعديل الاسم العربي والإنجليزي، البوسترات، القصة، والوسوم
                </p>
              </div>

              <div className="flex items-center gap-2">
                {!isNewItem && (
                  <button
                    type="button"
                    onClick={handleEnrichInModal}
                    disabled={enrichLoading}
                    className="flex items-center gap-2 px-3.5 py-2 rounded-xl bg-gradient-to-r from-purple-600 to-fuchsia-600 text-xs font-bold text-white hover:brightness-110 transition disabled:opacity-50"
                  >
                    <span>{enrichLoading ? "جارٍ الجلب..." : "✨ جلب فوري (TMDB/MAL)"}</span>
                  </button>
                )}
                <button
                  type="button"
                  onClick={() => setEditingItem(null)}
                  className="rounded-xl border border-white/10 p-2 text-white/60 hover:text-white"
                >
                  ✕
                </button>
              </div>
            </div>

            <div className="grid gap-5 sm:grid-cols-2">
              <div>
                <label className="block text-xs font-bold text-white/70 mb-1.5">العنوان باللغة العربية</label>
                <input
                  type="text"
                  value={modalForm.title_ar}
                  onChange={(e) => setModalForm({ ...modalForm, title_ar: e.target.value })}
                  placeholder="مثال: هجوم العمالقة أو إنسبشن"
                  className="w-full rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-sm text-white outline-none focus:border-fuchsia-500"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-white/70 mb-1.5">العنوان باللغة الإنجليزية</label>
                <input
                  type="text"
                  value={modalForm.title_en}
                  onChange={(e) => setModalForm({ ...modalForm, title_en: e.target.value })}
                  placeholder="e.g. Attack on Titan or Inception"
                  dir="ltr"
                  className="w-full rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-sm text-white outline-none focus:border-fuchsia-500 font-mono"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-white/70 mb-1.5">القسم والمكتبة</label>
                <select
                  value={modalForm.category_slug}
                  onChange={(e) => setModalForm({ ...modalForm, category_slug: e.target.value })}
                  className="w-full rounded-xl border border-white/10 bg-black/50 px-3.5 py-2.5 text-sm text-white outline-none focus:border-fuchsia-500"
                >
                  {categories.map((c) => (
                    <option key={c.slug} value={c.slug}>{c.name_ar || c.nameAr}</option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-xs font-bold text-white/70 mb-1.5">نوع العمل</label>
                <select
                  value={modalForm.type}
                  onChange={(e) => setModalForm({ ...modalForm, type: e.target.value })}
                  className="w-full rounded-xl border border-white/10 bg-black/50 px-3.5 py-2.5 text-sm text-white outline-none focus:border-fuchsia-500"
                >
                  <option value="movie">فيلم (Movie)</option>
                  <option value="series">مسلسل (Series)</option>
                  <option value="anime">أنمي (Anime)</option>
                </select>
              </div>

              <div>
                <label className="block text-xs font-bold text-white/70 mb-1.5">سنة الإنتاج</label>
                <input
                  type="number"
                  value={modalForm.release_year}
                  onChange={(e) => setModalForm({ ...modalForm, release_year: parseInt(e.target.value) || 0 })}
                  className="w-full rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-sm text-white outline-none focus:border-fuchsia-500"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-white/70 mb-1.5">التقييم (من 10)</label>
                <input
                  type="number"
                  step="0.1"
                  min="0"
                  max="10"
                  value={modalForm.rating}
                  onChange={(e) => setModalForm({ ...modalForm, rating: parseFloat(e.target.value) || 0 })}
                  className="w-full rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-sm text-white outline-none focus:border-fuchsia-500"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-white/70 mb-1.5">رابط أو مسار البوستر (Poster)</label>
                <input
                  type="text"
                  value={modalForm.poster_path}
                  onChange={(e) => setModalForm({ ...modalForm, poster_path: e.target.value })}
                  placeholder="/images/poster.png أو رابط https://"
                  dir="ltr"
                  className="w-full rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-xs text-white outline-none focus:border-fuchsia-500 font-mono"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-white/70 mb-1.5">رابط أو مسار البانر (Banner)</label>
                <input
                  type="text"
                  value={modalForm.banner_path}
                  onChange={(e) => setModalForm({ ...modalForm, banner_path: e.target.value })}
                  placeholder="/images/banner.png أو رابط https://"
                  dir="ltr"
                  className="w-full rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-xs text-white outline-none focus:border-fuchsia-500 font-mono"
                />
              </div>
            </div>

            {/* Genres */}
            <div>
              <label className="block text-xs font-bold text-white/70 mb-2">التصنيفات والوسوم (Genres)</label>
              <div className="flex flex-wrap gap-1.5">
                {ALL_GENRES.map((genre) => {
                  const selected = modalForm.genres.includes(genre);
                  return (
                    <button
                      key={genre}
                      type="button"
                      onClick={() => toggleGenreInModal(genre)}
                      className={`px-3 py-1 rounded-xl text-xs font-semibold transition ${
                        selected ? "bg-fuchsia-600 text-white shadow" : "bg-white/[0.05] text-white/60 hover:bg-white/10 hover:text-white"
                      }`}
                    >
                      {selected ? `✓ ${genre}` : `+ ${genre}`}
                    </button>
                  );
                })}
              </div>

              <div className="mt-2.5 flex items-center gap-2">
                <input
                  type="text"
                  value={customGenreInput}
                  onChange={(e) => setCustomGenreInput(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && addCustomGenre()}
                  placeholder="إضافة تصنيف مخصص آخر..."
                  className="rounded-xl border border-white/10 bg-black/30 px-3 py-1.5 text-xs text-white outline-none focus:border-fuchsia-500"
                />
                <button
                  type="button"
                  onClick={addCustomGenre}
                  className="px-3 py-1.5 rounded-xl bg-white/10 hover:bg-white/20 text-xs font-bold text-white"
                >
                  إضافة
                </button>
              </div>
            </div>

            {/* Plot */}
            <div>
              <label className="block text-xs font-bold text-white/70 mb-1.5">القصة والوصف (Overview / Plot)</label>
              <textarea
                rows={3}
                value={modalForm.plot_ar || modalForm.plot_en}
                onChange={(e) => setModalForm({ ...modalForm, plot_ar: e.target.value, plot_en: e.target.value })}
                placeholder="اكتب نبذة عن قصة الفيلم أو المسلسل..."
                className="w-full rounded-xl border border-white/10 bg-black/40 p-3 text-xs text-white outline-none focus:border-fuchsia-500"
              />
            </div>

            <div className="flex items-center justify-end gap-3 border-t border-white/10 pt-4">
              <button
                type="button"
                onClick={() => setEditingItem(null)}
                className="px-5 py-2.5 rounded-xl border border-white/10 bg-white/[0.05] text-xs font-bold text-white/70 hover:text-white"
              >
                إلغاء
              </button>
              <button
                type="button"
                onClick={handleSaveMedia}
                disabled={saveLoading}
                className="px-7 py-2.5 rounded-xl bg-gradient-to-r from-purple-600 via-fuchsia-600 to-purple-700 text-xs font-bold text-white shadow-lg shadow-purple-900/50 hover:brightness-110 disabled:opacity-50"
              >
                {saveLoading ? "جارٍ الحفظ..." : "💾 حفظ التعديلات"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ========================================================================= */}
      {/* MEDIA ITEM DELETE MODAL                                                   */}
      {/* ========================================================================= */}
      {deletingItem && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/85 p-4 backdrop-blur-md animate-fadeIn">
          <div className="w-full max-w-md rounded-3xl border border-rose-500/30 bg-[#0D0B18] p-6 text-center space-y-4 shadow-2xl">
            <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-rose-500/20 text-2xl text-rose-400">
              🗑️
            </div>
            <h3 className="text-xl font-bold text-white">تأكيد حذف العمل</h3>
            <p className="text-xs text-white/70 leading-relaxed">
              هل أنت متأكد من حذف العمل <span className="font-bold text-rose-300">"{deletingItem.title_ar || deletingItem.title_en}"</span>؟
            </p>
            <div className="flex items-center justify-center gap-3 pt-2">
              <button
                type="button"
                onClick={() => setDeletingItem(null)}
                className="px-5 py-2.5 rounded-xl border border-white/10 bg-white/[0.05] text-xs font-bold text-white"
              >
                إلغاء
              </button>
              <button
                type="button"
                onClick={confirmDeleteMedia}
                disabled={deleteLoading}
                className="px-6 py-2.5 rounded-xl bg-rose-600 hover:bg-rose-700 text-xs font-bold text-white shadow-lg shadow-rose-900/50 disabled:opacity-50"
              >
                {deleteLoading ? "جارٍ الحذف..." : "نعم، احذف الآن"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ========================================================================= */}
      {/* 3. TAB: INDEXING & DISKS (فهرسة واختيار المجلدات والأقراص)                */}
      {/* ========================================================================= */}
      {activeTab === "admin-indexing" && (
        <div className="space-y-6 animate-fadeIn">
          <GlassCard className="p-6">
            <div className="flex items-center justify-between border-b border-white/10 pb-4">
              <div>
                <h2 className="text-xl font-bold text-white flex items-center gap-2">
                  <Icon name="disk" className="h-5 w-5 text-fuchsia-400" />
                  <span>الأقراص ووحدات التخزين المتصلة (Storage Disks)</span>
                </h2>
                <p className="text-xs text-white/50 mt-1">انقر على أي قرص لاختياره تلقائياً في مسار الفهرسة</p>
              </div>
              <button
                type="button"
                onClick={handleScanDisks}
                disabled={disksLoading}
                className="flex items-center gap-2 px-3.5 py-2 rounded-xl bg-white/[0.06] border border-white/10 hover:bg-white/10 text-xs font-bold text-white transition disabled:opacity-50"
              >
                <Icon name="spark" className={`h-3.5 w-3.5 text-fuchsia-400 ${disksLoading ? "animate-spin" : ""}`} />
                <span>{disksLoading ? "جارٍ الفحص..." : "تحديث الأقراص"}</span>
              </button>
            </div>

            <div className="mt-5 grid gap-4 md:grid-cols-3">
              {disks.map((disk) => {
                const usedGB = (disk.used_space / 1073741824).toFixed(1);
                const totalGB = (disk.total_space / 1073741824).toFixed(1);
                const freeGB = (disk.free_space / 1073741824).toFixed(1);
                const percent = disk.used_percent || (disk.total_space > 0 ? (disk.used_space / disk.total_space) * 100 : 0);

                return (
                  <div
                    key={disk.disk_letter}
                    onClick={() => setIndexRoot(`${disk.disk_letter}:/`)}
                    className="cursor-pointer group p-4 rounded-2xl bg-white/[0.03] border border-white/10 hover:border-fuchsia-500/50 hover:bg-fuchsia-950/20 transition duration-300"
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex items-center gap-3">
                        <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br from-purple-800 to-fuchsia-900 text-white font-black text-lg shadow-md group-hover:scale-105 transition">
                          {disk.disk_letter}:
                        </div>
                        <div>
                          <p className="text-sm font-bold text-white group-hover:text-fuchsia-300 transition">
                            {disk.disk_label || `القرص ${disk.disk_letter}:`}
                          </p>
                          <p className="text-[11px] text-white/50">
                            متاح: <span className="text-emerald-400 font-semibold">{freeGB} GB</span> من {totalGB} GB
                          </p>
                        </div>
                      </div>
                      <span className="text-xs font-mono font-bold text-fuchsia-300 bg-fuchsia-500/10 px-2 py-1 rounded-lg">
                        {percent.toFixed(0)}%
                      </span>
                    </div>

                    <div className="mt-4 h-2 w-full rounded-full bg-white/10 overflow-hidden">
                      <div
                        className={`h-full rounded-full transition-all duration-500 ${
                          percent > 90 ? "bg-rose-500" : percent > 75 ? "bg-amber-500" : "bg-gradient-to-r from-purple-500 to-fuchsia-500"
                        }`}
                        style={{ width: `${Math.min(percent, 100)}%` }}
                      ></div>
                    </div>

                    <div className="mt-3 flex items-center justify-between text-[11px] text-white/40">
                      <span>مستهلك: {usedGB} GB</span>
                      <span className="text-fuchsia-400 group-hover:underline">اختر هذا القرص ↵</span>
                    </div>
                  </div>
                );
              })}
            </div>
          </GlassCard>

          <GlassCard className="p-6 border-fuchsia-500/30">
            <h2 className="text-xl font-bold text-white flex items-center gap-2">
              <Icon name="search" className="h-5 w-5 text-fuchsia-400" />
              <span>تحديد مسار المجلد وبدء الفهرسة الذكية</span>
            </h2>
            <p className="mt-1 text-sm text-white/60">
              اختر مساراً جاهزاً بالأسفل أو اكتب مسار أي مجلد أو قرص لبدء تنظيمه وفهرسته في ثوانٍ:
            </p>

            <div className="mt-4 flex flex-wrap gap-2">
              <span className="text-xs text-white/40 self-center ml-2">مسارات سريعة:</span>
              <button
                type="button"
                onClick={() => setIndexRoot("C:/Users/mousa/Desktop/project/NEXORA/server/testdata/test_media_root")}
                className="px-3 py-1.5 rounded-xl bg-fuchsia-600/20 border border-fuchsia-500/30 text-xs font-semibold text-fuchsia-300 hover:bg-fuchsia-600/30 transition"
              >
                📁 مجلد بيانات الاختبار التجريبية (test_media_root)
              </button>
              <button
                type="button"
                onClick={() => setIndexRoot("D:/Media")}
                className="px-3 py-1.5 rounded-xl bg-white/[0.05] border border-white/10 text-xs font-semibold text-white/80 hover:bg-white/10 transition"
              >
                💾 D:/Media
              </button>
              <button
                type="button"
                onClick={() => setIndexRoot("E:/Anime")}
                className="px-3 py-1.5 rounded-xl bg-white/[0.05] border border-white/10 text-xs font-semibold text-white/80 hover:bg-white/10 transition"
              >
                💾 E:/Anime
              </button>
            </div>

            <div className="mt-4 flex flex-col md:flex-row gap-3">
              <input
                value={indexRoot}
                onChange={(e) => setIndexRoot(e.target.value)}
                placeholder="مثال: D:/Media أو C:/Downloads/Anime"
                dir="ltr"
                className="flex-1 rounded-2xl border border-white/10 bg-black/40 px-4 py-3.5 text-sm text-white outline-none placeholder:text-white/30 focus:border-fuchsia-500 focus:ring-1 focus:ring-fuchsia-500 transition font-mono"
              />
              <button
                type="button"
                onClick={handleIndex}
                disabled={indexState === "loading" || !indexRoot.trim()}
                className="inline-flex items-center justify-center gap-2 rounded-2xl bg-gradient-to-r from-purple-600 via-fuchsia-600 to-purple-700 px-7 py-3.5 text-sm font-bold text-white shadow-lg shadow-purple-900/50 hover:brightness-110 transition disabled:opacity-50"
              >
                <Icon name="search" className={`h-4 w-4 ${indexState === "loading" ? "animate-spin" : ""}`} />
                <span>{indexState === "loading" ? "جارٍ الفهرسة الذكية..." : "🚀 بدء الفهرسة الشاملة"}</span>
              </button>
            </div>

            {indexState === "success" && indexResult && (
              <div className="mt-6 p-5 rounded-2xl bg-emerald-950/20 border border-emerald-500/30 space-y-4 animate-fadeIn">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2 text-emerald-400 font-bold text-sm">
                    <span className="h-2.5 w-2.5 rounded-full bg-emerald-400"></span>
                    <span>تمت الفهرسة ومعالجة التسميات بنجاح!</span>
                  </div>
                </div>

                <div className="grid gap-3 sm:grid-cols-4">
                  <div className="p-3.5 rounded-xl bg-black/30 border border-white/5 text-center">
                    <p className="text-2xl font-black text-white">{indexResult.scanned || 0}</p>
                    <p className="text-xs text-white/50 mt-1">ملف تم مسحه</p>
                  </div>
                  <div className="p-3.5 rounded-xl bg-black/30 border border-white/5 text-center">
                    <p className="text-2xl font-black text-emerald-400">{indexResult.imported || 0}</p>
                    <p className="text-xs text-white/50 mt-1">أُدخلت لقاعدة البيانات</p>
                  </div>
                  <div className="p-3.5 rounded-xl bg-black/30 border border-white/5 text-center">
                    <p className="text-2xl font-black text-cyan-400">{indexResult.searchSync?.indexed || 0}</p>
                    <p className="text-xs text-white/50 mt-1">مفهرسة بالبحث الفوري</p>
                  </div>
                  <div className="p-3.5 rounded-xl bg-black/30 border border-white/5 text-center">
                    <p className="text-2xl font-black text-fuchsia-400">{indexResult.inspected || 0}</p>
                    <p className="text-xs text-white/50 mt-1">مفحوصة تقنياً (ffprobe)</p>
                  </div>
                </div>
              </div>
            )}
          </GlassCard>
        </div>
      )}

      {/* ========================================================================= */}
      {/* 4. TAB: QUALITY & INTEGRITY                                               */}
      {/* ========================================================================= */}
      {activeTab === "admin-quality" && (
        <div className="space-y-6 animate-fadeIn">
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <GlassCard className="p-5 border-purple-500/20">
              <p className="text-xs font-bold text-white/50">إجمالي الأعمال والوسائط</p>
              <p className="mt-2 text-3xl font-black text-white">{qualityReport?.total_media || 0}</p>
              <p className="text-xs text-purple-400 mt-1">{qualityReport?.total_files || 0} ملف فيديو مسجل</p>
            </GlassCard>

            <GlassCard className="p-5 border-cyan-500/20">
              <p className="text-xs font-bold text-white/50">إجمالي الحجم المخزن</p>
              <p className="mt-2 text-3xl font-black text-cyan-300">{formatBytes(qualityReport?.total_size_bytes)}</p>
              <p className="text-xs text-white/40 mt-1">عبر كافة الأقراص</p>
            </GlassCard>

            <GlassCard className="p-5 border-amber-500/20">
              <p className="text-xs font-bold text-white/50">الحلقات المفقودة (ثغرات)</p>
              <p className="mt-2 text-3xl font-black text-amber-400">{qualityReport?.missing_episodes_count || 0}</p>
              <p className="text-xs text-amber-300/70 mt-1">حلقات تحتاج تحميل أو نقل</p>
            </GlassCard>

            <GlassCard className="p-5 border-rose-500/20">
              <p className="text-xs font-bold text-white/50">المساحة المهدرة (تكرارات)</p>
              <p className="mt-2 text-3xl font-black text-rose-400">{formatBytes(qualityReport?.total_wasted_bytes)}</p>
              <p className="text-xs text-rose-300/70 mt-1">{qualityReport?.duplicate_groups_count || 0} مجموعة مكررة</p>
            </GlassCard>
          </div>

          <GlassCard className="p-6">
            <div className="flex flex-wrap items-center justify-between gap-4 border-b border-white/10 pb-4">
              <div className="flex gap-2">
                <button
                  onClick={() => setQualityTab("missing")}
                  className={`px-4 py-2 rounded-xl text-xs font-bold transition ${
                    qualityTab === "missing" ? "bg-purple-600 text-white" : "bg-white/[0.05] text-white/60 hover:text-white"
                  }`}
                >
                  ⚠️ الحلقات المفقودة ({qualityReport?.missing_episodes_count || 0})
                </button>
                <button
                  onClick={() => setQualityTab("duplicates")}
                  className={`px-4 py-2 rounded-xl text-xs font-bold transition ${
                    qualityTab === "duplicates" ? "bg-purple-600 text-white" : "bg-white/[0.05] text-white/60 hover:text-white"
                  }`}
                >
                  🧹 كاشف التكرارات ({qualityReport?.duplicate_groups_count || 0})
                </button>
                <button
                  onClick={() => setQualityTab("corrupted")}
                  className={`px-4 py-2 rounded-xl text-xs font-bold transition ${
                    qualityTab === "corrupted" ? "bg-purple-600 text-white" : "bg-white/[0.05] text-white/60 hover:text-white"
                  }`}
                >
                  🛡️ الملفات المعطوبة ({qualityReport?.corrupted_files_count || 0})
                </button>
              </div>

              <button
                type="button"
                onClick={handleTriggerChecksums}
                disabled={checksumState === "loading"}
                className="flex items-center gap-2 px-4 py-2 rounded-xl bg-gradient-to-r from-purple-700 to-fuchsia-700 text-xs font-bold text-white disabled:opacity-50"
              >
                <span>{checksumState === "loading" ? "جارٍ الحساب..." : "حساب بصمات SHA-256 للمكتبة"}</span>
              </button>
            </div>

            {qualityTab === "missing" && (
              <div className="mt-5 space-y-3">
                {qualityReport?.missing_episodes?.length > 0 ? (
                  qualityReport.missing_episodes.map((item, idx) => (
                    <div key={idx} className="flex items-center justify-between p-4 rounded-2xl bg-white/[0.02] border border-amber-500/20">
                      <div className="flex items-center gap-3">
                        <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-amber-500/10 text-amber-400 font-bold text-xs">
                          E{item.missing_episode_number}
                        </div>
                        <div>
                          <p className="text-sm font-bold text-white">{item.show_title_ar || item.show_title_en} — الموسم {item.season_number}</p>
                          <p className="text-xs text-amber-300/80">حلقة ناقصة: {item.missing_episode_number}</p>
                        </div>
                      </div>
                      <div className="font-mono text-xs text-white/40 bg-black/40 px-3 py-1.5 rounded-xl" dir="ltr">
                        {item.suggested_filename}
                      </div>
                    </div>
                  ))
                ) : (
                  <div className="p-8 text-center text-white/50">✅ تسلسل الحلقات مكتمل ولا توجد فجوات!</div>
                )}
              </div>
            )}

            {qualityTab === "duplicates" && (
              <div className="mt-5 space-y-4">
                {qualityReport?.duplicate_groups?.length > 0 ? (
                  qualityReport.duplicate_groups.map((group, idx) => (
                    <div key={idx} className="p-4 rounded-2xl bg-white/[0.02] border border-white/10 space-y-2">
                      <div className="flex items-center justify-between text-xs">
                        <span className="font-mono text-fuchsia-400">SHA-256: {group.checksum?.substring(0, 16)}...</span>
                        <span className="text-rose-400 font-bold">الحجم: {formatBytes(group.file_size)}</span>
                      </div>
                      <div className="space-y-1">
                        {group.files?.map((f, fIdx) => (
                          <div key={fIdx} className="p-2.5 rounded-lg bg-black/40 text-xs font-mono text-white/70" dir="ltr">
                            {f.file_path}
                          </div>
                        ))}
                      </div>
                    </div>
                  ))
                ) : (
                  <div className="p-8 text-center text-white/50">✅ لا توجد ملفات مكررة مطابقة في المكتبة!</div>
                )}
              </div>
            )}

            {qualityTab === "corrupted" && (
              <div className="mt-5 space-y-3">
                {qualityReport?.corrupted_files?.length > 0 ? (
                  qualityReport.corrupted_files.map((file, idx) => (
                    <div key={idx} className="p-4 rounded-2xl bg-rose-500/10 border border-rose-500/30 space-y-1 text-xs">
                      <p className="font-bold text-rose-300">{file.show_title}</p>
                      <p className="font-mono text-white/60" dir="ltr">{file.file_path}</p>
                      <p className="text-rose-400">{file.error_output}</p>
                    </div>
                  ))
                ) : (
                  <div className="p-8 text-center text-white/50">✅ تم فحص تدفقات الفيديو ولا توجد ملفات معطوبة!</div>
                )}
              </div>
            )}
          </GlassCard>
        </div>
      )}

      {/* ========================================================================= */}
      {/* 5. TAB: MIGRATION & REORGANIZATION                                        */}
      {/* ========================================================================= */}
      {activeTab === "admin-migration" && (
        <section className="grid gap-6 xl:grid-cols-2 animate-fadeIn">
          <GlassCard className="p-6">
            <h2 className="text-xl font-bold text-white mb-2">معاينة خطة التنظيم الفيزيائي</h2>
            <div className="space-y-3">
              <input
                value={previewRoot}
                onChange={(e) => setPreviewRoot(e.target.value)}
                placeholder="D:/Downloads/Unorganized"
                dir="ltr"
                className="w-full rounded-xl border border-white/10 bg-black/40 p-3 text-xs text-white font-mono"
              />
              <button
                type="button"
                onClick={handlePreview}
                disabled={previewState === "loading"}
                className="px-5 py-2.5 rounded-xl bg-purple-600 text-xs font-bold text-white disabled:opacity-50"
              >
                {previewState === "loading" ? "جارٍ التحليل..." : "إنشاء المعاينة"}
              </button>
            </div>
            {previewResult && (
              <div className="mt-4 p-4 rounded-xl bg-black/40 text-xs space-y-2">
                <p>إجمالي الملفات: {previewResult.entries?.length || 0}</p>
                <p className="text-emerald-400">سيتم نقلها: {previewResult.moveCount || 0}</p>
                <p className="text-amber-400">مكررة: {previewResult.duplicateCount || 0}</p>
              </div>
            )}
          </GlassCard>

          <GlassCard className="p-6">
            <h2 className="text-xl font-bold text-white mb-2">تنفيذ النقل الآمن مع البصمة</h2>
            <div className="space-y-3">
              <input
                value={copyTarget}
                onChange={(e) => setCopyTarget(e.target.value)}
                placeholder="D:/Media_Organized"
                dir="ltr"
                className="w-full rounded-xl border border-white/10 bg-black/40 p-3 text-xs text-white font-mono"
              />
              <textarea
                rows={4}
                value={copySources}
                onChange={(e) => setCopySources(e.target.value)}
                placeholder="مسار كل ملف في سطر..."
                dir="ltr"
                className="w-full rounded-xl border border-white/10 bg-black/40 p-3 text-xs text-white font-mono"
              />
              <button
                type="button"
                onClick={handleCopy}
                disabled={copyState === "loading"}
                className="w-full py-3 rounded-xl bg-emerald-600 text-xs font-bold text-white disabled:opacity-50"
              >
                {copyState === "loading" ? "جارٍ النقل..." : "بدء النقل الآمن"}
              </button>
            </div>
          </GlassCard>
        </section>
      )}

      {/* ========================================================================= */}
      {/* 6. TAB: SYSTEM OVERVIEW                                                   */}
      {/* ========================================================================= */}
      {activeTab === "admin-overview" && (
        <div className="space-y-6 animate-fadeIn">
          <div className="grid gap-4 md:grid-cols-4">
            <GlassCard className="p-5">
              <p className="text-xs text-white/50">خادم الـ API</p>
              <p className="mt-2 text-2xl font-bold text-emerald-400">● يعمل</p>
            </GlassCard>
            <GlassCard className="p-5">
              <p className="text-xs text-white/50">قاعدة البيانات</p>
              <p className="mt-2 text-2xl font-bold text-emerald-400">● متصلة</p>
            </GlassCard>
            <GlassCard className="p-5">
              <p className="text-xs text-white/50">محرك البحث</p>
              <p className="mt-2 text-2xl font-bold text-cyan-400">● Meilisearch</p>
            </GlassCard>
            <GlassCard className="p-5">
              <p className="text-xs text-white/50">مراقب المجلدات</p>
              <p className="mt-2 text-2xl font-bold text-fuchsia-400">● نشط</p>
            </GlassCard>
          </div>

          <GlassCard className="p-6">
            <h2 className="text-lg font-bold text-white mb-3">إجراءات سريعة</h2>
            <div className="flex gap-3">
              <button
                type="button"
                onClick={onSyncIndex}
                className="px-4 py-2.5 rounded-xl bg-fuchsia-600/30 text-xs font-bold text-fuchsia-200 hover:bg-fuchsia-600/50"
              >
                مزامنة محرك البحث الفوري
              </button>
              <button
                type="button"
                onClick={handleScanDisks}
                className="px-4 py-2.5 rounded-xl bg-white/10 text-xs font-bold text-white hover:bg-white/20"
              >
                تحديث الأقراص
              </button>
            </div>
          </GlassCard>
        </div>
      )}
    </div>
  );
}
