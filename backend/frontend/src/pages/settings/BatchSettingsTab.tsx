import { useFormContext } from "react-hook-form";

import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { TabsContent } from "@/components/ui/tabs";

import type { SettingsFormValues } from "./useSettingsForm";

type BatchSettingsTabProps = {
  loading: boolean;
  hasDefaults: boolean;
  onReset: () => void;
  onSave: () => void;
  saving: boolean;
};

export function BatchSettingsTab({
  loading,
  hasDefaults,
  onReset,
  onSave,
  saving,
}: BatchSettingsTabProps) {
  const { register } = useFormContext<SettingsFormValues>();

  return (
    <TabsContent value="batches" forceMount>
      <Card>
        <CardHeader>
          <CardTitle>Batch processing</CardTitle>
          <p className="text-sm text-muted-foreground">
            Tune batch sizes and TTL windows for background jobs.
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
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="batch-max-requests">Max requests</Label>
                  <Input
                    id="batch-max-requests"
                    type="number"
                    min="1"
                    {...register("batchMaxRequests")}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="batch-max-concurrency">Max concurrency</Label>
                  <Input
                    id="batch-max-concurrency"
                    type="number"
                    min="1"
                    {...register("batchMaxConcurrency")}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="batch-default-ttl">Default TTL (seconds)</Label>
                  <Input
                    id="batch-default-ttl"
                    type="number"
                    min="60"
                    {...register("batchDefaultTtlSeconds")}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="batch-max-ttl">Max TTL (seconds)</Label>
                  <Input
                    id="batch-max-ttl"
                    type="number"
                    min="60"
                    {...register("batchMaxTtlSeconds")}
                  />
                </div>
              </div>
            </>
          )}
        </CardContent>
        <CardFooter className="flex justify-end gap-2">
          <Button variant="outline" onClick={onReset} disabled={!hasDefaults || loading}>
            Reset
          </Button>
          <Button onClick={onSave} disabled={saving || loading}>
            {saving ? "Saving…" : "Save changes"}
          </Button>
        </CardFooter>
      </Card>
    </TabsContent>
  );
}
