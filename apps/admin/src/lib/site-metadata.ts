import type { Metadata, Viewport } from "next";
import { cache } from "react";

type SiteChrome = Record<string, string>;

/** Local API origins are often down during `next build`; skip network (fail closed, no ECONNREFUSED spam). */
function isLocalApiBase(base: string): boolean {
  try {
    const host = new URL(base).hostname;
    return host === "localhost" || host === "127.0.0.1" || host === "::1";
  } catch {
    return false;
  }
}

const loadPublicSite = cache(async (): Promise<SiteChrome | null> => {
  const base = process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, "");
  if (!base) return null;
  // Production Vercel builds use a real API host and still fetch; local builds do not.
  if (process.env.NEXT_PHASE === "phase-production-build" && isLocalApiBase(base)) {
    return null;
  }
  try {
    const res = await fetch(`${base}/api/v1/public/auth-providers`, {
      next: { revalidate: 300 },
      signal: AbortSignal.timeout(8_000),
    });
    if (!res.ok) return null;
    const data = (await res.json()) as { site?: SiteChrome };
    return data.site ?? null;
  } catch {
    return null;
  }
});

/** Default tab icons (no OS media). FaviconThemeSync swaps to match `html.dark`. */
export const siteIcons: NonNullable<Metadata["icons"]> = {
  icon: [
    { url: "/icons/icon-32-light.png", type: "image/png", sizes: "32x32" },
    { url: "/icons/icon-16-light.png", type: "image/png", sizes: "16x16" },
    { url: "/favicon.ico", type: "image/x-icon", sizes: "any" },
  ],
  apple: [
    { url: "/icons/apple-touch-icon.png", sizes: "180x180", type: "image/png" },
    { url: "/apple-touch-icon.png", sizes: "180x180", type: "image/png" },
  ],
  shortcut: ["/icons/icon-32-light.png"],
};

/** themeColor belongs on viewport (not metadata) - Next.js App Router. */
export const siteViewport: Viewport = {
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#ffffff" },
    { media: "(prefers-color-scheme: dark)", color: "#000000" },
  ],
};

export async function buildAdminRootMetadata(): Promise<Metadata> {
  const site = await loadPublicSite();
  const title = site?.admin_seo_default_title?.trim() || "";
  const description = site?.admin_seo_default_description?.trim() || "";
  const robotsRaw = (site?.admin_seo_robots || "").toLowerCase();
  if (!title || !description) {
    return {
      robots: { index: false, follow: false },
      icons: siteIcons,
      manifest: "/site.webmanifest",
    };
  }
  return {
    title: { default: title, template: `%s · ${title}` },
    description,
    robots: {
      index: !robotsRaw.includes("noindex"),
      follow: !robotsRaw.includes("nofollow"),
    },
    icons: siteIcons,
    manifest: "/site.webmanifest",
  };
}

export async function loadAdminSiteChrome(): Promise<SiteChrome | null> {
  return loadPublicSite();
}
