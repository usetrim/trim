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
 * Download a PDF of the tax-invoice using the exact same article DOM + print
 * stylesheet as Print (light paper, letterhead bands, spacing). Opens a
 * dedicated print window containing only the invoice so Save-as-PDF matches
 * the Print pipeline end-to-end (not the separate Go raster PDF).
 */
export function downloadReceiptArticlePdf(
  root: HTMLElement,
  opts: { title?: string; filename?: string },
): void {
  if (!root.hasAttribute("data-receipt-print")) {
    root.setAttribute("data-receipt-print", "");
  }

  const title = (opts.title || document.title || "Tax invoice").trim();
  const filename = (opts.filename || "invoice").trim().replace(/\.pdf$/i, "");
  const w = window.open("", "_blank", "noopener,noreferrer");
  if (!w) {
    // Popup blocked - fall back to same-tab print of the live article.
    printReceiptArticle(root, title);
    return;
  }

  const doc = w.document;
  const styles = Array.from(document.querySelectorAll('link[rel="stylesheet"], style'))
    .map((n) => n.outerHTML)
    .join("\n");
  const clone = root.cloneNode(true) as HTMLElement;

  doc.open();
  doc.write(`<!DOCTYPE html><html><head><meta charset="utf-8"/><title>${escapeHtml(
    filename || title,
  )}</title>${styles}
<style>
  @page { margin: 10mm; }
  html, body {
    margin: 0 !important;
    padding: 0 !important;
    background: #fff !important;
    color: #171717 !important;
  }
  body { padding: 0 !important; }
  [data-receipt-print] {
    position: static !important;
    width: 100% !important;
    max-width: 100% !important;
    margin: 0 !important;
    box-shadow: none !important;
    border: none !important;
    background: #fff !important;
    color: #171717 !important;
    --background: 0 0% 100%;
    --foreground: 240 10% 3.9%;
    --card: 0 0% 100%;
    --muted: 240 4.8% 95.9%;
    --muted-foreground: 240 3.8% 46.1%;
    --border: 240 5.9% 90%;
    --trim-fg: #171717;
    --trim-muted: #5c5c5c;
    --trim-subtle: #8a8a8a;
    --trim-border: #e8e8e8;
    --trim-panel: #ffffff;
    --trim-panel-2: #f4f4f5;
    --trim-status: #27a644;
    -webkit-print-color-adjust: exact;
    print-color-adjust: exact;
  }
  [data-receipt-print] header,
  [data-receipt-print] footer {
    background-color: #f4f4f5 !important;
  }
</style></head><body></body></html>`);
  doc.close();
  doc.body.appendChild(doc.importNode(clone, true));

  const trigger = () => {
    try {
      w.focus();
      w.print();
    } finally {
      window.setTimeout(() => {
        try {
          w.close();
        } catch {
          /* ignore */
        }
      }, 1_000);
    }
  };

  // Wait a tick for styles/images (wordmark) to paint before print.
  w.requestAnimationFrame(() => {
    window.setTimeout(trigger, 250);
  });
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}
