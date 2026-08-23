import { useEffect, useMemo, useState } from "react";
import GlassCard from "../../components/GlassCard.jsx";
import Icon from "../../components/Icon.jsx";
import {
  classifyOriginsFromFolders,
  createMediaItem,
  deleteMediaItem,
  enrichMedia,
  getCategories,
  getMediaList,
  getQualityReport,
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
    title_ar: "", title_en: "", type: "movie", category_slug: "movies",
    plot_ar: "", plot_en: "", release_year: 2024, rating: 8.5,
    poster_path: "", banner_path: "", genres: [],
  });
  const [enrichLoading, setEnrichLoading] = useState(false);
  const [bulkEnrichStatus, setBulkEnrichStatus] = useState({ state: "idle", completed: 0, failed: 0, total: 0 });
  const [originClassifyState, setOriginClassifyState] = useState("idle");
  const [saveLoading, setSaveLoading] = useState(false);
  const [customGenreInput, setCustomGenreInput] = useState("");

  // Delete
  const [deletingItem, setDeletingItem] = useState(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  useEffect(() => {
    getCategories()
      .then((data) => { if (data.categories?.length > 0) setCategories(data.categories); })
      .catch(() => {});
  }, []);

  useEffect(() => { loadMediaItems(); }, [selectedCategory, managerSort, managerSearch]);

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
        if (!itemGenres.some((g) => g.includes(selectedGenre) || selectedGenre.includes(g))) return false;
      }
      if (selectedCountry !== "all") {
        const countryTerms = COUNTRY_TAGS[selectedCountry] || [];
        const searchable = (item.genres || []).join(" ").toLocaleLowerCase();
        if (!countryTerms.some((term) => searchable.includes(term.toLocaleLowerCase()))) return false;
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
      title_ar: item.title_ar || "", title_en: item.title_en || "",
      type: item.type || "movie",
      category_slug: item.category_slug || (selectedCategory !== "all" ? selectedCategory : "movies"),
      plot_ar: item.plot_ar || "", plot_en: item.plot_en || "",
      release_year: item.release_year || 2023, rating: item.rating || 8.0,
      poster_path: item.poster_path || "", banner_path: item.banner_path || "",
      genres: item.genres || [],
    });
  }

  function handleOpenAddNew() {
    setIsNewItem(true);
    setEditingItem({ id: "new" });
    setModalForm({
      title_ar: "", title_en: "",
      type: selectedCategory === "movies" ? "movie" : selectedCategory === "anime" ? "anime" : "series",
      category_slug: selectedCategory !== "all" ? selectedCategory : "movies",
      plot_ar: "", plot_en: "", release_year: 2024, rating: 8.5,
      poster_path: "", banner_path: "", genres: [],
    });
  }

  async function handleSaveMedia() {
    if (!modalForm.title_en.trim() && !modalForm.title_ar.trim()) {
      alert("يرجى إدخال اسم العمل");
      return;
    }
    setSaveLoading(true);
    try {
      if (isNewItem) { await createMediaItem(modalForm); }
      else { await updateMediaItem(editingItem.id, modalForm); }
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
          genres: enriched.genres?.length > 0 ? enriched.genres : prev.genres,
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
    if (targets.length === 0) { alert("لا توجد أعمال مطابقة للفلاتر الحالية لتحديثها."); return; }
    if (!window.confirm(`سيتم جلب وتوحيد بيانات ${targets.length} عملاً ظاهراً الآن من TMDB/MAL. قد يستغرق ذلك عدة دقائق. هل تريد المتابعة؟`)) return;
    let completed = 0, failed = 0;
    setBulkEnrichStatus({ state: "running", completed, failed, total: targets.length });
    for (const item of targets) {
      try { await enrichMedia(item.id); completed += 1; } catch { failed += 1; }
      setBulkEnrichStatus({ state: "running", completed, failed, total: targets.length });
    }
    setBulkEnrichStatus({ state: "done", completed, failed, total: targets.length });
    loadMediaItems();
  }

  async function handleClassifyOrigins() {
    if (!window.confirm("سيقرأ النظام مجلدات مكتبتك ويضيف تلقائياً وسوماً مثل عربي، تركي، وكوري. لن يحذف أي تصنيف موجود. هل تريد المتابعة؟")) return;
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
      genres: prev.genres.includes(genre) ? prev.genres.filter((g) => g !== genre) : [...prev.genres, genre],
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
      <div className="flex flex-wrap items-center gap-2">
        <button type="button" onClick={() => setSelectedCategory("all")}
          className={`px-4 py-2.5 rounded-2xl text-xs font-bold transition ${selectedCategory === "all" ? "bg-gradient-to-r from-purple-600 to-fuchsia-600 text-white shadow-lg shadow-purple-900/50" : "bg-white/[0.04] text-white/70 hover:bg-white/10 hover:text-white"}`}>
          🌟 كافة الأقسام ({totalMediaCount})
        </button>
        {categories.map((cat) => (
          <button key={cat.slug} type="button" onClick={() => setSelectedCategory(cat.slug)}
            className={`px-4 py-2.5 rounded-2xl text-xs font-bold transition ${selectedCategory === cat.slug ? "bg-gradient-to-r from-purple-600 to-fuchsia-600 text-white shadow-lg shadow-purple-900/50" : "bg-white/[0.04] text-white/70 hover:bg-white/10 hover:text-white"}`}>
            {cat.name_ar || cat.nameAr}
          </button>
        ))}
      </div>

      {/* Filter Control Center */}
      <GlassCard className="p-5 space-y-4">
        {/* Search, Sort, Actions */}
        <div className="flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3">
          <div className="relative flex-1">
            <input type="text" value={managerSearch} onChange={(e) => setManagerSearch(e.target.value)}
              placeholder="بحث سريع بالعنوان العربي أو الإنجليزي أو الكلمات المفتاحية..."
              className="w-full rounded-2xl border border-white/10 bg-black/40 px-4 py-3 pl-10 text-sm text-white outline-none placeholder:text-white/30 focus:border-fuchsia-500 transition" />
            <Icon name="search" className="absolute left-3.5 top-3.5 h-4 w-4 text-white/40" />
          </div>

          <div className="flex items-center gap-2">
            <span className="text-xs text-white/50 whitespace-nowrap">الترتيب:</span>
            <select value={managerSort} onChange={(e) => setManagerSort(e.target.value)}
              className="rounded-2xl border border-white/10 bg-black/50 px-3.5 py-3 text-xs font-bold text-white outline-none focus:border-fuchsia-500">
              <option value="latest">الأحدث إضافة</option>
              <option value="rating">الأعلى تقييماً ★</option>
              <option value="year">سنة الإنتاج 📅</option>
              <option value="title">أبجدي أ-ي</option>
            </select>
          </div>

          <button type="button" onClick={handleOpenAddNew}
            className="flex items-center justify-center gap-2 rounded-2xl bg-gradient-to-r from-emerald-600 via-teal-600 to-emerald-700 px-6 py-3 text-xs font-bold text-white shadow-lg shadow-emerald-900/40 hover:brightness-110 transition whitespace-nowrap">
            <span>➕ إضافة عمل جديد</span>
          </button>
          <button type="button" onClick={handleBulkEnrich}
            disabled={bulkEnrichStatus.state === "running" || filteredMediaItems.length === 0}
            className="flex items-center justify-center gap-2 rounded-2xl border border-fuchsia-400/30 bg-fuchsia-500/10 px-5 py-3 text-xs font-bold text-fuchsia-100 transition hover:bg-fuchsia-500/20 disabled:cursor-not-allowed disabled:opacity-50 whitespace-nowrap">
            <span>{bulkEnrichStatus.state === "running" ? `جارٍ التحديث ${bulkEnrichStatus.completed}/${bulkEnrichStatus.total}` : "✨ إصلاح بيانات الأعمال الظاهرة"}</span>
          </button>
          <button type="button" onClick={handleClassifyOrigins}
            disabled={originClassifyState === "running"}
            className="flex items-center justify-center gap-2 rounded-2xl border border-cyan-400/30 bg-cyan-500/10 px-5 py-3 text-xs font-bold text-cyan-100 transition hover:bg-cyan-500/20 disabled:cursor-not-allowed disabled:opacity-50 whitespace-nowrap">
            <span>{originClassifyState === "running" ? "جارٍ قراءة المجلدات..." : "🗂️ تصنيف البلد من المجلدات"}</span>
          </button>
        </div>

        {bulkEnrichStatus.state !== "idle" && (
          <div className={`rounded-xl border px-3 py-2 text-xs ${bulkEnrichStatus.state === "done" ? "border-emerald-400/20 bg-emerald-500/10 text-emerald-200" : "border-fuchsia-400/20 bg-fuchsia-500/10 text-fuchsia-100"}`}>
            {bulkEnrichStatus.state === "running" ? `يتم الآن توحيد البيانات: ${bulkEnrichStatus.completed} مكتمل، ${bulkEnrichStatus.failed} تعذّر.` : `اكتمل تحديث البيانات: ${bulkEnrichStatus.completed} عمل، وتعذّر ${bulkEnrichStatus.failed}. الآن ستعمل فلاتر البلد والنوع تلقائياً.`}
          </div>
        )}
        {originClassifyState.startsWith("done:") && <p className="text-xs text-cyan-200">تمت قراءة مجلدات المكتبة وتصنيف {originClassifyState.split(":")[1]} عملاً.</p>}

        {/* Dropdown Filters */}
        <div className="grid gap-3 sm:grid-cols-2 md:grid-cols-4 pt-3 border-t border-white/10">
          <div>
            <label className="block text-[11px] font-bold text-white/50 mb-1">📅 سنة الإنتاج</label>
            <select value={selectedYear} onChange={(e) => setSelectedYear(e.target.value)}
              className="w-full rounded-xl border border-white/10 bg-black/40 px-3 py-2 text-xs text-white outline-none focus:border-fuchsia-500">
              {YEARS_LIST.map((y) => <option key={y.id} value={y.id}>{y.label}</option>)}
            </select>
          </div>
          <div>
            <label className="block text-[11px] font-bold text-white/50 mb-1">★ التقييم</label>
            <select value={selectedRating} onChange={(e) => setSelectedRating(e.target.value)}
              className="w-full rounded-xl border border-white/10 bg-black/40 px-3 py-2 text-xs text-white outline-none focus:border-fuchsia-500">
              {RATINGS_LIST.map((r) => <option key={r.id} value={r.id}>{r.label}</option>)}
            </select>
          </div>
          <div>
            <label className="block text-[11px] font-bold text-white/50 mb-1">🌍 جهة الإنتاج / البلد</label>
            <select value={selectedCountry} onChange={(e) => setSelectedCountry(e.target.value)}
              className="w-full rounded-xl border border-white/10 bg-black/40 px-3 py-2 text-xs text-white outline-none focus:border-fuchsia-500">
              {COUNTRIES_LIST.map((c) => <option key={c.id} value={c.id}>{c.label}</option>)}
            </select>
          </div>
          <div>
            <label className="block text-[11px] font-bold text-white/50 mb-1">📺 الجودة والدقة</label>
            <select value={selectedQuality} onChange={(e) => setSelectedQuality(e.target.value)}
              className="w-full rounded-xl border border-white/10 bg-black/40 px-3 py-2 text-xs text-white outline-none focus:border-fuchsia-500">
              {QUALITIES_LIST.map((q) => <option key={q.id} value={q.id}>{q.label}</option>)}
            </select>
          </div>
        </div>

        {/* Genre Pills */}
        <div className="flex flex-wrap items-center gap-1.5 pt-3 border-t border-white/10">
          <span className="text-[11px] text-white/40 ml-2">نوع وتصنيف العمل (Genre):</span>
          <button type="button" onClick={() => setSelectedGenre("all")}
            className={`px-3 py-1 rounded-xl text-xs font-bold transition ${selectedGenre === "all" ? "bg-fuchsia-600 text-white shadow" : "bg-white/[0.05] text-white/60 hover:text-white"}`}>
            الكل
          </button>
          {ALL_GENRES.map((g) => (
            <button key={g} type="button" onClick={() => setSelectedGenre(g)}
              className={`px-2.5 py-1 rounded-xl text-xs font-medium transition ${selectedGenre === g ? "bg-fuchsia-600 text-white shadow" : "bg-white/[0.04] text-white/60 hover:bg-white/10 hover:text-white"}`}>
              {g}
            </button>
          ))}
        </div>
      </GlassCard>

      {/* Media Grid */}
      {mediaLoading ? (
        <div className="p-16 text-center text-white/60">
          <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-fuchsia-500 border-t-transparent" />
          <p className="mt-3 text-sm font-bold">جارٍ تحميل الأعمال...</p>
        </div>
      ) : filteredMediaItems.length === 0 ? (
        <GlassCard className="p-16 text-center">
          <p className="text-4xl">🎬</p>
          <h3 className="mt-3 text-lg font-bold text-white">لا توجد أعمال تطابق الفلاتر المحددة</h3>
          <p className="mt-1 text-xs text-white/50 max-w-md mx-auto">جرب تغيير خيارات الفلترة أو إضافة عمل جديد بنقرة زر.</p>
          <button type="button" onClick={handleOpenAddNew}
            className="mt-5 inline-flex items-center gap-2 rounded-2xl bg-fuchsia-600 px-5 py-2.5 text-xs font-bold text-white hover:bg-fuchsia-500 transition">
            <span>➕ إضافة عمل الآن</span>
          </button>
        </GlassCard>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
          {filteredMediaItems.map((item) => {
            const poster = item.poster_path || "/images/tokyo_ghoul_hero.png";
            return (
              <div key={item.id} className="group relative flex flex-col justify-between overflow-hidden rounded-2xl border border-white/10 bg-[#0E0C1A] hover:border-fuchsia-500/50 hover:shadow-xl hover:shadow-purple-950/50 transition duration-300">
                <div className="relative aspect-[2/3] w-full overflow-hidden bg-black/60">
                  <img src={poster} alt={item.title_en} className="h-full w-full object-cover group-hover:scale-105 transition duration-500"
                    onError={(e) => { e.target.src = "/images/tokyo_ghoul_hero.png"; }} />
                  <div className="absolute inset-0 bg-gradient-to-t from-[#0E0C1A] via-transparent to-black/40" />
                  <div className="absolute top-2.5 right-2.5 left-2.5 flex items-center justify-between">
                    <span className="flex items-center gap-1 rounded-lg bg-black/70 px-2 py-1 text-[11px] font-bold text-amber-300 backdrop-blur-md">
                      ★ {item.rating ? item.rating.toFixed(1) : "8.0"}
                    </span>
                    <span className="rounded-lg bg-black/70 px-2 py-1 text-[11px] font-mono font-bold text-white/80 backdrop-blur-md">
                      {item.release_year || "2023"}
                    </span>
                  </div>
                  <div className="absolute inset-0 flex items-center justify-center gap-2 bg-black/75 opacity-0 group-hover:opacity-100 transition-opacity backdrop-blur-sm">
                    <button type="button" onClick={() => handleOpenEdit(item)} className="flex h-10 w-10 items-center justify-center rounded-xl bg-fuchsia-600 text-white shadow-lg hover:scale-110 transition" title="تعديل العمل">✏️</button>
                    <button type="button" onClick={() => setDeletingItem(item)} className="flex h-10 w-10 items-center justify-center rounded-xl bg-rose-600 text-white shadow-lg hover:scale-110 transition" title="حذف العمل">🗑️</button>
                  </div>
                </div>
                <div className="p-3.5 space-y-2 flex-1 flex flex-col justify-between">
                  <div>
                    <h4 className="font-bold text-white text-sm line-clamp-1 group-hover:text-fuchsia-300 transition">{item.title_ar || item.title_en}</h4>
                    {item.title_ar && item.title_en && item.title_ar !== item.title_en && (
                      <p className="text-[11px] text-white/40 line-clamp-1 font-mono">{item.title_en}</p>
                    )}
                  </div>
                  <div className="flex flex-wrap gap-1">
                    {item.genres?.slice(0, 2).map((genre, idx) => (
                      <span key={idx} className="rounded-md bg-white/[0.06] border border-white/5 px-1.5 py-0.5 text-[10px] text-white/70">{genre}</span>
                    ))}
                  </div>
                  <div className="flex items-center justify-between border-t border-white/5 pt-2 text-[11px] text-white/40">
                    <span>{item.file_count || 1} ملف / حلقة</span>
                    <button type="button" onClick={() => handleOpenEdit(item)} className="text-fuchsia-400 font-bold hover:underline">تعديل ↵</button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Edit/Add Modal */}
      {editingItem && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/85 p-3 sm:p-6 backdrop-blur-md overflow-y-auto animate-fadeIn">
          <div className="relative w-full max-w-4xl rounded-3xl border border-fuchsia-500/30 bg-[#0D0B18] p-6 sm:p-8 shadow-2xl space-y-6 max-h-[92vh] overflow-y-auto">
            <div className="flex items-center justify-between border-b border-white/10 pb-4">
              <div>
                <h3 className="text-2xl font-black text-white">{isNewItem ? "➕ إضافة عمل سينمائي جديد" : "✏️ تعديل بيانات العمل والغلاف"}</h3>
                <p className="text-xs text-white/50 mt-1">تعديل الاسم العربي والإنجليزي، البوسترات، القصة، والوسوم</p>
              </div>
              <div className="flex items-center gap-2">
                {!isNewItem && (
                  <button type="button" onClick={handleEnrichInModal} disabled={enrichLoading}
                    className="flex items-center gap-2 px-3.5 py-2 rounded-xl bg-gradient-to-r from-purple-600 to-fuchsia-600 text-xs font-bold text-white hover:brightness-110 transition disabled:opacity-50">
                    <span>{enrichLoading ? "جارٍ الجلب..." : "✨ جلب فوري (TMDB/MAL)"}</span>
                  </button>
                )}
                <button type="button" onClick={() => setEditingItem(null)} className="rounded-xl border border-white/10 p-2 text-white/60 hover:text-white">✕</button>
              </div>
            </div>

            <div className="grid gap-5 sm:grid-cols-2">
              <div>
                <label className="block text-xs font-bold text-white/70 mb-1.5">العنوان باللغة العربية</label>
                <input type="text" value={modalForm.title_ar} onChange={(e) => setModalForm({ ...modalForm, title_ar: e.target.value })} placeholder="مثال: هجوم العمالقة أو إنسبشن" className="w-full rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-sm text-white outline-none focus:border-fuchsia-500" />
              </div>
              <div>
                <label className="block text-xs font-bold text-white/70 mb-1.5">العنوان باللغة الإنجليزية</label>
                <input type="text" value={modalForm.title_en} onChange={(e) => setModalForm({ ...modalForm, title_en: e.target.value })} placeholder="e.g. Attack on Titan" dir="ltr" className="w-full rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-sm text-white outline-none focus:border-fuchsia-500 font-mono" />
              </div>
              <div>
                <label className="block text-xs font-bold text-white/70 mb-1.5">القسم والمكتبة</label>
                <select value={modalForm.category_slug} onChange={(e) => setModalForm({ ...modalForm, category_slug: e.target.value })} className="w-full rounded-xl border border-white/10 bg-black/50 px-3.5 py-2.5 text-sm text-white outline-none focus:border-fuchsia-500">
                  {categories.map((c) => <option key={c.slug} value={c.slug}>{c.name_ar || c.nameAr}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-xs font-bold text-white/70 mb-1.5">نوع العمل</label>
                <select value={modalForm.type} onChange={(e) => setModalForm({ ...modalForm, type: e.target.value })} className="w-full rounded-xl border border-white/10 bg-black/50 px-3.5 py-2.5 text-sm text-white outline-none focus:border-fuchsia-500">
                  <option value="movie">فيلم (Movie)</option>
                  <option value="series">مسلسل (Series)</option>
                  <option value="anime">أنمي (Anime)</option>
                </select>
              </div>
              <div>
                <label className="block text-xs font-bold text-white/70 mb-1.5">سنة الإنتاج</label>
                <input type="number" value={modalForm.release_year} onChange={(e) => setModalForm({ ...modalForm, release_year: parseInt(e.target.value) || 0 })} className="w-full rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-sm text-white outline-none focus:border-fuchsia-500" />
              </div>
              <div>
                <label className="block text-xs font-bold text-white/70 mb-1.5">التقييم (من 10)</label>
                <input type="number" step="0.1" min="0" max="10" value={modalForm.rating} onChange={(e) => setModalForm({ ...modalForm, rating: parseFloat(e.target.value) || 0 })} className="w-full rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-sm text-white outline-none focus:border-fuchsia-500" />
              </div>
              <div>
                <label className="block text-xs font-bold text-white/70 mb-1.5">رابط أو مسار البوستر (Poster)</label>
                <input type="text" value={modalForm.poster_path} onChange={(e) => setModalForm({ ...modalForm, poster_path: e.target.value })} placeholder="/images/poster.png" dir="ltr" className="w-full rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-xs text-white outline-none focus:border-fuchsia-500 font-mono" />
              </div>
              <div>
                <label className="block text-xs font-bold text-white/70 mb-1.5">رابط أو مسار البانر (Banner)</label>
                <input type="text" value={modalForm.banner_path} onChange={(e) => setModalForm({ ...modalForm, banner_path: e.target.value })} placeholder="/images/banner.png" dir="ltr" className="w-full rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-xs text-white outline-none focus:border-fuchsia-500 font-mono" />
              </div>
            </div>

            {/* Genres */}
            <div>
              <label className="block text-xs font-bold text-white/70 mb-2">التصنيفات والوسوم (Genres)</label>
              <div className="flex flex-wrap gap-1.5">
                {ALL_GENRES.map((genre) => {
                  const selected = modalForm.genres.includes(genre);
                  return (
                    <button key={genre} type="button" onClick={() => toggleGenreInModal(genre)}
                      className={`px-3 py-1 rounded-xl text-xs font-semibold transition ${selected ? "bg-fuchsia-600 text-white shadow" : "bg-white/[0.05] text-white/60 hover:bg-white/10 hover:text-white"}`}>
                      {selected ? `✓ ${genre}` : `+ ${genre}`}
                    </button>
                  );
                })}
              </div>
              <div className="mt-2.5 flex items-center gap-2">
                <input type="text" value={customGenreInput} onChange={(e) => setCustomGenreInput(e.target.value)} onKeyDown={(e) => e.key === "Enter" && addCustomGenre()} placeholder="إضافة تصنيف مخصص آخر..." className="rounded-xl border border-white/10 bg-black/30 px-3 py-1.5 text-xs text-white outline-none focus:border-fuchsia-500" />
                <button type="button" onClick={addCustomGenre} className="px-3 py-1.5 rounded-xl bg-white/10 hover:bg-white/20 text-xs font-bold text-white">إضافة</button>
              </div>
            </div>

            {/* Plot */}
            <div>
              <label className="block text-xs font-bold text-white/70 mb-1.5">القصة والوصف (Overview / Plot)</label>
              <textarea rows={3} value={modalForm.plot_ar || modalForm.plot_en} onChange={(e) => setModalForm({ ...modalForm, plot_ar: e.target.value, plot_en: e.target.value })} placeholder="اكتب نبذة عن قصة الفيلم أو المسلسل..." className="w-full rounded-xl border border-white/10 bg-black/40 p-3 text-xs text-white outline-none focus:border-fuchsia-500" />
            </div>

            <div className="flex items-center justify-end gap-3 border-t border-white/10 pt-4">
              <button type="button" onClick={() => setEditingItem(null)} className="px-5 py-2.5 rounded-xl border border-white/10 bg-white/[0.05] text-xs font-bold text-white/70 hover:text-white">إلغاء</button>
              <button type="button" onClick={handleSaveMedia} disabled={saveLoading}
                className="px-7 py-2.5 rounded-xl bg-gradient-to-r from-purple-600 via-fuchsia-600 to-purple-700 text-xs font-bold text-white shadow-lg shadow-purple-900/50 hover:brightness-110 disabled:opacity-50">
                {saveLoading ? "جارٍ الحفظ..." : "💾 حفظ التعديلات"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirmation */}
      {deletingItem && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/85 p-4 backdrop-blur-md animate-fadeIn">
          <div className="w-full max-w-md rounded-3xl border border-rose-500/30 bg-[#0D0B18] p-6 text-center space-y-4 shadow-2xl">
            <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-rose-500/20 text-2xl text-rose-400">🗑️</div>
            <h3 className="text-xl font-bold text-white">تأكيد حذف العمل</h3>
            <p className="text-xs text-white/70 leading-relaxed">هل أنت متأكد من حذف العمل <span className="font-bold text-rose-300">"{deletingItem.title_ar || deletingItem.title_en}"</span>؟</p>
            <div className="flex items-center justify-center gap-3 pt-2">
              <button type="button" onClick={() => setDeletingItem(null)} className="px-5 py-2.5 rounded-xl border border-white/10 bg-white/[0.05] text-xs font-bold text-white">إلغاء</button>
              <button type="button" onClick={confirmDeleteMedia} disabled={deleteLoading}
                className="px-6 py-2.5 rounded-xl bg-rose-600 hover:bg-rose-700 text-xs font-bold text-white shadow-lg shadow-rose-900/50 disabled:opacity-50">
                {deleteLoading ? "جارٍ الحذف..." : "نعم، احذف الآن"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
