import { apiFetch, requireAccessToken } from "@/lib/api/client";
import { qk } from "@/lib/query-keys";
import type { CreateApiKeyResponse } from "@/types/api-keys";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function useCreateApiKey(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body?: {
      label?: string;
      hardware_uuid?: string;
      agent_id?: string;
    }) => {
      requireAccessToken(token);
      return apiFetch<CreateApiKeyResponse>("/api/v1/me/api-keys", {
        method: "POST",
        token,
        body: JSON.stringify(body ?? {}),
      });
    },
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: qk.apiKeys });
    },
  });
}

export function useRevokeApiKey(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (keyId: string) => {
      requireAccessToken(token);
      return apiFetch<{
        status: string;
        action_label: string;
        pending_label: string;
      }>(`/api/v1/me/api-keys/${keyId}`, {
        method: "DELETE",
        token,
      });
    },
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: qk.apiKeys });
    },
  });
}

export function useRegisterApiKeyDevice(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (args: {
      keyId: string;
      hardware_uuid: string;
      agent_id: string;
    }) => {
      requireAccessToken(token);
      return apiFetch<{
        status: string;
        action_label: string;
        pending_label: string;
      }>(`/api/v1/me/api-keys/${args.keyId}/devices`, {
        method: "POST",
        token,
        body: JSON.stringify({
          hardware_uuid: args.hardware_uuid,
          agent_id: args.agent_id,
        }),
      });
    },
    onSuccess: async (_data, vars) => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: qk.apiKeys }),
        qc.invalidateQueries({ queryKey: qk.apiKeyDevices(vars.keyId) }),
      ]);
    },
  });
}

export function useRemoveApiKeyDevice(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (args: { keyId: string; hardware_uuid: string }) => {
      requireAccessToken(token);
      const hw = encodeURIComponent(args.hardware_uuid);
      return apiFetch<{
        status: string;
        action_label: string;
        pending_label: string;
      }>(`/api/v1/me/api-keys/${args.keyId}/devices/${hw}`, {
        method: "DELETE",
        token,
      });
    },
    onSuccess: async (_data, vars) => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: qk.apiKeys }),
        qc.invalidateQueries({ queryKey: qk.apiKeyDevices(vars.keyId) }),
      ]);
    },
  });
}
