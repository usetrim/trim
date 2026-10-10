import { apiFetch } from "@/lib/api/client";
import { withQuery } from "@/lib/api/query-string";
import { qk } from "@/lib/query-keys";
import type { ReceiptsListResponse, SubscriptionStatus } from "@/types/billing";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function useSubscriptionStatus(token: string | undefined) {
  return useQuery({
    queryKey: qk.subscription,
    enabled: Boolean(token),
    // Sales activate / plan flips must not sit behind the global 5m soft-nav cache.
    staleTime: 0,
    refetchOnMount: "always",
    queryFn: () => apiFetch<SubscriptionStatus>("/api/v1/billing/subscription", { token }),
  });
}

export function useReceipts(
  token: string | undefined,
  skip: number,
  limit?: number,
  q?: string | null,
) {
  const pageSize = typeof limit === "number" && limit > 0 ? limit : 0;
  const search = (q || "").trim();
  return useQuery({
    queryKey: [...qk.receipts, skip, pageSize, search],
    enabled: Boolean(token) && pageSize > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      apiFetch<ReceiptsListResponse>(
        withQuery("/api/v1/billing/receipts", {
          skip,
          limit: pageSize,
          q: search || undefined,
        }),
        { token },
      ),
  });
}

export function useReceipt(token: string | undefined, receiptId: string) {
  return useQuery({
    queryKey: qk.receipt(receiptId),
    enabled: Boolean(token) && Boolean(receiptId),
    queryFn: () => apiFetch(`/api/v1/billing/receipts/${receiptId}`, { token }),
  });
}
