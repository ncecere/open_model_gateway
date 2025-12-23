import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { TabsContent } from "@/components/ui/tabs";

export function OverviewSettingsTab() {
  return (
    <TabsContent value="overview" forceMount>
      <Card>
        <CardHeader>
          <CardTitle>Budget workflow</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-muted-foreground">
          <p>
            Tenant budgets, warning thresholds, alert channels, and model access
            policies are configured per-tenant from the Tenants page. Those
            values are persisted in Postgres and enforced by the router on every
            request.
          </p>
          <p>
            Defaults shown above come from the active configuration and provide
            the fallback when a tenant does not have its own override.
          </p>
          <p>
            Looking for observability hooks or advanced provider knobs? Those
            settings still live in configuration for now and will gain UI
            controls in a future iteration.
          </p>
        </CardContent>
      </Card>
    </TabsContent>
  );
}
