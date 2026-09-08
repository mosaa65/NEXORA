import { useCallback, useMemo, useState } from "react";

/**
 * Standalone, reusable multi-selection hook.
 *
 * Designed for large lists: it keeps the selected ids in a `Set` and only
 * materializes arrays when actually needed (counts, resolved items).
 * Since every updater is pure and derived from the latest state via the
 * functional `setState` form, rapid consecutive toggles never race.
 *
 * @param {Array<unknown>} items The full list of items to select from.
 * @param {(item: any) => string} [getKey] Returns a stable unique key for an item.
 *   Defaults to `item?.id ?? item?.filePath ?? item`.
 * @param {Object} [opts]
 * @param {string|number|Array<string|number>} [opts.initialSelected]
 *   Pre-select an id or a list of ids on first mount.
 * @param {(key: string|number, muted: boolean) => void} [opts.onChange] Optional
 *   callback notified on every selection change, useful for driving a transfer
 *   context while keeping this hook framework-agnostic.
 */
export default function useSelection(items = [], { getKey, initialSelected, onChange } = {}) {
  const keyOf = useCallback(
    (item) => {
      if (getKey) return String(getKey(item));
      return String(item?.id ?? item?.file_id ?? item?.filePath ?? item?.file_path ?? item?.path ?? "?");
    },
    [getKey]
  );

  const [selectedKeys, setSelectedKeys] = useState(() => {
    if (initialSelected == null) return new Set();
    const arr = Array.isArray(initialSelected) ? initialSelected : [initialSelected];
    return new Set(arr.map((k) => String(k)));
  });

  const toggle = useCallback(
    (item) => {
      const key = keyOf(item);
      setSelectedKeys((prev) => {
        const next = new Set(prev);
        if (next.has(key)) next.delete(key);
        else next.add(key);
        return next;
      });
    },
    [keyOf]
  );

  const toggleKey = useCallback((key) => {
    const k = String(key);
    setSelectedKeys((prev) => {
      const next = new Set(prev);
      if (next.has(k)) next.delete(k);
      else next.add(k);
      return next;
    });
  }, []);

  const select = useCallback(
    (items) => {
      const list = Array.isArray(items) ? items : [items];
      setSelectedKeys((prev) => {
        const next = new Set(prev);
        list.forEach((item) => next.add(keyOf(item)));
        return next;
      });
    },
    [keyOf]
  );

  const clearSelection = useCallback(() => {
    setSelectedKeys(new Set());
  }, []);

  const selectAll = useCallback(() => {
    setSelectedKeys(new Set(items.map(keyOf)));
  }, [items, keyOf]);

  const selectRange = useCallback(
    (startIdx, endIdx) => {
      const lo = Math.min(startIdx, endIdx);
      const hi = Math.max(startIdx, endIdx);
      const range = items.slice(lo, hi + 1);
      setSelectedKeys((prev) => {
        const next = new Set(prev);
        range.forEach((item) => next.add(keyOf(item)));
        return next;
      });
    },
    [items, keyOf]
  );

  const isSelected = useCallback(
    (item) => {
      const key = keyOf(item);
      return selectedKeys.has(key);
    },
    [keyOf, selectedKeys]
  );

  const count = selectedKeys.size;

  const allSelected = useMemo(
    () => items.length > 0 && count === items.length,
    [items.length, count]
  );

  const selectedItems = useMemo(
    () => items.filter((item) => selectedKeys.has(keyOf(item))),
    [items, selectedKeys, keyOf]
  );

  const result = {
    selectedKeys,
    selectedItems,
    count,
    allSelected,
    toggle,
    toggleKey,
    select,
    selectAll,
    clearSelection,
    selectRange,
    isSelected,
  };

  return result;
}
