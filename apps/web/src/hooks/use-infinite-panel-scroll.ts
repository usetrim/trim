"use client";

import { useEffect, type RefObject } from "react";

/**
 * Infinite scroll for popover/panel lists: IntersectionObserver on a sentinel
 * plus scroll-near-bottom fallback. Rebinds when the panel opens or list layout
 * changes (Radix popovers often miss the first observer attach).
 */
export function useInfinitePanelScroll(opts: {
  enabled: boolean;
  scrollRef: RefObject<HTMLElement | null>;
  sentinelRef: RefObject<HTMLElement | null>;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  fetchNextPage: () => void;
  /** Rebind when page content/layout changes. */
  layoutKey: string | number;
}) {
  const {
    enabled,
    scrollRef,
    sentinelRef,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
    layoutKey,
  } = opts;

  useEffect(() => {
    // layoutKey intentionally rebinds after list length / open / fetch state changes.
    void layoutKey;
    if (!enabled || !hasNextPage) return;
    const root = scrollRef.current;
    const target = sentinelRef.current;
    if (!root || !target) return;

    let cancelled = false;
    const loadMore = () => {
      if (cancelled || !hasNextPage || isFetchingNextPage) return;
      fetchNextPage();
    };

    const nearBottom = () => {
      const remaining = root.scrollHeight - root.scrollTop - root.clientHeight;
      return remaining <= 64;
    };

    const onScroll = () => {
      if (nearBottom()) loadMore();
    };

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) loadMore();
      },
      { root, rootMargin: "80px 0px", threshold: 0 },
    );
    observer.observe(target);
    root.addEventListener("scroll", onScroll, { passive: true });

    // Short lists: sentinel is in view without scrolling - keep paging until full or done.
    const kick = () => {
      if (cancelled) return;
      if (nearBottom()) loadMore();
      const rootRect = root.getBoundingClientRect();
      const tRect = target.getBoundingClientRect();
      if (tRect.top <= rootRect.bottom + 80) loadMore();
    };
    kick();
    const raf = requestAnimationFrame(kick);
    const t = window.setTimeout(kick, 50);

    return () => {
      cancelled = true;
      observer.disconnect();
      root.removeEventListener("scroll", onScroll);
      cancelAnimationFrame(raf);
      window.clearTimeout(t);
    };
  }, [enabled, scrollRef, sentinelRef, hasNextPage, isFetchingNextPage, fetchNextPage, layoutKey]);
}
