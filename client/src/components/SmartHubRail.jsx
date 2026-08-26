import { useEffect, useState } from "react";
import SmartHubCard from "./SmartHubCard.jsx";
import { getSmartHubs } from "../lib/api.js";

export default function SmartHubRail({ scope, title, description, onOpen }) {
  const [hubs, setHubs] = useState([]);
  useEffect(() => { let alive=true; getSmartHubs(scope).then((data)=>alive&&setHubs(data.hubs||[])).catch(()=>alive&&setHubs([])); return()=>{alive=false}; }, [scope]);
  if (!hubs.length) return null;
  return <section className="space-y-4" dir="rtl"><div><h2 className="text-xl font-black text-[color:var(--text-primary)]">{title || "مجموعات ومحاور"}</h2>{description && <p className="mt-1 text-xs text-[color:var(--text-muted)]">{description}</p>}</div><div className="flex gap-3 overflow-x-auto px-1 pb-5 pt-3 scrollbar-none snap-x touch-pan-x">{hubs.map((hub)=><div key={hub.slug} className="w-[min(84vw,340px)] shrink-0 snap-start sm:w-80"><SmartHubCard hub={hub} onOpen={onOpen} /></div>)}</div></section>;
}
