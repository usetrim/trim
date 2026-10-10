import { adminFetch } from "@/lib/admin-api/client";
import { invalidateAfterStaffMutation } from "@/lib/query-keys";
import type { CreateRoleBody, InviteAdminBody, PatchRoleBody } from "@/types/admin";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function useCreateRole(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateRoleBody) =>
      adminFetch("/api/v1/admin/rbac/roles", {
        method: "POST",
        token,
        body: JSON.stringify(body),
      }),
    onSuccess: async () => {
      await invalidateAfterStaffMutation(qc);
    },
  });
}

export function usePatchRole(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: PatchRoleBody }) =>
      adminFetch(`/api/v1/admin/rbac/roles/${id}`, {
        method: "PATCH",
        token,
        body: JSON.stringify(body),
      }),
    onSuccess: async () => {
      await invalidateAfterStaffMutation(qc);
    },
  });
}

export function useDeleteRole(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      adminFetch(`/api/v1/admin/rbac/roles/${id}`, {
        method: "DELETE",
        token,
      }),
    onSuccess: async () => {
      await invalidateAfterStaffMutation(qc);
    },
  });
}

export function useInviteAdmin(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: InviteAdminBody) =>
      adminFetch("/api/v1/admin/rbac/admins", {
        method: "POST",
        token,
        body: JSON.stringify(body),
      }),
    onSuccess: async () => {
      await invalidateAfterStaffMutation(qc);
    },
  });
}

export function useRemoveAdmin(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) =>
      adminFetch(`/api/v1/admin/rbac/admins/${userId}`, {
        method: "DELETE",
        token,
      }),
    onSuccess: async () => {
      await invalidateAfterStaffMutation(qc);
    },
  });
}
