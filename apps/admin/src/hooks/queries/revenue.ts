import { adminFetch } from "@/lib/admin-api/client";
import { withQuery } from "@/lib/admin-api/query-string";
import { adminQk } from "@/lib/query-keys";
import type { RevenueResponse } from "@/types/admin/revenue";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function useAdminRevenue(token: string, range: string, createdFrom = "", createdTo = "") {
  const r = range.trim();
  const from = createdFrom.trim();
  const to = createdTo.trim();
  // Empty range is valid: API resolves the first active preset (fail-closed).
  // Custom still needs both bounds before fetching.
  const enabled = Boolean(token) && (r !== "custom" || (Boolean(from) && Boolean(to)));
  return useQuery({
    queryKey: [...adminQk.revenue, r || "__default__", from, to],
    enabled,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<RevenueResponse>(
        withQuery("/api/v1/admin/billing/revenue", {
          range: r || undefined,
          created_from: r === "custom" ? from || undefined : undefined,
          created_to: r === "custom" ? to || undefined : undefined,
        }),
        { token },
      ),
  });
}
