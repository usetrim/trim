import type { MetadataRoute } from "next";
import { flattenDocNav } from "@/lib/docs/nav";
import { loadSiteChrome } from "@/lib/site-metadata";

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const origin = (process.env.NEXT_PUBLIC_APP_URL || "").replace(/\/$/, "");
  if (!origin) return [];

  const site = await loadSiteChrome();
  const home = site?.path_home?.trim() || "/";
  const login = site?.path_login?.trim() || "/login";
  const privacy = site?.path_privacy?.trim() || "/privacy";
  const terms = site?.path_terms?.trim() || "/terms";
  const pricing = "/pricing";
  const docsRoot = site?.page_404_docs_href?.trim() || "/docs";
  const contact = site?.path_contact?.trim() || "";

  const now = new Date();
  const entries: MetadataRoute.Sitemap = [
    {
      url: `${origin}${home === "/" ? "" : home}` || origin,
      lastModified: now,
      changeFrequency: "weekly",
      priority: 1,
    },
    { url: `${origin}${pricing}`, lastModified: now, changeFrequency: "weekly", priority: 0.9 },
    { url: `${origin}${login}`, lastModified: now, changeFrequency: "monthly", priority: 0.6 },
    { url: `${origin}${privacy}`, lastModified: now, changeFrequency: "yearly", priority: 0.3 },
    { url: `${origin}${terms}`, lastModified: now, changeFrequency: "yearly", priority: 0.3 },
    { url: `${origin}${docsRoot}`, lastModified: now, changeFrequency: "weekly", priority: 0.8 },
  ];
  if (contact) {
    entries.push({
      url: `${origin}${contact}`,
      lastModified: now,
      changeFrequency: "monthly",
      priority: 0.7,
    });
  }

  for (const item of flattenDocNav()) {
    const slug = item.slug?.trim();
    if (!slug) continue;
    entries.push({
      url: `${origin}${docsRoot}/${slug}`,
      lastModified: now,
      changeFrequency: "monthly",
      priority: 0.7,
    });
  }

  return entries;
}
