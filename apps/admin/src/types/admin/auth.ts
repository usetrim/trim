export type AuthProviderCatalogItem = {
  id: string;
  display_name: string;
};

export type AuthSettings = {
  allowed_providers?: string[];
  catalog?: AuthProviderCatalogItem[];
  [key: string]: unknown;
};

export type AuthProvidersPayload = {
  items?: {
    id: string;
    display_name: string;
    action_label: string;
    pending_label: string;
  }[];
  login_title?: string;
  login_default_next?: string;
  admin_login_title?: string;
  admin_login_default_next?: string;
  admin_login_default_next_missing?: string;
  login_default_next_missing?: string;
  login_provider_disabled_message?: string;
  login_failed_message?: string;
  login_env_app_url_missing?: string;
  login_oauth_exchange_failed?: string;
  login_missing_supabase_env?: string;
  login_auth_providers_unavailable?: string;
  login_oauth_email_missing?: string;
  login_oauth_email_denied?: string;
  providers_empty_message?: string;
  default_page_size?: number;
  dialog_cancel?: string;
  dialog_close_label?: string;
  admin_forbidden_page?: string;
  admin_credentials_revoked?: string;
  admin_bootstrap_required?: string;
  site?: Record<string, string>;
  /** Per-provider OAuth chrome keys from site_messages. */
  [key: string]: unknown;
};

export type OAuthScopesPayload = {
  scopes: Record<string, string>;
};

export type StepUpResponse = {
  step_up_token: string;
  expires_in_sec: number;
};

export type TOTPStatus = {
  enrolled: boolean;
  status_label?: string;
  key_configured?: boolean;
};

export type TOTPBeginResponse = {
  secret: string;
  otpauth_url: string;
};

export type WebAuthnCredentialItem = {
  id: string;
  friendly_name?: string;
  created_at?: string;
  last_used_at?: string;
};

export type WebAuthnStatus = {
  enrolled: boolean;
  status_label?: string;
  rp_configured?: boolean;
  items?: WebAuthnCredentialItem[];
};

/** PublicKeyCredentialCreationOptions / RequestOptions JSON from go-webauthn. */
export type WebAuthnOptionsPayload = Record<string, unknown>;
