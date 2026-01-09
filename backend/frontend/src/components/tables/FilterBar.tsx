import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Search } from "lucide-react";
import { cn } from "@/lib/utils";

// ============================================================================
// Types
// ============================================================================

export type FilterOption = {
  value: string;
  label: string;
};

export type SelectFilterConfig = {
  type: "select";
  /** Unique key for this filter */
  key: string;
  /** Label displayed above the filter */
  label?: string;
  /** Current value */
  value: string;
  /** Called when value changes */
  onChange: (value: string) => void;
  /** Available options */
  options: FilterOption[];
  /** Placeholder text */
  placeholder?: string;
  /** Whether the filter is disabled */
  disabled?: boolean;
  /** Whether the filter is loading */
  loading?: boolean;
  /** Custom width class */
  className?: string;
};

export type SearchFilterConfig = {
  type: "search";
  /** Unique key for this filter */
  key: string;
  /** Label displayed above the filter */
  label?: string;
  /** Current value */
  value: string;
  /** Called when value changes */
  onChange: (value: string) => void;
  /** Placeholder text */
  placeholder?: string;
  /** Whether the filter is disabled */
  disabled?: boolean;
  /** Custom width class */
  className?: string;
};

export type FilterConfig = SelectFilterConfig | SearchFilterConfig;

export type FilterBarProps = {
  /** Array of filter configurations */
  filters: FilterConfig[];
  /** Layout style: 'inline' renders horizontally, 'stacked' uses grid */
  layout?: "inline" | "stacked";
  /** Number of columns for stacked layout (default: filters.length) */
  columns?: number;
  /** Additional class name */
  className?: string;
};

// ============================================================================
// Component
// ============================================================================

/**
 * FilterBar renders a set of filters for tables.
 * Supports search inputs and select dropdowns with flexible layouts.
 */
export function FilterBar({
  filters,
  layout = "inline",
  columns,
  className,
}: FilterBarProps) {
  const gridCols = columns ?? filters.length;

  return (
    <div
      className={cn(
        layout === "inline"
          ? "flex w-full flex-col gap-2 sm:flex-row sm:items-center"
          : `grid gap-4`,
        layout === "stacked" && gridCols === 2 && "md:grid-cols-2",
        layout === "stacked" && gridCols === 3 && "md:grid-cols-3",
        layout === "stacked" && gridCols >= 4 && "lg:grid-cols-4",
        className
      )}
    >
      {filters.map((filter) => (
        <FilterField key={filter.key} filter={filter} layout={layout} />
      ))}
    </div>
  );
}

// ============================================================================
// Filter Field
// ============================================================================

type FilterFieldProps = {
  filter: FilterConfig;
  layout: "inline" | "stacked";
};

function FilterField({ filter, layout }: FilterFieldProps) {
  const showLabel = layout === "stacked" && filter.label;

  if (filter.type === "search") {
    return (
      <div className={cn("space-y-2", filter.className)}>
        {showLabel && (
          <p className="text-sm font-medium text-muted-foreground">
            {filter.label}
          </p>
        )}
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={filter.value}
            onChange={(e) => filter.onChange(e.target.value)}
            placeholder={filter.placeholder}
            disabled={filter.disabled}
            className="pl-9"
          />
        </div>
      </div>
    );
  }

  if (filter.type === "select") {
    if (filter.loading) {
      return (
        <div className={cn("space-y-2", filter.className)}>
          {showLabel && (
            <p className="text-sm font-medium text-muted-foreground">
              {filter.label}
            </p>
          )}
          <Skeleton className="h-10 w-full" />
        </div>
      );
    }

    return (
      <div className={cn("space-y-2", filter.className)}>
        {showLabel && (
          <p className="text-sm font-medium text-muted-foreground">
            {filter.label}
          </p>
        )}
        <Select
          value={filter.value}
          onValueChange={filter.onChange}
          disabled={filter.disabled}
        >
          <SelectTrigger>
            <SelectValue placeholder={filter.placeholder} />
          </SelectTrigger>
          <SelectContent>
            {filter.options.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    );
  }

  return null;
}

// ============================================================================
// Convenience Creators
// ============================================================================

/**
 * Creates a search filter configuration.
 */
export function createSearchFilter(
  key: string,
  value: string,
  onChange: (value: string) => void,
  options?: Partial<Omit<SearchFilterConfig, "type" | "key" | "value" | "onChange">>
): SearchFilterConfig {
  return {
    type: "search",
    key,
    value,
    onChange,
    ...options,
  };
}

/**
 * Creates a select filter configuration.
 */
export function createSelectFilter(
  key: string,
  value: string,
  onChange: (value: string) => void,
  filterOptions: FilterOption[],
  options?: Partial<Omit<SelectFilterConfig, "type" | "key" | "value" | "onChange" | "options">>
): SelectFilterConfig {
  return {
    type: "select",
    key,
    value,
    onChange,
    options: filterOptions,
    ...options,
  };
}
