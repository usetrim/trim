"use client";

import { StepUpBar } from "@/components/admin/step-up-bar";
import { DualTableSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { type ColumnDef, DataTable } from "@/components/ui/data-table";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { usePatchAuthSettings } from "@/hooks/mutations/auth";
import { useAdminAuthSettings, useAdminOAuthScopes } from "@/hooks/queries/auth";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminToken } from "@/hooks/use-admin-token";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { useRequireStepUp } from "@/hooks/use-require-step-up";
import { useCallback, useEffect, useMemo, useState } from "react";

export function AuthSettingsClient({ initialToken }: { initialToken?: string }) {
  const token = useAdminToken(initialToken);
  const chrome = useAdminNavChrome(token);
  const settings = useAdminAuthSettings(token);
  const scopes = useAdminOAuthScopes(token);
  const patch = usePatchAuthSettings(token);
  const { err: stepErr, requireStepUp } = useRequireStepUp(token);
  const [editOpen, setEditOpen] = useState(false);
  const [draftSelected, setDraftSelected] = useState<string[]>([]);
  const title = chrome.data?.ADMIN_NAV_AUTH?.trim() || "";
  const pending =
    chrome.data?.ADMIN_PENDING_UPDATING?.trim() || chrome.data?.ADMIN_PENDING_SAVING?.trim() || "";
  const saveLabel = chrome.data?.ADMIN_ACTION_SAVE?.trim() || "";
  const providersTitle = chrome.data?.ADMIN_AUTH_PROVIDERS_TITLE?.trim() || "";
  const editSectionLabel = chrome.data?.ADMIN_SECTION_EDIT?.trim() || "";
  const scopesTitle = chrome.data?.ADMIN_AUTH_SCOPES_TITLE?.trim() || "";
  const colScope = chrome.data?.ADMIN_AUTH_COL_SCOPE?.trim() || "";
  const colValue = chrome.data?.ADMIN_AUTH_COL_VALUE?.trim() || "";
  const statusActive = chrome.data?.ADMIN_STATUS_ACTIVE?.trim() || "";
  const closeLabel =
    chrome.data?.ADMIN_DIALOG_CLOSE?.trim() || chrome.data?.ADMIN_CLOSE?.trim() || "";

  const allowedProviders = settings.data?.allowed_providers || [];

  useEffect(() => {
    if (!settings.data || editOpen) return;
    setDraftSelected([...(settings.data.allowed_providers || [])]);
  }, [settings.data, editOpen]);

  const openEdit = useCallback(() => {
    setDraftSelected([...allowedProviders]);
    setEditOpen(true);
  }, [allowedProviders]);

  const scopeRows = useMemo(() => {
    const bag = scopes.data?.scopes || {};
    return Object.entries(bag)
      .filter(([code, body]) => Boolean(code) && Boolean(body))
      .map(([code, body]) => ({ code, body }));
  }, [scopes.data]);

  const scopeColumns = useMemo<ColumnDef<{ code: string; body: string }>[]>(
    () => [
      {
        id: "code",
        header: colScope,
        cell: ({ row }) => <span className="text-muted-foreground">{row.original.code}</span>,
      },
      {
        id: "body",
        header: colValue,
        cell: ({ row }) => row.original.body,
      },
    ],
    [colScope, colValue],
  );

  type ProviderRow = { id: string; display_name: string };

  const providerColumns = useMemo<ColumnDef<ProviderRow>[]>(
    () => [
      {
        id: "display_name",
        header: providersTitle,
        cell: ({ row }) => row.original.display_name,
      },
      {
        id: "enabled",
        header: "",
        cell: ({ row }) => (allowedProviders.includes(row.original.id) ? statusActive : ""),
      },
    ],
    [providersTitle, allowedProviders, statusActive],
  );

  const bootReady = useListBootReady(1, Boolean(chrome.data), queryContentReady(settings));
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    const showScopes = Boolean(colScope && colValue);
    return (
      <div className="space-y-4">
        <Card>
          <CardContent className="pt-6">
            <Skeleton className="h-24 w-full rounded-md" />
          </CardContent>
        </Card>
        <DualTableSkeleton
          title={title || undefined}
          primaryColumns={[{ label: providersTitle || "" }, { label: "" }]}
          secondaryColumns={[
            { label: showScopes ? colScope || "" : "" },
            { label: showScopes ? colValue || "" : "" },
          ]}
          rows={6}
          showFilters={false}
        />
      </div>
    );
  }

  const catalog = (settings.data?.catalog || []).filter(
    (item): item is { id: string; display_name: string } => Boolean(item.id && item.display_name),
  );

  return (
    <div className="space-y-4">
      <StepUpBar token={token || ""} allowEnroll />
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      {stepErr ? <p className="text-sm text-destructive">{stepErr}</p> : null}
      <Card>
        <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2 space-y-0">
          {providersTitle ? <CardTitle className="text-sm">{providersTitle}</CardTitle> : null}
          {editSectionLabel ? (
            <Button type="button" size="sm" variant="outline" onClick={openEdit}>
              {editSectionLabel}
            </Button>
          ) : null}
        </CardHeader>
        <CardContent>
          <DataTable
            columns={providerColumns}
            data={catalog}
            getRowId={(row) => row.id}
            className="w-full"
            isFetching={settings.isFetching || patch.isPending}
            bordered={false}
          />
        </CardContent>
      </Card>
      {scopesTitle ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">{scopesTitle}</CardTitle>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={scopeColumns}
              data={scopeRows}
              getRowId={(row) => row.code}
              isFetching={scopes.isFetching && !scopes.isPending}
              bordered={false}
            />
          </CardContent>
        </Card>
      ) : null}
      <Dialog
        open={editOpen}
        onOpenChange={(open) => {
          if (!open) {
            setEditOpen(false);
            setDraftSelected([...allowedProviders]);
          } else {
            setEditOpen(true);
          }
        }}
      >
        <DialogContent closeLabel={closeLabel} className="max-w-lg">
          <DialogHeader>
            {editSectionLabel ? <DialogTitle>{editSectionLabel}</DialogTitle> : null}
          </DialogHeader>
          {providersTitle ? (
            <p className="text-sm font-medium text-foreground">{providersTitle}</p>
          ) : null}
          <ul className="space-y-2">
            {catalog.map((item) => {
              const checked = draftSelected.includes(item.id);
              return (
                <li key={item.id} className="flex items-center gap-2">
                  <Checkbox
                    checked={checked}
                    aria-label={item.display_name}
                    onCheckedChange={(v) => {
                      if (v === true) {
                        setDraftSelected((prev) =>
                          prev.includes(item.id) ? prev : [...prev, item.id],
                        );
                      } else {
                        setDraftSelected((prev) => prev.filter((id) => id !== item.id));
                      }
                    }}
                  />
                  <span className="text-sm text-foreground">{item.display_name}</span>
                </li>
              );
            })}
          </ul>
          {saveLabel ? (
            <DialogFooter>
              <Button
                type="button"
                isLoading={patch.isPending}
                pendingLabel={pending || saveLabel}
                disabled={draftSelected.length < 1}
                onClick={() => {
                  if (!requireStepUp()) return;
                  void patch.mutate(draftSelected, {
                    onSuccess: () => setEditOpen(false),
                  });
                }}
              >
                {saveLabel}
              </Button>
            </DialogFooter>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}
