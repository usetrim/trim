import { createBrowserClient } from "@supabase/ssr";

export function createClient() {
  const url = process.env.NEXT_PUBLIC_SUPABASE_URL;
  const key = process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY;
  if (!url || !key) {
    const missing = process.env.NEXT_PUBLIC_SUPABASE_ENV_MISSING?.trim();
    throw new Error(missing || "");
  }
  return createBrowserClient(url, key);
}
