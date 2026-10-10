export type QuotaResponse = {
  plan_tier: string;
  monthly_credit_limit: number;
  monthly_credit_used: number;
  purchased_topup_credits: number;
  remaining: number;
  unlimited?: boolean;
  unlimited_label?: string;
  code?: string;
  error?: string;
  tier_upgrade?: string;
  exhausted_title?: string;
  exhausted_body?: string;
  upgrade_action_label?: string;
};

export type PreferencesResponse = {
  compression_tier: string;
  deep_engine: string;
  deep_target_token: number;
  auto_start_with_ide: boolean;
  fast_tier_id: string;
  deep_tier_id: string;
  deep_target_token_min: number;
  deep_target_token_max: number;
  live_deep_min_input_tokens?: number;
  live_deep_oom_policy?: string;
  live_deep_warmup_on_start?: boolean;
  live_deep_skip_on_stream?: boolean;
  deep_v1_model?: string;
  deep_v2_model?: string;
  deep_long_model?: string;
  deep_v2_force_tokens?: string[];
  live_deep_min_hint?: string;
  live_deep_stream_hint?: string;
  provider_adapters?: Array<{
    id: string;
    enabled: boolean;
    sort_order: number;
    dialect: string;
    door_label: string;
    match_model_prefixes: string[];
    model_aliases: Record<string, string>;
    upstream_kind: string;
    upstream_path: string;
    upstream_base_url?: string;
    auth_mode: string;
    anthropic_version?: string;
    default_max_tokens?: number;
    require_alias: boolean;
  }>;
  openai_model_aliases?: Array<{
    client_model: string;
    upstream_model: string;
    upstream_host_contains: string;
    enabled: boolean;
  }>;
  provider_adapters_synced?: boolean;
  note: string;
  saved_message: string;
  engine_options: Array<{ id: string; label: string; hint?: string }>;
  save_action_label: string;
  save_pending_label: string;
  page_title: string;
  page_description: string;
  deep_hint: string;
  engine_hint?: string;
  target_hint?: string;
  auto_start_title?: string;
  auto_start_label?: string;
  auto_start_hint?: string;
  auto_start_note?: string;
  local_agent_title?: string;
  local_agent_status?: "online" | "offline" | "unknown";
  local_agent_status_label?: string;
  local_agent_hint?: string;
  back_action_label: string;
  back_href?: string;
  tier_title: string;
  deep_label: string;
  engine_label: string;
  target_label: string;
  cli_title: string;
  cli_help_lines: string[];
  keys_title: string;
  auth_provider: string;
  auth_provider_display: string;
  account_title: string;
  auth_provider_prefix: string;
  linked_identities: Array<{
    identity_id: string;
    provider: string;
    provider_display: string;
    email?: string;
    last_sign_in_at?: string;
    is_primary: boolean;
    is_last_used: boolean;
  }>;
  linkable_providers: Array<{
    id: string;
    display_name: string;
    action_label: string;
    pending_label: string;
  }>;
  link_hint: string;
  unlink_action_label: string;
  unlink_pending_label: string;
  unlink_confirm_message: string;
  unlink_last_blocked_message: string;
  unlink_failed_message: string;
  unlink_done_message?: string;
  identity_primary_label: string;
  identity_last_used_label: string;
  can_unlink: boolean;
  table_row_actions?: string;
};

export type PatchPreferencesBody = {
  compression_tier?: string;
  deep_engine?: string;
  deep_target_token?: number;
  auto_start_with_ide?: boolean;
};

export type AvatarSyncResponse = {
  status: string;
  avatar_url?: string;
  message?: string;
};

export type DeleteAccountResponse = {
  status: string;
  message: string;
};
