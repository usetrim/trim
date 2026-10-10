import { adminFetch } from "@/lib/admin-api/client";
import { adminQk } from "@/lib/query-keys";
import { useMutation, useQueryClient } from "@tanstack/react-query";

const refetchActive = { refetchType: "active" as const };

export function useWebhookReplay(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (eventId: string) =>
      adminFetch(`/api/v1/admin/observability/webhooks/${eventId}/replay`, {
        method: "POST",
        token,
      }),
    onSuccess: async (_data, eventId) => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.observabilityWebhooks, ...refetchActive }),
        qc.invalidateQueries({
          queryKey: adminQk.observabilityWebhook(eventId),
          ...refetchActive,
        }),
        qc.invalidateQueries({ queryKey: adminQk.observability, ...refetchActive }),
        qc.invalidateQueries({ queryKey: adminQk.dashboard, ...refetchActive }),
        qc.invalidateQueries({ queryKey: adminQk.audit, ...refetchActive }),
      ]);
    },
  });
}
