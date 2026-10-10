import type {
  DashboardSkeletonChrome,
  LandingPageSkeletonChrome,
  ReceiptSkeletonChrome,
  SettingsSkeletonChrome,
  TeamSkeletonChrome,
} from "@/components/skeletons/page-skeletons";
import type { AuthProvidersSiteChrome } from "@/types/auth";

export type DedicatedListSkeletonChrome = {
  page_eyebrow?: string;
  traces_title?: string;
  receipts_title?: string;
  enterprise_title?: string;
  traces_col_when?: string;
  traces_col_model?: string;
  traces_col_mode?: string;
  traces_col_tokens?: string;
  traces_col_latency?: string;
  traces_col_view?: string;
  traces_date_range?: string;
  traces_date_range_desc?: string;
  traces_search?: string;
  traces_search_desc?: string;
  receipts_col_date?: string;
  receipts_col_invoice?: string;
  receipts_col_status?: string;
  receipts_col_total?: string;
  receipts_col_view?: string;
  receipts_search?: string;
  receipts_search_desc?: string;
  receipts_sync_label?: string;
  enterprise_col_company?: string;
  enterprise_col_status?: string;
  enterprise_col_requested?: string;
  enterprise_col_offered?: string;
  enterprise_col_created?: string;
  enterprise_filter_status?: string;
  enterprise_filter_status_desc?: string;
  enterprise_search?: string;
  enterprise_search_desc?: string;
};

/** Map public auth-providers site chrome into layout-faithful page skeletons. */
export function dashboardSkeletonChrome(
  site?: AuthProvidersSiteChrome | null,
  header?: {
    primary?: boolean;
    topup?: boolean;
    portal?: boolean;
    team?: boolean;
    settings?: boolean;
    notifications?: boolean;
  } | null,
): DashboardSkeletonChrome | undefined {
  if (!site && !header) return undefined;
  const signOut = Boolean(
    site?.sign_out_action_label?.trim() && site?.sign_out_pending_label?.trim(),
  );
  return {
    metric_plan_label: site?.skeleton_metric_plan_label,
    metric_credits_label: site?.skeleton_metric_credits_label,
    metric_remaining_label: site?.skeleton_metric_remaining_label,
    metric_tokens_saved_label: site?.skeleton_metric_tokens_saved_label,
    metric_tab_label: site?.skeleton_metric_tab_label,
    metric_lines_added_label: site?.skeleton_metric_lines_added_label,
    metric_lines_deleted_label: site?.skeleton_metric_lines_deleted_label,
    chart_token_series_title: site?.skeleton_chart_token_series_title,
    chart_models_title: site?.skeleton_chart_models_title,
    chart_outcomes_title: site?.skeleton_chart_outcomes_title,
    chart_modes_title: site?.skeleton_chart_modes_title,
    traces_title: site?.skeleton_traces_title,
    receipts_title: site?.skeleton_receipts_title,
    traces_col_when: site?.skeleton_traces_col_when,
    traces_col_model: site?.skeleton_traces_col_model,
    traces_col_mode: site?.skeleton_traces_col_mode,
    traces_col_tokens: site?.skeleton_traces_col_tokens,
    traces_col_latency: site?.skeleton_traces_col_latency,
    receipts_col_date: site?.skeleton_receipts_col_date,
    receipts_col_invoice: site?.skeleton_receipts_col_invoice,
    receipts_col_status: site?.skeleton_receipts_col_status,
    receipts_col_total: site?.skeleton_receipts_col_total,
    receipts_col_view: site?.skeleton_receipts_col_view,
    // Default primary+topup on while subscription chrome is unknown (matches free/trial home).
    header_primary: header?.primary !== false,
    header_topup: header?.topup !== false,
    header_sign_out: signOut,
    header_portal: Boolean(header?.portal),
    header_team: Boolean(header?.team),
    header_settings: Boolean(header?.settings),
    header_notifications: header?.notifications !== false,
  };
}

