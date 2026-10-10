import { adminFetch } from "@/lib/admin-api/client";
import { adminQk } from "@/lib/query-keys";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function usePatchEmailTemplate(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ code, body }: { code: string; body: string }) =>
      adminFetch(`/api/v1/admin/email/templates/${encodeURIComponent(code)}`, {
        method: "PATCH",
        token,
        body: JSON.stringify({ body }),
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.emailTemplates }),
        qc.invalidateQueries({ queryKey: adminQk.chromeMessages }),
        qc.invalidateQueries({ queryKey: adminQk.chromeNav }),
        qc.invalidateQueries({ queryKey: adminQk.audit }),
      ]);
    },
  });
}
