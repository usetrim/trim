import type { PaginationMeta } from "@/types/admin";

export type NotificationChrome = {
  bell_aria: string;
  bell_aria_count_fmt?: string;
  panel_title: string;
  empty_message: string;
  mark_all_read: string;
  mark_all_pending: string;
  mark_read: string;
  mark_read_pending?: string;
  unread_label: string;
  loading_more?: string;
  poll_interval_ms: number;
  list_failed: string;
  mark_failed: string;
};

export type NotificationItem = {
  id: string;
  kind_code: string;
  title: string;
  body: string;
  href?: string;
  read: boolean;
  created_at: string;
};

export type NotificationsListResponse = {
  items: NotificationItem[];
  meta: PaginationMeta;
  chrome: NotificationChrome;
};

export type NotificationsUnreadResponse = {
  count: number;
  badge_label: string;
  chrome: NotificationChrome;
};
