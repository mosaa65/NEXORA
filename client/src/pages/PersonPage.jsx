import React, { useEffect, useState, useMemo } from "react";
import Icon from "../components/Icon.jsx";
import PageBackButton from "../components/PageBackButton.jsx";
import MediaCollection from "../components/MediaCollection.jsx";
import { getPersonMedia, resolveAPIURL } from "../lib/api.js";
import { useNavigationState } from "../context/NavigationStateContext.jsx";
import { useScrollRestoration } from "../hooks/useScrollRestoration.js";

const departmentLabels = {
  Acting: "تمثيل",
  Directing: "إخراج",
  Writing: "كتابة",
  Production: "إنتاج",
  "Visual Effects": "مؤثرات بصرية",
  Crew: "طاقم فني",
  Art: "فن وتصميم",
  Camera: "تصوير",
  Sound: "صوت",
  Creator: "إبداع",
  Editing: "مونتاج",
};

export default function PersonPage({ slug, onOpenMedia }) {
  const { getPageState, savePageState } = useNavigationState();
  const cacheKey = `person:${slug}`;
  const cachedState = useMemo(() => getPageState(cacheKey), [cacheKey, getPageState]);

  const [data, setData] = useState(() => cachedState?.data || null);
  const [error, setError] = useState("");

  // Use Scroll Restoration
  useScrollRestoration(cacheKey, Boolean(data), { data });

  useEffect(() => {
    let alive = true;
    const cached = getPageState(cacheKey);
    setError("");
    if (cached?.data) {
      setData(cached.data);
    } else {
      // تبديل الممثل من صفحة التفاصيل يجب ألا يُبقي بطاقة الممثل السابق ظاهرة
      // أثناء تحميل الصفحة الجديدة.
      setData(null);
      getPersonMedia(slug, { limit: 100 })
        .then((payload) => {
          if (alive) {
            setData(payload);
            savePageState(cacheKey, { data: payload });
          }
        })
        .catch((err) => {
          if (alive) setError(err.message || "تعذر تحميل أعمال الشخص");
        });
    }
    return () => {
      alive = false;
    };
  }, [slug, cacheKey, getPageState, savePageState]);

  if (error)
    return (
      <div className="rounded-xl border border-rose-400/30 bg-rose-950/15 p-7 text-center text-sm font-bold text-rose-200" dir="rtl">
        {error}
      </div>
    );
  if (!data) return <div className="h-80 animate-pulse rounded-3xl border border-white/10 bg-white/[0.03]" aria-label="جارٍ تحميل صفحة الشخص" />;

  const person = data.person || {};
  const name = person.name_ar || person.name_en || "شخص من المكتبة";
  const image = resolveAPIURL(person.profile_path);
  const department = person.known_for_department || "أعمال مرتبطة داخل مكتبتك";
  const departmentLabel = departmentLabels[department] || department;
  const localTotal = data.total || items.length;
  const popularity = Number(person.popularity || 0);
  const items = (data.items || []).map((item) => ({
    id: item.id,
    titleAr: item.title_ar,
    titleEn: item.title_en,
    type: item.type,
    year: item.release_year,
    rating: item.rating,
    posterPath: item.poster_path,
    status: item.status,
    seasonCount: item.season_count,
    tmdbSeasonCount: item.tmdb_season_count,
    totalSize: item.total_size,
    bestResolution: item.best_resolution,
    runtimeMinutes: item.runtime_minutes,
    hasArabicAudio: item.has_arabic_audio,
    hasArabicSubtitles: item.has_arabic_subtitles,
  }));

  return (
    <div className="space-y-8 pb-16" dir="rtl">
      <section className="relative isolate overflow-hidden rounded-3xl border border-[var(--border-default)] bg-[#10101a] shadow-[var(--shadow-lg)]">
        {image && (
          <img
            src={image}
            alt=""
            aria-hidden="true"
            className="pointer-events-none absolute -left-10 top-0 h-full w-[58%] object-cover opacity-[0.16] blur-[2px] mask-image-gradient"
            onError={(event) => {
              event.currentTarget.style.display = "none";
            }}
          />
        )}
        <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_18%_12%,rgba(45,212,191,0.17),transparent_28%),radial-gradient(circle_at_78%_92%,rgba(192,132,252,0.17),transparent_36%),linear-gradient(115deg,rgba(8,10,18,0.72),rgba(20,15,35,0.94))]" />
        <div className="pointer-events-none absolute -right-24 -top-32 h-72 w-72 rounded-full bg-cyan-400/10 blur-3xl" />
        <div className="pointer-events-none absolute -bottom-32 left-1/3 h-72 w-72 rounded-full bg-fuchsia-500/10 blur-3xl" />
        <div className="absolute right-4 top-4 z-10 sm:right-6 sm:top-6">
          <PageBackButton fallback="/directory/people" forceFallback />
        </div>
        <div className="relative grid min-h-[390px] items-end gap-7 p-5 pt-20 sm:min-h-[360px] sm:grid-cols-[auto_minmax(0,1fr)] sm:p-9 sm:pt-24 lg:p-11">
          <div className="relative mx-auto w-fit sm:mx-0">
            <div className="absolute -inset-3 rounded-[2rem] bg-gradient-to-br from-cyan-300/25 via-transparent to-fuchsia-400/25 blur-xl" />
            <div className="relative h-48 w-40 shrink-0 overflow-hidden rounded-[1.6rem] border border-white/20 bg-gradient-to-br from-cyan-950/70 via-purple-950/40 to-fuchsia-950/60 shadow-2xl ring-1 ring-white/10 sm:h-56 sm:w-44">
            {image ? (
              <img
                src={image}
                alt={name}
                onError={(event) => {
                  event.currentTarget.style.display = "none";
                  if (event.currentTarget.nextElementSibling) event.currentTarget.nextElementSibling.classList.remove("hidden");
                }}
                className="h-full w-full object-cover object-top transition duration-500 hover:scale-105"
              />
            ) : null}
            <div className={`${image ? "hidden" : ""} absolute inset-0 flex items-center justify-center p-3 text-center`}>
              <span className="flex h-16 w-16 sm:h-20 sm:w-20 items-center justify-center rounded-2xl border border-white/15 bg-black/20 text-cyan-200 shadow-inner">
                <Icon name="user" className="h-8 w-8 sm:h-10 sm:w-10" />
              </span>
            </div>
            </div>
            <span className="absolute -bottom-3 left-1/2 inline-flex -translate-x-1/2 items-center gap-1.5 whitespace-nowrap rounded-full border border-cyan-200/20 bg-[#101827]/90 px-3 py-1.5 text-[10px] font-black text-cyan-100 shadow-lg backdrop-blur">
              <Icon name="user" className="h-3.5 w-3.5" />
              ملف شخصي محلي
            </span>
          </div>
          <div className="min-w-0 space-y-4 text-center sm:text-right">
            <div className="space-y-1.5">
              <p className="text-xs font-black tracking-wide text-cyan-200/85">طاقم العمل في المكتبة</p>
              <h1 className="text-3xl font-black tracking-tight text-white sm:text-4xl lg:text-5xl">{name}</h1>
            </div>
            {person.name_ar && person.name_en && (
              <p className="text-sm font-semibold tracking-wide text-white/60" dir="ltr">
                {person.name_en}
              </p>
            )}
            <p className="max-w-2xl text-sm leading-7 text-white/70">استعرض الأعمال المتاحة لهذا الشخص داخل مكتبتك المحلية.</p>
            <div className="flex flex-wrap justify-center gap-2 sm:justify-start">
              <span className="inline-flex items-center gap-1.5 rounded-xl border border-cyan-300/20 bg-cyan-300/10 px-3 py-2 text-xs font-bold text-cyan-100 backdrop-blur-sm">
                <Icon name="film" className="h-4 w-4" />
                {localTotal} أعمال متاحة
              </span>
              <span className="inline-flex items-center gap-1.5 rounded-xl border border-white/15 bg-black/20 px-3 py-2 text-xs font-bold text-white/80 backdrop-blur-sm">
                <Icon name="tag" className="h-3.5 w-3.5 text-fuchsia-200" />
                {departmentLabel}
              </span>
              {popularity > 0 && (
                <span className="inline-flex items-center gap-1.5 rounded-xl border border-amber-300/15 bg-amber-300/10 px-3 py-2 text-xs font-bold text-amber-100 backdrop-blur-sm" title="مؤشر الشهرة من TMDB">
                  <Icon name="star" className="h-3.5 w-3.5 fill-current stroke-0" />
                  شهرة {new Intl.NumberFormat("ar").format(Math.round(popularity))}
                </span>
              )}
            </div>
          </div>
        </div>
      </section>
      <section className="space-y-4">
        <div className="flex flex-wrap items-end justify-between gap-3 border-b border-[var(--border-subtle)] pb-4">
          <div>
            <p className="text-xs font-black text-[var(--color-accent)]">فيلموغرافيا محلية</p>
            <h2 className="mt-1 text-2xl font-black text-[var(--text-primary)]">أعمال {name}</h2>
            <p className="mt-1 text-xs text-[var(--text-muted)]">فقط الأعمال الموجودة حاليًا على أقراص المكتبة.</p>
          </div>
          <span className="rounded-xl border border-[var(--border-default)] bg-[var(--bg-card)] px-3 py-2 text-xs font-bold text-[var(--text-secondary)]">
            {localTotal} عمل
          </span>
        </div>
        <MediaCollection items={items} onOpen={onOpenMedia} />
      </section>
    </div>
  );
}
