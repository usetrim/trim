import { adminFetch } from "@/lib/admin-api/client";
import { adminQk } from "@/lib/query-keys";
import type { WebAuthnStatus } from "@/types/admin";
import { useQuery } from "@tanstack/react-query";

export function useAdminWebAuthnStatus(token: string) {
  return useQuery({
    queryKey: adminQk.webauthn,
    enabled: Boolean(token),
    queryFn: () => adminFetch<WebAuthnStatus>("/api/v1/admin/auth/webauthn", { token }),
  });
}
