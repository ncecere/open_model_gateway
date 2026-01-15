import { useMemo, useState } from "react";
import { Layers, Search, X } from "lucide-react";

import type { ModelCatalogEntry } from "@/api/model-catalog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { TabsContent } from "@/components/ui/tabs";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { EmptyState } from "@/components/tables";

type DefaultModelsTabProps = {
  loading: boolean;
  defaultModels: string[];
  availableModels: ModelCatalogEntry[];
  catalogByAlias: Map<string, ModelCatalogEntry>;
  newDefaultModel: string;
  setNewDefaultModel: (value: string) => void;
  onAddModel: () => void;
  onRemoveModel: (alias: string) => void;
  addPending: boolean;
  removePending: boolean;
};

export function DefaultModelsTab({
  loading,
  defaultModels,
  availableModels,
  catalogByAlias,
  newDefaultModel,
  setNewDefaultModel,
  onAddModel,
  onRemoveModel,
  addPending,
  removePending,
}: DefaultModelsTabProps) {
  const [searchTerm, setSearchTerm] = useState("");

  const filteredModels = useMemo(() => {
    if (!searchTerm.trim()) {
      return defaultModels;
    }
    const query = searchTerm.toLowerCase();
    return defaultModels.filter((alias) => {
      const meta = catalogByAlias.get(alias);
      return (
        alias.toLowerCase().includes(query) ||
        meta?.provider?.toLowerCase().includes(query) ||
        meta?.provider_model?.toLowerCase().includes(query)
      );
    });
  }, [defaultModels, searchTerm, catalogByAlias]);

  return (
    <TabsContent value="models" className="space-y-6" forceMount>
      {/* Summary Stats */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Default Models</CardTitle>
            <Layers className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{defaultModels.length}</div>
            <p className="text-xs text-muted-foreground">
              Granted to all personal tenants
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Directory Card */}
      <Card>
        <CardHeader className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div className="space-y-1">
            <CardTitle>Default models</CardTitle>
            <p className="text-sm text-muted-foreground">
              These aliases are granted automatically to every personal tenant.
            </p>
          </div>
          <div className="flex w-full flex-col gap-2 md:w-auto md:flex-row md:items-center md:gap-3">
            <div className="w-full md:w-80">
              <Select
                value={newDefaultModel}
                onValueChange={setNewDefaultModel}
                disabled={availableModels.length === 0 || addPending}
              >
                <SelectTrigger>
                  <SelectValue
                    placeholder={
                      availableModels.length
                        ? "Select model to add"
                        : "All enabled models already granted"
                    }
                  />
                </SelectTrigger>
                <SelectContent className="max-h-72 overflow-y-auto">
                  {availableModels.map((entry) => (
                    <SelectItem key={entry.alias} value={entry.alias}>
                      <div className="flex flex-col text-left">
                        <span className="font-medium">{entry.alias}</span>
                        <span className="text-xs text-muted-foreground">
                          {entry.provider}
                        </span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <Button onClick={onAddModel} disabled={!newDefaultModel || addPending}>
              {addPending ? "Adding…" : "Add model"}
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Search */}
          {defaultModels.length > 0 && (
            <div className="relative">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Search by alias, provider, or model..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="pl-9"
              />
            </div>
          )}

          {loading ? (
            <div className="space-y-3">
              <Skeleton className="h-6 w-1/3" />
              <Skeleton className="h-6 w-1/2" />
              <Skeleton className="h-6 w-2/3" />
            </div>
          ) : defaultModels.length === 0 ? (
            <EmptyState
              message="No default models configured"
              description="Add models above to automatically grant them to all personal tenants."
            />
          ) : filteredModels.length === 0 ? (
            <EmptyState
              message="No models match your search"
              description="Try adjusting your search term."
            />
          ) : (
            <div className="rounded-lg border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Alias</TableHead>
                    <TableHead>Provider</TableHead>
                    <TableHead>Provider model</TableHead>
                    <TableHead className="w-20 text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredModels.map((alias) => {
                    const meta = catalogByAlias.get(alias);
                    return (
                      <TableRow key={alias}>
                        <TableCell className="font-medium">
                          {alias}
                          {!meta ? (
                            <p className="text-xs text-muted-foreground">
                              Missing catalog entry
                            </p>
                          ) : null}
                        </TableCell>
                        <TableCell>
                          {meta ? (
                            <Badge variant="secondary">{meta.provider}</Badge>
                          ) : (
                            <span className="text-sm text-muted-foreground">—</span>
                          )}
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {meta?.provider_model ?? "—"}
                        </TableCell>
                        <TableCell className="text-right">
                          <Button
                            variant="ghost"
                            size="icon"
                            aria-label={`Remove ${alias}`}
                            onClick={() => onRemoveModel(alias)}
                            disabled={removePending}
                          >
                            <X className="h-4 w-4" />
                          </Button>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </TabsContent>
  );
}
