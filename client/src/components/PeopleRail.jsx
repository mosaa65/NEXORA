import { useEffect, useState } from "react";
import Icon from "./Icon.jsx";
import { getPeople, resolveAPIURL } from "../lib/api.js";

function PersonCard({ person, onOpen }) {
  const name = person.name_ar || person.name_en;
  const image = resolveAPIURL(person.profile_path);
  return <button type="button" onClick={() => onOpen?.(person)} className="group relative flex w-40 shrink-0 flex-col overflow-hidden rounded-xl border border-[var(--border-default)] bg-[var(--bg-card)] text-right shadow-[var(--shadow-sm)] transition duration-300 hover:-translate-y-1 hover:border-cyan-300/55 hover:shadow-[var(--shadow-lg)] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--color-accent)] sm:w-44" dir="rtl" aria-label={`فتح أعمال ${name}`}>
    <div className="relative aspect-[4/5] overflow-hidden bg-gradient-to-br from-cyan-950/80 via-[#161321] to-fuchsia-950/70">{image ? <img src={image} alt="" loading="lazy" onError={(event) => { event.currentTarget.style.display = "none"; }} className="h-full w-full object-cover transition duration-700 group-hover:scale-105" /> : <span className="absolute inset-0 flex items-center justify-center text-cyan-100/70"><Icon name="user" className="h-10 w-10" /></span>}<span className="absolute inset-x-0 bottom-0 h-2/5 bg-gradient-to-t from-[#0b0915] to-transparent" /></div>
    <span className="space-y-1.5 p-3"><strong className="block line-clamp-1 text-sm font-black text-[var(--text-primary)]">{name}</strong>{person.name_ar && person.name_en && <small dir="ltr" className="block truncate text-left text-[10px] font-semibold text-[var(--text-muted)]">{person.name_en}</small>}<small className="inline-flex items-center gap-1 rounded-md bg-cyan-400/10 px-1.5 py-1 text-[10px] font-bold text-cyan-100"><Icon name="film" className="h-3 w-3" />{person.local_media_count} أعمال محلية</small></span>
  </button>;
}

export default function PeopleRail({ onOpen }) {
  const [people, setPeople] = useState([]);
  useEffect(() => { let alive = true; getPeople().then((data) => { if (alive) setPeople(data.people || []); }).catch(() => { if (alive) setPeople([]); }); return () => { alive = false; }; }, []);
  if (!people.length) return null;
  return <section className="space-y-4" dir="rtl" aria-label="نجوم المكتبة"><div><p className="text-xs font-bold text-cyan-300">روابط محفوظة محلياً</p><h2 className="mt-1 flex items-center gap-2 text-xl font-black text-[var(--text-primary)]"><Icon name="user" className="h-5 w-5 text-cyan-300" />نجوم تتكرر أعمالهم لديك</h2><p className="mt-1 text-xs text-[var(--text-muted)]">أعمال مرتبطة بالممثلين والمخرجين من بيانات مكتبتك فقط.</p></div><div className="flex gap-4 overflow-x-auto pb-3 scrollbar-none snap-x">{people.map((person) => <div key={person.slug} className="snap-start"><PersonCard person={person} onOpen={onOpen} /></div>)}</div></section>;
}
