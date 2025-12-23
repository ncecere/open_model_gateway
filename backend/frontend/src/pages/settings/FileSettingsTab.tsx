import { useFormContext } from "react-hook-form";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { TabsContent } from "@/components/ui/tabs";

import type { SettingsFormValues } from "./useSettingsForm";

type FileSettingsTabProps = {
  loading: boolean;
  hasDefaults: boolean;
  onReset: () => void;
  onSave: () => void;
  saving: boolean;
};

export function FileSettingsTab({
  loading,
  hasDefaults,
  onReset,
  onSave,
  saving,
}: FileSettingsTabProps) {
  const { register } = useFormContext<SettingsFormValues>();

  return (
    <TabsContent value="files" forceMount>
      <Card>
        <CardHeader>
          <CardTitle>File retention defaults</CardTitle>
          <p className="text-sm text-muted-foreground">
            Control max file sizes and retention windows used by uploads and batch outputs.
          </p>
        </CardHeader>
        <CardContent className="space-y-4">
          {loading ? (
            <div className="space-y-3">
              <Skeleton className="h-6 w-1/3" />
              <Skeleton className="h-6 w-1/2" />
              <Skeleton className="h-6 w-2/3" />
            </div>
          ) : (
            <>
              <div className="grid gap-4 md:grid-cols-3">
                <div className="space-y-2">
                  <Label htmlFor="file-max-size">Max file size (MB)</Label>
                  <Input
                    id="file-max-size"
                    type="number"
                    min="1"
                    {...register("fileMaxSizeMb")}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="file-default-ttl">Default TTL (seconds)</Label>
                  <Input
                    id="file-default-ttl"
                    type="number"
                    min="60"
                    {...register("fileDefaultTtlSeconds")}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="file-max-ttl">Max TTL (seconds)</Label>
                  <Input
                    id="file-max-ttl"
                    type="number"
                    min="60"
                    {...register("fileMaxTtlSeconds")}
                  />
                </div>
              </div>
              <div className="flex items-center justify-end gap-2">
                <Button variant="outline" onClick={onReset} disabled={!hasDefaults}>
                  Reset
                </Button>
                <Button onClick={onSave} disabled={saving || loading}>
                  {saving ? "Saving…" : "Save changes"}
                </Button>
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </TabsContent>
  );
}
