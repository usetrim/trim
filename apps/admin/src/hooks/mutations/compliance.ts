import { adminFetch } from "@/lib/admin-api/client";
import { downloadRowsAsXlsx } from "@/lib/export-xlsx";
import { adminQk } from "@/lib/query-keys";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function usePatchComplianceRetention(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: Record<string, unknown>) =>
      adminFetch("/api/v1/admin/compliance/retention", {
        method: "PATCH",
        token,
        body: JSON.stringify(body),
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.compliance }),
        qc.invalidateQueries({ queryKey: adminQk.audit }),
      ]);
    },
  });
}

export function usePurgeComplianceRetention(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { reason: string }) =>
      adminFetch<{ message?: string; result?: Record<string, unknown> }>(
        "/api/v1/admin/compliance/retention/purge",
        {
          method: "POST",
          token,
          body: JSON.stringify(body),
        },
      ),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.compliance }),
        qc.invalidateQueries({ queryKey: adminQk.audit }),
        qc.invalidateQueries({ queryKey: adminQk.observability }),
      ]);
    },
  });
}

export function useExportAccessReview(token: string) {
  return useMutation({
    mutationFn: async () => {
      const data = await adminFetch<{
        items?: Array<Record<string, unknown>>;
        generated_at?: string;
        filename?: string;
        message?: string;
      }>("/api/v1/admin/compliance/access-review/export", { token });
      const nameFromApi = (data.filename || "").trim();
      // Server filename only (site_messages). Fail closed; never invent a download name.
      if (!nameFromApi) {
        throw new Error("");
      }
      await downloadRowsAsXlsx(data.items ?? [], nameFromApi, "Access review");
      return data;
    },
  });
}

export function useAttestAccessReview(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: {
      period_label: string;
      notes: string;
      reason: string;
    }) =>
      adminFetch<{ id?: string; message?: string }>(
        "/api/v1/admin/compliance/access-review/attest",
        {
          method: "POST",
          token,
          body: JSON.stringify(body),
        },
      ),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.accessReview }),
        qc.invalidateQueries({ queryKey: adminQk.audit }),
      ]);
    },
  });
}
