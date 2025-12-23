import { useEffect } from "react";

type DefaultSelectionOptions<Item, Value> = {
  items: Item[];
  selected: Value | undefined;
  onChange: (value: Value | undefined) => void;
  getValue: (item: Item) => Value;
  getDefault?: (items: Item[]) => Value | undefined;
  selectFirst?: boolean;
};

export function useDefaultSelection<Item, Value>({
  items,
  selected,
  onChange,
  getValue,
  getDefault,
  selectFirst = true,
}: DefaultSelectionOptions<Item, Value>) {
  useEffect(() => {
    if (!items.length) {
      if (selected !== undefined) {
        onChange(undefined);
      }
      return;
    }

    const isValid =
      selected !== undefined &&
      items.some((item) => getValue(item) === selected);
    if (isValid) {
      return;
    }

    const next = getDefault
      ? getDefault(items)
      : selectFirst
        ? getValue(items[0])
        : undefined;

    if (next === undefined) {
      if (selected !== undefined) {
        onChange(undefined);
      }
      return;
    }

    if (next !== selected) {
      onChange(next);
    }
  }, [items, selected, onChange, getValue, getDefault, selectFirst]);
}
