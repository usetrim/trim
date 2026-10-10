"use client";

import { FormPageSkeleton, TabsSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { FetchProgressBar } from "@/components/ui/fetch-progress";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { usePatchBillingSettings } from "@/hooks/mutations/billing";
import { useAdminBillingSettings } from "@/hooks/queries/billing";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminToken } from "@/hooks/use-admin-token";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { useRequireStepUp } from "@/hooks/use-require-step-up";
import { chromeFieldDesc } from "@/lib/admin-field-desc";
import { useEffect, useRef, useState } from "react";

function numField(v: unknown): string {
  if (typeof v === "number") return String(v);
  if (typeof v === "string" && v.trim()) return v;
  return "";
}

function strField(v: unknown): string {
  return typeof v === "string" ? v : "";
}

export function BillingSettingsClient({
  initialToken,
}: {
  initialToken?: string;
}) {
  const token = useAdminToken(initialToken);
  const chrome = useAdminNavChrome(token);
  const settings = useAdminBillingSettings(token);
  const { err: stepErr, requireStepUp } = useRequireStepUp(token);
  const patch = usePatchBillingSettings(token);
  const hydratedRev = useRef("");

  const [annualDiscount, setAnnualDiscount] = useState("");
  const [currency, setCurrency] = useState("");
  const [defaultPageSize, setDefaultPageSize] = useState("");
  const [maxPageSize, setMaxPageSize] = useState("");
  const [skipToMax, setSkipToMax] = useState("");
  const [chartTopN, setChartTopN] = useState("");
  const [chartDays, setChartDays] = useState("");
  const [chartCacheTtl, setChartCacheTtl] = useState("");
  const [dateRangeMonths, setDateRangeMonths] = useState("");
  const [planInterval, setPlanInterval] = useState("");
  const [pricingBound, setPricingBound] = useState(false);
  const [cancelAtPeriodEnd, setCancelAtPeriodEnd] = useState(false);
  const [defaultSeats, setDefaultSeats] = useState("");
  const [minSeats, setMinSeats] = useState("");
  const [deepDefault, setDeepDefault] = useState("");
  const [deepMin, setDeepMin] = useState("");
  const [deepMax, setDeepMax] = useState("");
  const [liveDeepMin, setLiveDeepMin] = useState("");
  const [liveDeepOom, setLiveDeepOom] = useState("");
  const [liveDeepWarmup, setLiveDeepWarmup] = useState(false);
  const [liveDeepSkipStream, setLiveDeepSkipStream] = useState(false);
  const [deepV1Model, setDeepV1Model] = useState("");
  const [deepV2Model, setDeepV2Model] = useState("");
  const [deepLongModel, setDeepLongModel] = useState("");
  const [deepV2ForceTokens, setDeepV2ForceTokens] = useState("");
  const [allowDowngrades, setAllowDowngrades] = useState(false);
  const [prorationMode, setProrationMode] = useState("");
  const [monthlyToAnnual, setMonthlyToAnnual] = useState(false);

  const title = chrome.data?.ADMIN_NAV_SETTINGS_BILLING?.trim() || "";
  const pending =
    chrome.data?.ADMIN_PENDING_UPDATING?.trim() || chrome.data?.ADMIN_PENDING_SAVING?.trim() || "";
  const saveLabel = chrome.data?.ADMIN_ACTION_SAVE?.trim() || "";
  const editSectionLabel = chrome.data?.ADMIN_SECTION_EDIT?.trim() || "";
  const boundLabel = chrome.data?.ADMIN_BILLING_PRICING_BOUND?.trim() || "";
  const boundHint = chrome.data?.ADMIN_BILLING_PRICING_BOUND_HINT?.trim() || "";
  const unboundLabel = chrome.data?.ADMIN_PRICING_UNBOUND?.trim() || "";
  const cancelLabel = chrome.data?.ADMIN_BILLING_CANCEL_AT_PERIOD_END?.trim() || "";
  const annualLabel = chrome.data?.ADMIN_BILLING_ANNUAL_DISCOUNT?.trim() || "";
  const annualHint = chrome.data?.ADMIN_BILLING_ANNUAL_DISCOUNT_HINT?.trim() || "";
  const currencyLabel = chrome.data?.ADMIN_BILLING_CURRENCY?.trim() || "";
  const pageSizeLabel = chrome.data?.ADMIN_BILLING_DEFAULT_PAGE_SIZE?.trim() || "";
  const maxPageLabel = chrome.data?.ADMIN_BILLING_MAX_PAGE_SIZE?.trim() || "";
  const seatsLabel = chrome.data?.ADMIN_BILLING_DEFAULT_SEATS?.trim() || "";
  const minSeatsLabel = chrome.data?.ADMIN_BILLING_MIN_SEATS?.trim() || "";
  const deepDefLabel = chrome.data?.ADMIN_BILLING_DEEP_TARGET_DEFAULT?.trim() || "";
  const deepMinLabel = chrome.data?.ADMIN_BILLING_DEEP_TARGET_MIN?.trim() || "";
  const deepMaxLabel = chrome.data?.ADMIN_BILLING_DEEP_TARGET_MAX?.trim() || "";
  const liveDeepMinLabel = chrome.data?.ADMIN_BILLING_LIVE_DEEP_MIN?.trim() || "";
  const liveDeepOomLabel = chrome.data?.ADMIN_BILLING_LIVE_DEEP_OOM?.trim() || "";
  const liveDeepWarmupLabel = chrome.data?.ADMIN_BILLING_LIVE_DEEP_WARMUP?.trim() || "";
  const liveDeepSkipStreamLabel = chrome.data?.ADMIN_BILLING_LIVE_DEEP_SKIP_STREAM?.trim() || "";
  const deepV1Label = chrome.data?.ADMIN_BILLING_DEEP_V1_MODEL?.trim() || "";
  const deepV2Label = chrome.data?.ADMIN_BILLING_DEEP_V2_MODEL?.trim() || "";
  const deepLongLabel = chrome.data?.ADMIN_BILLING_DEEP_LONG_MODEL?.trim() || "";
  const deepForceLabel = chrome.data?.ADMIN_BILLING_DEEP_V2_FORCE_TOKENS?.trim() || "";
  const downgradeLabel = chrome.data?.ADMIN_BILLING_ALLOW_DOWNGRADES?.trim() || "";
  const prorationLabel = chrome.data?.ADMIN_BILLING_PRORATION_MODE?.trim() || "";
  const monthlyAnnualLabel = chrome.data?.ADMIN_BILLING_MONTHLY_TO_ANNUAL?.trim() || "";
  const skipLabel = chrome.data?.ADMIN_BILLING_SKIP_TO_MAX?.trim() || "";
  const chartTopLabel = chrome.data?.ADMIN_BILLING_CHART_TOP_N?.trim() || "";
  const chartDaysLabel = chrome.data?.ADMIN_BILLING_CHART_DAYS?.trim() || "";
  const chartCacheTtlLabel = chrome.data?.ADMIN_BILLING_CHART_CACHE_TTL?.trim() || "";
  const dateRangeLabel = chrome.data?.ADMIN_BILLING_DATE_RANGE_MONTHS?.trim() || "";
  const intervalLabel = chrome.data?.ADMIN_BILLING_PLAN_INTERVAL?.trim() || "";
  const tabPricing = chrome.data?.ADMIN_BILLING_TAB_PRICING?.trim() || "";
  const tabDeep = chrome.data?.ADMIN_BILLING_TAB_DEEP?.trim() || "";
  const tabLists = chrome.data?.ADMIN_BILLING_TAB_LISTS?.trim() || "";
  const tabPolicy = chrome.data?.ADMIN_BILLING_TAB_POLICY?.trim() || "";
  const billingTabLabels = [tabPricing, tabDeep, tabLists, tabPolicy].filter(Boolean);
  const useBillingTabs = billingTabLabels.length >= 2;
  const defaultBillingTab = tabPricing
    ? "pricing"
    : tabDeep
      ? "deep"
      : tabLists
        ? "lists"
        : "policy";

  const prorationOptions = (
    [
      ["prorated_immediately", chrome.data?.ADMIN_PRORATION_PRORATED_IMMEDIATELY],
      ["full_immediately", chrome.data?.ADMIN_PRORATION_FULL_IMMEDIATELY],
      ["prorated_next_billing_period", chrome.data?.ADMIN_PRORATION_PRORATED_NEXT],
      ["full_next_billing_period", chrome.data?.ADMIN_PRORATION_FULL_NEXT],
      ["do_not_bill", chrome.data?.ADMIN_PRORATION_DO_NOT_BILL],
    ] as const
  ).filter(([, label]) => Boolean(label?.trim()));

  useEffect(() => {
    const d = settings.data;
    if (!d) return;
    const rev = typeof d.updated_at === "string" ? d.updated_at : "";
    if (rev && hydratedRev.current === rev) return;
    if (!rev && hydratedRev.current === "*") return;
    hydratedRev.current = rev || "*";
    setAnnualDiscount(numField(d.annual_discount_percent));
    setCurrency(strField(d.default_currency));
    setDefaultPageSize(numField(d.default_page_size));
    setMaxPageSize(numField(d.max_page_size));
    setSkipToMax(numField(d.pagination_skip_to_max_pages));
    setChartTopN(numField(d.chart_top_n));
    setChartDays(numField(d.chart_series_days));
    setChartCacheTtl(numField(d.chart_cache_ttl_sec));
    setDateRangeMonths(numField(d.date_range_months));
    setPlanInterval(strField(d.default_plan_interval));
    setPricingBound(Boolean(d.pricing_bound));
    setCancelAtPeriodEnd(Boolean(d.allow_cancel_at_period_end));
    setDefaultSeats(numField(d.default_seat_quantity));
    setMinSeats(numField(d.min_seat_quantity));
    setDeepDefault(numField(d.default_deep_target_token));
    setDeepMin(numField(d.deep_target_token_min));
    setDeepMax(numField(d.deep_target_token_max));
    setLiveDeepMin(numField(d.live_deep_min_input_tokens));
    setLiveDeepOom(strField(d.live_deep_oom_policy));
    setLiveDeepWarmup(Boolean(d.live_deep_warmup_on_start));
    setLiveDeepSkipStream(Boolean(d.live_deep_skip_on_stream));
    setDeepV1Model(strField(d.deep_v1_model));
    setDeepV2Model(strField(d.deep_v2_model));
    setDeepLongModel(strField(d.deep_long_model));
    setDeepV2ForceTokens(
      Array.isArray(d.deep_v2_force_tokens)
        ? JSON.stringify(d.deep_v2_force_tokens)
        : strField(d.deep_v2_force_tokens),
    );
    setAllowDowngrades(Boolean(d.allow_downgrades));
    setProrationMode(strField(d.upgrade_proration_mode));
    setMonthlyToAnnual(Boolean(d.allow_monthly_to_annual_as_upgrade));
  }, [settings.data]);

  const bootReady = useListBootReady(1, Boolean(chrome.data), queryContentReady(settings));
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    return (
      <div className="space-y-4">
        {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
        <TabsSkeleton count={billingTabLabels.length >= 2 ? billingTabLabels.length : 4} />
        <FormPageSkeleton hideTitle fields={8} />
      </div>
    );
  }

  const pricingGrid = (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {annualLabel ? (
        <Field id="billing-annual" label={annualLabel} description={annualHint}>
          <Input
            id="billing-annual"
            value={annualDiscount}
            onChange={(e) => setAnnualDiscount(e.target.value)}
            placeholder={annualLabel}
          />
        </Field>
      ) : null}
      {currencyLabel ? (
        <Field
          id="billing-currency"
          label={currencyLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_CURRENCY_DESC")}
        >
          <Input
            id="billing-currency"
            value={currency}
            onChange={(e) => setCurrency(e.target.value)}
            placeholder={currencyLabel}
          />
        </Field>
      ) : null}
      {intervalLabel ? (
        <Field
          id="billing-interval"
          label={intervalLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_PLAN_INTERVAL_DESC")}
        >
          <Input
            id="billing-interval"
            value={planInterval}
            onChange={(e) => setPlanInterval(e.target.value)}
            placeholder={intervalLabel}
          />
        </Field>
      ) : null}
      {boundLabel || unboundLabel ? (
        <div className="space-y-1 text-sm text-muted-foreground sm:col-span-2 xl:col-span-3">
          {pricingBound ? (
            <>
              {boundLabel ? <p className="text-foreground">{boundLabel}</p> : null}
              {boundHint ? <p className="text-xs text-muted-foreground">{boundHint}</p> : null}
            </>
          ) : unboundLabel ? (
            <p className="text-foreground">{unboundLabel}</p>
          ) : null}
        </div>
      ) : null}
      {seatsLabel ? (
        <Field
          id="billing-seats"
          label={seatsLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_DEFAULT_SEATS_DESC")}
        >
          <Input
            id="billing-seats"
            value={defaultSeats}
            onChange={(e) => setDefaultSeats(e.target.value)}
            placeholder={seatsLabel}
          />
        </Field>
      ) : null}
      {minSeatsLabel ? (
        <Field
          id="billing-min-seats"
          label={minSeatsLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_MIN_SEATS_DESC")}
        >
          <Input
            id="billing-min-seats"
            value={minSeats}
            onChange={(e) => setMinSeats(e.target.value)}
            placeholder={minSeatsLabel}
          />
        </Field>
      ) : null}
    </div>
  );

  const deepGrid = (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {deepDefLabel ? (
        <Field
          id="billing-deep-default"
          label={deepDefLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_DEEP_TARGET_DEFAULT_DESC")}
        >
          <Input
            id="billing-deep-default"
            value={deepDefault}
            onChange={(e) => setDeepDefault(e.target.value)}
            placeholder={deepDefLabel}
          />
        </Field>
      ) : null}
      {deepMinLabel ? (
        <Field
          id="billing-deep-min"
          label={deepMinLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_DEEP_TARGET_MIN_DESC")}
        >
          <Input
            id="billing-deep-min"
            value={deepMin}
            onChange={(e) => setDeepMin(e.target.value)}
            placeholder={deepMinLabel}
          />
        </Field>
      ) : null}
      {deepMaxLabel ? (
        <Field
          id="billing-deep-max"
          label={deepMaxLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_DEEP_TARGET_MAX_DESC")}
        >
          <Input
            id="billing-deep-max"
            value={deepMax}
            onChange={(e) => setDeepMax(e.target.value)}
            placeholder={deepMaxLabel}
          />
        </Field>
      ) : null}
      {liveDeepMinLabel ? (
        <Field
          id="billing-live-deep-min"
          label={liveDeepMinLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_LIVE_DEEP_MIN_DESC")}
        >
          <Input
            id="billing-live-deep-min"
            value={liveDeepMin}
            onChange={(e) => setLiveDeepMin(e.target.value)}
            placeholder={liveDeepMinLabel}
          />
        </Field>
      ) : null}
      {liveDeepOomLabel ? (
        <Field
          id="billing-live-deep-oom"
          label={liveDeepOomLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_LIVE_DEEP_OOM_DESC")}
        >
          <Input
            id="billing-live-deep-oom"
            value={liveDeepOom}
            onChange={(e) => setLiveDeepOom(e.target.value)}
            placeholder={liveDeepOomLabel}
          />
        </Field>
      ) : null}
      {liveDeepWarmupLabel ? (
        <label className="flex items-center gap-2 text-sm text-[var(--trim-fg)] sm:col-span-2">
          <input
            type="checkbox"
            checked={liveDeepWarmup}
            onChange={(e) => setLiveDeepWarmup(e.target.checked)}
          />
          <span>{liveDeepWarmupLabel}</span>
        </label>
      ) : null}
      {liveDeepSkipStreamLabel ? (
        <label className="flex items-center gap-2 text-sm text-[var(--trim-fg)] sm:col-span-2">
          <input
            type="checkbox"
            checked={liveDeepSkipStream}
            onChange={(e) => setLiveDeepSkipStream(e.target.checked)}
          />
          <span>{liveDeepSkipStreamLabel}</span>
        </label>
      ) : null}
      {deepV1Label ? (
        <Field id="billing-deep-v1" label={deepV1Label}>
          <Input
            id="billing-deep-v1"
            value={deepV1Model}
            onChange={(e) => setDeepV1Model(e.target.value)}
            placeholder={deepV1Label}
          />
        </Field>
      ) : null}
      {deepV2Label ? (
        <Field id="billing-deep-v2" label={deepV2Label}>
          <Input
            id="billing-deep-v2"
            value={deepV2Model}
            onChange={(e) => setDeepV2Model(e.target.value)}
            placeholder={deepV2Label}
          />
        </Field>
      ) : null}
      {deepLongLabel ? (
        <Field id="billing-deep-long" label={deepLongLabel}>
          <Input
            id="billing-deep-long"
            value={deepLongModel}
            onChange={(e) => setDeepLongModel(e.target.value)}
            placeholder={deepLongLabel}
          />
        </Field>
      ) : null}
      {deepForceLabel ? (
        <Field id="billing-deep-force" label={deepForceLabel}>
          <Input
            id="billing-deep-force"
            value={deepV2ForceTokens}
            onChange={(e) => setDeepV2ForceTokens(e.target.value)}
            placeholder={deepForceLabel}
          />
        </Field>
      ) : null}
    </div>
  );

  const listsGrid = (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {pageSizeLabel ? (
        <Field
          id="billing-page-size"
          label={pageSizeLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_DEFAULT_PAGE_SIZE_DESC")}
        >
          <Input
            id="billing-page-size"
            value={defaultPageSize}
            onChange={(e) => setDefaultPageSize(e.target.value)}
            placeholder={pageSizeLabel}
          />
        </Field>
      ) : null}
      {maxPageLabel ? (
        <Field
          id="billing-max-page"
          label={maxPageLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_MAX_PAGE_SIZE_DESC")}
        >
          <Input
            id="billing-max-page"
            value={maxPageSize}
            onChange={(e) => setMaxPageSize(e.target.value)}
            placeholder={maxPageLabel}
          />
        </Field>
      ) : null}
      {skipLabel ? (
        <Field
          id="billing-skip"
          label={skipLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_SKIP_TO_MAX_DESC")}
        >
          <Input
            id="billing-skip"
            value={skipToMax}
            onChange={(e) => setSkipToMax(e.target.value)}
            placeholder={skipLabel}
          />
        </Field>
      ) : null}
      {chartTopLabel ? (
        <Field
          id="billing-chart-top"
          label={chartTopLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_CHART_TOP_N_DESC")}
        >
          <Input
            id="billing-chart-top"
            value={chartTopN}
            onChange={(e) => setChartTopN(e.target.value)}
            placeholder={chartTopLabel}
          />
        </Field>
      ) : null}
      {chartDaysLabel ? (
        <Field
          id="billing-chart-days"
          label={chartDaysLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_CHART_DAYS_DESC")}
        >
          <Input
            id="billing-chart-days"
            value={chartDays}
            onChange={(e) => setChartDays(e.target.value)}
            placeholder={chartDaysLabel}
          />
        </Field>
      ) : null}
      {chartCacheTtlLabel ? (
        <Field
          id="billing-chart-cache"
          label={chartCacheTtlLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_CHART_CACHE_TTL_DESC")}
        >
          <Input
            id="billing-chart-cache"
            value={chartCacheTtl}
            onChange={(e) => setChartCacheTtl(e.target.value)}
            placeholder={chartCacheTtlLabel}
          />
        </Field>
      ) : null}
      {dateRangeLabel ? (
        <Field
          id="billing-date-range"
          label={dateRangeLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_DATE_RANGE_MONTHS_DESC")}
        >
          <Input
            id="billing-date-range"
            value={dateRangeMonths}
            onChange={(e) => setDateRangeMonths(e.target.value)}
            placeholder={dateRangeLabel}
          />
        </Field>
      ) : null}
    </div>
  );

  const policyGrid = (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {prorationLabel && prorationOptions.length > 0 ? (
        <Field
          id="billing-proration"
          label={prorationLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_PRORATION_MODE_DESC")}
        >
          <Select value={prorationMode || undefined} onValueChange={(v) => setProrationMode(v)}>
            <SelectTrigger id="billing-proration" aria-label={prorationLabel}>
              <SelectValue placeholder={prorationLabel} />
            </SelectTrigger>
            <SelectContent>
              {prorationOptions.map(([value, label]) => (
                <SelectItem key={value} value={value}>
                  {label?.trim()}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
      ) : null}
      {cancelLabel ? (
        <Field
          id="billing-cancel"
          label={cancelLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_CANCEL_AT_PERIOD_END_DESC")}
        >
          <div className="flex items-center gap-2 pt-1">
            <Checkbox
              id="billing-cancel"
              checked={cancelAtPeriodEnd}
              onCheckedChange={(v) => setCancelAtPeriodEnd(v === true)}
            />
          </div>
        </Field>
      ) : null}
      {downgradeLabel ? (
        <Field
          id="billing-downgrade"
          label={downgradeLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_ALLOW_DOWNGRADES_DESC")}
        >
          <div className="flex items-center gap-2 pt-1">
            <Checkbox
              id="billing-downgrade"
              checked={allowDowngrades}
              onCheckedChange={(v) => setAllowDowngrades(v === true)}
            />
          </div>
        </Field>
      ) : null}
      {monthlyAnnualLabel ? (
        <Field
          id="billing-monthly-annual"
          label={monthlyAnnualLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_BILLING_MONTHLY_TO_ANNUAL_DESC")}
        >
          <div className="flex items-center gap-2 pt-1">
            <Checkbox
              id="billing-monthly-annual"
              checked={monthlyToAnnual}
              onCheckedChange={(v) => setMonthlyToAnnual(v === true)}
            />
          </div>
        </Field>
      ) : null}
    </div>
  );

  const settingsFields = useBillingTabs ? (
    <Tabs defaultValue={defaultBillingTab}>
      <TabsList>
        {tabPricing ? <TabsTrigger value="pricing">{tabPricing}</TabsTrigger> : null}
        {tabDeep ? <TabsTrigger value="deep">{tabDeep}</TabsTrigger> : null}
        {tabLists ? <TabsTrigger value="lists">{tabLists}</TabsTrigger> : null}
        {tabPolicy ? <TabsTrigger value="policy">{tabPolicy}</TabsTrigger> : null}
      </TabsList>
      {tabPricing ? (
        <TabsContent value="pricing" className="mt-4">
          {pricingGrid}
        </TabsContent>
      ) : null}
      {tabDeep ? (
        <TabsContent value="deep" className="mt-4">
          {deepGrid}
        </TabsContent>
      ) : null}
      {tabLists ? (
        <TabsContent value="lists" className="mt-4">
          {listsGrid}
        </TabsContent>
      ) : null}
      {tabPolicy ? (
        <TabsContent value="policy" className="mt-4">
          {policyGrid}
        </TabsContent>
      ) : null}
    </Tabs>
  ) : (
    <div className="space-y-6">
      {pricingGrid}
      {deepGrid}
      {listsGrid}
      {policyGrid}
    </div>
  );

  return (
    <div className="space-y-4">
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      {stepErr ? <p className="text-sm text-destructive">{stepErr}</p> : null}
      <Card className="overflow-hidden bg-card">
        <FetchProgressBar
          active={(settings.isFetching && !settings.isPending) || patch.isPending}
        />
        <CardContent className="space-y-4 pt-6">
          {editSectionLabel && !useBillingTabs ? (
            <p className="text-sm font-medium text-foreground">{editSectionLabel}</p>
          ) : null}
          {settingsFields}
          {saveLabel && pending ? (
            <div className="mt-4">
              <Button
                type="button"
                isLoading={patch.isPending}
                pendingLabel={pending}
                onClick={() => {
                  if (!requireStepUp()) return;
                  void patch.mutate({
                    annual_discount_percent: annualDiscount ? Number(annualDiscount) : undefined,
                    default_currency: currency || undefined,
                    default_page_size: defaultPageSize ? Number(defaultPageSize) : undefined,
                    max_page_size: maxPageSize ? Number(maxPageSize) : undefined,
                    pagination_skip_to_max_pages: skipToMax ? Number(skipToMax) : undefined,
                    chart_top_n: chartTopN ? Number(chartTopN) : undefined,
                    chart_series_days: chartDays ? Number(chartDays) : undefined,
                    chart_cache_ttl_sec: chartCacheTtl ? Number(chartCacheTtl) : undefined,
                    date_range_months: dateRangeMonths ? Number(dateRangeMonths) : undefined,
                    default_plan_interval: planInterval || undefined,
                    default_seat_quantity: defaultSeats ? Number(defaultSeats) : undefined,
                    min_seat_quantity: minSeats ? Number(minSeats) : undefined,
                    default_deep_target_token: deepDefault ? Number(deepDefault) : undefined,
                    deep_target_token_min: deepMin ? Number(deepMin) : undefined,
                    deep_target_token_max: deepMax ? Number(deepMax) : undefined,
                    live_deep_min_input_tokens:
                      liveDeepMin !== "" ? Number(liveDeepMin) : undefined,
                    live_deep_oom_policy: liveDeepOom || undefined,
                    live_deep_warmup_on_start: liveDeepWarmup,
                    live_deep_skip_on_stream: liveDeepSkipStream,
                    deep_v1_model: deepV1Model || undefined,
                    deep_v2_model: deepV2Model || undefined,
                    deep_long_model: deepLongModel || undefined,
                    deep_v2_force_tokens: (() => {
                      if (!deepV2ForceTokens.trim()) return undefined;
                      try {
                        return JSON.parse(deepV2ForceTokens) as string[];
                      } catch {
                        return undefined;
                      }
                    })(),
                    allow_cancel_at_period_end: cancelAtPeriodEnd,
                    allow_downgrades: allowDowngrades,
                    upgrade_proration_mode: prorationMode || undefined,
                    allow_monthly_to_annual_as_upgrade: monthlyToAnnual,
                  });
                }}
              >
                {saveLabel}
              </Button>
            </div>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
