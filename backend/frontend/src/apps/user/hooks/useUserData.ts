import { useMutation, useQueries, useQuery, useQueryClient, type UseQueryOptions } from "@tanstack/react-query";
import {
  changeUserPassword,
  getUserProfile,
  updateUserProfile,
  type ChangeUserPasswordRequest,
  type UpdateUserProfileRequest,
} from "../../../api/user/profile";
import {
  createUserAPIKey,
  createTenantAPIKey,
  getUserAPIKeyUsage,
  getTenantAPIKeyUsage,
  listUserAPIKeys,
  listTenantAPIKeys,
  revokeUserAPIKey,
  revokeTenantAPIKey,
  type CreateUserAPIKeyRequest,
} from "../../../api/user/api-keys";
import {
  getUserDashboard,
  getUserUsage,
  type UserUsageParams,
  type UserUsageSummary,
} from "../../../api/user/usage";
import {
  getTenantSummary,
  listUserTenants,
} from "../../../api/user/tenants";
import {
  cancelUserTenantBatch,
  listUserTenantBatches,
} from "../../../api/user/batches";
import { listUserFiles } from "../../../api/user/files";

export function useUserProfileQuery() {
  return useQuery({
    queryKey: ["user-profile"],
    queryFn: getUserProfile,
    staleTime: 60_000,
  });
}

export function useUpdateUserProfileMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (payload: UpdateUserProfileRequest) =>
      updateUserProfile(payload),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ["user-profile"] });
    },
  });
}

export function useChangePasswordMutation() {
  return useMutation({
    mutationFn: (payload: ChangeUserPasswordRequest) =>
      changeUserPassword(payload),
  });
}

export function useUserDashboardQuery(period: string = "7d", scope?: string) {
  return useQuery({
    queryKey: ["user-dashboard", period, scope ?? "personal"],
    queryFn: () => getUserDashboard(period, scope),
    staleTime: 30_000,
  });
}

export function useUserUsageQuery(
  params: UserUsageParams = {},
  options?: Omit<UseQueryOptions<UserUsageSummary>, "queryKey" | "queryFn">,
) {
  const { period = "30d", scope, start, end } = params;
  const rangeKey = start && end ? `${start}-${end}` : period;
  return useQuery({
    queryKey: ["user-usage", rangeKey, scope ?? "personal"],
    queryFn: () => getUserUsage({ period, scope, start, end }),
    ...options,
  });
}

export function useUserAPIKeysQuery() {
  return useQuery({
    queryKey: ["user-api-keys"],
    queryFn: listUserAPIKeys,
  });
}

export function useCreateUserAPIKeyMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateUserAPIKeyRequest) =>
      createUserAPIKey(payload),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ["user-api-keys"] });
    },
  });
}

export function useRevokeUserAPIKeyMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => revokeUserAPIKey(id),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ["user-api-keys"] });
    },
  });
}

export function useUserAPIKeyUsageQuery(apiKeyId?: string, period = "30d") {
  return useQuery({
    queryKey: ["user-api-key-usage", apiKeyId, period],
    queryFn: () =>
      apiKeyId ? getUserAPIKeyUsage(apiKeyId, period) : Promise.resolve(null),
    enabled: Boolean(apiKeyId),
  });
}

export function useUserTenantsQuery() {
  return useQuery({
    queryKey: ["user-tenants"],
    queryFn: listUserTenants,
  });
}

export function useTenantSummaryQuery(tenantId?: string) {
  return useQuery({
    queryKey: ["user-tenant-summary", tenantId],
    queryFn: () => (tenantId ? getTenantSummary(tenantId) : Promise.resolve(null)),
    enabled: Boolean(tenantId),
  });
}

export function useTenantAPIKeysQuery(tenantId?: string) {
  return useQuery({
    queryKey: ["tenant-api-keys", tenantId],
    queryFn: () =>
      tenantId ? listTenantAPIKeys(tenantId) : Promise.resolve(null),
    enabled: Boolean(tenantId),
  });
}

export function useCreateTenantAPIKeyMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: { tenantId: string; payload: CreateUserAPIKeyRequest }) =>
      createTenantAPIKey(input.tenantId, input.payload),
    onSuccess: (_data, variables) => {
      client.invalidateQueries({
        queryKey: ["tenant-api-keys", variables.tenantId],
      });
    },
  });
}

export function useRevokeTenantAPIKeyMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: { tenantId: string; apiKeyId: string }) =>
      revokeTenantAPIKey(input.tenantId, input.apiKeyId),
    onSuccess: (_data, variables) => {
      client.invalidateQueries({
        queryKey: ["tenant-api-keys", variables.tenantId],
      });
    },
  });
}

export function useTenantAPIKeyUsageQuery(
  tenantId?: string,
  apiKeyId?: string,
  period = "30d",
) {
  return useQuery({
    queryKey: ["tenant-api-key-usage", tenantId, apiKeyId, period],
    queryFn: () =>
      tenantId && apiKeyId
        ? getTenantAPIKeyUsage(tenantId, apiKeyId, period)
        : Promise.resolve(null),
    enabled: Boolean(tenantId && apiKeyId),
  });
}

export function useAllTenantAPIKeysQueries(tenantIds: string[]) {
  return useQueries({
    queries: tenantIds.map((tenantId) => ({
      queryKey: ["tenant-api-keys", tenantId],
      queryFn: () => listTenantAPIKeys(tenantId),
      enabled: Boolean(tenantId),
    })),
  });
}

export function useUserTenantBatchesQuery(tenantId?: string) {
  return useQuery({
    queryKey: ["user-tenant-batches", tenantId],
    queryFn: () =>
      tenantId ? listUserTenantBatches(tenantId) : Promise.resolve(null),
    enabled: Boolean(tenantId),
  });
}

export function useUserFilesQuery(tenantId?: string, limit = 50) {
  return useQuery({
    queryKey: ["user-files", tenantId, limit],
    queryFn: () =>
      tenantId ? listUserFiles(tenantId, { limit }) : Promise.resolve([]),
    enabled: Boolean(tenantId),
  });
}

export function useCancelUserTenantBatchMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: { tenantId: string; batchId: string }) =>
      cancelUserTenantBatch(input.tenantId, input.batchId),
    onSuccess: (_data, variables) => {
      client.invalidateQueries({
        queryKey: ["user-tenant-batches", variables.tenantId],
      });
    },
  });
}
