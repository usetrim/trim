import { adminFetch } from "@/lib/admin-api/client";
import { withQuery } from "@/lib/admin-api/query-string";
import { adminQk } from "@/lib/query-keys";
import type {
  NotificationsListResponse,
  NotificationsUnreadResponse,
} from "@/types/admin/notifications";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";

export function useAdminNotificationsUnread(token: string | undefined) {
  return useQuery({
    queryKey: adminQk.notificationsUnread,
    enabled: Boolean(token),
    refetchOnWindowFocus: true,
    queryFn: () =>
      adminFetch<NotificationsUnreadResponse>("/api/v1/admin/notifications/unread-count", {
        token,
      }),
    refetchInterval: (query) => {
      const ms = query.state.data?.chrome?.poll_interval_ms;
      return typeof ms === "number" && ms > 0 ? ms : false;
    },
  });
}

/** Skip-based infinite list for the admin notification bell (DB page size only). */
export function useAdminNotificationsInfinite(
  token: string | undefined,
  limit: number,
  enabled: boolean,
) {
  const pageSize = limit > 0 ? limit : 0;
  return useInfiniteQuery({
    queryKey: [...adminQk.notifications, "infinite", pageSize],
    enabled: Boolean(token) && pageSize > 0 && enabled,
    staleTime: 0,
    refetchOnMount: "always",
    initialPageParam: 0,
    queryFn: ({ pageParam }) =>
      adminFetch<NotificationsListResponse>(
        withQuery("/api/v1/admin/notifications", {
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
