"use client";

import {
  DataTableSkeleton,
  FormPageSkeleton,
  TabsSkeleton,
} from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { type ColumnDef, DataTable } from "@/components/ui/data-table";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import {
  useAdminUserForceLogout,
  useAdminUserGDPRErase,
  useAdminUserGDPRExport,
  useAdminUserNotes,
  useAdminUserQuota,
  useAdminUserRevokeKeys,
  useAdminUserStatus,
  useCreditGrant,
} from "@/hooks/mutations/users";
import { useAuthProviders } from "@/hooks/queries/auth";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminCreditGrants, useAdminTopupLedger, useAdminUser } from "@/hooks/queries/users";
import { useAdminToken } from "@/hooks/use-admin-token";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { useRequireStepUp } from "@/hooks/use-require-step-up";
import { chromeFieldDesc } from "@/lib/admin-field-desc";
import { joinChromeParts } from "@/lib/chrome-join";
import { formatDateTimeFull, formatDateTimeShort } from "@/lib/format-datetime";
import Link from "next/link";
import { useEffect, useMemo, useRef, useState } from "react";

export function UserDetailClient({
  initialToken,
  userId,
}: {
  initialToken?: string;
  userId: string;
}) {
  const token = useAdminToken(initialToken);
  const pageSize = useDefaultPageSize();
  const [creditSkip, setCreditSkip] = useState(0);
  const [topupSkip, setTopupSkip] = useState(0);
  const chrome = useAdminNavChrome(token);
  const { err: stepErr, requireStepUp } = useRequireStepUp(token);
  const providers = useAuthProviders();
  const user = useAdminUser(token, userId);
  const creditGrants = useAdminCreditGrants(token, userId, creditSkip, pageSize);
  const topupLedger = useAdminTopupLedger(token, userId, topupSkip, pageSize);
  const statusMut = useAdminUserStatus(token, userId);
  const notesMut = useAdminUserNotes(token, userId);
  const quotaMut = useAdminUserQuota(token, userId);
  const revokeMut = useAdminUserRevokeKeys(token, userId);
  const logoutMut = useAdminUserForceLogout(token, userId);
  const exportMut = useAdminUserGDPRExport(token, userId);
  const eraseMut = useAdminUserGDPRErase(token, userId);
  const creditMut = useCreditGrant(token);
  const [creditAmount, setCreditAmount] = useState("");
  const seededUserId = useRef("");

  const [status, setStatus] = useState("");
  const [reason, setReason] = useState("");
  const [notes, setNotes] = useState("");
  const [limit, setLimit] = useState("");
  const [topup, setTopup] = useState("");
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [confirmAction, setConfirmAction] = useState<"erase" | "logout" | null>(null);
  const [grantOpen, setGrantOpen] = useState(false);

  const labels = chrome.data ?? {};
  const htmlLang = labels.SITE_HTML_LANG?.trim() || "";
  const pendingSave =
    labels.ADMIN_PENDING_UPDATING?.trim() || labels.ADMIN_PENDING_SAVING?.trim() || "";
  const pendingCreating = labels.ADMIN_PENDING_CREATING?.trim() || "";
  // Prefer status-specific pending chrome when present; else shared saving chrome.
  const pendingSuspend = labels.ADMIN_PENDING_SUSPENDING?.trim() || "";
  const pendingDelete = labels.ADMIN_PENDING_DELETING?.trim() || "";
  const pendingDestructive = pendingDelete || pendingSave;
  const reasonRequired = labels.ADMIN_REASON_REQUIRED?.trim() || "";
  const reasonFieldDesc = labels.ADMIN_REASON_FIELD_DESC?.trim() || "";
  const filterStatusDesc = labels.ADMIN_FILTER_STATUS_DESC?.trim() || "";
  const revokeLabel = labels.ADMIN_ACTION_REVOKE_KEYS?.trim() || "";
  const logoutLabel = labels.ADMIN_ACTION_FORCE_LOGOUT?.trim() || "";
  const exportLabel = labels.ADMIN_ACTION_GDPR_EXPORT?.trim() || "";
  const eraseLabel = labels.ADMIN_ACTION_GDPR_ERASE?.trim() || "";
  const saveLabel = labels.ADMIN_ACTION_SAVE?.trim() || "";
  const creditLabel = labels.ADMIN_ACTION_CREDIT_GRANT?.trim() || "";
  const grantSectionLabel = labels.ADMIN_CREDITS_GRANT_TITLE?.trim() || "";
  const editSectionLabel = labels.ADMIN_SECTION_EDIT?.trim() || "";
  const dangerSectionLabel = labels.ADMIN_SECTION_DANGER?.trim() || "";
  const ledgerTitle = labels.ADMIN_CREDIT_LEDGER_TITLE?.trim() || "";
  const topupLedgerTitle = labels.ADMIN_TOPUP_LEDGER_TITLE?.trim() || "";
  const cancelLabel = providers.data?.dialog_cancel?.trim() || "";
  const secIdentities = labels.ADMIN_USER_SECTION_IDENTITIES?.trim() || "";
  const secWorkspaces = labels.ADMIN_USER_SECTION_WORKSPACES?.trim() || "";
  const secDevices = labels.ADMIN_USER_SECTION_DEVICES?.trim() || "";
  const secJa4 = labels.ADMIN_USER_SECTION_JA4?.trim() || "";
  const secLinked = labels.ADMIN_USER_SECTION_LINKED?.trim() || "";
  const secLinkedHw = labels.ADMIN_USER_SECTION_LINKED_HW?.trim() || "";
  const providerCol = labels.ADMIN_USERS_COL_PROVIDER?.trim() || "";
  const loginCountryLabel = labels.ADMIN_USER_LOGIN_COUNTRY?.trim() || "";
  const billToCountryLabel = labels.ADMIN_USER_BILL_TO_COUNTRY?.trim() || "";
  const colIdentityProvider = labels.ADMIN_USER_COL_PROVIDER?.trim() || "";
  const colIdentity = labels.ADMIN_USER_COL_IDENTITY?.trim() || "";
  const colWorkspace = labels.ADMIN_USER_COL_WORKSPACE?.trim() || "";
  const colRole = labels.ADMIN_USER_COL_ROLE?.trim() || "";
  const colDevice = labels.ADMIN_USER_COL_DEVICE?.trim() || "";
  const colJa4 = labels.ADMIN_USER_COL_JA4?.trim() || "";
  const colLinked = labels.ADMIN_USER_COL_LINKED?.trim() || "";
  const colLinkedHw = labels.ADMIN_USER_COL_LINKED_HW?.trim() || "";
  const colCreditAmount = labels.ADMIN_CREDIT_COL_AMOUNT?.trim() || "";
  const colCreditReason = labels.ADMIN_CREDIT_COL_REASON?.trim() || "";
  const colCreditWhen = labels.ADMIN_CREDIT_COL_WHEN?.trim() || "";
  const colTopupAmount = labels.ADMIN_TOPUP_COL_AMOUNT?.trim() || "";
  const colTopupSource = labels.ADMIN_TOPUP_COL_SOURCE?.trim() || "";
  const colTopupWhen = labels.ADMIN_TOPUP_COL_WHEN?.trim() || "";
  const sepDot = labels.ADMIN_UI_SEP_DOT?.trim() || "";
  const filterStatus = labels.ADMIN_FILTER_STATUS?.trim() || "";
  const notesLabel = labels.ADMIN_USER_NOTES?.trim() || "";
  const notesDesc = labels.ADMIN_USER_NOTES_DESC?.trim() || "";
  const quotaLimitLabel = labels.ADMIN_USER_QUOTA_LIMIT?.trim() || "";
  const quotaTopupLabel = labels.ADMIN_USER_QUOTA_TOPUP?.trim() || "";
  const creditAmountLabel = labels.ADMIN_CREDITS_GRANT_AMOUNT?.trim() || "";
  const statusOptions = (
    [
      ["active", labels.ADMIN_STATUS_ACTIVE],
      ["suspended", labels.ADMIN_STATUS_SUSPENDED],
      ["banned", labels.ADMIN_STATUS_BANNED],
      ["pending_delete", labels.ADMIN_STATUS_PENDING_DELETE],
      ["shadowbanned", labels.ADMIN_STATUS_SHADOWBANNED],
    ] as const
  ).filter(([, label]) => Boolean(label?.trim()));

  const identityColumns = useMemo<ColumnDef<Record<string, unknown>>[]>(
    () => [
      {
        id: "provider",
        header: colIdentityProvider,
        cell: ({ row }) => String(row.original.provider || ""),
      },
      {
        id: "email",
        header: colIdentity,
        cell: ({ row }) => String(row.original.email || ""),
      },
    ],
    [colIdentityProvider, colIdentity],
  );
  const workspaceColumns = useMemo<ColumnDef<Record<string, unknown>>[]>(
    () => [
      {
        id: "name",
        header: colWorkspace,
        cell: ({ row }) => String(row.original.name || ""),
      },
      {
        id: "role",
        header: colRole,
        cell: ({ row }) => String(row.original.role || ""),
      },
    ],
    [colWorkspace, colRole],
  );
  const deviceColumns = useMemo<ColumnDef<Record<string, unknown>>[]>(
    () => [
      {
        id: "hardware_uuid",
        header: colDevice,
        cell: ({ row }) => (
          <span className="font-mono text-xs">{String(row.original.hardware_uuid || "")}</span>
        ),
      },
    ],
    [colDevice],
  );
  const ja4Columns = useMemo<ColumnDef<Record<string, unknown>>[]>(
    () => [
      {
        id: "ja4_hash",
        header: colJa4,
        cell: ({ row }) => (
          <span className="font-mono text-xs">{String(row.original.ja4_hash || "")}</span>
        ),
      },
    ],
    [colJa4],
  );
  const linkedColumns = useMemo<ColumnDef<Record<string, unknown>>[]>(
    () => [
      {
        id: "email",
        header: colLinked,
        cell: ({ row }) => {
          const id = String(row.original.id || "");
          const email = String(row.original.email || "");
          if (!email) return "";
          return id ? (
            <Link href={`/users/${id}`} className="text-foreground hover:underline">
              {email}
            </Link>
          ) : (
            email
          );
        },
      },
      {
        id: "ja4_hash",
        header: colJa4,
        cell: ({ row }) =>
          row.original.ja4_hash ? (
            <span className="font-mono text-xs text-muted-foreground">
              {String(row.original.ja4_hash)}
            </span>
          ) : (
            ""
          ),
      },
    ],
    [colLinked, colJa4],
  );
  const linkedHwColumns = useMemo<ColumnDef<Record<string, unknown>>[]>(
    () => [
      {
        id: "email",
        header: colLinkedHw,
        cell: ({ row }) => {
          const id = String(row.original.id || "");
          const email = String(row.original.email || "");
          if (!email) return "";
          return id ? (
            <Link href={`/users/${id}`} className="text-foreground hover:underline">
              {email}
            </Link>
          ) : (
            email
          );
        },
      },
      {
        id: "hardware_uuid",
        header: colDevice,
        cell: ({ row }) =>
          row.original.hardware_uuid ? (
            <span className="font-mono text-xs text-muted-foreground">
              {String(row.original.hardware_uuid)}
            </span>
          ) : (
            ""
          ),
      },
    ],
    [colLinkedHw, colDevice],
  );
  const creditColumns = useMemo<ColumnDef<Record<string, unknown>>[]>(
    () => [
      {
        id: "credits",
        header: colCreditAmount,
        cell: ({ row }) => String(row.original.credits ?? ""),
      },
      {
        id: "reason",
        header: colCreditReason,
        cell: ({ row }) => String(row.original.reason || ""),
      },
      {
        id: "created_at",
        header: colCreditWhen,
        cell: ({ row }) => {
          const raw = String(row.original.created_at || "");
          return (
            <span title={formatDateTimeFull(raw, htmlLang) || undefined}>
              {formatDateTimeShort(raw, htmlLang)}
            </span>
          );
        },
      },
    ],
    [colCreditAmount, colCreditReason, colCreditWhen, htmlLang],
  );
  const topupColumns = useMemo<ColumnDef<Record<string, unknown>>[]>(
    () => [
      {
        id: "credits_granted",
        header: colTopupAmount,
        cell: ({ row }) => String(row.original.credits_granted ?? ""),
      },
      {
        id: "source",
        header: colTopupSource,
        cell: ({ row }) =>
          joinChromeParts([row.original.paddle_transaction_id, row.original.price_id], sepDot),
      },
      {
        id: "created_at",
        header: colTopupWhen,
        cell: ({ row }) => {
          const raw = String(row.original.created_at || "");
          return (
            <span title={formatDateTimeFull(raw, htmlLang) || undefined}>
              {formatDateTimeShort(raw, htmlLang)}
            </span>
          );
        },
      },
    ],
    [colTopupAmount, colTopupSource, colTopupWhen, sepDot, htmlLang],
  );

  // biome-ignore lint/correctness/useExhaustiveDependencies: clear seed gate when navigating users
  useEffect(() => {
    seededUserId.current = "";
  }, [userId]);

  useEffect(() => {
    const data = user.data as Record<string, unknown> | undefined;
    if (!data) return;
    const rowId = typeof data.id === "string" ? data.id : "";
    if (rowId && rowId !== userId) return;
    // Seed once per user - do not wipe in-progress edits when grants/quota invalidate.
    if (seededUserId.current === userId) return;
    seededUserId.current = userId;
    setStatus(typeof data.account_status === "string" ? data.account_status : "");
    setNotes(typeof data.admin_notes === "string" ? data.admin_notes : "");
    const quota =
      data.quota && typeof data.quota === "object" ? (data.quota as Record<string, unknown>) : null;
    if (quota) {
      setLimit(
        typeof quota.monthly_credit_limit === "number"
          ? String(quota.monthly_credit_limit)
          : typeof quota.monthly_credit_limit === "string"
            ? quota.monthly_credit_limit
            : "",
      );
      setTopup(
        typeof quota.purchased_topup_credits === "number"
          ? String(quota.purchased_topup_credits)
          : typeof quota.purchased_topup_credits === "string"
            ? quota.purchased_topup_credits
            : "",
      );
    }
  }, [userId, user.data]);

  const activityTabLabel =
    secIdentities || secWorkspaces || secDevices || secJa4 || secLinked || secLinkedHw;
  const tabEdit = editSectionLabel;
  const tabActivity = activityTabLabel;
  const tabCredits = grantSectionLabel;
  const tabDanger = dangerSectionLabel;
  const tabCount = [tabEdit, tabActivity, tabCredits, tabDanger].filter(Boolean).length;
  const useTabs = tabCount >= 2;
  const defaultTab = tabEdit
    ? "edit"
    : tabActivity
      ? "activity"
      : tabCredits
        ? "credits"
        : "danger";

  const bootReady = useListBootReady(pageSize, Boolean(chrome.data), queryContentReady(user));
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    return (
      <div className="space-y-4">
        <TabsSkeleton count={tabCount >= 2 ? tabCount : 4} />
        <div className="space-y-4">
          <Card>
            <CardContent className="space-y-3 pt-6">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-9 w-24" />
            </CardContent>
          </Card>
          <Card>
            <CardContent className="space-y-3 pt-6">
              <Skeleton className="h-4 w-28" />
              <Skeleton className="h-24 w-full" />
              <Skeleton className="h-9 w-24" />
            </CardContent>
          </Card>
        </div>
        <FormPageSkeleton hideTitle fields={3} />
        <DataTableSkeleton
          title=""
          columns={[{ label: "" }, { label: "" }]}
          rows={4}
          showFilters={false}
        />
      </div>
    );
  }

  const data = user.data as Record<string, unknown> | undefined;
  const email = typeof data?.email === "string" ? data.email : "";
  const quota =
    data?.quota && typeof data.quota === "object" ? (data.quota as Record<string, unknown>) : null;
  const identities = Array.isArray(data?.identities)
    ? (data?.identities as Array<Record<string, unknown>>)
    : [];
  const devices = Array.isArray(data?.devices)
    ? (data?.devices as Array<Record<string, unknown>>)
    : [];
  const ja4 = Array.isArray(data?.ja4_fingerprints)
    ? (data?.ja4_fingerprints as Array<Record<string, unknown>>)
    : [];
  const linked = Array.isArray(data?.linked_by_ja4)
    ? (data?.linked_by_ja4 as Array<Record<string, unknown>>)
    : [];
  const linkedHw = Array.isArray(data?.linked_by_hardware)
    ? (data?.linked_by_hardware as Array<Record<string, unknown>>)
    : [];
  const workspaces = Array.isArray(data?.workspaces)
    ? (data?.workspaces as Array<Record<string, unknown>>)
    : [];
  const authProvider = typeof data?.auth_provider === "string" ? data.auth_provider.trim() : "";
  const loginCountry =
    typeof data?.last_login_country === "string" ? data.last_login_country.trim() : "";
  const billToCountry =
    typeof data?.bill_to_country === "string" ? data.bill_to_country.trim() : "";

  const showEditSectionHeading = !useTabs;

  const editTabBody = (
    <>
      <div className="space-y-4">
        <Card>
          <CardContent className="space-y-4 pt-6">
            {showEditSectionHeading && editSectionLabel ? (
              <p className="text-sm font-medium text-foreground">{editSectionLabel}</p>
            ) : null}
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              {filterStatus && statusOptions.length > 0 ? (
                <Field id="user-status" label={filterStatus} description={filterStatusDesc}>
                  <Select value={status} onValueChange={setStatus}>
                    <SelectTrigger id="user-status" aria-label={filterStatus}>
                      <SelectValue placeholder={filterStatus} />
                    </SelectTrigger>
                    <SelectContent>
                      {statusOptions.map(([value, label]) => (
                        <SelectItem key={value} value={value}>
                          {label?.trim()}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </Field>
              ) : null}
              {reasonRequired ? (
                <Field
                  id="user-status-reason"
                  label={reasonRequired}
                  description={reasonFieldDesc}
                  className="sm:col-span-2 xl:col-span-2"
                >
                  <Textarea
                    id="user-status-reason"
                    value={reason}
                    onChange={(e) => setReason(e.target.value)}
                    placeholder={reasonRequired}
                  />
                </Field>
              ) : null}
            </div>
            {saveLabel && pendingSave ? (
              <Button
                type="button"
                isLoading={statusMut.isPending}
                pendingLabel={pendingSuspend || pendingSave}
                disabled={!status.trim() || (status !== "active" && !reason.trim())}
                onClick={() => {
                  if (!requireStepUp()) return;
                  void statusMut.mutate({
                    status,
                    reason,
                  });
                }}
              >
                {saveLabel}
              </Button>
            ) : null}
          </CardContent>
        </Card>
        {notesLabel ? (
          <Card>
            <CardContent className="space-y-4 pt-6">
              {showEditSectionHeading && editSectionLabel ? (
                <p className="text-sm font-medium text-foreground">{editSectionLabel}</p>
              ) : null}
              <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
                <Field
                  id="user-notes"
                  label={notesLabel}
                  description={notesDesc}
                  className="sm:col-span-2 xl:col-span-3"
                >
                  <Textarea
                    id="user-notes"
                    value={notes}
                    onChange={(e) => setNotes(e.target.value)}
                    placeholder={notesLabel}
                  />
                </Field>
              </div>
              {saveLabel && pendingSave ? (
                <Button
                  type="button"
                  isLoading={notesMut.isPending}
                  pendingLabel={pendingSave}
                  onClick={() => void notesMut.mutate(notes)}
                >
                  {saveLabel}
                </Button>
              ) : null}
            </CardContent>
          </Card>
        ) : null}
      </div>
      {quota && (quotaLimitLabel || quotaTopupLabel || reasonRequired) ? (
        <Card>
          <CardContent className="space-y-4 pt-6">
            {showEditSectionHeading && editSectionLabel ? (
              <p className="text-sm font-medium text-foreground">{editSectionLabel}</p>
            ) : null}
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              {quotaLimitLabel ? (
                <Field
                  id="user-quota-limit"
                  label={quotaLimitLabel}
                  {...chromeFieldDesc(labels, "ADMIN_USER_QUOTA_LIMIT_DESC")}
                >
                  <Input
                    id="user-quota-limit"
                    value={limit}
                    onChange={(e) => setLimit(e.target.value)}
                    placeholder={quotaLimitLabel}
                  />
                </Field>
              ) : null}
              {quotaTopupLabel ? (
                <Field
                  id="user-quota-topup"
                  label={quotaTopupLabel}
                  {...chromeFieldDesc(labels, "ADMIN_USER_QUOTA_TOPUP_DESC")}
                >
                  <Input
                    id="user-quota-topup"
                    value={topup}
                    onChange={(e) => setTopup(e.target.value)}
                    placeholder={quotaTopupLabel}
                  />
                </Field>
              ) : null}
              {reasonRequired ? (
                <Field
                  id="user-quota-reason"
                  label={reasonRequired}
                  description={reasonFieldDesc}
                  className="sm:col-span-2 xl:col-span-3"
                >
                  <Textarea
                    id="user-quota-reason"
                    value={reason}
                    onChange={(e) => setReason(e.target.value)}
                    placeholder={reasonRequired}
                  />
                </Field>
              ) : null}
            </div>
            {saveLabel && pendingSave ? (
              <Button
                type="button"
                isLoading={quotaMut.isPending}
                pendingLabel={pendingSave}
                onClick={() => {
                  if (!requireStepUp()) return;
                  void quotaMut.mutate({
                    monthly_credit_limit: limit ? Number(limit) : undefined,
                    purchased_topup_credits: topup ? Number(topup) : undefined,
                    reason,
                  });
                }}
              >
                {saveLabel}
              </Button>
            ) : null}
          </CardContent>
        </Card>
      ) : null}
    </>
  );

  const activityTabBody = (
    <>
      {identities.length > 0 ? (
        <Card>
          {secIdentities ? (
            <CardHeader>
              <CardTitle className="text-sm">{secIdentities}</CardTitle>
            </CardHeader>
          ) : null}
          <CardContent>
            <DataTable
              columns={identityColumns}
              data={identities.filter((row) => Boolean(row.provider) || Boolean(row.email))}
              getRowId={(_, i) => String(i)}
              isFetching={user.isFetching && !user.isPending}
              bordered={false}
            />
          </CardContent>
        </Card>
      ) : null}
      {workspaces.length > 0 ? (
        <Card>
          {secWorkspaces ? (
            <CardHeader>
              <CardTitle className="text-sm">{secWorkspaces}</CardTitle>
            </CardHeader>
          ) : null}
          <CardContent>
            <DataTable
              columns={workspaceColumns}
              data={workspaces.filter((row) => Boolean(row.name) || Boolean(row.role))}
              getRowId={(_, i) => String(i)}
              isFetching={user.isFetching && !user.isPending}
              bordered={false}
            />
          </CardContent>
        </Card>
      ) : null}
      {devices.length > 0 ? (
        <Card>
          {secDevices ? (
            <CardHeader>
              <CardTitle className="text-sm">{secDevices}</CardTitle>
            </CardHeader>
          ) : null}
          <CardContent>
            <DataTable
              columns={deviceColumns}
              data={devices.filter((row) => Boolean(String(row.hardware_uuid || "")))}
              getRowId={(_, i) => String(i)}
              isFetching={user.isFetching && !user.isPending}
              bordered={false}
            />
          </CardContent>
        </Card>
      ) : null}
      {ja4.length > 0 ? (
        <Card>
          {secJa4 ? (
            <CardHeader>
              <CardTitle className="text-sm">{secJa4}</CardTitle>
            </CardHeader>
          ) : null}
          <CardContent>
            <DataTable
              columns={ja4Columns}
              data={ja4.filter((row) => Boolean(String(row.ja4_hash || "")))}
              getRowId={(_, i) => String(i)}
              isFetching={user.isFetching && !user.isPending}
              bordered={false}
            />
          </CardContent>
        </Card>
      ) : null}
      {linked.length > 0 ? (
        <Card>
          {secLinked ? (
            <CardHeader>
              <CardTitle className="text-sm">{secLinked}</CardTitle>
            </CardHeader>
          ) : null}
          <CardContent>
            <DataTable
              columns={linkedColumns}
              data={linked.filter((row) => Boolean(String(row.email || "")))}
              getRowId={(row, i) => String(row.id || i)}
              isFetching={user.isFetching && !user.isPending}
              bordered={false}
            />
          </CardContent>
        </Card>
      ) : null}
      {linkedHw.length > 0 ? (
        <Card>
          {secLinkedHw ? (
            <CardHeader>
              <CardTitle className="text-sm">{secLinkedHw}</CardTitle>
            </CardHeader>
          ) : null}
          <CardContent>
            <DataTable
              columns={linkedHwColumns}
              data={linkedHw.filter((row) => Boolean(String(row.email || "")))}
              getRowId={(row, i) => String(row.id || i)}
              isFetching={user.isFetching && !user.isPending}
              bordered={false}
            />
          </CardContent>
        </Card>
      ) : null}
    </>
  );

  const grantOpenLabel = grantSectionLabel || creditLabel;

  const creditsTabBody = (
    <>
      {grantOpenLabel && creditAmountLabel && reasonRequired && creditLabel && pendingCreating ? (
        <Button type="button" onClick={() => setGrantOpen(true)}>
          {grantOpenLabel}
        </Button>
      ) : null}
      {grantSectionLabel &&
      creditAmountLabel &&
      reasonRequired &&
      creditLabel &&
      pendingCreating ? (
        <Dialog
          open={grantOpen}
          onOpenChange={(open) => {
            if (!open) setGrantOpen(false);
            else setGrantOpen(true);
          }}
        >
          <DialogContent closeLabel={cancelLabel} className="max-w-2xl">
            <DialogHeader>
              {grantSectionLabel ? <DialogTitle>{grantSectionLabel}</DialogTitle> : null}
            </DialogHeader>
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              {creditAmountLabel ? (
                <Field
                  id="user-credit-amount"
                  label={creditAmountLabel}
                  {...chromeFieldDesc(labels, "ADMIN_USER_CREDIT_AMOUNT_DESC")}
                >
                  <Input
                    id="user-credit-amount"
                    value={creditAmount}
                    onChange={(e) => setCreditAmount(e.target.value)}
                    placeholder={creditAmountLabel}
                  />
                </Field>
              ) : null}
              {reasonRequired ? (
                <Field
                  id="user-credit-reason"
                  label={reasonRequired}
                  description={reasonFieldDesc}
                  className="sm:col-span-2 xl:col-span-3"
                >
                  <Textarea
                    id="user-credit-reason"
                    value={reason}
                    onChange={(e) => setReason(e.target.value)}
                    placeholder={reasonRequired}
                  />
                </Field>
              ) : null}
            </div>
            <DialogFooter>
              <Button
                type="button"
                isLoading={creditMut.isPending}
                pendingLabel={pendingCreating}
                onClick={() => {
                  if (!requireStepUp()) return;
                  void creditMut
                    .mutateAsync({
                      user_id: userId,
                      credits: Number(creditAmount),
                      reason,
                    })
                    .then(() => {
                      setCreditAmount("");
                      setReason("");
                      setGrantOpen(false);
                    });
                }}
              >
                {creditLabel}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}
      {ledgerTitle && pageSize > 0 ? (
        <Card>
          <CardContent className="space-y-4 pt-6">
            <p className="text-sm font-medium text-foreground">{ledgerTitle}</p>
            <DataTable
              columns={creditColumns}
              data={((creditGrants.data?.items as Array<Record<string, unknown>>) || []).filter(
                (row) => Boolean(String(row.id || "")),
              )}
              meta={creditGrants.data?.meta}
              onPage={setCreditSkip}
              pageDisabled={creditGrants.isFetching}
              isFetching={
                (creditGrants.isFetching && !creditGrants.isPending) || creditMut.isPending
              }
              getRowId={(row) => String(row.id)}
              bordered={false}
            />
          </CardContent>
        </Card>
      ) : null}
      {topupLedgerTitle && pageSize > 0 ? (
        <Card>
          <CardContent className="space-y-4 pt-6">
            <p className="text-sm font-medium text-foreground">{topupLedgerTitle}</p>
            <DataTable
              columns={topupColumns}
              data={((topupLedger.data?.items as Array<Record<string, unknown>>) || []).filter(
                (row) => Boolean(String(row.id || "")),
              )}
              meta={topupLedger.data?.meta}
              onPage={setTopupSkip}
              pageDisabled={topupLedger.isFetching}
              isFetching={topupLedger.isFetching && !topupLedger.isPending}
              getRowId={(row) => String(row.id)}
              bordered={false}
            />
          </CardContent>
        </Card>
      ) : null}
    </>
  );

  const dangerTabBody = (
    <div className="space-y-2">
      {dangerSectionLabel && !useTabs && (revokeLabel || logoutLabel || eraseLabel) ? (
        <p className="text-sm font-medium text-foreground">{dangerSectionLabel}</p>
      ) : null}
      <div className="flex flex-wrap gap-2">
        {revokeLabel && pendingDestructive ? (
          <Button
            type="button"
            variant="secondary"
            isLoading={revokeMut.isPending}
            pendingLabel={pendingDestructive}
            onClick={() => {
              if (!requireStepUp()) return;
              void revokeMut.mutate();
            }}
          >
            {revokeLabel}
          </Button>
        ) : null}
        {logoutLabel && reasonRequired ? (
          <Button
            type="button"
            variant="secondary"
            onClick={() => {
              if (!requireStepUp()) return;
              setConfirmAction("logout");
              setConfirmOpen(true);
            }}
          >
            {logoutLabel}
          </Button>
        ) : null}
        {eraseLabel && reasonRequired ? (
          <Button
            type="button"
            variant="destructive"
            onClick={() => {
              if (!requireStepUp()) return;
              setConfirmAction("erase");
              setConfirmOpen(true);
            }}
          >
            {eraseLabel}
          </Button>
        ) : null}
        {exportLabel && pendingSave && reasonRequired ? (
          <Button
            type="button"
            variant="outline"
            isLoading={exportMut.isPending}
            pendingLabel={pendingSave}
            onClick={() => {
              if (!requireStepUp()) return;
              void exportMut.mutate(reason);
            }}
          >
            {exportLabel}
          </Button>
        ) : null}
      </div>
    </div>
  );

  const mainSections = useTabs ? (
    <Tabs defaultValue={defaultTab}>
      <TabsList>
        {tabEdit ? <TabsTrigger value="edit">{tabEdit}</TabsTrigger> : null}
        {tabActivity ? <TabsTrigger value="activity">{tabActivity}</TabsTrigger> : null}
        {tabCredits ? <TabsTrigger value="credits">{tabCredits}</TabsTrigger> : null}
        {tabDanger ? <TabsTrigger value="danger">{tabDanger}</TabsTrigger> : null}
      </TabsList>
      {tabEdit ? (
        <TabsContent value="edit" className="mt-4 space-y-4">
          {editTabBody}
          {!tabActivity ? activityTabBody : null}
          {!tabCredits ? creditsTabBody : null}
          {!tabDanger ? dangerTabBody : null}
        </TabsContent>
      ) : null}
      {tabActivity ? (
        <TabsContent value="activity" className="mt-4 space-y-4">
          {!tabEdit ? (
            <>
              {editTabBody}
              {activityTabBody}
              {!tabCredits ? creditsTabBody : null}
              {!tabDanger ? dangerTabBody : null}
            </>
          ) : (
            activityTabBody
          )}
        </TabsContent>
      ) : null}
      {tabCredits ? (
        <TabsContent value="credits" className="mt-4 space-y-4">
          {!tabEdit && !tabActivity ? (
            <>
              {editTabBody}
              {activityTabBody}
              {creditsTabBody}
              {!tabDanger ? dangerTabBody : null}
            </>
          ) : (
            creditsTabBody
          )}
        </TabsContent>
      ) : null}
      {tabDanger ? (
        <TabsContent value="danger" className="mt-4 space-y-4">
          {!tabEdit && !tabActivity && !tabCredits ? (
            <>
              {editTabBody}
              {activityTabBody}
              {creditsTabBody}
              {dangerTabBody}
            </>
          ) : (
            dangerTabBody
          )}
        </TabsContent>
      ) : null}
    </Tabs>
  ) : (
    <div className="space-y-4">
      {editTabBody}
      {activityTabBody}
      {creditsTabBody}
      {dangerTabBody}
    </div>
  );

  return (
    <div className="space-y-6">
      {stepErr ? <p className="text-sm text-destructive">{stepErr}</p> : null}
      {email ? <h1 className="text-xl font-semibold tracking-tight">{email}</h1> : null}
      {providerCol && authProvider ? (
        <p className="text-sm text-muted-foreground">
          {providerCol}: <span className="font-mono text-foreground">{authProvider}</span>
        </p>
      ) : null}
      {loginCountryLabel && loginCountry ? (
        <p className="text-sm text-muted-foreground">
          {loginCountryLabel}: <span className="font-mono text-foreground">{loginCountry}</span>
        </p>
      ) : null}
      {billToCountryLabel && billToCountry ? (
        <p className="text-sm text-muted-foreground">
          {billToCountryLabel}: <span className="font-mono text-foreground">{billToCountry}</span>
        </p>
      ) : null}
      {mainSections}
      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        description={reasonRequired}
        cancelLabel={cancelLabel}
        confirmLabel={confirmAction === "erase" ? eraseLabel : logoutLabel}
        pendingLabel={pendingDestructive}
        isPending={logoutMut.isPending || eraseMut.isPending}
        destructive={confirmAction === "erase"}
        onConfirm={() => {
          if (!requireStepUp()) return;
          if (confirmAction === "logout") {
            void logoutMut.mutate(reason, {
              onSuccess: () => setConfirmOpen(false),
            });
            return;
          }
          if (confirmAction === "erase") {
            void eraseMut.mutate(reason, {
              onSuccess: () => setConfirmOpen(false),
            });
          }
        }}
      />
    </div>
  );
}
