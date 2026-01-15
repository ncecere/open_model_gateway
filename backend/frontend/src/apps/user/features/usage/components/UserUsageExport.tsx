import { useState } from "react";
import { Download, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useToast } from "@/hooks/use-toast";

export interface UserUsageExportProps {
  onExport: (params: ExportParams) => Promise<void>;
  defaultStartDate?: string;
  defaultEndDate?: string;
  disabled?: boolean;
}

export interface ExportParams {
  startDate: string;
  endDate: string;
  format: "csv" | "json";
  granularity: "daily" | "hourly";
}

function formatDateInput(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function getDefaultDates(): { start: string; end: string } {
  const now = new Date();
  const end = formatDateInput(now);
  const start = formatDateInput(
    new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000)
  );
  return { start, end };
}

export function UserUsageExport({
  onExport,
  defaultStartDate,
  defaultEndDate,
  disabled,
}: UserUsageExportProps) {
  const { toast } = useToast();
  const [open, setOpen] = useState(false);
  const [exporting, setExporting] = useState(false);
  const defaults = getDefaultDates();

  const [startDate, setStartDate] = useState(defaultStartDate ?? defaults.start);
  const [endDate, setEndDate] = useState(defaultEndDate ?? defaults.end);
  const [format, setFormat] = useState<"csv" | "json">("csv");
  const [granularity, setGranularity] = useState<"daily" | "hourly">("daily");

  const handleExport = async () => {
    if (!startDate || !endDate) {
      toast({
        variant: "destructive",
        title: "Invalid dates",
        description: "Please select both start and end dates.",
      });
      return;
    }

    const startTime = new Date(startDate).getTime();
    const endTime = new Date(endDate).getTime();
    if (endTime < startTime) {
      toast({
        variant: "destructive",
        title: "Invalid date range",
        description: "End date must be after start date.",
      });
      return;
    }

    setExporting(true);
    try {
      await onExport({ startDate, endDate, format, granularity });
      toast({
        title: "Export started",
        description: "Your export is being prepared. It will download when ready.",
      });
      setOpen(false);
    } catch (error) {
      toast({
        variant: "destructive",
        title: "Export failed",
        description:
          error instanceof Error ? error.message : "Unable to export usage data.",
      });
    } finally {
      setExporting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm" disabled={disabled}>
          <Download className="mr-2 h-4 w-4" />
          Export
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Export Usage Data</DialogTitle>
          <DialogDescription>
            Download your usage data for the selected date range.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          <div className="grid gap-2">
            <Label htmlFor="export-start">Start date</Label>
            <Input
              id="export-start"
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="export-end">End date</Label>
            <Input
              id="export-end"
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="export-format">Format</Label>
            <Select value={format} onValueChange={(v) => setFormat(v as "csv" | "json")}>
              <SelectTrigger id="export-format">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="csv">CSV</SelectItem>
                <SelectItem value="json">JSON</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="export-granularity">Granularity</Label>
            <Select
              value={granularity}
              onValueChange={(v) => setGranularity(v as "daily" | "hourly")}
            >
              <SelectTrigger id="export-granularity">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="daily">Daily</SelectItem>
                <SelectItem value="hourly">Hourly</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button onClick={handleExport} disabled={exporting}>
            {exporting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Exporting...
              </>
            ) : (
              <>
                <Download className="mr-2 h-4 w-4" />
                Export
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
