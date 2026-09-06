import React, { useEffect, useState, useMemo } from "react";
import Icon from "../components/Icon.jsx";
import SmartHubCard from "../components/SmartHubCard.jsx";
import FilterToolbar from "../components/FilterToolbar.jsx";
import { getFranchises, getPeople, getSmartHubs, resolveAPIURL } from "../lib/api.js";
import { useNavigationState } from "../context/NavigationStateContext.jsx";
import { useScrollRestoration } from "../hooks/useScrollRestoration.js";

const directoryConfig = {
  hubs: { title: "كل المحاور والمجموعات الذكية", subtitle: "استكشف جميع المحاور الفنية المحفوظة في مكتبتك.", icon: "grid", tone: "fuchsia" },
  people: { title: "أبرز ممثلي المكتبة", subtitle: "الممثلون الرئيسيون وفق ترتيب طاقم TMDB، المرتبطون بأعمال مكتبتك.", icon: "user", tone: "cyan" },
  franchises: { title: "كل سلاسل الأفلام والعوالم", subtitle: "السلاسل السينمائية المرتبطة بالأعمال الموجودة في مكتبتك.", icon: "film", tone: "amber" },
};

const personWorkFilters = [
  { id: "all", label: "كل الممثلين الرئيسيين" },
  { id: "works-2", label: "عملان محليان فأكثر" },
  { id: "works-5", label: "5 أعمال محلية فأكثر" },
  { id: "works-10", label: "10 أعمال محلية فأكثر" },
];

const personSortOptions = [
  { id: "featured", label: "الأكثر ظهورًا في المكتبة" },
  { id: "works", label: "الأكثر أعمالًا محلية" },
  { id: "popular", label: "الأكثر شهرة" },
  { id: "name", label: "الاسم (أ - ي)" },
];

