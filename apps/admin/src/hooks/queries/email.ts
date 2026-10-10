import { adminFetch } from "@/lib/admin-api/client";
import { withQuery } from "@/lib/admin-api/query-string";
import { adminQk } from "@/lib/query-keys";
import type { ListResponse } from "@/types/admin";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

export type EmailTemplateItem = {
  code: string;
  kind: string;
  kind_label?: string;
  sort_order?: number;
  body?: string;
};

export type EmailTemplatesPayload = ListResponse<EmailTemplateItem> & {
  smtp_configured?: boolean;
  smtp_status_label?: string;
};

export function useAdminEmailTemplates(token: string, skip: number, limit: number, q = "") {
  const search = q.trim();
  return useQuery({
    queryKey: [...adminQk.emailTemplates, skip, limit, search],
    enabled: Boolean(token) && limit > 0,
    placeholderData: keepPreviousData,
    queryFn: () =>
      adminFetch<EmailTemplatesPayload>(
        withQuery("/api/v1/admin/email/templates", { skip, limit, q: search || undefined }),
        { token },
      ),
  });
}
