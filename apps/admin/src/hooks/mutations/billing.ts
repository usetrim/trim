import { adminFetch } from "@/lib/admin-api/client";
import { adminQk, invalidateAfterBillingMutation } from "@/lib/query-keys";
import type { PatchPlanBody } from "@/types/admin";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function usePatchPlan(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: PatchPlanBody }) =>
      adminFetch(`/api/v1/admin/billing/plans/${id}`, {
        method: "PATCH",
        token,
        body: JSON.stringify(body),
      }),
    onSuccess: async () => {
      await invalidateAfterBillingMutation(qc);
    },
  });
}

export function useCreatePlan(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: Record<string, unknown>) =>
      adminFetch<{ id: string }>("/api/v1/admin/billing/plans", {
        method: "POST",
        token,
        body: JSON.stringify(body),
      }),
    onSuccess: async () => {
      await invalidateAfterBillingMutation(qc);
    },
  });
}

export function useSyncPaddleCatalog(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      adminFetch("/api/v1/admin/billing/plans/sync-paddle", {
        method: "POST",
        token,
      }),
    onSuccess: async () => {
      await invalidateAfterBillingMutation(qc);
    },
  });
}

export function usePatchBillingSettings(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: Record<string, unknown>) =>
      adminFetch("/api/v1/admin/billing/settings", {
        method: "PATCH",
        token,
        body: JSON.stringify(body),
      }),
    onSuccess: async () => {
      await invalidateAfterBillingMutation(qc);
    },
  });
}

export function usePostDisputeNote(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: {
      paddle_transaction_id?: string;
      user_id?: string;
      note: string;
      status: string;
    }) =>
      adminFetch<{ id: string }>("/api/v1/admin/billing/disputes", {
        method: "POST",
        token,
        body: JSON.stringify(body),
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.receipts, refetchType: "active" }),
        qc.invalidateQueries({ queryKey: adminQk.disputes, refetchType: "active" }),
        qc.invalidateQueries({ queryKey: adminQk.audit, refetchType: "active" }),
      ]);
    },
  });
}

export function useReceiptResync(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      adminFetch<{ message?: string }>(
        `/api/v1/admin/billing/receipts/${encodeURIComponent(id)}/resync`,
        { method: "POST", token },
      ),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.receipts, refetchType: "active" }),
        qc.invalidateQueries({ queryKey: adminQk.audit, refetchType: "active" }),
      ]);
    },
  });
}

export function useReceiptPDFReissue(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      adminFetch<{ message?: string; paddle_invoice_pdf_url?: string }>(
        `/api/v1/admin/billing/receipts/${encodeURIComponent(id)}/pdf-reissue`,
        { method: "POST", token },
      ),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.receipts, refetchType: "active" }),
        qc.invalidateQueries({ queryKey: adminQk.audit, refetchType: "active" }),
      ]);
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

/** Download first-party admin tax-invoice PDF as a real file (not print). */
export function useDownloadAdminReceiptPdf(token: string) {
  return useMutation({
    mutationFn: async (args: {
      href: string;
      failedMessage?: string;
      filenameFmt?: string;
      displayId?: string;
    }) => {
      const href = (args.href || "").trim();
      const failed = args.failedMessage || "";
      if (!href.startsWith("/") || !token) {
        throw new Error(failed);
      }
      const headers = new Headers();
      headers.set("Accept", "application/pdf");
      headers.set("Authorization", `Bearer ${token}`);
      const base = process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, "") || "";
      if (!base) {
        throw new Error(failed);
      }
      let res: Response;
      try {
        res = await fetch(`${base}${href}`, {
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
