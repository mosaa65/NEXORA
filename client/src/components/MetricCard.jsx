export default function MetricCard({ label, value, hint }) {
  return (
    <div className="relative overflow-hidden rounded-3xl border border-white/10 bg-[linear-gradient(180deg,rgba(255,255,255,0.08),rgba(255,255,255,0.03))] p-5 text-right shadow-panel backdrop-blur-xl">
      <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-electric/80 to-transparent" />
      <p className="text-xs font-semibold text-white/45">{label}</p>
      <div className="mt-3 text-3xl font-bold text-white">{value}</div>
      <p className="mt-2 text-sm leading-6 text-white/60">{hint}</p>
    </div>
  );
}
