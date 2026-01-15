import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

import type { ModelFormState, PricingTiersFormState } from "../../types";
import { PricingTiersEditor } from "../PricingTiersEditor";

interface PricingTabProps {
  form: ModelFormState;
  onChange: (form: ModelFormState) => void;
}

export function PricingTab({ form, onChange }: PricingTabProps) {
  const handlePricingTiersChange = (tiers: PricingTiersFormState) => {
    onChange({
      ...form,
      pricing_tiers: tiers,
    });
  };

  return (
    <div className="grid gap-4">
      <div className="space-y-2">
        <p className="text-sm font-medium">Pricing configuration</p>
        <p className="text-xs text-muted-foreground">
          Configure pricing tiers for different usage scenarios (standard, batch, cached).
          Leave empty to use provider defaults.
        </p>
      </div>

      <PricingTiersEditor
        value={form.pricing_tiers}
        onChange={handlePricingTiersChange}
      />

      <div className="rounded-md border p-4">
        <div className="space-y-1 mb-4">
          <p className="text-sm font-medium">Legacy pricing (deprecated)</p>
          <p className="text-xs text-muted-foreground">
            These fields are maintained for backwards compatibility. Use the pricing tiers above instead.
          </p>
        </div>
        <div className="grid gap-4 sm:grid-cols-3">
          <div className="space-y-2">
            <Label htmlFor="price_input">Price per 1M input tokens</Label>
            <Input
              id="price_input"
              value={form.price_input}
              onChange={(event) =>
                onChange({ ...form, price_input: event.target.value })
              }
              placeholder="0.00"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="price_output">Price per 1M output tokens</Label>
            <Input
              id="price_output"
              value={form.price_output}
              onChange={(event) =>
                onChange({ ...form, price_output: event.target.value })
              }
              placeholder="0.00"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="currency">Currency</Label>
            <Input
              id="currency"
              value={form.currency}
              onChange={(event) =>
                onChange({ ...form, currency: event.target.value })
              }
              placeholder="USD"
            />
          </div>
        </div>
      </div>
    </div>
  );
}
