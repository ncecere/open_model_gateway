import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useToast } from "@/hooks/use-toast";
import { useUserFilesQuery, useUserTenantsQuery } from "../hooks/useUserData";
import type { UserFileRecord } from "../../../api/user/files";
import { downloadUserFile } from "../../../api/user/files";
import { getUserFileSettings } from "../../../api/user/settings";
import {
  UserFilesTable,
  UserFileDetailsDialog,
  useUserFileFilters,
} from "@/features/files";

export function UserFilesPage() {
  const { toast } = useToast();
  const { data: tenants, isLoading: tenantsLoading } = useUserTenantsQuery();
  const tenantOptions = tenants ?? [];
  const [selectedTenantId, setSelectedTenantId] = useState<string | undefined>(undefined);
  const [selectedFile, setSelectedFile] = useState<UserFileRecord | null>(null);
  const [files, setFiles] = useState<UserFileRecord[]>([]);
  const [afterCursor, setAfterCursor] = useState<string | undefined>(undefined);
  const [nextCursor, setNextCursor] = useState<string | undefined>(undefined);
  const [hasMore, setHasMore] = useState(false);

  useEffect(() => {
    if (selectedTenantId || tenantOptions.length === 0) {
      return;
    }
    const personal = tenantOptions.find((tenant) => tenant.is_personal);
    setSelectedTenantId(personal?.tenant_id ?? tenantOptions[0]?.tenant_id);
  }, [selectedTenantId, tenantOptions]);

  const {
    searchTerm,
    setSearchTerm,
    purposeFilter,
    setPurposeFilter,
    purposeOptions,
    filteredFiles,
  } = useUserFileFilters(files);

  const effectivePurpose = purposeFilter === "all" ? undefined : purposeFilter;

  useEffect(() => {
    setFiles([]);
    setAfterCursor(undefined);
    setNextCursor(undefined);
    setHasMore(false);
  }, [selectedTenantId, effectivePurpose]);

  const filesQuery = useUserFilesQuery(selectedTenantId, {
    limit: 25,
    after: afterCursor,
    purpose: effectivePurpose,
  });

  useEffect(() => {
    if (!filesQuery.data) {
      return;
    }
    setFiles((prev) => (afterCursor ? [...prev, ...filesQuery.data.files] : filesQuery.data.files));
    setHasMore(filesQuery.data.has_more);
    setNextCursor(filesQuery.data.next_cursor);
  }, [filesQuery.data, afterCursor]);

  const fileSettingsQuery = useQuery({
    queryKey: ["user-file-settings"],
    queryFn: () => getUserFileSettings(),
  });

  const formatDuration = (seconds?: number) => {
    if (!seconds || seconds <= 0) {
      return null;
    }
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

  const tenantLabelForFile = useMemo(() => {
    if (!selectedFile) {
      return undefined;
    }
    const tenant = tenantOptions.find((t) => t.tenant_id === selectedFile.tenant_id);
    if (!tenant) {
      return selectedFile.tenant_id;
    }
    return tenant.is_personal ? "Personal" : tenant.name;
  }, [selectedFile, tenantOptions]);

  const ttlDescription = useMemo(() => {
    const defaultTTL = formatDuration(fileSettingsQuery.data?.default_ttl_seconds);
    const maxTTL = formatDuration(fileSettingsQuery.data?.max_ttl_seconds);
    if (!defaultTTL || !maxTTL) {
      return "Files inherit the tenant’s TTL. Most entries expire automatically.";
    }
    return `Files inherit the tenant’s TTL (default ${defaultTTL}, max ${maxTTL}). Most entries expire automatically.`;
  }, [fileSettingsQuery.data]);

  const handleLoadMore = () => {
    if (!hasMore || !nextCursor) {
      return;
    }
    setAfterCursor(nextCursor);
  };

  const handleDownload = async (file: UserFileRecord) => {
    if (!selectedTenantId) {
      return;
    }
    try {
      const response = await downloadUserFile(selectedTenantId, file.id);
      const blob = new Blob([response.data], { type: file.content_type });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = file.filename;
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);
    } catch (error) {
      toast({
        variant: "destructive",
        title: "Download failed",
        description: "Unable to download file. Try again later.",
      });
    }
  };

  const isInitialLoading = filesQuery.isLoading && !afterCursor;
  const isFetchingMore = filesQuery.isFetching && Boolean(afterCursor);

  return (
    <div className="space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Files</h1>
        <p className="text-sm text-muted-foreground">{ttlDescription}</p>
      </header>

      <UserFilesTable
        tenants={tenantOptions}
        tenantsLoading={tenantsLoading}
        selectedTenantId={selectedTenantId}
        onTenantChange={setSelectedTenantId}
        searchTerm={searchTerm}
        onSearchChange={setSearchTerm}
        purposeFilter={purposeFilter}
        onPurposeChange={setPurposeFilter}
        purposeOptions={purposeOptions}
        files={filteredFiles}
        isLoading={isInitialLoading}
        isFetchingMore={isFetchingMore}
        hasMore={hasMore}
        onLoadMore={handleLoadMore}
        onViewFile={setSelectedFile}
        onDownload={handleDownload}
      />

      <UserFileDetailsDialog
        file={selectedFile}
        tenantLabel={tenantLabelForFile}
        open={Boolean(selectedFile)}
        onOpenChange={(open) => !open && setSelectedFile(null)}
      />
    </div>
  );
}
