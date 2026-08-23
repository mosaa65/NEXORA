import React from "react";

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
      className="fixed inset-0 z-[var(--z-modal)] flex items-center justify-center bg-[var(--bg-overlay)] p-3 sm:p-6 backdrop-blur-md overflow-y-auto animate-fadeIn"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose?.();
      }}
    >
      <div
        className={`
          relative w-full rounded-3xl border bg-[var(--bg-card)]
          p-6 sm:p-8 shadow-2xl space-y-5
          max-h-[92vh] overflow-y-auto
          ${SIZE_CLASSES[size] || SIZE_CLASSES.md}
          ${VARIANT_BORDER[variant] || VARIANT_BORDER.default}
          ${className}
        `.trim()}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        {(title || onClose) && (
          <div className="flex items-center justify-between border-b border-[var(--border-default)] pb-3">
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
                className="rounded-xl border border-[var(--border-default)] p-1.5 text-white/60 hover:text-white transition"
              >
                ✕
              </button>
            )}
          </div>
        )}

        {/* Content */}
        <div>{children}</div>

        {/* Footer Actions */}
        {actions && (
          <div className="flex items-center justify-end gap-3 border-t border-[var(--border-default)] pt-4">
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
        <h3 className="text-xl font-bold text-white">{title}</h3>
        {message && (
          <p className="text-xs text-white/70 leading-relaxed">{message}</p>
        )}
        <div className="flex items-center justify-center gap-3 pt-2">
          <button
            type="button"
            onClick={onClose}
            className="px-5 py-2.5 rounded-xl border border-white/10 bg-white/[0.05] text-xs font-bold text-white"
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
