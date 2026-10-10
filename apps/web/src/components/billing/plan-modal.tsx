"use client";

import { PlanGridSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FetchProgressBar } from "@/components/ui/fetch-progress";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { useCheckoutSession, useEnterpriseInquiry } from "@/hooks/mutations/billing";
import { useAuthProviders } from "@/hooks/queries/auth";
import { useSubscriptionStatus } from "@/hooks/queries/billing";
import { usePlans } from "@/hooks/queries/plans";
import { apiFetch } from "@/lib/api/client";
import { formatDateTimeShort } from "@/lib/format-datetime";
import { ensurePaddleSuccessNavigation, openTrimPaddleCheckout } from "@/lib/paddle-checkout";
import { invalidateBilling, qk, refreshBillingUntilSettled } from "@/lib/query-keys";
import { statusSoftBgClass, statusTextClass } from "@/lib/status-color";
import { cn, formatMoney } from "@/lib/utils";
import { type BillingInterval, useBillingUI } from "@/stores/billing-ui";
import type { CheckoutSession, PlansResponse } from "@/types/billing";
import { type Paddle, initializePaddle } from "@paddle/paddle-js";
import { useQueryClient } from "@tanstack/react-query";
import { Check } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  accessToken?: string;
  userId?: string;
  email?: string;
  /**
   * Pricing page: check out / inquire for this plan immediately.
   * When set with hideCatalog, the plan-grid dialog is not shown.
   */
  intentPlanId?: string | null;
  /** Hide the catalog grid (pricing already showed the cards). */
  hideCatalog?: boolean;
  /** Seat quantity chosen on the pricing page (per-seat Team / Enterprise). */
  intentSeatQuantity?: number | null;
};

function formatAnnualSaveHint(fmt: string, percent: number): string {
  const f = fmt.trim();
  if (!f || percent < 1) return "";
  if (!f.includes("%d")) return "";
  // printf-style: %d → percent, %% → literal %
  return f.replace(/%d/g, String(percent)).replace(/%%/g, "%");
}

function countCatalogCards(plans: PlansResponse["plans"] | undefined, topup: boolean): number {
  if (!plans?.length) return 0;
  return plans.filter((p) => (topup ? p.plan_kind === "topup" : p.plan_kind !== "topup")).length;
}

/** Skeleton card count from chrome + warm plans cache (same filters as the live grid). */
function resolvePlanSkeletonCount(opts: {
  topupMode: boolean;
  loadedPlans?: PlansResponse["plans"];
  planTopupCardCount?: number;
  planSubscriptionCardCount?: number;
  planCardCount?: number;
  cachedPlans: Array<PlansResponse["plans"] | undefined>;
}): number {
  const fromLoaded = countCatalogCards(opts.loadedPlans, opts.topupMode);
  if (fromLoaded > 0) return fromLoaded;

  let fromCache = 0;
  for (const plans of opts.cachedPlans) {
    fromCache = Math.max(fromCache, countCatalogCards(plans, opts.topupMode));
  }
  if (fromCache > 0) return fromCache;

  if (opts.topupMode) {
    if (typeof opts.planTopupCardCount === "number" && opts.planTopupCardCount > 0) {
      return opts.planTopupCardCount;
    }
    if (
      typeof opts.planCardCount === "number" &&
      typeof opts.planSubscriptionCardCount === "number" &&
      opts.planCardCount > opts.planSubscriptionCardCount
    ) {
      return opts.planCardCount - opts.planSubscriptionCardCount;
    }
    return 0;
  }

  if (typeof opts.planSubscriptionCardCount === "number" && opts.planSubscriptionCardCount > 0) {
    return opts.planSubscriptionCardCount;
  }
  if (typeof opts.planCardCount === "number" && opts.planCardCount > 0) {
    return opts.planCardCount;
  }
  return 0;
}

function planActionLabel(opts: {
  changeCode?: string;
  changeReason?: string;
  changeLabel?: string;
  canProceed?: boolean | null;
  upgradeEligible?: boolean | null;
}): { label: string; disabled: boolean; hint?: string } {
  // Labels and eligibility come only from the API. No client inventing CTAs.
  const canProceed =
    typeof opts.canProceed === "boolean" ? opts.canProceed : opts.upgradeEligible === true;

  if (opts.changeLabel) {
    return {
      label: opts.changeLabel,
      disabled: !canProceed,
      hint: opts.changeReason,
    };
  }

  // Fail closed: never invent a CTA when the API omitted change_label.
  return {
    label: "",
    disabled: true,
    hint: opts.changeReason,
  };
}

