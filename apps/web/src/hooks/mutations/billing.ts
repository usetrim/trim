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

function filenameStem(displayId: string | undefined): string {
  return (displayId || "")
    .trim()
    .replace(/[^\w.-]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function filenameFromContentDisposition(cd: string): string {
  const raw = (cd || "").trim();
  if (!raw) return "";
  const match = /filename\*=(?:UTF-8''|utf-8'')([^;]+)|filename="([^"]+)"|filename=([^;]+)/i.exec(
    raw,
  );
  const value = (match?.[1] || match?.[2] || match?.[3] || "").trim();
  if (!value) return "";
  try {
    return decodeURIComponent(value.replace(/^["']|["']$/g, ""));
  } catch {
    return value.replace(/^["']|["']$/g, "");
  }
}

function filenameFromChrome(fmt: string | undefined, displayId: string | undefined): string {
  const stem = filenameStem(displayId);
  const f = (fmt || "").trim();
  if (f.includes("%s") && stem) return f.replace("%s", stem);
  if (stem) return `receipt-${stem}.pdf`;
  return "receipt.pdf";
}

async function readErrorMessage(res: Response, fallback: string): Promise<string> {
  try {
    const body = (await res.json()) as { error?: unknown; message?: unknown };
    const err = typeof body.error === "string" ? body.error.trim() : "";
    if (err) return err;
    const msg = typeof body.message === "string" ? body.message.trim() : "";
    if (msg) return msg;
  } catch {
    /* keep fallback */
  }
  return fallback;
}

/** Download first-party tax-invoice PDF as a real file attachment (not print). */
export function useDownloadReceiptPdf(token: string | undefined) {
  return useMutation({
    mutationFn: async (args: {
      href: string;
      failedMessage?: string;
      filenameFmt?: string;
      displayId?: string;
    }) => {
      const access = requireAccessToken(token);
      const href = (args.href || "").trim();
      const failed = args.failedMessage || "";
      if (!href.startsWith("/")) {
        throw new Error(failed);
      }
      const headers = new Headers();
      headers.set("Accept", "application/pdf");
      headers.set("Authorization", `Bearer ${access}`);
      let res: Response;
      try {
        res = await fetch(`${getApiBase()}${href}`, {
          headers,
          signal: AbortSignal.timeout(60_000),
        });
      } catch {
        throw new Error(failed);
      }
      if (!res.ok) {
        throw new Error(await readErrorMessage(res, failed));
      }
      const buf = await res.arrayBuffer();
      if (!buf || buf.byteLength < 5) {
        throw new Error(failed);
      }
      const head = new TextDecoder("ascii").decode(
        new Uint8Array(buf, 0, Math.min(8, buf.byteLength)),
      );
      if (!head.startsWith("%PDF")) {
        throw new Error(failed);
      }
      const filename =
        (res.headers.get("X-Trim-Download-Filename") || "").trim() ||
        filenameFromContentDisposition(res.headers.get("Content-Disposition") || "") ||
        filenameFromChrome(args.filenameFmt, args.displayId);
      const blob = new Blob([buf], { type: "application/pdf" });
      const url = URL.createObjectURL(blob);
      try {
        const a = document.createElement("a");
        a.href = url;
        a.download = filename;
        a.rel = "noopener";
        a.style.display = "none";
        document.body.appendChild(a);
        a.click();
        a.remove();
      } finally {
        URL.revokeObjectURL(url);
      }
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
