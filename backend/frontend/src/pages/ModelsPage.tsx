import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { RefreshCcw } from "lucide-react";

import {
  deleteModel,
  listModelCatalog,
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
import { Separator } from "@/components/ui/separator";
import { useToast } from "@/hooks/use-toast";
import {
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

  const catalogQuery = useQuery({
    queryKey: CATALOG_QUERY_KEY,
    queryFn: listModelCatalog,
    staleTime: 60_000,
  });

  const models = catalogQuery.data ?? [];
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
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">
            Model catalog
          </h1>
          <p className="text-sm text-muted-foreground">
            Configure provider aliases, routing weights, pricing data, and
            deployment metadata.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="icon"
            onClick={() => catalogQuery.refetch()}
            disabled={catalogQuery.isFetching}
          >
            <RefreshCcw className="h-4 w-4" />
          </Button>
          <Button onClick={openCreate}>Add model</Button>
        </div>
      </div>
      <Separator />

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
        <CardContent>
          <ModelTable
            models={filteredModels}
            isLoading={catalogQuery.isLoading}
            hasAnyModels={models.length > 0}
            onEdit={openEdit}
            onDelete={setDeleteTarget}
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
    </div>
  );
}
