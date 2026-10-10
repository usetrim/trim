export type AdminUserListItem = {
  id: string;
  email: string;
  full_name?: string;
  account_status?: string;
  account_status_label?: string;
  last_login_country?: string;
  bill_to_country?: string;
  plan_tier?: string;
  auth_provider?: string;
  created_at?: string;
};

export type AdminUsersFilters = {
  q?: string;
  status?: string;
  country?: string;
  plan?: string;
  provider?: string;
  created_from?: string;
  created_to?: string;
};

export type AdminFilterOption = {
  value: string;
  label: string;
};

export type UserStatusBody = {
  status: string;
  reason: string;
};

export type UserQuotaBody = {
  monthly_credit_limit?: number;
  purchased_topup_credits?: number;
  reason: string;
};

export type CreditGrantBody = {
  user_id: string;
  credits: number;
  reason: string;
};
