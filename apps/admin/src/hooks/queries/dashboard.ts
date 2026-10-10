import { adminFetch } from "@/lib/admin-api/client";
import { adminQk } from "@/lib/query-keys";
import type { AdminDashboard } from "@/types/admin";
import { useQuery } from "@tanstack/react-query";

export function useAdminDashboard(token: string) {
  return useQuery({
    queryKey: adminQk.dashboard,
    enabled: Boolean(token),
    // Dashboard is many DB round-trips; avoid stacking 15s aborts into a blank home.
    retry: 1,
    queryFn: () => adminFetch<AdminDashboard>("/api/v1/admin/dashboard", { token }),
  });
}