/** Paddle preview amounts are major-unit strings (e.g. "12.34"). formatMoney expects cents. */
function formatPaddleAmount(
  amount: string | undefined,
  currency: string | undefined,
  locale: string,
) {
  if (!amount) return null;
  const code = (currency || "").trim();
  if (!code) return amount;
  const n = Number(amount);
  if (Number.isNaN(n)) return amount;
  // Decimal string = major units; integer string = already minor units (cents).
  const cents = amount.includes(".") ? Math.round(n * 100) : Math.round(n);
  return formatMoney(cents, code, locale);
}

export function PlanModal({
  open,
  onOpenChange,
  accessToken,
  userId,
  email,
  intentPlanId = null,
  hideCatalog = false,
  intentSeatQuantity = null,
}: Props) {
  const { interval, setInterval, planModalMode } = useBillingUI();
  const topupMode = planModalMode === "topup";
  const { data, isLoading, isFetching, error } = usePlans(interval, accessToken);
  const siteChrome = useAuthProviders();
  const qc = useQueryClient();
  const planCardCount = useMemo(() => {
    const cachedPlans = qc
      .getQueriesData<PlansResponse>({ queryKey: qk.plans })
      .map(([, row]) => row?.plans);
    return resolvePlanSkeletonCount({
      topupMode,
      loadedPlans: data?.plans,
      planTopupCardCount: siteChrome.data?.plan_topup_card_count,
      planSubscriptionCardCount: siteChrome.data?.plan_subscription_card_count,
      planCardCount: siteChrome.data?.plan_card_count,
      cachedPlans,
    });
  }, [topupMode, data?.plans, siteChrome.data, qc]);
  const moneyLocale = data?.money_locale?.trim() || "";
  const sub = useSubscriptionStatus(accessToken);
  const checkout = useCheckoutSession(accessToken);
  const enterpriseInquiry = useEnterpriseInquiry(accessToken);
  const [paddle, setPaddle] = useState<Paddle | null>(null);
  const [seatQty, setSeatQty] = useState(0);
  const [pendingUpgrade, setPendingUpgrade] = useState<CheckoutSession | null>(null);
  const [checkoutPlanId, setCheckoutPlanId] = useState<string | null>(null);
  const [enterpriseOpen, setEnterpriseOpen] = useState(false);
  const [companyName, setCompanyName] = useState("");
  const [estimatedSeats, setEstimatedSeats] = useState("");
  const [enterpriseMessage, setEnterpriseMessage] = useState("");
  const topupSectionRef = useRef<HTMLDivElement | null>(null);
  const intentStartedRef = useRef<string | null>(null);
  const settings = data?.settings;
  const popularBadge = (siteChrome.data?.site?.pricing_popular_badge || "").trim();
  const unlimitedFeatureLabel = (settings?.unlimited_feature_label || "").trim();
  const showCatalog = open && !hideCatalog;

  useEffect(() => {
    const fromSettings = data?.settings.default_plan_interval;
    const fromResponse = data?.interval;
    const next =
      fromSettings === "monthly" || fromSettings === "annual"
        ? fromSettings
        : fromResponse === "monthly" || fromResponse === "annual"
          ? fromResponse
          : null;
    if (interval == null && next) {
      setInterval(next);
    }
  }, [data?.settings.default_plan_interval, data?.interval, interval, setInterval]);

  useEffect(() => {
    if (!open) return;
    if (typeof intentSeatQuantity === "number" && intentSeatQuantity >= 1) {
      setSeatQty(intentSeatQuantity);
    }
  }, [open, intentSeatQuantity]);

  useEffect(() => {
    const n = data?.settings.default_seat_quantity;
    if (typeof n === "number" && n >= 1 && seatQty < 1) {
      setSeatQty(n);
    }
  }, [data?.settings.default_seat_quantity, seatQty]);

  // Warm plans as soon as the dialog opens so top-up/upgrade shimmers have catalog counts.
  useEffect(() => {
    if (!open) return;
    const intervals: Array<BillingInterval | null> =
      interval == null ? [null, "monthly", "annual"] : [interval];
    for (const iv of intervals) {
      void qc.prefetchQuery({
        queryKey: [...qk.plans, iv ?? "default", accessToken ? "auth" : "anon"],
        queryFn: () => {
          const q = iv != null ? `?interval=${encodeURIComponent(iv)}` : "";
          return apiFetch<PlansResponse>(`/api/v1/public/plans${q}`, { token: accessToken });
        },
        staleTime: 5 * 60_000,
      });
    }
  }, [open, interval, accessToken, qc]);

  const catalogSaveLabel = useMemo(() => {
    if (topupMode) return null;
    // Catalog cents math only (plan.savings_label). Never invent from settings %.
    const fromPlans = (data?.plans ?? [])
      .filter((p) => p.plan_kind !== "topup")
      .map((p) => p.savings_label)
      .filter((s): s is string => Boolean(s?.trim()));
    return fromPlans.length > 0 ? fromPlans[0].trim() : null;
  }, [data?.plans, topupMode]);

  const annualBillingLabel = (siteChrome.data?.site?.pricing_annual_billing || "").trim();
  const annualSaveHint =
    formatAnnualSaveHint(
      siteChrome.data?.site?.pricing_annual_save_hint_fmt || "",
      settings?.annual_discount_percent ?? 0,
    ) ||
    catalogSaveLabel ||
    "";
  const monthlyToggleLabel = (settings?.monthly_toggle_label || "").trim();
  const annualToggleLabel = (settings?.annual_toggle_label || "").trim();
  const switchAria =
    (interval === "annual" ? annualToggleLabel || annualBillingLabel : monthlyToggleLabel) ||
    annualBillingLabel ||
    undefined;

  const visiblePlans = useMemo(() => {
    const all = data?.plans ?? [];
    if (topupMode) {
      return all.filter((p) => p.plan_kind === "topup");
    }
    return all.filter((p) => p.plan_kind !== "topup");
  }, [data?.plans, topupMode]);

  const showSeatField = useMemo(() => {
    if (topupMode) return false;
    return visiblePlans.some((p) => p.per_seat);
  }, [topupMode, visiblePlans]);

  const dialogTitle = topupMode
    ? (settings?.topup_dialog_title || "").trim()
    : (settings?.plans_dialog_title || "").trim();

  const dialogDescription = useMemo(() => {
    if (topupMode) {
      return (settings?.topup_dialog_description || "").trim();
    }
    const parts: string[] = [];
    const base = (settings?.plans_dialog_description || "").trim();
    if (base) parts.push(base);
    if (interval === "annual" && catalogSaveLabel) {
      parts.push(catalogSaveLabel.endsWith(".") ? catalogSaveLabel : `${catalogSaveLabel}.`);
    }
    if (settings?.allow_downgrades === false) {
      const note = (settings.plans_dialog_upgrades_only_note || "").trim();
      if (note) parts.push(note);
    }
    return parts.join(" ");
  }, [
    topupMode,
    interval,
    settings?.topup_dialog_description,
    settings?.plans_dialog_description,
    settings?.allow_downgrades,
    settings?.plans_dialog_upgrades_only_note,
    catalogSaveLabel,
  ]);

  useEffect(() => {
    if (!open || !topupMode || isLoading || visiblePlans.length === 0) return;
    const el = topupSectionRef.current;
    if (!el) return;
    const t = window.setTimeout(() => {
      el.scrollIntoView({ behavior: "smooth", block: "start" });
    }, 50);
    return () => window.clearTimeout(t);
  }, [open, topupMode, isLoading, visiblePlans.length]);

  const pendingCheckoutPlanId = useRef("");

  useEffect(() => {
    const token = process.env.NEXT_PUBLIC_PADDLE_CLIENT_TOKEN;
    const env = process.env.NEXT_PUBLIC_PADDLE_ENV;
    if (!token || !env) return;
    void initializePaddle({
      environment: env === "production" ? "production" : "sandbox",
      token,
      eventCallback: (event) => {
        if (event.name === "checkout.completed") {
          void refreshBillingUntilSettled(qc);
          // Belt-and-suspenders if settings.successUrl redirect is slow/blocked.
          ensurePaddleSuccessNavigation(pendingCheckoutPlanId.current);
          return;
        }
        if (event.name === "checkout.closed") {
          void invalidateBilling(qc);
        }
      },
    }).then((instance) => {
      if (instance) setPaddle(instance);
    });
  }, [qc]);

  async function openPaddleCheckout(session: CheckoutSession) {
    const settings = data?.settings;
    if (!paddle) {
      toast.error(session.message || settings?.paddle_js_not_ready_message || "");
      return;
    }
    if (!session.price_id || !email) {
      toast.error(session.message || settings?.checkout_price_missing_message || "");
      return;
    }

    pendingCheckoutPlanId.current = session.plan_id?.trim() || "";
    openTrimPaddleCheckout({
      paddle,
      priceId: session.price_id,
      quantity: session.quantity,
      email,
      customData: session.custom_data ?? {},
      discountId: session.discount_id,
      discountCode: session.discount_code,
      planId: session.plan_id,
    });
    onOpenChange(false);
  }

  async function handleSelect(planId: string, planKind: string, perSeat: boolean) {
    const settings = data?.settings;
    if (planKind === "enterprise") {
      if (!accessToken || !userId || !email) {
        if (settings?.auth_required_enterprise_message) {
          toast.error(settings.auth_required_enterprise_message);
        }
        if (hideCatalog) onOpenChange(false);
        return;
      }
      setEnterpriseOpen(true);
      return;
    }
    if (!accessToken || !userId || !email) {
      if (settings?.auth_required_plan_message) {
        toast.error(settings.auth_required_plan_message);
      }
      if (hideCatalog) onOpenChange(false);
      return;
    }

    const quantity = perSeat
      ? seatQty >= 1
        ? seatQty
        : settings?.default_seat_quantity && settings.default_seat_quantity >= 1
          ? settings.default_seat_quantity
          : 0
      : settings?.default_checkout_quantity ?? 0;
    if (quantity < 1) {
      toast.error(settings?.seat_quantity_hint || "");
      if (hideCatalog) onOpenChange(false);
      return;
    }
    const checkoutInterval = planKind === "topup" || topupMode ? "monthly" : interval;
    if (!checkoutInterval) {
      // Interval hydrates from billing_settings.default_plan_interval; never
      // invent monthly or annual when it has not arrived yet.
      toast.error(settings?.checkout_unavailable_message || "");
      if (hideCatalog) onOpenChange(false);
      return;
    }

    try {
      setCheckoutPlanId(planId);
      const session = await checkout.mutateAsync({
        plan_id: planId,
        interval: checkoutInterval,
        quantity,
        user_id: userId,
        email,
        confirm: false,
      });

      if (session.action === "upgrade_preview" || session.code === "CONFIRM_REQUIRED") {
        setPendingUpgrade(session);
        return;
      }

      if (session.action === "upgrade") {
        if (!session.message) {
          if (settings?.upgrade_missing_api_message) {
            toast.error(settings.upgrade_missing_api_message);
          }
          if (hideCatalog) onOpenChange(false);
          return;
        }
        toast.success(session.message);
        onOpenChange(false);
        return;
      }

      if (
        session.action === "blocked" ||
        session.action === "same_plan" ||
        session.action === "same"
      ) {
        toast.error(session.message || settings?.change_blocked_default_message || "");
        if (hideCatalog) onOpenChange(false);
        return;
      }

      if (session.action !== "new_checkout" || !session.price_id) {
        toast.error(session.message || settings?.checkout_unavailable_message || "");
        if (hideCatalog) onOpenChange(false);
        return;
      }

      await openPaddleCheckout(session);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : settings?.checkout_request_failed_message || "");
      if (hideCatalog) onOpenChange(false);
    } finally {
      setCheckoutPlanId(null);
    }
  }

  // Pricing cards: jump straight to checkout / enterprise inquiry for the clicked plan.
  // biome-ignore lint/correctness/useExhaustiveDependencies: one-shot intent when plans catalog is ready
  useEffect(() => {
    if (!open || !hideCatalog || !intentPlanId || !data) return;
    if (intentStartedRef.current === intentPlanId) return;
    const plan = (data.plans ?? []).find((p) => p.id === intentPlanId);
    if (!plan) {
      toast.error(settings?.checkout_unavailable_message || "");
      onOpenChange(false);
      return;
    }
    // Wait for seat default hydration before per-seat checkout (Team / Enterprise sales path).
    if (
      plan.per_seat &&
      plan.plan_kind !== "enterprise" &&
      seatQty < 1 &&
      !(data.settings?.default_seat_quantity && data.settings.default_seat_quantity >= 1)
    ) {
      return;
    }
    intentStartedRef.current = intentPlanId;
    void handleSelect(plan.id, plan.plan_kind, plan.per_seat);
  }, [open, hideCatalog, intentPlanId, data, seatQty]);

  useEffect(() => {
    if (open) return;
    intentStartedRef.current = null;
    setEnterpriseOpen(false);
    setPendingUpgrade(null);
  }, [open]);

  async function confirmUpgrade() {
    if (!pendingUpgrade || !accessToken || !userId || !email) return;
    const settings = data?.settings;
    try {
      setCheckoutPlanId(pendingUpgrade.plan_id);
      const session = await checkout.mutateAsync({
        plan_id: pendingUpgrade.plan_id,
        interval: pendingUpgrade.interval,
        quantity: pendingUpgrade.quantity,
        user_id: userId,
        email,
        confirm: true,
      });
      setPendingUpgrade(null);
      if (session.action === "upgrade") {
        const preview = session.proration_preview;
        const dueAmount = formatPaddleAmount(
          preview?.grand_total || preview?.balance,
          preview?.currency_code || session.currency_code,
          moneyLocale,
        );
        const duePrefix = settings?.upgrade_due_prefix?.trim() || "";
        // Backend message only; optional due amount appended when both chrome pieces exist.
        const due =
          dueAmount && duePrefix ? `${settings?.meta_sep || ""}${duePrefix} ${dueAmount}` : "";
        if (!session.message) {
          if (settings?.upgrade_missing_api_message) {
            toast.error(settings.upgrade_missing_api_message);
          }
          return;
        }
        toast.success(session.message + due);
        onOpenChange(false);
        return;
      }
      toast.error(session.message || settings?.upgrade_not_applied_message || "");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : settings?.upgrade_request_failed_message || "");
    } finally {
      setCheckoutPlanId(null);
    }
  }

  async function submitEnterpriseInquiry() {
    const settings = data?.settings;
    const message = enterpriseMessage.trim();
    if (!message) {
      if (settings?.enterprise_message_required) {
        toast.error(settings.enterprise_message_required);
      }
      return;
    }
    const seats = Number.parseInt(estimatedSeats, 10);
    try {
      const res = await enterpriseInquiry.mutateAsync({
        company_name: companyName.trim() || undefined,
        estimated_seats: Number.isFinite(seats) && seats > 0 ? seats : undefined,
        message,
      });
      if (!res.message) {
        if (settings?.inquiry_missing_confirm_message) {
          toast.error(settings.inquiry_missing_confirm_message);
        }
        return;
      }
      toast.success(res.message);
      setEnterpriseOpen(false);
      setCompanyName("");
      setEstimatedSeats("");
      setEnterpriseMessage("");
      onOpenChange(false);
    } catch (e) {
      toast.error(
        e instanceof Error ? e.message : settings?.enterprise_inquiry_failed_message || "",
      );
    }
  }

  const preview = pendingUpgrade?.proration_preview;

  return (
    <>
      {showCatalog ? (
        <Dialog open={showCatalog} onOpenChange={onOpenChange}>
          <DialogContent
            className="w-full max-w-[calc(100vw-1.5rem)] sm:max-w-5xl"
            closeLabel={settings?.dialog_close_label}
          >
            <DialogHeader>
              <DialogTitle>{dialogTitle}</DialogTitle>
              {dialogDescription ? (
                <DialogDescription>{dialogDescription}</DialogDescription>
              ) : null}
            </DialogHeader>

            <FetchProgressBar active={isFetching && !isLoading} className="mb-2 w-full" />

            {!topupMode && sub.data?.has_active_paid ? (
              <p className="rounded border border-[var(--trim-border)] bg-[var(--trim-bg)] px-3 py-2 text-sm text-[var(--trim-muted)]">
                {settings?.active_plan_prefix}{" "}
                <span className="font-medium text-[var(--trim-fg)]">
                  {sub.data.plan_tier_label || ""}
                </span>
                {sub.data.billing_interval_label ? ` (${sub.data.billing_interval_label})` : ""}
                {(formatDateTimeShort(sub.data.expires_at, moneyLocale) ||
                  sub.data.expires_at_label) &&
                settings?.active_plan_expires_fmt
                  ? settings.active_plan_expires_fmt.replace(
                      "%s",
                      formatDateTimeShort(sub.data.expires_at, moneyLocale) ||
                        sub.data.expires_at_label ||
                        "",
                    )
                  : ""}
                .
              </p>
            ) : null}

            {isLoading && !data ? (
              <PlanGridSkeleton
                /* Top-up chrome count can lag on cold open; never flash an empty grid. */
                count={planCardCount > 0 ? planCardCount : topupMode ? 1 : 0}
                mode={topupMode ? "topup" : "upgrade"}
              />
            ) : null}

            {!topupMode && (!isLoading || data) ? (
              annualBillingLabel ? (
                <div className="flex flex-col items-center py-2">
                  <div className="inline-flex items-center gap-3 rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] px-3 py-2 shadow-[var(--trim-card-shadow)]">
                    <div className="min-w-0 text-left">
                      <p
                        className={cn(
                          "text-[11px] font-medium transition-colors sm:text-sm",
                          interval === "annual"
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
                            interval === "annual" ? "opacity-100" : "opacity-70",
                          )}
                        >
                          {annualSaveHint}
                        </p>
                      ) : null}
                    </div>
                    <Switch
                      checked={interval === "annual"}
                      onCheckedChange={(checked) => setInterval(checked ? "annual" : "monthly")}
                      aria-label={switchAria}
                    />
                  </div>
                </div>
              ) : monthlyToggleLabel || annualToggleLabel ? (
                <div className="flex flex-col items-center gap-2 py-2">
                  <div className="inline-flex rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] p-1 shadow-[var(--trim-card-shadow)]">
                    {monthlyToggleLabel ? (
                      <button
                        type="button"
                        className={cn(
                          "rounded-md px-4 py-1.5 text-sm transition",
                          interval === "monthly"
                            ? "bg-[var(--trim-ink)] text-[var(--trim-ink-inverse)]"
                            : "text-[var(--trim-muted)] hover:bg-[var(--trim-hover)] hover:text-[var(--trim-fg)]",
                        )}
                        onClick={() => setInterval("monthly")}
                      >
                        {monthlyToggleLabel}
                      </button>
                    ) : null}
                    {annualToggleLabel ? (
                      <button
                        type="button"
                        className={cn(
                          "rounded-md px-4 py-1.5 text-sm transition",
                          interval === "annual"
                            ? "bg-[var(--trim-ink)] text-[var(--trim-ink-inverse)]"
                            : "text-[var(--trim-muted)] hover:bg-[var(--trim-hover)] hover:text-[var(--trim-fg)]",
                        )}
                        onClick={() => setInterval("annual")}
                      >
                        {annualToggleLabel}
                      </button>
                    ) : null}
                  </div>
                  {annualSaveHint ? (
                    <p
                      className={cn(
                        "text-[11px] font-medium transition-opacity sm:text-xs",
                        statusTextClass,
                        interval === "annual" ? "opacity-100" : "opacity-70",
                      )}
                    >
                      {annualSaveHint}
                    </p>
                  ) : null}
                </div>
              ) : null
            ) : null}

            {!topupMode && showSeatField && (!isLoading || data) ? (
              settings?.seat_quantity_label ? (
                <Field
                  id="seat-qty"
                  label={settings.seat_quantity_label}
                  description={settings.seat_quantity_hint}
                  className="mx-auto max-w-xs"
                >
                  <Input
                    id="seat-qty"
                    type="number"
                    min={
                      settings?.min_seat_quantity && settings.min_seat_quantity >= 1
                        ? settings.min_seat_quantity
                        : undefined
                    }
                    max={
                      settings?.max_seat_quantity && settings.max_seat_quantity > 0
                        ? settings.max_seat_quantity
                        : undefined
                    }
                    value={seatQty}
                    placeholder={settings.seat_quantity_label}
                    onChange={(e) => {
                      const floor =
                        settings?.min_seat_quantity && settings.min_seat_quantity >= 1
                          ? settings.min_seat_quantity
                          : 0;
                      const n = Number(e.target.value);
                      if (!Number.isFinite(n) || floor < 1) return;
                      setSeatQty(Math.max(floor, n));
                    }}
                    className="h-9 w-24"
                  />
                </Field>
              ) : null
            ) : null}
            {error ? (
              <p className="text-sm text-destructive">
                {(error instanceof Error && error.message) ||
                  settings?.plans_load_failed_message ||
                  ""}
              </p>
            ) : null}

            {!isLoading || data ? (
              visiblePlans.length === 0 ? (
                <p className="py-8 text-center text-sm text-[var(--trim-muted)]">
                  {topupMode
                    ? (settings?.topup_dialog_empty || "").trim()
                    : (settings?.plans_load_failed_message || "").trim()}
                </p>
              ) : (
                <div className="grid gap-4 pt-3 md:grid-cols-2 xl:grid-cols-4">
                  {visiblePlans.map((plan, idx) => {
                    const price =
                      plan.effective_cents == null
                        ? ""
                        : plan.currency_code && moneyLocale
                          ? formatMoney(plan.effective_cents, plan.currency_code, moneyLocale)
                          : "";
                    const period =
                      plan.effective_cents == null
                        ? ""
                        : topupMode || interval === "monthly"
                          ? settings?.period_month_label || ""
                          : settings?.period_year_label || "";
                    const action = planActionLabel({
                      changeCode: plan.change_code,
                      changeReason: plan.change_reason,
                      changeLabel: plan.change_label,
                      canProceed: plan.can_proceed,
                      upgradeEligible: plan.upgrade_eligible,
                    });
                    const isFirstTopup = topupMode && idx === 0;
                    const isPopular = !topupMode && Boolean(plan.popular) && Boolean(popularBadge);
                    const showUnlimited =
                      !topupMode && Boolean(plan.unlimited) && Boolean(unlimitedFeatureLabel);
                    const features = (Array.isArray(plan.features) ? plan.features : []).filter(
                      (f) => {
                        if (!showUnlimited || !unlimitedFeatureLabel) return true;
                        return f.trim().toLowerCase() !== unlimitedFeatureLabel.toLowerCase();
                      },
                    );

                    return (
                      <article
                        key={plan.id}
                        ref={isFirstTopup ? topupSectionRef : undefined}
                        data-plan-kind={plan.plan_kind}
                        className={cn(
                          "relative flex flex-col rounded-2xl border bg-[var(--trim-panel)] p-6 pt-7 shadow-[var(--trim-card-shadow)] transition duration-300",
                          isPopular
                            ? "border-[var(--trim-ink)] ring-1 ring-[var(--trim-ink)]/20 -translate-y-0.5 shadow-[var(--trim-float-shadow)]"
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
                        {!topupMode && interval === "annual" && plan.savings_label ? (
                          <p className={cn("mt-2 text-xs font-medium", statusTextClass)}>
                            {plan.savings_label}
                          </p>
                        ) : null}
                        {plan.per_seat ? (
                          <p className="mt-2 text-xs text-[var(--trim-muted)]">
                            {seatQty === 1
                              ? (settings?.per_seat_one_fmt || "").replace(
                                  "%s",
                                  settings?.meta_sep || "",
                                )
                              : (settings?.per_seat_many_fmt || "")
                                  .replace("%s", settings?.meta_sep || "")
                                  .replace("%d", String(seatQty))}
                          </p>
                        ) : null}

                        <ul className="mt-6 mb-0 flex-1 space-y-2">
                          {features.map((feature) => (
                            <li
                              key={feature}
                              className="flex items-start gap-2 text-sm text-[var(--trim-muted)]"
                            >
                              <Check className="mt-0.5 h-4 w-4 shrink-0 text-[var(--trim-fg)]" />
                              <span>{feature}</span>
                            </li>
                          ))}
                        </ul>

                        {action.hint ? (
                          <p className="mt-3 text-xs text-[var(--trim-muted)]">{action.hint}</p>
                        ) : null}
                        <Button
                          className="mt-8 w-full"
                          variant={
                            action.disabled ? "outline" : isPopular ? "default" : "secondary"
                          }
                          disabled={
                            action.disabled || (plan.plan_kind === "enterprise" && enterpriseOpen)
                          }
                          // Enterprise only opens the inquiry form - "Sending inquiry…"
                          // belongs on the form submit button, not this card CTA.
                          isLoading={
                            plan.plan_kind !== "enterprise" &&
                            checkout.isPending &&
                            checkoutPlanId === plan.id
                          }
                          pendingLabel={
                            plan.plan_kind === "enterprise" ? undefined : plan.pending_label
                          }
                          onClick={() => void handleSelect(plan.id, plan.plan_kind, plan.per_seat)}
                        >
                          {action.label}
                        </Button>
                      </article>
                    );
                  })}
                </div>
              )
            ) : null}
          </DialogContent>
        </Dialog>
      ) : null}

      <Dialog
        open={Boolean(pendingUpgrade)}
        onOpenChange={(next) => {
          if (!next) {
            setPendingUpgrade(null);
            if (hideCatalog) onOpenChange(false);
          }
        }}
      >
        <DialogContent className="max-w-lg" closeLabel={settings?.dialog_close_label}>
          <DialogHeader>
            <DialogTitle>{settings?.upgrade_confirm_title}</DialogTitle>
            <DialogDescription>
              {(pendingUpgrade?.message || "").trim() ||
                (settings?.proration_review_hint || "").trim() ||
                (settings?.upgrade_proration_mode_fmt || "").trim().replace("%s", "").trim()}
            </DialogDescription>
          </DialogHeader>
          {pendingUpgrade ? (
            <div className="space-y-2 rounded border border-[var(--trim-border)] bg-[var(--trim-bg)] p-4 text-sm text-[var(--trim-muted)]">
              <p>
                {settings?.upgrade_plan_prefix}{" "}
                <span className="font-medium text-[var(--trim-fg)]">
                  {pendingUpgrade.plan_display_name}
                </span>{" "}
                ({pendingUpgrade.interval_label || pendingUpgrade.interval})
                {pendingUpgrade.quantity > 1 && settings?.upgrade_seats_fmt
                  ? `${settings.meta_sep || ""}${settings.upgrade_seats_fmt.replace(
                      "%d",
                      String(pendingUpgrade.quantity),
                    )}`
                  : ""}
              </p>
              {preview?.credit ? (
                <p>
                  {settings?.upgrade_credit_prefix}{" "}
                  {formatPaddleAmount(preview.credit, preview.currency_code, moneyLocale)}
                </p>
              ) : null}
              {preview?.subtotal ? (
                <p>
                  {settings?.upgrade_subtotal_prefix}{" "}
                  {formatPaddleAmount(preview.subtotal, preview.currency_code, moneyLocale)}
                </p>
              ) : null}
              {preview?.tax ? (
                <p>
                  {settings?.upgrade_tax_prefix}{" "}
                  {formatPaddleAmount(preview.tax, preview.currency_code, moneyLocale)}
                </p>
              ) : null}
              {formatPaddleAmount(
                preview?.grand_total || preview?.balance,
                preview?.currency_code,
                moneyLocale,
              ) ? (
                <p className="pt-2 text-base font-semibold text-[var(--trim-fg)]">
                  {settings?.upgrade_due_prefix}{" "}
                  {formatPaddleAmount(
                    preview?.grand_total || preview?.balance,
                    preview?.currency_code,
                    moneyLocale,
                  )}
                </p>
              ) : (
                <p className="pt-2 text-sm text-[var(--trim-muted)]">
                  {settings?.upgrade_preview_empty}
                </p>
              )}
            </div>
          ) : null}
          <DialogFooter className="gap-2 sm:gap-0">
            <Button variant="outline" onClick={() => setPendingUpgrade(null)}>
              {settings?.upgrade_cancel_label}
            </Button>
            <Button
              isLoading={checkout.isPending}
              pendingLabel={pendingUpgrade?.pending_label || undefined}
              disabled={!pendingUpgrade?.action_label}
              onClick={() => void confirmUpgrade()}
            >
              {pendingUpgrade?.action_label}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={enterpriseOpen}
        onOpenChange={(next) => {
          setEnterpriseOpen(next);
          if (!next && hideCatalog) onOpenChange(false);
        }}
      >
        <DialogContent className="max-w-lg" closeLabel={settings?.dialog_close_label}>
          <DialogHeader>
            <DialogTitle>{settings?.enterprise_dialog_title}</DialogTitle>
            <DialogDescription>
              {settings?.enterprise_dialog_description} (
              {email || settings?.enterprise_email_fallback || ""})
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            {settings?.enterprise_company_label ? (
              <Field
                id="ent-company"
                label={settings.enterprise_company_label}
                description={settings.enterprise_company_description}
              >
                <Input
                  id="ent-company"
                  value={companyName}
                  onChange={(e) => setCompanyName(e.target.value)}
                  placeholder={
                    settings.enterprise_company_placeholder || settings.enterprise_company_label
                  }
                />
              </Field>
            ) : null}
            {settings?.enterprise_seats_label ? (
              <Field
                id="ent-seats"
                label={settings.enterprise_seats_label}
                description={settings.seat_quantity_hint}
              >
                <Input
                  id="ent-seats"
                  type="number"
                  min={
                    settings?.min_seat_quantity && settings.min_seat_quantity >= 1
                      ? settings.min_seat_quantity
                      : undefined
                  }
                  max={
                    settings?.max_seat_quantity && settings.max_seat_quantity > 0
                      ? settings.max_seat_quantity
                      : undefined
                  }
                  value={estimatedSeats}
                  onChange={(e) => setEstimatedSeats(e.target.value)}
                  placeholder={
                    settings.enterprise_seats_placeholder || settings.enterprise_seats_label
                  }
                />
              </Field>
            ) : null}
            {settings?.enterprise_message_label ? (
              <Field
                id="ent-message"
                label={settings.enterprise_message_label}
                description={settings.enterprise_message_description}
              >
                <Textarea
                  id="ent-message"
                  rows={
                    settings?.enterprise_message_rows && settings.enterprise_message_rows >= 2
                      ? settings.enterprise_message_rows
                      : undefined
                  }
                  value={enterpriseMessage}
                  onChange={(e) => setEnterpriseMessage(e.target.value)}
                  placeholder={
                    settings.enterprise_message_placeholder || settings.enterprise_message_label
                  }
                />
              </Field>
            ) : null}
          </div>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button variant="outline" onClick={() => setEnterpriseOpen(false)}>
              {settings?.enterprise_cancel_label}
            </Button>
            <Button
              isLoading={enterpriseInquiry.isPending}
              pendingLabel={
                settings?.enterprise_send_pending_label ||
                (data?.plans ?? []).find((p) => p.plan_kind === "enterprise")?.pending_label ||
                undefined
              }
              onClick={() => void submitEnterpriseInquiry()}
            >
              {settings?.enterprise_send_label}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
