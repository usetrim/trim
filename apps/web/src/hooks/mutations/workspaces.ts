import { apiFetch, requireAccessToken } from "@/lib/api/client";
import { invalidateWorkspaces } from "@/lib/query-keys";
import type { AcceptInviteResponse, InviteMemberResponse } from "@/types/workspace";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function useCreateWorkspace(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: { name: string }) => {
      requireAccessToken(token);
      return apiFetch<{ id: string; name: string }>("/api/v1/workspaces", {
        method: "POST",
        token,
        body: JSON.stringify(body),
      });
    },
    onSuccess: async () => {
      await invalidateWorkspaces(qc);
    },
  });
}

export function useRenameWorkspace(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: { workspaceId: string; name: string }) => {
      requireAccessToken(token);
      return apiFetch<{ id: string; name: string; message?: string }>(
        `/api/v1/workspaces/${body.workspaceId}`,
        {
          method: "PATCH",
          token,
          body: JSON.stringify({ name: body.name }),
        },
      );
    },
    onSuccess: async () => {
      await invalidateWorkspaces(qc);
    },
  });
}

export function useDeleteWorkspace(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (workspaceId: string) => {
      requireAccessToken(token);
      return apiFetch<{ id: string; message?: string }>(`/api/v1/workspaces/${workspaceId}`, {
        method: "DELETE",
        token,
      });
    },
    onSuccess: async () => {
      await invalidateWorkspaces(qc);
    },
  });
}

export function useInviteWorkspaceMember(token: string | undefined, workspaceId: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: { email: string; role: string }) => {
      if (!workspaceId) requireAccessToken(undefined);
      requireAccessToken(token);
      return apiFetch<InviteMemberResponse>(`/api/v1/workspaces/${workspaceId}/invite`, {
        method: "POST",
        token,
        body: JSON.stringify(body),
      });
    },
    onSuccess: async () => {
      await invalidateWorkspaces(qc);
    },
  });
}

export function useRevokeWorkspaceInvite(token: string | undefined, workspaceId: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (inviteId: string) => {
      if (!workspaceId) requireAccessToken(undefined);
      requireAccessToken(token);
      return apiFetch<{
        status: string;
        action_label: string;
        pending_label: string;
      }>(`/api/v1/workspaces/${workspaceId}/invites/${inviteId}`, {
        method: "DELETE",
        token,
      });
    },
    onSuccess: async () => {
      await invalidateWorkspaces(qc);
    },
  });
}

export function useAcceptWorkspaceInvite(accessToken: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (inviteToken: string) => {
      requireAccessToken(accessToken);
      return apiFetch<AcceptInviteResponse>(`/api/v1/workspace-invites/${inviteToken}/accept`, {
        method: "POST",
        token: accessToken,
      });
    },
    onSuccess: async () => {
      await invalidateWorkspaces(qc);
    },
  });
}

export function useRemoveWorkspaceMember(token: string | undefined, workspaceId: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (memberId: string) => {
      if (!workspaceId) requireAccessToken(undefined);
      requireAccessToken(token);
      return apiFetch<{ status: string; user_id: string }>(
        `/api/v1/workspaces/${workspaceId}/members/${memberId}`,
        {
          method: "DELETE",
          token,
        },
      );
    },
    onSuccess: async () => {
      await invalidateWorkspaces(qc);
    },
  });
}

export function useUpdateWorkspaceMemberRole(
  token: string | undefined,
  workspaceId: string | null,
) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: { memberId: string; role: string }) => {
      if (!workspaceId) requireAccessToken(undefined);
      requireAccessToken(token);
      return apiFetch<{
        status: string;
        id: string;
        role: string;
        role_label?: string;
        message?: string;
      }>(`/api/v1/workspaces/${workspaceId}/members/${body.memberId}`, {
        method: "PATCH",
        token,
        body: JSON.stringify({ role: body.role }),
      });
    },
    onSuccess: async () => {
      await invalidateWorkspaces(qc);
    },
  });
}
