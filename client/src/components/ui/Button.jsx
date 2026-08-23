import React from "react";

/**
 * Unified Button component for the NEXORA design system.
 *
 * @param {"primary"|"secondary"|"ghost"|"danger"|"success"|"accent"} variant
 * @param {"xs"|"sm"|"md"|"lg"} size
 * @param {React.ReactNode} icon - Optional leading icon
 * @param {boolean} loading - Shows spinner when true
 * @param {boolean} disabled
 * @param {string} className - Additional classes
 */
const VARIANT_CLASSES = {
  primary:
    "bg-gradient-to-r from-purple-600 to-fuchsia-600 text-white shadow-lg shadow-purple-900/50 hover:brightness-110",
  secondary:
    "bg-white/[0.06] border border-white/10 text-white hover:bg-white/10",
  ghost:
    "bg-transparent text-white/70 hover:bg-white/[0.06] hover:text-white",
  danger:
    "bg-rose-600 text-white hover:bg-rose-700 shadow-lg shadow-rose-900/50",
  success:
    "bg-gradient-to-r from-emerald-600 via-teal-600 to-emerald-700 text-white shadow-lg shadow-emerald-900/40 hover:brightness-110",
  accent:
    "border border-fuchsia-400/30 bg-fuchsia-500/10 text-fuchsia-100 hover:bg-fuchsia-500/20",
};

const SIZE_CLASSES = {
  xs: "px-2.5 py-1 text-[11px] rounded-lg",
  sm: "px-3 py-1.5 text-xs rounded-xl",
  md: "px-5 py-2.5 text-xs rounded-2xl",
  lg: "px-7 py-3.5 text-sm rounded-2xl",
};

export default function Button({
  children,
  variant = "primary",
  size = "md",
  icon,
  loading = false,
  disabled = false,
  className = "",
  type = "button",
  ...props
}) {
  const isDisabled = disabled || loading;

  return (
    <button
      type={type}
      disabled={isDisabled}
      className={`
        inline-flex items-center justify-center gap-2 font-bold
        transition duration-200
        disabled:opacity-50 disabled:cursor-not-allowed
        ${VARIANT_CLASSES[variant] || VARIANT_CLASSES.primary}
        ${SIZE_CLASSES[size] || SIZE_CLASSES.md}
        ${className}
      `.trim()}
      {...props}
    >
      {loading ? (
        <span className="inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
      ) : icon ? (
        <span className="shrink-0">{icon}</span>
      ) : null}
      {children}
    </button>
  );
}
