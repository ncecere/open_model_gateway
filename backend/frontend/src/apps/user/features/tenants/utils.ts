import type { MemberBudget } from "@/api/user/tenants";

/**
 * Renders a member budget as a human-readable string.
 */
export function renderMemberBudget(budget?: MemberBudget): string {
  if (!budget || (!budget.budget_usd && !budget.warning_threshold && !budget.token_cap)) {
    return "Default";
  }
  const pieces: string[] = [];
  if (budget.budget_usd && budget.budget_usd > 0) {
    pieces.push(`$${budget.budget_usd.toFixed(2)}`);
  }
  if (budget.token_cap && budget.token_cap > 0) {
    pieces.push(`${budget.token_cap.toLocaleString()} tokens`);
  }
  if (budget.warning_threshold && budget.warning_threshold > 0) {
    pieces.push(`${Math.round(budget.warning_threshold * 100)}% warn`);
  }
  return pieces.join(" · ");
}

/**
 * Role options for tenant membership management.
 */
export const MANAGEABLE_ROLES = ["owner", "admin", "viewer", "user"] as const;

export type ManageableRole = (typeof MANAGEABLE_ROLES)[number];
