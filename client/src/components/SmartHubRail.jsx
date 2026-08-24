import { useEffect, useState } from "react";
import SmartHubCard from "./SmartHubCard.jsx";
import { getSmartHubs } from "../lib/api.js";

export default function SmartHubRail({ scope, title, description, onOpen }) {
  const [hubs, setHubs] = useState([]);
  useEffect(() => { let alive=true; getSmartHubs(scope).then((data)=>alive&&setHubs(data.hubs||[])).catch(()=>alive&&setHubs([])); return()=>{alive=false}; }, [scope]);
  if (!hubs.length) return null;
  return <section className="space-y-4" dir="rtl"><div><h2 className="text-xl font-black text-[color:var(--text-primary)]">{title || "مجموعات ومحاور"}</h2>{description && <p className="mt-1 text-xs text-[color:var(--text-muted)]">{description}</p>}</div><div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">{hubs.map((hub)=><SmartHubCard key={hub.slug} hub={hub} onOpen={onOpen} />)}</div></section>;
}