export function dedicatedListSkeletonChrome(
  site?: AuthProvidersSiteChrome | null,
): DedicatedListSkeletonChrome | undefined {
  if (!site) return undefined;
  return {
    page_eyebrow: site.skeleton_page_eyebrow,
    traces_title: site.skeleton_traces_title,
    receipts_title: site.skeleton_receipts_title,
    enterprise_title: site.skeleton_enterprise_title,
    traces_col_when: site.skeleton_traces_col_when,
    traces_col_model: site.skeleton_traces_col_model,
    traces_col_mode: site.skeleton_traces_col_mode,
    traces_col_tokens: site.skeleton_traces_col_tokens,
    traces_col_latency: site.skeleton_traces_col_latency,
    traces_col_view: site.skeleton_traces_col_view,
    traces_date_range: site.skeleton_traces_date_range,
    traces_date_range_desc: site.skeleton_traces_date_range_desc,
    traces_search: site.skeleton_traces_search,
    traces_search_desc: site.skeleton_traces_search_desc,
    receipts_col_date: site.skeleton_receipts_col_date,
    receipts_col_invoice: site.skeleton_receipts_col_invoice,
    receipts_col_status: site.skeleton_receipts_col_status,
    receipts_col_total: site.skeleton_receipts_col_total,
    receipts_col_view: site.skeleton_receipts_col_view,
    receipts_search: site.skeleton_receipts_search,
    receipts_search_desc: site.skeleton_receipts_search_desc,
    receipts_sync_label: site.skeleton_receipts_sync_label,
    enterprise_col_company: site.skeleton_enterprise_col_company,
    enterprise_col_status: site.skeleton_enterprise_col_status,
    enterprise_col_requested: site.skeleton_enterprise_col_requested,
    enterprise_col_offered: site.skeleton_enterprise_col_offered,
    enterprise_col_created: site.skeleton_enterprise_col_created,
    enterprise_filter_status: site.skeleton_enterprise_filter_status,
    enterprise_filter_status_desc: site.skeleton_enterprise_filter_status_desc,
    enterprise_search: site.skeleton_enterprise_search,
    enterprise_search_desc: site.skeleton_enterprise_search_desc,
  };
}

export function teamSkeletonChrome(
  site?: AuthProvidersSiteChrome | null,
): TeamSkeletonChrome | undefined {
  if (!site) return undefined;
  return {
    workspaces_title: site.skeleton_team_workspaces_title,
    members_title: site.skeleton_team_members_title,
    invites_title: site.skeleton_team_invites_title,
  };
}

export function receiptSkeletonChrome(
  site?: AuthProvidersSiteChrome | null,
): ReceiptSkeletonChrome | undefined {
  if (!site) return undefined;
  return {
    col_description: site.skeleton_receipt_col_description,
    col_product: site.skeleton_receipt_col_description,
    col_sku: site.skeleton_receipt_col_sku,
    col_qty: site.skeleton_receipt_col_qty,
    col_unit: site.skeleton_receipt_col_unit,
    col_amount: site.skeleton_receipt_col_amount,
    section_bill_to: site.skeleton_receipt_section_bill_to,
    section_period: site.skeleton_receipt_section_period,
    section_amount: site.skeleton_receipt_section_amount,
    section_status: site.skeleton_receipt_section_status,
    section_payment: site.skeleton_receipt_section_payment,
    footer: site.skeleton_receipt_footer,
  };
}

export function landingSkeletonChrome(
  site?: AuthProvidersSiteChrome | null,
  extras?: { login_title?: string } | null,
): LandingPageSkeletonChrome | undefined {
  if (!site && !extras?.login_title) return undefined;
  return {
    brand: site?.brand,
    nav_dashboard: site?.nav_dashboard,
    nav_sign_in: site?.nav_sign_in,
    login_title: extras?.login_title,
    nav_how: site?.nav_how,
    nav_pricing: site?.nav_pricing,
    nav_contact: site?.nav_contact,
    contact_page_heading: site?.contact_page_heading,
    contact_page_body: site?.contact_page_body,
    eyebrow: site?.eyebrow,
    headline: site?.headline,
    tagline: site?.tagline,
    cta_start: site?.cta_start,
    cta_source: site?.cta_source,
    cta_demo: site?.cta_demo,
    footer_privacy: site?.footer_privacy,
    footer_terms: site?.footer_terms,
    footer_github: site?.footer_github,
  };
}

export function settingsSkeletonChrome(
  site?: AuthProvidersSiteChrome | null,
): SettingsSkeletonChrome | undefined {
  if (!site) return undefined;
  return {
    page_title: site.skeleton_settings_page_title,
    page_description: site.skeleton_settings_page_description,
    tier_title: site.skeleton_settings_tier_title,
    deep_label: site.skeleton_settings_deep_label,
    cli_title: site.skeleton_settings_cli_title,
    keys_title: site.skeleton_settings_keys_title,
    save_label: site.skeleton_settings_save_label,
    back_label: site.skeleton_settings_back_label,
    account_title: site.skeleton_settings_account_title,
    auth_provider_prefix: site.skeleton_settings_auth_provider_prefix,
    link_hint: site.skeleton_settings_link_hint,
    unlink_label: site.skeleton_settings_unlink_label,
    row_actions_label: site.skeleton_settings_row_actions_label,
  };
}
