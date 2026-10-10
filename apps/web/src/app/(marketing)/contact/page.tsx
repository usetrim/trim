import { loadPublicAuthProviders } from "@/lib/public-auth-providers";
import type { Metadata } from "next";

export async function generateMetadata(): Promise<Metadata> {
  const data = await loadPublicAuthProviders();
  const site = data?.site;
  const title = site?.contact_page_title?.trim() || "";
  const description = site?.contact_meta_description?.trim() || "";
  if (!title) return {};
  return {
    title,
    description: description || undefined,
  };
}

/** Soft-nav: contact body is owned by LandingShell (stays mounted). */
export default function ContactPage() {
  return null;
}
