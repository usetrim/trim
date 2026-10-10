"use client";

import { AdminFilterBar } from "@/components/admin/admin-filter-bar";
import { MetricsPlusTableSkeleton } from "@/components/skeletons/page-skeletons";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { type ColumnDef, DataTable } from "@/components/ui/data-table";
import { DataTableRowActions } from "@/components/ui/data-table-row-actions";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useWebhookReplay } from "@/hooks/mutations/observability";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import {
  useAdminHeatmap,
  useAdminObservability,
  useAdminObservabilityWebhook,
  useAdminObservabilityWebhooks,
} from "@/hooks/queries/observability";
import { useDeferredDialogSelection } from "@/hooks/use-deferred-dialog-selection";
import { useAdminToken } from "@/hooks/use-admin-token";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { useRequireStepUp } from "@/hooks/use-require-step-up";
import { useEffect, useMemo, useState } from "react";

export function ObservabilityClient({
  initialToken,
}: {
  initialToken?: string;
}) {
  const token = useAdminToken(initialToken);
  const pageSize = useDefaultPageSize();
  const [skip, setSkip] = useState(0);
  const [qInput, setQInput] = useState("");
  const q = useDebouncedValue(qInput.trim(), 250);
  const {
    open: eventOpen,
    selectedId: selectedEventId,
    openWith: openEvent,
    close: closeEvent,
    onOpenChange: onEventOpenChange,
  } = useDeferredDialogSelection();
  const chrome = useAdminNavChrome(token);
  const stats = useAdminObservability(token);
  const heatmap = useAdminHeatmap(token);
  const webhooks = useAdminObservabilityWebhooks(token, skip, pageSize, q);
  const webhookDetail = useAdminObservabilityWebhook(token, selectedEventId);
  const replay = useWebhookReplay(token);
  const { err: stepErr, requireStepUp } = useRequireStepUp(token);

  const title = chrome.data?.ADMIN_NAV_OBSERVABILITY?.trim() || "";
  const pending =
    chrome.data?.ADMIN_PENDING_UPDATING?.trim() || chrome.data?.ADMIN_PENDING_SAVING?.trim() || "";
  const filterSearch = chrome.data?.ADMIN_FILTER_SEARCH?.trim() || "";
  const filterSearchDesc = chrome.data?.ADMIN_FILTER_SEARCH_DESC?.trim() || "";
  const replayLabel = chrome.data?.ADMIN_ACTION_WEBHOOK_REPLAY?.trim() || "";
  const rowActionsLabel = chrome.data?.ADMIN_TABLE_ROW_ACTIONS?.trim() || "";
  const totalLabel = chrome.data?.ADMIN_OBS_TOTAL_EVENTS?.trim() || "";
  const successLabel = chrome.data?.ADMIN_OBS_SUCCESS_24H?.trim() || "";
  const errorLabel = chrome.data?.ADMIN_OBS_ERROR_24H?.trim() || "";
  const tokensBeforeLabel = chrome.data?.ADMIN_OBS_TOKENS_BEFORE?.trim() || "";
  const tokensAfterLabel = chrome.data?.ADMIN_OBS_TOKENS_AFTER?.trim() || "";
  const byModeLabel = chrome.data?.ADMIN_OBS_BY_MODE?.trim() || "";
  const byModelLabel = chrome.data?.ADMIN_OBS_BY_MODEL?.trim() || "";
  const webhooksLabel = chrome.data?.ADMIN_OBS_WEBHOOKS?.trim() || "";
  const colEventId = chrome.data?.ADMIN_OBS_COL_EVENT_ID?.trim() || "";
  const colEventType = chrome.data?.ADMIN_OBS_COL_EVENT_TYPE?.trim() || "";
  const colProcessed = chrome.data?.ADMIN_OBS_COL_PROCESSED?.trim() || "";
  const colStatus = chrome.data?.ADMIN_OBS_COL_STATUS?.trim() || "";
  const payloadTitle = chrome.data?.ADMIN_OBS_PAYLOAD_TITLE?.trim() || "";
  const viewLabel = chrome.data?.ADMIN_ACTION_VIEW?.trim() || "";
  const closeLabel =
    chrome.data?.ADMIN_DIALOG_CLOSE?.trim() || chrome.data?.ADMIN_CLOSE?.trim() || "";
  const cliVersionLabel = chrome.data?.ADMIN_OBS_CLI_VERSION?.trim() || "";
  const cliForceLabel = chrome.data?.ADMIN_OBS_CLI_FORCE_UPGRADE?.trim() || "";
  const heatColCountry = chrome.data?.ADMIN_OBS_HEATMAP_COL_COUNTRY?.trim() || "";
  const heatColUsers = chrome.data?.ADMIN_OBS_HEATMAP_COL_USERS?.trim() || "";
  const heatColPaid = chrome.data?.ADMIN_OBS_HEATMAP_COL_PAID?.trim() || "";

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setSkip(0);
  }, [q]);

  const heatmapColumns = useMemo<
    ColumnDef<{
      country?: string;
      login_users?: number;
      paid_users?: number;
    }>[]
  >(
    () => [
      {
        id: "country",
        header: heatColCountry,
        cell: ({ row }) => row.original.country?.trim() || "",
      },
      {
        id: "login_users",
        header: heatColUsers,
        cell: ({ row }) => {
          const n = row.original.login_users;
          return n != null ? <span className="tabular-nums">{n}</span> : "";
        },
      },
      {
        id: "paid_users",
        header: heatColPaid,
        cell: ({ row }) =>
          row.original.paid_users != null ? (
            <span className="tabular-nums">{row.original.paid_users}</span>
          ) : (
            ""
          ),
      },
    ],
    [heatColCountry, heatColUsers, heatColPaid],
  );

  const webhookColumns = useMemo<
    ColumnDef<{
      event_id?: string;
      event_type?: string;
      processed_at?: string;
      process_status?: string;
      process_status_label?: string;
    }>[]
  >(() => {
    const cols: ColumnDef<{
      event_id?: string;
      event_type?: string;
      processed_at?: string;
      process_status?: string;
      process_status_label?: string;
    }>[] = [
      {
        id: "event_id",
        header: colEventId,
        cell: ({ row }) => (
          <button
            type="button"
            className="font-mono text-xs text-foreground hover:underline"
            onClick={() => openEvent(row.original.event_id?.trim() || "")}
          >
            {row.original.event_id?.trim() || ""}
          </button>
        ),
      },
      {
        id: "event_type",
        header: colEventType,
        cell: ({ row }) => row.original.event_type?.trim() || "",
      },
      {
        id: "processed_at",
        header: colProcessed,
        cell: ({ row }) => (
          <span className="text-muted-foreground">{row.original.processed_at?.trim() || ""}</span>
        ),
      },
      {
        id: "process_status",
        header: colStatus,
        cell: ({ row }) => row.original.process_status_label?.trim() || "",
      },
    ];
    if ((viewLabel || (replayLabel && pending)) && rowActionsLabel) {
      cols.push({
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const eventId = row.original.event_id?.trim() || "";
          if (!eventId) return null;
          const actions = [];
          if (viewLabel) {
            actions.push({
              id: "view",
              label: viewLabel,
              onSelect: () => openEvent(eventId),
            });
          }
          if (replayLabel && pending) {
            actions.push({
              id: "replay",
              label: replayLabel,
              isLoading: replay.isPending,
              pendingLabel: pending,
              onSelect: () => {
                if (!requireStepUp()) return;
                void replay.mutate(eventId);
              },
            });
          }
          return <DataTableRowActions triggerLabel={rowActionsLabel} actions={actions} />;
        },
      });
    }
    return cols;
  }, [
    colEventId,
    colEventType,
    colProcessed,
    colStatus,
    viewLabel,
    replayLabel,
    pending,
    rowActionsLabel,
    replay,
    requireStepUp,
    openEvent,
  ]);

  const bootReady = useListBootReady(
    pageSize,
    Boolean(chrome.data),
    queryContentReady(webhooks) && queryContentReady(stats),
  );
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    return (
      <MetricsPlusTableSkeleton
        title={title || undefined}
        leadingCards={2}
        metricCount={5}
        breakdownCards={3}
        columns={[colEventId, colEventType, colProcessed, colStatus]
          .filter(Boolean)
          .map((label) => ({ label }))}
        rows={pageSize > 0 ? pageSize : 8}
        showFilters
        filterCount={1}
      />
    );
  }

  const s = stats.data;
  const webhookItems = (webhooks.data?.items ?? []).filter((item) =>
    Boolean(item.event_id?.trim()),
  );

  return (
    <div className="space-y-6">
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      {stepErr ? <p className="text-sm text-destructive">{stepErr}</p> : null}

      {(cliVersionLabel && s?.cli?.min_cli_version) ||
      (cliForceLabel && s?.cli?.force_upgrade_notice) ? (
        <div className="grid gap-3 sm:grid-cols-2">
          {cliVersionLabel && s?.cli?.min_cli_version ? (
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-xs font-medium text-muted-foreground">
                  {cliVersionLabel}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="font-mono text-sm text-foreground">{s.cli.min_cli_version}</p>
              </CardContent>
            </Card>
          ) : null}
          {cliForceLabel && s?.cli?.force_upgrade_notice ? (
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-xs font-medium text-muted-foreground">
                  {cliForceLabel}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-foreground">{s.cli.force_upgrade_notice}</p>
              </CardContent>
            </Card>
          ) : null}
        </div>
      ) : null}

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {totalLabel && s?.total_events != null ? (
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-xs font-medium text-muted-foreground">
                {totalLabel}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-2xl tabular-nums text-foreground">{s.total_events}</p>
            </CardContent>
          </Card>
        ) : null}
        {successLabel && s?.last_24h?.success != null ? (
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-xs font-medium text-muted-foreground">
                {successLabel}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-2xl tabular-nums text-foreground">{s.last_24h.success}</p>
            </CardContent>
          </Card>
        ) : null}
        {errorLabel && s?.last_24h?.error != null ? (
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-xs font-medium text-muted-foreground">
                {errorLabel}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-2xl tabular-nums text-foreground">{s.last_24h.error}</p>
            </CardContent>
          </Card>
        ) : null}
        {tokensBeforeLabel && s?.tokens_before != null ? (
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-xs font-medium text-muted-foreground">
                {tokensBeforeLabel}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-2xl tabular-nums text-foreground">{s.tokens_before}</p>
            </CardContent>
          </Card>
        ) : null}
        {tokensAfterLabel && s?.tokens_after != null ? (
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-xs font-medium text-muted-foreground">
                {tokensAfterLabel}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-2xl tabular-nums text-foreground">{s.tokens_after}</p>
            </CardContent>
          </Card>
        ) : null}
      </div>

      {byModeLabel && (s?.by_mode?.length ?? 0) > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">{byModeLabel}</CardTitle>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={[
                {
                  id: "mode",
                  header: byModeLabel,
                  cell: ({ row }) => row.original.mode || "",
                },
                {
                  id: "_count",
                  header: "",
                  cell: ({ row }) =>
                    typeof row.original.count === "number" ? (
                      <span className="tabular-nums">{row.original.count}</span>
                    ) : (
                      ""
                    ),
                },
              ]}
              data={(s?.by_mode || []).filter((row) => Boolean(row.mode))}
              getRowId={(row) => row.mode || ""}
              isFetching={stats.isFetching && !stats.isPending}
              bordered={false}
            />
          </CardContent>
        </Card>
      ) : null}

      {byModelLabel && (s?.by_model?.length ?? 0) > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">{byModelLabel}</CardTitle>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={[
                {
                  id: "model",
                  header: byModelLabel,
                  cell: ({ row }) => <span className="truncate">{row.original.model || ""}</span>,
                },
                {
                  id: "_count",
                  header: "",
                  cell: ({ row }) =>
                    typeof row.original.count === "number" ? (
                      <span className="tabular-nums">{row.original.count}</span>
                    ) : (
                      ""
                    ),
                },
              ]}
              data={(s?.by_model || []).filter((row) => Boolean(row.model))}
              getRowId={(row) => row.model || ""}
              isFetching={stats.isFetching && !stats.isPending}
              bordered={false}
            />
          </CardContent>
        </Card>
      ) : null}

      {typeof heatmap.data?.title === "string" &&
      heatmap.data.title.trim() &&
      (heatmap.data?.items?.length ?? 0) > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">{heatmap.data.title.trim()}</CardTitle>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={heatmapColumns}
              data={(heatmap.data?.items || []).filter((row) => Boolean(row.country))}
              getRowId={(row) => row.country || ""}
              isFetching={heatmap.isFetching && !heatmap.isPending}
              bordered={false}
            />
          </CardContent>
        </Card>
      ) : null}

      <div className="space-y-3">
        {webhooksLabel ? (
          <h2 className="text-sm font-medium text-foreground">{webhooksLabel}</h2>
        ) : null}
        <AdminFilterBar
          filters={[
            {
              kind: "search",
              id: "obs-webhooks-search",
              label: filterSearch,
              description: filterSearchDesc,
              value: qInput,
              onChange: setQInput,
            },
          ]}
        />
        <Card>
          <CardContent className="pt-6">
            <DataTable
              columns={webhookColumns}
              data={webhookItems}
              meta={webhooks.data?.meta}
              onPage={setSkip}
              pageDisabled={webhooks.isFetching}
              isFetching={(webhooks.isFetching && !webhooks.isPending) || replay.isPending}
              getRowId={(row) => row.event_id?.trim() || ""}
              bordered={false}
            />
          </CardContent>
        </Card>
        <Dialog open={eventOpen} onOpenChange={onEventOpenChange}>
          <DialogContent closeLabel={closeLabel} className="max-w-4xl">
            <DialogHeader>
              {payloadTitle ? <DialogTitle>{payloadTitle}</DialogTitle> : null}
            </DialogHeader>
            {webhookDetail.isPending && !webhookDetail.data ? (
              <p className="text-sm text-muted-foreground">{selectedEventId}</p>
            ) : (
              <pre className="whitespace-pre-wrap break-all font-mono text-xs text-foreground">
                {webhookDetail.data?.payload != null
                  ? JSON.stringify(webhookDetail.data.payload, null, 2)
                  : webhookDetail.data?.payload_raw?.trim() || ""}
              </pre>
            )}
            {closeLabel ? (
              <DialogFooter>
                <Button type="button" variant="outline" onClick={closeEvent}>
                  {closeLabel}
                </Button>
              </DialogFooter>
            ) : null}
          </DialogContent>
        </Dialog>
      </div>
    </div>
  );
}
