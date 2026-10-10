import { apiFetch, rememberAuthNotSignedIn } from "@/lib/api/client";
import { qk } from "@/lib/query-keys";
import type { AuthProvidersResponse } from "@/types/auth";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function useAuthProviders() {
  return useQuery<AuthProvidersResponse>({
    queryKey: qk.authProviders,
    queryFn: async () => {
      const res = await apiFetch<AuthProvidersResponse>("/api/v1/public/auth-providers");
      rememberAuthNotSignedIn(res.auth_not_signed_in_message);
      return res;
    },
    staleTime: 5 * 60 * 1000,
    placeholderData: keepPreviousData,
  });
}
