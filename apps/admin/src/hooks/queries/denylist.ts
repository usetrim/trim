import { adminFetch } from "@/lib/admin-api/client";
import { withQuery } from "@/lib/admin-api/query-string";
import { adminQk } from "@/lib/query-keys";
import type { ListResponse } from "@/types/admin";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function useAdminDenylistEmail(token: string, skip: number, limit: number, q = "") {
  const search = q.trim();
  return useQuery({
    queryKey: [...adminQk.denylistEmail, skip, limit, search],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse>(
        withQuery("/api/v1/admin/denylist/email", { skip, limit, q: search || undefined }),
        { token },
      ),
  });
}

export function useAdminDenylistIP(token: string, skip: number, limit: number, q = "") {
  const search = q.trim();
  return useQuery({
    queryKey: [...adminQk.denylistIp, skip, limit, search],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse>(
        withQuery("/api/v1/admin/denylist/ip", { skip, limit, q: search || undefined }),
        { token },
      ),
  });
}

export function useAdminDenylistASN(token: string, skip: number, limit: number, q = "") {
  const search = q.trim();
  return useQuery({
    queryKey: [...adminQk.denylistAsn, skip, limit, search],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse>(
        withQuery("/api/v1/admin/denylist/asn", { skip, limit, q: search || undefined }),
        { token },
      ),
  });
}
