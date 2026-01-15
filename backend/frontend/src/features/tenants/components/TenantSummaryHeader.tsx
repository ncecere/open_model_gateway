import { Plus, RefreshCcw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/layouts";

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
    <PageHeader
      title="Tenants"
      description={`${activeCount} active · ${totalCount} total — manage lifecycle, memberships, and limits.`}
      actions={
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="icon"
            onClick={onRefresh}
            loading={refreshing}
          >
            <RefreshCcw className="h-4 w-4" />
          </Button>
          <Button onClick={onCreate}>
            <Plus className="h-4 w-4" />
            Create Tenant
          </Button>
        </div>
      }
    />
  );
}
