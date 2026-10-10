/**
 * Apply light/dark on <html> immediately, with transitions suppressed so
 * Tailwind transition-colors utilities do not cross-fade the theme.
 */
export type TrimThemeMode = "light" | "dark";

export function applyDocumentDarkClass(dark: boolean, mode?: TrimThemeMode): void {
  if (typeof document === "undefined") return;

  const root = document.documentElement;
  const style = document.createElement("style");
  style.setAttribute("data-trim-theme-lock", "");
  style.appendChild(
    document.createTextNode(
      "*,*::before,*::after{-webkit-transition:none!important;-moz-transition:none!important;-o-transition:none!important;-ms-transition:none!important;transition:none!important}",
    ),
  );
  document.head.appendChild(style);

  root.classList.toggle("dark", dark);
  root.style.colorScheme = dark ? "dark" : "light";
  if (mode) {
    root.dataset.trimTheme = mode;
  }

  void root.offsetHeight;

  const unlock = () => {
    style.parentNode?.removeChild(style);
  };
  requestAnimationFrame(() => {
    requestAnimationFrame(unlock);
  });
}

export function resolvePrefersDark(): boolean {
  if (typeof window === "undefined") return true;
  return window.matchMedia("(prefers-color-scheme: dark)").matches;
}
