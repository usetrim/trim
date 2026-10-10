import { create } from "zustand";

export type BillingInterval = "monthly" | "annual";
export type PlanModalMode = "plans" | "topup";

type BillingUIState = {
  /** null until hydrated from billing_settings.default_plan_interval (no invent annual). */
  interval: BillingInterval | null;
  setInterval: (interval: BillingInterval) => void;
  planModalOpen: boolean;
  setPlanModalOpen: (open: boolean) => void;
  /** Catalog mode: subscription plans vs credit top-up packs. */
  planModalMode: PlanModalMode;
  openPlanModal: (opts?: { mode?: PlanModalMode }) => void;
};

export const useBillingUI = create<BillingUIState>((set) => ({
  interval: null,
  setInterval: (interval) => set({ interval }),
  planModalOpen: false,
  setPlanModalOpen: (planModalOpen) => set({ planModalOpen }),
  planModalMode: "plans",
  openPlanModal: (opts) =>
    set({
      planModalOpen: true,
      planModalMode: opts?.mode === "topup" ? "topup" : "plans",
    }),
}));
