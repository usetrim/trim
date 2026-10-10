"use client";

import { AdminFilterBar } from "@/components/admin/admin-filter-bar";
import { FormThenTableSkeleton, TabsSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { type ColumnDef, DataTable } from "@/components/ui/data-table";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FetchProgressBar } from "@/components/ui/fetch-progress";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  useAttestAccessReview,
  useExportAccessReview,
  usePatchComplianceRetention,
  usePurgeComplianceRetention,
} from "@/hooks/mutations/compliance";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminAccessReview, useAdminCompliance } from "@/hooks/queries/compliance";
import { useAdminToken } from "@/hooks/use-admin-token";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { useRequireStepUp } from "@/hooks/use-require-step-up";
import { chromeFieldDesc } from "@/lib/admin-field-desc";
import { formatDateTimeFull, formatDateTimeShort } from "@/lib/format-datetime";
import { useEffect, useMemo, useRef, useState } from "react";

export function ComplianceClient({ initialToken }: { initialToken?: string }) {
  const token = useAdminToken(initialToken);
  const pageSize = useDefaultPageSize();
  const [reviewSkip, setReviewSkip] = useState(0);
  const [reviewQInput, setReviewQInput] = useState("");
  const reviewQ = useDebouncedValue(reviewQInput.trim(), 250);
  const chrome = useAdminNavChrome(token);
  const { err: stepErr, requireStepUp } = useRequireStepUp(token);
  const retention = useAdminCompliance(token);
  const review = useAdminAccessReview(token, reviewSkip, pageSize, reviewQ);
  const patch = usePatchComplianceRetention(token);
  const purge = usePurgeComplianceRetention(token);
  const exportReview = useExportAccessReview(token);
  const attest = useAttestAccessReview(token);
  const hydratedRev = useRef("");
  const [eventsTtl, setEventsTtl] = useState("");
  const [auditTtl, setAuditTtl] = useState("");
  const [auditExportMax, setAuditExportMax] = useState("");
  const [attestLimit, setAttestLimit] = useState("");
  const [breakGlassTtl, setBreakGlassTtl] = useState("");
  const [forceLogoutTtl, setForceLogoutTtl] = useState("");
  const [partitionMonthsAhead, setPartitionMonthsAhead] = useState("");
  const [partitionEnsureSec, setPartitionEnsureSec] = useState("");
  const [purgeReason, setPurgeReason] = useState("");
  const [periodLabel, setPeriodLabel] = useState("");
  const [attestNotes, setAttestNotes] = useState("");
  const [attestReason, setAttestReason] = useState("");
  const [attestOpen, setAttestOpen] = useState(false);
  const title = chrome.data?.ADMIN_NAV_COMPLIANCE?.trim() || "";
  const pendingCreating = chrome.data?.ADMIN_PENDING_CREATING?.trim() || "";
  const pendingUpdating =
    chrome.data?.ADMIN_PENDING_UPDATING?.trim() || chrome.data?.ADMIN_PENDING_SAVING?.trim() || "";
  const pendingSaving = chrome.data?.ADMIN_PENDING_SAVING?.trim() || "";
  const pendingDelete = chrome.data?.ADMIN_PENDING_DELETING?.trim() || "";
  const pendingDestructive = pendingDelete || pendingSaving;
  const saveLabel = chrome.data?.ADMIN_ACTION_SAVE?.trim() || "";
  const eventsLabel = chrome.data?.ADMIN_COMPLIANCE_EVENTS_TTL?.trim() || "";
  const auditLabel = chrome.data?.ADMIN_COMPLIANCE_AUDIT_TTL?.trim() || "";
  const auditExportLabel = chrome.data?.ADMIN_COMPLIANCE_AUDIT_EXPORT_MAX?.trim() || "";
  const attestLimitLabel = chrome.data?.ADMIN_COMPLIANCE_ATTEST_LIMIT?.trim() || "";
  const breakGlassLabel = chrome.data?.ADMIN_COMPLIANCE_BREAK_GLASS_TTL?.trim() || "";
  const forceLogoutTtlLabel = chrome.data?.ADMIN_COMPLIANCE_FORCE_LOGOUT_TTL?.trim() || "";
  const partitionAheadLabel = chrome.data?.ADMIN_COMPLIANCE_EVENTS_PARTITION_AHEAD?.trim() || "";
  const partitionEnsureLabel =
    chrome.data?.ADMIN_COMPLIANCE_EVENTS_PARTITION_ENSURE_SEC?.trim() || "";
  const reviewTitle = chrome.data?.ADMIN_COMPLIANCE_REVIEW_TITLE?.trim() || "";
  const filterSearch = chrome.data?.ADMIN_FILTER_SEARCH?.trim() || "";
  const filterSearchDesc = chrome.data?.ADMIN_FILTER_SEARCH_DESC?.trim() || "";
  const purgeLabel = chrome.data?.ADMIN_RETENTION_PURGE?.trim() || "";
  const reasonRequired = chrome.data?.ADMIN_REASON_REQUIRED?.trim() || "";
  const reasonFieldDesc = chrome.data?.ADMIN_REASON_FIELD_DESC?.trim() || "";
  const exportLabel = chrome.data?.ADMIN_ACCESS_REVIEW_EXPORT?.trim() || "";
  const attestLabel = chrome.data?.ADMIN_ACCESS_REVIEW_ATTEST?.trim() || "";
  const dangerSectionLabel = chrome.data?.ADMIN_SECTION_DANGER?.trim() || "";
  const editSectionLabel = chrome.data?.ADMIN_SECTION_EDIT?.trim() || "";
  const createSectionLabel = chrome.data?.ADMIN_SECTION_CREATE?.trim() || "";
  const periodLabelChrome = chrome.data?.ADMIN_ACCESS_REVIEW_PERIOD?.trim() || "";
  const notesLabel = chrome.data?.ADMIN_ACCESS_REVIEW_NOTES?.trim() || "";
  const historyLabel = chrome.data?.ADMIN_ACCESS_REVIEW_HISTORY?.trim() || "";
  const colEmail = chrome.data?.ADMIN_ACCESS_REVIEW_COL_EMAIL?.trim() || "";
  const colStatus = chrome.data?.ADMIN_ACCESS_REVIEW_COL_STATUS?.trim() || "";
  const colRole = chrome.data?.ADMIN_ACCESS_REVIEW_COL_ROLE?.trim() || "";
  const attestColPeriod = chrome.data?.ADMIN_ATTEST_COL_PERIOD?.trim() || "";
  const attestColWhen = chrome.data?.ADMIN_ATTEST_COL_WHEN?.trim() || "";
  const htmlLang = chrome.data?.SITE_HTML_LANG?.trim() || "";
  const attestColNotes = chrome.data?.ADMIN_ATTEST_COL_NOTES?.trim() || "";
  const closeLabel =
    chrome.data?.ADMIN_DIALOG_CLOSE?.trim() || chrome.data?.ADMIN_CLOSE?.trim() || "";

  const reviewColumns = useMemo<ColumnDef<Record<string, unknown>>[]>(
    () => [
      {
        id: "email",
        header: colEmail,
        cell: ({ row }) => {
          const email = typeof row.original.email === "string" ? row.original.email : "";
          const role =
            typeof row.original.role_name === "string" ? row.original.role_name.trim() : "";
          if (!email) return "";
          return (
            <span className="text-foreground">
              {email}
              {colRole && role ? <span className="ml-2 text-muted-foreground">{role}</span> : null}
            </span>
          );
        },
      },
      {
        id: "status",
        header: colStatus,
        cell: ({ row }) =>
          typeof row.original.status_label === "string" ? row.original.status_label.trim() : "",
      },
    ],
    [colEmail, colStatus, colRole],
  );

  const attestColumns = useMemo<ColumnDef<Record<string, unknown>>[]>(
    () => [
      {
        id: "period_label",
        header: attestColPeriod,
        cell: ({ row }) => String(row.original.period_label || ""),
      },
      {
        id: "created_at",
        header: attestColWhen,
        cell: ({ row }) => {
          const raw = String(row.original.created_at || "");
          return (
            <span title={formatDateTimeFull(raw, htmlLang) || undefined}>
              {formatDateTimeShort(raw, htmlLang)}
            </span>
          );
        },
      },
      {
        id: "notes",
        header: attestColNotes,
        cell: ({ row }) => String(row.original.notes || ""),
      },
    ],
    [attestColPeriod, attestColWhen, attestColNotes, htmlLang],
  );

  useEffect(() => {
    const d = retention.data;
    if (!d) return;
    const rev = typeof d.updated_at === "string" ? d.updated_at : "";
    if (rev && hydratedRev.current === rev) return;
    if (!rev && hydratedRev.current === "*") return;
    hydratedRev.current = rev || "*";
    setEventsTtl(typeof d.trim_events_ttl_days === "number" ? String(d.trim_events_ttl_days) : "");
    setAuditTtl(typeof d.audit_log_ttl_days === "number" ? String(d.audit_log_ttl_days) : "");
    setAuditExportMax(
      typeof d.audit_export_max_rows === "number" ? String(d.audit_export_max_rows) : "",
    );
    setAttestLimit(
      typeof d.access_review_attestations_limit === "number"
        ? String(d.access_review_attestations_limit)
        : "",
    );
    setBreakGlassTtl(
      typeof d.break_glass_ttl_minutes === "number" ? String(d.break_glass_ttl_minutes) : "",
    );
    setForceLogoutTtl(
      typeof d.force_logout_ttl_sec === "number" ? String(d.force_logout_ttl_sec) : "",
    );
    setPartitionMonthsAhead(
      typeof d.trim_events_partition_months_ahead === "number"
        ? String(d.trim_events_partition_months_ahead)
        : "",
    );
    setPartitionEnsureSec(
      typeof d.trim_events_partition_ensure_sec === "number"
        ? String(d.trim_events_partition_ensure_sec)
        : "",
    );
  }, [retention.data]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setReviewSkip(0);
  }, [reviewQ]);

  const retentionTabLabel = editSectionLabel;
  const reviewTabLabel = reviewTitle;
  const useComplianceTabs = Boolean(retentionTabLabel && reviewTabLabel);
  const [tab, setTab] = useState("retention");
  const primaryCompliance = tab === "review" ? review : retention;

  const bootReady = useListBootReady(
    pageSize,
    Boolean(chrome.data),
    queryContentReady(primaryCompliance),
  );
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    return (
      <div className="space-y-4">
        {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
        <TabsSkeleton count={2} />
        <FormThenTableSkeleton
          title={undefined}
          fields={8}
          columns={[colEmail, colStatus, colRole].map((label) => ({ label: label || "" }))}
          rows={pageSize > 0 ? pageSize : 8}
          showFilters
        />
      </div>
    );
  }

  const retentionPanel = (
    <Card className="overflow-hidden bg-card">
      <FetchProgressBar
        active={
          (retention.isFetching && !retention.isPending) || patch.isPending || purge.isPending
        }
      />
      <CardContent className="space-y-4 pt-6">
        {editSectionLabel && !useComplianceTabs ? (
          <p className="text-sm font-medium text-foreground">{editSectionLabel}</p>
        ) : null}
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {eventsLabel ? (
            <Field
              id="compliance-events-ttl"
              label={eventsLabel}
              {...chromeFieldDesc(chrome.data, "ADMIN_COMPLIANCE_EVENTS_TTL_DESC")}
            >
              <Input
                id="compliance-events-ttl"
                value={eventsTtl}
                onChange={(e) => setEventsTtl(e.target.value)}
                placeholder={eventsLabel}
              />
            </Field>
          ) : null}
          {auditLabel ? (
            <Field
              id="compliance-audit-ttl"
              label={auditLabel}
              {...chromeFieldDesc(chrome.data, "ADMIN_COMPLIANCE_AUDIT_TTL_DESC")}
            >
              <Input
                id="compliance-audit-ttl"
                value={auditTtl}
                onChange={(e) => setAuditTtl(e.target.value)}
                placeholder={auditLabel}
              />
            </Field>
          ) : null}
          {auditExportLabel ? (
            <Field
              id="compliance-audit-export"
              label={auditExportLabel}
              {...chromeFieldDesc(chrome.data, "ADMIN_COMPLIANCE_AUDIT_EXPORT_MAX_DESC")}
            >
              <Input
                id="compliance-audit-export"
                value={auditExportMax}
                onChange={(e) => setAuditExportMax(e.target.value)}
                placeholder={auditExportLabel}
              />
            </Field>
          ) : null}
          {attestLimitLabel ? (
            <Field
              id="compliance-attest-limit"
              label={attestLimitLabel}
              {...chromeFieldDesc(chrome.data, "ADMIN_COMPLIANCE_ATTEST_LIMIT_DESC")}
            >
              <Input
                id="compliance-attest-limit"
                value={attestLimit}
                onChange={(e) => setAttestLimit(e.target.value)}
                placeholder={attestLimitLabel}
              />
            </Field>
          ) : null}
          {breakGlassLabel ? (
            <Field
              id="compliance-break-glass-ttl"
              label={breakGlassLabel}
              {...chromeFieldDesc(chrome.data, "ADMIN_COMPLIANCE_BREAK_GLASS_TTL_DESC")}
            >
              <Input
                id="compliance-break-glass-ttl"
                value={breakGlassTtl}
                onChange={(e) => setBreakGlassTtl(e.target.value)}
                placeholder={breakGlassLabel}
              />
            </Field>
          ) : null}
          {forceLogoutTtlLabel ? (
            <Field
              id="compliance-force-logout-ttl"
              label={forceLogoutTtlLabel}
              {...chromeFieldDesc(chrome.data, "ADMIN_COMPLIANCE_FORCE_LOGOUT_TTL_DESC")}
            >
              <Input
                id="compliance-force-logout-ttl"
                value={forceLogoutTtl}
                onChange={(e) => setForceLogoutTtl(e.target.value)}
                placeholder={forceLogoutTtlLabel}
              />
            </Field>
          ) : null}
          {partitionAheadLabel ? (
            <Field
              id="compliance-partition-ahead"
              label={partitionAheadLabel}
              {...chromeFieldDesc(chrome.data, "ADMIN_COMPLIANCE_EVENTS_PARTITION_AHEAD_DESC")}
            >
              <Input
                id="compliance-partition-ahead"
                value={partitionMonthsAhead}
                onChange={(e) => setPartitionMonthsAhead(e.target.value)}
                placeholder={partitionAheadLabel}
              />
            </Field>
          ) : null}
          {partitionEnsureLabel ? (
            <Field
              id="compliance-partition-ensure"
              label={partitionEnsureLabel}
              {...chromeFieldDesc(chrome.data, "ADMIN_COMPLIANCE_EVENTS_PARTITION_ENSURE_SEC_DESC")}
            >
              <Input
                id="compliance-partition-ensure"
                value={partitionEnsureSec}
                onChange={(e) => setPartitionEnsureSec(e.target.value)}
                placeholder={partitionEnsureLabel}
              />
            </Field>
          ) : null}
        </div>
        <div className="mt-4 flex flex-wrap gap-2">
          {saveLabel && pendingUpdating ? (
            <Button
              type="button"
              isLoading={patch.isPending}
              pendingLabel={pendingUpdating}
              onClick={() => {
                if (!requireStepUp()) return;
                void patch.mutate({
                  trim_events_ttl_days: eventsTtl ? Number(eventsTtl) : undefined,
                  audit_log_ttl_days: auditTtl ? Number(auditTtl) : undefined,
                  audit_export_max_rows: auditExportMax ? Number(auditExportMax) : undefined,
                  access_review_attestations_limit: attestLimit ? Number(attestLimit) : undefined,
                  break_glass_ttl_minutes: breakGlassTtl ? Number(breakGlassTtl) : undefined,
                  force_logout_ttl_sec: forceLogoutTtl ? Number(forceLogoutTtl) : undefined,
                  trim_events_partition_months_ahead: partitionMonthsAhead
                    ? Number(partitionMonthsAhead)
                    : undefined,
                  trim_events_partition_ensure_sec: partitionEnsureSec
                    ? Number(partitionEnsureSec)
                    : undefined,
                });
              }}
            >
              {saveLabel}
            </Button>
          ) : null}
          {purgeLabel && reasonRequired && pendingDestructive ? (
            <>
              {dangerSectionLabel ? (
                <p className="w-full text-sm font-medium text-foreground">{dangerSectionLabel}</p>
              ) : null}
              <Field
                id="compliance-purge-reason"
                label={reasonRequired}
                description={reasonFieldDesc}
                className="w-full sm:min-w-[16rem] sm:flex-1"
              >
                <Input
                  id="compliance-purge-reason"
                  value={purgeReason}
                  onChange={(e) => setPurgeReason(e.target.value)}
                  placeholder={reasonRequired}
                />
              </Field>
              <Button
                type="button"
                variant="secondary"
                isLoading={purge.isPending}
                pendingLabel={pendingDestructive}
                onClick={() => {
                  if (!requireStepUp()) return;
                  void purge.mutate({ reason: purgeReason.trim() });
                }}
              >
                {purgeLabel}
              </Button>
            </>
          ) : null}
        </div>
      </CardContent>
    </Card>
  );

  const reviewPanel = (
    <div className="space-y-4">
      {exportLabel && pendingSaving ? (
        <Button
          type="button"
          variant="secondary"
          isLoading={exportReview.isPending}
          pendingLabel={pendingSaving}
          onClick={() => {
            if (!requireStepUp()) return;
            void exportReview.mutate();
          }}
        >
          {exportLabel}
        </Button>
      ) : null}
      {pageSize > 0 ? (
        <Card>
          <CardContent className="space-y-3 pt-6">
            {reviewTitle && !useComplianceTabs ? (
              <h2 className="text-sm font-medium text-foreground">{reviewTitle}</h2>
            ) : null}
            <AdminFilterBar
              filters={[
                {
                  kind: "search",
                  id: "compliance-review-search",
                  label: filterSearch,
                  description: filterSearchDesc,
                  value: reviewQInput,
                  onChange: setReviewQInput,
                },
              ]}
            />
            <DataTable
              columns={reviewColumns}
              data={((review.data?.items || []) as Array<Record<string, unknown>>).filter(
                (row) => typeof row.email === "string" && Boolean(row.email),
              )}
              meta={review.data?.meta}
              onPage={setReviewSkip}
              pageDisabled={review.isFetching}
              isFetching={(review.isFetching && !review.isPending) || attest.isPending}
              getRowId={(row, i) => String(row.user_id || i)}
              bordered={false}
            />
          </CardContent>
        </Card>
      ) : null}
      {attestLabel && periodLabelChrome && reasonRequired && pendingCreating ? (
        <Button type="button" onClick={() => setAttestOpen(true)}>
          {attestLabel}
        </Button>
      ) : null}
      {attestLabel && periodLabelChrome && reasonRequired && pendingCreating ? (
        <Dialog
          open={attestOpen}
          onOpenChange={(open) => {
            if (!open) setAttestOpen(false);
            else setAttestOpen(true);
          }}
        >
          <DialogContent closeLabel={closeLabel} className="max-w-2xl">
            <DialogHeader>
              {createSectionLabel ? <DialogTitle>{createSectionLabel}</DialogTitle> : null}
            </DialogHeader>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field
                id="compliance-attest-period"
                label={periodLabelChrome}
                {...chromeFieldDesc(chrome.data, "ADMIN_ACCESS_REVIEW_PERIOD_DESC")}
              >
                <Input
                  id="compliance-attest-period"
                  value={periodLabel}
                  onChange={(e) => setPeriodLabel(e.target.value)}
                  placeholder={periodLabelChrome}
                />
              </Field>
              {notesLabel ? (
                <Field
                  id="compliance-attest-notes"
                  label={notesLabel}
                  {...chromeFieldDesc(chrome.data, "ADMIN_ACCESS_REVIEW_NOTES_DESC")}
                >
                  <Input
                    id="compliance-attest-notes"
                    value={attestNotes}
                    onChange={(e) => setAttestNotes(e.target.value)}
                    placeholder={notesLabel}
                  />
                </Field>
              ) : null}
              <Field
                id="compliance-attest-reason"
                label={reasonRequired}
                {...chromeFieldDesc(chrome.data, "ADMIN_COMPLIANCE_ATTEST_REASON_DESC")}
              >
                <Input
                  id="compliance-attest-reason"
                  value={attestReason}
                  onChange={(e) => setAttestReason(e.target.value)}
                  placeholder={reasonRequired}
                />
              </Field>
            </div>
            <DialogFooter>
              <Button
                type="button"
                isLoading={attest.isPending}
                pendingLabel={pendingCreating}
                onClick={() => {
                  if (!requireStepUp()) return;
                  void attest.mutate(
                    {
                      period_label: periodLabel.trim(),
                      notes: attestNotes.trim(),
                      reason: attestReason.trim(),
                    },
                    {
                      onSuccess: () => {
                        setAttestOpen(false);
                        setPeriodLabel("");
                        setAttestNotes("");
                        setAttestReason("");
                      },
                    },
                  );
                }}
              >
                {attestLabel}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}
      {historyLabel && (review.data?.attestations?.length ?? 0) > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">{historyLabel}</CardTitle>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={attestColumns}
              data={(review.data?.attestations || []) as Array<Record<string, unknown>>}
              getRowId={(row, i) => String(row.id || i)}
              isFetching={review.isFetching && !review.isPending}
              bordered={false}
            />
          </CardContent>
        </Card>
      ) : null}
    </div>
  );

  const complianceBody = useComplianceTabs ? (
    <>
      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="retention">{retentionTabLabel}</TabsTrigger>
          <TabsTrigger value="review">{reviewTabLabel}</TabsTrigger>
        </TabsList>
      </Tabs>
      <div className={tab === "retention" ? "space-y-4" : "hidden"}>{retentionPanel}</div>
      <div className={tab === "review" ? "space-y-4" : "hidden"}>{reviewPanel}</div>
    </>
  ) : (
    <>
      {retentionPanel}
      {reviewPanel}
    </>
  );

  return (
    <div className="space-y-4">
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      {stepErr ? <p className="text-sm text-destructive">{stepErr}</p> : null}
      {complianceBody}
    </div>
  );
}
