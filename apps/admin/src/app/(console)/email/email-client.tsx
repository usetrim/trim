"use client";

import { AdminFilterBar } from "@/components/admin/admin-filter-bar";
import { DataTableSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { type ColumnDef, DataTable } from "@/components/ui/data-table";
import { DataTableRowActions } from "@/components/ui/data-table-row-actions";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field } from "@/components/ui/field";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { usePatchEmailTemplate } from "@/hooks/mutations/email";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminEmailTemplates } from "@/hooks/queries/email";
import { useAdminToken } from "@/hooks/use-admin-token";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { useDeferredDialogSelection } from "@/hooks/use-deferred-dialog-selection";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { useRequireStepUp } from "@/hooks/use-require-step-up";
import { chromeFieldDesc } from "@/lib/admin-field-desc";
import { useCallback, useEffect, useMemo, useState } from "react";

type EmailRow = {
  code: string;
  kind?: string;
  kind_label?: string;
  body?: string;
};

export function EmailTemplatesClient({
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
    open: dialogOpen,
    selectedId: selected,
    openWith,
    close: closeDialog,
    onOpenChange: onDialogOpenChange,
  } = useDeferredDialogSelection();
  const chrome = useAdminNavChrome(token);
  const templates = useAdminEmailTemplates(token, skip, pageSize, q);
  const patch = usePatchEmailTemplate(token);
  const { err: stepErr, requireStepUp } = useRequireStepUp(token);
  const [body, setBody] = useState("");
  const title = chrome.data?.ADMIN_NAV_EMAIL?.trim() || "";
  const pending =
    chrome.data?.ADMIN_PENDING_UPDATING?.trim() || chrome.data?.ADMIN_PENDING_SAVING?.trim() || "";
  const saveLabel = chrome.data?.ADMIN_ACTION_SAVE?.trim() || "";
  const colKind = chrome.data?.ADMIN_EMAIL_COL_KIND?.trim() || "";
  const colCode = chrome.data?.ADMIN_EMAIL_COL_CODE?.trim() || "";
  const filterSearch = chrome.data?.ADMIN_FILTER_SEARCH?.trim() || "";
  const filterSearchDesc = chrome.data?.ADMIN_FILTER_SEARCH_DESC?.trim() || "";
  const viewLabel = chrome.data?.ADMIN_ACTION_VIEW?.trim() || "";
  const rowActionsLabel = chrome.data?.ADMIN_TABLE_ROW_ACTIONS?.trim() || "";
  const editSectionLabel = chrome.data?.ADMIN_SECTION_EDIT?.trim() || "";
  const closeLabel =
    chrome.data?.ADMIN_DIALOG_CLOSE?.trim() || chrome.data?.ADMIN_CLOSE?.trim() || "";

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setSkip(0);
  }, [q]);

  // Body is seeded in openTemplate - do not rehydrate from list refetch while editing.
  const openTemplate = useCallback(
    (row: EmailRow) => {
      if (!row.code) return;
      openWith(row.code);
      setBody(row.body || "");
    },
    [openWith],
  );

  const columns = useMemo<ColumnDef<EmailRow>[]>(
    () => [
      {
        id: "kind",
        header: colKind,
        cell: ({ row }) => row.original.kind_label || "",
      },
      {
        id: "code",
        header: colCode,
        cell: ({ row }) => (
          <button
            type="button"
            className="font-mono text-xs text-foreground hover:underline"
            onClick={() => openTemplate(row.original)}
          >
            {row.original.code}
          </button>
        ),
      },
      {
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          if (!row.original.code || !viewLabel || !rowActionsLabel) return null;
          return (
            <DataTableRowActions
              triggerLabel={rowActionsLabel}
              actions={[
                {
                  id: "view",
                  label: viewLabel,
                  onSelect: () => openTemplate(row.original),
                },
              ]}
            />
          );
        },
      },
    ],
    [colKind, colCode, viewLabel, rowActionsLabel, openTemplate],
  );

  const bootReady = useListBootReady(pageSize, Boolean(chrome.data), queryContentReady(templates));
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    return (
      <div className="space-y-4">
        {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
        <Skeleton className="h-4 w-64" />
        <DataTableSkeleton
          title=""
          columns={[colKind, colCode].map((label) => ({ label: label || "" }))}
          rows={pageSize > 0 ? pageSize : 8}
          showFilters
          filterCount={1}
        />
      </div>
    );
  }

  const items = (templates.data?.items || []).filter((item) => Boolean(item.code)) as EmailRow[];
  const smtpLabel = templates.data?.smtp_status_label?.trim() || "";

  return (
    <div className="space-y-4">
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      {stepErr ? <p className="text-sm text-destructive">{stepErr}</p> : null}
      {smtpLabel ? (
        <p
          className={
            templates.data?.smtp_configured
              ? "text-sm text-foreground"
              : "text-sm text-muted-foreground"
          }
        >
          {smtpLabel}
        </p>
      ) : null}
      <AdminFilterBar
        filters={[
          {
            kind: "search",
            id: "email-templates-search",
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
            columns={columns}
            data={items}
            meta={templates.data?.meta}
            onPage={setSkip}
            pageDisabled={templates.isFetching}
            isFetching={(templates.isFetching && !templates.isPending) || patch.isPending}
            getRowId={(row) => row.code}
            bordered={false}
          />
        </CardContent>
      </Card>
      <Dialog open={dialogOpen} onOpenChange={onDialogOpenChange}>
        <DialogContent closeLabel={closeLabel} className="max-w-3xl">
          <DialogHeader>
            {editSectionLabel ? <DialogTitle>{editSectionLabel}</DialogTitle> : null}
          </DialogHeader>
          <div className="space-y-3">
            {selected ? (
              <p className="font-mono text-xs text-muted-foreground">{selected}</p>
            ) : null}
            {colCode ? (
              <Field
                id="email-template-body"
                label={colCode}
                {...chromeFieldDesc(chrome.data, "ADMIN_EMAIL_TEMPLATE_BODY_DESC")}
              >
                <Textarea
                  id="email-template-body"
                  value={body}
                  onChange={(e) => setBody(e.target.value)}
                  placeholder={colCode}
                />
              </Field>
            ) : null}
          </div>
          {saveLabel ? (
            <DialogFooter>
              <Button
                type="button"
                isLoading={patch.isPending}
                pendingLabel={pending || saveLabel}
                disabled={!body.trim()}
                onClick={() => {
                  if (!requireStepUp()) return;
                  void patch.mutate(
                    { code: selected, body: body.trim() },
                    {
                      onSuccess: () => closeDialog(),
                    },
                  );
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
