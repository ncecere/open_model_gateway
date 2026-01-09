/**
 * User Tenants API
 *
 * This module provides user-level access to tenant management.
 * Uses the factory pattern from api/shared/factory.ts.
 */

import { userApi } from "../userClient";
import { createResource, createNestedResource } from "../shared/factory";
import type {
  UserTenantRecord,
  UserMembershipRecord,
  MembershipRole,
  MemberBudget,
  BudgetSummary,
  TenantLimits,
  TenantLimitsPayload,
  TenantModelEntry,
} from "../shared/types";

// ============================================================================
// Types
// ============================================================================

/** User tenant type (alias for UserTenantRecord) */
export type UserTenant = UserTenantRecord;

/** Response for listing user tenants */
export type ListUserTenantsResponse = {
  tenants: UserTenant[];
};

/** Tenant budget summary */
export type TenantBudgetSummary = BudgetSummary;

/** Tenant membership */
export type TenantMembership = UserMembershipRecord;

/** Member budget update payload */
export type MemberBudgetUpdate = Partial<MemberBudget>;

/** Tenant summary response */
export type TenantSummary = {
  id: string;
  name: string;
  status: string;
  role: string;
  created_at: string;
  budget: TenantBudgetSummary;
};

/** Payload for inviting a tenant member */
export interface InviteTenantMemberPayload {
  email: string;
  role: MembershipRole;
}

/** Tenant limit update payload */
export type TenantLimitUpdate = Partial<TenantLimits>;

/** Tenant limit payload */
export type TenantLimitPayload = TenantLimitsPayload;

/** Tenant limit type */
export type TenantLimit = TenantLimits;

// Re-export shared types for convenience
export type { MembershipRole, MemberBudget, TenantModelEntry };

// ============================================================================
// Factory-based Resources
// ============================================================================

/** User tenants resource using factory pattern */
const tenantsResource = createResource<UserTenant>({
  basePath: "/tenants",
  client: userApi,
  transformListResponse: (data) => {
    const response = data as { tenants: UserTenant[] };
    return response.tenants;
  },
});

/** User tenant memberships nested resource */
const membershipsResource = createNestedResource<TenantMembership, InviteTenantMemberPayload>({
  parentPath: "/tenants",
  childPath: "memberships",
  client: userApi,
  transformListResponse: (data) => {
    const response = data as { memberships: TenantMembership[] };
    return response.memberships;
  },
});

// ============================================================================
// Tenant Functions
// ============================================================================

/** List tenants the current user has access to */
export async function listUserTenants(): Promise<UserTenant[]> {
  return tenantsResource.list();
}

/** Get tenant summary */
export async function getTenantSummary(tenantId: string): Promise<TenantSummary> {
  const { data } = await userApi.get<TenantSummary>(`/tenants/${tenantId}/summary`);
  return data;
}

// ============================================================================
// Membership Functions
// ============================================================================

/** List memberships for a tenant */
export async function listTenantMemberships(tenantId: string): Promise<TenantMembership[]> {
  return membershipsResource.list(tenantId);
}

/** Invite a member to a tenant */
export async function inviteTenantMember(
  tenantId: string,
  payload: InviteTenantMemberPayload,
): Promise<TenantMembership> {
  return membershipsResource.create(tenantId, payload);
}

/** Remove a member from a tenant */
export async function removeTenantMember(tenantId: string, userId: string): Promise<void> {
  return membershipsResource.remove(tenantId, userId);
}

/** Update a member's budget */
export async function updateTenantMemberBudget(
  tenantId: string,
  userId: string,
  payload: MemberBudgetUpdate,
): Promise<unknown> {
  const { data } = await userApi.put(
    `/tenants/${tenantId}/memberships/${userId}/budget`,
    payload,
  );
  return data;
}

// ============================================================================
// Rate Limit Functions
// ============================================================================

/** Get rate limits for a tenant */
export async function getTenantLimits(tenantId: string): Promise<TenantLimitPayload> {
  const { data } = await userApi.get<TenantLimitPayload>(`/tenants/${tenantId}/limits`);
  return data;
}

/** Update rate limits for a tenant */
export async function updateTenantLimits(
  tenantId: string,
  payload: TenantLimitUpdate,
): Promise<TenantLimitPayload> {
  const { data } = await userApi.put<TenantLimitPayload>(
    `/tenants/${tenantId}/limits`,
    payload,
  );
  return data;
}

/** Remove rate limit overrides from a tenant */
export async function deleteTenantLimits(tenantId: string): Promise<void> {
  await userApi.delete(`/tenants/${tenantId}/limits`);
}

// ============================================================================
// Model Functions
// ============================================================================

/** List models available to a tenant */
export async function listTenantModels(tenantId: string): Promise<TenantModelEntry[]> {
  const { data } = await userApi.get<{ models: TenantModelEntry[] }>(
    `/tenants/${tenantId}/models`,
  );
  return data.models;
}

/** Attach a model to a tenant */
export async function attachTenantModel(
  tenantId: string,
  alias: string,
): Promise<{ attached: boolean }> {
  const { data } = await userApi.post<{ attached: boolean }>(
    `/tenants/${tenantId}/models/${alias}`,
  );
  return data;
}

/** Detach a model from a tenant */
export async function detachTenantModel(tenantId: string, alias: string): Promise<void> {
  await userApi.delete(`/tenants/${tenantId}/models/${alias}`);
}

// ============================================================================
// Exported Resources (for advanced usage)
// ============================================================================

export const userTenantsResource = tenantsResource;
export const userMembershipsResource = membershipsResource;
