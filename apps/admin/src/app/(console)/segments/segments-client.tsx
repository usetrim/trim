"use client";

import { AdminFilterBar } from "@/components/admin/admin-filter-bar";
import { DataTableSkeleton, TabsSkeleton } from "@/components/skeletons/page-skeletons";
import { Card, CardContent } from "@/components/ui/card";
import { type ColumnDef, DataTable } from "@/components/ui/data-table";
import type { DateRangeValue } from "@/components/ui/date-range-picker";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useAdminBillingSettings, useAdminPlans } from "@/hooks/queries/billing";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import {
  useAdminSegmentsEnterprises,
  useAdminSegmentsIndividuals,
  useAdminSegmentsTeams,
} from "@/hooks/queries/segments";
import { useAdminUserCountries } from "@/hooks/queries/users";
import { useAdminToken } from "@/hooks/use-admin-token";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import {
  accountStatusFilterOptions,
  billingIntervalFilterOptions,
} from "@/lib/admin-filter-options";
import Link from "next/link";
import { useEffect, useMemo, useState } from "react";

type Row = Record<string, unknown>;

function str(v: unknown): string {
  return typeof v === "string" ? v : "";
}

function num(v: unknown): number | null {
  return typeof v === "number" ? v : null;
}

export function SegmentsClient({ initialToken }: { initialToken?: string }) {
  const token = useAdminToken(initialToken);
  const pageSize = useDefaultPageSize();
  const [skipInd, setSkipInd] = useState(0);
  const [skipTeams, setSkipTeams] = useState(0);
  const [skipEnt, setSkipEnt] = useState(0);
  const [plan, setPlan] = useState("");
  const [status, setStatus] = useState("");
  const [country, setCountry] = useState("");
  const [billingInterval, setBillingInterval] = useState("");
  const [createdRange, setCreatedRange] = useState<DateRangeValue>({ from: null, to: null });
  const [creditsLeftInput, setCreditsLeftInput] = useState("");
  const [qInput, setQInput] = useState("");
  const creditsLeftMax = useDebouncedValue(creditsLeftInput.trim(), 250);
  const q = useDebouncedValue(qInput.trim(), 250);
  const [teamPlan, setTeamPlan] = useState("");
  const [teamQInput, setTeamQInput] = useState("");
  const teamQ = useDebouncedValue(teamQInput.trim(), 250);
  const [teamCreatedRange, setTeamCreatedRange] = useState<DateRangeValue>({
    from: null,
    to: null,
  });
  const [entStatus, setEntStatus] = useState("");
  const [entCountry, setEntCountry] = useState("");
  const [entQInput, setEntQInput] = useState("");
  const [entCreatedRange, setEntCreatedRange] = useState<DateRangeValue>({
    from: null,
    to: null,
  });
  const entQ = useDebouncedValue(entQInput.trim(), 250);
  const chrome = useAdminNavChrome(token);
  const billing = useAdminBillingSettings(token);
  const plans = useAdminPlans(token, 0, pageSize > 0 ? Math.max(pageSize, 50) : 0);
  const countries = useAdminUserCountries(token, "segments");
  const individuals = useAdminSegmentsIndividuals(token, skipInd, pageSize, {
    plan,
    status,
    country,
    interval: billingInterval,
    created_from: createdRange.from || "",
    created_to: createdRange.to || "",
    credits_left_max: creditsLeftMax,
    q,
  });
  const teams = useAdminSegmentsTeams(token, skipTeams, pageSize, {
    plan: teamPlan,
    q: teamQ,
    created_from: teamCreatedRange.from || "",
    created_to: teamCreatedRange.to || "",
  });
  const enterprises = useAdminSegmentsEnterprises(token, skipEnt, pageSize, {
    status: entStatus,
    country: entCountry,
    q: entQ,
    created_from: entCreatedRange.from || "",
    created_to: entCreatedRange.to || "",
  });

  const labels = chrome.data ?? {};
  const title = labels.ADMIN_NAV_SEGMENTS?.trim() || "";
  const tabInd = labels.ADMIN_SEGMENT_INDIVIDUALS?.trim() || "";
  const tabTeams = labels.ADMIN_SEGMENT_TEAMS?.trim() || "";
  const tabEnt = labels.ADMIN_SEGMENT_ENTERPRISE?.trim() || "";
  const colEmail = labels.ADMIN_SEGMENT_COL_EMAIL?.trim() || "";
  const colPlan = labels.ADMIN_SEGMENT_COL_PLAN?.trim() || "";
  const colStatus = labels.ADMIN_SEGMENT_COL_STATUS?.trim() || "";
  const colCountry = labels.ADMIN_SEGMENT_COL_COUNTRY?.trim() || "";
  const colCredits = labels.ADMIN_SEGMENT_COL_CREDITS?.trim() || "";
  const colInterval = labels.ADMIN_SEGMENT_COL_INTERVAL?.trim() || "";
  const colChurn = labels.ADMIN_SEGMENT_COL_CHURN?.trim() || "";
  const colSavings = labels.ADMIN_SEGMENT_COL_SAVINGS?.trim() || "";
  const colName = labels.ADMIN_SEGMENT_COL_NAME?.trim() || "";
  const colSeats = labels.ADMIN_SEGMENT_COL_SEATS?.trim() || "";
  const colMembers = labels.ADMIN_SEGMENT_COL_MEMBERS?.trim() || "";
  const colPending = labels.ADMIN_SEGMENT_COL_PENDING?.trim() || "";
  const filterPlan = labels.ADMIN_FILTER_PLAN?.trim() || "";
  const filterStatus = labels.ADMIN_FILTER_STATUS?.trim() || "";
  const filterCountry = labels.ADMIN_FILTER_COUNTRY?.trim() || "";
  const filterInterval = labels.ADMIN_FILTER_INTERVAL?.trim() || "";
  const filterCreditsLeft = labels.ADMIN_FILTER_CREDITS_LEFT_MAX?.trim() || "";
  const filterSearch = labels.ADMIN_FILTER_SEARCH?.trim() || "";
  const filterSearchDesc = labels.ADMIN_FILTER_SEARCH_DESC?.trim() || "";
  const filterStatusDesc = labels.ADMIN_FILTER_STATUS_DESC?.trim() || "";
  const filterPlanDesc = labels.ADMIN_FILTER_PLAN_DESC?.trim() || "";
  const filterCountryDesc = labels.ADMIN_FILTER_COUNTRY_DESC?.trim() || "";
  const filterIntervalDesc = labels.ADMIN_FILTER_INTERVAL_DESC?.trim() || "";
  const filterCreditsLeftDesc = labels.ADMIN_FILTER_CREDITS_LEFT_DESC?.trim() || "";
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
  const creditsFmt = labels.ADMIN_SEGMENT_CREDITS_FMT?.trim() || "";
  const statusOptions = accountStatusFilterOptions(labels);
  const intervalOptions = billingIntervalFilterOptions(labels);
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

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setSkipInd(0);
  }, [
    plan,
    status,
    country,
    billingInterval,
    createdRange.from,
    createdRange.to,
    creditsLeftMax,
    q,
  ]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setSkipTeams(0);
  }, [teamPlan, teamQ, teamCreatedRange.from, teamCreatedRange.to]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setSkipEnt(0);
  }, [entStatus, entCountry, entQ, entCreatedRange.from, entCreatedRange.to]);

  const indColumns = useMemo<ColumnDef<Row>[]>(
    () => [
      {
        id: "email",
        header: colEmail,
        cell: ({ row }) => {
          const id = str(row.original.id);
          const email = str(row.original.email);
          if (!id || !email) return "";
          return (
            <Link href={`/users/${id}`} className="text-foreground hover:underline">
              {email}
            </Link>
          );
        },
      },
      {
        id: "plan_tier",
        header: colPlan,
        cell: ({ row }) => str(row.original.plan_tier),
      },
      {
        id: "account_status",
        header: colStatus,
        cell: ({ row }) => str(row.original.account_status_label),
      },
      {
        id: "last_login_country",
        header: colCountry,
        cell: ({ row }) => str(row.original.last_login_country),
      },
      {
        id: "credits",
        header: colCredits,
        cell: ({ row }) => {
          const used = num(row.original.credits_used);
          const limit = num(row.original.credits_limit);
          if (used == null && limit == null) return "";
          const usedStr = used != null ? String(used) : "";
          const limitStr = limit != null ? String(limit) : "";
          if (creditsFmt?.includes("{used}") && creditsFmt.includes("{limit}")) {
            return (
              <span className="tabular-nums">
                {creditsFmt.replaceAll("{used}", usedStr).replaceAll("{limit}", limitStr)}
              </span>
            );
          }
          if (usedStr) return <span className="tabular-nums">{usedStr}</span>;
          if (limitStr) return <span className="tabular-nums">{limitStr}</span>;
          return "";
        },
      },
      {
        id: "billing_interval",
        header: colInterval,
        cell: ({ row }) => str(row.original.billing_interval),
      },
      {
        id: "churn_risk",
        header: colChurn,
        cell: ({ row }) => str(row.original.churn_risk),
      },
      {
        id: "tokens_saved",
        header: colSavings,
        cell: ({ row }) => {
          const saved = num(row.original.tokens_saved);
          return saved != null ? <span className="tabular-nums">{saved}</span> : "";
        },
      },
    ],
    [
      colEmail,
      colPlan,
      colStatus,
      colCountry,
      colCredits,
      colInterval,
      colChurn,
      colSavings,
      creditsFmt,
    ],
  );

  const teamColumns = useMemo<ColumnDef<Row>[]>(
    () => [
      {
        id: "name",
        header: colName,
        cell: ({ row }) => str(row.original.name),
      },
      {
        id: "plan_tier",
        header: colPlan,
        cell: ({ row }) => str(row.original.plan_tier),
      },
      {
        id: "allocated_seats",
        header: colSeats,
        cell: ({ row }) => {
          const v = num(row.original.allocated_seats);
          return v != null ? <span className="tabular-nums">{v}</span> : "";
        },
      },
      {
        id: "members",
        header: colMembers,
        cell: ({ row }) => {
          const v = num(row.original.members);
          return v != null ? <span className="tabular-nums">{v}</span> : "";
        },
      },
      {
        id: "pending_invites",
        header: colPending,
        cell: ({ row }) => {
          const v = num(row.original.pending_invites);
          return v != null ? <span className="tabular-nums">{v}</span> : "";
        },
      },
    ],
    [colName, colPlan, colSeats, colMembers, colPending],
  );

  const entColumns = useMemo<ColumnDef<Row>[]>(
    () => [
      {
        id: "email",
        header: colEmail,
        cell: ({ row }) => {
          const id = str(row.original.id);
          const email = str(row.original.email);
          if (!id || !email) return "";
          return (
            <Link href={`/users/${id}`} className="text-foreground hover:underline">
              {email}
            </Link>
          );
        },
      },
      {
        id: "plan_tier",
        header: colPlan,
        cell: ({ row }) => str(row.original.plan_tier),
      },
      {
        id: "account_status",
        header: colStatus,
        cell: ({ row }) => str(row.original.account_status_label),
      },
      {
        id: "last_login_country",
        header: colCountry,
        cell: ({ row }) => str(row.original.last_login_country),
      },
      {
        id: "tokens_saved",
        header: colSavings,
        cell: ({ row }) => {
          const saved = num(row.original.tokens_saved);
          return saved != null ? <span className="tabular-nums">{saved}</span> : "";
        },
      },
      {
        id: "company_name",
        header: colName,
        cell: ({ row }) => str(row.original.company_name),
      },
    ],
    [colEmail, colPlan, colStatus, colCountry, colSavings, colName],
  );

  // Boot only: after chrome+pageSize, keep filters mounted while lists refetch (debounce-safe).
  const [tab, setTab] = useState("individuals");
  const primarySegment =
    tab === "teams" ? teams : tab === "enterprises" ? enterprises : individuals;

  const bootReady = useListBootReady(
    pageSize,
    Boolean(chrome.data),
    queryContentReady(primarySegment),
  );
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    const segmentTabs = [tabInd, tabTeams, tabEnt].filter(Boolean).length;
    return (
      <div className="space-y-4">
        {title ? (
          <h1 className="text-xl font-semibold tracking-tight text-muted-foreground">{title}</h1>
        ) : null}
        {segmentTabs >= 2 ? <TabsSkeleton count={segmentTabs} /> : <TabsSkeleton count={3} />}
        <DataTableSkeleton
          title=""
          columns={[
            colEmail,
            colPlan,
            colStatus,
            colCountry,
            colCredits,
            colInterval,
            colChurn,
            colSavings,
          ].map((label) => ({ label: label || "" }))}
          rows={pageSize > 0 ? pageSize : 8}
          showFilters
          filterCount={7}
          filterWideIndexes={[5]}
        />
      </div>
    );
  }

  // Keep every tab panel mounted (hide inactive) so search inputs never remount
  // on tab chrome updates or list refetches - same stability as Enterprise.
  return (
    <div className="space-y-4">
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          {tabInd ? <TabsTrigger value="individuals">{tabInd}</TabsTrigger> : null}
          {tabTeams ? <TabsTrigger value="teams">{tabTeams}</TabsTrigger> : null}
          {tabEnt ? <TabsTrigger value="enterprises">{tabEnt}</TabsTrigger> : null}
        </TabsList>
      </Tabs>

      <div className={tab === "individuals" ? "space-y-3" : "hidden"}>
        {/* Search-only bar first - identical stability model to Enterprise/Email. */}
        <AdminFilterBar
          filters={[
            {
              kind: "search",
              id: "seg-ind-search",
              label: filterSearch,
              description: filterSearchDesc,
              value: qInput,
              onChange: setQInput,
            },
            {
              kind: "search",
              id: "seg-ind-credits",
              label: filterCreditsLeft,
              description: filterCreditsLeftDesc,
              value: creditsLeftInput,
              onChange: setCreditsLeftInput,
            },
          ]}
        />
        <AdminFilterBar
          filters={[
            {
              kind: "select",
              id: "seg-ind-plan",
              label: filterPlan,
              description: filterPlanDesc,
              value: plan,
              onChange: setPlan,
              options: planOptions,
              allLabel: filterAll,
            },
            {
              kind: "select",
              id: "seg-ind-status",
              label: filterStatus,
              description: filterStatusDesc,
              value: status,
              onChange: setStatus,
              options: statusOptions,
              allLabel: filterAll,
            },
            {
              kind: "select",
              id: "seg-ind-country",
              label: filterCountry,
              description: filterCountryDesc,
              value: country,
              onChange: setCountry,
              options: countryOptions,
              allLabel: filterAll,
            },
            {
              kind: "select",
              id: "seg-ind-interval",
              label: filterInterval,
              description: filterIntervalDesc,
              value: billingInterval,
              onChange: setBillingInterval,
              options: intervalOptions,
              allLabel: filterAll,
            },
            {
              kind: "dateRange",
              id: "seg-ind-created-range",
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
              columns={indColumns}
              data={((individuals.data?.items || []) as Row[]).filter(
                (item) => str(item.id) && str(item.email),
              )}
              meta={individuals.data?.meta}
              onPage={setSkipInd}
              pageDisabled={individuals.isFetching}
              isFetching={individuals.isFetching && !individuals.isPending}
              getRowId={(row) => str(row.id)}
              bordered={false}
            />
          </CardContent>
        </Card>
      </div>

      <div className={tab === "teams" ? "space-y-3" : "hidden"}>
        <AdminFilterBar
          filters={[
            {
              kind: "search",
              id: "seg-team-search",
              label: filterSearch,
              description: filterSearchDesc,
              value: teamQInput,
              onChange: setTeamQInput,
            },
          ]}
        />
        <AdminFilterBar
          filters={[
            {
              kind: "select",
              id: "seg-team-plan",
              label: filterPlan,
              description: filterPlanDesc,
              value: teamPlan,
              onChange: setTeamPlan,
              options: planOptions,
              allLabel: filterAll,
            },
            {
              kind: "dateRange",
              id: "seg-team-created-range",
              label: filterCreatedFrom || dateRangePlaceholder,
              description: filterCreatedFromDesc,
              value: teamCreatedRange,
              onChange: setTeamCreatedRange,
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
              columns={teamColumns}
              data={((teams.data?.items || []) as Row[]).filter(
                (item) => str(item.id) && str(item.name),
              )}
              meta={teams.data?.meta}
              onPage={setSkipTeams}
              pageDisabled={teams.isFetching}
              isFetching={teams.isFetching && !teams.isPending}
              getRowId={(row) => str(row.id)}
              bordered={false}
            />
          </CardContent>
        </Card>
      </div>

      <div className={tab === "enterprises" ? "space-y-3" : "hidden"}>
        <AdminFilterBar
          filters={[
            {
              kind: "search",
              id: "seg-ent-search",
              label: filterSearch,
              description: filterSearchDesc,
              value: entQInput,
              onChange: setEntQInput,
            },
          ]}
        />
        <AdminFilterBar
          filters={[
            {
              kind: "select",
              id: "seg-ent-status",
              label: filterStatus,
              description: filterStatusDesc,
              value: entStatus,
              onChange: setEntStatus,
              options: statusOptions,
              allLabel: filterAll,
            },
            {
              kind: "select",
              id: "seg-ent-country",
              label: filterCountry,
              description: filterCountryDesc,
              value: entCountry,
              onChange: setEntCountry,
              options: countryOptions,
              allLabel: filterAll,
            },
            {
              kind: "dateRange",
              id: "seg-ent-created-range",
              label: filterCreatedFrom || dateRangePlaceholder,
              description: filterCreatedFromDesc,
              value: entCreatedRange,
              onChange: setEntCreatedRange,
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
              columns={entColumns}
              data={((enterprises.data?.items || []) as Row[]).filter(
                (item) => str(item.id) && str(item.email),
              )}
              meta={enterprises.data?.meta}
              onPage={setSkipEnt}
              pageDisabled={enterprises.isFetching}
              isFetching={enterprises.isFetching && !enterprises.isPending}
              getRowId={(row) => str(row.id)}
              bordered={false}
            />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
