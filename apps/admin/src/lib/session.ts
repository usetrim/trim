import { createClient } from "@/lib/supabase/server";

const GET_SESSION_MS = 5_000;

export async function getAccessToken(): Promise<string> {
  try {
    const supabase = await createClient();
    const result = await Promise.race([
      supabase.auth.getSession(),
      new Promise<{ data: { session: null } }>((resolve) =>
        setTimeout(() => resolve({ data: { session: null } }), GET_SESSION_MS),
      ),
    ]);
    return result.data.session?.access_token || "";
  } catch {
    // Fail closed to logged-out during build / missing env (pages redirect or return null).
    return "";
  }
}
