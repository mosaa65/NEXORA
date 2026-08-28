import React from "react";
import Icon from "../Icon.jsx";

/**
 * Unified Modal component for dialogs, confirmations, and forms.
 *
 * @param {boolean} isOpen - Controls visibility
 * @param {Function} onClose - Called when overlay is clicked or close button pressed
 * @param {string} title - Modal header title
 * @param {"sm"|"md"|"lg"|"xl"} size - Max width
 * @param {"default"|"danger"} variant - Border accent color
 */
const SIZE_CLASSES = {
  sm: "max-w-md",
  md: "max-w-lg",
  lg: "max-w-2xl",
  xl: "max-w-4xl",
};

const VARIANT_BORDER = {
  default: "border-fuchsia-500/30",
  danger: "border-rose-500/30",
};

export default function Modal({
  isOpen,
  onClose,
  title,
  subtitle,
  size = "md",
  variant = "default",
  children,
  actions,
  className = "",
}) {
  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-[var(--z-modal)] flex items-center justify-center overflow-hidden bg-[var(--bg-overlay)] p-0 backdrop-blur-md animate-fadeIn sm:p-4"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose?.();
      }}
    >
      <div
        className={`
          relative flex max-h-[100dvh] w-full flex-col overflow-hidden rounded-none border bg-[var(--bg-card)]
          shadow-2xl sm:max-h-[calc(100dvh-2rem)] sm:rounded-3xl
          ${SIZE_CLASSES[size] || SIZE_CLASSES.md}
          ${VARIANT_BORDER[variant] || VARIANT_BORDER.default}
          ${className}
        `.trim()}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        {(title || onClose) && (
          <div className="flex shrink-0 items-start justify-between gap-4 border-b border-[var(--border-default)] bg-[var(--bg-card)] px-5 py-4 sm:px-7">
            <div>
              {title && (
                <h3 className="text-xl font-bold text-[var(--text-primary)]">
                  {title}
                </h3>
              )}
              {subtitle && (
                <p className="text-xs text-[var(--text-muted)] mt-1">
                  {subtitle}
                </p>
              )}
            </div>
            {onClose && (
              <button
                type="button"
                onClick={onClose}
                aria-label="إغلاق النافذة"
                title="إغلاق"
                className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-[var(--border-default)] bg-[var(--bg-elevated)] text-[var(--text-muted)] transition hover:border-[var(--color-accent)] hover:bg-[var(--color-accent-light)] hover:text-[var(--text-primary)]"
              >
                <Icon name="close" className="h-4 w-4" />
              </button>
            )}
          </div>
        )}

        {/* Content */}
        <div className="min-h-0 flex-1 overflow-y-auto px-5 py-5 sm:px-7 sm:py-6">{children}</div>

        {/* Footer Actions */}
        {actions && (
          <div className="flex shrink-0 flex-wrap items-center justify-end gap-2 border-t border-[var(--border-default)] bg-[var(--bg-card)] px-5 py-4 sm:gap-3 sm:px-7 [&>button]:rounded-full [&>button]:max-sm:flex-1">
            {actions}
          </div>
        )}
      </div>
    </div>
  );
}

/**
 * Pre-built confirmation modal for delete/dangerous actions.
 */
export function ConfirmModal({
  isOpen,
  onClose,
  onConfirm,
  title = "تأكيد",
  message,
  confirmText = "نعم، تأكيد",
  cancelText = "إلغاء",
  loading = false,
  variant = "danger",
}) {
  if (!isOpen) return null;

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="sm" variant={variant}>
      <div className="text-center space-y-4">
        <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-rose-500/20 text-2xl text-rose-400">
          🗑️
        </div>
        <h3 className="text-xl font-bold text-[var(--text-primary)]">{title}</h3>
        {message && (
          <p className="text-xs text-[var(--text-secondary)] leading-relaxed">{message}</p>
        )}
        <div className="flex items-center justify-center gap-3 pt-2">
          <button
            type="button"
            onClick={onClose}
            className="px-5 py-2.5 rounded-xl border border-[var(--border-default)] bg-[var(--bg-surface)] text-xs font-bold text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition"
          >
            {cancelText}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={loading}
            className="px-6 py-2.5 rounded-xl bg-rose-600 hover:bg-rose-700 text-xs font-bold text-white shadow-lg shadow-rose-900/50 disabled:opacity-50"
          >
            {loading ? "جارٍ..." : confirmText}
          </button>
        </div>
      </div>
    </Modal>
  );
}
