import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  createTenant,
  updateTenantStatus,
  upsertTenantMembership,
  removeTenantMembership,
  type MembershipRole,
  type TenantStatus,
} from "@/api/tenants";
import { useToast } from "@/hooks/use-toast";
import { TENANTS_QUERY_KEY, TENANTS_DASHBOARD_KEY } from "../index";

const MEMBERSHIPS_QUERY_KEY = (tenantId?: string) =>
  ["tenant-memberships", tenantId] as const;

export type UseAdminTenantMutationsReturn = ReturnType<typeof useAdminTenantMutations>;

export function useAdminTenantMutations() {
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const createTenantMutation = useMutation({
    mutationFn: createTenant,
    onSuccess: (tenant) => {
      toast({
        title: "Tenant created",
        description: `${tenant.name} is now active`,
      });
      queryClient.invalidateQueries({ queryKey: TENANTS_QUERY_KEY });
      queryClient.invalidateQueries({ queryKey: TENANTS_DASHBOARD_KEY });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to create tenant",
        description: "Please retry in a moment.",
      });
    },
  });

  const updateStatusMutation = useMutation({
    mutationFn: updateTenantStatus,
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: TENANTS_QUERY_KEY });
      queryClient.invalidateQueries({ queryKey: TENANTS_DASHBOARD_KEY });
      queryClient.invalidateQueries({
        queryKey: MEMBERSHIPS_QUERY_KEY(variables.tenantId),
      });
      toast({ title: "Tenant status updated" });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to update status",
        description: "Check your permissions and try again.",
      });
    },
  });

  const upsertMembershipMutation = useMutation({
    mutationFn: ({
      tenantId,
      payload,
    }: {
      tenantId: string;
      payload: { email: string; role: MembershipRole };
    }) => upsertTenantMembership(tenantId, payload),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: MEMBERSHIPS_QUERY_KEY(variables.tenantId),
      });
      toast({ title: "Membership updated" });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to update membership",
        description: "Check the email and try again.",
      });
    },
  });

  const removeMembershipMutation = useMutation({
    mutationFn: ({ tenantId, userId }: { tenantId: string; userId: string }) =>
      removeTenantMembership(tenantId, userId),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: MEMBERSHIPS_QUERY_KEY(variables.tenantId),
      });
      toast({ title: "Membership removed" });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to remove membership",
        description: "Try again shortly.",
      });
    },
  });

  const handleStatusChange = async (tenantId: string, status: TenantStatus) => {
    try {
      await updateStatusMutation.mutateAsync({ tenantId, status });
    } catch (error) {
      console.error(error);
    }
  };

  return {
    createTenantMutation,
    updateStatusMutation,
    upsertMembershipMutation,
    removeMembershipMutation,
    handleStatusChange,
    MEMBERSHIPS_QUERY_KEY,
  };
}
