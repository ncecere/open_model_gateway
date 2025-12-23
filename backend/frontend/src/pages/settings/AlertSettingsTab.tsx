import { Controller, useFormContext } from "react-hook-form";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { TabsContent } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

import type { SettingsFormValues } from "./useSettingsForm";

export type TestEmailOption = { value: string; label: string };

type AlertSettingsTabProps = {
  loading: boolean;
  onReset: () => void;
  onSave: () => void;
  saving: boolean;
  testEmail: string;
  setTestEmail: (value: string) => void;
  testEmailType: string;
  setTestEmailType: (value: string) => void;
  testEmailOptions: TestEmailOption[];
  sendingTest: boolean;
  onSendTest: () => void;
};

export function AlertSettingsTab({
  loading,
  onReset,
  onSave,
  saving,
  testEmail,
  setTestEmail,
  testEmailType,
  setTestEmailType,
  testEmailOptions,
  sendingTest,
  onSendTest,
}: AlertSettingsTabProps) {
  const { register, control } = useFormContext<SettingsFormValues>();

  return (
    <TabsContent value="alerts" forceMount>
      <Card>
        <CardHeader>
          <CardTitle>Alert transports</CardTitle>
          <p className="text-sm text-muted-foreground">
            Manage SMTP and webhook delivery for budget alerts. Leave fields blank to disable a channel.
          </p>
        </CardHeader>
        <CardContent className="space-y-6">
          {loading ? (
            <Skeleton className="h-40 w-full" />
          ) : (
            <>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="smtp-host">SMTP host</Label>
                  <Input id="smtp-host" {...register("smtpHost")} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="smtp-port">SMTP port</Label>
                  <Input id="smtp-port" type="number" {...register("smtpPort")} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="smtp-username">SMTP username</Label>
                  <Input id="smtp-username" {...register("smtpUsername")} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="smtp-password">SMTP password / token</Label>
                  <Input id="smtp-password" type="password" {...register("smtpPassword")} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="smtp-from">From address</Label>
                  <Input id="smtp-from" {...register("smtpFrom")} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="smtp-timeout">Connect timeout (seconds)</Label>
                  <Input id="smtp-timeout" type="number" {...register("smtpTimeout")} />
                </div>
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="flex items-center justify-between rounded-lg border p-3">
                  <div>
                    <p className="text-sm font-medium">Use TLS / STARTTLS</p>
                    <p className="text-xs text-muted-foreground">
                      Attempts STARTTLS if supported by the server.
                    </p>
                  </div>
                  <Controller
                    name="smtpUseTLS"
                    control={control}
                    render={({ field }) => (
                      <Switch checked={field.value} onCheckedChange={field.onChange} />
                    )}
                  />
                </div>
                <div className="flex items-center justify-between rounded-lg border p-3">
                  <div>
                    <p className="text-sm font-medium">Skip TLS verification</p>
                    <p className="text-xs text-muted-foreground">
                      Only enable for local/self-signed servers.
                    </p>
                  </div>
                  <Controller
                    name="smtpSkipVerify"
                    control={control}
                    render={({ field }) => (
                      <Switch checked={field.value} onCheckedChange={field.onChange} />
                    )}
                  />
                </div>
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="webhook-timeout">Webhook timeout (seconds)</Label>
                  <Input id="webhook-timeout" type="number" {...register("webhookTimeout")} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="webhook-retries">Webhook max retries</Label>
                  <Input id="webhook-retries" type="number" {...register("webhookRetries")} />
                </div>
              </div>
              <div className="flex items-center justify-end gap-2">
                <Button variant="outline" onClick={onReset} disabled={loading}>
                  Reset
                </Button>
                <Button onClick={onSave} disabled={saving}>
                  {saving ? "Saving…" : "Save changes"}
                </Button>
              </div>
            </>
          )}
          <div className="space-y-2 border-t pt-4">
            <Label htmlFor="test-email">Send test email</Label>
            <div className="flex flex-col gap-2 md:flex-row md:items-end">
              <div className="space-y-2 md:flex-1">
                <Label className="sr-only" htmlFor="test-email">
                  Recipient email
                </Label>
                <Input
                  id="test-email"
                  placeholder="alerts@example.com"
                  value={testEmail}
                  onChange={(event) => setTestEmail(event.target.value)}
                />
              </div>
              <div className="space-y-2 md:w-56">
                <Label htmlFor="test-email-type">Template</Label>
                <Select value={testEmailType} onValueChange={setTestEmailType}>
                  <SelectTrigger id="test-email-type">
                    <SelectValue placeholder="Budget alert" />
                  </SelectTrigger>
                  <SelectContent>
                    {testEmailOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <Button
                type="button"
                variant="outline"
                disabled={!testEmail || sendingTest}
                onClick={onSendTest}
              >
                {sendingTest ? "Sending…" : "Send test"}
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </TabsContent>
  );
}
