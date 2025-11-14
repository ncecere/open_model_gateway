import { useEffect, useMemo, useState } from "react";
import { useUserFilesQuery, useUserTenantsQuery } from "../hooks/useUserData";
import type { UserFileRecord } from "../../../api/user/files";
import {
  UserFilesTable,
  UserFileDetailsDialog,
  useUserFileFilters,
} from "@/features/files";

export function UserFilesPage() {
  const { data: tenants, isLoading: tenantsLoading } = useUserTenantsQuery();
  const tenantOptions = tenants ?? [];
  const [selectedTenantId, setSelectedTenantId] = useState<string | undefined>(undefined);
  const [selectedFile, setSelectedFile] = useState<UserFileRecord | null>(null);

  useEffect(() => {
    if (selectedTenantId || tenantOptions.length === 0) {
      return;
    }
    const personal = tenantOptions.find((tenant) => tenant.is_personal);
    setSelectedTenantId(personal?.tenant_id ?? tenantOptions[0]?.tenant_id);
  }, [selectedTenantId, tenantOptions]);

  const filesQuery = useUserFilesQuery(selectedTenantId);
  const files = filesQuery.data ?? [];

  const {
    searchTerm,
    setSearchTerm,
    purposeFilter,
    setPurposeFilter,
    purposeOptions,
    filteredFiles,
  } = useUserFileFilters(files);

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

  return (
    <div className="space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Files</h1>
        <p className="text-sm text-muted-foreground">
          Review uploads tied to your personal or tenant accounts. Files inherit the tenant’s TTL, so most entries expire automatically.
        </p>
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
        isLoading={filesQuery.isLoading}
        onViewFile={setSelectedFile}
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
