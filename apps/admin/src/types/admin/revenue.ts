export type RevenueRangeOption = {
  id: string;
  label: string;
};

export type RevenueKpiItem = {
  id: string;
  label: string;
  value: string | number;
};

export type RevenueSeriesMeta = {
  id: string;
  label: string;
};

export type RevenueSeriesRow = {
  day: string;
  day_label: string;
  [key: string]: string | number;
};

export type RevenueByPlanRow = {
  plan_id: string;
  plan_label: string;
  plan_kind: string;
  revenue_cents: number;
  revenue_label: string;
  receipt_count: number;
  currency_code: string;
};

export type RevenueResponse = {
  range: string;
  from: string;
  to: string;
  range_options: RevenueRangeOption[];
  currency_code: string;
  kpi_items: RevenueKpiItem[];
  series_meta: RevenueSeriesMeta[];
  series: RevenueSeriesRow[];
  by_plan: RevenueByPlanRow[];
  date_range_months?: number;
  chrome: Record<string, string>;
};
