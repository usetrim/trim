import type { QueryClient } from "@tanstack/react-query";

/** Stable query-key roots. Prefer these in useQuery + invalidateQueries. */
export const adminQk = {
  me: ["admin-me"] as const,
  totp: ["admin-totp"] as const,
  webauthn: ["admin-webauthn"] as const,
  dashboard: ["admin-dashboard"] as const,
  users: ["admin-users"] as const,
  userCountries: ["admin-user-countries"] as const,
  user: (id: string) => ["admin-user", id] as const,
  plans: ["admin-plans"] as const,
  billingSettings: ["admin-billing-settings"] as const,
  subscriptions: ["admin-subscriptions"] as const,
  receipts: ["admin-receipts"] as const,
  revenue: ["admin-revenue"] as const,
  disputes: ["admin-disputes"] as const,
  enterprise: ["admin-enterprise"] as const,
  segmentsIndividuals: ["admin-segments-individuals"] as const,
  segmentsTeams: ["admin-segments-teams"] as const,
  segmentsEnterprises: ["admin-segments-enterprises"] as const,
  denylistEmail: ["admin-denylist-email"] as const,
  denylistIp: ["admin-denylist-ip"] as const,
  denylistAsn: ["admin-denylist-asn"] as const,
  chromeMessages: ["admin-chrome-messages"] as const,
  chromeNav: ["admin-chrome-nav"] as const,
  emailTemplates: ["admin-email-templates"] as const,
  legal: ["admin-legal"] as const,
  observability: ["admin-observability"] as const,
  observabilityWebhooks: ["admin-observability-webhooks"] as const,
  observabilityWebhook: (id: string) => ["admin-observability-webhook", id] as const,
  heatmap: ["admin-heatmap"] as const,
  audit: ["admin-audit"] as const,
  auditFilterOptions: ["admin-audit-filter-options"] as const,
  breakGlass: ["admin-break-glass"] as const,
  compliance: ["admin-compliance"] as const,
  accessReview: ["admin-access-review"] as const,
  distribution: ["admin-distribution"] as const,
  authSettings: ["admin-auth-settings"] as const,
  oauthScopes: ["admin-oauth-scopes"] as const,
  publicAuthProviders: ["public-auth-providers"] as const,
  product: ["admin-product"] as const,
  creditGrants: (userId = "") => ["admin-credit-grants", userId] as const,
  topupLedger: (userId = "") => ["admin-topup-ledger", userId] as const,
  roles: ["admin-roles"] as const,
  roleOptions: ["admin-role-options"] as const,
  permissions: ["admin-permissions"] as const,
  admins: ["admin-admins"] as const,
  notifications: ["admin-notifications"] as const,
  notificationsUnread: ["admin-notifications-unread"] as const,
} as const;

const refetchActive = { refetchType: "active" as const };

/** User detail + lists + KPIs that change when an operator mutates a user. */
export async function invalidateAfterUserMutation(qc: QueryClient, userId: string) {
  await Promise.all([
    qc.invalidateQueries({ queryKey: adminQk.user(userId), ...refetchActive }),
    qc.invalidateQueries({ queryKey: adminQk.users, ...refetchActive }),
    qc.invalidateQueries({ queryKey: adminQk.dashboard, ...refetchActive }),
    qc.invalidateQueries({
      queryKey: adminQk.segmentsIndividuals,
      ...refetchActive,
    }),
    qc.invalidateQueries({
      queryKey: adminQk.segmentsTeams,
      ...refetchActive,
    }),
    qc.invalidateQueries({
      queryKey: adminQk.segmentsEnterprises,
      ...refetchActive,
    }),
    qc.invalidateQueries({
      queryKey: adminQk.subscriptions,
      ...refetchActive,
    }),
    qc.invalidateQueries({ queryKey: adminQk.receipts, ...refetchActive }),
    qc.invalidateQueries({ queryKey: adminQk.enterprise, ...refetchActive }),
    qc.invalidateQueries({
      queryKey: adminQk.creditGrants(userId),
      ...refetchActive,
    }),
    qc.invalidateQueries({
      queryKey: ["admin-credit-grants"],
      ...refetchActive,
    }),
    qc.invalidateQueries({
      queryKey: adminQk.topupLedger(userId),
      ...refetchActive,
    }),
    qc.invalidateQueries({
      queryKey: ["admin-topup-ledger"],
      ...refetchActive,
    }),
    qc.invalidateQueries({ queryKey: adminQk.audit, ...refetchActive }),
  ]);
}

/** Plan / billing settings changes that ripple into lists and public chrome. */
export async function invalidateAfterBillingMutation(qc: QueryClient) {
  await Promise.all([
    qc.invalidateQueries({ queryKey: adminQk.plans, ...refetchActive }),
    qc.invalidateQueries({
      queryKey: adminQk.billingSettings,
      ...refetchActive,
    }),
    qc.invalidateQueries({
      queryKey: adminQk.subscriptions,
      ...refetchActive,
    }),
    qc.invalidateQueries({ queryKey: adminQk.receipts, ...refetchActive }),
    qc.invalidateQueries({ queryKey: adminQk.disputes, ...refetchActive }),
    qc.invalidateQueries({
      queryKey: adminQk.publicAuthProviders,
      ...refetchActive,
    }),
    qc.invalidateQueries({ queryKey: adminQk.users, ...refetchActive }),
    qc.invalidateQueries({
      queryKey: adminQk.segmentsIndividuals,
      ...refetchActive,
    }),
    qc.invalidateQueries({
      queryKey: adminQk.segmentsTeams,
      ...refetchActive,
    }),
    qc.invalidateQueries({
      queryKey: adminQk.segmentsEnterprises,
      ...refetchActive,
    }),
    qc.invalidateQueries({
      queryKey: ["admin-credit-grants"],
      ...refetchActive,
    }),
    qc.invalidateQueries({
      queryKey: ["admin-topup-ledger"],
      ...refetchActive,
    }),
    qc.invalidateQueries({ queryKey: adminQk.dashboard, ...refetchActive }),
    qc.invalidateQueries({ queryKey: adminQk.audit, ...refetchActive }),
  ]);
}

/** Staff / permission changes that affect shell me + access review. */
export async function invalidateAfterStaffMutation(qc: QueryClient) {
  await Promise.all([
    qc.invalidateQueries({ queryKey: adminQk.me, ...refetchActive }),
    qc.invalidateQueries({ queryKey: adminQk.admins, ...refetchActive }),
    qc.invalidateQueries({ queryKey: adminQk.roles, ...refetchActive }),
    qc.invalidateQueries({ queryKey: adminQk.roleOptions, ...refetchActive }),
    qc.invalidateQueries({ queryKey: adminQk.permissions, ...refetchActive }),
    qc.invalidateQueries({ queryKey: adminQk.accessReview, ...refetchActive }),
    qc.invalidateQueries({ queryKey: adminQk.breakGlass, ...refetchActive }),
    qc.invalidateQueries({ queryKey: adminQk.dashboard, ...refetchActive }),
    qc.invalidateQueries({ queryKey: adminQk.audit, ...refetchActive }),
  ]);
}
