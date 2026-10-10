"use client";

import {
  DataTableSkeleton,
  FormPageSkeleton,
  TabsSkeleton,
} from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { type ColumnDef, DataTable } from "@/components/ui/data-table";
import { FetchProgressBar } from "@/components/ui/fetch-progress";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { usePatchProductSettings } from "@/hooks/mutations/product";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminProduct } from "@/hooks/queries/product";
import { useAdminToken } from "@/hooks/use-admin-token";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { useRequireStepUp } from "@/hooks/use-require-step-up";
import { chromeFieldDesc } from "@/lib/admin-field-desc";
import { useEffect, useMemo, useRef, useState } from "react";

function numField(v: unknown): string {
  if (typeof v === "number") return String(v);
  if (typeof v === "string" && v.trim()) return v;
  return "";
}

type RuntimeRow = { key: string; label: string; value: string };

export function ProductClient({ initialToken }: { initialToken?: string }) {
  const token = useAdminToken(initialToken);
  const chrome = useAdminNavChrome(token);
  const product = useAdminProduct(token);
  const patch = usePatchProductSettings(token);
  const { err: stepErr, requireStepUp } = useRequireStepUp(token);
  const hydratedRev = useRef("");
  const [mode, setMode] = useState("");
  const [engine, setEngine] = useState("");
  const [treesitter, setTreesitter] = useState(false);
  const [deepAttach, setDeepAttach] = useState(false);
  const [modelRouting, setModelRouting] = useState(false);
  const [rateIp, setRateIp] = useState("");
  const [rateUser, setRateUser] = useState("");
  const [pow, setPow] = useState("");
  const [maxHw, setMaxHw] = useState("");
  const [maxJa4, setMaxJa4] = useState("");
  const [cfThreat, setCfThreat] = useState("");
  const [cliNotice, setCliNotice] = useState("");
  const [minCli, setMinCli] = useState("");
  const [churnHighUsage, setChurnHighUsage] = useState("");
  const [churnMedUsage, setChurnMedUsage] = useState("");
  const [churnHighIdle, setChurnHighIdle] = useState("");
  const [churnLowIdle, setChurnLowIdle] = useState("");
  const [balancedMin, setBalancedMin] = useState("");
  const [aggressiveMin, setAggressiveMin] = useState("");
  const [mildMin, setMildMin] = useState("");
  const [queueWarnDepth, setQueueWarnDepth] = useState("");
  const title = chrome.data?.ADMIN_NAV_PRODUCT?.trim() || "";
  const pending =
    chrome.data?.ADMIN_PENDING_UPDATING?.trim() || chrome.data?.ADMIN_PENDING_SAVING?.trim() || "";
  const saveLabel = chrome.data?.ADMIN_ACTION_SAVE?.trim() || "";
  const editSectionLabel = chrome.data?.ADMIN_SECTION_EDIT?.trim() || "";
  const modeLabel = chrome.data?.ADMIN_PRODUCT_MODE_LABEL?.trim() || "";
  const engineLabel = chrome.data?.ADMIN_PRODUCT_ENGINE_LABEL?.trim() || "";
  const treesitterLabel = chrome.data?.ADMIN_PRODUCT_TREESITTER?.trim() || "";
  const deepAttachLabel = chrome.data?.ADMIN_PRODUCT_DEEP_ATTACH?.trim() || "";
  const modelRoutingLabel = chrome.data?.ADMIN_PRODUCT_MODEL_ROUTING?.trim() || "";
  const runtimeTitle = chrome.data?.ADMIN_PRODUCT_RUNTIME_TITLE?.trim() || "";
  const rateIpLabel = chrome.data?.ADMIN_PRODUCT_RATE_IP?.trim() || "";
  const rateUserLabel = chrome.data?.ADMIN_PRODUCT_RATE_USER?.trim() || "";
  const powLabel = chrome.data?.ADMIN_PRODUCT_POW?.trim() || "";
  const maxHwLabel = chrome.data?.ADMIN_PRODUCT_MAX_HW?.trim() || "";
  const maxJa4Label = chrome.data?.ADMIN_PRODUCT_MAX_JA4?.trim() || "";
  const cfLabel = chrome.data?.ADMIN_PRODUCT_CF_THREAT?.trim() || "";
  const cliNoticeLabel = chrome.data?.ADMIN_PRODUCT_CLI_NOTICE?.trim() || "";
  const minCliLabel = chrome.data?.ADMIN_PRODUCT_MIN_CLI?.trim() || "";
  const churnHighUsageLabel = chrome.data?.ADMIN_PRODUCT_CHURN_HIGH_USAGE?.trim() || "";
  const churnMedUsageLabel = chrome.data?.ADMIN_PRODUCT_CHURN_MED_USAGE?.trim() || "";
  const churnHighIdleLabel = chrome.data?.ADMIN_PRODUCT_CHURN_HIGH_IDLE?.trim() || "";
  const churnLowIdleLabel = chrome.data?.ADMIN_PRODUCT_CHURN_LOW_IDLE?.trim() || "";
  const balancedMinLabel = chrome.data?.ADMIN_PRODUCT_FAST_BALANCED_MIN?.trim() || "";
  const aggressiveMinLabel = chrome.data?.ADMIN_PRODUCT_FAST_AGGRESSIVE_MIN?.trim() || "";
  const mildMinLabel = chrome.data?.ADMIN_PRODUCT_FAST_MILD_MIN?.trim() || "";
  const queueWarnLabel = chrome.data?.ADMIN_PRODUCT_QUEUE_WARN_DEPTH?.trim() || "";
  const tabDefaults = chrome.data?.ADMIN_PRODUCT_TAB_DEFAULTS?.trim() || "";
  const tabLimits = chrome.data?.ADMIN_PRODUCT_TAB_LIMITS?.trim() || "";
  const tabCli = chrome.data?.ADMIN_PRODUCT_TAB_CLI?.trim() || "";
  const tabChurn = chrome.data?.ADMIN_PRODUCT_TAB_CHURN?.trim() || "";
  const tabFast = chrome.data?.ADMIN_PRODUCT_TAB_FAST?.trim() || "";
  const productSectionTabs = [tabDefaults, tabLimits, tabCli, tabChurn, tabFast].filter(Boolean);
  const useProductSectionTabs = productSectionTabs.length >= 2;
  const defaultProductSectionTab = tabDefaults
    ? "defaults"
    : tabLimits
      ? "limits"
      : tabCli
        ? "cli"
        : tabChurn
          ? "churn"
          : "fast";
  useEffect(() => {
    const d = product.data;
    if (!d) return;
    const rev = typeof d.updated_at === "string" ? d.updated_at : "";
    if (rev && hydratedRev.current === rev) return;
    if (!rev && hydratedRev.current === "*") return;
    hydratedRev.current = rev || "*";
    setMode(typeof d.default_compression_mode === "string" ? d.default_compression_mode : "");
    setEngine(typeof d.default_deep_engine === "string" ? d.default_deep_engine : "");
    setTreesitter(Boolean(d.treesitter_required));
    setDeepAttach(Boolean(d.deep_attach_default));
    setModelRouting(Boolean(d.model_routing_enabled));
    setRateIp(numField(d.rate_limit_ip_per_min));
    setRateUser(numField(d.rate_limit_user_per_min));
    setPow(numField(d.pow_difficulty));
    setMaxHw(numField(d.max_accounts_per_hardware));
    setMaxJa4(numField(d.max_accounts_per_ja4));
    setCfThreat(numField(d.cf_threat_score_min));
    setBalancedMin(numField(d.fast_balanced_min_lines));
    setAggressiveMin(numField(d.fast_aggressive_min_lines));
    setMildMin(numField(d.fast_mild_min_lines));
    setCliNotice(typeof d.cli_force_upgrade_notice === "string" ? d.cli_force_upgrade_notice : "");
    setMinCli(typeof d.min_cli_version === "string" ? d.min_cli_version : "");
    setChurnHighUsage(numField(d.churn_high_usage_ratio));
    setChurnMedUsage(numField(d.churn_medium_usage_ratio));
    setChurnHighIdle(numField(d.churn_high_idle_days));
    setChurnLowIdle(numField(d.churn_low_idle_days));
    setQueueWarnDepth(numField(d.paddle_webhook_queue_warn_depth));
  }, [product.data]);

  const runtimeRows = useMemo<RuntimeRow[]>(() => {
    const items = product.data?.runtime_items;
    if (!Array.isArray(items)) return [];
    const rows: RuntimeRow[] = [];
    for (const raw of items) {
      if (!raw || typeof raw !== "object") continue;
      const row = raw as Record<string, unknown>;
      const id = typeof row.id === "string" ? row.id.trim() : "";
      const label = typeof row.label === "string" ? row.label.trim() : "";
      const value =
        typeof row.value === "string"
          ? row.value.trim()
          : row.value != null
            ? String(row.value).trim()
            : "";
      if (!id || !label || !value) continue;
      rows.push({ key: id, label, value });
    }
    return rows;
  }, [product.data]);

  const runtimeColumns = useMemo<ColumnDef<RuntimeRow>[]>(
    () => [
      {
        id: "_label",
        header: "",
        cell: ({ row }) => <span className="text-muted-foreground">{row.original.label}</span>,
      },
      {
        id: "_value",
        header: "",
        cell: ({ row }) => <span className="tabular-nums">{row.original.value}</span>,
      },
    ],
    [],
  );

  const showRuntimeTable = Boolean(runtimeTitle && runtimeRows.length > 0);
  const showSettingsTabs = Boolean(editSectionLabel && showRuntimeTable);

  // Soft-nav: warm React Query cache must not flash a full-page skeleton.
  const bootReady = useListBootReady(1, Boolean(chrome.data), queryContentReady(product));
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    return (
      <div className="space-y-4">
        {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
        <TabsSkeleton count={2} />
        <TabsSkeleton count={productSectionTabs.length > 0 ? productSectionTabs.length : 5} />
        <FormPageSkeleton hideTitle fields={6} />
        <DataTableSkeleton
          title=""
          columns={[{ label: runtimeTitle || "" }, { label: "" }]}
          rows={6}
          showFilters={false}
        />
      </div>
    );
  }

  const saveButton =
    saveLabel && pending ? (
      <Button
        type="button"
        isLoading={patch.isPending}
        pendingLabel={pending}
        onClick={() => {
          if (!requireStepUp()) return;
          void patch.mutate({
            default_compression_mode: mode || undefined,
            default_deep_engine: engine || undefined,
            treesitter_required: treesitter,
            deep_attach_default: deepAttach,
            model_routing_enabled: modelRouting,
            rate_limit_ip_per_min: rateIp ? Number(rateIp) : undefined,
            rate_limit_user_per_min: rateUser ? Number(rateUser) : undefined,
            pow_difficulty: pow ? Number(pow) : undefined,
            max_accounts_per_hardware: maxHw ? Number(maxHw) : undefined,
            max_accounts_per_ja4: maxJa4 ? Number(maxJa4) : undefined,
            cf_threat_score_min: cfThreat ? Number(cfThreat) : undefined,
            fast_balanced_min_lines: balancedMin ? Number(balancedMin) : undefined,
            fast_aggressive_min_lines: aggressiveMin ? Number(aggressiveMin) : undefined,
            fast_mild_min_lines: mildMin ? Number(mildMin) : undefined,
            cli_force_upgrade_notice: cliNotice,
            min_cli_version: minCli,
            churn_high_usage_ratio: churnHighUsage ? Number(churnHighUsage) : undefined,
            churn_medium_usage_ratio: churnMedUsage ? Number(churnMedUsage) : undefined,
            churn_high_idle_days: churnHighIdle ? Number(churnHighIdle) : undefined,
            churn_low_idle_days: churnLowIdle ? Number(churnLowIdle) : undefined,
            paddle_webhook_queue_warn_depth: queueWarnDepth ? Number(queueWarnDepth) : undefined,
          });
        }}
      >
        {saveLabel}
      </Button>
    ) : null;

  const defaultsGrid = (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {modeLabel ? (
        <Field
          id="product-mode"
          label={modeLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_MODE_DESC")}
        >
          <Input
            id="product-mode"
            value={mode}
            onChange={(e) => setMode(e.target.value)}
            placeholder={modeLabel}
          />
        </Field>
      ) : null}
      {engineLabel ? (
        <Field
          id="product-engine"
          label={engineLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_ENGINE_DESC")}
        >
          <Input
            id="product-engine"
            value={engine}
            onChange={(e) => setEngine(e.target.value)}
            placeholder={engineLabel}
          />
        </Field>
      ) : null}
      {treesitterLabel ? (
        <Field
          id="product-treesitter"
          label={treesitterLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_TREESITTER_DESC")}
        >
          <div className="flex items-center gap-2 pt-1">
            <Checkbox
              id="product-treesitter"
              checked={treesitter}
              onCheckedChange={(v) => setTreesitter(v === true)}
            />
          </div>
        </Field>
      ) : null}
      {deepAttachLabel ? (
        <Field
          id="product-deep-attach"
          label={deepAttachLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_DEEP_ATTACH_DESC")}
        >
          <div className="flex items-center gap-2 pt-1">
            <Checkbox
              id="product-deep-attach"
              checked={deepAttach}
              onCheckedChange={(v) => setDeepAttach(v === true)}
            />
          </div>
        </Field>
      ) : null}
      {modelRoutingLabel ? (
        <Field
          id="product-model-routing"
          label={modelRoutingLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_MODEL_ROUTING_DESC")}
        >
          <div className="flex items-center gap-2 pt-1">
            <Checkbox
              id="product-model-routing"
              checked={modelRouting}
              onCheckedChange={(v) => setModelRouting(v === true)}
            />
          </div>
        </Field>
      ) : null}
    </div>
  );

  const limitsGrid = (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {rateIpLabel ? (
        <Field
          id="product-rate-ip"
          label={rateIpLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_RATE_IP_DESC")}
        >
          <Input
            id="product-rate-ip"
            value={rateIp}
            onChange={(e) => setRateIp(e.target.value)}
            placeholder={rateIpLabel}
          />
        </Field>
      ) : null}
      {rateUserLabel ? (
        <Field
          id="product-rate-user"
          label={rateUserLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_RATE_USER_DESC")}
        >
          <Input
            id="product-rate-user"
            value={rateUser}
            onChange={(e) => setRateUser(e.target.value)}
            placeholder={rateUserLabel}
          />
        </Field>
      ) : null}
      {powLabel ? (
        <Field
          id="product-pow"
          label={powLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_POW_DESC")}
        >
          <Input
            id="product-pow"
            value={pow}
            onChange={(e) => setPow(e.target.value)}
            placeholder={powLabel}
          />
        </Field>
      ) : null}
      {maxHwLabel ? (
        <Field
          id="product-max-hw"
          label={maxHwLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_MAX_HW_DESC")}
        >
          <Input
            id="product-max-hw"
            value={maxHw}
            onChange={(e) => setMaxHw(e.target.value)}
            placeholder={maxHwLabel}
          />
        </Field>
      ) : null}
      {maxJa4Label ? (
        <Field
          id="product-max-ja4"
          label={maxJa4Label}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_MAX_JA4_DESC")}
        >
          <Input
            id="product-max-ja4"
            value={maxJa4}
            onChange={(e) => setMaxJa4(e.target.value)}
            placeholder={maxJa4Label}
          />
        </Field>
      ) : null}
      {cfLabel ? (
        <Field
          id="product-cf"
          label={cfLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_CF_THREAT_DESC")}
        >
          <Input
            id="product-cf"
            value={cfThreat}
            onChange={(e) => setCfThreat(e.target.value)}
            placeholder={cfLabel}
          />
        </Field>
      ) : null}
      {queueWarnLabel ? (
        <Field
          id="product-queue-warn"
          label={queueWarnLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_QUEUE_WARN_DEPTH_DESC")}
        >
          <Input
            id="product-queue-warn"
            value={queueWarnDepth}
            onChange={(e) => setQueueWarnDepth(e.target.value)}
            placeholder={queueWarnLabel}
          />
        </Field>
      ) : null}
    </div>
  );

  const cliGrid = (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {cliNoticeLabel ? (
        <Field
          id="product-cli-notice"
          label={cliNoticeLabel}
          className="sm:col-span-2 xl:col-span-3"
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_CLI_NOTICE_DESC")}
        >
          <Textarea
            id="product-cli-notice"
            value={cliNotice}
            onChange={(e) => setCliNotice(e.target.value)}
            placeholder={cliNoticeLabel}
          />
        </Field>
      ) : null}
      {minCliLabel ? (
        <Field
          id="product-min-cli"
          label={minCliLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_MIN_CLI_DESC")}
        >
          <Input
            id="product-min-cli"
            value={minCli}
            onChange={(e) => setMinCli(e.target.value)}
            placeholder={minCliLabel}
            className="font-mono text-xs"
          />
        </Field>
      ) : null}
    </div>
  );

  const churnGrid = (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {churnHighUsageLabel ? (
        <Field
          id="product-churn-high-usage"
          label={churnHighUsageLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_CHURN_HIGH_USAGE_DESC")}
        >
          <Input
            id="product-churn-high-usage"
            value={churnHighUsage}
            onChange={(e) => setChurnHighUsage(e.target.value)}
            placeholder={churnHighUsageLabel}
          />
        </Field>
      ) : null}
      {churnMedUsageLabel ? (
        <Field
          id="product-churn-med-usage"
          label={churnMedUsageLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_CHURN_MED_USAGE_DESC")}
        >
          <Input
            id="product-churn-med-usage"
            value={churnMedUsage}
            onChange={(e) => setChurnMedUsage(e.target.value)}
            placeholder={churnMedUsageLabel}
          />
        </Field>
      ) : null}
      {churnHighIdleLabel ? (
        <Field
          id="product-churn-high-idle"
          label={churnHighIdleLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_CHURN_HIGH_IDLE_DESC")}
        >
          <Input
            id="product-churn-high-idle"
            value={churnHighIdle}
            onChange={(e) => setChurnHighIdle(e.target.value)}
            placeholder={churnHighIdleLabel}
          />
        </Field>
      ) : null}
      {churnLowIdleLabel ? (
        <Field
          id="product-churn-low-idle"
          label={churnLowIdleLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_CHURN_LOW_IDLE_DESC")}
        >
          <Input
            id="product-churn-low-idle"
            value={churnLowIdle}
            onChange={(e) => setChurnLowIdle(e.target.value)}
            placeholder={churnLowIdleLabel}
          />
        </Field>
      ) : null}
    </div>
  );

  const fastGrid = (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {balancedMinLabel ? (
        <Field
          id="product-balanced-min"
          label={balancedMinLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_FAST_BALANCED_MIN_DESC")}
        >
          <Input
            id="product-balanced-min"
            value={balancedMin}
            onChange={(e) => setBalancedMin(e.target.value)}
            placeholder={balancedMinLabel}
          />
        </Field>
      ) : null}
      {aggressiveMinLabel ? (
        <Field
          id="product-aggressive-min"
          label={aggressiveMinLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_FAST_AGGRESSIVE_MIN_DESC")}
        >
          <Input
            id="product-aggressive-min"
            value={aggressiveMin}
            onChange={(e) => setAggressiveMin(e.target.value)}
            placeholder={aggressiveMinLabel}
          />
        </Field>
      ) : null}
      {mildMinLabel ? (
        <Field
          id="product-mild-min"
          label={mildMinLabel}
          {...chromeFieldDesc(chrome.data, "ADMIN_PRODUCT_FAST_MILD_MIN_DESC")}
        >
          <Input
            id="product-mild-min"
            value={mildMin}
            onChange={(e) => setMildMin(e.target.value)}
            placeholder={mildMinLabel}
          />
        </Field>
      ) : null}
    </div>
  );

  const stackedSettingsFields = (
    <div className="space-y-6">
      {defaultsGrid}
      {limitsGrid}
      {cliGrid}
      {churnGrid}
      {fastGrid}
    </div>
  );

  const tabbedSettingsFields = (
    <Tabs defaultValue={defaultProductSectionTab}>
      <TabsList>
        {tabDefaults ? <TabsTrigger value="defaults">{tabDefaults}</TabsTrigger> : null}
        {tabLimits ? <TabsTrigger value="limits">{tabLimits}</TabsTrigger> : null}
        {tabCli ? <TabsTrigger value="cli">{tabCli}</TabsTrigger> : null}
        {tabChurn ? <TabsTrigger value="churn">{tabChurn}</TabsTrigger> : null}
        {tabFast ? <TabsTrigger value="fast">{tabFast}</TabsTrigger> : null}
      </TabsList>
      {tabDefaults ? (
        <TabsContent value="defaults" className="mt-4">
          {defaultsGrid}
        </TabsContent>
      ) : null}
      {tabLimits ? (
        <TabsContent value="limits" className="mt-4">
          {limitsGrid}
        </TabsContent>
      ) : null}
      {tabCli ? (
        <TabsContent value="cli" className="mt-4">
          {cliGrid}
        </TabsContent>
      ) : null}
      {tabChurn ? (
        <TabsContent value="churn" className="mt-4">
          {churnGrid}
        </TabsContent>
      ) : null}
      {tabFast ? (
        <TabsContent value="fast" className="mt-4">
          {fastGrid}
        </TabsContent>
      ) : null}
    </Tabs>
  );

  const settingsForm = (
    <div className="space-y-6">
      {!showSettingsTabs && editSectionLabel && !useProductSectionTabs ? (
        <p className="text-sm font-medium text-foreground">{editSectionLabel}</p>
      ) : null}
      {useProductSectionTabs ? tabbedSettingsFields : stackedSettingsFields}
      {saveButton}
    </div>
  );

  const runtimeTable = showRuntimeTable ? (
    <DataTable
      columns={runtimeColumns}
      data={runtimeRows}
      getRowId={(row) => row.key}
      isFetching={product.isFetching && !product.isPending}
      bordered={false}
    />
  ) : null;

  return (
    <div className="space-y-4">
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      {stepErr ? <p className="text-sm text-destructive">{stepErr}</p> : null}
      {showSettingsTabs ? (
        <Tabs defaultValue="edit">
          <TabsList>
            <TabsTrigger value="edit">{editSectionLabel}</TabsTrigger>
            <TabsTrigger value="runtime">{runtimeTitle}</TabsTrigger>
          </TabsList>
          <TabsContent value="edit" className="mt-4">
            <Card className="overflow-hidden bg-card">
              <FetchProgressBar
                active={(product.isFetching && !product.isPending) || patch.isPending}
              />
              <CardContent className="pt-6">{settingsForm}</CardContent>
            </Card>
          </TabsContent>
          <TabsContent value="runtime" className="mt-4">
            <Card>
              <CardContent className="pt-6">{runtimeTable}</CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      ) : (
        <>
          <Card className="overflow-hidden bg-card">
            <FetchProgressBar
              active={(product.isFetching && !product.isPending) || patch.isPending}
            />
            <CardContent className="pt-6">{settingsForm}</CardContent>
          </Card>
          {showRuntimeTable ? (
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">{runtimeTitle}</CardTitle>
              </CardHeader>
              <CardContent>{runtimeTable}</CardContent>
            </Card>
          ) : null}
        </>
      )}
    </div>
  );
}
