import { adminFetch } from "@/lib/admin-api/client";
import { withQuery } from "@/lib/admin-api/query-string";
import { adminQk } from "@/lib/query-keys";
import type { ListResponse, PlanCatalogItem } from "@/types/admin";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function useAdminPlans(token: string, skip: number, limit: number, q = "") {
  const search = q.trim();
  return useQuery({
    queryKey: [...adminQk.plans, skip, limit, search],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse<PlanCatalogItem>>(
        withQuery("/api/v1/admin/billing/plans", {
          skip,
          limit,
          q: search || undefined,
        }),
        { token },
      ),
  });
}

export function useAdminBillingSettings(token: string) {
  return useQuery({
    queryKey: adminQk.billingSettings,
    enabled: Boolean(token),
    queryFn: () =>
      adminFetch<Record<string, unknown>>("/api/v1/admin/billing/settings", {
        token,
      }),
  });
}

export function useAdminSubscriptions(
  token: string,
  skip: number,
  limit: number,
  filters?: { q?: string; status?: string; plan?: string },
) {
  const q = filters?.q?.trim() || "";
  const status = filters?.status?.trim() || "";
  const plan = filters?.plan?.trim() || "";
  return useQuery({
    queryKey: [...adminQk.subscriptions, skip, limit, q, status, plan],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse>(
        withQuery("/api/v1/admin/billing/subscriptions", {
          skip,
          limit,
          q: q || undefined,
          status: status || undefined,
          plan: plan || undefined,
        }),
        { token },
      ),
  });
}

export function useAdminReceipts(token: string, skip: number, limit: number, q = "") {
  return useQuery({
    queryKey: [...adminQk.receipts, skip, limit, q],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse>(withQuery("/api/v1/admin/billing/receipts", { skip, limit, q }), {
        token,
      }),
  });
}

export function useAdminReceipt(token: string, receiptId: string) {
  const id = receiptId.trim();
  return useQuery({
    queryKey: [...adminQk.receipts, "detail", id],
    enabled: Boolean(token) && Boolean(id),
    queryFn: () =>
      adminFetch<Record<string, unknown>>(
        `/api/v1/admin/billing/receipts/${encodeURIComponent(id)}`,
        { token },
      ),
  });
}

export function useAdminDisputes(token: string, skip: number, limit: number, q = "") {
  const search = q.trim();
  return useQuery({
    queryKey: [...adminQk.disputes, skip, limit, search],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse>(
        withQuery("/api/v1/admin/billing/disputes", {
          skip,
          limit,
          q: search || undefined,
        }),
        { token },
      ),
  });
}
