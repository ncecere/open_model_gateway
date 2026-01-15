import type { PersonalTenantRecord, TenantStatus } from "@/api/tenants";

export type { PersonalTenantRecord, TenantStatus };

export interface UserFormState {
  email: string;
  name: string;
  password: string;
  sendInvite: boolean;
  status: TenantStatus;
}

export function emptyUserForm(): UserFormState {
  return {
    email: "",
    name: "",
    password: "",
    sendInvite: true,
    status: "active",
  };
}

export function userFormFromRecord(record: PersonalTenantRecord): UserFormState {
  return {
    email: record.user_email,
    name: record.user_name,
    password: "",
    sendInvite: false,
    status: record.status,
  };
}
