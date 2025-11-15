import { Badge } from "@/components/ui/badge";

type FileStatusBadgeProps = {
  status?: string;
  className?: string;
};

const STATUS_VARIANTS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  deleted: "destructive",
  error: "destructive",
  processed: "default",
  uploading: "outline",
  uploaded: "secondary",
};

export function FileStatusBadge({ status, className }: FileStatusBadgeProps) {
  const normalized = (status || "unknown").toLowerCase();
  const variant = STATUS_VARIANTS[normalized] ?? "secondary";
  return (
    <Badge variant={variant} className={className ? `${className} capitalize` : "capitalize"}>
      {formatFileStatus(normalized)}
    </Badge>
  );
}

export function formatFileStatus(status?: string) {
  const label = status?.trim() || "unknown";
  return label.replace(/-/g, " ");
}
