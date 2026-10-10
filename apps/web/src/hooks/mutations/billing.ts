import { apiFetch, getApiBase, readJSONOrFail, requireAccessToken } from "@/lib/api/client";
import { invalidateBilling } from "@/lib/query-keys";
import type { BillingInterval } from "@/stores/billing-ui";
import type { CheckoutSession, PortalSessionResponse, SyncReceiptsResponse } from "@/types/billing";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function useCheckoutSession(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: {
      plan_id: string;
      interval: BillingInterval;
      quantity?: number;
      user_id: string;
      email: string;
      inquiry_id?: string;
      confirm?: boolean;
    }) => {
      const headers = new Headers();
      headers.set("Accept", "application/json");
      headers.set("Content-Type", "application/json");
      if (token) headers.set("Authorization", `Bearer ${token}`);
      let res: Response;
      try {
        res = await fetch(`${getApiBase()}/api/v1/billing/checkout-session`, {
          method: "POST",
          headers,
          body: JSON.stringify(body),
          signal: AbortSignal.timeout(60_000),
        });
      } catch {
        throw new Error("");
      }
      const data = (await readJSONOrFail(res)) as CheckoutSession & {
        error?: string;
      };
      if (!res.ok) {
        if (data.message || data.code) {
          return data;
        }
        throw new Error(data.error || "");
      }
      return data;
    },
    onSuccess: async (data) => {
      if (data.action === "upgrade_preview") return;
      await invalidateBilling(qc);
    },
    meta: { skipToast: true },
  });
}

export function usePortalSession(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const headers = new Headers();
      headers.set("Accept", "application/json");
      if (token) headers.set("Authorization", `Bearer ${token}`);
      let res: Response;
      try {
        res = await fetch(`${getApiBase()}/api/v1/billing/portal-session`, {
          method: "POST",
          headers,
          signal: AbortSignal.timeout(60_000),
        });
      } catch {
        throw new Error("");
      }
      const data = (await readJSONOrFail(res)) as PortalSessionResponse;
      if (!res.ok) {
        throw new Error(data.error || "");
      }
      return data;
    },
    onSuccess: async () => {
      // Returning from Stripe should not show a stale plan/quota snapshot.
      await invalidateBilling(qc);
    },
    meta: { skipToast: true },
  });
}

export function useSyncReceipts(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiFetch<SyncReceiptsResponse>("/api/v1/billing/receipts/sync", {
        method: "POST",
        token,
      }),
    onSuccess: async () => {
      await invalidateBilling(qc);
    },
  });
}

/**
 * @deprecated Invoice Download must use printReceiptArticle (live article →
 * Save as PDF). Do not call the Go /pdf endpoint - layout diverges from Print.
 */
export function useDownloadReceiptPdf(_token: string | undefined) {
  return useMutation({
    mutationFn: async (args: {
      href: string;
      failedMessage?: string;
      filenameFmt?: string;
      displayId?: string;
    }) => {
      void _token;
      void args.href;
      void args.filenameFmt;
      void args.displayId;
      throw new Error(
        args.failedMessage ||
          "Invoice PDF download uses Print → Save as PDF (not the Go /pdf API).",
      );
    },
    meta: { skipToast: true },
  });
}

export function useEnterpriseInquiry(token: string | undefined) {
  return useMutation({
    mutationFn: async (body: {
      company_name?: string;
      estimated_seats?: number;
      message: string;
    }) => {
      requireAccessToken(token);
      return apiFetch<{ id: string; status: string; message: string }>(
        "/api/v1/billing/enterprise-inquiry",
        {
          method: "POST",
          token,
          body: JSON.stringify(body),
        },
      );
    },
    meta: { skipToast: true },
  });
}
