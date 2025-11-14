export type UserProfile = {
  id: string;
  email: string;
  name: string;
  created_at: string;
  updated_at: string;
  last_login_at?: string | null;
  personal_tenant_id?: string | null;
  is_super_admin?: boolean;
  can_change_password?: boolean;
};
