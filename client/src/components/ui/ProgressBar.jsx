import React from "react";

/**
 * Unified ProgressBar component.
 *
 * @param {number} percent - Value 0-100
 * @param {"sm"|"md"|"lg"} size - Height
 * @param {"primary"|"success"|"warning"|"danger"|"auto"} variant
 *   "auto" = changes color based on percent (green < 75, amber < 90, red >= 90)
 */
const SIZE_CLASSES = {
  sm: "h-1.5",
  md: "h-2",
  lg: "h-3",
};

const COLOR_CLASSES = {
  primary: "bg-gradient-to-r from-purple-500 to-fuchsia-500",
  success: "bg-emerald-500",
  warning: "bg-amber-500",
  danger: "bg-rose-500",
};

function getAutoColor(percent) {
  if (percent >= 90) return COLOR_CLASSES.danger;
  if (percent >= 75) return COLOR_CLASSES.warning;
  return COLOR_CLASSES.primary;
}

export default function ProgressBar({
  percent = 0,
  size = "md",
  variant = "auto",
  className = "",
  showLabel = false,
}) {
  const clamped = Math.min(Math.max(percent, 0), 100);
  const colorClass =
    variant === "auto" ? getAutoColor(clamped) : (COLOR_CLASSES[variant] || COLOR_CLASSES.primary);

  return (
    <div className={`space-y-1 ${className}`}>
      {showLabel && (
        <div className="flex items-center justify-between text-[11px] text-white/50">
          <span>{clamped.toFixed(0)}%</span>
        </div>
      )}
      <div
        className={`w-full rounded-full bg-white/10 overflow-hidden ${SIZE_CLASSES[size] || SIZE_CLASSES.md}`}
      >
        <div
          className={`h-full rounded-full transition-all duration-500 ${colorClass}`}
          style={{ width: `${clamped}%` }}
        />
      </div>
    </div>
  );
}
