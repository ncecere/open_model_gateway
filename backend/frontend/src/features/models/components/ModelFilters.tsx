import { Search } from "lucide-react";
import { CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

export type StatusFilter = "all" | "enabled" | "disabled";

type ModelFiltersProps = {
  enabledCount: number;
  totalCount: number;
  searchTerm: string;
  onSearchTermChange: (value: string) => void;
  providerOptions: string[];
  providerFilter: string;
  onProviderFilterChange: (value: string) => void;
  statusFilter: StatusFilter;
  onStatusFilterChange: (value: StatusFilter) => void;
};

export function ModelFilters({
  enabledCount,
  totalCount,
  searchTerm,
  onSearchTermChange,
  providerOptions,
  providerFilter,
  onProviderFilterChange,
  statusFilter,
  onStatusFilterChange,
}: ModelFiltersProps) {
  return (
    <>
      <div>
        <CardTitle>Catalog editor</CardTitle>
        <p className="text-sm text-muted-foreground">
          {enabledCount} enabled · {totalCount} total
        </p>
      </div>
      <div className="flex w-full flex-col gap-2 sm:flex-row sm:items-center sm:justify-end">
        <div className="relative sm:w-64">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={searchTerm}
            onChange={(event) => onSearchTermChange(event.target.value)}
            placeholder="Search alias or provider"
            className="pl-9"
          />
        </div>
        <Select
          value={providerFilter}
          onValueChange={(value) => onProviderFilterChange(value)}
        >
          <SelectTrigger className="sm:w-48">
            <SelectValue placeholder="All providers" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All providers</SelectItem>
            {providerOptions.map((provider) => (
              <SelectItem key={provider} value={provider}>
                {provider}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={statusFilter}
          onValueChange={(value) =>
            onStatusFilterChange(value as StatusFilter)
          }
        >
          <SelectTrigger className="sm:w-40">
            <SelectValue placeholder="All statuses" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All statuses</SelectItem>
            <SelectItem value="enabled">Enabled</SelectItem>
            <SelectItem value="disabled">Disabled</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </>
  );
}
