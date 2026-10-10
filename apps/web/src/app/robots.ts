import type { MetadataRoute } from "next";
import { loadSiteChrome } from "@/lib/site-metadata";

export default async function robots(): Promise<MetadataRoute.Robots> {
  const site = await loadSiteChrome();
  const origin = (process.env.NEXT_PUBLIC_APP_URL || "").replace(/\/$/, "");
  const allowIndex =
    (site?.seo_robots_index || "").toLowerCase().includes("index") &&
    !(site?.seo_robots_index || "").toLowerCase().includes("noindex");

  if (!origin || !allowIndex) {
    return {
      rules: { userAgent: "*", disallow: "/" },
    };
  }

  const dashboard = site?.path_dashboard?.trim() || "/dashboard";
  const settings = site?.path_settings?.trim() || "";
  const team = site?.path_team?.trim() || "";
  const traces = site?.path_traces?.trim() || "";
  const receiptsList = site?.path_receipts?.trim() || "";
  const enterprise = site?.path_enterprise?.trim() || "";
  const receipts = site?.path_receipts_prefix?.trim() || "";
  const cli = site?.path_cli_auth?.trim() || "";
  const invite = site?.path_invite_prefix?.trim() || "";
  const callback = site?.path_auth_callback?.trim() || "";

  const disallow = [
    dashboard,
    settings,
    team,
    traces,
    receiptsList,
    enterprise,
    receipts,
    cli,
    invite,
    callback,
  ].filter((p): p is string => Boolean(p));

  return {
    rules: {
      userAgent: "*",
      allow: "/",
      disallow,
    },
    sitemap: `${origin}/sitemap.xml`,
  };
}
