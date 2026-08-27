import { useEffect, useState } from "react";
import SmartHubCard from "./SmartHubCard.jsx";
import { getSmartHubs } from "../lib/api.js";

export default function SmartHubRail({ scope, title, description, onOpen, onViewAll }) {
  const [hubs, setHubs] = useState([]);
  useEffect(() => { let alive=true; getSmartHubs(scope).then((data)=>alive&&setHubs(data.hubs||[])).catch(()=>alive&&setHubs([])); return()=>{alive=false}; }, [scope]);
  if (!hubs.length) return null;
  return <section className="space-y-4" dir="rtl"><div className="flex items-end justify-between gap-3"><div><h2 className="text-xl font-black text-[var(--text-primary)]">{title || "مجموعات ومحاور"}</h2>{description && <p className="mt-1 text-xs text-[var(--text-muted)]">{description}</p>}</div>{onViewAll && <button type="button" onClick={onViewAll} className="inline-flex shrink-0 items-center gap-1.5 rounded-xl border border-[var(--border-default)] bg-[var(--bg-card)] px-3.5 py-2 text-xs font-bold text-[var(--color-accent)] shadow-[var(--shadow-sm)] transition hover:border-[var(--color-accent)] hover:bg-[var(--bg-elevated)]"><span>عرض الكل</span><span aria-hidden="true">←</span></button>}</div><div className="flex gap-3 overflow-x-auto px-1 pb-5 pt-3 scrollbar-none snap-x touch-pan-x">{hubs.map((hub)=><div key={hub.slug} className="w-[min(84vw,340px)] shrink-0 snap-start sm:w-80"><SmartHubCard hub={hub} onOpen={onOpen} /></div>)}</div></section>;
}
