"use client";

import { AdminFilterBar } from "@/components/admin/admin-filter-bar";
import { DataTableSkeleton } from "@/components/skeletons/page-skeletons";
import { Card, CardContent } from "@/components/ui/card";
import { type ColumnDef, DataTable } from "@/components/ui/data-table";
import { useAdminPlans, useAdminSubscriptions } from "@/hooks/queries/billing";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminToken } from "@/hooks/use-admin-token";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { subscriptionStatusFilterOptions } from "@/lib/admin-filter-options";
import { formatDateTimeFull, formatDateTimeShort } from "@/lib/format-datetime";
import { useEffect, useMemo, useState } from "react";

type SubRow = Record<string, unknown>;

export function SubscriptionsClient({
  initialToken,
}: {
  initialToken?: string;
}) {
  const token = useAdminToken(initialToken);
  const pageSize = useDefaultPageSize();
  const [skip, setSkip] = useState(0);
  const [qInput, setQInput] = useState("");
  const [status, setStatus] = useState("");
  const [plan, setPlan] = useState("");
  const q = useDebouncedValue(qInput.trim(), 250);
  const chrome = useAdminNavChrome(token);
  const plans = useAdminPlans(token, 0, pageSize > 0 ? Math.max(pageSize, 50) : 0);
  const subs = useAdminSubscriptions(token, skip, pageSize, { q, status, plan });
  const labels = chrome.data ?? {};
  const htmlLang = labels.SITE_HTML_LANG?.trim() || "";
  const title = labels.ADMIN_NAV_SUBSCRIPTIONS?.trim() || "";
  const filterSearch = labels.ADMIN_FILTER_SEARCH?.trim() || "";
  const filterStatus = labels.ADMIN_FILTER_STATUS?.trim() || "";
  const filterPlan = labels.ADMIN_FILTER_PLAN?.trim() || "";
  const filterSearchDesc = labels.ADMIN_FILTER_SEARCH_DESC?.trim() || "";
  const filterStatusDesc = labels.ADMIN_FILTER_STATUS_DESC?.trim() || "";
  const filterPlanDesc = labels.ADMIN_FILTER_PLAN_DESC?.trim() || "";
  const filterAll = labels.ADMIN_FILTER_ALL?.trim() || "";
  const colEmail = labels.ADMIN_SUB_COL_EMAIL?.trim() || "";
  const colPlan = labels.ADMIN_SUB_COL_PLAN?.trim() || "";
  const colStatus = labels.ADMIN_SUB_COL_STATUS?.trim() || "";
  const colPaddle = labels.ADMIN_SUB_COL_PADDLE?.trim() || "";
  const colExpires = labels.ADMIN_SUB_COL_EXPIRES?.trim() || "";
  const colWebhook = labels.ADMIN_SUB_LAST_WEBHOOK?.trim() || "";
  const statusOptions = subscriptionStatusFilterOptions(labels);
  const planOptions = useMemo(
    () =>
      (plans.data?.items || [])
        .map((p) => {
          const value = String(p.id || "").trim();
          const label = String(p.display_name || "").trim();
          return value && label ? { value, label } : null;
        })
        .filter((o): o is { value: string; label: string } => Boolean(o)),
    [plans.data?.items],
  );

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setSkip(0);
  }, [q, status, plan]);

  const columns = useMemo<ColumnDef<SubRow>[]>(
    () => [
      {
        id: "email",
        header: colEmail,
        cell: ({ row }) => String(row.original.email || ""),
      },
      {
        id: "plan_tier",
        header: colPlan,
        cell: ({ row }) => String(row.original.plan_tier || ""),
      },
      {
        id: "status",
        header: colStatus,
        cell: ({ row }) => String(row.original.status_label || ""),
      },
      {
        id: "paddle_subscription_id",
        header: colPaddle,
        cell: ({ row }) => (
          <span className="font-mono text-xs">
            {String(row.original.paddle_subscription_id || "")}
          </span>
        ),
      },
      {
        id: "expires",
        header: colExpires,
        cell: ({ row }) => {
          const raw = String(row.original.expires_at || "");
          return (
            <span title={formatDateTimeFull(raw, htmlLang) || undefined}>
              {formatDateTimeShort(raw, htmlLang)}
            </span>
          );
        },
      },
      {
        id: "last_webhook_at",
        header: colWebhook,
        cell: ({ row }) => {
          const raw = String(row.original.last_webhook_at || "");
          return (
            <span title={formatDateTimeFull(raw, htmlLang) || undefined}>
              {formatDateTimeShort(raw, htmlLang)}
            </span>
          );
        },
      },
    ],
    [colEmail, colPlan, colStatus, colPaddle, colExpires, colWebhook, htmlLang],
  );

  const bootReady = useListBootReady(pageSize, Boolean(chrome.data), queryContentReady(subs));
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    return (
      <DataTableSkeleton
        title={title}
        columns={[colEmail, colPlan, colStatus, colPaddle, colExpires, colWebhook]
          .filter(Boolean)
          .map((label) => ({ label }))}
        rows={pageSize > 0 ? pageSize : 8}
        showFilters
        filterCount={3}
      />
    );
  }

  return (
    <div className="space-y-4">
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      <AdminFilterBar
        filters={[
          {
            kind: "search",
            id: "subs-search",
            label: filterSearch,
            description: filterSearchDesc || "",
            value: qInput,
            onChange: setQInput,
          },
          {
            kind: "select",
            id: "subs-status",
            label: filterStatus,
            description: filterStatusDesc,
            value: status,
            onChange: setStatus,
            options: statusOptions,
            allLabel: filterAll,
          },
          {
            kind: "select",
            id: "subs-plan",
            label: filterPlan,
            description: filterPlanDesc,
            value: plan,
            onChange: setPlan,
            options: planOptions,
            allLabel: filterAll,
          },
        ]}
      />
      <Card>
        <CardContent className="pt-6">
          <DataTable
            columns={columns}
            data={((subs.data?.items as SubRow[]) ?? []).filter((row) =>
              Boolean(String(row.id || "")),
            )}
            meta={subs.data?.meta}
            onPage={setSkip}
            pageDisabled={subs.isFetching}
            isFetching={subs.isFetching && !subs.isPending}
            getRowId={(row) => String(row.id || "")}
            bordered={false}
          />
        </CardContent>
      </Card>
    </div>
  );
}
