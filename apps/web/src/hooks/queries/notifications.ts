import { apiFetch } from "@/lib/api/client";
import { withQuery } from "@/lib/api/query-string";
import { qk } from "@/lib/query-keys";
import type { NotificationsListResponse, NotificationsUnreadResponse } from "@/types/notifications";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";

export function useNotificationsUnread(token: string | undefined) {
  return useQuery({
    queryKey: qk.notificationsUnread,
    enabled: Boolean(token),
    staleTime: 0,
    refetchOnWindowFocus: true,
    queryFn: async () => {
      const data = await apiFetch<NotificationsUnreadResponse>(
        "/api/v1/me/notifications/unread-count",
        { token },
      );
      return data;
    },
    refetchInterval: (query) => {
      const ms = query.state.data?.chrome?.poll_interval_ms;
      return typeof ms === "number" && ms > 0 ? ms : false;
    },
  });
}

/** Skip-based infinite list for the notification bell (DB page size only). */
export function useNotificationsInfinite(
  token: string | undefined,
  limit: number,
  enabled: boolean,
) {
  const pageSize = limit > 0 ? limit : 0;
  return useInfiniteQuery({
    queryKey: [...qk.notifications, "infinite", pageSize],
    enabled: Boolean(token) && pageSize > 0 && enabled,
    staleTime: 0,
    refetchOnMount: "always",
    initialPageParam: 0,
    queryFn: ({ pageParam }) =>
      apiFetch<NotificationsListResponse>(
        withQuery("/api/v1/me/notifications", {
          skip: pageParam,
          limit: pageSize,
        }),
        { token },
      ),
    getNextPageParam: (last) => {
      if (!last.meta?.has_more) return undefined;
      const next = last.meta.next_skip;
      return typeof next === "number" && next >= 0 ? next : undefined;
    },
  });
}
