"use client";

import { InviteAcceptSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useAcceptWorkspaceInvite } from "@/hooks/mutations/workspaces";
import { useAuthProviders } from "@/hooks/queries/auth";
import { useInvitePreview } from "@/hooks/queries/workspaces";
import { formatDateTimeShort } from "@/lib/format-datetime";
import { createClient } from "@/lib/supabase/client";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export function InviteAcceptClient() {
  const params = useParams<{ token: string }>();
  const token = typeof params.token === "string" ? params.token : "";
  const router = useRouter();
  const preview = useInvitePreview(token || null);
  const authProviders = useAuthProviders();
  const [accessToken, setAccessToken] = useState<string | undefined>();
  const [sessionReady, setSessionReady] = useState(false);
  const [acceptError, setAcceptError] = useState<string | null>(null);
  const acceptMutation = useAcceptWorkspaceInvite(accessToken);
  const htmlLang = authProviders.data?.site?.html_lang?.trim() || "";
  const expiresDisplay =
    formatDateTimeShort(preview.data?.expires_at, htmlLang) ||
    preview.data?.expires_at_label?.trim() ||
    "";

  useEffect(() => {
    const supabase = createClient();
    void supabase.auth.getSession().then(({ data }) => {
      setAccessToken(data.session?.access_token);
      setSessionReady(true);
    });
  }, []);

  if (!sessionReady || preview.isLoading) {
    return (
      <InviteAcceptSkeleton
        chrome={{
          eyebrow: preview.data?.eyebrow_label,
          title: preview.data?.workspace_name,
          body: preview.data?.body_message,
          status_title: preview.data?.status_title,
          accept_label: preview.data?.accept_action_label,
        }}
      />
    );
  }

  if (preview.error) {
    return (
      <div className="mx-auto flex min-h-screen w-full max-w-lg flex-col justify-center px-6 py-16">
        <p className="text-sm text-destructive">
          {(preview.error instanceof Error && preview.error.message) || ""}
        </p>
        {preview.data?.sign_in_href ? (
          <Button asChild variant="outline" className="mt-4 w-fit">
            <Link href={preview.data.sign_in_href}>{preview.data?.sign_in_action_label || ""}</Link>
          </Button>
        ) : null}
      </div>
    );
  }

  const data = preview.data;
  const canAccept = data?.status === "pending" && Boolean(accessToken);

  return (
    <div className="mx-auto flex min-h-screen w-full max-w-lg flex-col justify-center px-6 py-16">
      <p className="text-sm text-[var(--trim-muted)]">{data?.eyebrow_label}</p>
      <h1 className="mt-2 text-3xl font-semibold tracking-tight text-[var(--trim-fg)]">
        {data?.workspace_name || ""}
      </h1>
      <p className="mt-2 text-sm text-[var(--trim-muted)]">
        {data?.role_prefix}{" "}
        <span className="text-[var(--trim-fg)]">{data?.role_label || data?.role}</span>{" "}
        {data?.for_prefix} {data?.email_masked}. {data?.body_message}
      </p>

      <Card className="mt-8 border-[var(--trim-border)] bg-[var(--trim-panel)]">
        <CardHeader>
          <CardTitle className="text-base text-[var(--trim-fg)]">{data?.status_title}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4 text-sm text-[var(--trim-muted)]">
          <p>
            {data?.status_prefix}{" "}
            <span className="text-[var(--trim-fg)]">{data?.status_label || ""}</span>
          </p>
          <p>
            {data?.expires_prefix} {expiresDisplay}
          </p>
          {acceptError ? <p className="text-destructive">{acceptError}</p> : null}
          {acceptMutation.error ? (
            <p className="text-destructive">
              {(acceptMutation.error instanceof Error && acceptMutation.error.message) || ""}
            </p>
          ) : null}

          {!accessToken && data?.sign_in_href && data?.invite_path_prefix ? (
            <Button asChild className="w-full">
              <Link
                href={`${data.sign_in_href}?next=${encodeURIComponent(
                  `${data.invite_path_prefix}${token}`,
                )}`}
              >
                {data?.sign_in_action_label}
              </Link>
            </Button>
          ) : null}

          {canAccept ? (
            <Button
              className="w-full"
              disabled={!data?.accept_action_label}
              isLoading={acceptMutation.isPending}
              pendingLabel={data?.accept_pending_label || undefined}
              onClick={() => {
                setAcceptError(null);
                acceptMutation.mutate(token, {
                  onSuccess: (res) => {
                    if (res.redirect_href) {
                      router.push(res.redirect_href);
                    }
                  },
                  onError: (err) => {
                    setAcceptError(err instanceof Error ? err.message : "");
                  },
                });
              }}
            >
              {data?.accept_action_label}
            </Button>
          ) : null}

          {accessToken && data?.status !== "pending" && data?.open_team_href ? (
            <Button asChild variant="outline" className="w-full">
              <Link href={data.open_team_href}>{data?.open_team_action_label || ""}</Link>
            </Button>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
