import { useMemo, useState } from "react";
import { ChevronDown, X } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";

export type ComparisonOption = {
  value: string;
  label: string;
  hint?: string;
};

export type ComparisonSelectorProps = {
  title: string;
  description?: string;
  options: ComparisonOption[];
  selected: string[];
  onToggle: (value: string) => void;
  loading?: boolean;
};

export function ComparisonSelector({
  title,
  description,
  options,
  selected,
  onToggle,
  loading,
}: ComparisonSelectorProps) {
  const [query, setQuery] = useState("");

  const filtered = useMemo(() => {
    if (!query) {
      return options;
    }
    const value = query.trim().toLowerCase();
    if (!value) {
      return options;
    }
    return options.filter((option) => {
      const label = option.label.toLowerCase();
      const hint = option.hint?.toLowerCase() ?? "";
      return label.includes(value) || hint.includes(value);
    });
  }, [options, query]);

  const summary = selected.length
    ? `${selected.length} selected`
    : `Select ${title.toLowerCase()}`;

  return (
    <div className="space-y-3">
      <div>
        <p className="text-sm font-medium">{title}</p>
        {description ? (
          <p className="text-xs text-muted-foreground">{description}</p>
        ) : null}
      </div>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" className="w-full justify-between">
            <span className="truncate text-left text-sm">{summary}</span>
            <ChevronDown className="ml-2 h-4 w-4 opacity-50" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent className="w-72 p-0" align="start">
          <div className="border-b p-2">
            <Input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={`Search ${title.toLowerCase()}`}
              className="h-8 text-sm"
            />
          </div>
          <div className="max-h-64 overflow-y-auto">
            {loading ? (
              <div className="space-y-2 p-2">
                {[...Array(4)].map((_, idx) => (
                  <Skeleton key={idx} className="h-8 w-full" />
                ))}
              </div>
            ) : filtered.length ? (
              filtered.map((option) => {
                const isSelected = selected.includes(option.value);
                return (
                  <DropdownMenuCheckboxItem
                    key={option.value}
                    checked={isSelected}
                    className="flex items-center justify-between text-sm"
                    onCheckedChange={() => onToggle(option.value)}
                  >
                    <span className="truncate">{option.label}</span>
                    {option.hint ? (
                      <span className="ml-2 text-xs text-muted-foreground">
                        {option.hint}
                      </span>
                    ) : null}
                  </DropdownMenuCheckboxItem>
                );
              })
            ) : (
              <p className="p-3 text-sm text-muted-foreground">
                {query ? `No results for "${query}".` : "No options available."}
              </p>
            )}
          </div>
        </DropdownMenuContent>
      </DropdownMenu>
      {selected.length ? (
        <div className="flex flex-wrap gap-2">
          {selected.map((value) => {
            const option = options.find((opt) => opt.value === value);
            return (
              <Badge key={value} variant="secondary" className="flex items-center gap-1">
                {option?.label ?? value}
                <button
                  type="button"
                  className="rounded bg-transparent p-0.5 text-muted-foreground hover:text-foreground"
                  onClick={() => onToggle(value)}
                  aria-label="Remove selection"
                >
                  <X className="h-3 w-3" />
                </button>
              </Badge>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
