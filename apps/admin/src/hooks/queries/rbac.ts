import { adminFetch } from "@/lib/admin-api/client";
import { withQuery } from "@/lib/admin-api/query-string";
import { adminQk } from "@/lib/query-keys";
import type {
  AdminPermission,
  AdminRole,
  AdminRoleOption,
  AdminStaff,
  ListResponse,
} from "@/types/admin";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function useAdminRoles(token: string, skip: number, limit: number, q = "") {
  const search = q.trim();
  return useQuery({
    queryKey: [...adminQk.roles, skip, limit, search],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse<AdminRole>>(
        withQuery("/api/v1/admin/rbac/roles", { skip, limit, q: search || undefined }),
        { token },
      ),
  });
}

export function useAdminRoleOptions(token: string) {
  return useQuery({
    queryKey: adminQk.roleOptions,
    enabled: Boolean(token),
    queryFn: () =>
      adminFetch<ListResponse<AdminRoleOption>>("/api/v1/admin/rbac/roles/options", { token }),
  });
}

export function useAdminPermissions(token: string) {
  return useQuery({
    queryKey: adminQk.permissions,
    enabled: Boolean(token),
    queryFn: () =>
      adminFetch<ListResponse<AdminPermission>>("/api/v1/admin/rbac/permissions", { token }),
  });
}

export function useAdminAdmins(token: string, skip: number, limit: number, q = "") {
  const search = q.trim();
  return useQuery({
    queryKey: [...adminQk.admins, skip, limit, search],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse<AdminStaff>>(
        withQuery("/api/v1/admin/rbac/admins", { skip, limit, q: search || undefined }),
        { token },
      ),
  });
}
