import type { PaginationMeta } from "./common";

export type HeatmapItem = {
  country: string;
  users?: number;
  login_users?: number;
  paid_users?: number;
};

export type ObservabilityStats = {
  total_events?: number;
  tokens_before?: number;
  tokens_after?: number;
  last_24h?: {
    success?: number;
    error?: number;
  };
  by_mode?: Array<{ mode?: string; count?: number }>;
  by_model?: Array<{ model?: string; count?: number }>;
  cli?: {
    min_cli_version?: string;
    force_upgrade_notice?: string;
  };
  usage_series?: UsageDaySeries[];
  usage_days?: UsageAxisDay[];
  usage_group_by_options?: NamedOption[];
  usage_group_by_selected?: string;
  usage_title?: string;
  usage_subtitle?: string;
  usage_y_axis?: string;
  usage_today_label?: string;
  usage_group_by_prefix?: string;
  usage_empty?: string;
  usage_tooltip_breakdown?: string;
  usage_tooltip_daily_total?: string;
  usage_tooltip_cumulative_total?: string;
  usage_tooltip_share_fmt?: string;
  today_day?: string;
  loc_heatmap?: LocHeatmapDay[];
  loc_heatmap_total?: number;
  loc_heatmap_scopes?: NamedOption[];
  loc_heatmap_scope_selected?: string;
  loc_heatmap_title?: string;
  loc_heatmap_empty_fmt?: string;
  loc_heatmap_value_fmt?: string;
  loc_heatmap_weekday_labels?: NamedOption[];
  loc_heatmap_stats?: LocHeatmapStat[];
};

export type NamedOption = {
  id: string;
  label: string;
};

export type UsageAxisDay = {
  day: string;
  day_label?: string;
  daily_total?: number;
  daily_total_label?: string;
  cum_total?: number;
  cum_total_label?: string;
};

export type UsageDaySeries = {
  day: string;
  day_label?: string;
  series_id: string;
  series_label: string;
  tokens: number;
  daily_tokens?: number;
  tokens_label?: string;
  daily_tokens_label?: string;
};

export type LocHeatmapDay = {
  day: string;
  day_label_long?: string;
  weekday_mon0?: number;
  month_label?: string;
  value: number;
  intensity?: number;
};

export type LocHeatmapStat = {
  id: string;
  label: string;
  value: string;
};

export type WebhookEventItem = {
  event_id?: string;
  event_type?: string;
  processed_at?: string;
  process_status?: string;
  last_error?: string;
};

export type WebhookEventDetail = WebhookEventItem & {
  payload?: unknown;
  payload_raw?: string;
};

export type DistributionStatItem = {
  id?: string;
  day?: string;
  source?: string;
  source_label?: string;
  metric?: string;
  metric_label?: string;
  country?: string;
  path?: string;
  value?: number;
};

export type DistributionStatsResponse = {
  range?: string;
  items?: DistributionStatItem[];
  meta?: PaginationMeta;
};

export type DistributionSyncResponse = {
  github_notice?: string;
  channel_notices?: string[];
  install_hits_aggregated?: boolean;
  github_configured?: boolean;
};
