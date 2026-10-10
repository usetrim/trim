import { apiFetch } from "@/lib/api/client";
import { withQuery } from "@/lib/api/query-string";
import { qk } from "@/lib/query-keys";
import type { ApiKeyDevicesResponse, ApiKeysListResponse } from "@/types/api-keys";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function useApiKeys(token: string | undefined, skip = 0, limit?: number, search = "") {
  const pageSize = typeof limit === "number" && limit > 0 ? limit : 0;
  const q = search.trim();
  return useQuery({
    queryKey: [...qk.apiKeys, skip, pageSize, q],
    enabled: Boolean(token) && pageSize > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      apiFetch<ApiKeysListResponse>(
        withQuery("/api/v1/me/api-keys", { skip, limit: pageSize, q: q || undefined }),
        {
          token,
        },
      ),
  });
}

export function useApiKeyDevices(token: string | undefined, keyId: string | null) {
  return useQuery({
    queryKey: qk.apiKeyDevices(keyId || ""),
    enabled: Boolean(token) && Boolean(keyId),
    queryFn: () =>
      apiFetch<ApiKeyDevicesResponse>(`/api/v1/me/api-keys/${keyId}/devices`, {
        token,
      }),
  });
}
