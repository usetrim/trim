import { adminFetch } from "@/lib/admin-api/client";
import { adminQk } from "@/lib/query-keys";
import type { DistributionSyncResponse } from "@/types/admin";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function useDistributionSync(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      adminFetch<DistributionSyncResponse>("/api/v1/admin/distribution/sync", {
        method: "POST",
        token,
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.distribution }),
        qc.invalidateQueries({ queryKey: adminQk.audit }),
      ]);
    },
  });
}
