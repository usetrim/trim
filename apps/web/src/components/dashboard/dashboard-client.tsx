"use client";

import { PlanModal } from "@/components/billing/plan-modal";
import { LocHeatmap } from "@/components/dashboard/loc-heatmap";
import { ModeBreakdownChart } from "@/components/dashboard/mode-breakdown-chart";
import { ModelBreakdownChart } from "@/components/dashboard/model-breakdown-chart";
import { StatusBreakdownChart } from "@/components/dashboard/status-breakdown-chart";
import { TokenSavingsChart } from "@/components/dashboard/token-savings-chart";
import { UsageStackedChart } from "@/components/dashboard/usage-stacked-chart";
import {
  ChartSkeleton,
  DashboardPageSkeleton,
  LocHeatmapSkeleton,
  MetricCardsSkeleton,
  MetricValueSkeleton,
} from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { FetchProgressBar } from "@/components/ui/fetch-progress";
import { usePortalSession } from "@/hooks/mutations/billing";
import { useAvatarSync, useDeleteAccount } from "@/hooks/mutations/me";
import { useAuthProviders } from "@/hooks/queries/auth";
import { useSubscriptionStatus } from "@/hooks/queries/billing";
import { useEventStats, useQuota } from "@/hooks/queries/me";
import { formatDateTimeShort } from "@/lib/format-datetime";
import { dashboardSkeletonChrome } from "@/lib/skeleton-chrome";
import { createClient } from "@/lib/supabase/client";
import { toastApiError } from "@/lib/toast-api";
import { useBillingUI } from "@/stores/billing-ui";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";

