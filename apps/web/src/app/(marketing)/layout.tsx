"use client";

import { LandingShell } from "@/components/landing/landing-shell";

export default function MarketingLayout({ children }: { children: React.ReactNode }) {
  return <LandingShell>{children}</LandingShell>;
}
