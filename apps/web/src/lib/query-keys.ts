import type { QueryClient } from "@tanstack/react-query";

/** Stable query-key roots. Prefer these in useQuery + invalidateQueries. */
export const qk = {
  authProviders: ["auth-providers"] as const,
  quota: ["quota"] as const,
  subscription: ["subscription"] as const,
  plans: ["plans"] as const,
  receipts: ["receipts"] as const,
  /** Detail key is ["receipt", id] - not under the receipts list prefix. */
  receipt: (id: string) => ["receipt", id] as const,
  receiptRoot: ["receipt"] as const,
  events: ["events"] as const,
  eventStats: ["event-stats"] as const,
  preferences: ["preferences"] as const,
  apiKeys: ["api-keys"] as const,
  apiKeyDevices: (keyId: string) => ["api-key-devices", keyId] as const,
  workspaces: ["workspaces"] as const,
  workspaceMembers: ["workspace-members"] as const,
  workspaceInvites: ["workspace-invites"] as const,
  invitePreview: ["workspace-invite-preview"] as const,
  notifications: ["notifications"] as const,
  notificationsUnread: ["notifications-unread"] as const,
  enterpriseInquiries: ["enterprise-inquiries"] as const,
} as const;

const refetchActive = { refetchType: "active" as const };

/** Billing / usage surfaces that must refresh together after checkout or Stripe sync. */
export async function invalidateBilling(qc: QueryClient) {
  await Promise.all([
    qc.invalidateQueries({ queryKey: qk.quota, ...refetchActive }),
    qc.invalidateQueries({ queryKey: qk.subscription, ...refetchActive }),
    qc.invalidateQueries({ queryKey: qk.plans, ...refetchActive }),
    qc.invalidateQueries({ queryKey: qk.receipts, ...refetchActive }),
    qc.invalidateQueries({ queryKey: qk.receiptRoot, ...refetchActive }),
    qc.invalidateQueries({ queryKey: qk.events, ...refetchActive }),
    qc.invalidateQueries({ queryKey: qk.eventStats, ...refetchActive }),
    qc.invalidateQueries({ queryKey: qk.authProviders, ...refetchActive }),
    qc.invalidateQueries({ queryKey: qk.enterpriseInquiries, ...refetchActive }),
  ]);
}

type SubscriptionSnap = {
  plan_tier?: string;
  has_active_paid?: boolean;
  status?: string;
};

function subscriptionChanged(
  before: SubscriptionSnap | undefined,
  after: SubscriptionSnap | undefined,
): boolean {
  if (!after) return false;
  if (!before) return true;
  return (
    after.plan_tier !== before.plan_tier ||
    after.has_active_paid !== before.has_active_paid ||
    after.status !== before.status
  );
}

/**
 * After Paddle checkout.completed, webhook may lag. Poll subscription/quota
 * until the paid state changes (or attempts exhausted).
 */
export async function refreshBillingUntilSettled(
  qc: QueryClient,
  opts?: { attempts?: number; delayMs?: number },
) {
  const attempts = opts?.attempts ?? 12;
  const delayMs = opts?.delayMs ?? 1500;
  const before = qc.getQueryData<SubscriptionSnap>(qk.subscription);
  await invalidateBilling(qc);
  for (let i = 0; i < attempts; i++) {
    await new Promise((r) => setTimeout(r, delayMs));
    await Promise.all([
      qc.refetchQueries({ queryKey: qk.subscription }),
      qc.refetchQueries({ queryKey: qk.quota }),
      qc.refetchQueries({ queryKey: qk.receipts }),
    ]);
    const after = qc.getQueryData<SubscriptionSnap>(qk.subscription);
    if (subscriptionChanged(before, after)) {
      await invalidateBilling(qc);
      return;
    }
  }
  await invalidateBilling(qc);
}

/** Workspace membership graph after create / invite / accept / remove. */
export async function invalidateWorkspaces(qc: QueryClient) {
  await Promise.all([
    qc.invalidateQueries({ queryKey: qk.workspaces, ...refetchActive }),
    qc.invalidateQueries({ queryKey: qk.workspaceMembers, ...refetchActive }),
    qc.invalidateQueries({ queryKey: qk.workspaceInvites, ...refetchActive }),
    qc.invalidateQueries({ queryKey: qk.invitePreview, ...refetchActive }),
  ]);
}

/** Full account surface after destructive or identity-changing actions. */
export async function invalidateAccount(qc: QueryClient) {
  await Promise.all([
    invalidateBilling(qc),
    invalidateWorkspaces(qc),
    qc.invalidateQueries({ queryKey: qk.preferences, ...refetchActive }),
    qc.invalidateQueries({ queryKey: qk.apiKeys, ...refetchActive }),
  ]);
}
