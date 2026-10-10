import { createClient } from "@/lib/supabase/server";

export async function getServerAccessToken(): Promise<string | undefined> {
  const supabase = await createClient();
  const {
    data: { session },
  } = await supabase.auth.getSession();
  return session?.access_token;
}
