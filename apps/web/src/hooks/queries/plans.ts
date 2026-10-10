import { apiFetch } from "@/lib/api/client";
import { qk } from "@/lib/query-keys";
import type { BillingInterval } from "@/stores/billing-ui";
import type { PlansResponse } from "@/types/billing";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function usePlans(interval: BillingInterval | null, token?: string) {
  return useQuery({
    queryKey: [...qk.plans, interval ?? "default", token ? "auth" : "anon"],
    queryFn: () => {
      const q = interval != null ? `?interval=${encodeURIComponent(interval)}` : "";
      return apiFetch<PlansResponse>(`/api/v1/public/plans${q}`, {
        token,
      });
    },
    placeholderData: keepPreviousData,
    staleTime: 5 * 60_000,
    gcTime: 30 * 60_000,
  });
}
