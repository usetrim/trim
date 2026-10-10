import { adminFetch } from "@/lib/admin-api/client";
import { invalidateAfterStaffMutation } from "@/lib/query-keys";
import type { BreakGlassCreateBody } from "@/types/admin";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function usePostBreakGlass(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: BreakGlassCreateBody) =>
      adminFetch("/api/v1/admin/break-glass", {
        method: "POST",
        token,
        body: JSON.stringify(body),
      }),
    onSuccess: async () => {
      await invalidateAfterStaffMutation(qc);
    },
  });
}

export function useResolveBreakGlass(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, approve }: { id: string; approve: boolean }) =>
      adminFetch(`/api/v1/admin/break-glass/${id}/${approve ? "approve" : "deny"}`, {
        method: "POST",
        token,
      }),
    onSuccess: async () => {
      await invalidateAfterStaffMutation(qc);
    },
  });
}

export function useRevokeBreakGlass(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      adminFetch(`/api/v1/admin/break-glass/${id}/revoke`, {
        method: "POST",
        token,
      }),
    onSuccess: async () => {
      await invalidateAfterStaffMutation(qc);
    },
  });
}
