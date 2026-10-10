import { adminFetch } from "@/lib/admin-api/client";
import { adminQk } from "@/lib/query-keys";
import type { AdminMe } from "@/types/admin";
import { useQuery } from "@tanstack/react-query";

export function useAdminMe(token: string) {
  return useQuery({
    queryKey: adminQk.me,
    enabled: Boolean(token),
    // Soft-nav reuses /me while fresh; step-up / staff mutations invalidate.
    staleTime: 5 * 60_000,
    refetchOnMount: true,
    refetchOnWindowFocus: false,
    queryFn: () => adminFetch<AdminMe>("/api/v1/admin/me", { token }),
    retry: false,
  });
}
