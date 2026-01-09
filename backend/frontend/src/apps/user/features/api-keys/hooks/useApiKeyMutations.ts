import { useToast } from "@/hooks/use-toast";
import {
  useCreateUserAPIKeyMutation,
  useCreateTenantAPIKeyMutation,
  useRevokeUserAPIKeyMutation,
  useRevokeTenantAPIKeyMutation,
  useRotateUserAPIKeyMutation,
  useRotateTenantAPIKeyMutation,
} from "@/apps/user/hooks/useUserData";
import type { CreateUserAPIKeyRequest, UserAPIKey } from "@/api/user/api-keys";
import type { IssuedSecret } from "../components/IssuedSecretCard";

export type UseApiKeyMutationsOptions = {
  selectedTenantId?: string;
  selectedTenantName?: string;
  tenantOptions: Array<{ tenant_id: string; name: string }>;
  onIssued: (secret: IssuedSecret) => void;
  onPersonalDialogClose?: () => void;
  onTenantDialogClose?: () => void;
};

export function useApiKeyMutations({
  selectedTenantId,
  selectedTenantName,
  tenantOptions,
  onIssued,
  onPersonalDialogClose,
  onTenantDialogClose,
}: UseApiKeyMutationsOptions) {
  const { toast } = useToast();

  const createMutation = useCreateUserAPIKeyMutation();
  const revokeMutation = useRevokeUserAPIKeyMutation();
  const rotateMutation = useRotateUserAPIKeyMutation();
  const tenantCreateMutation = useCreateTenantAPIKeyMutation();
  const tenantRevokeMutation = useRevokeTenantAPIKeyMutation();
  const tenantRotateMutation = useRotateTenantAPIKeyMutation();

  const handleCopy = (value: string) => {
    navigator.clipboard.writeText(value).then(() => {
      toast({ title: "Copied to clipboard" });
    });
  };

  const handlePersonalCreate = async (payload: CreateUserAPIKeyRequest) => {
    try {
      const result = await createMutation.mutateAsync(payload);
      onIssued({
        scope: "personal",
        name: result.api_key.name,
        prefix: result.api_key.prefix,
        secret: result.secret,
        token: result.token,
      });
      onPersonalDialogClose?.();
      toast({ title: "API key created", description: "Copy the secret now." });
    } catch (error) {
      console.error(error);
      toast({ variant: "destructive", title: "Failed to create key" });
      throw error;
    }
  };

  const handleTenantCreate = async (payload: CreateUserAPIKeyRequest) => {
    if (!selectedTenantId) {
      toast({
        variant: "destructive",
        title: "Select a tenant",
        description: "Choose a tenant before creating a key.",
      });
      throw new Error("No tenant selected");
    }

    try {
      const result = await tenantCreateMutation.mutateAsync({
        tenantId: selectedTenantId,
        payload,
      });
      onIssued({
        scope: "tenant",
        tenantName: selectedTenantName,
        name: result.api_key.name,
        prefix: result.api_key.prefix,
        secret: result.secret,
        token: result.token,
      });
      onTenantDialogClose?.();
      toast({
        title: "Tenant API key created",
        description: `Key issued for ${selectedTenantName ?? "tenant"}.`,
      });
    } catch (error) {
      console.error(error);
      toast({ variant: "destructive", title: "Failed to create tenant key" });
      throw error;
    }
  };

  const handleRevoke = async (id: string) => {
    try {
      await revokeMutation.mutateAsync(id);
      toast({ title: "API key revoked" });
    } catch (error) {
      console.error(error);
      toast({ variant: "destructive", title: "Failed to revoke key" });
    }
  };

  const handleTenantRevoke = async (keyId: string) => {
    if (!selectedTenantId) {
      toast({ variant: "destructive", title: "Select a tenant first" });
      return;
    }
    try {
      await tenantRevokeMutation.mutateAsync({
        tenantId: selectedTenantId,
        apiKeyId: keyId,
      });
      toast({ title: "Tenant API key revoked" });
    } catch (error) {
      console.error(error);
      toast({ variant: "destructive", title: "Failed to revoke tenant key" });
    }
  };

  const handleRotate = async (key: UserAPIKey) => {
    try {
      const result = await rotateMutation.mutateAsync(key.id);
      onIssued({
        scope: "personal",
        name: result.api_key.name,
        prefix: result.api_key.prefix,
        secret: result.secret,
        token: result.token,
      });
      toast({ title: "API key rotated", description: "Copy the new secret now." });
    } catch (error) {
      console.error(error);
      toast({ variant: "destructive", title: "Failed to rotate key" });
    }
  };

  const handleTenantRotate = async (key: UserAPIKey) => {
    const tenantId = key.tenant_id;
    if (!tenantId) {
      toast({ variant: "destructive", title: "Missing tenant for this key" });
      return;
    }
    try {
      const result = await tenantRotateMutation.mutateAsync({
        tenantId,
        apiKeyId: key.id,
      });
      const tenantName =
        tenantOptions.find((t) => t.tenant_id === tenantId)?.name ?? selectedTenantName;
      onIssued({
        scope: "tenant",
        tenantName,
        name: result.api_key.name,
        prefix: result.api_key.prefix,
        secret: result.secret,
        token: result.token,
      });
      toast({
        title: "Tenant API key rotated",
        description: "Copy the new secret now.",
      });
    } catch (error) {
      console.error(error);
      toast({ variant: "destructive", title: "Failed to rotate tenant key" });
    }
  };

  return {
    handleCopy,
    handlePersonalCreate,
    handleTenantCreate,
    handleRevoke,
    handleTenantRevoke,
    handleRotate,
    handleTenantRotate,
    createMutation,
    tenantCreateMutation,
    revokeMutation,
    tenantRevokeMutation,
    rotateMutation,
    tenantRotateMutation,
  };
}
