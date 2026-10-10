export type AdminRole = {
  id: string;
  name: string;
  slug: string;
  permissions?: string[];
  is_system?: boolean;
  is_owner?: boolean;
};

export type AdminRoleOption = {
  id: string;
  name: string;
  slug: string;
};

export type AdminPermission = {
  code: string;
  description: string;
};

export type AdminStaff = {
  user_id: string;
  email: string;
  role_slug: string;
  status: string;
  status_label?: string;
};

export type CreateRoleBody = {
  name: string;
  slug: string;
  permissions: string[];
};

export type PatchRoleBody = {
  name?: string;
  description?: string;
  permissions?: string[];
};

export type InviteAdminBody = {
  email: string;
  role_id: string;
};
