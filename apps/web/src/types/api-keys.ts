import type { PaginationMeta } from "./common";

export type ApiKeyItem = {
  id: string;
  key_prefix: string;
  revoked: boolean;
  created_at: string;
  expires_at?: string | null;
  device_count?: number;
  device_bound?: boolean;
};

export type ApiKeysListResponse = {
  items: ApiKeyItem[];
  meta: PaginationMeta;
  intro_message?: string;
  empty_message?: string;
  status_active_label?: string;
  status_revoked_label?: string;
  created_prefix_label?: string;
  issue_action_label: string;
  issue_pending_label: string;
  revoke_action_label: string;
  revoke_pending_label: string;
  revoke_confirm_message?: string;
  refresh_action_label: string;
  refresh_pending_label: string;
  key_prefix_ellipsis?: string;
  table_select_all?: string;
  table_select_row?: string;
  table_selected_fmt?: string;
  table_row_actions?: string;
  table_bulk_revoke?: string;
  table_clear_selection?: string;
  device_hint?: string;
  device_register_label?: string;
  device_register_pending?: string;
  device_hw_label?: string;
  device_agent_label?: string;
  device_agent_options?: Array<{ id: string; label: string }>;
  device_bound_label?: string;
  device_unbound_label?: string;
  device_count_fmt?: string;
  ci_agent_hint?: string;
  search_placeholder?: string;
  search_description?: string;
  q?: string;
};

export type ApiKeyDeviceItem = {
  hardware_uuid: string;
  agent_id?: string;
  first_seen_at: string;
  last_seen_at: string;
};

export type ApiKeyDevicesResponse = {
  items: ApiKeyDeviceItem[];
  device_hint?: string;
  device_hw_label?: string;
  device_agent_label?: string;
  device_register_label?: string;
  device_register_pending?: string;
  device_remove_label?: string;
  device_remove_pending?: string;
  ci_agent_hint?: string;
};

export type CreateApiKeyResponse = {
  api_key: string;
  id: string;
  key_prefix: string;
  created_at: string;
  device_bound: boolean;
  action_label: string;
  pending_label: string;
  copy_action_label: string;
  copy_pending_label: string;
  copied_action_label: string;
  fresh_key_hint?: string;
  device_hint?: string;
  message?: string;
};
