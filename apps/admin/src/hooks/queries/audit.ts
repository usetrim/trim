import { adminFetch } from "@/lib/admin-api/client";
import { withQuery } from "@/lib/admin-api/query-string";
import { adminQk } from "@/lib/query-keys";
import type { AdminFilterOption, ListResponse } from "@/types/admin";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export type AuditFilterOptions = {
  actions: AdminFilterOption[];
  actors: AdminFilterOption[];
  resources: AdminFilterOption[];
};

export function useAdminAudit(
  token: string,
  skip: number,
  limit: number,
  filters?: {
    action?: string;
    actor?: string;
    resource?: string;
    created_from?: string;
    created_to?: string;
  },
) {
  const action = filters?.action?.trim() || "";
  const actor = filters?.actor?.trim() || "";
  const resource = filters?.resource?.trim() || "";
  const createdFrom = filters?.created_from?.trim() || "";
  const createdTo = filters?.created_to?.trim() || "";
  return useQuery({
    queryKey: [...adminQk.audit, skip, limit, action, actor, resource, createdFrom, createdTo],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse>(
        withQuery("/api/v1/admin/audit", {
          skip,
          limit,
          action: action || undefined,
          actor: actor || undefined,
          resource: resource || undefined,
          created_from: createdFrom || undefined,
          created_to: createdTo || undefined,
        }),
        { token },
      ),
  });
}

export function useAdminAuditFilterOptions(token: string) {
  return useQuery({
    queryKey: adminQk.auditFilterOptions,
    enabled: Boolean(token),
    queryFn: () => adminFetch<AuditFilterOptions>("/api/v1/admin/audit/filter-options", { token }),
  });
}
