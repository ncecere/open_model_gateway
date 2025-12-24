import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import {
  TenantModelRow,
  renderMemberBudget,
} from "../TenantsPage";
import type { TenantModelEntry } from "@/api/user/tenants";

describe("tenant model access UI", () => {
  const baseModel: TenantModelEntry = {
    alias: "gpt-test",
    provider: "openai",
    model_type: "llm",
    enabled: true,
    tenant_assignable: true,
    attached: false,
  };

  it("disables the toggle when the user cannot manage", () => {
    render(
      <table>
        <tbody>
          <TenantModelRow
            model={baseModel}
            canManage={false}
            onToggle={vi.fn()}
            isUpdating={false}
          />
        </tbody>
      </table>,
    );

    const toggle = screen.getByRole("switch");
    expect(toggle).toBeDisabled();
  });

  it("disables the toggle when the model is not assignable", () => {
    render(
      <table>
        <tbody>
          <TenantModelRow
            model={{ ...baseModel, tenant_assignable: false, attached: false }}
            canManage
            onToggle={vi.fn()}
            isUpdating={false}
          />
        </tbody>
      </table>,
    );

    const toggle = screen.getByRole("switch");
    expect(toggle).toBeDisabled();
    expect(screen.getByText("Not assignable")).toBeInTheDocument();
  });
});

describe("member budget formatting", () => {
  it("returns Default when no budget is set", () => {
    expect(renderMemberBudget(undefined)).toBe("Default");
  });

  it("renders budget values when present", () => {
    const label = renderMemberBudget({
      budget_usd: 25.5,
      warning_threshold: 0.75,
      token_cap: 1000,
    });
    expect(label).toBe("$25.50 · 1,000 tokens · 75% warn");
  });
});
