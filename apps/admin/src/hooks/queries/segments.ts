import { adminFetch } from "@/lib/admin-api/client";
import { withQuery } from "@/lib/admin-api/query-string";
import { adminQk } from "@/lib/query-keys";
import type { ListResponse } from "@/types/admin";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export type SegmentFilters = {
  plan?: string;
  status?: string;
  country?: string;
  interval?: string;
  created_from?: string;
  created_to?: string;
  credits_left_max?: string;
  q?: string;
};

export function useAdminSegmentsIndividuals(
  token: string,
  skip: number,
  limit: number,
  filters: SegmentFilters = {},
) {
  return useQuery({
    queryKey: [
      ...adminQk.segmentsIndividuals,
      skip,
      limit,
      filters.plan || "",
      filters.status || "",
      filters.country || "",
      filters.interval || "",
      filters.created_from || "",
      filters.created_to || "",
      filters.credits_left_max || "",
      filters.q || "",
    ],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse>(
        withQuery("/api/v1/admin/segments/individuals", {
          skip,
          limit,
          plan: filters.plan,
          status: filters.status,
          country: filters.country,
          interval: filters.interval,
          created_from: filters.created_from,
          created_to: filters.created_to,
          credits_left_max: filters.credits_left_max,
          q: filters.q,
        }),
        { token },
      ),
  });
}

export function useAdminSegmentsTeams(
  token: string,
  skip: number,
  limit: number,
  filters: SegmentFilters = {},
) {
  return useQuery({
    queryKey: [
      ...adminQk.segmentsTeams,
      skip,
      limit,
      filters.plan || "",
      filters.q || "",
      filters.created_from || "",
      filters.created_to || "",
    ],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse>(
        withQuery("/api/v1/admin/segments/teams", {
          skip,
          limit,
          plan: filters.plan,
          q: filters.q,
          created_from: filters.created_from,
          created_to: filters.created_to,
        }),
        { token },
      ),
  });
}

export function useAdminSegmentsEnterprises(
  token: string,
  skip: number,
  limit: number,
  filters: SegmentFilters = {},
) {
  return useQuery({
    queryKey: [
      ...adminQk.segmentsEnterprises,
      skip,
      limit,
      filters.plan || "",
      filters.status || "",
      filters.country || "",
      filters.interval || "",
      filters.created_from || "",
      filters.created_to || "",
      filters.credits_left_max || "",
      filters.q || "",
    ],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<ListResponse>(
        withQuery("/api/v1/admin/segments/enterprises", {
          skip,
          limit,
          plan: filters.plan,
          status: filters.status,
          country: filters.country,
          interval: filters.interval,
          created_from: filters.created_from,
          created_to: filters.created_to,
          credits_left_max: filters.credits_left_max,
          q: filters.q,
        }),
        { token },
      ),
  });
}
