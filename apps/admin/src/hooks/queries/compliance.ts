import { adminFetch } from "@/lib/admin-api/client";
import { withQuery } from "@/lib/admin-api/query-string";
import { adminQk } from "@/lib/query-keys";
import type { ListResponse } from "@/types/admin";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function useAdminCompliance(token: string) {
  return useQuery({
    queryKey: adminQk.compliance,
    enabled: Boolean(token),
    queryFn: () =>
      adminFetch<Record<string, unknown>>("/api/v1/admin/compliance/retention", { token }),
  });
}

export function useAdminAccessReview(token: string, skip: number, limit: number, q = "") {
  const search = q.trim();
  return useQuery({
    queryKey: [...adminQk.accessReview, skip, limit, search],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<
        ListResponse<Record<string, unknown>> & {
          attestations?: Array<Record<string, unknown>>;
          generated_at?: string;
        }
      >(
        withQuery("/api/v1/admin/compliance/access-review", {
          skip,
          limit,
          q: search || undefined,
        }),
        {
          token,
        },
      ),
  });
}
