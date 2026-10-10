"use client";

import { Button } from "@/components/ui/button";
import { FetchProgressBar } from "@/components/ui/fetch-progress";
import { FormattedWhen } from "@/components/ui/formatted-when";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Skeleton } from "@/components/ui/skeleton";
import {
  useMarkAllNotificationsRead,
  useMarkNotificationRead,
} from "@/hooks/mutations/notifications";
import { useAuthProviders } from "@/hooks/queries/auth";
import { useNotificationsInfinite, useNotificationsUnread } from "@/hooks/queries/notifications";
import { useInfinitePanelScroll } from "@/hooks/use-infinite-panel-scroll";
import { useNotificationHrefNavigate } from "@/hooks/use-notification-href-navigate";
import { cn } from "@/lib/utils";
import type { NotificationItem } from "@/types/notifications";
import { Bell } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

function bellAriaWithCount(aria: string, count: number, fmt: string | undefined): string {
  if (!aria) return "";
  if (count <= 0) return aria;
  const template = (fmt || "").trim();
  if (!template || !template.includes("{aria}") || !template.includes("{count}")) {
    return aria;
  }
  return template.replaceAll("{aria}", aria).replaceAll("{count}", String(count));
}

export function NotificationBell({
  accessToken,
}: {
  accessToken?: string;
}) {
  const [open, setOpen] = useState(false);
  const unread = useNotificationsUnread(accessToken);
  const auth = useAuthProviders();
  const resolvedPageSize =
    auth.data?.default_page_size && auth.data.default_page_size > 0
      ? auth.data.default_page_size
      : 0;
  const [stablePageSize, setStablePageSize] = useState(resolvedPageSize);
  useEffect(() => {
    if (resolvedPageSize > 0) setStablePageSize(resolvedPageSize);
  }, [resolvedPageSize]);
  const pageSize = resolvedPageSize > 0 ? resolvedPageSize : stablePageSize;

  const list = useNotificationsInfinite(accessToken, pageSize, open);
  const markOne = useMarkNotificationRead(accessToken);
  const markAll = useMarkAllNotificationsRead(accessToken);
  const navigateHref = useNotificationHrefNavigate();
  const sentinelRef = useRef<HTMLDivElement | null>(null);
  const scrollRef = useRef<HTMLDivElement | null>(null);

  const chrome = unread.data?.chrome || list.data?.pages?.[0]?.chrome;
  const bellAria = chrome?.bell_aria?.trim() || "";
  const panelTitle = chrome?.panel_title?.trim() || "";
  const count = unread.data?.count ?? 0;
  const badgeLabel = (unread.data?.badge_label || "").trim();
  const triggerAria = bellAriaWithCount(bellAria, count, chrome?.bell_aria_count_fmt);
  const loadingMoreLabel = chrome?.loading_more?.trim() || "";
  const markReadPending = chrome?.mark_read_pending?.trim() || "";
  const htmlLang = auth.data?.site?.html_lang?.trim() || "";

  const items = useMemo<NotificationItem[]>(() => {
    const pages = list.data?.pages ?? [];
    const out: NotificationItem[] = [];
    const seen = new Set<string>();
    for (const page of pages) {
      for (const item of page.items ?? []) {
        if (!item?.id || seen.has(item.id)) continue;
        seen.add(item.id);
        out.push(item);
      }
    }
    return out;
  }, [list.data?.pages]);

  const fetchNextPage = useCallback(() => {
    void list.fetchNextPage();
  }, [list.fetchNextPage]);

  useInfinitePanelScroll({
    enabled: open && Boolean(bellAria && panelTitle) && !list.isPending,
    scrollRef,
    sentinelRef,
    hasNextPage: Boolean(list.hasNextPage),
    isFetchingNextPage: list.isFetchingNextPage,
    fetchNextPage,
    layoutKey: `${items.length}:${list.isFetchingNextPage ? 1 : 0}:${open ? 1 : 0}`,
  });

  // Reserve the header slot while unread chrome settles - never flash null → bell.
  if (!bellAria || !panelTitle) {
    if (accessToken && unread.isPending && !unread.data && !unread.isError) {
      return <Skeleton className="h-8 w-8 shrink-0 rounded-md" aria-hidden />;
    }
    return null;
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="relative px-2"
          aria-label={triggerAria}
        >
          <Bell className="h-4 w-4" />
          {badgeLabel ? (
            <span
              className="absolute -right-1 -top-1 flex h-4 min-w-4 items-center justify-center rounded-full border border-[var(--trim-panel)] bg-[var(--trim-ink)] px-1 text-[10px] font-semibold tabular-nums text-[var(--trim-ink-inverse)]"
              aria-hidden
            >
              {badgeLabel}
            </span>
          ) : null}
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="end"
        sticky="always"
        sideOffset={8}
        collisionPadding={12}
        className="trim-float z-[100] w-[min(20rem,calc(100vw-1.5rem))] border-[var(--trim-border-strong)] bg-[var(--trim-panel)] p-0"
      >
        <div className="flex items-center justify-between border-b border-[var(--trim-border)] px-3 py-2">
          <p className="text-sm font-medium text-[var(--trim-fg)]">{panelTitle}</p>
          {chrome?.mark_all_read && count > 0 ? (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-7 text-xs"
              isLoading={Boolean(chrome.mark_all_pending?.trim()) && markAll.isPending}
              pendingLabel={chrome.mark_all_pending || undefined}
              onClick={() => markAll.mutate()}
            >
              {chrome.mark_all_read}
            </Button>
          ) : null}
        </div>
        <FetchProgressBar
          active={
            (list.isFetching && !list.isPending && items.length > 0) ||
            markOne.isPending ||
            markAll.isPending
          }
        />
        <div ref={scrollRef} className="max-h-80 overflow-y-auto">
          {list.isError ? (
            <p className="px-3 py-4 text-sm text-destructive">{chrome?.list_failed || ""}</p>
          ) : null}
          {list.isPending && items.length === 0 ? (
            <div className="space-y-3 px-3 py-4" aria-busy="true">
              <div className="h-4 w-40 animate-pulse rounded bg-[var(--trim-track)]" />
              <div className="h-3 w-full animate-pulse rounded bg-[var(--trim-track)]" />
              <div className="h-3 w-3/4 animate-pulse rounded bg-[var(--trim-track)]" />
            </div>
          ) : null}
          {!list.isPending && items.length === 0 ? (
            <p className="px-3 py-6 text-sm text-[var(--trim-muted)]">
              {chrome?.empty_message || ""}
            </p>
          ) : null}
          {items.map((item) => {
            const clickable = Boolean(item.href?.startsWith("/")) && !item.href?.startsWith("//");
            const go = () => {
              if (!clickable || !item.href) return;
              if (!item.read) markOne.mutate(item.id);
              setOpen(false);
              navigateHref(item.href);
            };
            return (
              <div
                key={item.id}
                role={clickable ? "link" : undefined}
                tabIndex={clickable ? 0 : undefined}
                className={cn(
                  "border-b border-[var(--trim-border)] px-3 py-3 text-sm last:border-0",
                  !item.read ? "bg-[var(--trim-hover)]" : "",
                  clickable
                    ? "cursor-pointer text-left transition hover:bg-[var(--trim-hover)]"
                    : "",
                )}
                onClick={clickable ? go : undefined}
                onKeyDown={
                  clickable
                    ? (e) => {
                        if (e.key !== "Enter" && e.key !== " ") return;
                        e.preventDefault();
                        go();
                      }
                    : undefined
                }
              >
                <p className="font-medium text-[var(--trim-fg)]">{item.title}</p>
                <p className="mt-1 text-xs text-[var(--trim-muted)]">{item.body}</p>
                <FormattedWhen
                  value={item.created_at}
                  locale={htmlLang}
                  variant="relative"
                  className="mt-1 block text-[10px] text-[var(--trim-muted)]"
                />
                {!item.read && chrome?.mark_read ? (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="mt-2 h-7 px-0 text-xs"
                    isLoading={Boolean(markReadPending) && markOne.isPending}
                    pendingLabel={markReadPending || undefined}
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      markOne.mutate(item.id);
                    }}
                  >
                    {chrome.mark_read}
                  </Button>
                ) : null}
              </div>
            );
          })}
          {list.isFetchingNextPage && loadingMoreLabel ? (
            <p
              className="px-3 py-2 text-center text-xs text-[var(--trim-muted)]"
              aria-live="polite"
              aria-busy="true"
            >
              {loadingMoreLabel}
            </p>
          ) : null}
          <div ref={sentinelRef} className="h-4 w-full shrink-0" aria-hidden />
        </div>
      </PopoverContent>
    </Popover>
  );
}
