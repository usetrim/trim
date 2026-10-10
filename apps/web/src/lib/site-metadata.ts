import { loadPublicAuthProviders } from "@/lib/public-auth-providers";
import type { AuthProvidersSiteChrome } from "@/types/auth";
import type { Metadata, Viewport } from "next";

function appOrigin(): string {
  return (process.env.NEXT_PUBLIC_APP_URL || "").replace(/\/$/, "");
}

function parseRobots(raw: string | undefined): Metadata["robots"] | undefined {
  const v = (raw || "").trim().toLowerCase();
  if (!v) return undefined;
  const index = !v.includes("noindex");
  const follow = !v.includes("nofollow");
  return { index, follow };
}

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

/** Root / marketing metadata from auth-providers site chrome + NEXT_PUBLIC_APP_URL. */
export async function buildWebRootMetadata(): Promise<Metadata> {
  const data = await loadPublicAuthProviders();
  const site = data?.site;
  const title = site?.seo_default_title?.trim() || site?.brand?.trim() || "";
  const description = site?.seo_default_description?.trim() || site?.tagline?.trim() || "";
  const template = site?.seo_title_template?.trim() || "";
  const origin = appOrigin();
  const ogSite = site?.seo_og_site_name?.trim() || title;
  const ogType = site?.seo_og_type?.trim() || "website";
  const twitterCard = site?.seo_twitter_card?.trim() || "";
  const robots = parseRobots(site?.seo_robots_index);

  if (!title || !description) {
    return {
      icons: siteIcons,
      manifest: "/site.webmanifest",
    };
  }

  const meta: Metadata = {
    title: template.includes("%s") ? { default: title, template } : title,
    description,
    applicationName: ogSite || undefined,
    robots,
    icons: siteIcons,
    manifest: "/site.webmanifest",
  };

  if (origin) {
    meta.metadataBase = new URL(origin);
    meta.alternates = { canonical: "/" };
    meta.openGraph = {
      type: ogType as "website",
      locale: site?.html_lang?.trim() || undefined,
      url: origin,
      siteName: ogSite || undefined,
      title,
      description,
    };
    if (twitterCard) {
      meta.twitter = {
        card: twitterCard as "summary_large_image",
        title,
        description,
      };
    }
  }

  return meta;
}

export async function loadSiteChrome(): Promise<AuthProvidersSiteChrome | null> {
  const data = await loadPublicAuthProviders();
  return data?.site ?? null;
}
