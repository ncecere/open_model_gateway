import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  inviteTenantMember,
  removeTenantMember,
  updateTenantMemberBudget,
  updateTenantLimits,
  deleteTenantLimits,
  attachTenantModel,
  detachTenantModel,
  type MembershipRole,
} from "@/api/user/tenants";
import { useToast } from "@/hooks/use-toast";

export type MemberBudgetUpdatePayload = {
  budget_usd?: number;
  warning_threshold?: number;
  token_cap?: number;
};

export type TenantLimitPayloadRequest = {
  requests_per_minute: number;
  tokens_per_minute: number;
  parallel_requests: number;
};

export type InviteMemberPayload = {
  email: string;
  role: MembershipRole;
};

type UseTenantMutationsOptions = {
  tenantId: string | undefined;
  onInviteSuccess?: () => void;
  onRemoveSuccess?: () => void;
  onBudgetUpdateSuccess?: () => void;
  onLimitsUpdateSuccess?: () => void;
  onLimitsResetSuccess?: () => void;
};

export function useTenantMutations({
  tenantId,
  onInviteSuccess,
  onRemoveSuccess,
  onBudgetUpdateSuccess,
  onLimitsUpdateSuccess,
  onLimitsResetSuccess,
}: UseTenantMutationsOptions) {
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const invalidateMemberships = () => {
    queryClient.invalidateQueries({
      queryKey: ["user-tenant-memberships", tenantId],
    });
  };

  const invalidateLimits = () => {
    queryClient.invalidateQueries({
      queryKey: ["user-tenant-limits", tenantId],
    });
  };

  const invalidateModels = () => {
    queryClient.invalidateQueries({
      queryKey: ["user-tenant-models", tenantId],
    });
  };

  const inviteMutation = useMutation({
    mutationFn: (payload: InviteMemberPayload) => {
      if (!tenantId) {
        return Promise.reject(new Error("tenant not selected"));
      }
      return inviteTenantMember(tenantId, payload);
    },
    onSuccess: () => {
      toast({ title: "Member added" });
      invalidateMemberships();
      onInviteSuccess?.();
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to add member",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const removeMutation = useMutation({
    mutationFn: (userId: string) => {
      if (!tenantId) {
        return Promise.reject(new Error("tenant not selected"));
      }
      return removeTenantMember(tenantId, userId);
    },
    onSuccess: () => {
      toast({ title: "Membership removed" });
      invalidateMemberships();
      onRemoveSuccess?.();
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to remove member",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const budgetMutation = useMutation({
    mutationFn: ({ userId, payload }: { userId: string; payload: MemberBudgetUpdatePayload }) => {
      if (!tenantId) {
        return Promise.reject(new Error("tenant not selected"));
      }
      return updateTenantMemberBudget(tenantId, userId, payload);
    },
    onSuccess: () => {
      toast({ title: "Member budget updated" });
      invalidateMemberships();
      onBudgetUpdateSuccess?.();
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to update budget",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const limitMutation = useMutation({
    mutationFn: (payload: TenantLimitPayloadRequest) => {
      if (!tenantId) {
        return Promise.reject(new Error("tenant not selected"));
      }
      return updateTenantLimits(tenantId, payload);
    },
    onSuccess: () => {
      toast({ title: "Tenant limits updated" });
      invalidateLimits();
      onLimitsUpdateSuccess?.();
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to update limits",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const clearLimitsMutation = useMutation({
    mutationFn: () => {
      if (!tenantId) {
        return Promise.reject(new Error("tenant not selected"));
      }
      return deleteTenantLimits(tenantId);
    },
    onSuccess: () => {
      toast({ title: "Tenant limits reset" });
      invalidateLimits();
      onLimitsResetSuccess?.();
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to reset limits",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const attachModelMutation = useMutation({
    mutationFn: (alias: string) => {
      if (!tenantId) {
        return Promise.reject(new Error("tenant not selected"));
      }
      return attachTenantModel(tenantId, alias);
    },
    onSuccess: () => {
      invalidateModels();
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to attach model",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const detachModelMutation = useMutation({
    mutationFn: (alias: string) => {
      if (!tenantId) {
        return Promise.reject(new Error("tenant not selected"));
      }
      return detachTenantModel(tenantId, alias);
    },
    onSuccess: () => {
      invalidateModels();
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to detach model",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  return {
    inviteMutation,
    removeMutation,
    budgetMutation,
    limitMutation,
    clearLimitsMutation,
    attachModelMutation,
    detachModelMutation,
  };
}
