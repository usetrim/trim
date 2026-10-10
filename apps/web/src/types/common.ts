export type PaginationMeta = {
  skip: number;
  limit: number;
  total: number;
  has_more: boolean;
  next_skip: number | null;
  prev_skip: number | null;
  first_skip?: number | null;
  last_skip?: number | null;
  page: number;
  total_pages: number;
  prev_action_label: string;
  next_action_label: string;
  first_action_label?: string;
  last_action_label?: string;
  skip_to_label?: string;
  skip_to_action_label?: string;
  skip_to_pending_label?: string;
  skip_to_invalid?: string;
  /** Backend OFFSET per 1-based page; page_skips[page-1] for skip-to. */
  page_skips?: number[];
  page_summary: string;
};
