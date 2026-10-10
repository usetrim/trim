import { adminFetch } from "@/lib/admin-api/client";
import { adminQk } from "@/lib/query-keys";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function usePatchChromeMessage(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ code, body }: { code: string; body: string }) =>
      adminFetch(`/api/v1/admin/chrome/messages/${encodeURIComponent(code)}`, {
        method: "PATCH",
        token,
        body: JSON.stringify({ body }),
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.chromeMessages }),
        qc.invalidateQueries({ queryKey: adminQk.chromeNav }),
        qc.invalidateQueries({ queryKey: adminQk.audit }),
      ]);
    },
  });
}

export function usePatchLegalSection(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: Record<string, unknown> }) =>
      adminFetch(`/api/v1/admin/chrome/legal/${id}`, {
        method: "PATCH",
        token,
        body: JSON.stringify(body),
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.legal }),
        qc.invalidateQueries({ queryKey: adminQk.audit }),
      ]);
    },
  });
}