const personDepartmentLabels = {
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

function DirectoryCard({ kind, item, onOpen }) {
  const isPerson = kind === "people";
  const [imgError, setImgError] = useState(false);

  if (kind === "hubs") {
    return <SmartHubCard hub={item} onOpen={onOpen} />;
  }

  const rawImage = isPerson ? item.profile_path : (item.backdrop_path || item.poster_path);
  const image = resolveAPIURL(rawImage);
  return (
    <button
      type="button"
      onClick={() => onOpen(item)}
      className={`group overflow-hidden rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] text-right shadow-[var(--shadow-sm)] transition duration-300 hover:-translate-y-1 hover:border-[var(--color-info)] hover:shadow-[var(--shadow-lg)] ${isPerson ? "" : "min-h-48"}`}
      dir="rtl"
    >
      <div className={`relative overflow-hidden ${isPerson ? "aspect-[4/5] bg-gradient-to-br from-cyan-950/40 via-purple-950/20 to-fuchsia-950/30" : "h-32 bg-black/20"}`}>
        {!imgError && image ? (
          <img
            src={image}
            alt=""
            className="h-full w-full object-cover transition duration-500 group-hover:scale-105"
            onError={() => setImgError(true)}
          />
        ) : (
          <div className="absolute inset-0 flex items-center justify-center p-3 text-center">
            <span className="flex h-14 w-14 sm:h-16 sm:w-16 items-center justify-center rounded-2xl border border-[var(--border-default)] bg-[var(--bg-elevated)] text-[var(--color-info)] shadow-inner">
              <Icon name={isPerson ? "user" : "film"} className="h-7 w-7 sm:h-8 sm:w-8" />
            </span>
          </div>
        )}
      </div>
      <span className="block space-y-1 p-3.5 sm:p-4">
        <strong className="block truncate text-xs sm:text-sm font-black text-[var(--text-primary)]">
          {isPerson ? (item.name_ar || item.name_en) : (item.title_ar || item.title_en)}
        </strong>
        {isPerson && item.name_ar && item.name_en && (
          <small className="block truncate text-left text-[10px] sm:text-[11px] font-semibold text-[var(--text-muted)]" dir="ltr">
            {item.name_en}
          </small>
        )}
        <small className="inline-flex items-center gap-1 rounded-md bg-[var(--color-info-light)] px-1.5 py-0.5 text-[10px] font-bold text-[var(--color-info)]">
          <Icon name="film" className="h-3 w-3" />
          {isPerson ? `${item.local_media_count || 0} أعمال محلية` : `${item.local_item_count || 0} أفلام متاحة`}
        </small>
      </span>
    </button>
  );
}

export default function DirectoryPage({ kind = "hubs", onOpen }) {
  const { getPageState, savePageState } = useNavigationState();
  const cacheKey = `directory:${kind}`;
  const cachedState = useMemo(() => getPageState(cacheKey), [cacheKey, getPageState]);

  const [items, setItems] = useState(() => cachedState?.items || null);
  const [query, setQuery] = useState(() => cachedState?.query || "");
  const [sort, setSort] = useState(() => cachedState?.sort || "featured");
  const [workFilter, setWorkFilter] = useState(() => cachedState?.workFilter || "all");
  const [departmentFilter, setDepartmentFilter] = useState(() => cachedState?.departmentFilter || "all");
  const config = directoryConfig[kind] || directoryConfig.hubs;
  const personDepartmentFilters = useMemo(() => {
    const departments = [...new Set((items || []).map((person) => person.known_for_department).filter(Boolean))].sort((left, right) => left.localeCompare(right));
    return [
      { id: "all", label: "كل التخصصات" },
      ...departments.map((department) => ({ id: department, label: personDepartmentLabels[department] || department })),
    ];
  }, [items]);

  // Use Scroll Restoration
  useScrollRestoration(cacheKey, Boolean(items), { items, query, sort, workFilter, departmentFilter });

  useEffect(() => {
    let alive = true;
    const cached = getPageState(cacheKey);
    if (!cached || !cached.items) {
      const load = kind === "people" ? getPeople(100) : kind === "franchises" ? getFranchises(100) : getSmartHubs();
      load
        .then((data) => {
          if (alive) {
            const list = data?.[kind === "people" ? "people" : kind === "franchises" ? "franchises" : "hubs"] || [];
            setItems(list);
            savePageState(cacheKey, { items: list, query, sort, workFilter, departmentFilter });
          }
        })
        .catch(() => {
          if (alive) setItems([]);
        });
    }
    return () => {
      alive = false;
    };
  }, [kind, cacheKey, getPageState, savePageState]);

  const visibleItems = useMemo(() => {
    if (kind !== "people") return items || [];

    const normalizedQuery = query.trim().toLocaleLowerCase();
    const minimumWorks = workFilter === "works-2" ? 2 : workFilter === "works-5" ? 5 : workFilter === "works-10" ? 10 : 0;
    const people = (items || []).filter((person) => {
      const searchable = `${person.name_ar || ""} ${person.name_en || ""}`.toLocaleLowerCase();
      return (
        (!normalizedQuery || searchable.includes(normalizedQuery)) &&
        Number(person.local_media_count || 0) >= minimumWorks &&
        (departmentFilter === "all" || person.known_for_department === departmentFilter)
      );
    });

    return [...people].sort((left, right) => {
      if (sort === "works") {
        return Number(right.local_media_count || 0) - Number(left.local_media_count || 0) || Number(right.popularity || 0) - Number(left.popularity || 0);
      }
      if (sort === "popular") {
        return Number(right.popularity || 0) - Number(left.popularity || 0) || Number(right.local_media_count || 0) - Number(left.local_media_count || 0);
      }
      if (sort === "name") {
        return (left.name_ar || left.name_en || "").localeCompare(right.name_ar || right.name_en || "", "ar");
      }
      return Number(right.local_media_count || 0) - Number(left.local_media_count || 0) || Number(right.popularity || 0) - Number(left.popularity || 0);
    });
  }, [items, kind, query, sort, workFilter, departmentFilter]);

  if (!items) return <div className="min-h-72 animate-pulse rounded-2xl bg-[var(--bg-surface)]" />;

  return (
    <div className="space-y-7 pb-16 text-right" dir="rtl">
      <header className="flex flex-wrap items-end justify-between gap-4 border-b border-[var(--border-subtle)] pb-5">
        <div>
          <p className="text-xs font-bold text-[var(--color-accent)]">دليل المكتبة الموحد</p>
          <h1 className="mt-1 flex items-center gap-2 text-2xl font-black text-[var(--text-primary)]">
            <Icon name={config.icon} className="h-6 w-6 text-[var(--color-accent)]" />
            {config.title}
          </h1>
          <p className="mt-2 text-sm text-[var(--text-muted)]">{config.subtitle}</p>
        </div>
        <button
          type="button"
          onClick={() => {
            window.location.hash = "#/";
          }}
          className="inline-flex items-center gap-2 rounded-xl border border-[var(--border-default)] bg-[var(--bg-surface)] px-4 py-2.5 text-xs font-bold text-[var(--text-primary)] transition hover:bg-[var(--bg-elevated)]"
        >
          <Icon name="arrowRight" className="h-4 w-4" />
          العودة
        </button>
      </header>

      {kind === "people" && items.length > 0 && (
        <FilterToolbar
          showOriginFilter={false}
          showGenreFilter={false}
          showTypeFilter={false}
          formats={personWorkFilters}
          formatLabel="عدد الأعمال المحلية"
          activeFormat={workFilter}
          onSelectFormat={setWorkFilter}
          statuses={personDepartmentFilters}
          statusLabel="التخصص في المنصة"
          activeStatus={departmentFilter}
          onSelectStatus={setDepartmentFilter}
          sorts={personSortOptions}
          activeSort={sort}
          onSelectSort={setSort}
          searchQuery={query}
          onSearchChange={setQuery}
          resultCount={visibleItems.length}
          onResetFilters={() => {
            setQuery("");
            setWorkFilter("all");
            setDepartmentFilter("all");
            setSort("featured");
          }}
        />
      )}

      {items.length ? visibleItems.length ? (
        <div
          className={`grid grid-cols-2 gap-2.5 sm:gap-4 ${
            kind === "people"
              ? "sm:grid-cols-3 lg:grid-cols-5"
              : "sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
          }`}
        >
          {visibleItems.map((item) => (
            <DirectoryCard key={item.slug || item.id} kind={kind} item={item} onOpen={onOpen} />
          ))}
        </div>
      ) : (
        <div className="rounded-2xl border border-dashed border-[var(--border-default)] p-16 text-center text-sm text-[var(--text-muted)]">
          لا يوجد ممثل رئيسي يطابق البحث أو الفلاتر الحالية.
        </div>
      ) : (
        <div className="rounded-2xl border border-dashed border-[var(--border-default)] p-16 text-center text-sm text-[var(--text-muted)]">
          لا توجد بيانات متاحة حاليًا.
        </div>
      )}
    </div>
  );
}
