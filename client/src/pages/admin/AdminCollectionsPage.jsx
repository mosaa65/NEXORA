import { useEffect, useState } from "react";
import { Modal } from "../../components/ui/index.js";
import Icon from "../../components/Icon.jsx";
import { deleteCollection, getCategories, getCollections, getMediaList, saveCollection } from "../../lib/api.js";

const blank = { slug: "", title_ar: "", title_en: "", description_ar: "", description_en: "", artwork_path: "", artwork_position: "center center", accent: "violet", target_category_slug: "", target_filters: "{}", priority: 0, is_active: true, item_ids: [] };
const accents = ["violet", "amber", "cyan", "rose", "emerald"];

function mediaTitle(item) {
  return item.title_ar || item.title_en || item.original_title || item.original_name || `عمل #${item.id}`;
}

export default function AdminCollectionsPage() {
  const [collections, setCollections] = useState([]);
  const [categories, setCategories] = useState([]);
  const [mediaItems, setMediaItems] = useState([]);
  const [form, setForm] = useState(null);
  const [mediaQuery, setMediaQuery] = useState("");
  const [saving, setSaving] = useState(false);

  const load = () => getCollections().then((x) => setCollections(x.collections || [])).catch(() => setCollections([]));

  useEffect(() => {
    load();
    getCategories().then((x) => setCategories(x.categories || [])).catch(() => {});
    getMediaList({ limit: 1000, sort: "rating" }).then((x) => setMediaItems(x.items || [])).catch(() => setMediaItems([]));
  }, []);

  const set = (key, value) => setForm((current) => ({ ...current, [key]: value }));

  async function submit() {
    let filters;
    try { filters = JSON.parse(form.target_filters || "{}"); } catch { alert("الفلاتر يجب أن تكون JSON صحيحًا."); return; }
    setSaving(true);
    try {
      await saveCollection({ ...form, target_filters: filters, item_ids: form.item_ids || [] });
      setForm(null);
      load();
    } catch (error) {
      alert(`تعذر الحفظ: ${error.message}`);
    } finally {
      setSaving(false);
    }
  }

  async function remove(item) {
    if (!window.confirm(`حذف مجموعة «${item.title_ar || item.title_en}»؟`)) return;
    await deleteCollection(item.id);
    load();
  }

  function toggleMedia(mediaId) {
    setForm((current) => {
      const ids = current.item_ids || [];
      return { ...current, item_ids: ids.includes(mediaId) ? ids.filter((id) => id !== mediaId) : [...ids, mediaId] };
    });
  }

  function moveMedia(index, direction) {
    setForm((current) => {
      const ids = [...(current.item_ids || [])];
      const next = index + direction;
      if (next < 0 || next >= ids.length) return current;
      [ids[index], ids[next]] = [ids[next], ids[index]];
      return { ...current, item_ids: ids };
    });
  }

  const filteredMedia = mediaItems.filter((item) => mediaTitle(item).toLowerCase().includes(mediaQuery.trim().toLowerCase())).slice(0, 40);

  return <div className="space-y-6 text-right" dir="rtl">
    <header className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div><p className="text-xs font-bold text-fuchsia-400">الواجهة الرئيسية</p><h2 className="text-2xl font-black text-white">المجموعات والعروض التحريرية</h2><p className="mt-1 text-xs text-white/55">تحكم بالمحتوى والصورة والترتيب الذي يظهر في العرض الرئيسي.</p></div>
      <button onClick={() => setForm({ ...blank })} className="inline-flex min-h-11 items-center gap-2 rounded-xl bg-gradient-to-l from-violet-600 to-fuchsia-600 px-5 text-sm font-bold text-white"><Icon name="plus" className="h-4 w-4" />إضافة مجموعة</button>
    </header>

    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      {collections.map((item) => <article key={item.id} className="overflow-hidden rounded-2xl border border-white/10 bg-white/[.035]"><div className="h-24 bg-cover bg-center" style={{ backgroundImage: item.artwork_path ? `linear-gradient(rgba(12,10,22,.35),rgba(12,10,22,.85)),url('${item.artwork_path}')` : undefined }} /><div className="space-y-3 p-4"><div className="flex items-start justify-between gap-3"><div><h3 className="font-black text-white">{item.title_ar || item.title_en}</h3><p className="text-xs text-white/45" dir="ltr">{item.title_en}</p></div><span className={`rounded-full px-2 py-1 text-[10px] font-bold ${item.is_active ? "bg-emerald-500/15 text-emerald-300" : "bg-white/10 text-white/45"}`}>{item.is_active ? "نشطة" : "متوقفة"}</span></div><p className="line-clamp-2 text-xs leading-6 text-white/60">{item.description_ar || item.description_en || "لا يوجد وصف."}</p><div className="flex items-center justify-between border-t border-white/10 pt-3 text-xs"><span className="text-white/45">{item.target_category_slug || "كل الأقسام"} · {item.item_count || 0} عمل</span><span className="flex gap-3"><button onClick={() => setForm({ ...item, item_ids: item.item_ids || [], target_filters: JSON.stringify(item.target_filters || {}, null, 2) })} className="font-bold text-fuchsia-300">تعديل</button><button onClick={() => remove(item)} className="font-bold text-rose-300">حذف</button></span></div></div></article>)}
      {collections.length === 0 && <p className="rounded-2xl border border-dashed border-white/15 p-8 text-center text-sm text-white/50 md:col-span-2 xl:col-span-3">لا توجد مجموعات بعد. أضف أول مجموعة لتظهر في الحاوية الرئيسية.</p>}
    </div>

    {form && <Modal isOpen={Boolean(form)} onClose={() => setForm(null)} title={form.id ? "تعديل المجموعة" : "مجموعة جديدة"} subtitle="حدد هوية العرض والمحتوى وترتيبه في الصفحة الرئيسية" size="xl" className="admin-form-modal sm:max-w-6xl" actions={<><button onClick={() => setForm(null)} className="min-h-11 rounded-xl border border-[var(--border-default)] px-5 text-sm font-bold text-[var(--text-secondary)] transition hover:bg-[var(--bg-elevated)] hover:text-[var(--text-primary)]">إلغاء</button><button onClick={submit} disabled={saving} className="min-h-11 rounded-xl bg-gradient-to-l from-violet-600 to-fuchsia-600 px-6 text-sm font-black text-white disabled:opacity-50">{saving ? "جارٍ الحفظ..." : "حفظ المجموعة"}</button></>}>
      <div className="grid gap-4 lg:grid-cols-2">
        <div className="grid gap-4 sm:grid-cols-2">
          {[['title_ar','العنوان العربي'],['title_en','العنوان الإنجليزي'],['slug','المعرّف slug'],['artwork_path','مسار الصورة المحلية'],['artwork_position','موضع الصورة'],['priority','الترتيب']].map(([key, label]) => <label key={key} className="text-xs font-bold text-white/70">{label}<input value={form[key] ?? ""} type={key === "priority" ? "number" : "text"} dir={key.includes("en") || key === "slug" ? "ltr" : undefined} onChange={(event) => set(key, key === "priority" ? Number(event.target.value) : event.target.value)} className="mt-1.5 w-full rounded-xl border border-white/10 bg-black/25 px-3 py-2.5 text-sm font-medium text-white outline-none focus:border-fuchsia-400" /></label>)}
          <label className="text-xs font-bold text-white/70">اللون<select value={form.accent} onChange={(event) => set("accent", event.target.value)} className="mt-1.5 w-full rounded-xl border border-white/10 bg-black/25 px-3 py-2.5 text-white">{accents.map((accent) => <option key={accent}>{accent}</option>)}</select></label>
          <label className="text-xs font-bold text-white/70">القسم المستهدف<select value={form.target_category_slug} onChange={(event) => set("target_category_slug", event.target.value)} className="mt-1.5 w-full rounded-xl border border-white/10 bg-black/25 px-3 py-2.5 text-white"><option value="">كل الأقسام</option>{categories.map((category) => <option key={category.slug} value={category.slug}>{category.name_ar || category.slug}</option>)}</select></label>
          <label className="sm:col-span-2 text-xs font-bold text-white/70">الوصف العربي<textarea value={form.description_ar} onChange={(event) => set("description_ar", event.target.value)} className="mt-1.5 min-h-20 w-full rounded-xl border border-white/10 bg-black/25 px-3 py-2.5 text-sm text-white outline-none focus:border-fuchsia-400" /></label>
          <label className="sm:col-span-2 text-xs font-bold text-white/70">الوصف الإنجليزي<textarea dir="ltr" value={form.description_en} onChange={(event) => set("description_en", event.target.value)} className="mt-1.5 min-h-20 w-full rounded-xl border border-white/10 bg-black/25 px-3 py-2.5 text-sm text-white outline-none focus:border-fuchsia-400" /></label>
          <label className="sm:col-span-2 text-xs font-bold text-white/70">الفلاتر (JSON)<textarea dir="ltr" value={form.target_filters} onChange={(event) => set("target_filters", event.target.value)} className="mt-1.5 min-h-20 w-full rounded-xl border border-white/10 bg-black/25 px-3 py-2.5 font-mono text-xs text-white outline-none focus:border-fuchsia-400" /></label>
          <label className="sm:col-span-2 flex items-center gap-2 text-xs font-bold text-white/70"><input type="checkbox" checked={Boolean(form.is_active)} onChange={(event) => set("is_active", event.target.checked)} className="h-4 w-4 accent-fuchsia-500" />إظهار المجموعة في العرض الرئيسي</label>
        </div>

        <section className="rounded-2xl border border-white/10 bg-black/20 p-4"><div className="flex items-center justify-between gap-3"><div><h4 className="font-black text-white">عناصر المجموعة</h4><p className="mt-1 text-[11px] text-white/45">اختر الأعمال ورتب ظهورها في العرض.</p></div><span className="rounded-full bg-fuchsia-500/15 px-2.5 py-1 text-[11px] font-bold text-fuchsia-200">{form.item_ids?.length || 0} مختار</span></div>
          <input value={mediaQuery} onChange={(event) => setMediaQuery(event.target.value)} placeholder="ابحث عن عمل لإضافته..." className="mt-4 w-full rounded-xl border border-white/10 bg-white/5 px-3 py-2.5 text-xs text-white outline-none placeholder:text-white/35 focus:border-fuchsia-400" />
          <div className="mt-3 max-h-56 space-y-1 overflow-y-auto pr-1">{filteredMedia.map((item) => { const selected = (form.item_ids || []).includes(item.id); return <button type="button" key={item.id} onClick={() => toggleMedia(item.id)} className={`flex w-full items-center justify-between gap-3 rounded-xl border px-3 py-2 text-right text-xs transition ${selected ? "border-fuchsia-400/50 bg-fuchsia-500/15 text-white" : "border-white/5 bg-white/[.025] text-white/70 hover:bg-white/[.06]"}`}><span className="min-w-0 truncate">{mediaTitle(item)}</span><span className="shrink-0 text-[10px] text-white/40">{selected ? "مضاف" : "إضافة"}</span></button>; })}</div>
          <div className="mt-4 border-t border-white/10 pt-3"><p className="mb-2 text-[11px] font-bold text-white/55">الترتيب الحالي</p><div className="max-h-44 space-y-1 overflow-y-auto">{(form.item_ids || []).map((id, index) => { const item = mediaItems.find((candidate) => candidate.id === id); return <div key={id} className="flex items-center gap-2 rounded-lg bg-white/5 px-2 py-1.5 text-xs text-white/80"><span className="w-5 text-center text-white/35">{index + 1}</span><span className="min-w-0 flex-1 truncate">{item ? mediaTitle(item) : `عمل #${id}`}</span><button type="button" onClick={() => moveMedia(index, -1)} className="px-1 text-white/50 hover:text-white" aria-label="رفع العنصر">↑</button><button type="button" onClick={() => moveMedia(index, 1)} className="px-1 text-white/50 hover:text-white" aria-label="خفض العنصر">↓</button><button type="button" onClick={() => toggleMedia(id)} className="px-1 text-rose-300" aria-label="إزالة العنصر">×</button></div>; })}</div></div>
        </section>
      </div>
    </Modal>}
  </div>;
}
