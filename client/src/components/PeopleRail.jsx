import { useEffect, useState } from "react";
import Icon from "./Icon.jsx";
import { getPeople, resolveAPIURL } from "../lib/api.js";

function PersonCard({ person, onOpen }) {
  const name = person.name_ar || person.name_en;
  const image = resolveAPIURL(person.profile_path);
  const [imgError, setImgError] = useState(!image);

  return (
    <button
      type="button"
      onClick={() => onOpen?.(person)}
      className="group relative flex w-36 sm:w-44 shrink-0 flex-col overflow-hidden rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] text-right shadow-[var(--shadow-sm)] transition duration-300 hover:-translate-y-1 hover:border-[var(--color-info)] hover:shadow-[var(--shadow-lg)] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--color-accent)]"
      dir="rtl"
      aria-label={`فتح أعمال ${name}`}
    >
      <div className="relative aspect-[4/5] w-full overflow-hidden bg-gradient-to-br from-cyan-950/40 via-purple-950/20 to-fuchsia-950/30">
        {!imgError && image ? (
          <img
            src={image}
            alt={name}
            loading="lazy"
            onError={() => setImgError(true)}
            className="h-full w-full object-cover transition duration-700 group-hover:scale-105"
          />
        ) : (
          <div className="absolute inset-0 flex items-center justify-center p-3 text-center">
            <span className="flex h-14 w-14 sm:h-16 sm:w-16 items-center justify-center rounded-2xl border border-[var(--border-default)] bg-[var(--bg-elevated)] text-[var(--color-info)] shadow-inner">
              <Icon name="user" className="h-7 w-7 sm:h-8 sm:w-8" />
            </span>
          </div>
        )}
        <span className="absolute inset-x-0 bottom-0 h-1/3 bg-gradient-to-t from-black/60 to-transparent pointer-events-none" />
      </div>
      <span className="space-y-1.5 p-3 sm:p-3.5">
        <strong className="block line-clamp-1 text-xs sm:text-sm font-black text-[var(--text-primary)]">{name}</strong>
        {person.name_ar && person.name_en && (
          <small dir="ltr" className="block truncate text-left text-[10px] sm:text-[11px] font-semibold text-[var(--text-muted)]">
            {person.name_en}
          </small>
        )}
        <small className="inline-flex items-center gap-1 rounded-md bg-[var(--color-info-light)] px-1.5 py-0.5 sm:px-2 sm:py-1 text-[10px] font-bold text-[var(--color-info)]">
          <Icon name="film" className="h-3 w-3" />
          {person.local_media_count} أعمال محلية
        </small>
      </span>
    </button>
  );
}

export default function PeopleRail({ onOpen, onViewAll }) {
  const [people, setPeople] = useState([]);
  useEffect(() => { let alive = true; getPeople().then((data) => { if (alive) setPeople(data.people || []); }).catch(() => { if (alive) setPeople([]); }); return () => { alive = false; }; }, []);
  if (!people.length) return null;
  return <section className="space-y-4" dir="rtl" aria-label="نجوم المكتبة"><div className="flex items-end justify-between gap-3"><div><p className="text-xs font-bold text-[var(--color-info)]">روابط محفوظة محلياً</p><h2 className="mt-1 flex items-center gap-2 text-xl font-black text-[var(--text-primary)]"><Icon name="user" className="h-5 w-5 text-[var(--color-info)]" />نجوم تتكرر أعمالهم لديك</h2><p className="mt-1 text-xs text-[var(--text-muted)]">أعمال مرتبطة بالممثلين والمخرجين من بيانات مكتبتك فقط.</p></div>{onViewAll && <button type="button" onClick={onViewAll} className="inline-flex shrink-0 items-center gap-1.5 rounded-xl border border-[var(--border-default)] bg-[var(--bg-card)] px-3.5 py-2 text-xs font-bold text-[var(--color-info)] shadow-[var(--shadow-sm)] transition hover:border-[var(--color-info)] hover:bg-[var(--bg-elevated)]"><span>عرض الكل</span><span aria-hidden="true">←</span></button>}</div><div className="flex gap-4 overflow-x-auto px-1 pb-5 pt-3 scrollbar-none snap-x">{people.map((person) => <div key={person.slug} className="snap-start"><PersonCard person={person} onOpen={onOpen} /></div>)}</div></section>;
}
