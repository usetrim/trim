import type { PaginationMeta } from "./common";

export type EventListItem = {
  id: string;
  request_id: string | null;
  model: string | null;
  tokens_before: number;
  tokens_after: number;
  latency_ms: number;
  mode: string;
  mode_label?: string;
  status: string;
  status_label?: string;
  created_at: string;
  created_at_label?: string;
};

export type EventsListResponse = {
  items: EventListItem[];
  meta: PaginationMeta;
  empty_message?: string;
  col_when?: string;
  col_model?: string;
  col_mode?: string;
  col_tokens?: string;
  col_latency?: string;
  tokens_sep?: string;
  latency_unit?: string;
  date_from?: string;
  date_to?: string;
  date_range_placeholder?: string;
  date_range_description?: string;
  date_range_clear?: string;
  date_range_apply?: string;
  date_range_months?: number;
  search_placeholder?: string;
  search_description?: string;
  q?: string;
  col_status?: string;
  col_request_id?: string;
  col_view?: string;
  open_action_label?: string;
  table_row_actions?: string;
  preview_field_description?: string;
  table_select_all?: string;
  table_select_row?: string;
  table_selected_fmt?: string;
  table_bulk_delete?: string;
  table_clear_selection?: string;
  delete_action_label?: string;
  delete_pending_label?: string;
  delete_confirm_message?: string;
  bulk_delete_confirm_message?: string;
};

export type EventStats = {
  model_breakdown: Array<{
    name: string;
    count: number;
    tokens?: number;
    saved?: number;
  }>;
  status_breakdown: Array<{ name: string; code?: string; count: number }>;
  mode_breakdown: Array<{ name: string; code?: string; count: number }>;
  token_series: Array<{
    day: string;
    day_label?: string;
    tokens_before: number;
    tokens_after: number;
    tokens_saved: number;
  }>;
  acceptance: {
    tab_suggestions_shown: number;
    tab_suggestions_accepted: number;
    acceptance_rate: number;
  };
  loc: {
    ai_lines_added: number;
    ai_lines_deleted: number;
  };
  total_events: number;
  success_count?: number;
  success_rate?: number;
  empty_models_message?: string;
  empty_status_message?: string;
  empty_modes_message?: string;
  empty_series_message?: string;
  success_rate_prefix?: string;
  runs_series_name?: string;
  scope_full_label?: string;
  scope_page_label?: string;
  traces_unit?: string;
  status_success_label?: string;
  status_error_label?: string;
  status_success_code?: string;
  runs_fmt?: string;
  model_tooltip_fmt?: string;
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
