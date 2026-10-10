import { CheckoutSuccessClient } from "@/components/billing/checkout-success-client";
import type { Metadata } from "next";

export const metadata: Metadata = {
  robots: { index: false, follow: false },
};

/** Paddle settings.successUrl target after overlay checkout completes. */
export default function BillingCheckoutSuccessPage() {
  return <CheckoutSuccessClient />;
}
