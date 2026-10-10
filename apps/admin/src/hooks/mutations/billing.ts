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

/**
 * @deprecated Invoice Download must use printReceiptArticle (live article →
 * Save as PDF). Do not call the Go /pdf endpoint — layout diverges from Print.
 */
export function useDownloadAdminReceiptPdf(_token: string) {
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
