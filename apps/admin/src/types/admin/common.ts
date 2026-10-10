export type ApiError = Error & {
  body?: Record<string, unknown>;
  status?: number;
};

export type PaginationMeta = {
  skip?: number;
  limit?: number;
  total?: number;
  page?: number;
  total_pages?: number;
  prev_skip?: number | null;
  next_skip?: number | null;
  first_skip?: number | null;
  last_skip?: number | null;
  has_more?: boolean;
  page_skips?: number[];
  page_summary?: string;
  prev_action_label?: string;
  next_action_label?: string;
  first_action_label?: string;
  last_action_label?: string;
  first_pending_label?: string;
  prev_pending_label?: string;
  next_pending_label?: string;
  last_pending_label?: string;
  skip_to_label?: string;
  skip_to_action_label?: string;
  skip_to_pending_label?: string;
  skip_to_invalid?: string;
};

export type ListResponse<T = Record<string, unknown>> = {
  items: T[];
  meta?: PaginationMeta;
};

export type ChromeMap = Record<string, string>;
