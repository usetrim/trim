/**
 * Print / Save-as-PDF the live tax-invoice article only.
 * Single pipeline for both Print and Download buttons (no Go /pdf).
 *
 * Dark mode: temporarily force light theme on <html> before window.print().
 * Otherwise Tailwind dark tokens / near-white text paint on white PDF paper
 * and the invoice looks blank or washed out.
 *
 * Do not run onAfterPrint while the system print dialog is still open
 * (short timeouts mutate DOM mid-dialog and diverge Download vs Print PDFs).
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

  const html = document.documentElement;
  const hadDark = html.classList.contains("dark");
  const priorScheme = html.style.colorScheme;
  const priorTrimTheme = html.getAttribute("data-trim-theme");

  // Force light surface tokens + light wordmarks for the print snapshot.
  if (hadDark) {
    html.classList.remove("dark");
  }
  html.style.colorScheme = "light";
  html.setAttribute("data-trim-theme", "light");
  html.setAttribute("data-trim-print-light", "1");

  let done = false;
  let fallbackTimer = 0;
  const cleanup = () => {
    if (done) return;
    done = true;
    if (fallbackTimer) {
      window.clearTimeout(fallbackTimer);
      fallbackTimer = 0;
    }
    document.title = priorTitle;
    window.removeEventListener("afterprint", cleanup);

    html.removeAttribute("data-trim-print-light");
    if (hadDark) {
      html.classList.add("dark");
    }
    html.style.colorScheme = priorScheme;
    if (priorTrimTheme) {
      html.setAttribute("data-trim-theme", priorTrimTheme);
    } else {
      html.removeAttribute("data-trim-theme");
    }

    try {
      onAfterPrint?.();
    } catch {
      /* ignore */
    }
  };
  window.addEventListener("afterprint", cleanup);

  // Double rAF: let the light-theme repaint settle before the print snapshot.
  window.requestAnimationFrame(() => {
    window.requestAnimationFrame(() => {
      window.print();
      // Ultimate fallback only (browsers that never emit afterprint).
      fallbackTimer = window.setTimeout(cleanup, 120_000);
    });
  });
}

/**
 * Download = Print. Identical live article + identical window.print() /
 * Save-as-PDF path as the Print button. Never fetches /api/.../pdf (Go).
 */
export function downloadReceiptArticlePdf(
  root: HTMLElement,
  opts: { title?: string; filename?: string; onAfterPrint?: () => void },
): void {
  void opts.filename;
  const printTitle = (opts.title || document.title || "").trim();
  printReceiptArticle(root, printTitle || undefined, opts.onAfterPrint);
}
