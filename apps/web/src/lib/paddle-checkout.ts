import type { Paddle } from "@paddle/paddle-js";

/** Post-checkout landing (absolute URL required by Paddle settings.successUrl). */
export const PADDLE_CHECKOUT_SUCCESS_PATH = "/billing/success";

export function paddleCheckoutSuccessUrl(opts?: { planId?: string }): string {
  if (typeof window === "undefined") return "";
  const u = new URL(PADDLE_CHECKOUT_SUCCESS_PATH, window.location.origin);
  u.searchParams.set("checkout", "success");
  const planId = opts?.planId?.trim();
  if (planId) u.searchParams.set("plan", planId);
  return u.toString();
}

export type OpenTrimPaddleCheckoutArgs = {
  paddle: Paddle;
  priceId: string;
  quantity: number;
  email: string;
  customData?: Record<string, unknown>;
  discountId?: string | null;
  discountCode?: string | null;
  planId?: string;
};

/**
 * Open Paddle overlay checkout with a success redirect.
 * Without settings.successUrl, Paddle leaves the customer on its built-in
 * "transaction completed" overlay (no app navigation).
 * @see https://developer.paddle.com/build/checkout/handle-success-post-checkout
 */
export function openTrimPaddleCheckout(args: OpenTrimPaddleCheckoutArgs): void {
  const successUrl = paddleCheckoutSuccessUrl({ planId: args.planId });
  const openArgs = {
    settings: {
      displayMode: "overlay" as const,
      successUrl,
    },
    items: [{ priceId: args.priceId, quantity: args.quantity }],
    customer: { email: args.email },
    customData: args.customData ?? {},
  };
  if (args.discountId) {
    args.paddle.Checkout.open({ ...openArgs, discountId: args.discountId });
    return;
  }
  if (args.discountCode) {
    args.paddle.Checkout.open({ ...openArgs, discountCode: args.discountCode });
    return;
  }
  args.paddle.Checkout.open(openArgs);
}

/** After checkout.completed, ensure we leave the overlay host page if redirect stalls. */
export function ensurePaddleSuccessNavigation(planId?: string, delayMs = 2200): void {
  if (typeof window === "undefined") return;
  const target = paddleCheckoutSuccessUrl({ planId });
  if (!target) return;
  window.setTimeout(() => {
    const path = window.location.pathname;
    if (
      path === PADDLE_CHECKOUT_SUCCESS_PATH ||
      path.startsWith(`${PADDLE_CHECKOUT_SUCCESS_PATH}/`)
    ) {
      return;
    }
    if (new URLSearchParams(window.location.search).get("checkout") === "success") {
      return;
    }
    window.location.assign(target);
  }, delayMs);
}
