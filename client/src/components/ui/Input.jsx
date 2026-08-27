import React from "react";

/**
 * Unified Input component for the NEXORA design system.
 *
 * @param {"text"|"number"|"password"|"email"|"search"} type
 * @param {"sm"|"md"|"lg"} size
 * @param {string} label - Optional label text
 * @param {string} dir - Text direction (default: inherits)
 * @param {boolean} mono - Use monospace font
 */
const SIZE_CLASSES = {
  sm: "px-3 py-1.5 text-xs rounded-xl",
  md: "px-3.5 py-2.5 text-sm rounded-xl",
  lg: "px-4 py-3.5 text-sm rounded-2xl",
};

export default function Input({
  label,
  size = "md",
  mono = false,
  className = "",
  id,
  ...props
}) {
  const inputId = id || (label ? label.replace(/\s+/g, "-").toLowerCase() : undefined);

  return (
    <div className="space-y-1">
      {label && (
        <label
          htmlFor={inputId}
          className="block text-xs font-bold text-[var(--text-secondary)]"
        >
          {label}
        </label>
      )}
      <input
        id={inputId}
        className={`
          w-full border border-[var(--border-default)] bg-[var(--bg-input)]
          text-[var(--text-primary)] outline-none
          placeholder:text-[var(--text-muted)]
          focus:border-[var(--color-accent)] focus:ring-1 focus:ring-[var(--color-accent)]
          transition duration-200
          ${mono ? "font-mono" : ""}
          ${SIZE_CLASSES[size] || SIZE_CLASSES.md}
          ${className}
        `.trim()}
        {...props}
      />
    </div>
  );
}

/**
 * Unified Textarea component.
 */
export function Textarea({
  label,
  rows = 3,
  className = "",
  id,
  ...props
}) {
  const textareaId = id || (label ? label.replace(/\s+/g, "-").toLowerCase() : undefined);

  return (
    <div className="space-y-1">
      {label && (
        <label
          htmlFor={textareaId}
          className="block text-xs font-bold text-[var(--text-secondary)]"
        >
          {label}
        </label>
      )}
      <textarea
        id={textareaId}
        rows={rows}
        className={`
          w-full rounded-xl border border-[var(--border-default)] bg-[var(--bg-input)]
          p-3 text-xs text-[var(--text-primary)] outline-none
          placeholder:text-[var(--text-muted)]
          focus:border-[var(--color-accent)]
          transition duration-200
          ${className}
        `.trim()}
        {...props}
      />
    </div>
  );
}

/**
 * Unified Select component.
 */
export function Select({
  label,
  options = [],
  size = "md",
  className = "",
  id,
  ...props
}) {
  const selectId = id || (label ? label.replace(/\s+/g, "-").toLowerCase() : undefined);

  return (
    <div className="space-y-1">
      {label && (
        <label
          htmlFor={selectId}
          className="block text-[11px] font-bold text-[var(--text-muted)]"
        >
          {label}
        </label>
      )}
      <select
        id={selectId}
        className={`
          w-full border border-[var(--border-default)] bg-[var(--bg-input)]
          text-[var(--text-primary)] outline-none
          focus:border-[var(--color-accent)]
          ${SIZE_CLASSES[size] || SIZE_CLASSES.md}
          ${className}
        `.trim()}
        {...props}
      >
        {options.map((opt) => (
          <option key={opt.value ?? opt.id} value={opt.value ?? opt.id} className="bg-[var(--bg-card)] text-[var(--text-primary)]">
            {opt.label}
          </option>
        ))}
      </select>
    </div>
  );
}
