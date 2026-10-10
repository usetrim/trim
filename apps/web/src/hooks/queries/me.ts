import { apiFetch } from "@/lib/api/client";
import { withQuery } from "@/lib/api/query-string";
import { qk } from "@/lib/query-keys";
import type { EventStats, EventsListResponse } from "@/types/events";
import type { PreferencesResponse, QuotaResponse } from "@/types/me";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function useQuota(token: string | undefined) {
  return useQuery({
    queryKey: qk.quota,
    enabled: Boolean(token),
    // Must track plan_catalog.unlimited / paid webhook entitlements immediately (no 5m soft-nav cache).
    staleTime: 0,
    refetchOnMount: "always",
    queryFn: () => apiFetch<QuotaResponse>("/api/v1/me/quota", { token }),
  });
}

export function useEvents(
  token: string | undefined,
  skip: number,
  limit?: number,
  dateFrom?: string | null,
  dateTo?: string | null,
  q?: string | null,
) {
  const pageSize = typeof limit === "number" && limit > 0 ? limit : 0;
  const from = (dateFrom || "").trim();
  const to = (dateTo || "").trim();
  const search = (q || "").trim();
  return useQuery({
    queryKey: [...qk.events, skip, pageSize, from, to, search],
    enabled: Boolean(token) && pageSize > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      apiFetch<EventsListResponse>(
        withQuery("/api/v1/me/events", {
          skip,
          limit: pageSize,
          from: from || undefined,
          to: to || undefined,
          q: search || undefined,
        }),
        { token },
      ),
  });
}

export function useEventStats(token: string | undefined, groupBy?: string, heatmapScope?: string) {
  const gb = (groupBy || "").trim();
  const hs = (heatmapScope || "").trim();
  return useQuery({
    queryKey: [...qk.eventStats, gb, hs],
    enabled: Boolean(token),
    placeholderData: keepPreviousData,
    // Cold chart build + remote DB can exceed default retry stacking.
    retry: 1,
    queryFn: () =>
      apiFetch<EventStats>(
        withQuery("/api/v1/me/events/stats", {
          group_by: gb || undefined,
          heatmap_scope: hs || undefined,
        }),
        { token },
      ),
  });
}

export function usePreferences(token: string | undefined) {
  return useQuery({
    queryKey: qk.preferences,
    enabled: Boolean(token),
    // Settings must mirror live profile/chrome immediately (defaults + admin chrome edits).
    staleTime: 0,
    refetchOnMount: "always",
    queryFn: () => apiFetch<PreferencesResponse>("/api/v1/me/preferences", { token }),
  });
}
