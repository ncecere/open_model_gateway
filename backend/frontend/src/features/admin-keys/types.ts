import type { AdminKeyRecord, AdminKeyScope } from "@/api/admin-keys";

export type { AdminKeyRecord, AdminKeyScope };

export type AdminKeyStatus = "active" | "expired" | "revoked";

export type AdminKeyScopeFilter = "all" | AdminKeyScope;
export type AdminKeyStatusFilter = "all" | AdminKeyStatus;

export interface AdminKeyFormState {
  name: string;
  scope: AdminKeyScope;
  expiresDays: string;
}

export function emptyAdminKeyForm(): AdminKeyFormState {
  return {
    name: "",
    scope: "admin",
    expiresDays: "30",
  };
}

export function getKeyStatus(key: AdminKeyRecord): AdminKeyStatus {
  if (key.revoked_at) return "revoked";
  if (key.expires_at && new Date(key.expires_at) < new Date()) return "expired";
  return "active";
}

export function isKeyActive(key: AdminKeyRecord): boolean {
  return getKeyStatus(key) === "active";
}
