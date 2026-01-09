import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  createTenantApiKey,
  revokeTenantApiKey,
  type CreateApiKeyRequest,
} from "@/api/tenants";
import { useToast } from "@/hooks/use-toast";

const ADMIN_API_KEYS_QUERY_KEY = ["admin-api-keys"] as const;

export function useAdminKeyMutations() {
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const createKeyMutation = useMutation({
    mutationFn: ({
      tenantId,
      payload,
    }: {
      tenantId: string;
      payload: CreateApiKeyRequest;
    }) => createTenantApiKey(tenantId, payload),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ADMIN_API_KEYS_QUERY_KEY });
      toast({
        title: "API key created",
        description: `${data.api_key.name} issued successfully`,
      });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to create key",
        description: "Please retry.",
      });
    },
  });

  const revokeKeyMutation = useMutation({
    mutationFn: ({
      tenantId,
      apiKeyId,
    }: {
      tenantId: string;
      apiKeyId: string;
    }) => revokeTenantApiKey(tenantId, apiKeyId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ADMIN_API_KEYS_QUERY_KEY });
      toast({ title: "API key revoked" });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to revoke key",
        description: "Try again in a moment.",
      });
    },
  });

  return {
    createKeyMutation,
    revokeKeyMutation,
    ADMIN_API_KEYS_QUERY_KEY,
  };
}
