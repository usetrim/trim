"use client";

import { Button } from "@/components/ui/button";
import { FetchProgressBar } from "@/components/ui/fetch-progress";
import { FormattedWhen } from "@/components/ui/formatted-when";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Skeleton } from "@/components/ui/skeleton";
import {
  useAdminMarkAllNotificationsRead,
  useAdminMarkNotificationRead,
} from "@/hooks/mutations/notifications";
import { useAuthProviders } from "@/hooks/queries/auth";
import {
  useAdminNotificationsInfinite,
  useAdminNotificationsUnread,
} from "@/hooks/queries/notifications";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { useInfinitePanelScroll } from "@/hooks/use-infinite-panel-scroll";
import { useNotificationHrefNavigate } from "@/hooks/use-notification-href-navigate";
import { cn } from "@/lib/utils";
import type { NotificationItem } from "@/types/admin/notifications";
import { Bell } from "lucide-react";
import { useCallback, useMemo, useRef, useState } from "react";

function bellAriaWithCount(aria: string, count: number, fmt: string | undefined): string {
  if (!aria) return "";
  if (count <= 0) return aria;
  const template = (fmt || "").trim();
  if (!template || !template.includes("{aria}") || !template.includes("{count}")) {
    return aria;
  }
  return template.replaceAll("{aria}", aria).replaceAll("{count}", String(count));
}

export function AdminNotificationBell({ token }: { token: string }) {
  const [open, setOpen] = useState(false);
  const pageSize = useDefaultPageSize();
  const auth = useAuthProviders();
  const unread = useAdminNotificationsUnread(token);
  const list = useAdminNotificationsInfinite(token, pageSize, open);
  const markOne = useAdminMarkNotificationRead(token);
  const markAll = useAdminMarkAllNotificationsRead(token);
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
  const htmlLang =
    (typeof auth.data?.site?.html_lang === "string" ? auth.data.site.html_lang : "").trim() || "";
  const markReadPending = chrome?.mark_read_pending?.trim() || "";

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
    if (token && unread.isPending && !unread.data && !unread.isError) {
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
              className="absolute -right-1 -top-1 flex h-4 min-w-4 items-center justify-center rounded-full border border-background bg-foreground px-1 text-[10px] font-semibold tabular-nums text-background"
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
        className="z-[100] w-[min(20rem,calc(100vw-1.5rem))] border-border bg-popover p-0 shadow-lg"
      >
        <div className="flex items-center justify-between border-b border-border px-3 py-2">
          <p className="text-sm font-medium text-foreground">{panelTitle}</p>
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
              <div className="h-4 w-40 animate-pulse rounded bg-muted" />
              <div className="h-3 w-full animate-pulse rounded bg-muted" />
              <div className="h-3 w-3/4 animate-pulse rounded bg-muted" />
            </div>
          ) : null}
          {!list.isPending && items.length === 0 ? (
            <p className="px-3 py-6 text-sm text-muted-foreground">{chrome?.empty_message || ""}</p>
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
                  "border-b border-border px-3 py-3 text-sm last:border-0",
                  !item.read ? "bg-accent/40" : "",
                  clickable ? "cursor-pointer text-left transition hover:bg-accent/60" : "",
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
                <p className="font-medium text-foreground">{item.title}</p>
                <p className="mt-1 text-xs text-muted-foreground">{item.body}</p>
                <FormattedWhen
                  value={item.created_at}
                  locale={htmlLang}
                  variant="relative"
                  className="mt-1 block text-[10px] text-muted-foreground"
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
              className="px-3 py-2 text-center text-xs text-muted-foreground"
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
