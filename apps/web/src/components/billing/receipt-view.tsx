"use client";

import { TrimWordmark } from "@/components/brand/trim-wordmark";
import { ReceiptDetailSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { FetchProgressBar } from "@/components/ui/fetch-progress";
import { useAuthProviders } from "@/hooks/queries/auth";
import { useReceipt } from "@/hooks/queries/billing";
import { downloadReceiptArticlePdf, printReceiptArticle } from "@/lib/receipt-print";
import { receiptSkeletonChrome } from "@/lib/skeleton-chrome";
import { formatMoney } from "@/lib/utils";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";

type Detail = {
  id: string;
  paddle_transaction_id: string;
  paddle_invoice_number: string | null;
  paddle_invoice_pdf_url: string | null;
  display_id?: string;
  money_locale?: string;
  back_href?: string;
  status: string;
  status_label?: string;
  currency_code: string;
  subtotal_cents: number;
  tax_cents: number;
  total_cents: number;
  tax_rate_bps: number;
  tax_rate_percent?: string;
  bill_to_name: string | null;
  bill_to_email: string;
  bill_to_company: string | null;
  bill_to_address_line1: string | null;
  bill_to_address_line2: string | null;
  bill_to_country: string | null;
  tax_id: string | null;
  period_start: string | null;
  period_end: string | null;
  paid_at: string | null;
  created_at: string;
  company_legal_name: string;
  company_support_email: string;
  company_logo_url?: string;
  company_address_line1?: string;
  company_address_line2?: string;
  company_country?: string;
  company_locality_line?: string;
  company_vat_id?: string;
  company_vat_prefix?: string;
  company_registration?: string;
  print_action_label?: string;
  print_pending_label?: string;
  print_pending_ms?: string;
  document_title?: string;
  period_label?: string;
  paid_at_label?: string;
  download_pdf_action_label?: string;
  download_pdf_pending_label?: string;
  download_pdf_failed_message?: string;
  download_pdf_done_message?: string;
  download_pdf_filename_fmt?: string;
  first_party_pdf_href?: string;
  back_action_label?: string;
  seller_missing_message?: string;
  section_bill_to?: string;
  section_invoice_from?: string;
  section_invoice_details?: string;
  section_transaction?: string;
  section_tax_breakdown?: string;
  section_period?: string;
  section_payment?: string;
  bill_to_locality_line?: string;
  col_description?: string;
  col_product?: string;
  col_qty?: string;
  col_unit?: string;
  col_tax_rate?: string;
  col_amount?: string;
  label_subtotal?: string;
  label_tax?: string;
  label_tax_id?: string;
  label_total?: string;
  label_amount_paid?: string;
  label_invoice_reference?: string;
  label_transaction_id?: string;
  label_currency?: string;
  label_tax_percent?: string;
  label_tax_total?: string;
  merchant_via?: string;
  header_meta_sep?: string;
  not_found_message?: string;
  footer?: string;
  line_items: Array<{
    position: number;
    description: string;
    quantity: number;
    unit_amount_cents: number;
    amount_cents: number;
    price_name?: string | null;
    period_label?: string | null;
  }>;
  payment_method_summary?: string | null;
};

function countryDisplayName(code: string | null | undefined, locale: string): string {
  const raw = (code || "").trim();
  if (!raw) return "";
  if (raw.length !== 2) return raw;
  try {
    const name = new Intl.DisplayNames([locale || "en"], { type: "region" }).of(raw.toUpperCase());
    return (name || raw).trim();
  } catch {
    return raw.toUpperCase();
  }
}

function InlineDetail({ label, value }: { label?: string; value?: string }) {
  const l = (label || "").trim();
  const v = (value || "").trim();
  if (!l || !v) return null;
  return (
    <p className="text-[13px] leading-relaxed text-foreground print:text-zinc-900">
      <span className="font-semibold text-foreground">{l}:</span> {v}
    </p>
  );
}

export function ReceiptView({
  receiptId,
  accessToken,
}: {
  receiptId: string;
  accessToken?: string;
}) {
  const [printing, setPrinting] = useState(false);
  const invoiceRef = useRef<HTMLElement | null>(null);
  const autoPrintOnce = useRef(false);
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { data, isPending, isFetching, error } = useReceipt(accessToken, receiptId);
  const authProviders = useAuthProviders();
  const publicChrome = receiptSkeletonChrome(authProviders.data?.site);
  const receipt = data as Detail | undefined;

  // List "Download" → ?download=1 opens this page and runs the Print pipeline.
  useEffect(() => {
    if (autoPrintOnce.current) return;
    if (!receipt || isPending) return;
    if (!(receipt.company_legal_name || "").trim()) return;
    const want = searchParams.get("download") === "1" || searchParams.get("print") === "1";
    if (!want) return;

    const run = () => {
      const el = invoiceRef.current;
      if (!el) return false;
      autoPrintOnce.current = true;
      const displayId = (receipt.display_id || receipt.id || "invoice").trim();
      const fmt = (receipt.download_pdf_filename_fmt || "").trim();
      const filename = fmt.includes("%s")
        ? fmt.replace("%s", displayId)
        : fmt || `trim-invoice-${displayId}`;
      downloadReceiptArticlePdf(el, {
        title: receipt.document_title || undefined,
        filename,
      });
      const done = receipt.download_pdf_done_message || "";
      if (done) toast.success(done);
      router.replace(pathname, { scroll: false });
      return true;
    };

    if (run()) return;
    // Article mounts after paint; retry once.
    const t = window.setTimeout(() => {
      run();
    }, 100);
    return () => window.clearTimeout(t);
  }, [receipt, isPending, searchParams, router, pathname]);

  if (!accessToken || (isPending && !receipt)) {
    return (
      <ReceiptDetailSkeleton
        lineRows={receipt?.line_items?.length ?? 1}
        chrome={
          receipt
            ? {
                col_description: receipt.col_description,
                col_product: receipt.col_product,
                col_qty: receipt.col_qty,
                col_unit: receipt.col_unit,
                col_tax_rate: receipt.col_tax_rate,
                col_amount: receipt.col_amount,
                section_bill_to: receipt.section_bill_to,
                section_invoice_from: receipt.section_invoice_from,
                section_invoice_details: receipt.section_invoice_details,
                section_transaction: receipt.section_transaction,
                section_tax_breakdown: receipt.section_tax_breakdown,
                section_period: receipt.section_period,
                section_payment: receipt.section_payment,
                label_subtotal: receipt.label_subtotal,
                label_tax: receipt.label_tax,
                label_total: receipt.label_total,
                label_amount_paid: receipt.label_amount_paid,
                footer: receipt.footer,
              }
            : publicChrome
        }
      />
    );
  }
  if (error || !receipt) {
    const err = error as
      | (Error & {
          body?: {
            not_found_message?: string;
            back_action_label?: string;
            back_href?: string;
          };
        })
      | null;
    return (
      <div className="p-8">
        <p className="text-sm text-destructive">
          {err?.body?.not_found_message || err?.message || ""}
        </p>
        <Button asChild variant="outline" className="mt-4">
          <Link href={err?.body?.back_href || ""}>{err?.body?.back_action_label || ""}</Link>
        </Button>
      </div>
    );
  }

  const brand = receipt.company_legal_name?.trim() || "";
  const documentNumber = receipt.display_id?.trim() || "";
  const moneyLocale = receipt.money_locale?.trim() || "";
  const paidLabel = (receipt.paid_at_label || "").trim();
  const periodLabel = (receipt.period_label || "").trim();
  const totalFmt = moneyLocale
    ? formatMoney(receipt.total_cents, receipt.currency_code, moneyLocale)
    : "";
  const metaSep = (receipt.header_meta_sep || " - ").trim() || " - ";
  const productCol = (receipt.col_product || receipt.col_description || "").trim();
  const taxPercent = (receipt.tax_rate_percent || "").trim();
  const billToCountry = countryDisplayName(receipt.bill_to_country, moneyLocale);
  const backHref = (receipt.back_href || "").trim();

  if (!brand) {
    return (
      <div className="p-8">
        <p className="text-sm text-destructive">{receipt.seller_missing_message || ""}</p>
        <Button asChild variant="outline" className="mt-4">
          <Link href={backHref}>{receipt.back_action_label || ""}</Link>
        </Button>
      </div>
    );
  }

  if (!moneyLocale) {
    return (
      <div className="p-8">
        <p className="text-sm text-destructive">
          {process.env.NEXT_PUBLIC_MONEY_LOCALE_MISSING?.trim() || ""}
        </p>
        <Button asChild variant="outline" className="mt-4">
          <Link href={backHref}>{receipt.back_action_label || ""}</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="mx-auto w-full max-w-[920px] px-3 py-6 sm:px-6 sm:py-10 print:max-w-none print:px-0 print:py-0">
      <FetchProgressBar active={isFetching && !isPending} className="mb-4 print:hidden" />
      <div className="mb-5 flex flex-wrap items-center justify-between gap-3 print:hidden">
        <Button asChild variant="outline" size="sm">
          <Link href={backHref}>{receipt.back_action_label}</Link>
        </Button>
        <div className="flex flex-wrap gap-2">
          {receipt.download_pdf_action_label ? (
            <Button
              variant="outline"
              size="sm"
              isLoading={printing}
              pendingLabel={receipt.download_pdf_pending_label || undefined}
              onClick={() => {
                const el = invoiceRef.current;
                const failed = receipt.download_pdf_failed_message || "";
                if (!el) {
                  toast.error(failed);
                  return;
                }
                // Download = Print (same live article → Save as PDF). No Go PDF.
                const displayId = (receipt.display_id || receipt.id || "invoice").trim();
                const fmt = (receipt.download_pdf_filename_fmt || "").trim();
                const filename = fmt.includes("%s")
                  ? fmt.replace("%s", displayId)
                  : fmt || `trim-invoice-${displayId}`;
                setPrinting(true);
                downloadReceiptArticlePdf(el, {
                  title: receipt.document_title || undefined,
                  filename,
                });
                const done = receipt.download_pdf_done_message || "";
                if (done) toast.success(done);
                const raw = receipt.print_pending_ms?.trim() || "";
                const ms = Number.parseInt(raw, 10);
                window.setTimeout(
                  () => setPrinting(false),
                  Number.isFinite(ms) && ms > 0 ? ms : 800,
                );
              }}
            >
              {receipt.download_pdf_action_label}
            </Button>
          ) : null}
          <Button
            size="sm"
            isLoading={printing}
            pendingLabel={receipt.print_pending_label || undefined}
            disabled={!receipt.print_action_label}
            onClick={() => {
              const el = invoiceRef.current;
              if (!el) return;
              setPrinting(true);
              printReceiptArticle(el, receipt.document_title || undefined);
              const raw = receipt.print_pending_ms?.trim() || "";
              const ms = Number.parseInt(raw, 10);
              if (Number.isFinite(ms) && ms > 0) {
                window.setTimeout(() => setPrinting(false), ms);
              } else {
                setPrinting(false);
              }
            }}
          >
            {receipt.print_action_label}
          </Button>
        </div>
      </div>

      <article
        ref={invoiceRef}
        data-receipt-print
        className="bg-card text-foreground print:bg-white print:text-zinc-900"
      >
        <header
          data-receipt-band
          className="bg-muted px-5 py-6 sm:px-8 sm:py-7 print:bg-zinc-100"
          style={{ backgroundColor: "var(--receipt-band, #f4f4f5)" }}
        >
          <div className="flex flex-col gap-5 sm:flex-row sm:items-start sm:justify-between">
            <div className="min-w-0 space-y-1.5">
              <div className="flex flex-wrap items-center gap-2.5">
                <h1 className="text-[1.35rem] font-bold tracking-tight text-foreground sm:text-[1.5rem] print:text-zinc-900">
                  {receipt.document_title}
                </h1>
                {receipt.status_label ? (
                  <span className="inline-flex items-center rounded bg-[var(--trim-status,#27a644)] px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide text-white print:bg-[#27a644] print:text-white">
                    {receipt.status_label}
                  </span>
                ) : null}
              </div>
              {paidLabel || totalFmt ? (
                <p className="text-[13px] text-muted-foreground print:text-zinc-500">
                  {paidLabel}
                  {paidLabel && totalFmt ? metaSep : null}
                  {totalFmt ? (
                    <span className="font-semibold text-foreground print:text-zinc-800">
                      {totalFmt}
                    </span>
                  ) : null}
                </p>
              ) : null}
            </div>
            <div className="flex shrink-0 flex-col items-start gap-1 sm:items-end">
              {receipt.company_logo_url ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={receipt.company_logo_url}
                  alt={brand}
                  className="h-8 w-auto max-w-[140px] object-contain sm:h-9 print:brightness-0"
                />
              ) : (
                <>
                  <span className="print:hidden">
                    <TrimWordmark size="lg" alt={brand} priority />
                  </span>
                  <span className="hidden print:inline-flex">
                    <TrimWordmark size="lg" alt={brand} ink="black" />
                  </span>
                </>
              )}
              <p className="text-[15px] font-bold tracking-tight text-foreground print:text-zinc-900">
                {brand}
              </p>
              {receipt.merchant_via ? (
                <p className="text-[11px] text-muted-foreground print:text-zinc-500">
                  {receipt.merchant_via}
                </p>
              ) : null}
            </div>
          </div>
        </header>

        <section className="grid gap-10 px-5 py-8 sm:px-8 md:grid-cols-2 print:grid-cols-2">
          <div className="min-w-0 space-y-2">
            <h2 className="text-[13px] font-bold text-foreground print:text-zinc-900">
              {receipt.section_bill_to}
            </h2>
            <div className="space-y-0.5 text-[13px] leading-relaxed text-foreground/90 print:text-zinc-800">
              {receipt.bill_to_name ? <p>{receipt.bill_to_name}</p> : null}
              {receipt.bill_to_company ? <p>{receipt.bill_to_company}</p> : null}
              <p className="break-all">{receipt.bill_to_email}</p>
              {receipt.bill_to_address_line1 ? <p>{receipt.bill_to_address_line1}</p> : null}
              {receipt.bill_to_address_line2 ? <p>{receipt.bill_to_address_line2}</p> : null}
              {receipt.bill_to_locality_line ? <p>{receipt.bill_to_locality_line}</p> : null}
              {billToCountry ? <p>{billToCountry}</p> : null}
              {receipt.tax_id && receipt.label_tax_id ? (
                <p className="pt-1">
                  <span className="font-semibold">{receipt.label_tax_id}:</span> {receipt.tax_id}
                </p>
              ) : null}
            </div>
            {receipt.payment_method_summary ? (
              <p className="pt-2 text-[13px] text-foreground/90 print:text-zinc-800">
                <span className="font-semibold">{receipt.section_payment}:</span>{" "}
                {receipt.payment_method_summary}
              </p>
            ) : null}
          </div>

          <div className="min-w-0 space-y-2">
            <h2 className="text-[13px] font-bold text-foreground print:text-zinc-900">
              {receipt.section_invoice_from}
            </h2>
            <div className="space-y-0.5 text-[13px] leading-relaxed text-foreground/90 print:text-zinc-800">
              <p>{brand}</p>
              {receipt.company_address_line1 ? <p>{receipt.company_address_line1}</p> : null}
              {receipt.company_address_line2 ? <p>{receipt.company_address_line2}</p> : null}
              {receipt.company_locality_line ? <p>{receipt.company_locality_line}</p> : null}
              {receipt.company_country ? <p>{receipt.company_country}</p> : null}
              {receipt.company_vat_id ? (
                <p className="pt-1">
                  <span className="font-semibold">
                    {receipt.company_vat_prefix || ""}
                    {receipt.company_vat_prefix ? ":" : ""}
                  </span>{" "}
                  {receipt.company_vat_id}
                </p>
              ) : null}
              {receipt.company_registration ? <p>{receipt.company_registration}</p> : null}
            </div>
          </div>
        </section>

        <section className="px-5 pb-6 sm:px-8">
          <h2 className="text-[13px] font-bold text-foreground print:text-zinc-900">
            {receipt.section_invoice_details}
          </h2>
          <div className="mt-3 space-y-1">
            <InlineDetail label={receipt.label_invoice_reference} value={documentNumber} />
            <InlineDetail label={receipt.section_period} value={periodLabel} />
            <InlineDetail
              label={receipt.label_transaction_id}
              value={receipt.paddle_transaction_id}
            />
            <InlineDetail label={receipt.label_currency} value={receipt.currency_code} />
          </div>
        </section>

        <div className="mx-5 border-t border-border sm:mx-8 print:border-zinc-200" />

        <section className="px-5 py-6 sm:px-8">
          <h2 className="mb-4 text-[13px] font-bold text-foreground print:text-zinc-900">
            {receipt.section_transaction}
          </h2>

          <div className="overflow-x-auto">
            <table className="w-full min-w-[520px] border-collapse text-left text-[13px] print:min-w-0">
              <thead>
                <tr className="border-b border-border print:border-zinc-200">
                  <th className="pb-2.5 pr-3 text-[12px] font-bold text-foreground print:text-zinc-900">
                    {productCol}
                  </th>
                  <th className="px-2 pb-2.5 text-right text-[12px] font-bold text-foreground print:text-zinc-900">
                    {receipt.col_qty}
                  </th>
                  <th className="px-2 pb-2.5 text-right text-[12px] font-bold text-foreground print:text-zinc-900">
                    {receipt.col_unit}
                  </th>
                  <th className="px-2 pb-2.5 text-right text-[12px] font-bold text-foreground print:text-zinc-900">
                    {receipt.col_tax_rate}
                  </th>
                  <th className="pb-2.5 pl-2 text-right text-[12px] font-bold text-foreground print:text-zinc-900">
                    {receipt.col_amount}
                  </th>
                </tr>
              </thead>
              <tbody>
                {receipt.line_items.map((item) => {
                  const linePeriod = (item.period_label || periodLabel || "").trim();
                  const linePriceName = (item.price_name || "").trim();
                  return (
                    <tr
                      key={item.position}
                      className="border-b border-border print:border-zinc-200"
                    >
                      <td className="py-3.5 pr-3 align-top">
                        <p className="font-bold text-foreground print:text-zinc-900">
                          {item.description}
                        </p>
                        {linePeriod ? (
                          <p className="mt-0.5 text-[12px] text-muted-foreground print:text-zinc-500">
                            {linePeriod}
                          </p>
                        ) : null}
                        {linePriceName ? (
                          <p className="mt-0.5 text-[12px] text-sky-600 dark:text-sky-400 print:text-sky-700">
                            {linePriceName}
                          </p>
                        ) : null}
                      </td>
                      <td className="px-2 py-3.5 text-right align-top tabular-nums text-muted-foreground print:text-zinc-700">
                        {item.quantity}
                      </td>
                      <td className="px-2 py-3.5 text-right align-top tabular-nums text-muted-foreground print:text-zinc-700">
                        {formatMoney(item.unit_amount_cents, receipt.currency_code, moneyLocale)}
                      </td>
                      <td className="px-2 py-3.5 text-right align-top tabular-nums text-muted-foreground print:text-zinc-700">
                        {taxPercent}
                      </td>
                      <td className="py-3.5 pl-2 text-right align-top font-bold tabular-nums text-foreground print:text-zinc-900">
                        {formatMoney(item.amount_cents, receipt.currency_code, moneyLocale)}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>

          <div className="mt-2 flex justify-end">
            <dl className="w-full max-w-[240px] text-[13px]">
              <div className="flex items-center justify-between gap-8 border-b border-border py-2 text-muted-foreground print:border-zinc-200 print:text-zinc-700">
                <dt>{receipt.label_subtotal}</dt>
                <dd className="tabular-nums">
                  {formatMoney(receipt.subtotal_cents, receipt.currency_code, moneyLocale)}
                </dd>
              </div>
              <div className="flex items-center justify-between gap-8 border-b border-border py-2 text-muted-foreground print:border-zinc-200 print:text-zinc-700">
                <dt>{receipt.label_tax}</dt>
                <dd className="tabular-nums">
                  {formatMoney(receipt.tax_cents, receipt.currency_code, moneyLocale)}
                </dd>
              </div>
              <div className="flex items-center justify-between gap-8 border-b border-border py-2 text-foreground print:border-zinc-200 print:text-zinc-900">
                <dt>{receipt.label_total}</dt>
                <dd className="tabular-nums">{totalFmt}</dd>
              </div>
              <div className="flex items-center justify-between gap-8 border-t-2 border-foreground/25 py-2.5 text-[15px] font-bold text-foreground print:border-zinc-800 print:text-zinc-900">
                <dt>{receipt.label_amount_paid}</dt>
                <dd className="tabular-nums">{totalFmt}</dd>
              </div>
            </dl>
          </div>

          <div className="mt-8 max-w-[240px]">
            <h2 className="text-[13px] font-bold text-foreground print:text-zinc-900">
              {receipt.section_tax_breakdown}
            </h2>
            <table className="mt-2 w-full border-collapse text-[13px]">
              <thead>
                <tr className="border-b border-border print:border-zinc-200">
                  <th className="pb-2 text-left text-[12px] font-bold text-foreground print:text-zinc-900">
                    {receipt.label_tax_percent}
                  </th>
                  <th className="pb-2 text-right text-[12px] font-bold text-foreground print:text-zinc-900">
                    {receipt.label_tax}
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr className="border-b border-border print:border-zinc-200">
                  <td className="py-2 tabular-nums text-foreground/90 print:text-zinc-800">
                    {taxPercent}
                  </td>
                  <td className="py-2 text-right tabular-nums text-foreground/90 print:text-zinc-800">
                    {formatMoney(receipt.tax_cents, receipt.currency_code, moneyLocale)}
                  </td>
                </tr>
                <tr>
                  <td className="pt-2.5 font-bold text-foreground print:text-zinc-900">
                    {receipt.label_tax_total}
                  </td>
                  <td className="pt-2.5 text-right font-bold tabular-nums text-foreground print:text-zinc-900">
                    {formatMoney(receipt.tax_cents, receipt.currency_code, moneyLocale)}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        {receipt.footer ? (
          <footer
            data-receipt-band
            className="mt-4 bg-muted px-5 py-7 text-center sm:px-8 print:bg-zinc-100"
            style={{ backgroundColor: "var(--receipt-band, #f4f4f5)" }}
          >
            <div className="mx-auto flex max-w-lg flex-col items-center gap-3">
              <p className="text-[12px] leading-relaxed text-muted-foreground print:text-zinc-600">
                {receipt.footer}
              </p>
              <span className="print:hidden opacity-70">
                <TrimWordmark size="sm" alt={brand} />
              </span>
              <span className="hidden print:inline-flex opacity-70">
                <TrimWordmark size="sm" alt={brand} ink="black" />
              </span>
              <p className="text-[11px] leading-relaxed text-muted-foreground print:text-zinc-500">
                {[
                  brand,
                  receipt.company_address_line1,
                  receipt.company_locality_line,
                  receipt.company_country,
                ]
                  .map((p) => (p || "").trim())
                  .filter(Boolean)
                  .join(", ")}
              </p>
            </div>
          </footer>
        ) : null}
      </article>
    </div>
  );
}
