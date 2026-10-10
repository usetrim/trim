import { adminFetch } from "@/lib/admin-api/client";
import { adminQk } from "@/lib/query-keys";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function usePatchProductSettings(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: Record<string, unknown>) =>
      adminFetch("/api/v1/admin/product/settings", {
        method: "PATCH",
        token,
        body: JSON.stringify(body),
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.product }),
        qc.invalidateQueries({ queryKey: adminQk.audit }),
      ]);
    },
  });
}