export function DashboardClient({
  accessToken,
  userId,
  email,
  fullName,
  sessionReady = true,
}: {
  accessToken?: string;
  userId?: string;
  email?: string;
  fullName?: string;
  /** False until browser getSession settles - avoids forever-skeleton when query is disabled. */
  sessionReady?: boolean;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { planModalOpen, setPlanModalOpen, openPlanModal } = useBillingUI();
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [usageGroupBy, setUsageGroupBy] = useState("");
  const [heatmapScope, setHeatmapScope] = useState("");
  const [actionNotice, setActionNotice] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const openedBillingQueryRef = useRef(false);
  const deleteAccount = useDeleteAccount(accessToken);
  const avatarSync = useAvatarSync(accessToken);
  const portal = usePortalSession(accessToken);
  const subscription = useSubscriptionStatus(accessToken);
  const authProviders = useAuthProviders();
  const dialogCancel = authProviders.data?.site?.dialog_cancel?.trim() || "";

  // Pricing / login deep-link: /dashboard?open=plans|topup → open PlanModal once, then clean URL.
  useEffect(() => {
    if (openedBillingQueryRef.current) return;
    const open = (searchParams.get("open") || "").trim();
    if (open !== "plans" && open !== "topup") return;
    openedBillingQueryRef.current = true;
    openPlanModal({ mode: open === "topup" ? "topup" : "plans" });
    const params = new URLSearchParams(searchParams.toString());
    params.delete("open");
    const q = params.toString();
    router.replace(q ? `${pathname}?${q}` : pathname, { scroll: false });
  }, [searchParams, openPlanModal, router, pathname]);

  const quota = useQuota(accessToken);
  const pageSize =
    (subscription.data?.default_page_size && subscription.data.default_page_size > 0
      ? subscription.data.default_page_size
      : 0) ||
    (authProviders.data?.default_page_size && authProviders.data.default_page_size > 0
      ? authProviders.data.default_page_size
      : 0);
  const eventStats = useEventStats(accessToken, usageGroupBy, heatmapScope);

  // Progress vs total pool (monthly limit + top-up). Matches remaining = (limit - used) + topup.
  const usagePct = useMemo(() => {
    if (!quota.data) return 0;
    if (quota.data.unlimited) return 0;
    const limit = Math.max(0, quota.data.monthly_credit_limit);
    const topup =
      typeof quota.data.purchased_topup_credits === "number"
        ? Math.max(0, quota.data.purchased_topup_credits)
        : 0;
    const pool = limit + topup;
    if (pool <= 0) return 0;
    return Math.min(100, Math.round((Math.max(0, quota.data.monthly_credit_used) / pool) * 100));
  }, [quota.data]);

  const savedTokensAllTime = useMemo(() => {
    return (eventStats.data?.token_series ?? []).reduce(
      (sum, row) => sum + Math.max(0, row.tokens_saved ?? 0),
      0,
    );
  }, [eventStats.data]);

  const acceptancePct = useMemo(() => {
    const rate = eventStats.data?.acceptance?.acceptance_rate;
    if (typeof rate !== "number") return null;
    return Math.round(rate * 1000) / 10;
  }, [eventStats.data]);

  const pageChrome = dashboardSkeletonChrome(authProviders.data?.site, {
    primary: subscription.data ? Boolean(subscription.data.primary_action_label?.trim()) : true,
    topup: subscription.data ? Boolean(subscription.data.buy_topup_label?.trim()) : true,
    portal: Boolean(
      subscription.data?.has_active_paid && subscription.data?.portal_action_label?.trim(),
    ),
    // Team/Settings live in the shared sidebar shell now - not page header slots.
    team: false,
    settings: false,
    notifications: false,
  });
  const authBounceRef = useRef(false);
  // Prefer env path immediately so bounce does not wait on auth-providers chrome.
  const loginPath = (
    process.env.NEXT_PUBLIC_APP_PATH_LOGIN ||
    authProviders.data?.site?.path_login ||
    ""
  ).trim();

  // Auth failures: bounce to login (toast lives on login). Never forever-skeleton.
  useEffect(() => {
    if (authBounceRef.current) return;
    if (!sessionReady || !subscription.isError || subscription.data) return;
    const status = (subscription.error as { status?: number } | null)?.status;
    if (status !== 401 && status !== 403) return;
    if (!loginPath) return;
    authBounceRef.current = true;
    void (async () => {
      const dest = `${loginPath}?error=session_rejected`;
      try {
        await Promise.race([
          createClient().auth.signOut(),
          new Promise<void>((resolve) => setTimeout(resolve, 2500)),
        ]);
      } catch {
        // Still leave the protected surface.
      } finally {
        window.location.replace(dest);
      }
    })();
  }, [sessionReady, subscription.isError, subscription.data, subscription.error, loginPath]);

  // Disabled query (no token) must not look like "still loading" forever.
  if (!sessionReady || !accessToken) {
    return <DashboardPageSkeleton chrome={pageChrome} tableRows={pageSize} embedded />;
  }
  if (subscription.isPending && !subscription.data && !subscription.error) {
    return <DashboardPageSkeleton chrome={pageChrome} tableRows={pageSize} embedded />;
  }
  if (subscription.isError && !subscription.data) {
    const status = (subscription.error as { status?: number } | null)?.status;
    if (status === 401 || status === 403) {
      return <DashboardPageSkeleton chrome={pageChrome} tableRows={pageSize} embedded />;
    }
    const errMsg =
      (subscription.error as Error)?.message?.trim() ||
      authProviders.data?.login_failed_message?.trim() ||
      "";
    return (
      <div className="mx-auto flex min-h-[50vh] w-full max-w-[1400px] flex-col items-center justify-center gap-4 px-6 py-16">
        {errMsg ? (
          <p className="max-w-md text-center text-sm text-[var(--trim-muted)]">{errMsg}</p>
        ) : null}
        {loginPath ? (
          <Button
            variant="outline"
            onClick={() => {
              void createClient()
                .auth.signOut()
                .finally(() => {
                  router.replace(loginPath);
                });
            }}
          >
            {authProviders.data?.site?.nav_sign_in || ""}
          </Button>
        ) : null}
      </div>
    );
  }
  if (!subscription.data) {
    return <DashboardPageSkeleton chrome={pageChrome} tableRows={pageSize} embedded />;
  }

  const sub = subscription.data;
  const htmlLang = sub.money_locale?.trim() || authProviders.data?.site?.html_lang?.trim() || "";
  const expiresDisplay =
    formatDateTimeShort(sub.expires_at, htmlLang) || sub.expires_at_label?.trim() || "";

  return (
    <div className="w-full space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:flex-wrap sm:items-end sm:justify-between">
        <div className="min-w-0">
          <p className="text-sm text-[var(--trim-muted)]">{subscription.data?.page_eyebrow}</p>
          <h1 className="text-2xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-3xl">
            {fullName && subscription.data?.welcome_prefix
              ? `${subscription.data.welcome_prefix} ${fullName}`
              : subscription.data?.welcome_guest || ""}
          </h1>
          <p className="mt-1 text-sm text-[var(--trim-muted)]">
            {subscription.data?.page_description}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {subscription.data?.primary_action_label ? (
            <Button onClick={() => openPlanModal({ mode: "plans" })}>
              {subscription.data.primary_action_label}
            </Button>
          ) : null}
          {subscription.data?.buy_topup_label ? (
            <Button variant="outline" onClick={() => openPlanModal({ mode: "topup" })}>
              {subscription.data.buy_topup_label}
            </Button>
          ) : null}
          {subscription.data?.has_active_paid && subscription.data.portal_action_label ? (
            <Button
              variant="outline"
              isLoading={portal.isPending}
              pendingLabel={subscription.data.portal_pending_label || undefined}
              onClick={() => {
                portal.mutate(undefined, {
                  onSuccess: (data) => {
                    if (data.overview_url) {
                      window.open(data.overview_url, "_blank", "noopener,noreferrer");
                      return;
                    }
                    toastApiError(new Error(subscription.data?.portal_url_missing_message || ""));
                  },
                  onError: (err) => {
                    toastApiError(err);
                  },
                });
              }}
            >
              {subscription.data.portal_action_label}
            </Button>
          ) : null}
        </div>
      </div>

      <div className="w-full">
        <div className="mt-4 flex flex-wrap items-center gap-3 rounded-lg border border-[var(--trim-border)] bg-[var(--trim-panel)] px-4 py-3 text-sm text-[var(--trim-muted)] shadow-[var(--trim-card-shadow)]">
          <span>
            {sub.status_prefix} <span className="text-[var(--trim-fg)]">{sub.status_label}</span>
          </span>
          {sub.plan_tier ? (
            <span>
              {sub.tier_prefix}{" "}
              <span className="text-[var(--trim-fg)]">{sub.plan_tier_label || ""}</span>
            </span>
          ) : null}
          {sub.billing_interval ? (
            <span>
              {sub.interval_prefix}{" "}
              <span className="text-[var(--trim-fg)]">{sub.billing_interval_label || ""}</span>
            </span>
          ) : null}
          {expiresDisplay ? (
            <span>
              {sub.renews_prefix} <span className="text-[var(--trim-fg)]">{expiresDisplay}</span>
            </span>
          ) : null}
          {sub.downgrade_policy_message ? (
            <span className="text-xs text-[var(--trim-muted)]">{sub.downgrade_policy_message}</span>
          ) : null}
        </div>

        {quota.data &&
        !quota.data.unlimited &&
        quota.data.remaining <= 0 &&
        subscription.data?.quota_exhausted_title ? (
          <div className="mt-4 rounded-lg border border-[var(--trim-border)] bg-[var(--trim-bg)] p-4 shadow-[var(--trim-card-shadow)]">
            <p className="text-sm font-semibold text-[var(--trim-fg)]">
              {subscription.data.quota_exhausted_title}
            </p>
            {subscription.data.quota_exhausted_body ? (
              <p className="mt-1 text-sm text-[var(--trim-muted)]">
                {subscription.data.quota_exhausted_body}
              </p>
            ) : null}
            <div className="mt-3 flex flex-wrap gap-2">
              {subscription.data.primary_action_label ? (
                <Button size="sm" onClick={() => openPlanModal({ mode: "plans" })}>
                  {subscription.data.primary_action_label}
                </Button>
              ) : null}
              {subscription.data.buy_topup_label ? (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => openPlanModal({ mode: "topup" })}
                >
                  {subscription.data.buy_topup_label}
                </Button>
              ) : null}
            </div>
          </div>
        ) : null}

        <div className="mt-4 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm font-medium text-[var(--trim-muted)]">
                {subscription.data?.metric_plan_label}
              </CardTitle>
            </CardHeader>
            <CardContent>
              {subscription.data?.plan_tier_label ? (
                <p className="text-2xl font-semibold text-[var(--trim-fg)]">
                  {subscription.data.plan_tier_label}
                </p>
              ) : (
                <MetricValueSkeleton />
              )}
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle className="text-sm font-medium text-[var(--trim-muted)]">
                {subscription.data?.metric_credits_label}
              </CardTitle>
            </CardHeader>
            <CardContent>
              {quota.data ? (
                <>
                  <p className="text-2xl font-semibold text-[var(--trim-fg)]">
                    {quota.data.unlimited
                      ? quota.data.unlimited_label?.trim() || null
                      : `${quota.data.monthly_credit_used} / ${quota.data.monthly_credit_limit}`}
                  </p>
                  {!quota.data.unlimited ? (
                    <div className="mt-3 h-1.5 overflow-hidden rounded trim-track">
                      <div
                        className="h-full bg-[var(--trim-ink)]"
                        style={{ width: `${usagePct}%` }}
                      />
                    </div>
                  ) : null}
                </>
              ) : (
                <MetricValueSkeleton />
              )}
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle className="text-sm font-medium text-[var(--trim-muted)]">
                {subscription.data?.metric_remaining_label}
              </CardTitle>
            </CardHeader>
            <CardContent>
              {quota.data ? (
                <>
                  <p className="text-2xl font-semibold text-[var(--trim-fg)]">
                    {quota.data.unlimited
                      ? quota.data.unlimited_label?.trim() || null
                      : quota.data.remaining}
                  </p>
                  {!quota.data.unlimited &&
                  subscription.data?.metric_topup_label &&
                  typeof quota.data.purchased_topup_credits === "number" ? (
                    <p className="mt-1 text-xs text-[var(--trim-muted)]">
                      {subscription.data.metric_topup_label}
                      {subscription.data.meta_sep || " · "}
                      {quota.data.purchased_topup_credits}
                    </p>
                  ) : null}
                </>
              ) : (
                <MetricValueSkeleton />
              )}
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle className="text-sm font-medium text-[var(--trim-muted)]">
                {subscription.data?.metric_tokens_saved_label}
              </CardTitle>
            </CardHeader>
            <CardContent>
              {eventStats.isPending && !eventStats.data ? (
                <MetricValueSkeleton />
              ) : (
                <p className="text-2xl font-semibold text-[var(--trim-fg)]">
                  {subscription.data?.money_locale
                    ? savedTokensAllTime.toLocaleString(subscription.data.money_locale)
                    : String(savedTokensAllTime)}
                </p>
              )}
            </CardContent>
          </Card>
        </div>

        {eventStats.isPending && !eventStats.data ? (
          <div className="mt-4">
            <MetricCardsSkeleton
              labels={[
                subscription.data?.metric_tab_label,
                subscription.data?.metric_lines_added_label,
                subscription.data?.metric_lines_deleted_label,
              ]}
              showCreditsBar={false}
              showHintLine
            />
          </div>
        ) : (
          <div className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <Card>
              <CardHeader>
                <CardTitle className="text-sm font-medium text-[var(--trim-muted)]">
                  {subscription.data?.metric_tab_label}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <>
                  <p className="text-2xl font-semibold text-[var(--trim-fg)]">
                    {acceptancePct === null ? "-" : `${acceptancePct}%`}
                  </p>
                  <p className="mt-1 text-xs text-[var(--trim-muted)]">
                    {eventStats.data?.acceptance.tab_suggestions_accepted ?? 0} /{" "}
                    {eventStats.data?.acceptance.tab_suggestions_shown ?? 0}{" "}
                    {subscription.data?.acceptance_hint}
                  </p>
                </>
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-sm font-medium text-[var(--trim-muted)]">
                  {subscription.data?.metric_lines_added_label}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-semibold text-[var(--trim-fg)]">
                  {subscription.data?.money_locale
                    ? (eventStats.data?.loc.ai_lines_added ?? 0).toLocaleString(
                        subscription.data.money_locale,
                      )
                    : String(eventStats.data?.loc.ai_lines_added ?? 0)}
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-sm font-medium text-[var(--trim-muted)]">
                  {subscription.data?.metric_lines_deleted_label}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-semibold text-[var(--trim-fg)]">
                  {subscription.data?.money_locale
                    ? (eventStats.data?.loc.ai_lines_deleted ?? 0).toLocaleString(
                        subscription.data.money_locale,
                      )
                    : String(eventStats.data?.loc.ai_lines_deleted ?? 0)}
                </p>
              </CardContent>
            </Card>
          </div>
        )}

        <div className="mt-8 space-y-8">
          {eventStats.isPending && !eventStats.data ? (
            <>
              <ChartSkeleton variant="usage" />
              <LocHeatmapSkeleton />
            </>
          ) : (
            <>
              {eventStats.data?.usage_title?.trim() &&
              (eventStats.data.usage_group_by_options?.length ?? 0) > 0 ? (
                <Card className="min-w-0">
                  <CardContent className="min-w-0 overflow-visible pt-6">
                    <UsageStackedChart
                      title={eventStats.data.usage_title}
                      subtitle={eventStats.data.usage_subtitle}
                      yAxisLabel={eventStats.data.usage_y_axis}
                      todayLabel={eventStats.data.usage_today_label}
                      todayDay={eventStats.data.today_day}
                      groupByPrefix={eventStats.data.usage_group_by_prefix}
                      groupByOptions={eventStats.data.usage_group_by_options}
                      groupBySelected={usageGroupBy || eventStats.data.usage_group_by_selected}
                      onGroupByChange={setUsageGroupBy}
                      series={eventStats.data.usage_series}
                      days={eventStats.data.usage_days}
                      emptyMessage={eventStats.data.usage_empty}
                      tooltipBreakdown={eventStats.data.usage_tooltip_breakdown}
                      tooltipDailyTotal={eventStats.data.usage_tooltip_daily_total}
                      tooltipCumulativeTotal={eventStats.data.usage_tooltip_cumulative_total}
                      tooltipShareFmt={eventStats.data.usage_tooltip_share_fmt}
                      isFetching={eventStats.isFetching && !eventStats.isPending}
                      locale={htmlLang}
                    />
                  </CardContent>
                </Card>
              ) : null}
              {eventStats.data?.loc_heatmap_title?.trim() &&
              (eventStats.data.loc_heatmap_scopes?.length ?? 0) > 0 &&
              (eventStats.data.loc_heatmap?.length ?? 0) > 0 ? (
                <Card className="min-w-0">
                  <CardContent className="min-w-0 overflow-visible pt-6">
                    <LocHeatmap
                      title={eventStats.data.loc_heatmap_title}
                      total={eventStats.data.loc_heatmap_total}
                      scopes={eventStats.data.loc_heatmap_scopes}
                      scopeSelected={heatmapScope || eventStats.data.loc_heatmap_scope_selected}
                      onScopeChange={setHeatmapScope}
                      days={eventStats.data.loc_heatmap}
                      emptyFmt={eventStats.data.loc_heatmap_empty_fmt}
                      valueFmt={eventStats.data.loc_heatmap_value_fmt}
                      weekdayLabels={eventStats.data.loc_heatmap_weekday_labels}
                      stats={eventStats.data.loc_heatmap_stats}
                      isFetching={eventStats.isFetching && !eventStats.isPending}
                      locale={htmlLang}
                    />
                  </CardContent>
                </Card>
              ) : null}
            </>
          )}
        </div>

        <div className="mt-8">
          {eventStats.isPending && !eventStats.data ? (
            <ChartSkeleton title={subscription.data?.chart_token_series_title} />
          ) : (
            <Card className="min-w-0">
              <div className="overflow-hidden">
                <FetchProgressBar active={eventStats.isFetching && !eventStats.isPending} />
              </div>
              <CardHeader className="space-y-1 pb-2">
                <CardTitle className="text-base sm:text-lg">
                  {subscription.data?.chart_token_series_title}
                </CardTitle>
              </CardHeader>
              <CardContent className="min-w-0 overflow-visible pt-0">
                <TokenSavingsChart
                  series={eventStats.data?.token_series ?? []}
                  seriesName={subscription.data?.metric_tokens_saved_label}
                  emptyMessage={eventStats.data?.empty_series_message}
                  locale={htmlLang}
                />
              </CardContent>
            </Card>
          )}
        </div>

        <div className="mt-8 grid grid-cols-1 gap-4 sm:gap-6 md:grid-cols-2 xl:grid-cols-3">
          {eventStats.isPending && !eventStats.data ? (
            <>
              <ChartSkeleton title={subscription.data?.chart_models_title} />
              <ChartSkeleton title={subscription.data?.chart_outcomes_title} variant="outcomes" />
              <ChartSkeleton title={subscription.data?.chart_modes_title} />
            </>
          ) : (
            <>
              <Card className="min-w-0">
                <div className="overflow-hidden">
                  <FetchProgressBar active={eventStats.isFetching && !eventStats.isPending} />
                </div>
                <CardHeader className="space-y-1 pb-2">
                  <CardTitle className="text-base sm:text-lg">
                    {subscription.data?.chart_models_title}
                  </CardTitle>
                </CardHeader>
                <CardContent className="min-w-0 overflow-visible pt-0">
                  <ModelBreakdownChart
                    breakdown={eventStats.data?.model_breakdown ?? []}
                    emptyMessage={eventStats.data?.empty_models_message}
                    tooltipFmt={eventStats.data?.model_tooltip_fmt}
                    locale={subscription.data?.money_locale}
                  />
                </CardContent>
              </Card>
              <Card className="min-w-0">
                <div className="overflow-hidden">
                  <FetchProgressBar active={eventStats.isFetching && !eventStats.isPending} />
                </div>
                <CardHeader className="space-y-1 pb-2">
                  <CardTitle className="text-base sm:text-lg">
                    {subscription.data?.chart_outcomes_title}
                  </CardTitle>
                </CardHeader>
                <CardContent className="min-w-0 overflow-visible pt-0">
                  <StatusBreakdownChart
                    breakdown={eventStats.data?.status_breakdown ?? []}
                    successRate={eventStats.data?.success_rate}
                    successCount={eventStats.data?.success_count}
                    chrome={{
                      empty_message: eventStats.data?.empty_status_message,
                      success_rate_prefix: eventStats.data?.success_rate_prefix,
                      runs_series_name: eventStats.data?.runs_series_name,
                      scope_full_label: eventStats.data?.scope_full_label,
                      scope_page_label: eventStats.data?.scope_page_label,
                      traces_unit: eventStats.data?.traces_unit,
                      status_success_label: eventStats.data?.status_success_label,
                      status_error_label: eventStats.data?.status_error_label,
                      status_success_code: eventStats.data?.status_success_code,
                      meta_sep: subscription.data?.meta_sep,
                    }}
                  />
                </CardContent>
              </Card>
              <Card className="min-w-0">
                <div className="overflow-hidden">
                  <FetchProgressBar active={eventStats.isFetching && !eventStats.isPending} />
                </div>
                <CardHeader className="space-y-1 pb-2">
                  <CardTitle className="text-base sm:text-lg">
                    {subscription.data?.chart_modes_title}
                  </CardTitle>
                </CardHeader>
                <CardContent className="min-w-0 overflow-visible pt-0">
                  <ModeBreakdownChart
                    breakdown={eventStats.data?.mode_breakdown ?? []}
                    emptyMessage={eventStats.data?.empty_modes_message}
                    runsFmt={eventStats.data?.runs_fmt}
                    locale={subscription.data?.money_locale}
                  />
                </CardContent>
              </Card>
            </>
          )}
        </div>

        <p className="mt-6 text-xs text-[var(--trim-muted)]">
          {email && subscription.data?.signed_in_prefix
            ? `${subscription.data.signed_in_prefix} ${email}`
            : ""}
          {eventStats.data && subscription.data?.events_suffix
            ? `${subscription.data.meta_sep || ""}${
                subscription.data.money_locale
                  ? eventStats.data.total_events.toLocaleString(subscription.data.money_locale)
                  : String(eventStats.data.total_events)
              } ${subscription.data.events_suffix}`
            : null}
        </p>

        <Card className="mt-8 border-[var(--trim-border)]">
          <CardHeader>
            <CardTitle className="text-sm font-medium text-[var(--trim-muted)]">
              {subscription.data?.account_title}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {actionNotice ? <p className="text-sm text-[var(--trim-fg)]">{actionNotice}</p> : null}
            {actionError ? <p className="text-sm text-destructive">{actionError}</p> : null}
            <p className="text-sm text-[var(--trim-muted)]">{subscription.data?.avatar_hint}</p>
            <Button
              variant="outline"
              isLoading={avatarSync.isPending}
              pendingLabel={subscription.data?.avatar_sync_pending_label || undefined}
              disabled={!accessToken || !subscription.data?.avatar_sync_action_label}
              onClick={() => {
                setActionNotice(null);
                setActionError(null);
                void avatarSync
                  .mutateAsync()
                  .then(async (res) => {
                    const supabase = createClient();
                    await supabase.auth.refreshSession();
                    setActionNotice(res.message || "");
                  })
                  .catch(() => {
                    setActionError(subscription.data?.avatar_sync_failed_message || "");
                  });
              }}
            >
              {subscription.data?.avatar_sync_action_label}
            </Button>
            <p className="text-sm text-[var(--trim-muted)]">{subscription.data?.delete_hint}</p>
            <Button
              variant="destructive"
              disabled={
                !accessToken ||
                !subscription.data?.account_delete_action_label ||
                !subscription.data?.account_delete_confirm_message ||
                !dialogCancel
              }
              onClick={() => setDeleteConfirmOpen(true)}
            >
              {subscription.data?.account_delete_action_label}
            </Button>
            <ConfirmDialog
              open={deleteConfirmOpen}
              onOpenChange={setDeleteConfirmOpen}
              description={subscription.data?.account_delete_confirm_message || ""}
              cancelLabel={dialogCancel}
              confirmLabel={subscription.data?.account_delete_action_label || ""}
              pendingLabel={subscription.data?.account_delete_pending_label || undefined}
              isPending={deleteAccount.isPending}
              destructive
              onConfirm={() => {
                void deleteAccount
                  .mutateAsync()
                  .then(async () => {
                    setDeleteConfirmOpen(false);
                    const supabase = createClient();
                    await supabase.auth.signOut();
                    const home =
                      (subscription.data?.path_home || "").trim() ||
                      (authProviders.data?.site?.path_home || "").trim();
                    if (home) {
                      router.push(home);
                    }
                  })
                  .catch(() => {
                    setDeleteConfirmOpen(false);
                    setActionError(subscription.data?.account_delete_failed_message || "");
                  });
              }}
            />
            {subscription.data?.privacy_link_label &&
            subscription.data?.terms_link_label &&
            subscription.data?.path_privacy &&
            subscription.data?.path_terms ? (
              <p className="text-xs text-[var(--trim-muted)]">
                <Link
                  href={subscription.data.path_privacy}
                  className="underline underline-offset-4"
                >
                  {subscription.data.privacy_link_label}
                </Link>
                {subscription.data.meta_sep || ""}
                <Link href={subscription.data.path_terms} className="underline underline-offset-4">
                  {subscription.data.terms_link_label}
                </Link>
              </p>
            ) : null}
          </CardContent>
        </Card>

        <PlanModal
          open={planModalOpen}
          onOpenChange={setPlanModalOpen}
          accessToken={accessToken}
          userId={userId}
          email={email}
        />
      </div>
    </div>
  );
}
