import { apiFetch } from "@/lib/api/client";
import { withQuery } from "@/lib/api/query-string";
import { qk } from "@/lib/query-keys";
import type { MeEnterpriseInquiriesResponse } from "@/types/enterprise";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function useMeEnterpriseInquiries(
  token: string | undefined,
  skip: number,
  limit?: number,
  status?: string | null,
  q?: string | null,
) {
  const pageSize = typeof limit === "number" && limit > 0 ? limit : 0;
  const statusFilter = (status || "").trim();
  const search = (q || "").trim();
  return useQuery({
    queryKey: [...qk.enterpriseInquiries, skip, pageSize, statusFilter, search],
    enabled: Boolean(token) && pageSize > 0,
    staleTime: 0,
    refetchOnMount: "always",
    placeholderData: keepPreviousData,
    queryFn: () =>
      apiFetch<MeEnterpriseInquiriesResponse>(
        withQuery("/api/v1/me/enterprise-inquiries", {
          skip,
          limit: pageSize,
          status: statusFilter || undefined,
          q: search || undefined,
        }),
        { token },
      ),
  });
}
