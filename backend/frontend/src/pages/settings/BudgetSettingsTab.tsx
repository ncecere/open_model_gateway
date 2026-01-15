import { Controller, useFormContext } from "react-hook-form";

import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { TabsContent } from "@/components/ui/tabs";

import type { SettingsFormValues } from "./useSettingsForm";

export type RefreshOption = { value: string; label: string };

type BudgetSettingsTabProps = {
  loading: boolean;
  hasDefaults: boolean;
  metadataDescription: string | null;
  refreshOptions: RefreshOption[];
  onReset: () => void;
  onSave: () => void;
  saving: boolean;
};

export function BudgetSettingsTab({
  loading,
  hasDefaults,
  metadataDescription,
  refreshOptions,
  onReset,
  onSave,
  saving,
}: BudgetSettingsTabProps) {
  const { register, control } = useFormContext<SettingsFormValues>();

  return (
    <TabsContent value="budgets" forceMount>
      <Card>
        <CardHeader>
          <div className="space-y-1">
            <CardTitle>Budget defaults</CardTitle>
            <p className="text-sm text-muted-foreground">
              {metadataDescription
                ? `${metadataDescription} Detailed history view coming soon.`
                : "Detailed history view coming soon. Save changes to capture metadata."}
            </p>
          </div>
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
                  <Label htmlFor="default-budget">Default monthly budget (USD)</Label>
                  <Input
                    id="default-budget"
                    type="number"
                    min="0"
                    step="0.01"
                    {...register("budgetDefaultUsd")}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="warning-threshold">Warning threshold (0-1)</Label>
                  <Input
                    id="warning-threshold"
                    type="number"
                    min="0"
                    max="1"
                    step="0.01"
                    {...register("budgetWarningThreshold")}
                  />
                </div>
                <div className="space-y-2">
                  <Label>Refresh schedule</Label>
                  <Controller
                    name="budgetRefreshSchedule"
                    control={control}
                    render={({ field }) => (
                      <Select value={field.value} onValueChange={field.onChange}>
                        <SelectTrigger>
                          <SelectValue placeholder="Select schedule" />
                        </SelectTrigger>
                        <SelectContent>
                          {refreshOptions.map((option) => (
                            <SelectItem key={option.value} value={option.value}>
                              {option.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    )}
                  />
                </div>
              </div>
              <div className="grid gap-4 md:grid-cols-3">
                <div className="space-y-2">
                  <Label htmlFor="alert-cooldown">Alert cooldown (seconds)</Label>
                  <Input
                    id="alert-cooldown"
                    type="number"
                    min="60"
                    {...register("budgetAlertCooldown")}
                  />
                </div>
                <div className="md:col-span-3 space-y-2">
                  <Label>Alert emails (comma or line separated)</Label>
                  <Textarea
                    placeholder="alerts@example.com, ops@example.com"
                    rows={2}
                    {...register("budgetAlertEmails")}
                  />
                </div>
                <div className="md:col-span-3 space-y-2">
                  <Label>Alert webhooks (comma or line separated)</Label>
                  <Textarea
                    placeholder="https://hooks.slack.com/..."
                    rows={2}
                    {...register("budgetAlertWebhooks")}
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
