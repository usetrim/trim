import { adminFetch } from "@/lib/admin-api/client";
import { withQuery } from "@/lib/admin-api/query-string";
import { adminQk } from "@/lib/query-keys";
import type {
  HeatmapItem,
  ListResponse,
  ObservabilityStats,
  WebhookEventDetail,
  WebhookEventItem,
} from "@/types/admin";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function useAdminObservability(token: string, groupBy?: string, heatmapScope?: string) {
  const gb = (groupBy || "").trim();
  const hs = (heatmapScope || "").trim();
  return useQuery({
    queryKey: [...adminQk.observability, gb, hs],
    enabled: Boolean(token),
    retry: 1,
    queryFn: () =>
      adminFetch<ObservabilityStats>(
        withQuery("/api/v1/admin/observability/events/stats", {
          group_by: gb || undefined,
          heatmap_scope: hs || undefined,
        }),
        { token },
      ),
  });
}

export function useAdminObservabilityWebhooks(token: string, skip: number, limit: number, q = "") {
  const search = q.trim();
  return useQuery({
    queryKey: [...adminQk.observabilityWebhooks, skip, limit, search],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse<WebhookEventItem>>(
        withQuery("/api/v1/admin/observability/webhooks", {
          skip,
          limit,
          q: search || undefined,
        }),
        { token },
      ),
  });
}

export function useAdminObservabilityWebhook(token: string, eventId: string) {
  return useQuery({
    queryKey: adminQk.observabilityWebhook(eventId),
    enabled: Boolean(token) && Boolean(eventId),
    queryFn: () =>
      adminFetch<WebhookEventDetail>(
        `/api/v1/admin/observability/webhooks/${encodeURIComponent(eventId)}`,
        { token },
      ),
  });
}

export function useAdminHeatmap(token: string) {
  return useQuery({
    queryKey: adminQk.heatmap,
    enabled: Boolean(token),
    queryFn: () =>
      adminFetch<ListResponse<HeatmapItem> & { title?: string }>(
        "/api/v1/admin/observability/heatmap",
        { token },
      ),
  });
}
