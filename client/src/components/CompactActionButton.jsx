import Icon from "./Icon";

/**
 * Compact icon-only action button inspired by Windows 11 File Explorer's
 * toolbar. Modern, small, square button with a tooltip describing the action.
 * The action's purpose is communicated purely via the icon; the `title`
 * attribute keeps full clarity on hover.
 *
 * @param {Object} props
 * @param {string} props.icon Icon name (see Icon.jsx).
 * @param {string} [props.label] Falls back to the tooltip `title`.
 * @param {boolean} [props.active] When true, shows the "pressed"/selected state.
 * @param {boolean} [props.disabled]
 * @param {() => void} [props.onClick]
 */
export default function CompactActionButton({
  icon,
  label,
  active = false,
  disabled = false,
  onClick,
  className = "",
  title,
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      title={title || label}
      aria-label={title || label}
      aria-pressed={active}
      className={`group inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border transition select-none active:scale-95 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--color-accent)]
        ${
          active
            ? "border-[var(--color-accent)]/50 bg-[var(--bg-elevated)] text-[var(--color-accent)] shadow-[var(--shadow-sm)]"
            : "border-[var(--border-default)] bg-[var(--bg-surface)] text-[var(--text-secondary)] hover:border-[var(--color-accent)]/40 hover:text-[var(--text-primary)] hover:bg-[var(--bg-elevated)]"
        }
        ${disabled ? "pointer-events-none opacity-45" : ""} ${className}`}
    >
      <Icon name={icon} className="h-4 w-4" />
    </button>
  );
}