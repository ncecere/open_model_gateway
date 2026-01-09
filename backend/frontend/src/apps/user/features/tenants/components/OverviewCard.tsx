import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export type OverviewCardProps = {
  label: string;
  value: number | string;
  help?: string;
};

export function OverviewCard({ label, value, help }: OverviewCardProps) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {label}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-bold">{value}</p>
        {help && <p className="text-xs text-muted-foreground">{help}</p>}
      </CardContent>
    </Card>
  );
}
