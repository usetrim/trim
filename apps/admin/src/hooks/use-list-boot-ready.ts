"use client";

import { useEffect, useState } from "react";

/**
 * Primary query is ready to leave the page shimmer.
 *
 * Professional cache policy (with QueryClient staleTime + refetchOnMount: true):
 * - Cold load (no cache): shimmer until the first result arrives.
 * - Soft-nav with fresh cache: show content immediately (no remount refetch).
 * - Soft-nav after mutation invalidate (stale): remount refetches; list hooks
 *   keepPreviousData + FetchProgressBar - not a full-page skeleton.
 * - Debounced search changes the query key; the boot latch keeps filters mounted.
 */
export function queryContentReady(
  q: {
    data?: unknown;
    isPending: boolean;
    isFetching?: boolean;
  },
  opts?: {
    /** When false (query disabled), do not block boot on this query. */
    enabled?: boolean;
  },
): boolean {
  if (opts?.enabled === false) {
    return true;
  }

  // Cold load: nothing to paint yet.
  if (q.data === undefined && q.isPending) {
    return false;
  }

  // Warm cache (fresh or stale-while-revalidate) - leave full-page shimmer.
  if (q.data !== undefined) {
    return true;
  }

  // Settled empty / error with no payload.
  return !q.isPending;
}

/**
 * Leave shimmer when pageSize + chrome + primary content are ready.
 * Latch so later refetch/invalidation/search cannot remount filters mid-typing.
 * Always start unlatched on mount so cold soft-nav can shimmer again.
 */
export function useListBootReady(
  pageSize: number,
  chromeReady: boolean,
  contentReady = true,
): boolean {
  const nowReady = pageSize > 0 && chromeReady && contentReady;
  const [latched, setLatched] = useState(false);

  useEffect(() => {
    if (nowReady) setLatched(true);
  }, [nowReady]);

  return latched || nowReady;
}
