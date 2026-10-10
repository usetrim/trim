/**
 * Print / Save-as-PDF the live tax-invoice article only.
 * Uses the same DOM + CSS as the dashboard receipt page (no alternate layout).
 */
export function printReceiptArticle(root: HTMLElement, documentTitle?: string): void {
  const priorTitle = document.title;
  if (documentTitle?.trim()) {
    document.title = documentTitle.trim();
  }

  const cleanup = () => {
    document.title = priorTitle;
    window.removeEventListener("afterprint", cleanup);
  };
  window.addEventListener("afterprint", cleanup);

  // Ensure the article is the print root (CSS targets [data-receipt-print]).
  if (!root.hasAttribute("data-receipt-print")) {
    root.setAttribute("data-receipt-print", "");
  }

  window.print();

  // Fallback if afterprint never fires (some browsers).
  window.setTimeout(cleanup, 2_000);
}

/**
 * Download = Print. Same live article + same window.print() / Save-as-PDF
 * pipeline as the Print button. Never uses the separate Go first-party PDF
 * (that raster layout diverges: spacing, alignment, "Payment methodvisa").
 */
export function downloadReceiptArticlePdf(
  root: HTMLElement,
  opts: { title?: string; filename?: string },
): void {
  const filename = (opts.filename || "").trim().replace(/\.pdf$/i, "");
  const title = (filename || opts.title || document.title || "Tax invoice").trim();
  printReceiptArticle(root, title);
}
