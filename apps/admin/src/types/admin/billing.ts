export type PatchPlanBody = {
  display_name?: string;
  description?: string;
  plan_kind?: string;
  plan_rank?: number;
  price_monthly_cents?: number;
  price_yearly_cents?: number;
  credits_monthly?: number;
  per_seat?: boolean;
  unlimited?: boolean;
  popular?: boolean;
  is_public?: boolean;
  is_active?: boolean;
  sort_order?: number;
  features?: unknown;
  /** When true (default on server), push amounts to Paddle and store pri_/pro_ ids. */
  sync_to_paddle?: boolean;
};

export type PlanCatalogItem = {
  id?: string;
  display_name?: string;
  description?: string;
  plan_kind?: string;
  plan_rank?: number | null;
  sort_order?: number | null;
  features?: unknown;
  price_monthly_cents?: number | null;
  price_yearly_cents?: number | null;
  credits_monthly?: number | null;
  per_seat?: boolean;
  unlimited?: boolean;
  popular?: boolean;
  is_public?: boolean;
  is_active?: boolean;
  paddle_product_id?: string;
  paddle_price_id_monthly?: string;
  paddle_price_id_yearly?: string;
  paddle_price_id_topup?: string;
  [key: string]: unknown;
};
