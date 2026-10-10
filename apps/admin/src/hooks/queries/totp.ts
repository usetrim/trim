import { adminFetch } from "@/lib/admin-api/client";
import { adminQk } from "@/lib/query-keys";
import type { TOTPStatus } from "@/types/admin";
import { useQuery } from "@tanstack/react-query";

export function useAdminTOTPStatus(token: string) {
  return useQuery({
    queryKey: adminQk.totp,
    enabled: Boolean(token),
    queryFn: () => adminFetch<TOTPStatus>("/api/v1/admin/auth/totp", { token }),
  });
}
