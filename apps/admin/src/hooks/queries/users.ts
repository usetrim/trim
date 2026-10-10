import { adminFetch } from "@/lib/admin-api/client";
import { withQuery } from "@/lib/admin-api/query-string";
import { adminQk } from "@/lib/query-keys";
import type {
  AdminFilterOption,
  AdminUserListItem,
  AdminUsersFilters,
  ListResponse,
} from "@/types/admin";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export function useAdminUsers(
  token: string,
  skip: number,
  limit: number,
  filters: AdminUsersFilters = {},
) {
  const q = filters.q?.trim() || "";
  const status = filters.status?.trim() || "";
  const country = filters.country?.trim() || "";
  const plan = filters.plan?.trim() || "";
  const provider = filters.provider?.trim() || "";
  const createdFrom = filters.created_from?.trim() || "";
  const createdTo = filters.created_to?.trim() || "";
  return useQuery({
    queryKey: [
      ...adminQk.users,
      skip,
      limit,
      q,
      status,
      country,
      plan,
      provider,
      createdFrom,
      createdTo,
    ],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse<AdminUserListItem>>(
        withQuery("/api/v1/admin/users", {
          skip,
          limit,
          q: q || undefined,
          status: status || undefined,
          country: country || undefined,
          plan: plan || undefined,
          provider: provider || undefined,
          created_from: createdFrom || undefined,
          created_to: createdTo || undefined,
        }),
        { token },
      ),
  });
}

/** Distinct last_login_country codes from profiles (shared users + segments filters). */
export function useAdminUserCountries(token: string, source: "users" | "segments" = "users") {
  const path =
    source === "segments" ? "/api/v1/admin/segments/countries" : "/api/v1/admin/users/countries";
  return useQuery({
    queryKey: [...adminQk.userCountries, source],
    enabled: Boolean(token),
    queryFn: () => adminFetch<ListResponse<AdminFilterOption>>(path, { token }),
  });
}

export function useAdminUser(token: string, id: string) {
  return useQuery({
    queryKey: adminQk.user(id),
    enabled: Boolean(token) && Boolean(id),
    queryFn: () =>
      adminFetch<Record<string, unknown>>(`/api/v1/admin/users/${id}`, {
        token,
      }),
  });
}

export function useAdminCreditGrants(token: string, userId: string, skip: number, limit: number) {
  return useQuery({
    queryKey: [...adminQk.creditGrants(userId), skip, limit],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse>(
        withQuery("/api/v1/admin/billing/credits", {
          skip,
          limit,
          ...(userId ? { user_id: userId } : {}),
        }),
        { token },
      ),
  });
}

export function useAdminTopupLedger(token: string, userId: string, skip: number, limit: number) {
  return useQuery({
    queryKey: [...adminQk.topupLedger(userId), skip, limit],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse>(
        withQuery("/api/v1/admin/billing/topups", {
          skip,
          limit,
          ...(userId ? { user_id: userId } : {}),
        }),
        { token },
      ),
  });
}
