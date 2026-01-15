import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { RefreshCcw, Plus } from "lucide-react";
import { PageHeader } from "@/components/layouts";

import {
  deleteModel,
  listModelCatalog,
  listModelStatuses,
  type ModelCatalogEntry,
  type ModelCatalogUpsertRequest,
  upsertModel,
} from "@/api/model-catalog";
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
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { useToast } from "@/hooks/use-toast";
import { useTheme } from "@/providers/ThemeProvider";
import {
  BulkActionBar,
  ModelEditorDialog,
  ModelFilters,
  ModelTable,
  createEmptyModelForm,
  mapEntryToForm,
  type ModelFormState,
} from "@/features/models";

const CATALOG_QUERY_KEY = ["model-catalog"] as const;

export function ModelsPage() {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const { resolvedTheme } = useTheme();

  const catalogQuery = useQuery({
    queryKey: CATALOG_QUERY_KEY,
    queryFn: listModelCatalog,
    staleTime: 60_000,
  });

  const models = catalogQuery.data ?? [];
  const statusQuery = useQuery({
    queryKey: ["model-catalog", "status"],
    queryFn: listModelStatuses,
    refetchInterval: 60_000,
  });
  const statusMap = useMemo(() => {
    const map: Record<string, string> = {};
    statusQuery.data?.forEach((entry) => {
      map[entry.alias] = entry.status;
    });
    return map;
  }, [statusQuery.data]);
  const providerOptions = useMemo(() => {
    const unique = Array.from(new Set(models.map((model) => model.provider)));
    return unique.sort((a, b) => a.localeCompare(b));
  }, [models]);
  const enabledCount = useMemo(
    () => models.filter((model) => model.enabled).length,
    [models],
  );

  const [editorOpen, setEditorOpen] = useState(false);
  const [editingEntry, setEditingEntry] = useState<ModelCatalogEntry | null>(
    null,
  );
  const [form, setForm] = useState<ModelFormState>(() => createEmptyModelForm());
  const [deleteTarget, setDeleteTarget] = useState<ModelCatalogEntry | null>(
    null,
  );
  const [searchTerm, setSearchTerm] = useState("");
  const [providerFilter, setProviderFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState<"all" | "enabled" | "disabled">(
    "all",
  );
  const [selectedAliases, setSelectedAliases] = useState<Set<string>>(new Set());
  const [bulkDeleteOpen, setBulkDeleteOpen] = useState(false);
  const [bulkLoading, setBulkLoading] = useState(false);

  const upsertMutation = useMutation({
    mutationFn: upsertModel,
    onSuccess: () => {
      toast({ title: "Model saved" });
      queryClient.invalidateQueries({ queryKey: CATALOG_QUERY_KEY });
      closeEditor();
    },
    onError: (error: Error) => {
      toast({
        variant: "destructive",
        title: "Failed to save model",
        description: error.message,
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteModel,
    onSuccess: () => {
      toast({ title: "Model removed" });
      queryClient.invalidateQueries({ queryKey: CATALOG_QUERY_KEY });
      setDeleteTarget(null);
    },
    onError: (error: Error) => {
      toast({
        variant: "destructive",
        title: "Failed to delete model",
        description: error.message,
      });
    },
  });

  // Helper to convert ModelCatalogEntry to UpsertRequest
  const entryToRequest = (entry: ModelCatalogEntry, overrides: Partial<ModelCatalogUpsertRequest> = {}): ModelCatalogUpsertRequest => ({
    alias: entry.alias,
    provider: entry.provider,
    provider_model: entry.provider_model,
    model_type: entry.model_type,
    context_window: entry.context_window,
    max_output_tokens: entry.max_output_tokens,
    modalities: entry.modalities,
    supports_tools: entry.supports_tools,
    price_input: entry.price_input,
    price_output: entry.price_output,
    currency: entry.currency,
    enabled: entry.enabled,
    tenant_assignable: entry.tenant_assignable,
    deployment: entry.deployment,
    endpoint: entry.endpoint,
    api_key: entry.apiKey,
    api_version: entry.api_version,
    region: entry.region,
    metadata: entry.metadata,
    weight: entry.weight,
    ...overrides,
  });

  // Bulk operations
  const handleBulkEnable = async () => {
    if (selectedAliases.size === 0) return;
    setBulkLoading(true);
    try {
      const modelsToUpdate = models.filter((m) => selectedAliases.has(m.alias) && !m.enabled);
      for (const model of modelsToUpdate) {
        await upsertModel(entryToRequest(model, { enabled: true }));
      }
      toast({ title: `${modelsToUpdate.length} model(s) enabled` });
      queryClient.invalidateQueries({ queryKey: CATALOG_QUERY_KEY });
      setSelectedAliases(new Set());
    } catch (error) {
      toast({
        variant: "destructive",
        title: "Failed to enable models",
        description: error instanceof Error ? error.message : "Unknown error",
      });
    } finally {
      setBulkLoading(false);
    }
  };

  const handleBulkDisable = async () => {
    if (selectedAliases.size === 0) return;
    setBulkLoading(true);
    try {
      const modelsToUpdate = models.filter((m) => selectedAliases.has(m.alias) && m.enabled);
      for (const model of modelsToUpdate) {
        await upsertModel(entryToRequest(model, { enabled: false }));
      }
      toast({ title: `${modelsToUpdate.length} model(s) disabled` });
      queryClient.invalidateQueries({ queryKey: CATALOG_QUERY_KEY });
      setSelectedAliases(new Set());
    } catch (error) {
      toast({
        variant: "destructive",
        title: "Failed to disable models",
        description: error instanceof Error ? error.message : "Unknown error",
      });
    } finally {
      setBulkLoading(false);
    }
  };

  const handleBulkDelete = async () => {
    if (selectedAliases.size === 0) return;
    setBulkLoading(true);
    try {
      const aliasesToDelete = Array.from(selectedAliases);
      for (const alias of aliasesToDelete) {
        await deleteModel(alias);
      }
      toast({ title: `${aliasesToDelete.length} model(s) deleted` });
      queryClient.invalidateQueries({ queryKey: CATALOG_QUERY_KEY });
      setSelectedAliases(new Set());
      setBulkDeleteOpen(false);
    } catch (error) {
      toast({
        variant: "destructive",
        title: "Failed to delete models",
        description: error instanceof Error ? error.message : "Unknown error",
      });
    } finally {
      setBulkLoading(false);
    }
  };

  const openCreate = () => {
    setEditingEntry(null);
    setForm(createEmptyModelForm());
    setEditorOpen(true);
  };

  const openEdit = (entry: ModelCatalogEntry) => {
    setEditingEntry(entry);
    setForm(mapEntryToForm(entry));
    setEditorOpen(true);
  };

  const openClone = (entry: ModelCatalogEntry) => {
    setEditingEntry(null); // Clone creates a new entry
    const clonedForm = mapEntryToForm(entry);
    setForm({
      ...clonedForm,
      alias: "", // Must be unique, so clear it
    });
    setEditorOpen(true);
  };

  const closeEditor = () => {
    setEditorOpen(false);
    setEditingEntry(null);
    setForm(createEmptyModelForm());
  };

  const handleSubmit = (payload: ModelCatalogUpsertRequest) => {
    upsertMutation.mutate(payload);
  };


  const filteredModels = useMemo(() => {
    const term = searchTerm.trim().toLowerCase();
    return models.filter((model) => {
      const matchesTerm =
        !term ||
        model.alias.toLowerCase().includes(term) ||
        model.provider.toLowerCase().includes(term) ||
        model.provider_model.toLowerCase().includes(term) ||
        (model.deployment ?? "").toLowerCase().includes(term);
      const matchesProvider =
        providerFilter === "all" || model.provider === providerFilter;
      const matchesStatus =
        statusFilter === "all" ||
        (statusFilter === "enabled" ? model.enabled : !model.enabled);
      return matchesTerm && matchesProvider && matchesStatus;
    });
  }, [models, searchTerm, providerFilter, statusFilter]);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Model Catalog"
        description="Configure provider aliases, routing weights, pricing data, and deployment metadata."
        actions={
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="icon"
              onClick={() => catalogQuery.refetch()}
              loading={catalogQuery.isFetching}
            >
              <RefreshCcw className="h-4 w-4" />
            </Button>
            <Button onClick={openCreate}>
              <Plus className="h-4 w-4" />
              Add Model
            </Button>
          </div>
        }
      />

      <Card>
        <CardHeader className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <ModelFilters
            enabledCount={enabledCount}
            totalCount={models.length}
            searchTerm={searchTerm}
            onSearchTermChange={setSearchTerm}
            providerOptions={providerOptions}
            providerFilter={providerFilter}
            onProviderFilterChange={setProviderFilter}
            statusFilter={statusFilter}
            onStatusFilterChange={setStatusFilter}
          />
        </CardHeader>
        <CardContent className="space-y-4">
          <BulkActionBar
            selectedCount={selectedAliases.size}
            onEnable={handleBulkEnable}
            onDisable={handleBulkDisable}
            onDelete={() => setBulkDeleteOpen(true)}
            onClear={() => setSelectedAliases(new Set())}
            isLoading={bulkLoading}
          />
          <ModelTable
            models={filteredModels}
            isLoading={catalogQuery.isLoading}
            hasAnyModels={models.length > 0}
            statuses={statusMap}
            onEdit={openEdit}
            onClone={openClone}
            onDelete={setDeleteTarget}
            theme={resolvedTheme}
            selectedAliases={selectedAliases}
            onSelectionChange={setSelectedAliases}
          />
        </CardContent>
      </Card>

      <ModelEditorDialog
        open={editorOpen}
        onOpenChange={(open) => {
          if (!open) {
            closeEditor();
          } else {
            setEditorOpen(true);
          }
        }}
        form={form}
        onChange={setForm}
        onSubmit={handleSubmit}
        loading={upsertMutation.isPending}
        mode={editingEntry ? "edit" : "create"}
      />

      <AlertDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete model?</AlertDialogTitle>
            <AlertDialogDescription>
              This will remove the <strong>{deleteTarget?.alias}</strong> alias
              from the router. Clients will no longer be able to request it.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteMutation.isPending}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={deleteMutation.isPending}
              onClick={() =>
                deleteTarget && deleteMutation.mutate(deleteTarget.alias)
              }
            >
              {deleteMutation.isPending ? "Removing..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={bulkDeleteOpen}
        onOpenChange={(open) => !open && setBulkDeleteOpen(false)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {selectedAliases.size} models?</AlertDialogTitle>
            <AlertDialogDescription>
              This will remove {selectedAliases.size} model{selectedAliases.size === 1 ? "" : "s"} from the router.
              Clients will no longer be able to request them.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={bulkLoading}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={bulkLoading}
              onClick={handleBulkDelete}
            >
              {bulkLoading ? "Deleting..." : `Delete ${selectedAliases.size} models`}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
