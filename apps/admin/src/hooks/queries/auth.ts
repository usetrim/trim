import { adminFetch } from "@/lib/admin-api/client";
import { adminQk } from "@/lib/query-keys";
import type { AuthProvidersPayload, AuthSettings, OAuthScopesPayload } from "@/types/admin";
import { useQuery } from "@tanstack/react-query";

export function useAdminAuthSettings(token: string) {
  return useQuery({
    queryKey: adminQk.authSettings,
    enabled: Boolean(token),
    queryFn: () => adminFetch<AuthSettings>("/api/v1/admin/auth/settings", { token }),
  });
}

export function useAdminOAuthScopes(token: string) {
  return useQuery({
    queryKey: adminQk.oauthScopes,
    enabled: Boolean(token),
    queryFn: () =>
      adminFetch<OAuthScopesPayload>("/api/v1/admin/auth/oauth-scopes", {
        token,
      }),
  });
}

export function useAuthProvidersPublic() {
  return useQuery({
    queryKey: adminQk.publicAuthProviders,
    queryFn: () => adminFetch<AuthProvidersPayload>("/api/v1/public/auth-providers"),
  });
}

export function useAuthProviders() {
  return useAuthProvidersPublic();
}
