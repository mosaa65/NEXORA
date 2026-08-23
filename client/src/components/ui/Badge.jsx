import React from "react";

/**
 * Unified Badge component for labels, tags, and status indicators.
 *
 * @param {"default"|"primary"|"success"|"warning"|"danger"|"info"|"accent"} variant
 * @param {"sm"|"md"} size
 */
const VARIANT_CLASSES = {
  default: "bg-white/[0.06] border-white/5 text-white/70",
  primary: "bg-purple-500/20 border-purple-500/30 text-purple-300",
  success: "bg-emerald-500/20 border-emerald-500/30 text-emerald-300",
  warning: "bg-amber-500/20 border-amber-500/30 text-amber-300",
  danger: "bg-rose-500/20 border-rose-500/30 text-rose-300",
  info: "bg-cyan-500/20 border-cyan-500/30 text-cyan-300",
  accent: "bg-fuchsia-500/10 border-fuchsia-500/30 text-fuchsia-300",
};

const SIZE_CLASSES = {
  sm: "px-1.5 py-0.5 text-[10px]",
  md: "px-2.5 py-1 text-[11px]",
};

export default function Badge({
  children,
  variant = "default",
  size = "sm",
  className = "",
  ...props
}) {
  return (
    <span
      className={`
        inline-flex items-center gap-1 rounded-md border font-bold
        ${VARIANT_CLASSES[variant] || VARIANT_CLASSES.default}
        ${SIZE_CLASSES[size] || SIZE_CLASSES.sm}
        ${className}
      `.trim()}
      {...props}
    >
      {children}
    </span>
  );
}
