import { apiFetch } from "@/lib/api/client";
import { qk } from "@/lib/query-keys";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function useMarkNotificationRead(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<{ status: string; message?: string }>(`/api/v1/me/notifications/${id}/read`, {
        method: "POST",
        token,
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: qk.notifications }),
        qc.invalidateQueries({ queryKey: qk.notificationsUnread }),
      ]);
    },
  });
}

export function useMarkAllNotificationsRead(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiFetch<{ status: string; message?: string }>("/api/v1/me/notifications/read-all", {
        method: "POST",
        token,
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: qk.notifications }),
        qc.invalidateQueries({ queryKey: qk.notificationsUnread }),
      ]);
    },
  });
}
