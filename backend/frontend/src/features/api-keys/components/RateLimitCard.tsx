import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { RateLimitInfo } from "@/api/tenants";
import { formatRateValue } from "../utils";

type RateLimitCardProps = {
  title: string;
  details?: RateLimitInfo;
};

export function RateLimitCard({ title, details }: RateLimitCardProps) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-2 text-sm">
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground">Requests/min</span>
          <span className="font-medium text-foreground">
            {formatRateValue(details?.requests_per_minute)}
          </span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground">Tokens/min</span>
          <span className="font-medium text-foreground">
            {formatRateValue(details?.tokens_per_minute)}
          </span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground">Parallel in-flight</span>
          <span className="font-medium text-foreground">
            {formatRateValue(details?.parallel_requests)}
          </span>
        </div>
      </CardContent>
    </Card>
  );
}
