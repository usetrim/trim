"use client";

import { OAuthProviderIcon } from "@/components/auth/oauth-provider-icon";
import { ApiKeysListSkeleton, SettingsPageSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  type ColumnDef,
  DataTable,
  type RowSelectionState,
  getSelectedRowIds,
} from "@/components/ui/data-table";
import {
  type DataTableRowAction,
  DataTableRowActions,
} from "@/components/ui/data-table-row-actions";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import {
  useCreateApiKey,
  useRegisterApiKeyDevice,
  useRemoveApiKeyDevice,
  useRevokeApiKey,
} from "@/hooks/mutations/api-keys";
import { usePatchPreferences } from "@/hooks/mutations/me";
import { useApiKeyDevices, useApiKeys } from "@/hooks/queries/api-keys";
import { useAuthProviders } from "@/hooks/queries/auth";
import { usePreferences } from "@/hooks/queries/me";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { skeletonPageRows, useDefaultPageSize } from "@/hooks/use-default-page-size";
import { useDeferredDialogSelection } from "@/hooks/use-deferred-dialog-selection";
import { authProviderSlotCount } from "@/lib/auth-providers";
import { formatDateTimeShort } from "@/lib/format-datetime";
import { buildOAuthProviderOptions } from "@/lib/oauth-provider-options";
import { settingsSkeletonChrome } from "@/lib/skeleton-chrome";
import { createClient } from "@/lib/supabase/client";
import { toastApiError, toastApiSuccess } from "@/lib/toast-api";
import type { ApiKeyItem } from "@/types/api-keys";
import type { PreferencesResponse } from "@/types/me";
import { useSearchParams } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";

const COPY_FEEDBACK_MS = 1600;

type LinkedIdentity = PreferencesResponse["linked_identities"][number];

