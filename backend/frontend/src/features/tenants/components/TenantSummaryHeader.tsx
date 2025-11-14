import { PlusCircle, RefreshCcw } from "lucide-react";

import { Button } from "@/components/ui/button";

type TenantSummaryHeaderProps = {
  activeCount: number;
  totalCount: number;
  onRefresh: () => void;
  refreshing: boolean;
  onCreate: () => void;
};

export function TenantSummaryHeader({
  activeCount,
  totalCount,
  onRefresh,
  refreshing,
  onCreate,
}: TenantSummaryHeaderProps) {
  return (
    <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Tenants</h1>
        <p className="text-sm text-muted-foreground">
          {activeCount} active · {totalCount} total — manage lifecycle, memberships, and limits.
        </p>
      </div>
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="icon"
          onClick={onRefresh}
          disabled={refreshing}
        >
          <RefreshCcw className="h-4 w-4" />
        </Button>
        <Button onClick={onCreate}>
          <PlusCircle className="mr-2 h-4 w-4" />
          Create tenant
        </Button>
      </div>
    </div>
  );
}
