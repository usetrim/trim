import { adminFetch } from "@/lib/admin-api/client";
import { withQuery } from "@/lib/admin-api/query-string";
import { adminQk } from "@/lib/query-keys";
import type { DistributionStatsResponse } from "@/types/admin";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function useAdminDistribution(token: string, range: string, skip: number, limit: number) {
  return useQuery({
    queryKey: [...adminQk.distribution, range, skip, limit],
    enabled: Boolean(token) && Boolean(range) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<DistributionStatsResponse>(
        withQuery("/api/v1/admin/distribution/stats", { range, skip, limit }),
        { token },
      ),
  });
}
