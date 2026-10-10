/**
 * Print / Save-as-PDF the live tax-invoice article only.
 * Single pipeline for both Print and Download buttons (no Go /pdf).
 */
export function printReceiptArticle(
  root: HTMLElement,
  documentTitle?: string,
  onAfterPrint?: () => void,
): void {
  const priorTitle = document.title;
  if (documentTitle?.trim()) {
    document.title = documentTitle.trim();
  }

  if (!root.hasAttribute("data-receipt-print")) {
    root.setAttribute("data-receipt-print", "");
  }

  let done = false;
  const cleanup = () => {
    if (done) return;
    done = true;
    document.title = priorTitle;
    window.removeEventListener("afterprint", cleanup);
    try {
      onAfterPrint?.();
    } catch {
      /* ignore */
    }
  };
  window.addEventListener("afterprint", cleanup);

  // Let the browser apply @media print against the settled live article
  // (same paint path for Print and Download — no clone window, no Go PDF).
  window.requestAnimationFrame(() => {
    window.requestAnimationFrame(() => {
      window.print();
      // Fallback if afterprint never fires.
      window.setTimeout(cleanup, 2_000);
    });
  });
}

/**
 * Download = Print. Identical live article + identical window.print() /
 * Save-as-PDF path as the Print button. Never fetches /api/.../pdf (Go).
 *
 * Uses the invoice document title for the print stylesheet context (same as
 * Print). Suggested Save-as-PDF filename is applied via document.title when
 * `filename` is provided — layout/CSS are unchanged.
 */
export function downloadReceiptArticlePdf(
  root: HTMLElement,
  opts: { title?: string; filename?: string; onAfterPrint?: () => void },
): void {
  const docTitle = (opts.title || "").trim();
  const filename = (opts.filename || "").trim().replace(/\.pdf$/i, "");
  // Prefer filename for the saved file name; fall back to the same title Print uses.
  const printTitle = (filename || docTitle || document.title || "Tax invoice").trim();
  printReceiptArticle(root, printTitle, opts.onAfterPrint);
}
