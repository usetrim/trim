import type { DedicatedListToolbarChrome } from "@/components/skeletons/page-skeletons";
import {
  dedicatedListSkeletonChrome,
  type DedicatedListSkeletonChrome,
} from "@/lib/skeleton-chrome";
import type { AuthProvidersSiteChrome } from "@/types/auth";

type ColumnSpec = { label: string; width: string };

export type DedicatedListSkeletonProps = {
  title: string;
  eyebrow: string;
  rows: number;
  toolbar: "dateRange" | "sync" | "statusSearch";
  toolbarChrome: DedicatedListToolbarChrome;
  columns: ColumnSpec[];
};

/**
 * Single source of truth for Traces / Receipts / Enterprise list skeletons.
 * Boot + page clients must use these builders so refresh cannot flash a different shape.
 */
export function tracesListSkeletonProps(
  site: AuthProvidersSiteChrome | null | undefined,
  rows: number,
  overrides?: {
    title?: string;
    eyebrow?: string;
    colWhen?: string;
    colModel?: string;
    colMode?: string;
    colTokens?: string;
    colLatency?: string;
    colView?: string;
    dateRange?: string;
    dateRangeDesc?: string;
    search?: string;
    searchDesc?: string;
  },
): DedicatedListSkeletonProps {
  const chrome = dedicatedListSkeletonChrome(site);
  return {
    title: overrides?.title || chrome?.traces_title || "",
    eyebrow: overrides?.eyebrow || chrome?.page_eyebrow || "",
    rows,
    toolbar: "dateRange",
    toolbarChrome: {
      primaryLabel: overrides?.dateRange || chrome?.traces_date_range || "",
      primaryDescription: overrides?.dateRangeDesc || chrome?.traces_date_range_desc || "",
      searchLabel: overrides?.search || chrome?.traces_search || "",
      searchDescription: overrides?.searchDesc || chrome?.traces_search_desc || "",
    },
    columns: tracesListColumns(chrome, overrides),
  };
}

export function receiptsListSkeletonProps(
  site: AuthProvidersSiteChrome | null | undefined,
  rows: number,
  overrides?: {
    title?: string;
    eyebrow?: string;
    colDate?: string;
    colInvoice?: string;
    colStatus?: string;
    colTotal?: string;
    colView?: string;
    syncLabel?: string;
    search?: string;
    searchDesc?: string;
  },
): DedicatedListSkeletonProps {
  const chrome = dedicatedListSkeletonChrome(site);
  return {
    title: overrides?.title || chrome?.receipts_title || "",
    eyebrow: overrides?.eyebrow || chrome?.page_eyebrow || "",
    rows,
    toolbar: "sync",
    toolbarChrome: {
      primaryLabel: overrides?.syncLabel || chrome?.receipts_sync_label || "",
      searchLabel: overrides?.search || chrome?.receipts_search || "",
      searchDescription: overrides?.searchDesc || chrome?.receipts_search_desc || "",
    },
    columns: receiptsListColumns(chrome, overrides),
  };
}

export function enterpriseListSkeletonProps(
  site: AuthProvidersSiteChrome | null | undefined,
  rows: number,
  overrides?: {
    title?: string;
    eyebrow?: string;
    colCompany?: string;
    colStatus?: string;
    colRequested?: string;
    colOffered?: string;
    colCreated?: string;
    filterStatus?: string;
    filterStatusDesc?: string;
    search?: string;
    searchDesc?: string;
  },
): DedicatedListSkeletonProps {
  const chrome = dedicatedListSkeletonChrome(site);
  return {
    title: overrides?.title || chrome?.enterprise_title || "",
    eyebrow: overrides?.eyebrow || chrome?.page_eyebrow || "",
    rows,
    toolbar: "statusSearch",
    toolbarChrome: {
      primaryLabel: overrides?.filterStatus || chrome?.enterprise_filter_status || "",
      primaryDescription:
        overrides?.filterStatusDesc || chrome?.enterprise_filter_status_desc || "",
      searchLabel: overrides?.search || chrome?.enterprise_search || "",
      searchDescription: overrides?.searchDesc || chrome?.enterprise_search_desc || "",
    },
    columns: enterpriseListColumns(chrome, overrides),
  };
}

function tracesListColumns(
  chrome: DedicatedListSkeletonChrome | undefined,
  overrides?: {
    colWhen?: string;
    colModel?: string;
    colMode?: string;
    colTokens?: string;
    colLatency?: string;
    colView?: string;
  },
): ColumnSpec[] {
  return [
    { label: "", width: "h-4 w-4" },
    { label: overrides?.colWhen || chrome?.traces_col_when || "", width: "h-4 w-28" },
    { label: overrides?.colModel || chrome?.traces_col_model || "", width: "h-4 w-24" },
    { label: overrides?.colMode || chrome?.traces_col_mode || "", width: "h-4 w-16" },
    { label: overrides?.colTokens || chrome?.traces_col_tokens || "", width: "h-4 w-20" },
    { label: overrides?.colLatency || chrome?.traces_col_latency || "", width: "h-4 w-16" },
    { label: overrides?.colView || chrome?.traces_col_view || "", width: "h-4 w-10" },
    { label: "", width: "h-8 w-8" },
  ];
}

function receiptsListColumns(
  chrome: DedicatedListSkeletonChrome | undefined,
  overrides?: {
    colDate?: string;
    colInvoice?: string;
    colStatus?: string;
    colTotal?: string;
    colView?: string;
  },
): ColumnSpec[] {
  return [
    { label: "", width: "h-4 w-4" },
    { label: overrides?.colDate || chrome?.receipts_col_date || "", width: "h-4 w-20" },
    { label: overrides?.colInvoice || chrome?.receipts_col_invoice || "", width: "h-4 w-24" },
    { label: overrides?.colStatus || chrome?.receipts_col_status || "", width: "h-4 w-16" },
    { label: overrides?.colTotal || chrome?.receipts_col_total || "", width: "h-4 w-16" },
    { label: overrides?.colView || chrome?.receipts_col_view || "", width: "h-4 w-10" },
    { label: "", width: "h-8 w-8" },
  ];
}

function enterpriseListColumns(
  chrome: DedicatedListSkeletonChrome | undefined,
  overrides?: {
    colCompany?: string;
    colStatus?: string;
    colRequested?: string;
    colOffered?: string;
    colCreated?: string;
  },
): ColumnSpec[] {
  return [
    { label: "", width: "h-4 w-4" },
    { label: overrides?.colCompany || chrome?.enterprise_col_company || "", width: "h-4 w-32" },
    { label: overrides?.colStatus || chrome?.enterprise_col_status || "", width: "h-4 w-20" },
    {
      label: overrides?.colRequested || chrome?.enterprise_col_requested || "",
      width: "h-4 w-16",
    },
    { label: overrides?.colOffered || chrome?.enterprise_col_offered || "", width: "h-4 w-16" },
    { label: overrides?.colCreated || chrome?.enterprise_col_created || "", width: "h-4 w-24" },
    { label: "", width: "h-4 w-10" },
    { label: "", width: "h-8 w-16" },
    { label: "", width: "h-8 w-8" },
  ];
}
