import React from "react";

/**
 * Unified GlassCard component — the signature NEXORA glass panel.
 * Replaces the old GlassCard.jsx with design token support.
 *
 * @param {"default"|"elevated"|"bordered"} variant
 * @param {boolean} hover - Enable hover glow effect
 * @param {boolean} interactive - Cursor pointer + click feedback
 */
export default function Card({
  children,
  variant = "default",
  hover = false,
  interactive = false,
  className = "",
  ...props
}) {
  const base = `
    relative overflow-hidden rounded-3xl
    border border-[var(--border-default)]
    backdrop-blur-[var(--glass-blur)]
  `;

  const variantClasses = {
    default: "bg-[var(--bg-card)]",
    elevated: "bg-[var(--bg-elevated)] shadow-lg",
    bordered: "bg-transparent border-2",
  };

  const hoverClass = hover
    ? "hover:border-fuchsia-500/50 hover:shadow-xl hover:shadow-purple-950/50 transition duration-300"
    : "";

  const interactiveClass = interactive
    ? "cursor-pointer active:scale-[0.98] transition-transform"
    : "";

  return (
    <div
      className={`
        ${base}
        ${variantClasses[variant] || variantClasses.default}
        ${hoverClass}
        ${interactiveClass}
        ${className}
      `.trim()}
      {...props}
    >
      {children}
    </div>
  );
}

/**
 * Metric stat card for dashboards.
 */
export function MetricCard({
  label,
  value,
  subtext,
  accentColor = "text-white",
  className = "",
}) {
  return (
    <Card className={`p-5 ${className}`}>
      <p className="text-xs font-bold text-white/50">{label}</p>
      <p className={`mt-2 text-3xl font-black ${accentColor}`}>{value}</p>
      {subtext && <p className="text-xs text-white/40 mt-1">{subtext}</p>}
    </Card>
  );
}
