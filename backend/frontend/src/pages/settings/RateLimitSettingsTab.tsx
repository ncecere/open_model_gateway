import { useFormContext } from "react-hook-form";

import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { TabsContent } from "@/components/ui/tabs";

import type { SettingsFormValues } from "./useSettingsForm";

type RateLimitSettingsTabProps = {
  loading: boolean;
  hasDefaults: boolean;
  onReset: () => void;
  onSave: () => void;
  saving: boolean;
};

export function RateLimitSettingsTab({
  loading,
  hasDefaults,
  onReset,
  onSave,
  saving,
}: RateLimitSettingsTabProps) {
  const { register } = useFormContext<SettingsFormValues>();

  return (
    <TabsContent value="rate-limits" forceMount>
      <Card>
        <CardHeader>
          <CardTitle>Rate limit defaults</CardTitle>
          <p className="text-sm text-muted-foreground">
            Define the baseline RPM/TPM/parallel ceilings applied to every API key
            and tenant unless an override is configured.
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
                  <Label htmlFor="requests-per-minute">Requests per minute</Label>
                  <Input
                    id="requests-per-minute"
                    type="number"
                    min="1"
                    {...register("rateRequestsPerMinute")}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="tokens-per-minute">Tokens per minute</Label>
                  <Input
                    id="tokens-per-minute"
                    type="number"
                    min="1"
                    {...register("rateTokensPerMinute")}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="parallel-key">Parallel requests (per key)</Label>
                  <Input
                    id="parallel-key"
                    type="number"
                    min="1"
                    {...register("rateParallelKey")}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="parallel-tenant">Parallel requests (per tenant)</Label>
                  <Input
                    id="parallel-tenant"
                    type="number"
                    min="1"
                    {...register("rateParallelTenant")}
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
