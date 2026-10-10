import { adminFetch } from "@/lib/admin-api/client";
import { adminQk } from "@/lib/query-keys";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function usePatchAuthSettings(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (allowed_providers: string[]) =>
      adminFetch("/api/v1/admin/auth/settings", {
        method: "PATCH",
        token,
        body: JSON.stringify({ allowed_providers }),
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.authSettings }),
        qc.invalidateQueries({ queryKey: adminQk.publicAuthProviders }),
        qc.invalidateQueries({ queryKey: adminQk.oauthScopes }),
        qc.invalidateQueries({ queryKey: adminQk.audit }),
      ]);
    },
  });
}
