"use client";

import { TrimWordmarkLink } from "@/components/brand/trim-wordmark";
import { Button } from "@/components/ui/button";
import { useAuthProviders } from "@/hooks/queries/auth";
import { statusSoftBgClass, statusTextClass } from "@/lib/status-color";
import { cn } from "@/lib/utils";
import { Check } from "lucide-react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef } from "react";

function CheckoutSuccessInner() {
  const router = useRouter();
  const search = useSearchParams();
  const auth = useAuthProviders();
  const site = auth.data?.site;
  const ran = useRef(false);

  const title = (site?.checkout_success_title || "").trim();
  const body = (site?.checkout_success_body || "").trim();
  const dashboardLabel = (
    site?.checkout_success_dashboard_label ||
    site?.nav_dashboard ||
    ""
  ).trim();
  const receiptsLabel = (site?.checkout_success_receipts_label || site?.nav_receipts || "").trim();
  const brand = (site?.brand || "").trim() || "Trim";
  const dashboardHref =
    site?.path_dashboard?.startsWith("/") && !site.path_dashboard.startsWith("//")
      ? site.path_dashboard
      : "/dashboard";
  const receiptsHref =
    site?.path_receipts?.startsWith("/") && !site.path_receipts.startsWith("//")
      ? site.path_receipts
      : "/dashboard/receipts";
  const plan = search.get("plan")?.trim() || "";
  const dashboardContinueHref = plan
    ? `${dashboardHref}?checkout=success&plan=${encodeURIComponent(plan)}`
    : `${dashboardHref}?checkout=success`;

  useEffect(() => {
    if (ran.current) return;
    ran.current = true;
    const t = window.setTimeout(() => {
      router.replace(dashboardContinueHref);
    }, 2800);
    return () => window.clearTimeout(t);
  }, [router, dashboardContinueHref]);

  return (
    <main className="flex min-h-screen flex-col bg-[var(--trim-bg)] text-[var(--trim-fg)]">
      <header className="flex items-center justify-between border-b border-[var(--trim-border)] bg-[var(--trim-panel)] px-5 py-3">
        <TrimWordmarkLink href="/" size="md" alt={brand} priority />
      </header>
      <div className="mx-auto flex w-full max-w-lg flex-1 flex-col items-center justify-center px-6 py-16 text-center">
        <div
          className={cn(
            "flex h-14 w-14 items-center justify-center rounded-full",
            statusSoftBgClass,
            statusTextClass,
          )}
          aria-hidden
        >
          <Check className="h-7 w-7" strokeWidth={2.5} />
        </div>
        {title ? (
          <h1 className="font-display mt-6 text-[1.5rem] font-medium tracking-[-0.02em]">
            {title}
          </h1>
        ) : null}
        {body ? (
          <p className="mt-3 text-[14px] leading-relaxed text-[var(--trim-muted)]">{body}</p>
        ) : null}
        <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
          {dashboardLabel ? (
            <Button asChild>
              <Link href={dashboardContinueHref}>{dashboardLabel}</Link>
            </Button>
          ) : null}
          {receiptsLabel ? (
            <Button asChild variant="outline">
              <Link href={receiptsHref}>{receiptsLabel}</Link>
            </Button>
          ) : null}
        </div>
      </div>
    </main>
  );
}

export function CheckoutSuccessClient() {
  return (
    <Suspense fallback={null}>
      <CheckoutSuccessInner />
    </Suspense>
  );
}
