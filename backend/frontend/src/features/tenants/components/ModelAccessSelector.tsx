import type { ModelCatalogEntry } from "@/api/model-catalog";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Skeleton } from "@/components/ui/skeleton";

interface ModelAccessSelectorProps {
  title: string;
  description?: string;
  models: ModelCatalogEntry[];
  selected: string[];
  onToggle: (alias: string, checked: boolean) => void;
  onSelectAll: () => void;
  onClear: () => void;
  isLoading?: boolean;
  disabled?: boolean;
}

export function ModelAccessSelector({
  title,
  description,
  models,
  selected,
  onToggle,
  onSelectAll,
  onClear,
  isLoading,
  disabled,
}: ModelAccessSelectorProps) {
  const selectionSummary = `${selected.length} of ${models.length} models selected`;

  return (
    <div className="space-y-3">
      <div className="flex flex-col gap-2">
        <div className="flex flex-col gap-1 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <p className="text-sm font-medium leading-none">{title}</p>
            {description ? (
              <p className="text-sm text-muted-foreground">{description}</p>
            ) : null}
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={onSelectAll}
              disabled={disabled || models.length === 0}
            >
              Select all
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={onClear}
              disabled={disabled || selected.length === 0}
            >
              Clear
            </Button>
          </div>
        </div>
        <p className="text-xs text-muted-foreground">{selectionSummary}</p>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      ) : models.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No models available. Add catalog entries first.
        </p>
      ) : (
        <div className="max-h-60 divide-y overflow-y-auto rounded-md border">
          {models.map((entry) => {
            const checked = selected.includes(entry.alias);
            return (
              <label
                key={entry.alias}
                htmlFor={`model-access-${entry.alias}`}
                className="flex items-center justify-between gap-3 p-3 text-sm"
              >
                <div className="flex items-center gap-3">
                  <Checkbox
                    id={`model-access-${entry.alias}`}
                    checked={checked}
                    onCheckedChange={(value) =>
                      onToggle(entry.alias, value === true)
                    }
                    disabled={disabled}
                  />
                  <div>
                    <p className="font-medium">{entry.alias}</p>
                    <p className="text-xs text-muted-foreground">
                      {entry.provider} · {entry.provider_model}
                    </p>
                  </div>
                </div>
                {!entry.enabled ? (
                  <span className="text-xs text-muted-foreground">disabled</span>
                ) : null}
              </label>
            );
          })}
        </div>
      )}
    </div>
  );
}
