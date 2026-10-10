"use client";

import { AdminFilterBar } from "@/components/admin/admin-filter-bar";
import { DualTableSkeleton } from "@/components/skeletons/page-skeletons";
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
import { Textarea } from "@/components/ui/textarea";
import { usePatchChromeMessage, usePatchLegalSection } from "@/hooks/mutations/chrome";
import { useAdminChromeMessages, useAdminLegal, useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminToken } from "@/hooks/use-admin-token";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { useDeferredDialogSelection } from "@/hooks/use-deferred-dialog-selection";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { useRequireStepUp } from "@/hooks/use-require-step-up";
import { chromeFieldDesc } from "@/lib/admin-field-desc";
import { joinChromeParts } from "@/lib/chrome-join";
import { useEffect, useMemo, useState } from "react";

type MsgRow = Record<string, unknown>;
type LegalRow = Record<string, unknown>;

export function ChromeClient({ initialToken }: { initialToken?: string }) {
  const token = useAdminToken(initialToken);
  const pageSize = useDefaultPageSize();
  const [skip, setSkip] = useState(0);
  const [legalSkip, setLegalSkip] = useState(0);
  const [qInput, setQInput] = useState("");
  const q = useDebouncedValue(qInput.trim(), 250);
  const [legalQInput, setLegalQInput] = useState("");
  const legalQ = useDebouncedValue(legalQInput.trim(), 250);
  const {
    open: msgOpen,
    selectedId: editCode,
    openWith: openMsg,
    close: closeMsg,
    onOpenChange: onMsgOpenChange,
    setSelectedId: setEditCode,
  } = useDeferredDialogSelection();
  const [editBody, setEditBody] = useState("");
  const {
    open: legalOpen,
    selectedId: legalId,
    openWith: openLegal,
    close: closeLegal,
    onOpenChange: onLegalOpenChange,
  } = useDeferredDialogSelection();
  const [legalBody, setLegalBody] = useState("");
  const [legalPublished, setLegalPublished] = useState(false);
  /** Scope spinner to the clicked legal footer action (save | publish | unpublish). */
  const [legalFooterAction, setLegalFooterAction] = useState<
    null | "save" | "publish" | "unpublish"
  >(null);
  const chrome = useAdminNavChrome(token);
  const messages = useAdminChromeMessages(token, skip, pageSize, q);
  const legal = useAdminLegal(token, legalSkip, pageSize, legalQ);
  const patch = usePatchChromeMessage(token);
  const patchLegal = usePatchLegalSection(token);
  const { err: stepErr, requireStepUp } = useRequireStepUp(token);
  const title = chrome.data?.ADMIN_NAV_CHROME?.trim() || "";
  const pending =
    chrome.data?.ADMIN_PENDING_UPDATING?.trim() || chrome.data?.ADMIN_PENDING_SAVING?.trim() || "";
  const saveLabel = chrome.data?.ADMIN_ACTION_SAVE?.trim() || "";
  const viewLabel = chrome.data?.ADMIN_ACTION_VIEW?.trim() || "";
  const rowActionsLabel = chrome.data?.ADMIN_TABLE_ROW_ACTIONS?.trim() || "";
  const filterSearch = chrome.data?.ADMIN_FILTER_SEARCH?.trim() || "";
  const filterSearchDesc = chrome.data?.ADMIN_FILTER_SEARCH_DESC?.trim() || "";
  const closeLabel =
    chrome.data?.ADMIN_DIALOG_CLOSE?.trim() || chrome.data?.ADMIN_CLOSE?.trim() || "";
  const publishLabel = chrome.data?.ADMIN_LEGAL_PUBLISH?.trim() || "";
  const unpublishLabel = chrome.data?.ADMIN_LEGAL_UNPUBLISH?.trim() || "";
  const colCode = chrome.data?.ADMIN_CHROME_COL_CODE?.trim() || "";
  const colBody = chrome.data?.ADMIN_CHROME_COL_BODY?.trim() || "";
  const colLegalId = chrome.data?.ADMIN_LEGAL_COL_ID?.trim() || "";
  const colLegalStatus = chrome.data?.ADMIN_LEGAL_COL_STATUS?.trim() || "";
  const sepDot = chrome.data?.ADMIN_UI_SEP_DOT?.trim() || "";
  const editSectionLabel = chrome.data?.ADMIN_SECTION_EDIT?.trim() || "";

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setSkip(0);
  }, [q]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setLegalSkip(0);
  }, [legalQ]);

  // Body/published are seeded when the operator opens a row - do not rehydrate from
  // list refetch while the dialog is open (that wipes in-progress edits).

  const msgColumns = useMemo<ColumnDef<MsgRow>[]>(() => {
    const cols: ColumnDef<MsgRow>[] = [];
    if (colCode) {
      cols.push({
        id: "code",
        header: colCode,
        cell: ({ row }) => (
          <button
            type="button"
            className="font-mono text-xs text-foreground hover:underline"
            onClick={() => {
              openMsg(String(row.original.code || ""));
              setEditBody(String(row.original.body || ""));
            }}
          >
            {String(row.original.code || "")}
          </button>
        ),
      });
    }
    if (colBody) {
      cols.push({
        id: "body",
        header: colBody,
        cell: ({ row }) => (
          <span className="line-clamp-2 text-muted-foreground">
            {String(row.original.body || "")}
          </span>
        ),
      });
    }
    if (viewLabel && rowActionsLabel) {
      cols.push({
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const code = String(row.original.code || "").trim();
          if (!code) return null;
          return (
            <DataTableRowActions
              triggerLabel={rowActionsLabel}
              actions={[
                {
                  id: "edit",
                  label: viewLabel,
                  onSelect: () => {
                    openMsg(code);
                    setEditBody(String(row.original.body || ""));
                  },
                },
              ]}
            />
          );
        },
      });
    }
    return cols;
  }, [colCode, colBody, viewLabel, rowActionsLabel, openMsg]);

  const legalColumns = useMemo<ColumnDef<LegalRow>[]>(
    () => [
      {
        id: "section",
        header: colLegalId,
        cell: ({ row }) => {
          const id = String(row.original.id || "");
          const line = joinChromeParts([row.original.page, row.original.heading], sepDot);
          if (!id || !line) return "";
          return (
            <button
              type="button"
              className="text-left text-foreground hover:underline"
              onClick={() => {
                openLegal(id);
                setLegalBody(String(row.original.body || ""));
                setLegalPublished(Boolean(String(row.original.published_at || "").trim()));
              }}
            >
              {line}
            </button>
          );
        },
      },
      {
        id: "status",
        header: colLegalStatus,
        cell: ({ row }) => String(row.original.publish_status_label || ""),
      },
      {
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const id = String(row.original.id || "");
          if (!id || !viewLabel || !rowActionsLabel) return null;
          return (
            <DataTableRowActions
              triggerLabel={rowActionsLabel}
              actions={[
                {
                  id: "view",
                  label: viewLabel,
                  onSelect: () => {
                    openLegal(id);
                    setLegalBody(String(row.original.body || ""));
                    setLegalPublished(Boolean(String(row.original.published_at || "").trim()));
                  },
                },
              ]}
            />
          );
        },
      },
    ],
    [colLegalId, colLegalStatus, sepDot, viewLabel, rowActionsLabel, openLegal],
  );

  const bootReady = useListBootReady(pageSize, Boolean(chrome.data), queryContentReady(messages));
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    return (
      <DualTableSkeleton
        title={title || undefined}
        primaryColumns={[colCode, colBody].map((label) => ({ label: label || "" }))}
        secondaryColumns={[{ label: "" }, { label: "" }]}
        rows={pageSize > 0 ? pageSize : 8}
        showFilters
        filterCount={1}
      />
    );
  }

  return (
    <div className="space-y-6">
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      {stepErr ? <p className="text-sm text-destructive">{stepErr}</p> : null}
      <AdminFilterBar
        filters={[
          {
            kind: "search",
            id: "chrome-search",
            label: filterSearch,
            description: filterSearchDesc,
            value: qInput,
            onChange: setQInput,
          },
        ]}
      />
      {msgColumns.length > 0 ? (
        <Card>
          <CardContent className="pt-6">
            <DataTable
              columns={msgColumns}
              data={((messages.data?.items as MsgRow[]) ?? []).filter((item) =>
                Boolean(String(item.code || "")),
              )}
              meta={messages.data?.meta}
              onPage={setSkip}
              pageDisabled={messages.isFetching}
              isFetching={(messages.isFetching && !messages.isPending) || patch.isPending}
              getRowId={(row) => String(row.code || "")}
              bordered={false}
            />
          </CardContent>
        </Card>
      ) : null}
      <Dialog
        open={msgOpen}
        onOpenChange={(open) => {
          onMsgOpenChange(open);
          if (!open) setEditBody("");
        }}
      >
        <DialogContent closeLabel={closeLabel} className="max-w-3xl">
          <DialogHeader>
            {editSectionLabel ? <DialogTitle>{editSectionLabel}</DialogTitle> : null}
          </DialogHeader>
          <div className="space-y-3">
            {colCode ? (
              <Field
                id="chrome-edit-code"
                label={colCode}
                {...chromeFieldDesc(chrome.data, "ADMIN_CHROME_CODE_DESC")}
              >
                <Textarea
                  id="chrome-edit-code"
                  value={editCode}
                  onChange={(e) => setEditCode(e.target.value)}
                  placeholder={colCode}
                />
              </Field>
            ) : null}
            {colBody ? (
              <Field
                id="chrome-edit-body"
                label={colBody}
                {...chromeFieldDesc(chrome.data, "ADMIN_CHROME_BODY_DESC")}
              >
                <Textarea
                  id="chrome-edit-body"
                  value={editBody}
                  onChange={(e) => setEditBody(e.target.value)}
                  placeholder={colBody}
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
                disabled={!editCode.trim() || !editBody.trim()}
                onClick={() => {
                  if (!requireStepUp()) return;
                  void patch.mutate(
                    {
                      code: editCode.trim(),
                      body: editBody.trim(),
                    },
                    {
                      onSuccess: () => closeMsg(),
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
      {(legal.data?.items?.length ?? 0) > 0 || (legal.data?.meta?.total ?? 0) > 0 || legalQ ? (
        <>
          <AdminFilterBar
            filters={[
              {
                kind: "search",
                id: "chrome-legal-search",
                label: filterSearch,
                description: filterSearchDesc,
                value: legalQInput,
                onChange: setLegalQInput,
              },
            ]}
          />
          <Card>
            <CardContent className="pt-6">
              <DataTable
                columns={legalColumns}
                data={((legal.data?.items as LegalRow[]) ?? []).filter(
                  (item) =>
                    Boolean(String(item.id || "")) &&
                    Boolean(joinChromeParts([item.page, item.heading], sepDot)),
                )}
                meta={legal.data?.meta}
                onPage={setLegalSkip}
                pageDisabled={legal.isFetching}
                isFetching={(legal.isFetching && !legal.isPending) || patchLegal.isPending}
                getRowId={(row) => String(row.id)}
                bordered={false}
              />
            </CardContent>
          </Card>
        </>
      ) : null}
      <Dialog
        open={legalOpen}
        onOpenChange={(open) => {
          onLegalOpenChange(open);
          if (!open) {
            setLegalBody("");
            setLegalPublished(false);
            setLegalFooterAction(null);
          }
        }}
      >
        <DialogContent closeLabel={closeLabel} className="max-w-3xl">
          <DialogHeader>
            {editSectionLabel ? <DialogTitle>{editSectionLabel}</DialogTitle> : null}
          </DialogHeader>
          <div className="space-y-3">
            {colBody ? (
              <Field
                id="chrome-legal-body"
                label={colBody}
                {...chromeFieldDesc(chrome.data, "ADMIN_LEGAL_BODY_DESC")}
              >
                <Textarea
                  id="chrome-legal-body"
                  value={legalBody}
                  onChange={(e) => setLegalBody(e.target.value)}
                  placeholder={colBody}
                />
              </Field>
            ) : null}
          </div>
          {saveLabel || (publishLabel && !legalPublished) || (unpublishLabel && legalPublished) ? (
            <DialogFooter>
              {saveLabel ? (
                <Button
                  type="button"
                  disabled={patchLegal.isPending && legalFooterAction !== "save"}
                  isLoading={patchLegal.isPending && legalFooterAction === "save"}
                  pendingLabel={pending || saveLabel}
                  onClick={() => {
                    if (!requireStepUp()) return;
                    setLegalFooterAction("save");
                    void patchLegal.mutate(
                      {
                        id: legalId,
                        body: { body: legalBody },
                      },
                      {
                        onSettled: () => setLegalFooterAction(null),
                        onSuccess: () => closeLegal(),
                      },
                    );
                  }}
                >
                  {saveLabel}
                </Button>
              ) : null}
              {publishLabel && !legalPublished ? (
                <Button
                  type="button"
                  variant="secondary"
                  disabled={patchLegal.isPending && legalFooterAction !== "publish"}
                  isLoading={patchLegal.isPending && legalFooterAction === "publish"}
                  pendingLabel={pending || publishLabel}
                  onClick={() => {
                    if (!requireStepUp()) return;
                    setLegalFooterAction("publish");
                    void patchLegal.mutate(
                      {
                        id: legalId,
                        body: { publish: true },
                      },
                      {
                        onSettled: () => setLegalFooterAction(null),
                        onSuccess: () => closeLegal(),
                      },
                    );
                  }}
                >
                  {publishLabel}
                </Button>
              ) : null}
              {unpublishLabel && legalPublished ? (
                <Button
                  type="button"
                  variant="secondary"
                  disabled={patchLegal.isPending && legalFooterAction !== "unpublish"}
                  isLoading={patchLegal.isPending && legalFooterAction === "unpublish"}
                  pendingLabel={pending || unpublishLabel}
                  onClick={() => {
                    if (!requireStepUp()) return;
                    setLegalFooterAction("unpublish");
                    void patchLegal.mutate(
                      {
                        id: legalId,
                        body: { publish: false },
                      },
                      {
                        onSettled: () => setLegalFooterAction(null),
                        onSuccess: () => closeLegal(),
                      },
                    );
                  }}
                >
                  {unpublishLabel}
                </Button>
              ) : null}
            </DialogFooter>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}
