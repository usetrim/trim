import { adminFetch } from "@/lib/admin-api/client";
import { withQuery } from "@/lib/admin-api/query-string";
import { adminQk } from "@/lib/query-keys";
import type { ListResponse } from "@/types/admin";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function useAdminEnterprise(
  token: string,
  skip: number,
  limit: number,
  q = "",
  status = "",
) {
  const search = q.trim();
  const statusFilter = status.trim();
  return useQuery({
    queryKey: [...adminQk.enterprise, skip, limit, search, statusFilter],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse>(
        withQuery("/api/v1/admin/enterprise/inquiries", {
          skip,
          limit,
          q: search || undefined,
          status: statusFilter || undefined,
        }),
        { token },
      ),
  });
}
