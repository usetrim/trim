"use client";

import { useBrowserAuthProfile } from "@/components/auth/user-menu";
import { PlanModal } from "@/components/billing/plan-modal";
import { LandingPricingCardSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { usePlans } from "@/hooks/queries/plans";
import { statusSoftBgClass, statusTextClass } from "@/lib/status-color";
import { cn, formatMoney } from "@/lib/utils";
import { type BillingInterval, useBillingUI } from "@/stores/billing-ui";
import type { AuthProvidersSiteChrome } from "@/types/auth";
import { Check } from "lucide-react";
import Link from "next/link";
import { startTransition, useEffect, useMemo, useState } from "react";

type Props = {
  site: AuthProvidersSiteChrome;
  signInHref: string;
  /** Title lives in sticky left rail on desktop; keep heading for mobile only. */
  sidebarChrome?: boolean;
};

function formatAnnualSaveHint(fmt: string, percent: number): string {
  const f = fmt.trim();
  if (!f || percent < 1) return "";
  if (!f.includes("%d")) return "";
  // printf-style: %d → percent, %% → literal %
  return f.replace(/%d/g, String(percent)).replace(/%%/g, "%");
}

/** Signed-out CTA: login, then return to pricing (checkout stays on this page after auth). */
function resolveSignedOutPricingHref(loginPath: string, pricingPath: string): string {
  const login = loginPath.trim();
  const pricing = pricingPath.trim() || "/pricing";
  if (!login) return pricing;
  if (!pricing.startsWith("/") || pricing.startsWith("//")) return login;
  return `${login}?next=${encodeURIComponent(pricing)}`;
}

export function LandingPricing({ site, signInHref, sidebarChrome = false }: Props) {
  const title = (site.pricing_title || "").trim();
  const subtitle = (site.pricing_subtitle || "").trim();
  const { ready, signedIn, profile, accessToken, userId } = useBrowserAuthProfile();
  const { setInterval } = useBillingUI();
  const [billingInterval, setBillingInterval] = useState<BillingInterval>("monthly");
  const [checkoutOpen, setCheckoutOpen] = useState(false);
  const [intentPlanId, setIntentPlanId] = useState<string | null>(null);
  const [seatQty, setSeatQty] = useState(0);
  // Prefetch both intervals so switch flips never wait on a cold request.
  const monthlyPlans = usePlans("monthly", signedIn ? accessToken : undefined);
  const annualPlans = usePlans("annual", signedIn ? accessToken : undefined);
  const plans = billingInterval === "monthly" ? monthlyPlans : annualPlans;

  const moneyLocale = plans.data?.money_locale?.trim() || "";
  const settings = plans.data?.settings;
  const list = useMemo(() => {
    const rows = plans.data?.plans ?? [];
    return rows.filter((p) => p.plan_kind !== "topup");
  }, [plans.data?.plans]);
  const showSeatField = useMemo(() => list.some((p) => p.per_seat), [list]);

  useEffect(() => {
    const n = settings?.default_seat_quantity;
    if (typeof n === "number" && n >= 1 && seatQty < 1) {
      setSeatQty(n);
    }
  }, [settings?.default_seat_quantity, seatQty]);

  const popularBadge = (site.pricing_popular_badge || "").trim();
  const annualBillingLabel = (site.pricing_annual_billing || "").trim();
  const unlimitedFeatureLabel = (settings?.unlimited_feature_label || "").trim();
  const annualDiscount = settings?.annual_discount_percent ?? 0;
  const annualSaveHint = formatAnnualSaveHint(
    site.pricing_annual_save_hint_fmt || "",
    annualDiscount,
  );
  const switchAria =
    (billingInterval === "annual"
      ? site.pricing_annual || settings?.annual_toggle_label
      : site.pricing_monthly || settings?.monthly_toggle_label) ||
    annualBillingLabel ||
    undefined;

  const pricingPath = "/pricing";
  const loginHref = useMemo(() => {
    if (!ready || signedIn) return "";
    return resolveSignedOutPricingHref(
      signInHref || site.nav_sign_in_href || site.path_login || "",
      pricingPath,
    );
  }, [ready, signedIn, signInHref, site.nav_sign_in_href, site.path_login]);

  function startPlanCheckout(planId: string) {
    // Keep pricing interval in sync with checkout (seat qty passed into PlanModal).
    setInterval(billingInterval);
    setIntentPlanId(planId);
    setCheckoutOpen(true);
  }

  if (!title) return null;

  return (
    <section id="pricing" className="w-full px-5 py-16 sm:px-8 xl:px-12">
      <div className="mx-auto flex w-full max-w-6xl flex-col items-center text-center">
        <div className={cn("max-w-2xl", sidebarChrome ? "lg:hidden" : undefined)}>
          <h2 className="font-display text-xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-2xl">
            {title}
          </h2>
          {subtitle ? (
            <p className="mt-3 text-sm text-[var(--trim-muted)] sm:text-base">{subtitle}</p>
          ) : null}
        </div>

        {annualBillingLabel ? (
          <div
            className={cn(
              "flex flex-col items-center gap-2",
              sidebarChrome ? "mt-6 lg:mt-0" : "mt-8",
            )}
          >
            <div className="inline-flex items-center gap-3 rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] px-3 py-2 shadow-[var(--trim-card-shadow)]">
              <div className="min-w-0 text-left">
                <p
                  className={cn(
                    "text-[11px] font-medium transition-colors sm:text-sm",
                    billingInterval === "annual"
                      ? "text-[var(--trim-fg)]"
                      : "text-[var(--trim-muted)]",
                  )}
                >
                  {annualBillingLabel}
                </p>
                {annualSaveHint ? (
                  <p
                    className={cn(
                      "text-[11px] font-medium transition-opacity sm:text-xs",
                      statusTextClass,
                      billingInterval === "annual" ? "opacity-100" : "opacity-70",
                    )}
                  >
                    {annualSaveHint}
                  </p>
                ) : null}
              </div>
              <Switch
                checked={billingInterval === "annual"}
                onCheckedChange={(checked) =>
                  startTransition(() => setBillingInterval(checked ? "annual" : "monthly"))
                }
                aria-label={switchAria}
              />
            </div>
          </div>
        ) : site.pricing_monthly ||
          site.pricing_annual ||
          settings?.monthly_toggle_label ||
          settings?.annual_toggle_label ? (
          <div
            className={cn(
              "inline-flex rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] p-1 shadow-[var(--trim-card-shadow)]",
              sidebarChrome ? "mt-6 lg:mt-0" : "mt-6",
            )}
          >
            <button
              type="button"
              className={`rounded-md px-4 py-1.5 text-sm transition ${
                billingInterval === "monthly"
                  ? "bg-[var(--trim-ink)] text-[var(--trim-ink-inverse)]"
                  : "text-[var(--trim-muted)] hover:bg-[var(--trim-hover)] hover:text-[var(--trim-fg)]"
              }`}
              onClick={() => startTransition(() => setBillingInterval("monthly"))}
            >
              {site.pricing_monthly || settings?.monthly_toggle_label || ""}
            </button>
            <button
              type="button"
              className={`rounded-md px-4 py-1.5 text-sm transition ${
                billingInterval === "annual"
                  ? "bg-[var(--trim-ink)] text-[var(--trim-ink-inverse)]"
                  : "text-[var(--trim-muted)] hover:bg-[var(--trim-hover)] hover:text-[var(--trim-fg)]"
              }`}
              onClick={() => startTransition(() => setBillingInterval("annual"))}
            >
              {site.pricing_annual || settings?.annual_toggle_label || ""}
            </button>
          </div>
        ) : null}

        {showSeatField && settings?.seat_quantity_label ? (
          <Field
            id="pricing-seat-qty"
            label={settings.seat_quantity_label}
            description={settings.seat_quantity_hint}
            className="mx-auto mt-4 max-w-xs text-left"
          >
            <Input
              id="pricing-seat-qty"
              type="number"
              min={
                settings.min_seat_quantity && settings.min_seat_quantity >= 1
                  ? settings.min_seat_quantity
                  : undefined
              }
              max={
                settings.max_seat_quantity && settings.max_seat_quantity > 0
                  ? settings.max_seat_quantity
                  : undefined
              }
              value={seatQty > 0 ? seatQty : ""}
              placeholder={settings.seat_quantity_label}
              onChange={(e) => {
                const floor =
                  settings.min_seat_quantity && settings.min_seat_quantity >= 1
                    ? settings.min_seat_quantity
                    : 0;
                const n = Number(e.target.value);
                if (!Number.isFinite(n) || floor < 1) return;
                setSeatQty(Math.max(floor, n));
              }}
              className="h-9 w-24"
            />
          </Field>
        ) : null}
      </div>

      {plans.isError ? (
        <p className="mx-auto mt-8 max-w-lg text-center text-sm text-[var(--trim-muted)]">
          {(plans.error as Error)?.message || settings?.plans_load_failed_message || ""}
        </p>
      ) : null}

      <div className="mx-auto mt-10 grid w-full max-w-none gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {plans.isPending && !plans.data
          ? [0, 1, 2, 3].map((i) => (
              <LandingPricingCardSkeleton
                key={`price-sk-${i}`}
                /* Catalog sort: Free, Pro, Team, Enterprise - Free+Enterprise unlimited, Pro popular. */
                showUnlimitedBadge={i === 0 || i === 3}
                showPopularBadge={i === 1}
              />
            ))
          : list.map((plan) => {
              let price = "";
              try {
                if (plan.effective_cents != null && moneyLocale) {
                  price = formatMoney(plan.effective_cents, plan.currency_code, moneyLocale);
                }
              } catch {
                price = "";
              }
              const period =
                plan.effective_interval === "annual"
                  ? settings?.period_year_label
                  : settings?.period_month_label;
              const isPopular = Boolean(plan.popular) && Boolean(popularBadge);
              const showUnlimited = Boolean(plan.unlimited) && Boolean(unlimitedFeatureLabel);
              const features = (plan.features || []).filter((f) => {
                if (!showUnlimited || !unlimitedFeatureLabel) return true;
                return f.trim().toLowerCase() !== unlimitedFeatureLabel.toLowerCase();
              });
              // Signed-in catalog rows carry backend CTA state (e.g. Free → Included /
              // can_proceed false). Prefer that over the marketing "Choose plan" copy.
              const ctaLabel =
                (signedIn && plan.change_label?.trim()) ||
                site.pricing_cta ||
                plan.change_label ||
                "";
              const canProceed = plan.can_proceed !== false;
              const actionHint = plan.change_reason?.trim() || "";
              const checkoutBusy = checkoutOpen && intentPlanId === plan.id;
              const perSeatLine = plan.per_seat
                ? seatQty === 1
                  ? (settings?.per_seat_one_fmt || "").replace("%s", settings?.meta_sep || "")
                  : (settings?.per_seat_many_fmt || "")
                      .replace("%s", settings?.meta_sep || "")
                      .replace("%d", String(seatQty > 0 ? seatQty : ""))
                : "";

              return (
                <article
                  key={plan.id}
                  className={cn(
                    "relative flex flex-col rounded-2xl border bg-[var(--trim-panel)] p-6 pt-7 shadow-[var(--trim-card-shadow)] transition duration-300",
                    isPopular
                      ? "border-[var(--trim-ink)] ring-1 ring-[var(--trim-ink)]/20 lg:-translate-y-1 lg:shadow-[var(--trim-float-shadow)]"
                      : "border-[var(--trim-border)] hover:border-[var(--trim-ink)] hover:ring-1 hover:ring-[var(--trim-ink)]/20",
                  )}
                >
                  {isPopular ? (
                    <span className="absolute -top-3 left-1/2 z-10 -translate-x-1/2">
                      <span className="inline-flex animate-[trim-badge-in_420ms_ease-out] items-center rounded-md bg-[var(--trim-ink)] px-3 py-1 text-[11px] font-semibold tracking-wide text-[var(--trim-ink-inverse)] shadow-[var(--trim-card-shadow)]">
                        {popularBadge}
                      </span>
                    </span>
                  ) : null}

                  <div className="flex min-h-[1.5rem] items-start justify-between gap-2">
                    <h3 className="text-lg font-semibold text-[var(--trim-fg)]">
                      {plan.display_name}
                    </h3>
                  </div>

                  {showUnlimited ? (
                    <span
                      className={cn(
                        "mt-3 inline-flex w-fit max-w-full animate-[trim-badge-in_520ms_ease-out] items-center gap-1.5 rounded-md border px-2.5 py-1 text-[11px] font-semibold tracking-wide",
                        statusSoftBgClass,
                        statusTextClass,
                        "border-[var(--trim-status,#27a644)]/35",
                      )}
                    >
                      <span
                        aria-hidden
                        className="h-1.5 w-1.5 shrink-0 rounded-full bg-[var(--trim-status,#27a644)] shadow-[0_0_0_3px_color-mix(in_srgb,var(--trim-status,#27a644)_22%,transparent)]"
                      />
                      <span className="truncate">{unlimitedFeatureLabel}</span>
                    </span>
                  ) : null}

                  <p className="mt-2 text-sm text-[var(--trim-muted)]">{plan.description}</p>
                  <div className="mt-6 flex items-baseline gap-2">
                    {price ? (
                      <>
                        <span className="font-display text-3xl font-semibold tracking-tight text-[var(--trim-fg)]">
                          {price}
                        </span>
                        {period ? (
                          <span className="text-sm text-[var(--trim-muted)]">{period}</span>
                        ) : null}
                      </>
                    ) : (
                      <span className="text-sm text-[var(--trim-muted)]">
                        {plan.change_label || ""}
                      </span>
                    )}
                  </div>
                  {billingInterval === "annual" && plan.savings_label ? (
                    <p className={cn("mt-2 text-xs font-medium", statusTextClass)}>
                      {plan.savings_label}
                    </p>
                  ) : null}
                  {perSeatLine ? (
                    <p className="mt-2 text-xs text-[var(--trim-muted)]">{perSeatLine}</p>
                  ) : null}
                  <ul className="mt-6 mb-0 flex-1 space-y-2">
                    {features.map((f) => (
                      <li
                        key={f}
                        className="flex items-start gap-2 text-sm text-[var(--trim-muted)]"
                      >
                        <Check className="mt-0.5 h-4 w-4 shrink-0 text-[var(--trim-fg)]" />
                        <span>{f}</span>
                      </li>
                    ))}
                  </ul>
                  {actionHint ? (
                    <p className="mt-3 text-xs text-[var(--trim-muted)]">{actionHint}</p>
                  ) : null}
                  {signedIn && accessToken && userId ? (
                    <Button
                      type="button"
                      className="mt-8 w-full"
                      variant={!canProceed ? "outline" : isPopular ? "default" : "secondary"}
                      // Modal open ≠ network pending. Real spinners live inside PlanModal
                      // (checkout / "Sending inquiry…"); avoid reusing plan.pending_label here.
                      disabled={!ready || checkoutBusy || !canProceed}
                      title={actionHint || undefined}
                      onClick={() => {
                        if (!canProceed) return;
                        startPlanCheckout(plan.id);
                      }}
                    >
                      {ctaLabel}
                    </Button>
                  ) : loginHref ? (
                    <Button
                      asChild
                      className="mt-8 w-full"
                      variant={isPopular ? "default" : "secondary"}
                      disabled={!ready}
                    >
                      <Link href={loginHref}>{ctaLabel}</Link>
                    </Button>
                  ) : (
                    <Button
                      type="button"
                      className="mt-8 w-full"
                      variant={isPopular ? "default" : "secondary"}
                      disabled
                    >
                      {ctaLabel}
                    </Button>
                  )}
                </article>
              );
            })}
      </div>

      {/* Direct checkout / enterprise inquiry for the clicked plan - no second plan picker. */}
      {signedIn && accessToken && userId ? (
        <PlanModal
          open={checkoutOpen}
          onOpenChange={(next) => {
            setCheckoutOpen(next);
            if (!next) setIntentPlanId(null);
          }}
          accessToken={accessToken}
          userId={userId}
          email={profile.email}
          intentPlanId={intentPlanId}
          intentSeatQuantity={seatQty >= 1 ? seatQty : null}
          hideCatalog
        />
      ) : null}
    </section>
  );
}