export function SettingsClient({ accessToken }: { accessToken?: string }) {
  const prefs = usePreferences(accessToken);
  const patch = usePatchPreferences(accessToken);
  const authProviders = useAuthProviders();
  const searchParams = useSearchParams();
  const pageChrome = settingsSkeletonChrome(authProviders.data?.site);
  const dialogCancel = authProviders.data?.site?.dialog_cancel?.trim() || "";
  const htmlLang = authProviders.data?.site?.html_lang?.trim() || "";
  const [keysSkip, setKeysSkip] = useState(0);
  const [keysSearchInput, setKeysSearchInput] = useState("");
  const keysSearch = useDebouncedValue(keysSearchInput.trim(), 250);
  const [keysSel, setKeysSel] = useState<RowSelectionState>({});
  const [revokeConfirmId, setRevokeConfirmId] = useState<string | null>(null);
  const [bulkRevokeConfirm, setBulkRevokeConfirm] = useState(false);
  const [bulkRevokePending, setBulkRevokePending] = useState(false);
  const [unlinkIdentityId, setUnlinkIdentityId] = useState<string | null>(null);
  const [unlinkBlockedOpen, setUnlinkBlockedOpen] = useState(false);
  const [pendingLinkProvider, setPendingLinkProvider] = useState<string | null>(null);
  const [unlinking, setUnlinking] = useState(false);
  const [accountError, setAccountError] = useState<string | null>(null);
  const pageSize = useDefaultPageSize();
  const skeletonRows = skeletonPageRows(pageSize);
  const apiKeys = useApiKeys(
    accessToken,
    keysSkip,
    pageSize > 0 ? pageSize : undefined,
    keysSearch,
  );
  const createKey = useCreateApiKey(accessToken);
  const revokeKey = useRevokeApiKey(accessToken);
  const registerDevice = useRegisterApiKeyDevice(accessToken);
  const removeDevice = useRemoveApiKeyDevice(accessToken);

  const [tier, setTier] = useState("");
  const [engine, setEngine] = useState("");
  const [target, setTarget] = useState<number | null>(null);
  const [autoStart, setAutoStart] = useState<boolean | null>(null);
  const [dirty, setDirty] = useState(false);
  const [freshKey, setFreshKey] = useState<string | null>(null);
  const [copyLabel, setCopyLabel] = useState<string | null>(null);
  const copyResetRef = useRef<number | null>(null);
  const {
    open: deviceDialogOpen,
    selectedId: deviceKeyIdRaw,
    openWith: openDeviceDialog,
    close: closeDeviceSelection,
    onOpenChange: onDeviceOpenChange,
  } = useDeferredDialogSelection();
  const deviceKeyId = deviceKeyIdRaw.trim() ? deviceKeyIdRaw : null;
  const [deviceHw, setDeviceHw] = useState("");
  const [deviceAgent, setDeviceAgent] = useState("ide");
  const [deviceError, setDeviceError] = useState<string | null>(null);
  const keyDevices = useApiKeyDevices(accessToken, deviceKeyId);

  useEffect(() => {
    return () => {
      if (copyResetRef.current != null) {
        window.clearTimeout(copyResetRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (!prefs.data) return;
    setTier(prefs.data.compression_tier);
    setEngine(prefs.data.deep_engine);
    setTarget(prefs.data.deep_target_token);
    setAutoStart(Boolean(prefs.data.auto_start_with_ide));
    setDirty(false);
  }, [prefs.data]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setKeysSkip(0);
  }, [keysSearch]);

  useEffect(() => {
    const errorParam = searchParams.get("error");
    if (!errorParam || !authProviders.data) return;
    const chrome = authProviders.data;
    if (errorParam === "oauth_link_failed") {
      const msg = chrome.login_oauth_link_failed || chrome.login_failed_message || "";
      setAccountError(msg);
      toastApiError(new Error(msg));
      return;
    }
    if (errorParam === "oauth_email_missing") {
      const msg = chrome.login_oauth_email_missing || chrome.login_failed_message || "";
      setAccountError(msg);
      toastApiError(new Error(msg));
      return;
    }
    if (errorParam === "oauth_email_denied") {
      const msg = chrome.login_oauth_email_denied || chrome.login_failed_message || "";
      setAccountError(msg);
      toastApiError(new Error(msg));
      return;
    }
    if (errorParam === "provider_not_allowed" || errorParam === "provider_missing") {
      const msg = chrome.login_provider_disabled_message || "";
      setAccountError(msg);
      toastApiError(new Error(msg));
      return;
    }
    if (errorParam === "auth_providers_unavailable") {
      const msg = chrome.login_auth_providers_unavailable || "";
      setAccountError(msg);
      toastApiError(new Error(msg));
      return;
    }
    if (errorParam === "missing_supabase_env") {
      const msg = chrome.login_missing_supabase_env || "";
      setAccountError(msg);
      toastApiError(new Error(msg));
      return;
    }
    const fallback = chrome.login_failed_message || "";
    setAccountError(fallback);
    toastApiError(new Error(fallback));
  }, [authProviders.data, searchParams]);

  const identityRowActions = prefs.data?.table_row_actions?.trim() || "";

  const identityColumns = useMemo<ColumnDef<LinkedIdentity>[]>(
    () => [
      {
        id: "provider",
        header: prefs.data?.account_title || "",
        cell: ({ row }) => {
          const identity = row.original;
          const prefix = prefs.data?.auth_provider_prefix || "";
          return (
            <div className="flex min-w-0 flex-wrap items-center gap-3">
              <OAuthProviderIcon provider={identity.provider} />
              {prefix ? <span className="text-[var(--trim-muted)]">{prefix}</span> : null}
              <span className="text-[var(--trim-fg)]">{identity.provider_display}</span>
              {identity.email ? (
                <span className="truncate text-[var(--trim-muted)]">{identity.email}</span>
              ) : null}
              {identity.is_primary && prefs.data?.identity_primary_label ? (
                <span className="rounded-md border border-[var(--trim-border)] px-2 py-0.5 text-xs text-[var(--trim-muted)]">
                  {prefs.data.identity_primary_label}
                </span>
              ) : null}
              {identity.is_last_used && prefs.data?.identity_last_used_label ? (
                <span className="rounded-md border border-[var(--trim-border)] px-2 py-0.5 text-xs text-[var(--trim-muted)]">
                  {prefs.data.identity_last_used_label}
                </span>
              ) : null}
            </div>
          );
        },
      },
      {
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const identity = row.original;
          const unlinkLabel = prefs.data?.unlink_action_label?.trim() || "";
          if (!identityRowActions || !unlinkLabel || !dialogCancel) {
            return null;
          }
          return (
            <DataTableRowActions
              triggerLabel={identityRowActions}
              actions={[
                {
                  id: "unlink",
                  label: unlinkLabel,
                  destructive: true,
                  // Keep clickable when last provider: show blocked toast instead of a dead menu item.
                  onSelect: () => {
                    if (!prefs.data?.can_unlink) {
                      const blocked = prefs.data?.unlink_last_blocked_message?.trim() || "";
                      setAccountError(blocked || null);
                      if (blocked) {
                        toastApiError(new Error(blocked));
                        setUnlinkBlockedOpen(true);
                      }
                      return;
                    }
                    setAccountError(null);
                    setUnlinkIdentityId(identity.identity_id);
                  },
                },
              ]}
            />
          );
        },
      },
    ],
    [prefs.data, dialogCancel, identityRowActions],
  );

  const keysSelectAll = apiKeys.data?.table_select_all?.trim() || "";
  const keysSelectRow = apiKeys.data?.table_select_row?.trim() || "";
  const keysSelectedFmt = apiKeys.data?.table_selected_fmt?.trim() || "";
  const keysRowActions = apiKeys.data?.table_row_actions?.trim() || "";
  const keysBulkRevoke = apiKeys.data?.table_bulk_revoke?.trim() || "";
  const keysClearSel = apiKeys.data?.table_clear_selection?.trim() || "";
  const keysSelectionChrome =
    keysSelectAll && keysSelectRow
      ? {
          selectAllLabel: keysSelectAll,
          selectRowLabel: keysSelectRow,
          selectedCountFmt: keysSelectedFmt || undefined,
        }
      : null;

  const apiKeyColumns = useMemo<ColumnDef<ApiKeyItem>[]>(
    () => [
      {
        id: "key",
        header: prefs.data?.keys_title || "",
        cell: ({ row }) => {
          const k = row.original;
          const sep = authProviders.data?.site?.meta_sep || "";
          const countFmt = apiKeys.data?.device_count_fmt?.trim() || "";
          const boundLabel = k.device_bound
            ? apiKeys.data?.device_bound_label
            : apiKeys.data?.device_unbound_label;
          const countLabel =
            countFmt && typeof k.device_count === "number"
              ? countFmt.replace("%d", String(k.device_count))
              : "";
          return (
            <div className="min-w-0">
              <p className="font-mono text-[var(--trim-fg)]">
                {k.key_prefix}
                {apiKeys.data?.key_prefix_ellipsis || ""}
              </p>
              <p className="text-xs text-[var(--trim-muted)]">
                {k.revoked ? apiKeys.data?.status_revoked_label : apiKeys.data?.status_active_label}
                {apiKeys.data?.created_prefix_label
                  ? `${sep}${apiKeys.data.created_prefix_label} ${formatDateTimeShort(
                      k.created_at,
                      htmlLang,
                    )}`
                  : ""}
              </p>
              {boundLabel ? (
                <p className="text-xs text-[var(--trim-muted)]">
                  {boundLabel}
                  {countLabel ? `${sep}${countLabel}` : ""}
                </p>
              ) : null}
            </div>
          );
        },
      },
      {
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const k = row.original;
          if (k.revoked) return null;
          const revokeLabel = apiKeys.data?.revoke_action_label?.trim() || "";
          const registerLabel = apiKeys.data?.device_register_label?.trim() || "";
          if (
            (!revokeLabel || !apiKeys.data?.revoke_confirm_message || !dialogCancel) &&
            !registerLabel
          ) {
            return null;
          }
          const actions: DataTableRowAction[] = [];
          if (registerLabel) {
            actions.push({
              id: "register-device",
              label: registerLabel,
              onSelect: () => {
                openDeviceDialog(k.id);
                setDeviceHw("");
                setDeviceAgent("ide");
                setDeviceError(null);
              },
            });
          }
          if (revokeLabel && apiKeys.data?.revoke_confirm_message && dialogCancel) {
            actions.push({
              id: "revoke",
              label: revokeLabel,
              destructive: true,
              onSelect: () => setRevokeConfirmId(k.id),
            });
          }
          if (!actions.length || !keysRowActions) return null;
          return <DataTableRowActions triggerLabel={keysRowActions} actions={actions} />;
        },
      },
    ],
    [
      prefs.data,
      apiKeys.data,
      authProviders.data,
      dialogCancel,
      keysRowActions,
      openDeviceDialog,
      htmlLang,
    ],
  );

  if (!prefs.data && !prefs.error) {
    return (
      <SettingsPageSkeleton
        chrome={pageChrome}
        keyRows={skeletonRows}
        providerSlots={authProviderSlotCount(
          authProviders.data?.provider_count ?? authProviders.data?.items?.length,
        )}
        identityRows={Math.max(
          1,
          authProviderSlotCount(
            authProviders.data?.provider_count ?? authProviders.data?.items?.length,
          ) || 1,
        )}
        embedded
      />
    );
  }

  async function connectProvider(provider: string) {
    setPendingLinkProvider(provider);
    setAccountError(null);
    try {
      const origin = window.location.origin;
      const callbackPath = (authProviders.data?.site?.path_auth_callback || "").trim();
      const settingsPath = (authProviders.data?.site?.path_settings || "").trim();
      const intent = (authProviders.data?.oauth_link_intent || "").trim();
      const failed =
        authProviders.data?.login_oauth_link_failed ||
        authProviders.data?.login_failed_message ||
        "";
      if (!callbackPath || !settingsPath || !intent) {
        throw new Error(failed);
      }
      const options = buildOAuthProviderOptions(
        provider,
        authProviders.data,
        `${origin}${callbackPath}?next=${encodeURIComponent(
          settingsPath,
        )}&intent=${encodeURIComponent(intent)}`,
        failed,
      );
      const supabase = createClient();
      const { error } = await supabase.auth.linkIdentity({
        provider: provider as "google" | "github" | "gitlab",
        options,
      });
      if (error) {
        throw new Error(failed);
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : "";
      setAccountError(msg);
      toastApiError(e);
      setPendingLinkProvider(null);
    }
  }

  async function disconnectIdentity(identityId: string) {
    if (!prefs.data?.can_unlink) {
      const blocked = prefs.data?.unlink_last_blocked_message || "";
      setAccountError(blocked);
      toastApiError(new Error(blocked));
      return;
    }
    setUnlinking(true);
    setAccountError(null);
    try {
      const supabase = createClient();
      const { data, error: userError } = await supabase.auth.getUser();
      if (userError || !data.user) {
        throw new Error(prefs.data?.unlink_failed_message || "");
      }
      const identity = (data.user.identities ?? []).find(
        (item) => item.identity_id === identityId || item.id === identityId,
      );
      if (!identity) {
        throw new Error(prefs.data?.unlink_failed_message || "");
      }
      const { error } = await supabase.auth.unlinkIdentity(identity);
      if (error) {
        throw new Error(prefs.data?.unlink_failed_message || "");
      }
      await prefs.refetch();
      toastApiSuccess({ message: prefs.data?.unlink_done_message || "" });
    } catch (e) {
      const msg = e instanceof Error ? e.message : "";
      setAccountError(msg);
      toastApiError(e);
    } finally {
      setUnlinking(false);
      setUnlinkIdentityId(null);
    }
  }

  async function copyKey(value: string) {
    const copied = createKey.data?.copied_action_label?.trim() || "";
    if (!copied || !value.trim()) return;
    if (copyResetRef.current != null) {
      window.clearTimeout(copyResetRef.current);
      copyResetRef.current = null;
    }
    try {
      await navigator.clipboard.writeText(value);
      // Clipboard is near-instant: never use Button isLoading (spinner/width flash).
      setCopyLabel(copied);
      copyResetRef.current = window.setTimeout(() => {
        setCopyLabel(null);
        copyResetRef.current = null;
      }, COPY_FEEDBACK_MS);
    } catch {
      setCopyLabel(null);
    }
  }

  function closeDeviceDialog() {
    closeDeviceSelection();
    setDeviceHw("");
    setDeviceAgent("ide");
    setDeviceError(null);
  }

  const keysMeta = apiKeys.data?.meta;
  const selectedKeyIds = getSelectedRowIds(keysSel);
  const selectedActiveKeyIds = selectedKeyIds.filter((id) => {
    const item = (apiKeys.data?.items ?? []).find((k) => k.id === id);
    return item && !item.revoked;
  });
  const showKeysBulkRevoke = Boolean(
    keysBulkRevoke && apiKeys.data?.revoke_confirm_message && apiKeys.data?.revoke_action_label,
  );

  async function confirmBulkRevoke() {
    if (!selectedActiveKeyIds.length) {
      setBulkRevokeConfirm(false);
      return;
    }
    setBulkRevokePending(true);
    try {
      for (const id of selectedActiveKeyIds) {
        await revokeKey.mutateAsync(id);
      }
      setKeysSel({});
      setBulkRevokeConfirm(false);
    } finally {
      setBulkRevokePending(false);
    }
  }

  return (
    <div className="w-full space-y-6">
      <div className="min-w-0">
        <h1 className="text-2xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-3xl">
          {prefs.data?.page_title}
        </h1>
        <p className="mt-1 max-w-2xl text-sm text-[var(--trim-muted)]">
          {prefs.data?.page_description}
        </p>
      </div>

      <div className="w-full">
        {prefs.error ? (
          <p className="mb-4 text-sm text-destructive">
            {(prefs.error instanceof Error && prefs.error.message) || ""}
          </p>
        ) : null}

        {prefs.data?.account_title ? (
          <Card className="mb-6 border-[var(--trim-border)] bg-[var(--trim-panel)]">
            <CardHeader>
              <CardTitle className="text-[var(--trim-fg)]">{prefs.data.account_title}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {accountError ? <p className="text-sm text-destructive">{accountError}</p> : null}
              <DataTable
                columns={identityColumns}
                data={prefs.data.linked_identities ?? []}
                bordered={false}
                getRowId={(row) => row.identity_id}
                isFetching={prefs.isFetching && !prefs.isPending}
              />
              {prefs.data.link_hint ? (
                <p
                  className={
                    (prefs.data.linked_identities ?? []).length === 0
                      ? "py-2 text-sm text-[var(--trim-muted)]"
                      : "text-xs text-[var(--trim-muted)]"
                  }
                >
                  {prefs.data.link_hint}
                </p>
              ) : null}
              {(prefs.data.linkable_providers ?? []).length > 0 ? (
                <div className="flex flex-wrap gap-2">
                  {prefs.data.linkable_providers.map((item) => (
                    <Button
                      key={item.id}
                      type="button"
                      variant="secondary"
                      disabled={pendingLinkProvider !== null}
                      isLoading={pendingLinkProvider === item.id}
                      pendingLabel={item.pending_label}
                      onClick={() => void connectProvider(item.id)}
                    >
                      <OAuthProviderIcon provider={item.id} />
                      {item.action_label}
                    </Button>
                  ))}
                </div>
              ) : null}
            </CardContent>
          </Card>
        ) : null}

        <div className="grid gap-6 lg:grid-cols-2">
          <Card className="border-[var(--trim-border)] bg-[var(--trim-panel)]">
            <CardHeader>
              <CardTitle className="text-[var(--trim-fg)]">{prefs.data?.tier_title}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <FetchProgressBar
                active={(prefs.isFetching && !prefs.isPending) || patch.isPending}
                className="-mx-6 mb-2 w-[calc(100%+3rem)]"
              />
              <div className="flex items-center justify-between gap-4 rounded-lg border border-[var(--trim-border)] bg-[var(--trim-bg)] px-4 py-3">
                <div>
                  <p className="text-sm font-medium text-[var(--trim-fg)]">
                    {prefs.data?.deep_label}
                  </p>
                  <p className="text-xs text-[var(--trim-muted)]">{prefs.data?.deep_hint}</p>
                </div>
                <Switch
                  checked={Boolean(prefs.data?.deep_tier_id) && tier === prefs.data?.deep_tier_id}
                  onCheckedChange={(on) => {
                    const fastId = (prefs.data?.fast_tier_id || "").trim();
                    const deepId = (prefs.data?.deep_tier_id || "").trim();
                    if (!fastId || !deepId) return;
                    setTier(on ? deepId : fastId);
                    setDirty(true);
                  }}
                  aria-label={prefs.data?.deep_label || undefined}
                />
              </div>

              {prefs.data?.auto_start_label ? (
                <div className="flex items-center justify-between gap-4 rounded-lg border border-[var(--trim-border)] bg-[var(--trim-bg)] px-4 py-3">
                  <div>
                    <p className="text-sm font-medium text-[var(--trim-fg)]">
                      {prefs.data.auto_start_title}
                    </p>
                    {prefs.data.auto_start_hint ? (
                      <p className="text-xs text-[var(--trim-muted)]">
                        {prefs.data.auto_start_hint}
                      </p>
                    ) : null}
                    {prefs.data.auto_start_note ? (
                      <p className="mt-1 text-xs text-[var(--trim-muted)]">
                        {prefs.data.auto_start_note}
                      </p>
                    ) : null}
                    {prefs.data.local_agent_title && prefs.data.local_agent_status_label ? (
                      <p className="mt-2 text-xs text-[var(--trim-muted)]">
                        <span className="text-[var(--trim-fg)]">
                          {prefs.data.local_agent_title}
                        </span>
                        {": "}
                        {prefs.data.local_agent_status_label}
                        {prefs.data.local_agent_hint ? (
                          <span className="block pt-1">{prefs.data.local_agent_hint}</span>
                        ) : null}
                      </p>
                    ) : null}
                  </div>
                  <Switch
                    checked={autoStart === true}
                    disabled={autoStart === null || patch.isPending}
                    onCheckedChange={(on) => {
                      setAutoStart(on);
                      setDirty(true);
                      // One-click: persist auto-start immediately (does not require Save for this bit).
                      patch.mutate(
                        { auto_start_with_ide: on },
                        {
                          onSuccess: () => setDirty(false),
                          onError: () => {
                            setAutoStart(!on);
                          },
                        },
                      );
                    }}
                    aria-label={prefs.data.auto_start_label}
                  />
                </div>
              ) : null}

              <div className="space-y-2">
                {prefs.data?.engine_label ? (
                  <Field
                    id="deep-engine"
                    label={prefs.data.engine_label}
                    description={[
                      prefs.data.engine_hint,
                      (prefs.data.engine_options ?? []).find((opt) => opt.id === engine)?.hint,
                    ]
                      .map((s) => (typeof s === "string" ? s.trim() : ""))
                      .filter(Boolean)
                      .join(" ")}
                  >
                    <Select
                      value={engine}
                      disabled={!prefs.data?.deep_tier_id || tier !== prefs.data.deep_tier_id}
                      onValueChange={(v) => {
                        setEngine(v);
                        setDirty(true);
                      }}
                    >
                      <SelectTrigger id="deep-engine">
                        <SelectValue placeholder={prefs.data.engine_label} />
                      </SelectTrigger>
                      <SelectContent>
                        {(prefs.data?.engine_options ?? []).map((opt) => (
                          <SelectItem key={opt.id} value={opt.id}>
                            {opt.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </Field>
                ) : null}
              </div>

              <div className="space-y-2">
                {prefs.data?.target_label ? (
                  <Field
                    id="target-token"
                    label={prefs.data.target_label}
                    description={prefs.data.target_hint}
                  >
                    <Input
                      id="target-token"
                      type="number"
                      min={
                        prefs.data?.deep_target_token_min && prefs.data.deep_target_token_min >= 1
                          ? prefs.data.deep_target_token_min
                          : undefined
                      }
                      max={
                        prefs.data?.deep_target_token_max &&
                        prefs.data.deep_target_token_max >= (prefs.data.deep_target_token_min ?? 0)
                          ? prefs.data.deep_target_token_max
                          : undefined
                      }
                      value={target ?? ""}
                      placeholder={prefs.data.target_label}
                      disabled={
                        !prefs.data?.deep_tier_id ||
                        tier !== prefs.data.deep_tier_id ||
                        target === null
                      }
                      onChange={(e) => {
                        const n = Number(e.target.value);
                        if (!Number.isFinite(n)) return;
                        setTarget(n);
                        setDirty(true);
                      }}
                    />
                  </Field>
                ) : null}
              </div>

              {prefs.data?.live_deep_min_hint ? (
                <p className="text-sm text-[var(--trim-muted)]">{prefs.data.live_deep_min_hint}</p>
              ) : null}
              {prefs.data?.live_deep_stream_hint ? (
                <p className="text-sm text-[var(--trim-muted)]">
                  {prefs.data.live_deep_stream_hint}
                </p>
              ) : null}
              {typeof prefs.data?.live_deep_min_input_tokens === "number" ? (
                <p className="text-sm text-[var(--trim-muted)]">
                  live_deep_min_input_tokens={prefs.data.live_deep_min_input_tokens}
                  {prefs.data.live_deep_oom_policy
                    ? ` · oom=${prefs.data.live_deep_oom_policy}`
                    : ""}
                  {typeof prefs.data.live_deep_skip_on_stream === "boolean"
                    ? ` · skip_stream=${prefs.data.live_deep_skip_on_stream}`
                    : ""}
                </p>
              ) : null}

              <Button
                disabled={
                  !dirty ||
                  patch.isPending ||
                  !prefs.data?.save_action_label ||
                  !tier ||
                  !engine ||
                  target === null ||
                  target <= 0 ||
                  autoStart === null
                }
                isLoading={patch.isPending}
                pendingLabel={prefs.data?.save_pending_label || undefined}
                onClick={() => {
                  if (target === null || target <= 0) return;
                  if (autoStart === null) return;
                  patch.mutate(
                    {
                      compression_tier: tier,
                      deep_engine: engine,
                      deep_target_token: target,
                      auto_start_with_ide: autoStart,
                    },
                    { onSuccess: () => setDirty(false) },
                  );
                }}
              >
                {prefs.data?.save_action_label}
              </Button>
              {patch.isError ? (
                <p className="text-sm text-destructive">
                  {(patch.error instanceof Error && patch.error.message) || ""}
                </p>
              ) : null}
              {patch.isSuccess && !dirty ? (
                <p className="text-sm text-[var(--trim-muted)]">
                  {patch.data?.saved_message || prefs.data?.saved_message || ""}
                </p>
              ) : null}
            </CardContent>
          </Card>

          <Card className="border-[var(--trim-border)] bg-[var(--trim-panel)]">
            <CardHeader>
              <CardTitle className="text-[var(--trim-fg)]">{prefs.data?.cli_title}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm text-[var(--trim-muted)]">
              {(prefs.data?.cli_help_lines ?? []).map((line) => (
                <p key={line}>
                  <code className="text-[var(--trim-fg)]">{line}</code>
                </p>
              ))}
              {prefs.data?.note ? (
                <p className="border-t border-[var(--trim-border)] pt-3 text-xs text-[var(--trim-muted)]">
                  {prefs.data.note}
                </p>
              ) : null}
            </CardContent>
          </Card>

          <Card className="border-[var(--trim-border)] bg-[var(--trim-panel)] lg:col-span-2">
            <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-3">
              <CardTitle className="text-[var(--trim-fg)]">{prefs.data?.keys_title}</CardTitle>
              <div className="flex flex-wrap gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={!apiKeys.data?.refresh_action_label}
                  isLoading={apiKeys.isFetching}
                  pendingLabel={apiKeys.data?.refresh_pending_label || undefined}
                  onClick={() => void apiKeys.refetch()}
                >
                  {apiKeys.data?.refresh_action_label}
                </Button>
                <Button
                  size="sm"
                  disabled={!apiKeys.data?.issue_action_label}
                  isLoading={createKey.isPending}
                  pendingLabel={apiKeys.data?.issue_pending_label || undefined}
                  onClick={() => {
                    setFreshKey(null);
                    setCopyLabel(null);
                    createKey.mutate(
                      {},
                      {
                        onSuccess: (data) => {
                          setFreshKey(data.api_key);
                        },
                      },
                    );
                  }}
                >
                  {apiKeys.data?.issue_action_label}
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {apiKeys.data?.intro_message ? (
                <p className="text-sm text-[var(--trim-muted)]">{apiKeys.data.intro_message}</p>
              ) : null}
              {apiKeys.data?.search_placeholder?.trim() ? (
                <Field
                  id="settings-api-keys-search"
                  label={apiKeys.data.search_placeholder}
                  description={apiKeys.data.search_description}
                  className="w-full sm:max-w-md"
                >
                  <Input
                    id="settings-api-keys-search"
                    value={keysSearchInput}
                    onChange={(e) => setKeysSearchInput(e.target.value)}
                    placeholder={apiKeys.data.search_placeholder}
                    autoComplete="off"
                  />
                </Field>
              ) : null}
              {createKey.error ? (
                <p className="text-sm text-destructive">
                  {(createKey.error instanceof Error && createKey.error.message) || ""}
                </p>
              ) : null}
              {apiKeys.error ? (
                <p className="text-sm text-destructive">
                  {(apiKeys.error instanceof Error && apiKeys.error.message) || ""}
                </p>
              ) : null}
              {freshKey ? (
                <div className="rounded-md border border-[var(--trim-border)] bg-[var(--trim-bg)] p-3">
                  <p className="text-xs text-[var(--trim-muted)]">
                    {createKey.data?.fresh_key_hint || ""}
                  </p>
                  <p className="mt-1 break-all font-mono text-xs text-[var(--trim-fg)]">
                    {freshKey}
                  </p>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="mt-2 min-w-[5.5rem]"
                    disabled={!createKey.data?.copy_action_label}
                    onClick={() => void copyKey(freshKey)}
                  >
                    {copyLabel || createKey.data?.copy_action_label}
                  </Button>
                </div>
              ) : null}
              {apiKeys.isPending && !apiKeys.data ? (
                <ApiKeysListSkeleton rows={skeletonRows} />
              ) : (
                <>
                  <DataTable
                    columns={apiKeyColumns}
                    data={apiKeys.data?.items ?? []}
                    bordered={false}
                    getRowId={(row) => row.id}
                    meta={keysMeta}
                    onPage={
                      keysMeta
                        ? (s) => {
                            setKeysSkip(s);
                            setKeysSel({});
                          }
                        : undefined
                    }
                    pageDisabled={apiKeys.isFetching}
                    pageInputId="settings-keys-skip-to"
                    isFetching={apiKeys.isFetching && !apiKeys.isPending}
                    selection={
                      keysSelectionChrome
                        ? {
                            chrome: keysSelectionChrome,
                            rowSelection: keysSel,
                            onRowSelectionChange: setKeysSel,
                            toolbar: showKeysBulkRevoke ? (
                              <div className="flex flex-wrap gap-2">
                                <Button
                                  type="button"
                                  size="sm"
                                  variant="destructive"
                                  disabled={!selectedActiveKeyIds.length || !dialogCancel}
                                  onClick={() => setBulkRevokeConfirm(true)}
                                >
                                  {keysBulkRevoke}
                                </Button>
                                {keysClearSel ? (
                                  <Button
                                    type="button"
                                    size="sm"
                                    variant="outline"
                                    onClick={() => setKeysSel({})}
                                  >
                                    {keysClearSel}
                                  </Button>
                                ) : null}
                              </div>
                            ) : undefined,
                          }
                        : undefined
                    }
                  />
                  {(apiKeys.data?.items ?? []).length === 0 ? (
                    <p className="py-3 text-sm text-[var(--trim-muted)]">
                      {apiKeys.data?.empty_message || ""}
                    </p>
                  ) : null}
                </>
              )}
              {revokeKey.error ? (
                <p className="text-sm text-destructive">
                  {(revokeKey.error instanceof Error && revokeKey.error.message) || ""}
                </p>
              ) : null}
            </CardContent>
          </Card>
        </div>
        <Dialog
          open={deviceDialogOpen}
          onOpenChange={(next) => {
            if (!next && !registerDevice.isPending && !removeDevice.isPending) {
              onDeviceOpenChange(false);
              setDeviceHw("");
              setDeviceAgent("ide");
              setDeviceError(null);
            }
          }}
        >
          <DialogContent closeLabel={dialogCancel} className="max-w-md">
            <DialogHeader>
              {apiKeys.data?.device_register_label ? (
                <DialogTitle>
                  {apiKeys.data.device_register_label}
                  {deviceKeyId ? ` · ${deviceKeyId.slice(0, 8)}…` : ""}
                </DialogTitle>
              ) : null}
            </DialogHeader>
            <div className="space-y-3">
              {(keyDevices.data?.items ?? []).length > 0 ? (
                <ul className="space-y-2">
                  {keyDevices.data?.items.map((d) => {
                    const removeLabel = keyDevices.data?.device_remove_label?.trim() || "";
                    return (
                      <li
                        key={d.hardware_uuid}
                        className="flex flex-wrap items-center justify-between gap-2 border-b border-[var(--trim-border)] pb-2 last:border-0"
                      >
                        <div className="min-w-0">
                          <p className="truncate font-mono text-xs text-[var(--trim-fg)]">
                            {d.hardware_uuid}
                          </p>
                          {d.agent_id ? (
                            <p className="text-xs text-[var(--trim-muted)]">{d.agent_id}</p>
                          ) : null}
                        </div>
                        {removeLabel ? (
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            disabled={removeDevice.isPending}
                            isLoading={
                              removeDevice.isPending &&
                              removeDevice.variables?.hardware_uuid === d.hardware_uuid
                            }
                            pendingLabel={keyDevices.data?.device_remove_pending || undefined}
                            onClick={() => {
                              if (!deviceKeyId) return;
                              setDeviceError(null);
                              removeDevice.mutate(
                                { keyId: deviceKeyId, hardware_uuid: d.hardware_uuid },
                                {
                                  onError: (err) => {
                                    setDeviceError(err instanceof Error ? err.message : "");
                                  },
                                },
                              );
                            }}
                          >
                            {removeLabel}
                          </Button>
                        ) : null}
                      </li>
                    );
                  })}
                </ul>
              ) : null}
              {apiKeys.data?.device_hw_label ? (
                <Field
                  id="device-hw"
                  label={apiKeys.data.device_hw_label}
                  description={apiKeys.data.device_hint}
                >
                  <Input
                    id="device-hw"
                    value={deviceHw}
                    onChange={(e) => setDeviceHw(e.target.value)}
                    placeholder={apiKeys.data.device_hw_label}
                    autoComplete="off"
                    spellCheck={false}
                  />
                </Field>
              ) : null}
              {apiKeys.data?.device_agent_label &&
              (apiKeys.data.device_agent_options ?? []).length > 0 ? (
                <Field
                  id="device-agent"
                  label={apiKeys.data.device_agent_label}
                  description={apiKeys.data.ci_agent_hint}
                >
                  <Select value={deviceAgent} onValueChange={setDeviceAgent}>
                    <SelectTrigger id="device-agent">
                      <SelectValue placeholder={apiKeys.data.device_agent_label} />
                    </SelectTrigger>
                    <SelectContent>
                      {(apiKeys.data.device_agent_options ?? []).map((opt) => (
                        <SelectItem key={opt.id} value={opt.id}>
                          {opt.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </Field>
              ) : null}
              {deviceError ? <p className="text-sm text-destructive">{deviceError}</p> : null}
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                disabled={registerDevice.isPending || removeDevice.isPending}
                onClick={closeDeviceDialog}
              >
                {dialogCancel}
              </Button>
              <Button
                type="button"
                disabled={
                  !deviceKeyId ||
                  !deviceHw.trim() ||
                  !deviceAgent.trim() ||
                  !apiKeys.data?.device_register_label ||
                  registerDevice.isPending
                }
                isLoading={registerDevice.isPending}
                pendingLabel={apiKeys.data?.device_register_pending || undefined}
                onClick={() => {
                  if (!deviceKeyId) return;
                  setDeviceError(null);
                  registerDevice.mutate(
                    {
                      keyId: deviceKeyId,
                      hardware_uuid: deviceHw.trim(),
                      agent_id: deviceAgent.trim(),
                    },
                    {
                      onSuccess: () => {
                        setDeviceHw("");
                        setDeviceError(null);
                      },
                      onError: (err) => {
                        setDeviceError(err instanceof Error ? err.message : "");
                      },
                    },
                  );
                }}
              >
                {apiKeys.data?.device_register_label}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
        <ConfirmDialog
          open={Boolean(unlinkIdentityId)}
          onOpenChange={(open) => {
            if (!open && !unlinking) setUnlinkIdentityId(null);
          }}
          description={prefs.data?.unlink_confirm_message || ""}
          cancelLabel={dialogCancel}
          confirmLabel={prefs.data?.unlink_action_label || ""}
          pendingLabel={prefs.data?.unlink_pending_label || undefined}
          isPending={unlinking}
          destructive
          onConfirm={() => {
            if (!unlinkIdentityId) return;
            void disconnectIdentity(unlinkIdentityId);
          }}
        />
        <Dialog open={unlinkBlockedOpen} onOpenChange={setUnlinkBlockedOpen}>
          <DialogContent closeLabel={dialogCancel} className="max-w-md">
            <DialogHeader>
              {prefs.data?.unlink_action_label ? (
                <DialogTitle>{prefs.data.unlink_action_label}</DialogTitle>
              ) : null}
            </DialogHeader>
            {prefs.data?.unlink_last_blocked_message ? (
              <p className="text-sm text-[var(--trim-muted)]">
                {prefs.data.unlink_last_blocked_message}
              </p>
            ) : null}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setUnlinkBlockedOpen(false)}>
                {dialogCancel}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
        <ConfirmDialog
          open={Boolean(revokeConfirmId)}
          onOpenChange={(open) => {
            if (!open && !revokeKey.isPending) setRevokeConfirmId(null);
          }}
          description={apiKeys.data?.revoke_confirm_message || ""}
          cancelLabel={dialogCancel}
          confirmLabel={apiKeys.data?.revoke_action_label || ""}
          pendingLabel={apiKeys.data?.revoke_pending_label || undefined}
          isPending={revokeKey.isPending}
          destructive
          onConfirm={() => {
            if (!revokeConfirmId) return;
            revokeKey.mutate(revokeConfirmId, {
              onSettled: () => setRevokeConfirmId(null),
              onSuccess: () =>
                setKeysSel((prev) => {
                  const next = { ...prev };
                  delete next[revokeConfirmId];
                  return next;
                }),
            });
          }}
        />
        <ConfirmDialog
          open={bulkRevokeConfirm}
          onOpenChange={(open) => {
            if (!open && !bulkRevokePending) setBulkRevokeConfirm(false);
          }}
          description={apiKeys.data?.revoke_confirm_message || ""}
          cancelLabel={dialogCancel}
          confirmLabel={keysBulkRevoke || apiKeys.data?.revoke_action_label || ""}
          pendingLabel={apiKeys.data?.revoke_pending_label || undefined}
          isPending={bulkRevokePending}
          destructive
          onConfirm={() => {
            void confirmBulkRevoke();
          }}
        />
      </div>
    </div>
  );
}
