"use client";

import { CliAuthPageSkeleton } from "@/components/skeletons/page-skeletons";
import { useAuthProviders } from "@/hooks/queries/auth";

/** Suspense fallback with layout-faithful chrome from auth-providers when warm. */
export function CliAuthSuspenseFallback() {
  const authProviders = useAuthProviders();
  return (
    <CliAuthPageSkeleton
      chrome={{
        eyebrow: authProviders.data?.cli?.eyebrow,
        title: authProviders.data?.cli?.title,
        body: authProviders.data?.cli?.body,
        copy_label: authProviders.data?.cli?.copy_action_label,
        issue_label: authProviders.data?.cli?.issue_another_action_label,
        footer: authProviders.data?.cli?.footer,
      }}
    />
  );
}
