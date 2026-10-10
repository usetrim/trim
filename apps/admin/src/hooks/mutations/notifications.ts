import { adminFetch } from "@/lib/admin-api/client";
import { adminQk } from "@/lib/query-keys";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function useAdminMarkNotificationRead(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      adminFetch<{ status: string; message?: string }>(`/api/v1/admin/notifications/${id}/read`, {
        method: "POST",
        token,
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.notifications }),
        qc.invalidateQueries({ queryKey: adminQk.notificationsUnread }),
      ]);
    },
  });
}

export function useAdminMarkAllNotificationsRead(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      adminFetch<{ status: string; message?: string }>("/api/v1/admin/notifications/read-all", {
        method: "POST",
        token,
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.notifications }),
        qc.invalidateQueries({ queryKey: adminQk.notificationsUnread }),
      ]);
    },
  });
}
