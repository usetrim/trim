import { adminFetch } from "@/lib/admin-api/client";
import { withQuery } from "@/lib/admin-api/query-string";
import { adminQk } from "@/lib/query-keys";
import type { ChromeMap, LegalSectionRow, ListResponse, SiteMessageRow } from "@/types/admin";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function useAdminChromeMessages(token: string, skip: number, limit: number, q = "") {
  const search = q.trim();
  return useQuery({
    queryKey: [...adminQk.chromeMessages, skip, limit, search],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse<SiteMessageRow>>(
        withQuery("/api/v1/admin/chrome/messages", {
          skip,
          limit,
          q: search || undefined,
        }),
        { token },
      ),
  });
}

export function useAdminLegal(token: string, skip: number, limit: number, q = "") {
  const search = q.trim();
  return useQuery({
    queryKey: [...adminQk.legal, skip, limit, search],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse<LegalSectionRow>>(
        withQuery("/api/v1/admin/chrome/legal", {
          skip,
          limit,
          q: search || undefined,
        }),
        { token },
      ),
  });
}

export function useAdminNavChrome(token: string) {
  return useQuery({
    queryKey: adminQk.chromeNav,
    enabled: Boolean(token),
    // Shell labels: warm across soft-nav; chrome mutations invalidate when copy changes.
    staleTime: 5 * 60_000,
    refetchOnMount: true,
    refetchOnWindowFocus: false,
    queryFn: async (): Promise<ChromeMap> => {
      const data = await adminFetch<{ messages?: ChromeMap }>("/api/v1/admin/chrome/ui-map", {
        token,
      });
      return data.messages ?? {};
    },
  });
}

/** Alias used by shell. */
export const useAdminChromeNav = useAdminNavChrome;
