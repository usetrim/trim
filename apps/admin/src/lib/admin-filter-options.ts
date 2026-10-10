/** Shared filter option builders from backend chrome maps (no client invent). */

export type ChromeMap = Record<string, string | undefined>;

export type FilterOption = { value: string; label: string };

const ALL = "__all__";

export function filterAllSentinel(): string {
  return ALL;
}

export function isFilterAll(value: string): boolean {
  return !value || value === ALL;
}

/** Account status dropdown options from ADMIN_STATUS_* chrome. */
export function accountStatusFilterOptions(chrome: ChromeMap): FilterOption[] {
  return (
    [
      ["active", chrome.ADMIN_STATUS_ACTIVE],
      ["suspended", chrome.ADMIN_STATUS_SUSPENDED],
      ["banned", chrome.ADMIN_STATUS_BANNED],
      ["pending_delete", chrome.ADMIN_STATUS_PENDING_DELETE],
      ["shadowbanned", chrome.ADMIN_STATUS_SHADOWBANNED],
    ] as const
  ).flatMap(([value, label]) => {
    const trimmed = label?.trim() || "";
    return trimmed ? [{ value, label: trimmed }] : [];
  });
}

/** Subscription status dropdown options from STATUS_* chrome. */
export function subscriptionStatusFilterOptions(chrome: ChromeMap): FilterOption[] {
  return (
    [
      ["active", chrome.STATUS_ACTIVE],
      ["trialing", chrome.STATUS_TRIALING],
      ["past_due", chrome.STATUS_PAST_DUE],
      ["paused", chrome.STATUS_PAUSED],
      ["canceled", chrome.STATUS_CANCELED],
      ["expired", chrome.STATUS_EXPIRED],
    ] as const
  ).flatMap(([value, label]) => {
    const trimmed = label?.trim() || "";
    return trimmed ? [{ value, label: trimmed }] : [];
  });
}

/** Enterprise inquiry workflow status options from ADMIN_ENTERPRISE_STATUS_* chrome. */
export function enterpriseInquiryStatusFilterOptions(chrome: ChromeMap): FilterOption[] {
  return (
    [
      ["new", chrome.ADMIN_ENTERPRISE_STATUS_NEW],
      ["contacted", chrome.ADMIN_ENTERPRISE_STATUS_CONTACTED],
      ["offered", chrome.ADMIN_ENTERPRISE_STATUS_OFFERED],
      ["closed", chrome.ADMIN_ENTERPRISE_STATUS_CLOSED],
      ["activated", chrome.ADMIN_ENTERPRISE_STATUS_ACTIVATED],
    ] as const
  ).flatMap(([value, label]) => {
    const trimmed = label?.trim() || "";
    return trimmed ? [{ value, label: trimmed }] : [];
  });
}

/** Billing interval dropdown options from BILLING_INTERVAL_* chrome. */
export function billingIntervalFilterOptions(chrome: ChromeMap): FilterOption[] {
  return (
    [
      ["month", chrome.BILLING_INTERVAL_MONTH],
      ["year", chrome.BILLING_INTERVAL_YEAR],
    ] as const
  ).flatMap(([value, label]) => {
    const trimmed = label?.trim() || "";
    return trimmed ? [{ value, label: trimmed }] : [];
  });
}
