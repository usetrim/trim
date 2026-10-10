import { adminFetch } from "@/lib/admin-api/client";
import { downloadRowsAsXlsx } from "@/lib/export-xlsx";
import { adminQk } from "@/lib/query-keys";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export type AuditExportResponse = {
  items: Array<Record<string, unknown>>;
  generated_at?: string;
  filename?: string;
  message?: string;
};

export function useExportAudit(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const data = await adminFetch<AuditExportResponse>("/api/v1/admin/audit/export", { token });
      const nameFromApi = (data.filename || "").trim();
      // Server filename only (site_messages). Fail closed; never invent a download name.
      if (!nameFromApi) {
        throw new Error("");
      }
      await downloadRowsAsXlsx(data.items ?? [], nameFromApi, "Audit");
      return data;
    },
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: adminQk.audit });
    },
  });
}
