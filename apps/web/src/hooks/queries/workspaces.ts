import { apiFetch } from "@/lib/api/client";
import { withQuery } from "@/lib/api/query-string";
import { qk } from "@/lib/query-keys";
import type {
  InvitePreviewResponse,
  WorkspaceInvitesResponse,
  WorkspaceMembersResponse,
  WorkspacesListResponse,
} from "@/types/workspace";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function useWorkspaces(
  token: string | undefined,
  skip: number,
  limit?: number,
  search = "",
) {
  const pageSize = typeof limit === "number" && limit > 0 ? limit : 0;
  const q = search.trim();
  return useQuery({
    queryKey: [...qk.workspaces, skip, pageSize, q],
    enabled: Boolean(token) && pageSize > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      apiFetch<WorkspacesListResponse>(
        withQuery("/api/v1/workspaces", { skip, limit: pageSize, q: q || undefined }),
        {
          token,
        },
      ),
  });
}

/** Keep prior page only when the workspace id is unchanged (pagination), not on workspace switch. */
function keepSameWorkspacePreviousData<T>(workspaceId: string | null) {
  return (
    previousData: T | undefined,
    previousQuery: { queryKey: readonly unknown[] } | undefined,
  ) => {
    if (!previousData || !previousQuery) return undefined;
    const prevWorkspaceId = previousQuery.queryKey[1];
    if (prevWorkspaceId !== workspaceId) return undefined;
    return previousData;
  };
}

export function useWorkspaceMembers(
  token: string | undefined,
  workspaceId: string | null,
  skip: number,
  limit?: number,
  search = "",
) {
  const pageSize = typeof limit === "number" && limit > 0 ? limit : 0;
  const q = search.trim();
  return useQuery({
    queryKey: [...qk.workspaceMembers, workspaceId, skip, pageSize, q],
    enabled: Boolean(token) && Boolean(workspaceId) && pageSize > 0,
    placeholderData: keepSameWorkspacePreviousData<WorkspaceMembersResponse>(workspaceId),
    queryFn: () =>
      apiFetch<WorkspaceMembersResponse>(
        withQuery(`/api/v1/workspaces/${workspaceId}/members`, {
          skip,
          limit: pageSize,
          q: q || undefined,
        }),
        { token },
      ),
  });
}

export function useWorkspaceInvites(
  token: string | undefined,
  workspaceId: string | null,
  skip: number,
  limit?: number,
  search = "",
) {
  const pageSize = typeof limit === "number" && limit > 0 ? limit : 0;
  const q = search.trim();
  return useQuery({
    queryKey: [...qk.workspaceInvites, workspaceId, skip, pageSize, q],
    enabled: Boolean(token) && Boolean(workspaceId) && pageSize > 0,
    placeholderData: keepSameWorkspacePreviousData<WorkspaceInvitesResponse>(workspaceId),
    queryFn: () =>
      apiFetch<WorkspaceInvitesResponse>(
        withQuery(`/api/v1/workspaces/${workspaceId}/invites`, {
          skip,
          limit: pageSize,
          q: q || undefined,
        }),
        { token },
      ),
  });
}

export function useInvitePreview(token: string | null) {
  return useQuery({
    queryKey: [...qk.invitePreview, token],
    enabled: Boolean(token),
    queryFn: () => apiFetch<InvitePreviewResponse>(`/api/v1/public/workspace-invites/${token}`),
  });
}
