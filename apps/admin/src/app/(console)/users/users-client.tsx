"use client";

import { AdminFilterBar } from "@/components/admin/admin-filter-bar";
import { DataTableSkeleton } from "@/components/skeletons/page-skeletons";
import { Card, CardContent } from "@/components/ui/card";
import { type ColumnDef, DataTable } from "@/components/ui/data-table";
import { DataTableRowActions } from "@/components/ui/data-table-row-actions";
import type { DateRangeValue } from "@/components/ui/date-range-picker";
import { useAdminAuthSettings } from "@/hooks/queries/auth";
import { useAdminBillingSettings, useAdminPlans } from "@/hooks/queries/billing";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminUserCountries, useAdminUsers } from "@/hooks/queries/users";
import { useAdminToken } from "@/hooks/use-admin-token";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { accountStatusFilterOptions } from "@/lib/admin-filter-options";
import type { AdminUserListItem } from "@/types/admin";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";

export function UsersClient({ initialToken }: { initialToken?: string }) {
  const token = useAdminToken(initialToken);
  const router = useRouter();
  const pageSize = useDefaultPageSize();
  const [skip, setSkip] = useState(0);
  const [qInput, setQInput] = useState("");
  const [status, setStatus] = useState("");
  const [plan, setPlan] = useState("");
  const [country, setCountry] = useState("");
  const [provider, setProvider] = useState("");
  const [createdRange, setCreatedRange] = useState<DateRangeValue>({ from: null, to: null });
  const q = useDebouncedValue(qInput.trim(), 250);
  const chrome = useAdminNavChrome(token);
  const billing = useAdminBillingSettings(token);
  const authSettings = useAdminAuthSettings(token);
  const plans = useAdminPlans(token, 0, pageSize > 0 ? Math.max(pageSize, 50) : 0);
  const countries = useAdminUserCountries(token, "users");
  const users = useAdminUsers(token, skip, pageSize, {
    q,
    status,
    plan,
    country,
    provider,
    created_from: createdRange.from || "",
    created_to: createdRange.to || "",
  });

  const labels = chrome.data || {};
  const title = labels.ADMIN_NAV_USERS?.trim() || "";
  const searchPlaceholder = labels.ADMIN_USERS_SEARCH?.trim() || "";
  const searchDesc = labels.ADMIN_USERS_SEARCH_DESC?.trim() || "";
  const filterStatus = labels.ADMIN_FILTER_STATUS?.trim() || "";
  const filterStatusDesc = labels.ADMIN_FILTER_STATUS_DESC?.trim() || "";
  const filterPlan = labels.ADMIN_FILTER_PLAN?.trim() || "";
  const filterPlanDesc = labels.ADMIN_FILTER_PLAN_DESC?.trim() || "";
  const filterCountry = labels.ADMIN_FILTER_COUNTRY?.trim() || "";
  const filterCountryDesc = labels.ADMIN_FILTER_COUNTRY_DESC?.trim() || "";
  const filterProvider = labels.ADMIN_FILTER_PROVIDER?.trim() || "";
  const filterProviderDesc = labels.ADMIN_FILTER_PROVIDER_DESC?.trim() || "";
  const filterCreatedFrom = labels.ADMIN_FILTER_CREATED_FROM?.trim() || "";
  const filterCreatedFromDesc = labels.ADMIN_FILTER_CREATED_FROM_DESC?.trim() || "";
  const filterAll = labels.ADMIN_FILTER_ALL?.trim() || "";
  const dateRangePlaceholder = labels.ADMIN_DATE_RANGE_PLACEHOLDER?.trim() || "";
  const dateRangeClear = labels.ADMIN_DATE_RANGE_CLEAR?.trim() || "";
  const dateRangeApply = labels.ADMIN_DATE_RANGE_APPLY?.trim() || "";
  const htmlLang = labels.SITE_HTML_LANG?.trim() || "";
  const dateRangeMonthsRaw = billing.data?.date_range_months;
  const dateRangeMonths =
    typeof dateRangeMonthsRaw === "number" && Number.isFinite(dateRangeMonthsRaw)
      ? dateRangeMonthsRaw
      : 0;
  const colEmail = labels.ADMIN_USERS_COL_EMAIL?.trim() || "";
  const colStatus = labels.ADMIN_USERS_COL_STATUS?.trim() || "";
  const colPlan = labels.ADMIN_USERS_COL_PLAN?.trim() || "";
  const colCountry = labels.ADMIN_USERS_COL_COUNTRY?.trim() || "";
  const colProvider = labels.ADMIN_USERS_COL_PROVIDER?.trim() || "";
  const viewLabel = labels.ADMIN_ACTION_VIEW?.trim() || "";
  const rowActionsLabel = labels.ADMIN_TABLE_ROW_ACTIONS?.trim() || "";
  const statusOptions = accountStatusFilterOptions(labels);
  const planOptions = useMemo(
    () =>
      (plans.data?.items || [])
        .map((p) => {
          const value = String(p.id || "").trim();
          const label = String(p.display_name || p.id || "").trim();
          return value && label ? { value, label } : null;
        })
        .filter((o): o is { value: string; label: string } => Boolean(o)),
    [plans.data?.items],
  );
  const countryOptions = useMemo(
    () =>
      (countries.data?.items || [])
        .map((item) => {
          const value = String(item.value || "")
            .trim()
            .toUpperCase();
          const label = String(item.label || item.value || "")
            .trim()
            .toUpperCase();
          return value && label ? { value, label } : null;
        })
        .filter((o): o is { value: string; label: string } => Boolean(o)),
    [countries.data?.items],
  );
  const providerOptions = useMemo(() => {
    const allowed = new Set(
      (authSettings.data?.allowed_providers || [])
        .map((p) => p.trim().toLowerCase())
        .filter(Boolean),
    );
    const fromCatalog = (authSettings.data?.catalog || [])
      .map((item) => {
        const value = String(item.id || "")
          .trim()
          .toLowerCase();
        const label = String(item.display_name || "").trim();
        return value && label && allowed.has(value) ? { value, label } : null;
      })
      .filter((o): o is { value: string; label: string } => Boolean(o));
    if (fromCatalog.length) return fromCatalog;
    // Fail-closed: no invent labels from raw ids when catalog chrome is missing.
    return [];
  }, [authSettings.data?.allowed_providers, authSettings.data?.catalog]);

  const openUser = useCallback(
    (row: AdminUserListItem) => {
      const id = row.id?.trim() || "";
      if (!id) return;
      router.push(`/users/${id}`);
    },
    [router],
  );

  // Reset page when live filters change.
  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setSkip(0);
  }, [q, status, plan, country, provider, createdRange.from, createdRange.to]);

  const columns = useMemo<ColumnDef<AdminUserListItem>[]>(
    () => [
      {
        id: "email",
        accessorKey: "email",
        header: colEmail,
        cell: ({ row }) => {
          const email = row.original.email?.trim() || "";
          const id = row.original.id?.trim() || "";
          if (!email || !id) return "";
          return (
            <button
              type="button"
              className="text-left text-foreground hover:underline"
              onClick={() => openUser(row.original)}
            >
              {email}
            </button>
          );
        },
      },
      {
        id: "account_status",
        accessorKey: "account_status_label",
        header: colStatus,
        cell: ({ row }) => row.original.account_status_label?.trim() || "",
      },
      {
        id: "plan_tier",
        accessorKey: "plan_tier",
        header: colPlan,
        cell: ({ row }) => row.original.plan_tier?.trim() || "",
      },
      {
        id: "last_login_country",
        accessorKey: "last_login_country",
        header: colCountry,
        cell: ({ row }) => row.original.last_login_country?.trim() || "",
      },
      {
        id: "auth_provider",
        accessorKey: "auth_provider",
        header: colProvider,
        cell: ({ row }) => row.original.auth_provider?.trim() || "",
      },
      {
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const id = row.original.id?.trim() || "";
          if (!id || !viewLabel || !rowActionsLabel) return null;
          return (
            <DataTableRowActions
              triggerLabel={rowActionsLabel}
              actions={[
                {
                  id: "view",
                  label: viewLabel,
                  onSelect: () => openUser(row.original),
                },
              ]}
            />
          );
        },
      },
    ],
    [colEmail, colStatus, colPlan, colCountry, colProvider, viewLabel, rowActionsLabel, openUser],
  );

  const bootReady = useListBootReady(pageSize, Boolean(chrome.data), queryContentReady(users));
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    return (
      <div className="space-y-4">
        {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
        <DataTableSkeleton
          title=""
          columns={[colEmail, colStatus, colPlan, colCountry, colProvider].map((label) => ({
            label: label || "",
          }))}
          rows={pageSize > 0 ? pageSize : 8}
          filterCount={6}
          filterWideIndexes={[5]}
        />
      </div>
    );
  }

  const items = (users.data?.items || []).filter((row) => Boolean(row.id?.trim()));

  return (
    <div className="space-y-4">
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      <AdminFilterBar
        filters={[
          {
            kind: "search",
            id: "users-search",
            label: searchPlaceholder,
            description: searchDesc,
            value: qInput,
            onChange: setQInput,
          },
          {
            kind: "select",
            id: "users-status",
            label: filterStatus,
            description: filterStatusDesc,
            value: status,
            onChange: setStatus,
            options: statusOptions,
            allLabel: filterAll,
          },
          {
            kind: "select",
            id: "users-plan",
            label: filterPlan,
            description: filterPlanDesc,
            value: plan,
            onChange: setPlan,
            options: planOptions,
            allLabel: filterAll,
          },
          {
            kind: "select",
            id: "users-country",
            label: filterCountry,
            description: filterCountryDesc,
            value: country,
            onChange: setCountry,
            options: countryOptions,
            allLabel: filterAll,
          },
          {
            kind: "select",
            id: "users-provider",
            label: filterProvider,
            description: filterProviderDesc,
            value: provider,
            onChange: setProvider,
            options: providerOptions,
            allLabel: filterAll,
          },
          {
            // dateRange last (Apply-only) so calendar chrome load never remounts search.
            kind: "dateRange",
            id: "users-created-range",
            label: filterCreatedFrom || dateRangePlaceholder,
            description: filterCreatedFromDesc,
            value: createdRange,
            onChange: setCreatedRange,
            placeholder: dateRangePlaceholder,
            clearLabel: dateRangeClear,
            applyLabel: dateRangeApply,
            locale: htmlLang,
            numberOfMonths: dateRangeMonths,
          },
        ]}
      />
      <Card>
        <CardContent className="pt-6">
          <DataTable
            columns={columns}
            data={items}
            meta={users.data?.meta}
            onPage={setSkip}
            pageDisabled={users.isFetching}
            isFetching={users.isFetching && !users.isPending}
            getRowId={(row) => row.id}
            bordered={false}
          />
        </CardContent>
      </Card>
    </div>
  );
}
