import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { listTenants, listPersonalTenants } from "@/api/tenants";
import {
  deleteFile,
  listFiles,
  type AdminFileRecord,
  downloadAdminFileContent,
} from "@/api/files";
import { useToast } from "@/hooks/use-toast";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { AdminFilesTable, FileDetailsDialog } from "@/features/files";
import { Separator } from "@/components/ui/separator";
import { getFileSettings } from "@/api/runtime-settings";

const FILE_PAGE_SIZE = 20;

export function FilesPage() {
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const tenantsQuery = useQuery({
    queryKey: ["tenants", "list"],
    queryFn: () => listTenants({ limit: 200 }),
  });
  const personalTenantsQuery = useQuery({
    queryKey: ["tenants", "personal"],
    queryFn: () => listPersonalTenants({ limit: 500 }),
  });

  const personalTenantIds = useMemo(() => {
    const ids = new Set<string>();
    personalTenantsQuery.data?.personal_tenants.forEach((tenant) =>
      ids.add(tenant.tenant_id),
    );
    return ids;
  }, [personalTenantsQuery.data]);

  const tenantOptions = useMemo(
    () => (tenantsQuery.data?.tenants ?? []).filter((tenant) => !personalTenantIds.has(tenant.id)),
    [tenantsQuery.data, personalTenantIds],
  );

  const [tenantFilter, setTenantFilter] = useState("all");
  const [purposeFilter, setPurposeFilter] = useState("all");
  const [stateFilter, setStateFilter] = useState("active");
  const [searchInput, setSearchInput] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [offset, setOffset] = useState(0);
  const [selectedFile, setSelectedFile] = useState<AdminFileRecord | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<AdminFileRecord | null>(null);

  useEffect(() => {
    const handle = setTimeout(() => {
      setSearchQuery(searchInput.trim());
      setOffset(0);
    }, 300);
    return () => clearTimeout(handle);
  }, [searchInput]);

  useEffect(() => {
    setOffset(0);
  }, [tenantFilter, purposeFilter, stateFilter]);

  const filesQuery = useQuery({
    queryKey: [
      "admin-files",
      tenantFilter,
      purposeFilter,
      stateFilter,
      searchQuery,
      offset,
      FILE_PAGE_SIZE,
    ],
    queryFn: () =>
      listFiles({
        tenant_id: tenantFilter === "all" ? undefined : tenantFilter,
        purpose: purposeFilter === "all" ? undefined : purposeFilter,
        state: stateFilter,
        q: searchQuery || undefined,
        limit: FILE_PAGE_SIZE,
        offset,
      }),
  });

  const deleteMutation = useMutation({
    mutationFn: (fileId: string) => deleteFile(fileId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-files"] });
      toast({ title: "File deleted" });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to delete file",
        description: "Try again in a moment.",
      });
    },
  });

  const files = filesQuery.data?.data ?? [];
  const total = filesQuery.data?.total ?? 0;
  const hasPrev = offset > 0;
  const hasNext = offset + files.length < total;

  const fileSettingsQuery = useQuery({
    queryKey: ["admin-file-settings"],
    queryFn: getFileSettings,
  });

  const formatDuration = (seconds?: number) => {
    if (!seconds || seconds <= 0) return null;
    const days = seconds / 86400;
    if (Number.isInteger(days)) {
      return `${days} day${days === 1 ? "" : "s"}`;
    }
    const hours = seconds / 3600;
    if (Number.isInteger(hours)) {
      return `${hours} hour${hours === 1 ? "" : "s"}`;
    }
    return `${seconds} seconds`;
  };

  const ttlDescription = useMemo(() => {
    const defaultTTL = formatDuration(fileSettingsQuery.data?.default_ttl_seconds);
    const maxTTL = formatDuration(fileSettingsQuery.data?.max_ttl_seconds);
    if (!defaultTTL || !maxTTL) {
      return "Review file uploads across tenants for auditing and compliance.";
    }
    return `Review file uploads across tenants (default TTL ${defaultTTL}, max ${maxTTL}).`;
  }, [fileSettingsQuery.data]);

  const handleDownload = (file: AdminFileRecord) => {
    downloadAdminFileContent(file.id);
  };

  const handleFiltersChange = (next: Partial<{ tenant: string; purpose: string; state: string }>) => {
    if (next.tenant !== undefined) {
      setTenantFilter(next.tenant);
    }
    if (next.purpose !== undefined) {
      setPurposeFilter(next.purpose);
    }
    if (next.state !== undefined) {
      setStateFilter(next.state);
    }
  };

  const handlePageChange = (direction: "next" | "prev") => {
    setOffset((current) => {
      if (direction === "next") {
        return current + FILE_PAGE_SIZE;
      }
      return Math.max(0, current - FILE_PAGE_SIZE);
    });
  };

  return (
    <div className="space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Files</h1>
        <p className="text-sm text-muted-foreground">{ttlDescription}</p>
      </header>

      <Separator />

      <AdminFilesTable
        tenants={tenantOptions}
        personalTenantIds={personalTenantIds}
        files={files}
        total={total}
        isLoading={tenantsQuery.isLoading || filesQuery.isLoading}
        hasPrev={hasPrev}
        hasNext={hasNext}
        filters={{ tenant: tenantFilter, purpose: purposeFilter, state: stateFilter, search: searchInput }}
        offset={offset}
        pageSize={FILE_PAGE_SIZE}
        onFiltersChange={handleFiltersChange}
        onSearchChange={setSearchInput}
        onLoadMore={handlePageChange}
        onViewDetails={setSelectedFile}
        onDelete={setDeleteTarget}
        onDownload={handleDownload}
      />

      <AlertDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete file</AlertDialogTitle>
            <AlertDialogDescription>
              This will move the file into a deleted state immediately. Downstream caches may continue to serve it briefly.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={deleteMutation.isPending}
              onClick={() => {
                if (!deleteTarget) return;
                deleteMutation.mutate(deleteTarget.id);
                setDeleteTarget(null);
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <FileDetailsDialog
        file={selectedFile}
        isPersonal={Boolean(selectedFile && personalTenantIds.has(selectedFile.tenant_id))}
        open={Boolean(selectedFile)}
        onOpenChange={(open) => !open && setSelectedFile(null)}
      />

    </div>
  );
}
