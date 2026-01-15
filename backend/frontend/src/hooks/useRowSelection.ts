import { useCallback, useMemo, useState } from "react";

export interface RowSelectionState<T> {
  /** Set of selected row IDs. */
  selectedIds: Set<string>;
  /** Whether all visible rows are selected. */
  isAllSelected: boolean;
  /** Whether some (but not all) rows are selected. */
  isIndeterminate: boolean;
  /** Number of selected rows. */
  selectedCount: number;
  /** Whether a specific row is selected. */
  isSelected: (id: string) => boolean;
  /** Toggle selection of a specific row. */
  toggleRow: (id: string) => void;
  /** Select multiple rows. */
  selectRows: (ids: string[]) => void;
  /** Deselect multiple rows. */
  deselectRows: (ids: string[]) => void;
  /** Toggle all visible rows. */
  toggleAll: () => void;
  /** Clear all selections. */
  clearSelection: () => void;
  /** Get the selected items from the data array. */
  getSelectedItems: () => T[];
}

interface UseRowSelectionOptions<T> {
  /** Function to get a unique ID from each item. */
  getItemId: (item: T) => string;
  /** The current data array. */
  data: T[];
}

/**
 * Hook for managing row selection state in tables.
 * Supports single, multi, and all selection modes.
 */
export function useRowSelection<T>({
  getItemId,
  data,
}: UseRowSelectionOptions<T>): RowSelectionState<T> {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  const visibleIds = useMemo(
    () => new Set(data.map(getItemId)),
    [data, getItemId]
  );

  const isAllSelected = useMemo(
    () => visibleIds.size > 0 && [...visibleIds].every((id) => selectedIds.has(id)),
    [visibleIds, selectedIds]
  );

  const isIndeterminate = useMemo(
    () => !isAllSelected && [...visibleIds].some((id) => selectedIds.has(id)),
    [visibleIds, selectedIds, isAllSelected]
  );

  const selectedCount = useMemo(
    () => [...selectedIds].filter((id) => visibleIds.has(id)).length,
    [selectedIds, visibleIds]
  );

  const isSelected = useCallback(
    (id: string) => selectedIds.has(id),
    [selectedIds]
  );

  const toggleRow = useCallback((id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  const selectRows = useCallback((ids: string[]) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      ids.forEach((id) => next.add(id));
      return next;
    });
  }, []);

  const deselectRows = useCallback((ids: string[]) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      ids.forEach((id) => next.delete(id));
      return next;
    });
  }, []);

  const toggleAll = useCallback(() => {
    setSelectedIds((prev) => {
      if (isAllSelected) {
        // Deselect all visible
        const next = new Set(prev);
        visibleIds.forEach((id) => next.delete(id));
        return next;
      } else {
        // Select all visible
        const next = new Set(prev);
        visibleIds.forEach((id) => next.add(id));
        return next;
      }
    });
  }, [isAllSelected, visibleIds]);

  const clearSelection = useCallback(() => {
    setSelectedIds(new Set());
  }, []);

  const getSelectedItems = useCallback(() => {
    return data.filter((item) => selectedIds.has(getItemId(item)));
  }, [data, selectedIds, getItemId]);

  return {
    selectedIds,
    isAllSelected,
    isIndeterminate,
    selectedCount,
    isSelected,
    toggleRow,
    selectRows,
    deselectRows,
    toggleAll,
    clearSelection,
    getSelectedItems,
  };
}
