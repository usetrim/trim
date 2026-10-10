import { adminFetch } from "@/lib/admin-api/client";
import { adminQk } from "@/lib/query-keys";
import { useQuery } from "@tanstack/react-query";

export function useAdminProduct(token: string) {
  return useQuery({
    queryKey: adminQk.product,
    enabled: Boolean(token),
    queryFn: () =>
      adminFetch<Record<string, unknown>>("/api/v1/admin/product/settings", {
        token,
      }),
  });
}
