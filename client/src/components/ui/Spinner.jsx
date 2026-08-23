import React from "react";

/**
 * Unified Spinner component for loading states.
 *
 * @param {"sm"|"md"|"lg"} size
 * @param {string} label - Optional loading text
 */
const SIZE_CLASSES = {
  sm: "h-4 w-4 border-2",
  md: "h-8 w-8 border-4",
  lg: "h-12 w-12 border-4",
};

export default function Spinner({
  size = "md",
  label,
  className = "",
}) {
  return (
    <div className={`flex flex-col items-center justify-center gap-3 ${className}`}>
      <div
        className={`
          animate-spin rounded-full
          border-fuchsia-500 border-t-transparent
          ${SIZE_CLASSES[size] || SIZE_CLASSES.md}
        `.trim()}
      />
      {label && (
        <p className="text-sm font-bold text-white/60">{label}</p>
      )}
    </div>
  );
}
