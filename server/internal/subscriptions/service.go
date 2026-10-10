package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrPricingUnbound is returned when billing_settings.pricing_bound is false.
// Operators must sync the admin plan catalog to Paddle first.
var ErrPricingUnbound = errors.New("billing pricing unbound")

type ActiveSubscription struct {
	ID                   string
	PaddleSubscriptionID string
	PaddleCustomerID     string
	Status               string
	PriceID              string
	PlanTier             string
	BillingInterval      string
	PeriodStart          time.Time
	PeriodEnd            time.Time
	ExpiresAt            time.Time
	PlanRank             int
}

type PlanTarget struct {
	ID                string
	DisplayName       string
	PlanKind          string
	PlanRank          int
	CreditsMonthly    int
	PerSeat           bool
	Unlimited         bool
	CurrencyCode      string
	PriceMonthlyCents *int
	PriceYearlyCents  *int
	PaddleMonthly     *string
	PaddleYearly      *string
	PaddleTopup       *string
}

type Policy struct {
	AllowDowngrades             bool
	AllowCancelAtPeriodEnd      bool
	UpgradeProrationMode        string
	AllowMonthlyToAnnualUpgrade bool
	AnnualDiscountPercent       int
	ApplyPaddleDiscountOnAnnual bool
	PaddleDiscountID            *string
	PaddleDiscountCode          *string
	PricingBound                bool
}

type ChangeKind string

const (
	ChangeNewCheckout ChangeKind = "new_checkout"
	ChangeUpgrade     ChangeKind = "upgrade"
	ChangeBlocked     ChangeKind = "blocked"
	ChangeSame        ChangeKind = "same_plan"
)

type Decision struct {
	Kind          ChangeKind `json:"kind"`
	Reason        string     `json:"reason"`
	Code          string     `json:"code"`
	ActionLabel   string     `json:"action_label"`
	CanProceed    bool       `json:"can_proceed"`
	CurrentTier   string     `json:"current_tier,omitempty"`
	CurrentRank   int        `json:"current_rank"`
	TargetTier    string     `json:"target_tier"`
	TargetRank    int        `json:"target_rank"`
	IsExpired     bool       `json:"is_expired"`
	HasActivePaid bool       `json:"has_active_paid"`
}

type Service struct {
	DB     *pgxpool.Pool
	ReadDB *pgxpool.Pool
}

func NewService(db, readDB *pgxpool.Pool) *Service {
	if readDB == nil {
		readDB = db
	}
	return &Service{DB: db, ReadDB: readDB}
}

func (s *Service) readPool() *pgxpool.Pool {
	if s != nil && s.ReadDB != nil {
		return s.ReadDB
	}
	if s == nil {
		return nil
	}
	return s.DB
}

func (s *Service) LoadPolicy(ctx context.Context) (Policy, error) {
	var p Policy
	err := s.readPool().QueryRow(ctx, `
		select allow_downgrades,
		       coalesce(allow_cancel_at_period_end, true),
		       upgrade_proration_mode, allow_monthly_to_annual_as_upgrade,
		       annual_discount_percent, apply_paddle_discount_on_annual,
		       paddle_discount_id, paddle_discount_code,
		       coalesce(pricing_bound, false)
		from public.billing_settings where id = 'default'
	`).Scan(
		&p.AllowDowngrades, &p.AllowCancelAtPeriodEnd,
		&p.UpgradeProrationMode, &p.AllowMonthlyToAnnualUpgrade,
		&p.AnnualDiscountPercent, &p.ApplyPaddleDiscountOnAnnual,
		&p.PaddleDiscountID, &p.PaddleDiscountCode,
		&p.PricingBound,
	)
	if err != nil {
		return Policy{}, fmt.Errorf("billing_settings: %w", err)
	}
	if !p.PricingBound {
		return Policy{}, ErrPricingUnbound
	}
	// Self-serve checkout is upgrade-only while an unexpired paid period remains.
	// Product law: mid-cycle self-serve is upgrade-only. Coerce so a DB row of
	// allow_downgrades=true cannot advertise a capability DecideChange refuses
	// (checkout + Paddle webhook entitlement writes share this policy).
	if p.AllowDowngrades {
		p.AllowDowngrades = false
	}
	return p, nil
}

func (s *Service) LoadPlan(ctx context.Context, planID string) (PlanTarget, error) {
	var p PlanTarget
	err := s.readPool().QueryRow(ctx, `
		select id, display_name, plan_kind, plan_rank, credits_monthly, per_seat, unlimited, currency_code,
		       price_monthly_cents, price_yearly_cents,
		       paddle_price_id_monthly, paddle_price_id_yearly, paddle_price_id_topup
		from public.plan_catalog
		where id = $1 and is_active = true and is_public = true
	`, planID).Scan(
		&p.ID, &p.DisplayName, &p.PlanKind, &p.PlanRank, &p.CreditsMonthly, &p.PerSeat, &p.Unlimited, &p.CurrencyCode,
		&p.PriceMonthlyCents, &p.PriceYearlyCents,
		&p.PaddleMonthly, &p.PaddleYearly, &p.PaddleTopup,
	)
	if err != nil {
		return PlanTarget{}, err
	}
	return p, nil
}

func (s *Service) GetActiveSubscription(ctx context.Context, userID string) (*ActiveSubscription, error) {
	var sub ActiveSubscription
	var interval *string
	err := s.DB.QueryRow(ctx, `
		select s.id::text, s.paddle_subscription_id, s.paddle_customer_id, s.status, s.price_id,
		       s.plan_tier, s.billing_interval, s.current_period_start, s.current_period_end,
		       coalesce(s.expires_at, s.current_period_end),
		       p.plan_rank
		from public.subscriptions s
		inner join public.plan_catalog p on p.id = s.plan_tier
		where s.user_id = $1
		  and s.status in ('active', 'trialing', 'past_due', 'canceled')
		  and coalesce(s.expires_at, s.current_period_end) > now()
		order by coalesce(s.expires_at, s.current_period_end) desc
		limit 1
	`, userID).Scan(
		&sub.ID, &sub.PaddleSubscriptionID, &sub.PaddleCustomerID, &sub.Status, &sub.PriceID,
		&sub.PlanTier, &interval, &sub.PeriodStart, &sub.PeriodEnd, &sub.ExpiresAt, &sub.PlanRank,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if interval != nil {
		sub.BillingInterval = *interval
	}
	return &sub, nil
}

func (s *Service) DecideChange(ctx context.Context, userID, targetPlanID, targetInterval string) (Decision, PlanTarget, *ActiveSubscription, Policy, error) {
	policy, err := s.LoadPolicy(ctx)
	if err != nil {
		return Decision{}, PlanTarget{}, nil, Policy{}, err
	}
	target, err := s.LoadPlan(ctx, targetPlanID)
	if err != nil {
		return Decision{}, PlanTarget{}, nil, policy, fmt.Errorf("%s", MessageForCode("PLAN_NOT_FOUND"))
	}
	targetInterval = strings.ToLower(strings.TrimSpace(targetInterval))
	if targetInterval != "monthly" && targetInterval != "annual" {
		return Decision{}, PlanTarget{}, nil, policy, fmt.Errorf("%s", MessageForCode("PLAN_INTERVAL_INVALID"))
	}

	active, err := s.GetActiveSubscription(ctx, userID)
	if err != nil {
		return Decision{}, target, nil, policy, err
	}

	d := Decision{
		TargetTier: target.ID,
		TargetRank: target.PlanRank,
	}

	// One-time top-ups never change subscription tier; always new checkout.
	if target.PlanKind == "topup" {
		d.Kind = ChangeNewCheckout
		d.CanProceed = true
		d.Code = "TOPUP_CHECKOUT"
		d.Reason = MessageForCode("DECIDE_TOPUP_CHECKOUT")
		return decideOK(d, target, active, policy)
	}

	if IsDefaultPlanID(target.ID) || target.PlanKind == "enterprise" || strings.EqualFold(strings.TrimSpace(target.ID), "enterprise") {
		d.Kind = ChangeBlocked
		d.CanProceed = false
		d.Code = "NOT_SELF_SERVE"
		d.Reason = MessageForCode("DECIDE_NOT_SELF_SERVE")
		return decideOK(d, target, active, policy)
	}

	if active == nil {
		d.Kind = ChangeNewCheckout
		d.CanProceed = true
		d.IsExpired = true
		d.Code = "NEW_CHECKOUT"
		d.Reason = MessageForCode("DECIDE_NEW_CHECKOUT")
		return decideOK(d, target, nil, policy)
	}

	d.HasActivePaid = true
	d.CurrentTier = active.PlanTier
	d.CurrentRank = active.PlanRank
	d.IsExpired = false

	samePlan := strings.EqualFold(active.PlanTier, target.ID)
	sameInterval := normalizeInterval(active.BillingInterval) == targetInterval

	if samePlan && sameInterval {
		d.Kind = ChangeSame
		d.CanProceed = false
		d.Code = "SAME_PLAN"
		d.Reason = MessageForCode("DECIDE_SAME_PLAN")
		return decideOK(d, target, active, policy)
	}

	// Monthly to annual on the same tier counts as an upgrade when policy allows.
	if samePlan && normalizeInterval(active.BillingInterval) == "monthly" && targetInterval == "annual" {
		if policy.AllowMonthlyToAnnualUpgrade {
			if strings.EqualFold(active.Status, "canceled") {
				d.Kind = ChangeNewCheckout
				d.CanProceed = true
				d.Code = "INTERVAL_UPGRADE_NEW_CHECKOUT"
				d.Reason = MessageForCode("DECIDE_INTERVAL_UPGRADE_NEW_CHECKOUT")
				return decideOK(d, target, active, policy)
			}
			d.Kind = ChangeUpgrade
			d.CanProceed = true
			d.Code = "INTERVAL_UPGRADE"
			d.Reason = MessageForCode("DECIDE_INTERVAL_UPGRADE")
			return decideOK(d, target, active, policy)
		}
		d.Kind = ChangeBlocked
		d.CanProceed = false
		d.Code = "INTERVAL_CHANGE_DISABLED"
		d.Reason = MessageForCode("DECIDE_INTERVAL_CHANGE_DISABLED")
		return decideOK(d, target, active, policy)
	}

	// Annual to monthly on same plan is a downgrade of commitment.
	// Product rule: only upgrades are self-serve while unexpired.
	if samePlan && normalizeInterval(active.BillingInterval) == "annual" && targetInterval == "monthly" {
		d.Kind = ChangeBlocked
		d.CanProceed = false
		d.Code = "DOWNGRADE_BLOCKED"
		d.Reason = MessageForCode("DECIDE_DOWNGRADE_ANNUAL_TO_MONTHLY")
		return decideOK(d, target, active, policy)
	}

	if target.PlanRank > active.PlanRank {
		if strings.EqualFold(active.Status, "canceled") {
			d.Kind = ChangeNewCheckout
			d.CanProceed = true
			d.Code = "UPGRADE_NEW_CHECKOUT"
			d.Reason = MessageForCode("DECIDE_UPGRADE_NEW_CHECKOUT")
			return decideOK(d, target, active, policy)
		}
		d.Kind = ChangeUpgrade
		d.CanProceed = true
		d.Code = "UPGRADE_ALLOWED"
		d.Reason = MessageForCode("DECIDE_UPGRADE_ALLOWED")
		return decideOK(d, target, active, policy)
	}

	if target.PlanRank < active.PlanRank {
		// Product rule: self-serve is upgrade-only while an unexpired paid period remains.
		// LoadPolicy coerces allow_downgrades to false; DecideChange never enables checkout downgrades.
		// After ExpireIfNeeded clears the paid period, the same lower plan becomes a new checkout.
		d.Kind = ChangeBlocked
		d.CanProceed = false
		d.Code = "DOWNGRADE_BLOCKED"
		d.Reason = MessageForCode("DECIDE_DOWNGRADE_BLOCKED")
		until := active.ExpiresAt
		if until.IsZero() {
			until = active.PeriodEnd
		}
		if !until.IsZero() {
			if fmtMsg := MessageForCode("DECIDE_DOWNGRADE_BLOCKED_UNTIL_FMT"); fmtMsg != "" {
				d.Reason = fmt.Sprintf(fmtMsg, until.UTC().Format(time.RFC3339))
			}
		}
		return decideOK(d, target, active, policy)
	}

	// Same rank, different plan id (unusual) or interval edge cases.
	d.Kind = ChangeBlocked
	d.CanProceed = false
	d.Code = "CHANGE_NOT_ALLOWED"
	d.Reason = MessageForCode("DECIDE_CHANGE_NOT_ALLOWED")
	return decideOK(d, target, active, policy)
}

func normalizeInterval(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	switch v {
	case "year", "yearly", "annual", "annually":
		return "annual"
	case "month", "monthly":
		return "monthly"
	default:
		return v
	}
}

// ActionLabelForCode returns the UI CTA string for a decision code.
// Only site_messages ACTION:<code> after SeedAndRefreshSiteMessages.
// Never invents English in-process; empty means the UI must not invent a label.
func ActionLabelForCode(code string) string {
	if v, ok := cachedSiteMessage("ACTION:" + code); ok {
		return v
	}
	return ""
}

func builtinActionLabelForCode(code string) string {
	switch code {
	case "FREE_INCLUDED":
		return "Included"
	case "ENTERPRISE_CONTACT":
		return "Contact sales"
	case "PRICE_NOT_CONFIGURED":
		return "Price not configured"
	case "TOPUP_CHECKOUT":
		return "Buy credits"
	case "NEW_CHECKOUT", "UPGRADE_NEW_CHECKOUT", "INTERVAL_UPGRADE_NEW_CHECKOUT":
		return "Subscribe"
	case "INTERVAL_UPGRADE":
		return "Upgrade to annual"
	case "UPGRADE_ALLOWED":
		return "Upgrade"
	case "SAME_PLAN":
		return "Current plan"
	case "DOWNGRADE_BLOCKED", "DOWNGRADE_UNSUPPORTED":
		return "Downgrade blocked"
	case "NOT_SELF_SERVE", "INTERVAL_CHANGE_DISABLED", "CHANGE_NOT_ALLOWED":
		return "Unavailable"
	case "DASHBOARD_CHANGE_PLAN":
		return "Change plan"
	case "DASHBOARD_UPGRADE":
		return "Upgrade plan"
	case "PORTAL_OPEN":
		return "Manage billing"
	case "CONFIRM_REQUIRED":
		return "Confirm and upgrade"
	case "RECEIPTS_SYNC":
		return "Sync from Paddle"
	case "AVATAR_SYNC":
		return "Sync avatar"
	case "ACCOUNT_DELETE":
		return "Delete account"
	case "SIGN_OUT":
		return "Sign out"
	case "WORKSPACE_CREATE":
		return "Create"
	case "WORKSPACE_INVITE":
		return "Invite"
	case "WORKSPACE_INVITE_ACCEPT":
		return "Accept invite"
	case "WORKSPACE_INVITE_REVOKE":
		return "Revoke"
	case "WORKSPACE_INVITE_COPY":
		return "Copy invite link"
	case "WORKSPACE_INVITE_COPIED":
		return "Copied"
	case "WORKSPACE_INVITE_SIGN_IN":
		return "Sign in to accept"
	case "WORKSPACE_REMOVE":
		return "Remove"
	case "WORKSPACE_LEAVE":
		return "Leave"
	case "WORKSPACE_RENAME":
		return "Rename"
	case "WORKSPACE_RENAME_SAVE":
		return "Save name"
	case "WORKSPACE_DELETE":
		return "Delete workspace"
	case "WORKSPACE_ROLE_CHANGE":
		return "Change role"
	case "API_KEY_REVOKE":
		return "Revoke key"
	case "API_KEY_LIST_REFRESH":
		return "Refresh keys"
	case "PREFERENCES_SAVE":
		return "Save preferences"
	case "API_KEY_ISSUE":
		return "Issue key"
	case "API_KEY_ISSUE_RETRY":
		return "Retry"
	case "API_KEY_ISSUE_ANOTHER":
		return "Issue another"
	case "API_KEY_COPY":
		return "Copy key"
	case "API_KEY_COPIED":
		return "Copied"
	case "API_KEY_DEVICE_REGISTER":
		return "Register device"
	case "API_KEY_DEVICE_REMOVE":
		return "Remove device"
	case "RECEIPT_PRINT":
		return "Print"
	case "RECEIPT_DOWNLOAD_PDF":
		return "Download PDF"
	case "AUTH_OAUTH_GOOGLE":
		return "Continue with Google"
	case "AUTH_OAUTH_GITHUB":
		return "Continue with GitHub"
	case "AUTH_OAUTH_GITLAB":
		return "Continue with GitLab"
	case "AUTH_LINK_GOOGLE":
		return "Connect Google"
	case "AUTH_LINK_GITHUB":
		return "Connect GitHub"
	case "AUTH_LINK_GITLAB":
		return "Connect GitLab"
	case "AUTH_UNLINK":
		return "Disconnect"
	case "PAGINATION_SKIP_TO":
		return "Go"
	case "ENTERPRISE_SEND":
		return "Send inquiry"
	default:
		// Fail-closed: never invent Subscribe / Unavailable for unknown or empty codes.
		return ""
	}
}

// PendingLabelForCode returns the button loading label for a decision code.
// Only site_messages PENDING:<code> after SeedAndRefreshSiteMessages.
// Empty string means the control must not invent a spinner label.
func PendingLabelForCode(code string) string {
	if v, ok := cachedSiteMessage("PENDING:" + code); ok {
		return v
	}
	return ""
}

func builtinPendingLabelForCode(code string) string {
	switch code {
	case "UPGRADE_ALLOWED", "INTERVAL_UPGRADE", "CONFIRM_REQUIRED":
		return "Applying upgrade..."
	case "TOPUP_CHECKOUT", "NEW_CHECKOUT", "UPGRADE_NEW_CHECKOUT", "INTERVAL_UPGRADE_NEW_CHECKOUT":
		return "Opening checkout..."
	case "ENTERPRISE_CONTACT", "ENTERPRISE_SEND":
		return "Sending inquiry..."
	case "DASHBOARD_CHANGE_PLAN", "DASHBOARD_UPGRADE":
		return "Opening plans..."
	case "PORTAL_OPEN":
		return "Opening billing..."
	case "RECEIPTS_SYNC":
		return "Syncing receipts..."
	case "AVATAR_SYNC":
		return "Syncing avatar..."
	case "ACCOUNT_DELETE":
		return "Deleting account..."
	case "SIGN_OUT":
		return "Signing out..."
	case "WORKSPACE_CREATE":
		return "Creating workspace..."
	case "WORKSPACE_INVITE":
		return "Inviting..."
	case "WORKSPACE_INVITE_ACCEPT":
		return "Accepting invite..."
	case "WORKSPACE_INVITE_REVOKE":
		return "Revoking..."
	case "WORKSPACE_INVITE_COPY":
		return "Copying..."
	case "WORKSPACE_INVITE_COPIED":
		return ""
	case "WORKSPACE_INVITE_SIGN_IN":
		return "Redirecting..."
	case "WORKSPACE_REMOVE":
		return "Removing..."
	case "WORKSPACE_LEAVE":
		return "Leaving..."
	case "WORKSPACE_RENAME":
		return "Renaming..."
	case "WORKSPACE_RENAME_SAVE":
		return "Saving..."
	case "WORKSPACE_DELETE":
		return "Deleting..."
	case "WORKSPACE_ROLE_CHANGE":
		return "Updating role..."
	case "PREFERENCES_SAVE":
		return "Updating..."
	case "API_KEY_ISSUE", "API_KEY_ISSUE_RETRY", "API_KEY_ISSUE_ANOTHER":
		return "Issuing key..."
	case "API_KEY_REVOKE":
		return "Revoking key..."
	case "API_KEY_LIST_REFRESH":
		return "Refreshing..."
	case "API_KEY_COPY":
		return "Copying..."
	case "API_KEY_COPIED":
		return ""
	case "API_KEY_DEVICE_REGISTER":
		return "Registering..."
	case "API_KEY_DEVICE_REMOVE":
		return "Removing..."
	case "RECEIPT_PRINT":
		return "Printing..."
	case "RECEIPT_DOWNLOAD_PDF":
		return "Downloading PDF..."
	case "AUTH_OAUTH_GOOGLE":
		return "Redirecting to Google..."
	case "AUTH_OAUTH_GITHUB":
		return "Redirecting to GitHub..."
	case "AUTH_OAUTH_GITLAB":
		return "Redirecting to GitLab..."
	case "AUTH_LINK_GOOGLE":
		return "Connecting Google..."
	case "AUTH_LINK_GITHUB":
		return "Connecting GitHub..."
	case "AUTH_LINK_GITLAB":
		return "Connecting GitLab..."
	case "AUTH_UNLINK":
		return "Disconnecting..."
	case "PAGINATION_SKIP_TO":
		return "Skipping..."
	case "PAGINATION_FIRST":
		return "Loading first..."
	case "PAGINATION_PREV":
		return "Loading previous..."
	case "PAGINATION_NEXT":
		return "Loading next..."
	case "PAGINATION_LAST":
		return "Loading last..."
	case "FREE_INCLUDED", "SAME_PLAN", "DOWNGRADE_BLOCKED", "DOWNGRADE_UNSUPPORTED",
		"NOT_SELF_SERVE", "INTERVAL_CHANGE_DISABLED", "CHANGE_NOT_ALLOWED", "PRICE_NOT_CONFIGURED":
		return ""
	default:
		// Unknown codes must not invent marketing copy; leave spinner text empty.
		return ""
	}
}

// IsDefaultPlanID reports whether id matches site_messages DEFAULT_PLAN_TIER.
// Empty DEFAULT_PLAN_TIER fails closed (never invents "free").
func IsDefaultPlanID(id string) bool {
	defaultTier := strings.ToLower(strings.TrimSpace(MessageForCode("DEFAULT_PLAN_TIER")))
	if defaultTier == "" {
		return false
	}
	return strings.ToLower(strings.TrimSpace(id)) == defaultTier
}

// MessageForCode returns backend-owned user-facing copy for billing and auth UX.
// Prefers public.site_messages after SeedAndRefreshSiteMessages (DB-driven).
// Before the cache is warmed (tests without DB), falls back to builtins.
// After warm, missing rows fail closed to empty (no invent).
// MessageForCode returns DB-backed chrome only (same fail-closed contract as
// ActionLabelForCode). Builtins exist solely for SeedAndRefreshSiteMessages.
func MessageForCode(code string) string {
	if v, ok := cachedSiteMessage(code); ok {
		return v
	}
	return ""
}

func builtinMessageForCode(code string) string {
	switch code {
	case "PRICE_NOT_CONFIGURED_REASON":
		return "This plan is not synced to Paddle yet. An operator must sync the plan catalog from the admin dashboard."
	case "PRICE_NOT_CONFIGURED":
		return "This plan is not synced to Paddle yet. An operator must sync the plan catalog from the admin dashboard."
	case "PRICE_CATALOG_DRIFT":
		return "Checkout blocked: Paddle price no longer matches admin catalog amounts. Re-sync plans to Paddle from the admin dashboard."
	case "RECEIPT_PERIOD_JOIN_FMT":
		return "%s - %s"
	case "RECEIPT_TAX_RATE_FMT":
		return " (%0.2f%%)"
	case "AUTH_REQUIRED_PLAN":
		return "Sign in is required to change plans."
	case "AUTH_REQUIRED_ENTERPRISE":
		return "Sign in is required for an enterprise inquiry."
	case "ENTERPRISE_MESSAGE_REQUIRED":
		return "Describe your team needs before sending."
	case "PADDLE_JS_NOT_READY":
		return "Paddle.js is not initialized. Set NEXT_PUBLIC_PADDLE_CLIENT_TOKEN and NEXT_PUBLIC_PADDLE_ENV."
	case "CHECKOUT_PRICE_MISSING":
		return "Checkout unavailable: the API did not return a Paddle price_id."
	case "UPGRADE_MISSING_API_MESSAGE":
		return "Upgrade completed but the API returned no confirmation message."
	case "INQUIRY_MISSING_CONFIRM":
		return "Inquiry stored but the API returned no confirmation message."
	case "CHANGE_BLOCKED_DEFAULT":
		return "That plan change is not available for your account right now."
	case "CHECKOUT_UNAVAILABLE_DEFAULT":
		return "Checkout unavailable for this plan."
	case "UPGRADE_NOT_APPLIED":
		return "Upgrade was not applied."
	case "CHECKOUT_REQUEST_FAILED":
		return "Checkout request failed."
	case "UPGRADE_REQUEST_FAILED":
		return "Upgrade request failed."
	case "ENTERPRISE_INQUIRY_FAILED":
		return "Enterprise inquiry failed."
	case "ENTERPRISE_INQUIRY_RECEIVED":
		return "Enterprise inquiry received. Our team will follow up using your account email."
	case "STATUS_FREE":
		return "free"
	case "STATUS_ACTIVE":
		return "active"
	case "STATUS_EXPIRED":
		return "expired"
	case "STATUS_CANCELED":
		return "canceled"
	case "STATUS_PAST_DUE":
		return "past due"
	case "STATUS_TRIALING":
		return "trialing"
	case "STATUS_PAUSED":
		return "paused"
	case "CHECKOUT_QUANTITY_INVALID":
		return "Seat quantity must be at least 1."
	case "DOWNGRADE_POLICY":
		return "Self-serve downgrades are disabled. Upgrades use Paddle proration on remaining unexpired usage."
	case "DOWNGRADE_POLICY_UNTIL_FMT":
		return "Self-serve plan downgrades are disabled while your current plan is active through %s. Upgrades use Paddle proration on remaining unexpired usage. You can cancel at period end from Manage billing."
	case "PORTAL_CANCEL_DISABLED":
		return "Subscription cancellation from the portal is disabled for this account. Contact support."
	case "RECEIPT_PDF_FILENAME_REQUIRED":
		return "Receipt PDF filename template is not configured."
	case "ACCOUNT_DELETE_CONFIRM":
		return "Permanently delete your account and all associated data? This cannot be undone."
	case "ACCOUNT_DELETE_FAILED":
		return "Account deletion failed. Try again or contact support."
	case "ACCOUNT_DELETE_SUCCESS":
		return "Account and personal data were deleted per GDPR right to be forgotten."
	case "WORKSPACE_LEAVE_CONFIRM":
		return "Leave this workspace? You will lose access until invited again."
	case "WORKSPACE_REMOVE_CONFIRM":
		return "Remove this member from the workspace?"
	case "WORKSPACE_DELETE_CONFIRM":
		return "Delete this workspace permanently? Members and pending invites are removed. This cannot be undone."
	case "WORKSPACE_BULK_DELETE_CONFIRM":
		return "Delete the selected workspaces permanently? Members and pending invites are removed. This cannot be undone."
	case "TABLE_BULK_DELETE":
		return "Delete selected"
	case "WORKSPACE_RENAME_TITLE":
		return "Rename workspace"
	case "WORKSPACE_ROLE_CHANGE_TITLE":
		return "Change member role"
	case "WORKSPACE_ROLE_LABEL":
		return "Role"
	case "WORKSPACE_ROLE_DESCRIPTION":
		return "Owners and admins manage the workspace. Members use shared seats and credits."
	case "WORKSPACE_RENAMED":
		return "Workspace renamed."
	case "WORKSPACE_DELETED":
		return "Workspace deleted."
	case "WORKSPACE_MEMBER_ROLE_UPDATED":
		return "Member role updated."
	case "WS_RENAME_FORBIDDEN":
		return "Only workspace owners can rename this workspace."
	case "WS_DELETE_FORBIDDEN":
		return "Only workspace owners can delete this workspace."
	case "WS_ROLE_CHANGE_FORBIDDEN":
		return "You cannot change this member role."
	case "WS_INVALID_MEMBER_ROLE":
		return "That role is not allowed."
	case "WS_CANNOT_DEMOTE_LAST_OWNER":
		return "Cannot change role: the workspace must keep at least one owner."
	case "WS_WORKSPACE_NOT_FOUND":
		return "Workspace not found."
	case "WS_MEMBER_NOT_FOUND":
		return "Member not found."
	case "WS_RENAME_FAILED":
		return "Could not rename the workspace."
	case "WS_DELETE_FAILED":
		return "Could not delete the workspace."
	case "WS_ROLE_CHANGE_FAILED":
		return "Could not update the member role."
	case "API_KEY_REVOKE_CONFIRM":
		return "Revoke this API key? CLI and gateway requests using it will stop working."
	case "API_KEY_INTRO":
		return "Keys authenticate trim login and CLI proxy calls. The full secret is shown once at issue time."
	case "API_KEY_EMPTY":
		return "No API keys yet. Issue one here or via trim login."
	case "API_KEY_SEARCH":
		return "Search API keys"
	case "API_KEY_SEARCH_DESC":
		return "Filters by key prefix, id, or status as you type."
	case "API_KEY_STATUS_ACTIVE":
		return "active"
	case "API_KEY_STATUS_REVOKED":
		return "revoked"
	case "API_KEY_CREATED_PREFIX":
		return "created"
	case "API_KEY_FRESH_HINT":
		return "New key (copy now; it will not be shown again)"
	case "API_KEY_FRESH_DEVICE_REQUIRED":
		return "Copy this key now. It will not authenticate until you register a hardware UUID below (Settings → Register device)."
	case "API_KEY_DEVICE_HINT":
		return "Every API key must be device-bound. CLI login binds the issuing machine. For IDE/CI or dashboard-issued keys, register hardware under Settings (signed-in) before the key will authenticate."
	case "API_KEY_DEVICE_HW_LABEL":
		return "Hardware UUID"
	case "API_KEY_DEVICE_AGENT_LABEL":
		return "Agent ID (cli, ide, or ci)"
	case "API_KEY_DEVICE_AGENT_CLI":
		return "cli"
	case "API_KEY_DEVICE_AGENT_IDE":
		return "ide"
	case "API_KEY_DEVICE_AGENT_CI":
		return "ci"
	case "API_KEY_DEVICE_BOUND_LABEL":
		return "Device-bound"
	case "API_KEY_DEVICE_UNBOUND_LABEL":
		return "Not device-bound"
	case "API_KEY_DEVICE_COUNT_FMT":
		return "%d device(s)"
	case "API_KEY_DEVICE_HW_REQUIRED":
		return "Hardware UUID is required."
	case "API_KEY_DEVICE_NOT_FOUND":
		return "Device registration not found."
	case "API_KEY_DEVICE_UNBOUND":
		return "This API key has no registered devices. Sign in to the dashboard, open Settings → API keys, and register this machine's hardware UUID before using the key."
	case "CI_AGENT_HINT":
		return "CI jobs must send X-Trim-Agent-Id: ci and X-Hardware-UUID for device-bound keys."
	case "RECEIPT_LABEL_TAX_ID":
		return "Tax ID"
	case "PAGINATION_PREV":
		return "Prev"
	case "PAGINATION_NEXT":
		return "Next"
	case "PAGINATION_FIRST":
		return "First"
	case "PAGINATION_LAST":
		return "Last"
	case "PAGINATION_SKIP_TO_LABEL":
		return "Skip to page"
	case "PAGINATION_SKIP_TO_INVALID":
		return "That page is outside the available range."
	case "PAGINATION_PAGE_SUMMARY_FMT":
		return "Page %d of %d (%d total)"
	case "STATS_UNKNOWN_LABEL":
		return "unknown"
	case "CLI_AUTH_EYEBROW":
		return "Trim CLI"
	case "CLI_AUTH_TITLE":
		return "Connect your terminal"
	case "CLI_AUTH_BODY":
		return "Paste this key into the terminal where trim login is waiting. The key is shown once."
	case "CLI_AUTH_ISSUE_FAILED":
		return "Failed to issue API key"
	case "CLI_AUTH_FOOTER":
		return "After pasting, run trim status to confirm quota, then trim start."
	case "CLI_AUTH_COPIED_RESET_MS":
		return "2000"
	case "RECEIPT_PRINT_PENDING_MS":
		return "400"
	case "RECEIPT_NOT_FOUND":
		return "Receipt not found."
	case "RECEIPT_DOCUMENT_TITLE":
		return "Tax invoice"
	case "RECEIPT_PERIOD_ONE_TIME":
		return "One-time"
	case "RECEIPT_DOWNLOAD_PDF":
		return "Download PDF"
	case "RECEIPT_BACK":
		return "Back"
	case "RECEIPT_SELLER_MISSING":
		return "Receipt seller identity is not configured (COMPANY_LEGAL_NAME)."
	case "RECEIPT_SECTION_BILL_TO":
		return "Invoice to"
	case "RECEIPT_SECTION_INVOICE_FROM":
		return "Invoice from"
	case "RECEIPT_SECTION_INVOICE_DETAILS":
		return "Invoice details"
	case "RECEIPT_SECTION_TRANSACTION":
		return "Transaction"
	case "RECEIPT_SECTION_TAX_BREAKDOWN":
		return "Tax breakdown"
	case "RECEIPT_SECTION_PERIOD":
		return "Billing period"
	case "RECEIPT_SECTION_AMOUNT":
		return "Amount due"
	case "RECEIPT_SECTION_PAYMENT":
		return "Payment method"
	case "RECEIPT_FOOTER":
		return "Questions about this invoice? Contact support using the address on this document. Card statements may show Paddle as the merchant of record."
	case "RECEIPT_ISSUED_PREFIX":
		return "Issued"
	case "RECEIPT_COL_DESCRIPTION":
		return "Product"
	case "RECEIPT_COL_PRODUCT":
		return "Product"
	case "RECEIPT_COL_SKU":
		return "SKU"
	case "RECEIPT_COL_QTY":
		return "Qty"
	case "RECEIPT_COL_UNIT":
		return "Unit price"
	case "RECEIPT_COL_TAX_RATE":
		return "Tax rate"
	case "RECEIPT_COL_AMOUNT":
		return "Amount"
	case "RECEIPT_LABEL_SUBTOTAL":
		return "Subtotal"
	case "RECEIPT_LABEL_TAX":
		return "VAT"
	case "RECEIPT_LABEL_TOTAL":
		return "Total"
	case "RECEIPT_LABEL_AMOUNT_PAID":
		return "Amount paid"
	case "RECEIPT_LABEL_INVOICE_REFERENCE":
		return "Invoice reference"
	case "RECEIPT_LABEL_TRANSACTION_ID":
		return "Transaction"
	case "RECEIPT_LABEL_CURRENCY":
		return "Currency code"
	case "RECEIPT_LABEL_TAX_PERCENT":
		return "Tax %"
	case "RECEIPT_LABEL_TAX_TOTAL":
		return "Tax total"
	case "RECEIPT_TAX_RATE_ZERO":
		return "0%"
	case "RECEIPT_TAX_RATE_PERCENT_FMT":
		return "%0.2f%%"
	case "RECEIPT_MERCHANT_VIA":
		return "via Paddle.com"
	case "RECEIPT_HEADER_META_SEP":
		return " - "
	case "RECEIPT_PRICE_NAME_MONTHLY_FMT":
		return "%s (monthly)"
	case "RECEIPT_PRICE_NAME_YEARLY_FMT":
		return "%s (yearly)"
	case "RECEIPT_EMPTY_SKU":
		return ""
	case "PLANS_DIALOG_DESCRIPTION":
		return "Pick a plan that fits how you use Trim. Switch between monthly and annual billing anytime before checkout."
	case "PLANS_DIALOG_UPGRADES_ONLY_NOTE":
		return "You can upgrade to a higher plan. Downgrades are not available while your current plan is active."
	case "TOPUP_DIALOG_TITLE":
		return "Buy credit top-up"
	case "TOPUP_DIALOG_DESCRIPTION":
		return "Add cloud credits to your account without changing your plan. Credits apply after checkout completes."
	case "TOPUP_DIALOG_EMPTY":
		return "No top-up packs are available right now. Check back later or upgrade your plan instead."
	case "PLANS_ANNUAL_TOGGLE":
		return "Annual"
	case "PLANS_MONTHLY_TOGGLE":
		return "Monthly"
	case "AUTH_PROVIDERS_EMPTY":
		return "No sign-in providers are enabled. Set ALLOWED_AUTH_PROVIDERS on the API."
	case "LOGIN_TITLE":
		return "Sign in to Trim"
	case "LOGIN_PROVIDER_DISABLED":
		return "That sign-in provider is not enabled for this deployment."
	case "LOGIN_FAILED":
		return "Sign in failed."
	case "LOGIN_ENV_APP_URL_MISSING":
		return "NEXT_PUBLIC_APP_URL is not set"
	case "LOGIN_OAUTH_EXCHANGE_FAILED":
		return "Sign-in could not be completed. Try again."
	case "LOGIN_OAUTH_EMAIL_MISSING":
		return "This sign-in provider did not share an email address. Grant email access and try again."
	case "AUTH_OAUTH_LINK_INTENT":
		return "link"
	case "LOGIN_OAUTH_LINK_FAILED":
		return "Could not connect that sign-in provider. Try again."
	case "ACCOUNT_IDENTITIES_UNAVAILABLE":
		return "Signed-in account identities could not be loaded from the database."
	case "PREFERENCES_LINK_HINT":
		return "Connect another provider so you can sign in with it later."
	case "PREFERENCES_UNLINK_CONFIRM":
		return "Disconnect this sign-in provider from your account?"
	case "PREFERENCES_UNLINK_LAST_BLOCKED":
		return "Keep at least one sign-in provider connected."
	case "PREFERENCES_UNLINK_FAILED":
		return "Could not disconnect that sign-in provider. Try again."
	case "PREFERENCES_UNLINK_DONE":
		return "Sign-in provider disconnected."
	case "PREFERENCES_IDENTITY_PRIMARY_LABEL":
		return "Primary"
	case "PREFERENCES_IDENTITY_LAST_USED_LABEL":
		return "Last used"
	case "LOGIN_MISSING_SUPABASE_ENV":
		return "Supabase environment is not configured."
	case "LOGIN_DEFAULT_NEXT":
		return "/dashboard"
	case "LOGIN_DEFAULT_NEXT_MISSING":
		return "Sign-in destination is not configured. Set LOGIN_DEFAULT_NEXT in site_messages."
	case "APP_PATH_DASHBOARD":
		return "/dashboard"
	case "APP_PATH_TEAM":
		return "/dashboard/team"
	case "APP_PATH_SETTINGS":
		return "/dashboard/settings"
	case "APP_PATH_RECEIPTS_PREFIX":
		return "/dashboard/receipts/"
	case "APP_PATH_LOGIN":
		return "/login"
	case "APP_PATH_PRIVACY":
		return "/privacy"
	case "APP_PATH_TERMS":
		return "/terms"
	case "APP_PATH_HOME":
		return "/"
	case "APP_PATH_UPGRADE":
		return "/dashboard"
	case "APP_PATH_AUTH_CALLBACK":
		return "/auth/callback"
	case "APP_PATH_CLI_AUTH":
		return "/cli/auth"
	case "APP_PATH_INVITE_PREFIX":
		return "/invite/"
	case "DEFAULT_PLAN_TIER":
		return "free"
	case "API_KEY_PREFIX_ELLIPSIS":
		return "..."
	case "SITE_HTML_LANG":
		return "en"
	case "LOCAL_PREVIEW_TRUNC_SUFFIX":
		return "\n… (truncated)"
	case "LOCAL_TUI_TRUNC_SUFFIX":
		return "…"
	case "LOCAL_STATS_URL_FMT":
		return "http://127.0.0.1:{port}/v1/stats"
	case "LOCAL_TUI_HTTP_TIMEOUT_UNSET":
		return "http timeout not configured"
	case "LOCAL_HTML_LANG":
		return "en"
	case "LANDING_BRAND":
		return "Trim"
	case "LANDING_NAV_DASHBOARD":
		return "Dashboard"
	case "LANDING_NAV_SIGN_IN":
		return "Sign in"
	case "LANDING_EYEBROW":
		return "Open-core context optimization"
	case "LANDING_HEADLINE":
		return "Cut LLM context by 80%. Keep the code that matters."
	case "LANDING_TAGLINE":
		return "Shrink noisy context on your machine so you pay for signal, not vendor dumps. Trim cloud meters the account; it never needs your raw context pack."
	case "LANDING_CTA_START":
		return "Get started"
	case "LANDING_CTA_SOURCE":
		return "View source"
	case "LANDING_FOOTER_PRIVACY":
		return "Privacy"
	case "LANDING_FOOTER_TERMS":
		return "Terms"
	case "LANDING_FOOTER_GITHUB":
		return "GitHub"
	case "LANDING_COPYRIGHT_FMT":
		return "© %d Trim"
	case "LANDING_SOURCE_URL":
		return "https://github.com/usetrim/trim"
	case "LANDING_INSTALL_SNIPPET":
		return "curl -fsSL https://use-trim.com/install.sh | sh\ntrim start\n# Cursor -> Override Base URL -> http://localhost:8888/v1"
	case "LANDING_NAV_PRICING":
		return "Pricing"
	case "LANDING_DEMO_TITLE":
		return "See exactly what Trim cuts"
	case "LANDING_DEMO_SUBTITLE":
		return "Auto showcase of real prompt shapes. Hover to pause. Click a tab to jump."
	case "LANDING_DEMO_BEFORE_LABEL":
		return "Before Trim"
	case "LANDING_DEMO_AFTER_LABEL":
		return "After Trim"
	case "LANDING_DEMO_RUN_LABEL":
		return "Run Trim"
	case "LANDING_DEMO_REPLAY_LABEL":
		return "Replay"
	case "LANDING_DEMO_TOKENS_IN_FMT":
		return "%s tokens in"
	case "LANDING_DEMO_TOKENS_OUT_FMT":
		return "%s tokens out"
	case "LANDING_DEMO_SAVED_FMT":
		return "%s%% less context"
	case "LANDING_DEMO_BEFORE_BODY":
		return "// vendor/react/index.js (pulled into context)\nexport function createElement() { /* 400 lines */ }\n\n// build/logs/ci-run-88421.txt\n[INFO] compiling...\n[WARN] deprecated dep\n[DEBUG] cache miss x 220\n\n// node_modules/.cache/noise.ts\nconst _unused = Array(500).fill(0);\n\n// src/auth/session.ts (signal)\nexport async function requireUser(req) {\n  const session = await getSession(req);\n  if (!session) throw new AuthError(\"unauthorized\");\n  return session.user;\n}"
	case "LANDING_DEMO_AFTER_BODY":
		return "// src/auth/session.ts (kept)\nexport async function requireUser(req) {\n  const session = await getSession(req);\n  if (!session) throw new AuthError(\"unauthorized\");\n  return session.user;\n}\n\n// vendor/react: reduced to a short outline\n// createElement() { /* omitted */ }\n\n// build logs: pruned\n// [noise lines removed]"
	case "LANDING_HOW_TITLE":
		return "How Trim works"
	case "LANDING_HOW_1_TITLE":
		return "Install and start the local proxy"
	case "LANDING_HOW_1_BODY":
		return "trim start binds an OpenAI-compatible proxy on localhost. Compression runs locally, then the slim prompt goes to your model provider."
	case "LANDING_HOW_2_TITLE":
		return "Wire Cursor (or any compatible IDE)"
	case "LANDING_HOW_2_BODY":
		return "Override Base URL to the URL trim start printed (example http://127.0.0.1:8888/v1). Requests route through Trim before the model."
	case "LANDING_HOW_3_TITLE":
		return "Trim then meter"
	case "LANDING_HOW_3_BODY":
		return "Noise is cut locally. Cloud meters usage, quotas, and receipts after you sign in - not your source tree."
	case "LANDING_PRICING_TITLE":
		return "Pricing"
	case "LANDING_PRICING_SUBTITLE":
		return "Start free. Cloud meters entitlements and billing; compression stays on your machine."
	case "LANDING_PRICING_CTA":
		return "Choose plan"
	case "LANDING_PRICING_ANNUAL":
		return "Annual"
	case "LANDING_PRICING_MONTHLY":
		return "Monthly"
	case "LANDING_PRICING_POPULAR_BADGE":
		return "Popular"
	case "LANDING_PRICING_ANNUAL_BILLING":
		return "Annual billing"
	case "LANDING_PRICING_ANNUAL_SAVE_HINT_FMT":
		return "Save %d%% on yearly plans"
	case "LANDING_INSTALL_TITLE":
		return "Install in one minute"
	case "LANDING_CTA_DEMO":
		return "See live demo"
	case "LANDING_HERO_NOTE":
		return "Compression runs on your machine. Trim cloud handles login, quotas, and billing, not your full context. Only the slim prompt you choose goes upstream."
	case "LANDING_NAV_WHY":
		return "Why Trim"
	case "LANDING_NAV_HOW":
		return "How it works"
	case "LANDING_NAV_INSTALL":
		return "Install"
	case "LANDING_STATS_TITLE":
		return "Less context. Lower bills."
	case "LANDING_STATS_SUBTITLE":
		return "Illustrative numbers for a heavy Cursor-style coding session. Your savings depend on how noisy the prompt is."
	case "LANDING_STAT_1_VALUE":
		return "80%"
	case "LANDING_STAT_1_LABEL":
		return "Fewer tokens sent to the model"
	case "LANDING_STAT_2_VALUE":
		return "5x"
	case "LANDING_STAT_2_LABEL":
		return "More room for real work in the same budget"
	case "LANDING_STAT_3_VALUE":
		return "$"
	case "LANDING_STAT_3_LABEL":
		return "USD saved on input tokens you never needed"
	case "LANDING_STAT_3_HINT":
		return "Example: a $40 input bill can drop toward $8 when most of the prompt was noise."
	case "LANDING_FLOW_TITLE":
		return "How Trim works"
	case "LANDING_FLOW_SUBTITLE":
		return "End to end: from install on your machine to a smaller bill on every chat."
	case "LANDING_FLOW_1_TITLE":
		return "Install Trim and start the local proxy"
	case "LANDING_FLOW_1_BODY":
		return "Run the one-line installer, then trim start. Trim listens on TRIM_PORT (example http://127.0.0.1:8888/v1) as an OpenAI-compatible endpoint. Your code stays on this machine."
	case "LANDING_FLOW_2_TITLE":
		return "Point your IDE at Trim"
	case "LANDING_FLOW_2_BODY":
		return "In Cursor: Settings → Models → OpenAI Compatible → Override Base URL to the Trim listen URL. No new editor. Same Composer and agents."
	case "LANDING_FLOW_3_TITLE":
		return "Every request hits Trim first"
	case "LANDING_FLOW_3_BODY":
		return "Chat and agents POST through the local proxy. Fast Mode drops vendor dumps, lockfiles, and log spam; it keeps the files and stacks you are actually editing."
	case "LANDING_FLOW_4_TITLE":
		return "The model sees less. You see the savings."
	case "LANDING_FLOW_4_BODY":
		return "A slim prompt goes upstream. Same task, far fewer input tokens. Dashboard shows tokens in → out, percent cut, and USD direction so the win is measurable."
	case "LANDING_USE_TITLE":
		return "Get started in three steps"
	case "LANDING_USE_SUBTITLE":
		return "Same path as above - short version."
	case "LANDING_USE_1_TITLE":
		return "Install and run trim start"
	case "LANDING_USE_1_BODY":
		return "curl the install script (or use your package manager), then start the local proxy."
	case "LANDING_USE_2_TITLE":
		return "Set Override Base URL"
	case "LANDING_USE_2_BODY":
		return "Cursor → Models → OpenAI Compatible → URL from trim start (example http://127.0.0.1:8888/v1)"
	case "LANDING_USE_3_TITLE":
		return "Code as usual - check the dashboard"
	case "LANDING_USE_3_BODY":
		return "Composer and agents already go through Trim. Open the dashboard to see tokens trimmed and USD saved."
	case "LANDING_WHY_TITLE":
		return "Why teams choose Trim"
	case "LANDING_WHY_SUBTITLE":
		return "Local compression. Thin cloud control plane. The IDE tools you already use."
	case "LANDING_WHY_1_TITLE":
		return "Your context stays local"
	case "LANDING_WHY_1_BODY":
		return "Fast Mode compresses on your machine before anything is forwarded. Trim cloud meters the account; it does not need the raw pack."
	case "LANDING_WHY_2_TITLE":
		return "Works with the tools you already use"
	case "LANDING_WHY_2_BODY":
		return "Point your IDE override base URL at Trim. No new editor. No migration project."
	case "LANDING_WHY_3_TITLE":
		return "Open-core and inspectable"
	case "LANDING_WHY_3_BODY":
		return "Core proxy is open. You can read what runs on your laptop before you trust it."
	case "LANDING_WHY_4_TITLE":
		return "Fast Mode and Deep Mode"
	case "LANDING_WHY_4_BODY":
		return "Start with Fast Mode on the live proxy. Turn Deep Mode on in Preferences (then trim config sync) so trim start also runs Deep after Fast. Or use trim compress --deep for file/batch jobs."
	case "LANDING_WHY_5_TITLE":
		return "Quotas and receipts in one place"
	case "LANDING_WHY_5_BODY":
		return "Cloud meters usage, plans, and invoices so finance and engineering share the same story."
	case "LANDING_WHY_6_TITLE":
		return "Built for real agent traffic"
	case "LANDING_WHY_6_BODY":
		return "Agents love dumping the world into context. Trim is for that world: noisy, repetitive, expensive."
	case "LANDING_WHY_7_TITLE":
		return "Windows, macOS, and Linux"
	case "LANDING_WHY_7_BODY":
		return "Same install story everywhere. Your team does not need three different tools to save tokens."
	case "LANDING_WHY_8_TITLE":
		return "Model-agnostic savings"
	case "LANDING_WHY_8_BODY":
		return "Works with OpenAI-compatible endpoints. Shrink input once; keep the same habit when you switch models."
	case "LANDING_WHY_9_TITLE":
		return "Faster turns, not just cheaper ones"
	case "LANDING_WHY_9_BODY":
		return "Smaller prompts often mean less wait. You feel the win in latency as well as in the bill."
	case "LANDING_WHY_10_TITLE":
		return "Free to try, clear to upgrade"
	case "LANDING_WHY_10_BODY":
		return "Install the local proxy in a minute. Sign in for Trim cloud metering, plans, and team receipts; upstream models still require network."
	case "LANDING_WHY_11_TITLE":
		return "Built for regulated teams"
	case "LANDING_WHY_11_BODY":
		return "Heavy context is trimmed on the laptop. Cloud sees auth, quotas, and billing, not your private repo dump. Upstream still receives only the slim prompt you send."
	case "LANDING_WHY_12_TITLE":
		return "ROI you can show your manager"
	case "LANDING_WHY_12_BODY":
		return "Tokens in, tokens out, percent cut, and USD direction. Finance understands the dashboard in one glance."
	case "LANDING_INSTALL_SUBTITLE":
		return "Local proxy compresses on your machine. Sign in for Trim cloud auth, quotas, and paid plans; chat still needs an upstream model."
	case "LANDING_DEMO_COST_HINT":
		return "Illustrative: 12.8k -> 2.1k tokens. At $3 / 1M input tokens that is about $0.038 -> $0.006 per turn (83% less USD on input)."
	case "LEGAL_PRIVACY_TITLE":
		return "Privacy Policy"
	case "LEGAL_TERMS_TITLE":
		return "Terms of Service"
	case "LEGAL_UPDATED_PREFIX":
		return "Last updated:"
	case "LEGAL_UPDATED_DATE":
		return "24 March 2026"
	case "LEGAL_LINK_PRIVACY":
		return "Privacy Policy"
	case "LEGAL_LINK_TERMS":
		return "Terms of Service"
	case "LEGAL_LINK_HOME":
		return "Home"
	case "AUTH_NOT_SIGNED_IN":
		return "Not signed in"
	case "LEGAL_SUPPORT_EMAIL_MISSING":
		return "Support email is not configured for this deployment."
	case "LEGAL_PRIVACY_INTRO":
		return "This Privacy Policy describes how Trim collects, uses, and shares information when you use the hosted Service at use-trim.com."
	case "LEGAL_PRIVACY_SEC_AUTH_HEADING":
		return "Account and authentication"
	case "LEGAL_PRIVACY_SEC_AUTH_BODY":
		return "Sign-in uses identity providers enabled for the Service. We receive account identifiers the provider shares with us, such as account id, email, display name, and avatar. We do not offer password or phone sign-up."
	case "LEGAL_PRIVACY_SEC_USAGE_HEADING":
		return "Usage, billing, and quotas"
	case "LEGAL_PRIVACY_SEC_USAGE_BODY":
		return "We store metered usage, subscription status, receipts, and credit balances needed to operate quotas and billing. Our payment partner processes payments as Merchant of Record and may issue tax invoices under its terms. Clearing local client files does not reset hosted quotas."
	case "LEGAL_PRIVACY_SEC_DEVICE_HEADING":
		return "Security and abuse prevention"
	case "LEGAL_PRIVACY_SEC_DEVICE_BODY":
		return "To protect accounts and free-tier abuse, we may store limited device binding signals, IP address, and related security telemetry. These signals are used for security enforcement, not for advertising."
	case "LEGAL_PRIVACY_SEC_LOCAL_HEADING":
		return "Local processing"
	case "LEGAL_PRIVACY_SEC_LOCAL_BODY":
		return "The Trim client can process prompts and project context on your device before requests are sent to the AI model provider you configure. Optional product analytics, when enabled, can be turned off in the product."
	case "LEGAL_PRIVACY_SEC_RIGHTS_HEADING":
		return "Your rights"
	case "LEGAL_PRIVACY_SEC_RIGHTS_BODY":
		return "Depending on where you live, you may request access, correction, deletion, or export of personal account data. The dashboard may provide self-serve tools. Billing records required by law may be retained as needed."
	case "LEGAL_PRIVACY_SEC_RIGHTS_CONTACT_LEAD":
		return "Contact"
	case "LEGAL_PRIVACY_SEC_RIGHTS_CONTACT_TRAIL":
		return "for privacy requests."
	case "LEGAL_TERMS_INTRO":
		return "These Terms of Service govern access to and use of Trim's hosted websites, dashboard, API, CLI authentication, and related services at use-trim.com. Open-source components remain subject to their repository licenses."
	case "LEGAL_TERMS_SEC_ACCOUNTS_HEADING":
		return "Accounts"
	case "LEGAL_TERMS_SEC_ACCOUNTS_BODY":
		return "You must sign in with an allowed identity provider. You are responsible for activity under your account. We may suspend accounts that violate these Terms or present risk to the Service or other users."
	case "LEGAL_TERMS_SEC_BILLING_HEADING":
		return "Plans and billing"
	case "LEGAL_TERMS_SEC_BILLING_BODY":
		return "Plan prices and allowances are shown in the product at purchase time. Payments are handled by our payment partner as Merchant of Record. Subscriptions renew until canceled. Exhausted quotas may block paid features until you upgrade or purchase additional capacity."
	case "LEGAL_TERMS_SEC_USE_HEADING":
		return "Acceptable use"
	case "LEGAL_TERMS_SEC_USE_BODY":
		return "You may not misuse the Service, bypass metering or authentication, attack the Service without authorization, distribute malware, infringe others' rights, or resell access except as expressly allowed by your plan. We may require clients to meet minimum version requirements for security reasons."
	case "LEGAL_TERMS_SEC_DISCLAIMER_HEADING":
		return "Disclaimer"
	case "LEGAL_TERMS_SEC_DISCLAIMER_BODY":
		return "Context optimization changes payloads before they reach upstream models. The Service is provided as available without warranties beyond those required by law. We do not guarantee particular savings or model quality outcomes."
	case "LEGAL_TERMS_SEC_CONTACT_HEADING":
		return "Contact"
	case "LEGAL_TERMS_SEC_CONTACT_LEAD":
		return "Questions:"
	case "LEGAL_TERMS_SEC_CONTACT_TRAIL":
		return "."
	case "INVITE_OPEN_TEAM":
		return "Open team"
	case "INVITE_EYEBROW":
		return "Workspace invite"
	case "INVITE_STATUS_TITLE":
		return "Invite status"
	case "INVITE_BODY":
		return "Sign in with the invited email using an allowed social provider, then accept."
	case "INVITE_ROLE_PREFIX":
		return "You were invited as"
	case "INVITE_FOR_PREFIX":
		return "for"
	case "INVITE_STATUS_PREFIX":
		return "Status:"
	case "INVITE_EXPIRES_PREFIX":
		return "Expires:"
	case "EVENTS_EMPTY":
		return "No traces yet. Run trim start while logged in, or route Cursor through the cloud gateway."
	case "PREFERENCES_NOTE":
		return "Deep Mode runs on your machine for live IDE proxy (trim start) and trim compress. Engines are not loaded in Trim cloud; auth and quotas still use Trim cloud when licensed."
	case "PREFERENCES_SAVED":
		return "Saved. On your machine run: trim config sync"
	case "PREFERENCES_PAGE_TITLE":
		return "Compression defaults"
	case "PREFERENCES_PAGE_DESCRIPTION":
		return "When Deep Mode is on, the live IDE proxy (trim start) runs Fast then Deep (LLMLingua) using your engine and target_token from this page, subject to billing live_deep_min_input_tokens. Off = Fast only. Trim cloud never loads Deep engines; auth, quotas, and billing still use Trim cloud."
	case "DOCS_FAST_VS_DEEP_LIVE":
		return "Live proxy runs Fast Mode on every turn. When account preferences set compression_tier=deep, the local proxy also runs Deep Mode (LLMLingua) after Fast when input tokens meet billing live_deep_min_input_tokens (0 = all turns), unless live_deep_skip_on_stream skips Deep on stream=true. Models and OOM policy come from billing settings via trim config sync."
	case "PREFERENCES_DEEP_HINT":
		return "On = Deep Mode for live IDE proxy (trim start) and trim compress. Live Deep also respects billing min input tokens and OOM policy (synced). Off = Fast Mode only on the live proxy and for compress defaults. Engine and target apply whenever Deep is on. Run trim config sync after saving."
	case "PREFERENCES_ENGINE_HINT":
		return "Default selected: LLMLingua-2 (v2). Pick long (needs a question) or v1 if you prefer. Applies to live proxy Deep and trim compress when Deep Mode is on."
	case "PREFERENCES_TARGET_HINT":
		return "Target token budget for Deep Mode (live proxy and trim compress). Only applies when Deep Mode is on."
	case "PREFERENCES_BACK_LABEL":
		return "Back to dashboard"
	case "WORKSPACE_BACK_LABEL":
		return "Back to dashboard"
	case "WORKSPACE_PAGE_EYEBROW":
		return "Team"
	case "WORKSPACE_PAGE_TITLE":
		return "Workspaces"
	case "WORKSPACE_PAGE_DESCRIPTION":
		return "Shared seats and pooled credits. Invite by email; invitees sign in with an allowed social provider using that address and accept the link."
	case "WORKSPACE_LIST_TITLE":
		return "Your workspaces"
	case "WORKSPACE_NAME_PLACEHOLDER":
		return "Workspace name"
	case "WORKSPACE_NAME_LABEL":
		return "Workspace name"
	case "WORKSPACE_NAME_DESC":
		return "Shown to members and on invite emails."
	case "WORKSPACE_SEARCH":
		return "Search workspaces"
	case "WORKSPACE_SEARCH_DESC":
		return "Filters by name, plan, role, or workspace id as you type."
	case "WORKSPACE_MEMBERS_SEARCH":
		return "Search members"
	case "WORKSPACE_MEMBERS_SEARCH_DESC":
		return "Filters by email, name, or role as you type."
	case "WORKSPACE_MEMBERS_EMPTY":
		return "No members match this search."
	case "WORKSPACE_INVITES_SEARCH":
		return "Search invites"
	case "WORKSPACE_INVITES_SEARCH_DESC":
		return "Filters by email, role, or status as you type."
	case "WORKSPACE_MEMBERS_TITLE":
		return "Members"
	case "WORKSPACE_INVITES_TITLE":
		return "Pending invites"
	case "WORKSPACE_INVITE_EMAIL_PLACEHOLDER":
		return "colleague@company.com"
	case "WORKSPACE_INVITE_EMAIL_LABEL":
		return "Invite email"
	case "WORKSPACE_INVITE_EMAIL_DESC":
		return "Invitee must sign in with an allowed social provider using this exact email."
	case "WORKSPACE_INVITE_EMAIL_SUBJECT_NAMED_FMT":
		return "You are invited to Trim workspace %s"
	case "WORKSPACE_INVITE_EMAIL_SUBJECT_FALLBACK":
		return "You are invited to a Trim workspace"
	case "WORKSPACE_INVITE_EMAIL_BODY_FMT":
		return "You have been invited to join a Trim workspace as %s.\n\nOpen this link to accept (sign in with the same email address):\n\n%s\n\nThis invite expires at %s (UTC).\n"
	case "WORKSPACE_INVITE_MSG_SHARE":
		return "Share this invite link. The invitee signs in with an allowed social provider using this email, then accepts."
	case "WORKSPACE_INVITE_MSG_SENT":
		return "Invite email sent. The invitee can also use the invite_url if the email is delayed."
	case "WORKSPACE_INVITE_MSG_SMTP_FAILED":
		return "invite email could not be sent; share the invite_url manually"
	case "WORKSPACE_INVITE_MSG_SMTP_OFF":
		return "SMTP is not configured. Share the invite_url with the invitee."
	case "AUTH_GITHUB_OAUTH_SCOPES":
		return "read:user user:email"
	case "AUTH_PROVIDER_DISPLAY_GOOGLE":
		return "Google"
	case "AUTH_PROVIDER_DISPLAY_GITHUB":
		return "GitHub"
	case "AUTH_PROVIDER_DISPLAY_GITLAB":
		return "GitLab"
	case "WORKSPACE_SELECT_HINT":
		return "Select a workspace to manage members and invites."
	case "WORKSPACE_EMPTY":
		return "No workspaces yet. Create one to invite teammates."
	case "WORKSPACE_INVITE_URL_LABEL":
		return "Invite link"
	case "WORKSPACE_INVITES_EMPTY":
		return "No pending invites."
	case "PREFERENCES_ACCOUNT_TITLE":
		return "Signed-in account"
	case "PREFERENCES_AUTH_PROVIDER_PREFIX":
		return "Signed in with"
	case "ACCOUNT_AUTH_PROVIDER_MISSING":
		return "Signed-in account provider is missing from your profile."
	case "PREFERENCES_TIER_TITLE":
		return "Default tier"
	case "PREFERENCES_DEEP_LABEL":
		return "Deep Mode"
	case "PREFERENCES_ENGINE_LABEL":
		return "Deep engine"
	case "PREFERENCES_TARGET_LABEL":
		return "Deep target tokens"
	case "PREFERENCES_CLI_TITLE":
		return "CLI usage"
	case "PREFERENCES_CLI_HELP_1":
		return "trim compress file.go uses Fast Mode by default."
	case "PREFERENCES_CLI_HELP_2":
		return "trim compress docs.txt --mode deep --engine v2"
	case "PREFERENCES_CLI_HELP_3":
		return "trim compress --bootstrap installs Deep Mode dependencies on your machine (may download packages)."
	case "PREFERENCES_CLI_HELP_4":
		return "trim config set default-tier deep"
	case "PREFERENCES_CLI_HELP_5":
		return "trim config sync copies this dashboard profile into ~/.config/trim/preferences.json."
	case "PREFERENCES_CLI_HELP_6":
		return "trim autostart enable|disable|status - everyday proxy auto-start with IDE (syncs preference)."
	case "PREFERENCES_AUTO_START_TITLE":
		return "Start Trim with your IDE"
	case "PREFERENCES_AUTO_START_LABEL":
		return "Starts local Trim when this IDE opens"
	case "PREFERENCES_AUTO_START_HINT":
		return "Keeps the local Trim proxy ready while you work (structural Fast path on the proxy - not Deep/ML). Uncheck anytime. The browser cannot start Trim by itself - CLI, daemon, or IDE extension apply this setting on your machine."
	case "PREFERENCES_AUTO_START_NOTE":
		return "This preference syncs to your device via trim config sync. Local enforcers: trim autostart, trim daemon, and the Trim IDE extension."
	case "DEFAULT_AUTO_START_WITH_IDE":
		return "true"
	case "DEFAULT_AUTO_START_WITH_IDE_MISSING":
		return "DEFAULT_AUTO_START_WITH_IDE is not configured in site_messages"
	case "DEFAULT_AUTO_START_WITH_IDE_INVALID":
		return "DEFAULT_AUTO_START_WITH_IDE must be true or false"
	case "CLI_AUTOSTART_ENABLED":
		return "Auto-start with IDE: on"
	case "CLI_AUTOSTART_DISABLED":
		return "Auto-start with IDE: off"
	case "CLI_AUTOSTART_UNSET":
		return "Auto-start with IDE: unset (fail-closed; run trim config sync or trim autostart enable)"
	case "CLI_AUTOSTART_ENABLE_HINT":
		return "Enable with: trim autostart enable"
	case "CLI_AUTOSTART_DISABLE_HINT":
		return "Disable with: trim autostart disable"
	case "CLI_AUTOSTART_ENABLED_OK":
		return "Auto-start with IDE enabled. Daemon install attempted when supported."
	case "CLI_AUTOSTART_DISABLED_OK":
		return "Auto-start with IDE disabled. Daemon uninstall attempted when supported."
	case "CLI_AUTOSTART_DAEMON_HINT":
		return "OS login start uses trim daemon install/uninstall. Extension attaches to a healthy local proxy."
	case "CLI_AUTOSTART_DAEMON_SKIPPED_OFF":
		return "Auto-start with IDE is off - daemon run exiting without starting proxy"
	case "CLI_AUTOSTART_DAEMON_SKIPPED_UNSET":
		return "Auto-start preference unset - fail-closed, daemon run exiting"
	case "CLI_CONFIG_GET_AUTOSTART_FMT":
		return "auto-start-with-ide=%v"
	case "CLI_HELP_AUTOSTART_SHORT":
		return "Auto-start proxy with your IDE (synced preference)"
	case "CLI_HELP_AUTOSTART_STATUS_SHORT":
		return "Show whether auto-start with IDE is on"
	case "CLI_HELP_AUTOSTART_ENABLE_SHORT":
		return "Enable auto-start and install OS login daemon when supported"
	case "CLI_HELP_AUTOSTART_DISABLE_SHORT":
		return "Disable auto-start and uninstall OS login daemon when supported"
	case "LOCAL_AGENT_ONLINE_WITHIN_SEC":
		return "900"
	case "LOCAL_AGENT_ONLINE_WITHIN_SEC_MISSING":
		return "LOCAL_AGENT_ONLINE_WITHIN_SEC is not configured in site_messages"
	case "LOCAL_AGENT_ONLINE_WITHIN_SEC_INVALID":
		return "LOCAL_AGENT_ONLINE_WITHIN_SEC must be a positive integer (seconds)"
	case "PREFERENCES_LOCAL_AGENT_TITLE":
		return "Local agent"
	case "PREFERENCES_LOCAL_AGENT_ONLINE":
		return "Online (device seen recently)"
	case "PREFERENCES_LOCAL_AGENT_OFFLINE":
		return "Offline (no recent device activity)"
	case "PREFERENCES_LOCAL_AGENT_UNKNOWN":
		return "Unknown (no registered devices yet)"
	case "PREFERENCES_LOCAL_AGENT_HINT":
		return "Status uses last cloud heartbeat from a registered CLI or IDE device. The browser cannot probe localhost; open your IDE or run trim start on the machine to refresh."
	case "PREFERENCES_LOCAL_AGENT_UNAVAILABLE":
		return "Local agent status chrome is incomplete in site_messages"
	case "IDE_AUTOSTART_ENSURING":
		return "Ensuring local Trim proxy is running…"
	case "IDE_AUTOSTART_PROXY_OK":
		return "Local Trim proxy is healthy"
	case "IDE_AUTOSTART_PROXY_STARTED":
		return "Started local Trim proxy"
	case "IDE_AUTOSTART_PROXY_FAILED_FMT":
		return "Could not start local Trim proxy (%s)"
	case "IDE_AUTOSTART_SKIPPED_OFF":
		return "Auto-start with IDE is off - proxy not started by extension"
	case "IDE_AUTOSTART_SKIPPED_UNSET":
		return "Auto-start preference unset - fail-closed, proxy not started"
	case "IDE_AUTOSTART_CLI_MISSING":
		return "trim CLI not found on PATH - install Trim CLI for auto-start"
	case "IDE_PROXY_HEALTH_URL":
		return "http://127.0.0.1:8888/health"
	case "IDE_AUTOSTART_SETTLE_MS":
		return "1500"
	case "IDE_AUTOSTART_STOP_ON_QUIT":
		return "false"
	case "IDE_PROXY_SHUTDOWN_URL":
		return "http://127.0.0.1:8888/v1/control/shutdown"
	case "IDE_AUTOSTART_STATUS_OK":
		return "Trim · proxy ready"
	case "IDE_AUTOSTART_STATUS_FAILED":
		return "Trim · proxy unavailable"
	case "IDE_AUTOSTART_WARN_FAILED":
		return "Trim could not start the local proxy. Check Output → Trim. The IDE was not blocked."
	case "IDE_AUTOSTART_SKIPPED_LOCAL_OFF":
		return "Auto-start overridden off in Trim extension settings"
	case "IDE_AUTOSTART_SKIPPED_MANAGED":
		return "Auto-start skipped by managed policy (DO_NOT_TRACK / TRIM_AUTOSTART_DISABLED / autostart.off)"
	case "IDE_AUTOSTART_SHUTDOWN_TIMEOUT_MS":
		return "5000"
	case "IDE_AUTOSTART_PREF_POLL_MS":
		return "60000"
	case "IDE_AUTOSTART_PREF_POLL_MS_INVALID":
		return "IDE_AUTOSTART_PREF_POLL_MS must be 0 or within AUTOSTART_PREF_POLL_MIN_MS..AUTOSTART_PREF_POLL_MAX_MS"
	case "CLI_AUTOSTART_PREF_POLL_MS":
		return "60000"
	case "CLI_AUTOSTART_PREF_POLL_MS_INVALID":
		return "CLI_AUTOSTART_PREF_POLL_MS must be 0 or within AUTOSTART_PREF_POLL_MIN_MS..AUTOSTART_PREF_POLL_MAX_MS"
	case "CLI_AUTOSTART_ENFORCER_STOPPED":
		return "Auto-start preference is off - daemon enforcer stopping local proxy"
	case "AUTOSTART_PREF_POLL_MIN_MS":
		return "15000"
	case "AUTOSTART_PREF_POLL_MAX_MS":
		return "600000"
	case "AUTOSTART_PREF_POLL_BOUNDS_INVALID":
		return "AUTOSTART_PREF_POLL_MIN_MS/MAX_MS must be positive integers with min <= max"
	case "AUTOSTART_TIMEOUT_MIN_MS":
		return "1"
	case "AUTOSTART_TIMEOUT_MAX_MS":
		return "120000"
	case "AUTOSTART_TIMEOUT_BOUNDS_INVALID":
		return "AUTOSTART_TIMEOUT_MIN_MS/MAX_MS must be positive integers with min <= max"
	case "DAEMON_RESTART_MIN_SEC":
		return "1"
	case "DAEMON_RESTART_MAX_SEC":
		return "3600"
	case "DAEMON_RESTART_BOUNDS_INVALID":
		return "DAEMON_RESTART_MIN_SEC/MAX_SEC must be positive integers with min <= max"
	case "IDE_AUTOSTART_HEALTH_AFTER_START_FAILED":
		return "health check failed after start"
	case "IDE_HTTP_TIMEOUT_MIN_SEC":
		return "1"
	case "IDE_HTTP_TIMEOUT_MAX_SEC":
		return "120"
	case "IDE_AUTO_FLUSH_MIN_SEC":
		return "0"
	case "IDE_AUTO_FLUSH_MAX_SEC":
		return "600"
	case "IDE_AUTO_FLUSH_MIN_ENABLED_SEC":
		return "15"
	case "IDE_HTTP_TIMEOUT_INVALID":
		return "trim.httpTimeoutSec outside IDE_HTTP_TIMEOUT_MIN_SEC..MAX_SEC (fail-closed)"
	case "IDE_AUTO_FLUSH_INVALID":
		return "trim.autoFlushSeconds outside IDE_AUTO_FLUSH bounds (fail-closed)"
	case "IDE_CONFIG_TRACK_EDITS_MISSING":
		return "trim.trackDocumentEdits missing (fail-closed)"
	case "IDE_CONFIG_MIN_LINES_INVALID":
		return "trim.minLinesForAiHeuristic invalid (fail-closed)"
	case "IDE_CONFIG_API_URL_EMPTY":
		return "trim.apiUrl empty - set Settings → Trim → API URL (fail-closed)"
	case "IDE_CONFIG_API_URL_INVALID":
		return "trim.apiUrl must be an absolute http(s) URL (fail-closed)"
	case "CLI_PROXY_HEALTH_TIMEOUT_MS":
		return "2000"
	case "CLI_PROXY_FALLBACK_UNCOMPRESSED":
		return "true"
	case "CLI_PROXY_FALLBACK_UNCOMPRESSED_INVALID":
		return "CLI_PROXY_FALLBACK_UNCOMPRESSED must be true or false"
	case "PROXY_ACTIVE_FILE_PROTECTION":
		return "true"
	case "PROXY_ACTIVE_FILE_PROTECTION_INVALID":
		return "PROXY_ACTIVE_FILE_PROTECTION must be true or false"
	case "PROXY_OPTIONS_REFRESH_MS":
		return "5000"
	case "PROXY_OPTIONS_REFRESH_MS_INVALID":
		return "PROXY_OPTIONS_REFRESH_MS must be a positive integer"
	case "RUNTIME_OVERRIDES_CACHE_MS":
		return "5000"
	case "RUNTIME_OVERRIDES_CACHE_MS_INVALID":
		return "RUNTIME_OVERRIDES_CACHE_MS must be a positive integer"
	case "CLI_PROXY_SHUTDOWN_TIMEOUT_MS":
		return "5000"
	case "CLI_HTTP_SHUTDOWN_TIMEOUT_MS":
		return "5000"
	case "CLI_AUTOSTART_SKIPPED_MANAGED":
		return "Auto-start skipped (TRIM_AUTOSTART_DISABLED=1 or local autostart.off)"
	case "CLI_AUTOSTART_SKIPPED_DNT":
		return "Auto-start skipped (DO_NOT_TRACK=1)"
	case "CLI_AUTOSTART_MANAGED_OFF_HINT":
		return "Managed off: unset TRIM_AUTOSTART_DISABLED and remove ~/.config/trim/autostart.off, then trim autostart enable"
	case "CLI_STOP_ON_DISABLE_OK":
		return "Requested stop of running local proxy after auto-start disable"
	case "CLI_DAEMON_RESTART_SEC":
		return "3"
	case "CLI_CONFIG_GET_AUTOSTART_UNSET":
		return "unset"
	case "CLI_AUTOSTART_SYNC_DAEMON_OK":
		return "Local daemon install/uninstall aligned with synced auto-start preference."
	case "CLI_HELP_STOP_SHORT":
		return "Stop the local Trim proxy"
	case "CLI_STOP_OK":
		return "Local Trim proxy stop requested"
	case "CLI_STOP_FAILED_FMT":
		return "Could not stop local Trim proxy (%s)"
	case "CLI_STOP_URL_MISSING":
		return "Proxy shutdown URL is not configured (site_messages IDE_PROXY_SHUTDOWN_URL / CLI chrome)"
	case "PREFERENCES_KEYS_TITLE":
		return "API keys"
	case "ENGINE_V2_LABEL":
		return "LLMLingua-2 (v2) · recommended"
	case "ENGINE_LONG_LABEL":
		return "LongLLMLingua (long)"
	case "ENGINE_V1_LABEL":
		return "LLMLingua (v1)"
	case "ENGINE_V2_HINT":
		return "Usually faster and lighter. Best default for most Deep compress jobs on your machine."
	case "ENGINE_LONG_HINT":
		return "Question-aware ranking for long documents. Requires a question (CLI: --question)."
	case "ENGINE_V1_HINT":
		return "Classic LLMLingua. Strong general compression when you want the original engine."
	case "RECEIPTS_EMPTY":
		return "No receipts yet. Completed Paddle transactions appear here."
	case "PLANS_ANNUAL_SAVINGS_FMT":
		return "Save %d%%"
	case "DASHBOARD_PAGE_EYEBROW":
		return "Dashboard"
	case "DASHBOARD_WELCOME_PREFIX":
		return "Welcome,"
	case "DASHBOARD_WELCOME_GUEST":
		return "Welcome to Trim"
	case "DASHBOARD_PAGE_DESCRIPTION":
		return "Usage, traces, and receipts from your Trim cloud account."
	case "DASHBOARD_STATUS_PREFIX":
		return "Status:"
	case "DASHBOARD_TIER_PREFIX":
		return "Tier:"
	case "DASHBOARD_INTERVAL_PREFIX":
		return "Interval:"
	case "DASHBOARD_RENEWS_PREFIX":
		return "Renews / ends:"
	case "DASHBOARD_NAV_TEAM":
		return "Team"
	case "DASHBOARD_NAV_SETTINGS":
		return "Settings"
	case "DASHBOARD_METRIC_PLAN":
		return "Plan"
	case "DASHBOARD_METRIC_CREDITS":
		return "Credits used"
	case "DASHBOARD_METRIC_REMAINING":
		return "Remaining"
	case "DASHBOARD_METRIC_TOKENS_SAVED":
		return "Tokens saved (30d)"
	case "DASHBOARD_METRIC_TAB":
		return "Tab acceptance"
	case "DASHBOARD_METRIC_LINES_ADDED":
		return "AI lines added"
	case "DASHBOARD_METRIC_LINES_DELETED":
		return "AI lines deleted"
	case "DASHBOARD_ACCEPTANCE_HINT":
		return "(IDE ingest when available)"
	case "DASHBOARD_CHART_TOKEN_SERIES":
		return "Token savings (30 day series)"
	case "DASHBOARD_CHART_MODELS":
		return "Model breakdown"
	case "DASHBOARD_CHART_OUTCOMES":
		return "Request outcomes"
	case "DASHBOARD_CHART_MODES":
		return "Compression modes"
	case "DASHBOARD_TRACES_TITLE":
		return "Recent traces"
	case "DASHBOARD_RECEIPTS_TITLE":
		return "Receipts"
	case "DASHBOARD_ACCOUNT_TITLE":
		return "Account"
	case "DASHBOARD_AVATAR_HINT":
		return "Optionally re-host your OAuth avatar on Cloudinary when CLOUDINARY_* is configured on the API."
	case "DASHBOARD_DELETE_HINT":
		return "Delete your Trim account and personal data (GDPR right to be forgotten). Billing ledger rows stay anonymized for tax records."
	case "DASHBOARD_SIGNED_IN_PREFIX":
		return "Signed in as"
	case "DASHBOARD_EVENTS_SUFFIX":
		return "cloud events"
	case "DASHBOARD_META_SEP":
		return " · "
	case "DASHBOARD_USAGE_TITLE":
		return "Your Usage"
	case "DASHBOARD_USAGE_SUBTITLE":
		return "Your usage per day across this billing period"
	case "DASHBOARD_USAGE_GROUP_BY_PREFIX":
		return "Group By:"
	case "DASHBOARD_USAGE_GROUP_MODEL":
		return "Model"
	case "DASHBOARD_USAGE_GROUP_MODE":
		return "Mode"
	case "DASHBOARD_USAGE_Y_AXIS":
		return "Cumulative Tokens"
	case "DASHBOARD_USAGE_TODAY":
		return "Today"
	case "DASHBOARD_USAGE_EMPTY":
		return "No usage recorded in this period yet. Charts fill in after Trim records proxy events from the CLI or cloud gateway."
	case "DASHBOARD_USAGE_TOOLTIP_BREAKDOWN":
		return "Daily breakdown"
	case "DASHBOARD_USAGE_TOOLTIP_DAILY_TOTAL":
		return "Daily total"
	case "DASHBOARD_USAGE_TOOLTIP_CUMULATIVE_TOTAL":
		return "Cumulative total"
	case "DASHBOARD_USAGE_TOOLTIP_SHARE_FMT":
		return "{pct}%"
	case "DASHBOARD_HEATMAP_TITLE":
		return "AI Line Edits"
	case "DASHBOARD_HEATMAP_SCOPE_ALL":
		return "All"
	case "DASHBOARD_HEATMAP_SCOPE_TAB":
		return "Tab"
	case "DASHBOARD_HEATMAP_EMPTY_FMT":
		return "{date}\nNo lines edited"
	case "DASHBOARD_HEATMAP_VALUE_FMT":
		return "{date}\n{count} lines edited"
	case "DASHBOARD_HEATMAP_EMPTY_FMT_ALL":
		return "{date}\nNo lines edited"
	case "DASHBOARD_HEATMAP_VALUE_FMT_ALL":
		return "{date}\n{count} lines edited"
	case "DASHBOARD_HEATMAP_EMPTY_FMT_TAB":
		return "{date}\nNo tab accepts"
	case "DASHBOARD_HEATMAP_VALUE_FMT_TAB":
		return "{date}\n{count} tab accepts"
	case "DASHBOARD_HEATMAP_WD_MON":
		return "M"
	case "DASHBOARD_HEATMAP_WD_WED":
		return "W"
	case "DASHBOARD_HEATMAP_WD_FRI":
		return "F"
	case "DASHBOARD_HEATMAP_STAT_MOST_ACTIVE_MONTH":
		return "Most Active Month"
	case "DASHBOARD_HEATMAP_STAT_MOST_ACTIVE_DAY":
		return "Most Active Day"
	case "DASHBOARD_HEATMAP_STAT_LONGEST_STREAK":
		return "Longest Streak"
	case "DASHBOARD_HEATMAP_STAT_CURRENT_STREAK":
		return "Current Streak"
	case "DASHBOARD_HEATMAP_STREAK_FMT":
		return "{count}d"
	case "ADMIN_USAGE_TITLE":
		return "Platform Usage"
	case "ADMIN_USAGE_SUBTITLE":
		return "Platform usage per day across this window"
	case "ADMIN_USAGE_GROUP_BY_PREFIX":
		return "Group By:"
	case "ADMIN_USAGE_GROUP_MODEL":
		return "Model"
	case "ADMIN_USAGE_GROUP_MODE":
		return "Mode"
	case "ADMIN_USAGE_Y_AXIS":
		return "Cumulative Tokens"
	case "ADMIN_USAGE_TODAY":
		return "Today"
	case "ADMIN_USAGE_EMPTY":
		return "No platform usage recorded in this window yet. Charts fill in after Trim records proxy events."
	case "ADMIN_USAGE_TOOLTIP_BREAKDOWN":
		return "Daily breakdown"
	case "ADMIN_USAGE_TOOLTIP_DAILY_TOTAL":
		return "Daily total"
	case "ADMIN_USAGE_TOOLTIP_CUMULATIVE_TOTAL":
		return "Cumulative total"
	case "ADMIN_USAGE_TOOLTIP_SHARE_FMT":
		return "{pct}%"
	case "ADMIN_HEATMAP_TITLE":
		return "AI Line Edits"
	case "ADMIN_HEATMAP_SCOPE_ALL":
		return "All"
	case "ADMIN_HEATMAP_SCOPE_TAB":
		return "Tab"
	case "ADMIN_HEATMAP_EMPTY_FMT":
		return "{date}\nNo lines edited"
	case "ADMIN_HEATMAP_VALUE_FMT":
		return "{date}\n{count} lines edited"
	case "ADMIN_HEATMAP_EMPTY_FMT_ALL":
		return "{date}\nNo lines edited"
	case "ADMIN_HEATMAP_VALUE_FMT_ALL":
		return "{date}\n{count} lines edited"
	case "ADMIN_HEATMAP_EMPTY_FMT_TAB":
		return "{date}\nNo tab accepts"
	case "ADMIN_HEATMAP_VALUE_FMT_TAB":
		return "{date}\n{count} tab accepts"
	case "ADMIN_HEATMAP_WD_MON":
		return "M"
	case "ADMIN_HEATMAP_WD_WED":
		return "W"
	case "ADMIN_HEATMAP_WD_FRI":
		return "F"
	case "ADMIN_HEATMAP_STAT_MOST_ACTIVE_MONTH":
		return "Most Active Month"
	case "ADMIN_HEATMAP_STAT_MOST_ACTIVE_DAY":
		return "Most Active Day"
	case "ADMIN_HEATMAP_STAT_LONGEST_STREAK":
		return "Longest Streak"
	case "ADMIN_HEATMAP_STAT_CURRENT_STREAK":
		return "Current Streak"
	case "ADMIN_HEATMAP_STREAK_FMT":
		return "{count}d"
	case "AVATAR_SYNC_FAILED":
		return "Avatar sync failed."
	case "WORKSPACE_META_SEP":
		return " · "
	case "WORKSPACE_META_MEMBERS_UNIT":
		return " members"
	case "WORKSPACE_META_CREDITS_UNIT":
		return " credits"
	case "WORKSPACE_INVITE_EXPIRES_PREFIX":
		return "expires "
	case "ENTERPRISE_DIALOG_TITLE":
		return "Request Enterprise"
	case "ENTERPRISE_DIALOG_DESCRIPTION":
		return "Tell us about your team. We store this inquiry and follow up on your signed-in email."
	case "ENTERPRISE_EMAIL_FALLBACK":
		return "your account email"
	case "ENTERPRISE_COMPANY_LABEL":
		return "Company name"
	case "ENTERPRISE_SEATS_LABEL":
		return "Estimated seats"
	case "ENTERPRISE_MESSAGE_LABEL":
		return "What do you need?"
	case "ENTERPRISE_COMPANY_PLACEHOLDER":
		return "Acme Engineering"
	case "ENTERPRISE_SEATS_PLACEHOLDER":
		return "e.g. 25"
	case "ENTERPRISE_MESSAGE_PLACEHOLDER":
		return "SSO, dedicated gateway, seat count, procurement timeline..."
	case "ENTERPRISE_COMPANY_DESC":
		return "Legal or trade name we should use when following up."
	case "ENTERPRISE_MESSAGE_DESC":
		return "Include SSO, seat count, timeline, or procurement needs so sales can respond."
	case "ENTERPRISE_CANCEL":
		return "Cancel"
	case "ENTERPRISE_SEND":
		return "Send inquiry"
	case "PLANS_LOAD_FAILED":
		return "Failed to load plans"
	case "PRORATION_REVIEW_HINT":
		return "Your bill will be adjusted for unused time on your current plan. Review the totals below before confirming."
	case "RECEIPTS_SYNC_FAILED":
		return "Receipt sync failed."
	case "PLANS_SEAT_HINT":
		return "Used for per-seat team plans"
	case "PLANS_SEAT_LABEL":
		return "Team seats"
	case "PLANS_PER_SEAT_ONE_FMT":
		return "per seat%s 1 seat"
	case "PLANS_PER_SEAT_MANY_FMT":
		return "per seat%s %d seats"
	case "PLANS_PERIOD_YEAR":
		return "/ year"
	case "PLANS_PERIOD_MONTH":
		return "/ month"
	case "UPGRADE_CONFIRM_TITLE":
		return "Confirm upgrade"
	case "UPGRADE_PRORATION_MODE_FMT":
		return "Your bill will be adjusted for unused time on your current plan. Review the totals below before confirming."
	case "UPGRADE_PLAN_PREFIX":
		return "Plan:"
	case "PLANS_DIALOG_TITLE":
		return "Choose your Trim plan"
	case "ACTIVE_PLAN_PREFIX":
		return "Active plan:"
	case "ACTIVE_PLAN_EXPIRES_FMT":
		return ". Expires %s"
	case "PRORATION_PREFIX":
		return "Proration:"
	case "UPGRADE_SEATS_FMT":
		return "%d seats"
	case "UPGRADE_CREDIT_PREFIX":
		return "Credit:"
	case "UPGRADE_SUBTOTAL_PREFIX":
		return "Subtotal:"
	case "UPGRADE_TAX_PREFIX":
		return "Tax:"
	case "UPGRADE_DUE_PREFIX":
		return "Due now:"
	case "UPGRADE_PREVIEW_EMPTY":
		return "Paddle did not return immediate totals for this preview."
	case "UPGRADE_CANCEL":
		return "Cancel"
	case "EVENTS_COL_WHEN":
		return "When"
	case "EVENTS_COL_MODEL":
		return "Model"
	case "EVENTS_COL_MODE":
		return "Mode"
	case "EVENTS_COL_TOKENS":
		return "Tokens"
	case "EVENTS_COL_LATENCY":
		return "Latency"
	case "EVENTS_COL_STATUS":
		return "Status"
	case "EVENTS_COL_REQUEST_ID":
		return "Request id"
	case "EVENTS_COL_VIEW":
		return "View"
	case "EVENTS_OPEN_LABEL":
		return "View"
	case "EVENTS_PREVIEW_FIELD_DESC":
		return "Trace field from this run. Use search to find other traces."
	case "EVENTS_TOKENS_SEP":
		return " to "
	case "EVENTS_LATENCY_UNIT":
		return " ms"
	case "EVENTS_DATE_RANGE_PLACEHOLDER":
		return "Filter by date"
	case "EVENTS_DATE_RANGE_DESC":
		return "Limits traces to the selected UTC calendar days. Clear to show all."
	case "EVENTS_DATE_RANGE_CLEAR":
		return "Clear dates"
	case "EVENTS_DATE_RANGE_APPLY":
		return "Done"
	case "EVENTS_DATE_FROM_INVALID":
		return "from must be YYYY-MM-DD"
	case "EVENTS_DATE_TO_INVALID":
		return "to must be YYYY-MM-DD"
	case "EVENTS_DATE_RANGE_ORDER":
		return "from must be on or before to"
	case "EVENTS_SEARCH":
		return "Search traces"
	case "EVENTS_SEARCH_DESC":
		return "Filters by model, mode, status, or request id as you type."
	case "RECEIPTS_COL_DATE":
		return "Date"
	case "RECEIPTS_COL_INVOICE":
		return "Invoice"
	case "RECEIPTS_COL_STATUS":
		return "Status"
	case "RECEIPTS_COL_TOTAL":
		return "Total"
	case "RECEIPTS_COL_VIEW":
		return "View"
	case "RECEIPTS_OPEN_LABEL":
		return "Open"
	case "RECEIPTS_SEARCH":
		return "Search receipts"
	case "RECEIPTS_SEARCH_DESC":
		return "Filters by invoice id, status, or bill-to as you type."
	case "RECEIPTS_PREVIEW_FIELD_DESC":
		return "Summary from this receipt. Open the full receipt for line items and PDF."
	case "UPGRADE_CONFIRM_REQUIRED_MSG":
		return "Confirm this upgrade to apply proration and charge the balance due now."
	case "UPGRADE_APPLIED_MSG":
		return "Upgrade applied. Unused time on your current plan was credited via Paddle proration."
	case "CHART_EMPTY_MODELS":
		return "Model breakdown appears after Trim records proxy events."
	case "CHART_EMPTY_STATUS":
		return "Success rate appears after Trim records proxy events."
	case "CHART_EMPTY_MODES":
		return "Compression mode mix appears after Trim records proxy events with mode."
	case "CHART_EMPTY_SERIES":
		return "Charts appear after Trim records proxy events from the CLI or cloud gateway."
	case "DIALOG_CLOSE":
		return "Close"
	case "DIALOG_CANCEL":
		return "Cancel"
	case "CHART_SUCCESS_RATE_PREFIX":
		return "Success rate"
	case "CHART_RUNS_SERIES":
		return "Runs"
	case "CHART_SCOPE_FULL":
		return "full history"
	case "CHART_SCOPE_PAGE":
		return "this page"
	case "CHART_TRACES_UNIT":
		return "traces"
	case "CHART_STATUS_SUCCESS":
		return "Success"
	case "CHART_STATUS_ERROR":
		return "Error"
	case "CHART_RUNS_FMT":
		return "%s runs"
	case "CHART_MODEL_TOOLTIP_FMT":
		return "%s tokens (%s runs, %s saved)"
	case "WORKSPACE_ROLE_OWNER":
		return "Owner"
	case "WORKSPACE_ROLE_ADMIN":
		return "Admin"
	case "WORKSPACE_ROLE_MEMBER":
		return "Member"
	case "PLAN_TIER_FREE":
		return "Free"
	case "PLAN_TIER_PRO":
		return "Pro"
	case "PLAN_TIER_TEAM":
		return "Team"
	case "PLAN_TIER_ENTERPRISE":
		return "Enterprise"
	case "BILLING_INTERVAL_MONTH":
		return "month"
	case "BILLING_INTERVAL_YEAR":
		return "year"
	case "EVENT_MODE_PROXY":
		return "Proxy"
	case "EVENT_MODE_FAST":
		return "Fast"
	case "EVENT_MODE_DEEP":
		return "Deep"
	case "EVENT_MODE_BALANCED":
		return "Balanced"
	case "EVENT_MODE_AGGRESSIVE":
		return "Aggressive"
	case "EVENT_MODE_MILD":
		return "Mild"
	case "EVENT_MODE_CUSTOM":
		return "Custom"
	case "EVENT_MODE_LOCAL_PROXY":
		return "Local proxy"
	case "EVENT_STATUS_SUCCESS":
		return "Success"
	case "EVENT_STATUS_ERROR":
		return "Error"
	case "RECEIPT_STATUS_COMPLETED":
		return "PAID"
	case "RECEIPT_STATUS_REFUNDED":
		return "Refunded"
	case "RECEIPT_STATUS_PAST_DUE":
		return "Past due"
	case "INVITE_STATUS_PENDING":
		return "Pending"
	case "INVITE_STATUS_ACCEPTED":
		return "Accepted"
	case "INVITE_STATUS_EXPIRED":
		return "Expired"
	case "INVITE_STATUS_REVOKED":
		return "Revoked"
	case "CLI_LOGIN_SIGN_IN_FMT":
		return "Sign in with %s in the browser."
	case "CLI_LOGIN_OPENING_BROWSER":
		return "Opening browser for social sign-in..."
	case "CLI_LOGIN_PASTE_KEY":
		return "After you finish in the browser, paste the API key shown on the page:"
	case "CLI_LOGIN_KEY_REQUIRED":
		return "api key is required"
	case "CLI_LOGIN_SUCCESS":
		return "Logged in. Token stored in the OS keychain (or secure credentials file)."
	case "CLI_LOGOUT_SUCCESS":
		return "Logged out. API token removed from the OS keychain."
	case "CLI_LOGOUT_ALREADY":
		return "Already logged out. No API token was stored."
	case "RECEIPT_VAT_ID_PREFIX":
		return "VAT Number"
	case "RECEIPT_PDF_CURRENCY_REQUIRED":
		return "Receipt currency is missing; cannot render PDF amounts."
	case "RECEIPT_PDF_TITLE_REQUIRED":
		return "Receipt document title is missing; cannot render PDF."
	case "RECEIPT_MONEY_LOCALE_REQUIRED":
		return "Receipt money locale is not configured (SITE_HTML_LANG)."
	case "RECEIPT_PDF_MONEY_LOCALE_REQUIRED":
		return "Receipt PDF money locale is missing; cannot format amounts."
	case "CLI_CHROME_UNAVAILABLE":
		return "trim chrome unavailable; ensure the API is reachable (GET /api/v1/public/auth-providers)"
	case "CLI_BROWSER_UNSUPPORTED_FMT":
		return "open browser is not supported on %s"
	case "CLI_STATUS_NOT_LOGGED_IN":
		return "Not logged in. Run: trim login"
	case "CLI_LOGIN_PHRASE_OR":
		return " or "
	case "CLI_LOGIN_PHRASE_COMMA":
		return ", "
	case "CLI_LOGIN_PHRASE_COMMA_OR":
		return ", or "
	case "CLI_SETUP_ENDPOINT_FMT":
		return "Trim local endpoint: %s"
	case "CLI_SETUP_NONE_FOUND":
		return "No IDE settings directories found. Install Cursor, VS Code, Windsurf, or Zed, or set env vars below."
	case "CLI_SETUP_UPDATED_HEADER":
		return "Updated:"
	case "CLI_SETUP_SHELL_HEADER":
		return "Shell / Claude Code / Continue / Aider (same Trim listen host; different paths and env):"
	case "CLI_SETUP_JETBRAINS_HINT":
		return "JetBrains AI Assistant: Settings → Tools → AI Assistant → enable OpenAI-compatible"
	case "CLI_SETUP_API_ENDPOINT_FMT":
		return "  API endpoint: %s"
	case "CLI_SETUP_NEXT_HEADER":
		return "Next steps:"
	case "CLI_SETUP_STEP_START":
		return "1. Run: trim start   (or: trim daemon install)"
	case "CLI_SETUP_STEP_RESTART":
		return "2. Restart your IDE so Models picks up the override"
	case "CLI_SETUP_STEP_LOGIN":
		return "3. Run: trim login"
	case "CLI_SETUP_STEP_DASHBOARD_FMT":
		return "4. Dashboard: %s"
	case "CLI_SETUP_OPTIONAL_TLS":
		return "Optional TLS: trim setup --tls"
	case "CLI_SETUP_TLS_THEN":
		return "Then:"
	case "CLI_SETUP_TLS_GENERATED_FMT":
		return "Generated local TLS materials:\n  cert: %s\nTrust this cert in your OS/IDE trust store, then set TRIM_TLS_CERT and TRIM_TLS_KEY and run trim start --tls."
	case "CLI_SETUP_TLS_CERT_FMT":
		return "  set TRIM_TLS_CERT=%s"
	case "CLI_SETUP_TLS_KEY_FMT":
		return "  set TRIM_TLS_KEY=%s"
	case "CLI_SETUP_TLS_KEY_PATH_FMT":
		return "  key:  %s"
	case "CLI_SETUP_SHELL_OPENAI_FMT":
		return "  export OPENAI_BASE_URL=%q   # Cursor / OpenAI-compat clients (Gemini, GPT, Claude-via-adapter)"
	case "CLI_SETUP_SHELL_ANTHROPIC_FMT":
		return "  export ANTHROPIC_BASE_URL=%q   # Claude Code: Trim listen ORIGIN only (no /v1; Claude Code appends /v1/messages)"
	case "CLI_SETUP_TLS_START":
		return "  trim start"
	case "CLI_SETUP_TLS_POINT_IDE":
		return "  Point IDE base URL to https://localhost:<port>/v1"
	case "CLI_SETUP_TLS_ENV_REQUIRED":
		return "TRIM_TLS_ORG and TRIM_TLS_VALIDITY_DAYS are required for trim setup --tls"
	case "CLI_SETUP_TLS_DAYS_INVALID":
		return "TRIM_TLS_VALIDITY_DAYS must be a positive integer"
	case "CLI_POW_CLIENT_REQUIRED":
		return "HTTP client required (set TRIM_CLI_HTTP_TIMEOUT_SEC)"
	case "CLI_POW_428_MISSING":
		return "server returned 428 without challenge header"
	case "CLI_POW_SOLVE_FAILED":
		return "failed to solve proof-of-work challenge"
	case "CLI_POW_PARSE_FAILED_FMT":
		return "parse pow challenge: %v"
	case "CLI_SETUP_WROTE_FMT":
		return "%s → %s"
	case "CLI_SETUP_PRODUCT_CURSOR":
		return "Cursor"
	case "CLI_SETUP_PRODUCT_VSCODE":
		return "VS Code"
	case "CLI_SETUP_PRODUCT_WINDSURF":
		return "Windsurf"
	case "CLI_SETUP_PRODUCT_CONTINUE":
		return "Continue"
	case "CLI_SETUP_PRODUCT_ZED":
		return "Zed"
	case "CLI_SETUP_PRODUCT_JETBRAINS":
		return "JetBrains hint"
	case "CLI_SETUP_CONTINUE_NO_MODEL":
		return "Continue config has no openai-compatible model with apiBase. Edit ~/.continue/config.yaml (see trim setup Continue block / docs/ide/continue), or set TRIM_SETUP_DEFAULT_MODEL and TRIM_SETUP_DEFAULT_MODEL_TITLE then re-run trim setup."
	case "CLI_START_RUNNING_FMT":
		return "Trim proxy running on %s://localhost:%s"
	case "CLI_START_DASHBOARD_FMT":
		return "Local dashboard: %s://localhost:%s/dashboard"
	case "CLI_START_MODE_FMT":
		return "Compression mode: %s"
	case "CLI_START_POINT_IDE_FMT":
		return "Point IDE Base URL at %s://localhost:%s/v1 (Cursor Override / Continue apiBase). VS Code Chat Custom Endpoint needs …/v1/chat/completions - see docs."
	case "CLI_START_RULES_FMT":
		return "Loaded .tokenignore / .trimrc rules (%d never-trim, %d ignore)"
	case "CLI_START_STOPPED":
		return "Trim proxy stopped"
	case "CLI_ERR_FMT":
		return "error: %v\n"
	case "CLI_PROXY_FAILED_FMT":
		return "proxy failed: %v"
	case "CLI_MAIN_ERR_FMT":
		return "trim: %v\n"
	case "DECIDE_DOWNGRADE_BLOCKED_UNTIL_FMT":
		return "Downgrade is blocked while your current plan is active. You can choose a lower plan after %s."
	case "CLI_DAEMON_UNSUPPORTED_FMT":
		return "daemon %s is not supported on %s"
	case "CLI_DAEMON_LAUNCHD_INSTALLED_FMT":
		return "launchd plist installed: %s"
	case "CLI_DAEMON_LAUNCHD_MISSING":
		return "launchd service not installed. Run: trim daemon install"
	case "CLI_DAEMON_SYSTEMD_INSTALLED_FMT":
		return "systemd user unit installed: %s"
	case "CLI_DAEMON_SYSTEMD_MISSING":
		return "systemd user unit not installed. Run: trim daemon install"
	case "CLI_DAEMON_WINDOWS_MISSING":
		return "Windows scheduled task not installed. Run: trim daemon install"
	case "CLI_DAEMON_LAUNCHD_OK_FMT":
		return "Installed launchd agent: %s"
	case "CLI_DAEMON_LAUNCHD_HINT":
		return "Trim will start on login. Logs: ~/Library/Logs/trim.log"
	case "CLI_DAEMON_LAUNCHD_REMOVED":
		return "Removed launchd agent"
	case "CLI_DAEMON_SYSTEMD_OK_FMT":
		return "Installed systemd user unit: %s"
	case "CLI_DAEMON_SYSTEMD_REMOVED":
		return "Removed systemd user unit"
	case "CLI_DAEMON_WINDOWS_OK":
		return "Installed Windows scheduled task: TrimProxy"
	case "CLI_DAEMON_WINDOWS_REMOVED":
		return "Removed Windows scheduled task: TrimProxy"
	case "CLI_TELEMETRY_ENABLED":
		return "Telemetry: enabled"
	case "CLI_TELEMETRY_DISABLE_HINT":
		return "Disable with: trim telemetry disable"
	case "CLI_TELEMETRY_ENV_HINT":
		return "Or set DO_NOT_TRACK=1 / TRIM_TELEMETRY_DISABLED=1"
	case "CLI_TELEMETRY_DISABLED":
		return "Telemetry: disabled"
	case "CLI_TELEMETRY_ENABLE_HINT":
		return "Enable with: trim telemetry enable"
	case "CLI_TELEMETRY_DISABLED_OK":
		return "Telemetry disabled (~/.config/trim/telemetry.off)"
	case "CLI_TELEMETRY_ENABLED_OK":
		return "Telemetry enabled (respects DO_NOT_TRACK and TRIM_TELEMETRY_DISABLED)"
	case "CLI_COMPRESS_FILE_REQUIRED":
		return "file path required (or use --bootstrap)"
	case "CLI_COMPRESS_MODE_REQUIRED":
		return "set --mode fast|deep, use --deep, or run: trim config set default-tier fast"
	case "CLI_COMPRESS_MODE_INVALID":
		return "mode must be fast or deep"
	case "CLI_TREESITTER_REQUIRED":
		return "This .trimrc custom rule needs the enhanced Trim build. The default install cannot apply those custom queries."
	case "RECEIPT_DOWNLOAD_PDF_FAILED":
		return "Could not download the receipt PDF. Try again or use Print."
	case "RECEIPT_PDF_FILENAME_FMT":
		return "receipt-%s.pdf"
	case "CLI_COMPRESS_DEEP_PREFS_REQUIRED":
		return "Deep Mode requires deep_engine and target_token (run: trim config sync, or trim config set deep-engine / target-token, or pass --engine and --target-token)"
	case "CLI_COMPRESS_DEEP_RESULT_FMT":
		return "Deep (%s): %d -> %d tokens (%s)"
	case "CLI_COMPRESS_WROTE_FMT":
		return "Wrote %s (%s/%s, ~%d -> ~%d chars)"
	case "CLI_STATS_LIVE_HEADER":
		return "Live proxy:"
	case "CLI_STATS_LIVE_OFFLINE_FMT":
		return "Live proxy offline (%v)"
	case "CLI_STATS_URL_FMT":
		return "http://127.0.0.1:{port}/v1/stats"
	case "CLI_STATS_URL_MISSING":
		return "CLI_STATS_URL_FMT must include {port} in site_messages"
	case "BILLING_SETTINGS_UNAVAILABLE":
		return "Billing settings are unavailable"
	case "BILLING_PRICING_UNBOUND":
		return "Billing pricing is not bound yet. An operator must set plan amounts and currency in the admin dashboard and sync the catalog to Paddle."
	case "CONFIG_ENV_NEXT_PUBLIC_API_URL_MISSING":
		return "NEXT_PUBLIC_API_URL is not set"
	case "CLI_STATS_SQLITE_TODAY_FMT":
		return "SQLite today (~/.trim/metrics.db): requests=%d before=%d after=%d saved~$%.4f last_ms=%.1f"
	case "CLI_STATS_SQLITE_SERIES_HEADER":
		return "SQLite last 14 days:"
	case "CLI_STATS_SQLITE_SERIES_ROW_FMT":
		return "  %s  req=%d  %d->%d  ~$%.4f  avg_ms=%.1f"
	case "CLI_STATS_TUI_TIP":
		return "Tip: trim stats --tui   (or: trim tui)"
	case "CLI_STATS_DEEP_STATUS_FMT":
		return "Deep: %s"
	case "CLI_STATS_DEEP_STAGE_FMT":
		return "Deep stage: %s (%.1f%% saved)"
	case "CLI_STATS_LAST_REQUEST_FMT":
		return "Last request (whole body): %s (%.1f%% saved)"
	case "CLI_STATS_DOOR_FMT":
		return "Last door: %s"
	case "CLI_STATS_DASHBOARD_TIP_FMT":
		// Println + {port} replace only (not fmt.Printf) - use a single % so users see "0%".
		return "Local meter: http://127.0.0.1:{port}/dashboard → Show savings detail (0% whole-body Saved on Claude Code chrome is often normal)."
	case "CLI_STATUS_API_FMT":
		return "API status: %s"
	case "CLI_DEEP_BOOTSTRAP_REQS":
		return "Installing Deep Mode dependencies on this machine (not in Trim cloud)."
	case "CLI_DEEP_BOOTSTRAP_PIP":
		return "Installing Deep Mode dependencies on this machine (not in Trim cloud)."
	case "CLI_DEEP_REQUIREMENTS_MISSING":
		return "Deep Mode requirements file not found beside the CLI. Pack it with the release or set TRIM_OPTIMIZER_PY to a tree that includes the requirements file."
	case "CLI_DEEP_AUTO_INSTALL":
		return "Deep Mode runtime not found. Installing locally once..."
	case "CLI_DEEP_BIN_ENV_MISSING_FMT":
		return "TRIM_OPTIMIZER_BIN not found: %s"
	case "CLI_DEEP_BIN_MISSING":
		return "Frozen Deep Mode binary not found (set TRIM_OPTIMIZER_BIN or pack trim-deep next to the CLI)."
	case "CLI_DEEP_PY_ENV_MISSING_FMT":
		return "TRIM_OPTIMIZER_PY not found: %s"
	case "CLI_DEEP_PY_MISSING":
		return "Deep Mode runtime not found beside the CLI (pack it with the release or set TRIM_OPTIMIZER_PY)."
	case "CLI_DEEP_LLM_MISSING_FMT":
		return "Deep Mode dependencies missing (%v): %s; run: trim compress --bootstrap"
	case "CLI_DEEP_PIP_FAILED":
		return "Installing Deep Mode dependencies failed"
	case "CLI_DEEP_TARGET_REQUIRED":
		return "deep_target_token must be a positive integer (set via trim config or preferences)"
	case "CLI_DEEP_ENGINE_REQUIRED":
		return "deep engine required (v1, long, or v2); set via --engine or trim config sync"
	case "CLI_DEEP_QUESTION_REQUIRED":
		return "engine=long requires --question (no invent default)"
	case "CLI_DEEP_OPTIMIZE_FMT":
		return "deep optimize: %s"
	case "CLI_DEEP_TIMEOUT":
		return "deep optimize timed out"
	case "CLI_DEEP_PARSE_FMT":
		return "deep optimize parse: %v (stderr=%s)"
	case "CLI_PROXY_DEEP_FAILED_FMT":
		return "Live Deep Mode failed: %v"
	case "CLI_PROXY_LIVE_DEEP_MODE_FMT":
		return "deep/%s"
	case "CLI_PROXY_LIVE_DEEP_ENABLED_FMT":
		return "Live Deep Mode on (engine %s, target_token %d, min_input_tokens %d, oom=%s, skip_stream=%t). Synced from account preferences / billing settings."
	case "CLI_PROXY_DEEP_COMPACT_STUB":
		return "(prior turns compacted by Deep Mode)"
	case "CLI_PROXY_DEEP_CHROME_REQUIRED":
		return "Live Deep Mode chrome missing from Trim cloud (CLI_PROXY_LIVE_DEEP_MODE_FMT / CLI_PROXY_DEEP_COMPACT_STUB). Sync site_messages and retry."
	case "CLI_PROXY_DEEP_PREFS_REQUIRED":
		return "Live Deep Mode requires compression_tier=deep plus deep_engine and deep_target_token. Save Dashboard → Settings, then run: trim config sync"
	case "CLI_PROXY_DEEP_SKIPPED_MIN_FMT":
		return "Live Deep skipped (input ~%d tokens < min %d). Fast Mode only for this request."
	case "CLI_PROXY_DEEP_SKIPPED_STREAM":
		return "Live Deep skipped (stream=true and billing live_deep_skip_on_stream). Fast Mode only for this request."
	case "CLI_PROXY_DEEP_OOM_SKIPPED":
		return "Live Deep skipped after OOM/memory error (billing live_deep_oom_policy=skip). Fast Mode only for this request."
	case "CLI_PROXY_DEEP_RUNTIME_REQUIRED":
		return "Live Deep requires synced billing Deep models and oom policy (trim config sync). Device map stays local: TRIM_DEEP_DEVICE_MAP."
	case "CLI_PROXY_ADAPTER_REQUIRED":
		return "Provider adapters are not synced. Run trim config sync while logged in, then trim start. Adapters come from Trim cloud (provider_adapters); the proxy does not invent routing."
	case "CLI_PROXY_ADAPTER_UNKNOWN_DIALECT_FMT":
		return "Provider adapter %q has unsupported dialect %q. Update provider_adapters in the database (supported: anthropic_messages, openai_compat)."
	case "CLI_PROXY_ADAPTER_MODEL_REQUIRED":
		return "OpenAI-compat adapter routing requires a non-empty model field on /v1/chat/completions."
	case "CLI_PROXY_ADAPTER_ALIAS_REQUIRED_FMT":
		return "Model %q matched adapter %q but require_alias is on and no model_aliases entry exists. Add an alias in Admin → Provider adapters or use a real upstream model id."
	case "CLI_PROXY_ADAPTER_TRANSLATE_FMT":
		return "Provider adapter %q failed to translate the request: %v"
	case "CLI_PROXY_ADAPTER_AUTH_FMT":
		return "Provider adapter %q auth mapping failed: %v. For Claude via Cursor, put your Anthropic API key in the OpenAI API key field (Bearer → x-api-key)."
	case "CLI_PROXY_ADAPTER_RESPONSE_FMT":
		return "Provider adapter %q failed to translate the upstream response: %v"
	case "CLI_PROXY_ADAPTER_ENABLED_FMT":
		return "OpenAI-compat provider adapters loaded (%d enabled). Model prefixes route via DB (Claude Messages and same-shape hosts with upstream_base_url). GET /v1/models lists DB provider_discoverable_models. Unmatched chat models return a clear Trim error (no silent host invent)."
	case "CLI_PROXY_MODELS_NOT_SYNCED":
		return "Discoverable models are not synced. Run trim config sync while logged in, then restart trim start."
	case "CLI_PROXY_MODELS_METHOD":
		return "GET only on /v1/models"
	case "CLI_PROXY_ERR_BASE_URL_DOUBLE_V1":
		return "Invalid path /v1/v1/…. For Claude Code set ANTHROPIC_BASE_URL to the Trim listen origin only (e.g. http://127.0.0.1:8888) with no /v1 suffix. For Cursor / OpenAI SDKs use Base URL …/v1 (single /v1)."
	case "CLI_PROXY_ERR_MODEL_NOT_FOUND_FMT":
		return "Upstream rejected model %q (not found or not allowed for this API key). Pick another model from GET /v1/models, use a Trim alias from Admin → Provider adapters, or check your provider plan entitlements."
	case "CLI_PROXY_ERR_UNKNOWN_MODEL_FMT":
		return "No provider adapter matched model %q. Pick an id from GET /v1/models, use a DB alias (trim-claude-*, trim-gemini-*, …), or add a match_model_prefixes row in provider_adapters. Trim does not invent a host for unknown models."
	case "CLI_PROXY_ERR_UPSTREAM_AUTH_FMT":
		return "Upstream rejected the API key for model %q (authentication failed). Use a key that matches that model's provider (Anthropic for Claude, OpenAI for GPT, Gemini/DeepSeek/Mistral keys for those hosts). For Claude via Cursor put the Anthropic key in the OpenAI API key field."
	case "CLI_PROXY_ERR_UPSTREAM_QUOTA_FMT":
		return "Upstream rate limit or quota exhausted for model %q. Wait and retry, check the provider billing/plan, or pick another model from GET /v1/models."
	case "CLI_PROXY_ERR_UPSTREAM_UNAVAILABLE_FMT":
		return "Upstream provider is temporarily unavailable for model %q (gateway/host error). Retry later, or pick another model/provider from GET /v1/models. This is not a Trim Base URL mistake."
	case "CLI_SETUP_CLAUDE_CODE_BLOCK_FMT":
		return "Claude Code (Door A) - copy/paste:\n  export ANTHROPIC_BASE_URL=%q\n  export ANTHROPIC_API_KEY=sk-ant-…   # prefer workspace-scoped key\n  # optional discovery:\n  export CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1\n  # org/multi-workspace keys only:\n  # export TRIM_ANTHROPIC_WORKSPACE_ID=wrkspc_…   # then restart trim start\n  # or send header anthropic-workspace-id on each request"
	case "CLI_SETUP_CONTINUE_BLOCK_FMT":
		return "Continue (OpenAI door) - edit ~/.continue/config.yaml (not config.json on modern Continue):\n  provider: openai\n  model: trim-claude-sonnet   # or id from GET /v1/models\n  apiBase: %q\n  apiKey: <Anthropic key for Claude / OpenAI key for GPT>\n  capabilities: [tool_use]\n  useResponsesApi: false\n  Then: Continue sidebar → Reload config → pick Claude via Trim. Docs: /docs/ide/continue"
	case "CLI_SETUP_VSCODE_CHAT_BLOCK_FMT":
		return "VS Code Chat Custom Endpoint (Ask) - Chat: Manage Language Models → Add Models → Custom Endpoint:\n  url: %q   # must be full …/v1/chat/completions (not only …/v1)\n  id: trim-claude-sonnet\n  apiKey: wizard ${input:chat.lm.secret…} only (never paste raw sk- into the JSON)\n  File: Code/User/chatLanguageModels.json\n  Prefer Continue for Agent file edits. Docs: /docs/ide/vscode"
	case "CLI_PROXY_MODELS_UPSTREAM_NOT_FOUND_FMT":
		return "Unknown or disabled provider adapter %q for upstream model discovery. Use an enabled openai_compat adapter id from provider_adapters (e.g. openai, gemini, deepseek, mistral)."
	case "CLI_PROXY_MODELS_UPSTREAM_AUTH":
		return "Authorization Bearer is required for GET /v1/models?adapter=… (upstream key-accurate discovery)."
	case "CLI_PROXY_MODELS_UPSTREAM_UNSUPPORTED_FMT":
		return "Adapter %q cannot proxy upstream /models (openai_compat only). Default GET /v1/models without ?adapter= still returns the DB catalog."
	case "CLI_PROXY_MODELS_UPSTREAM_ADAPTER_REQUIRED":
		return "Upstream model discovery requires ?adapter=<id> (enabled openai_compat provider_adapters id)."
	case "CLI_PROXY_DOOR_OPENAI":
		return "openai"
	case "CLI_PROXY_DOOR_ANTHROPIC":
		return "anthropic"
	case "LOCAL_STATS_LAST_DOOR":
		return "Last door"
	case "LOCAL_TUI_LABEL_LAST_DOOR":
		return "Last door"
	case "ADMIN_NAV_PROVIDER_ADAPTERS":
		return "Provider adapters"
	case "ADMIN_PROVIDER_ADAPTERS_TITLE":
		return "Provider adapters"
	case "ADMIN_PROVIDER_ADAPTERS_DESC":
		return "OpenAI /v1/chat/completions plugins. anthropic_messages translates to /v1/messages. openai_compat keeps OpenAI shape and routes by match_model_prefixes to upstream_base_url (GPT, Gemini, DeepSeek, Mistral, …). No daily UPSTREAM_OPENAI_URL flipping. Synced to CLI via preferences."
	case "ADMIN_OPENAI_MODEL_ALIASES_TITLE":
		return "OpenAI model aliases"
	case "ADMIN_OPENAI_MODEL_ALIASES_DESC":
		return "Rewrite client model ids before routing (e.g. trim-gemini-* → gemini-*). Optional upstream_host_contains scopes an alias to a specific OpenAI upstream host."
	case "DOCS_PROVIDER_ADAPTERS_SUMMARY":
		return "Trim exposes one Base URL with two native doors plus DB-driven adapters: Claude via anthropic_messages, and same-shape hosts (GPT, Gemini, DeepSeek, Mistral) via openai_compat + upstream_base_url. GET /v1/models advertises provider_discoverable_models from the database. No OpenRouter. Keep Cursor Base URL = Trim …/v1; Claude Code ANTHROPIC_BASE_URL = Trim origin without /v1."
	case "DOCS_CONNECT_IDE_SUMMARY":
		return "Point Cursor/VS Code OpenAI Base URL at Trim …/v1. Point Claude Code at Trim listen origin via ANTHROPIC_BASE_URL (no /v1 suffix). Run trim config sync so provider adapters, aliases, and discoverable models load from the database."
	case "DOCS_TROUBLESHOOT_BASE_URL":
		return "Do not set Cursor Override OpenAI Base URL to https://api.anthropic.com (bypasses Trim; Anthropic’s OpenAI-compat layer is limited / not their long-term production path). Do not mix Cursor Anthropic BYOK with Trim OpenAI override. Claude in Cursor: Trim Base URL + Claude model + Anthropic key in the OpenAI key field + trim config sync. Claude Code: ANTHROPIC_BASE_URL=Trim listen origin only (no /v1 suffix)."
	case "PREFERENCES_CLI_HELP_7":
		return "Claude in Cursor: same OpenAI Base URL → Trim; pick a Claude / trim-claude-* model; put your Anthropic key in the OpenAI API key field. Prefer a workspace-scoped Anthropic key. Org/multi-workspace keys: set TRIM_ANTHROPIC_WORKSPACE_ID once (or send anthropic-workspace-id). Adapters sync via trim config sync (no OpenRouter)."
	case "PREFERENCES_CLI_HELP_8":
		return "Claude Code: export ANTHROPIC_BASE_URL to the Trim listen origin only (e.g. http://127.0.0.1:8888) with no /v1 suffix - Claude Code appends /v1/messages. Optional: CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1 to load models from Trim GET /v1/models. Prefer a workspace-scoped Anthropic key; org keys need TRIM_ANTHROPIC_WORKSPACE_ID or anthropic-workspace-id."
	case "PREFERENCES_CLI_HELP_9":
		return "GPT / Gemini / DeepSeek / Mistral: same OpenAI Base URL → Trim. DB provider_adapters (openai_compat) pick the host from the model prefix. Put that host’s key in the OpenAI key field for the model you pick. After trim config sync, unmatched prefixes return a clear Trim error (no silent invent). UPSTREAM_OPENAI_URL is only the OpenAI-door default when adapters are not synced. Do not edit .env every day."
	case "PREFERENCES_LIVE_DEEP_MIN_HINT":
		return "Live Deep runs only when estimated compressible input tokens are at least live_deep_min_input_tokens from billing settings (0 = every turn). Lower values run Deep on more chats (more CPU/latency). Synced by trim config sync."
	case "PREFERENCES_LIVE_DEEP_STREAM_HINT":
		return "Billing may skip live Deep when the client sets stream=true (live_deep_skip_on_stream). Synced by trim config sync."
	case "ADMIN_BILLING_LIVE_DEEP_SKIP_STREAM":
		return "Skip Deep on stream requests"
	case "ADMIN_BILLING_LIVE_DEEP_SKIP_STREAM_DESC":
		return "When on, live proxy skips Deep for stream=true chat (lower latency). When off, Deep still runs on the full body before upstream streaming."
	case "ADMIN_BILLING_LIVE_DEEP_MIN":
		return "Live Deep min input tokens"
	case "ADMIN_BILLING_LIVE_DEEP_MIN_DESC":
		return "Skip Deep on the live proxy when estimated input tokens are below this (0 = every turn). Lowering increases Deep coverage and local CPU/latency. Synced to CLI."
	case "ADMIN_BILLING_LIVE_DEEP_OOM":
		return "Live Deep OOM policy"
	case "ADMIN_BILLING_LIVE_DEEP_OOM_DESC":
		return "fail = fail-closed on memory errors; skip = Fast-only for that request."
	case "ADMIN_BILLING_LIVE_DEEP_WARMUP":
		return "Warm Deep on trim start"
	case "ADMIN_BILLING_LIVE_DEEP_WARMUP_DESC":
		return "When on, trim start warms the local Deep engine once after binding prefs."
	case "ADMIN_BILLING_DEEP_V1_MODEL":
		return "Deep v1 model id"
	case "ADMIN_BILLING_DEEP_V2_MODEL":
		return "Deep v2 model id"
	case "ADMIN_BILLING_DEEP_LONG_MODEL":
		return "Deep long model id"
	case "ADMIN_BILLING_DEEP_V2_FORCE_TOKENS":
		return "Deep v2 force_tokens JSON"
	case "CLI_PREFS_ENGINE_INVALID":
		return "deep_engine must be v1, long, or v2"
	case "CLI_PREFS_TARGET_NONNEG":
		return "target_token must be a non-negative integer"
	case "CLI_PREFS_TIER_REQUIRED":
		return "default_tier must be set (fast or deep) via trim config set or trim config sync"
	case "CLI_PREFS_ENGINE_REQUIRED":
		return "deep_engine must be set (v1, long, or v2) via trim config set or trim config sync"
	case "CLI_PREFS_TARGET_REQUIRED":
		return "target_token must be a positive integer (trim config set target-token <n> or trim config sync)"
	case "CLI_CONFIG_GET_TIER_FMT":
		return "default-tier=%s"
	case "CLI_CONFIG_GET_ENGINE_FMT":
		return "deep-engine=%s"
	case "CLI_CONFIG_GET_TARGET_FMT":
		return "target-token=%d"
	case "CLI_CONFIG_UNKNOWN_KEY_FMT":
		return "unknown key %q (default-tier|deep-engine|target-token)"
	case "CLI_CONFIG_TARGET_POSITIVE":
		return "target-token must be a positive integer"
	case "CLI_CONFIG_SAVED_FMT":
		return "Saved %s"
	case "CLI_CONFIG_NOT_LOGGED_IN":
		return "not logged in. Run: trim login"
	case "CLI_CONFIG_SYNC_FAILED_FMT":
		return "preferences sync failed (%s): %s"
	case "CLI_CONFIG_SYNCED_FMT":
		return "Synced: default-tier=%s deep-engine=%s target-token=%d"
	case "CLI_HELP_ROOT_SHORT":
		return "Local Fast Mode proxy for AI coding tools"
	case "CLI_HELP_ROOT_LONG":
		return "Trim keeps a local Fast Mode proxy on your machine so IDE traffic uses less noisy context.\n\nEveryday: start, stop, status, stats, autostart, login, setup\nAdvanced: tui, compress, daemon, telemetry, config\n\nSimple savings check after trim start:\n  trim stats\n  open http://127.0.0.1:<port>/dashboard → Show savings detail\n\ntrim status = cloud quota. trim stats = local Deep status / savings meter.\n\nRun \"trim help <command>\" for details. Prefer trim start / autostart for daily use; compress and Deep Mode are file/batch tools, not the everyday IDE path."
	case "CLI_HELP_GROUP_EVERYDAY":
		return "Everyday Commands:"
	case "CLI_HELP_GROUP_ADVANCED":
		return "Advanced Commands:"
	case "CLI_HELP_START_SHORT":
		return "Start the local Trim Fast Mode proxy"
	case "CLI_HELP_START_LONG":
		return "Starts the local proxy for IDE traffic. Fast Mode always runs. When account preferences set Deep Mode on, each request also runs Deep (LLMLingua) after Fast before forwarding upstream.\n\nAfter traffic flows, check savings clearly with:\n  trim stats\n  http://127.0.0.1:<port>/dashboard → Show savings detail\n\nWhole-body Saved %% can look low on agent chrome; Deep stage and Deep status explain why."
	case "CLI_HELP_START_FLAG_PORT":
		return "Override TRIM_PORT for this process"
	case "CLI_HELP_LOGIN_SHORT":
		return "Sign in via browser"
	case "CLI_HELP_LOGOUT_SHORT":
		return "Sign out and clear the stored API token"
	case "CLI_HELP_UNINSTALL_SHORT":
		return "Fully remove Trim from this machine (proxy, daemon, data, Deep caches, binary)"
	case "CLI_HELP_UNINSTALL_LONG":
		return "Stops the proxy, removes the OS login daemon, clears credentials, reverts IDE Base URL overrides from trim setup, deletes ~/.trim and ~/.config/trim, cleans Deep Mode caches from site_messages lists, removes sidecars, and deletes the CLI binary/PATH when possible. Use --keep-data to skip deleting local data directories and Deep caches."
	case "CLI_HELP_UNINSTALL_FLAG_KEEP_DATA":
		return "Keep ~/.trim, ~/.config/trim, Deep Mode caches, and the CLI binary (only stop/daemon/logout/setup revert)"
	case "CLI_HELP_UNINSTALL_FLAG_PURGE":
		return "Deprecated: uninstall purges by default; use --keep-data to retain local data"
	case "CLI_UNINSTALL_STARTING":
		return "Uninstalling Trim from this machine…"
	case "CLI_UNINSTALL_STOPPED":
		return "Local proxy stop attempted."
	case "CLI_UNINSTALL_DAEMON_OK":
		return "OS login daemon removed (when supported)."
	case "CLI_UNINSTALL_LOGOUT_OK":
		return "CLI credentials cleared."
	case "CLI_UNINSTALL_SETUP_REVERTED_FMT":
		return "Reverted IDE/local proxy Base URL in %s"
	case "CLI_UNINSTALL_SETUP_NONE":
		return "No trim setup Base URL overrides found to revert."
	case "CLI_UNINSTALL_CONFIG_REMOVED_FMT":
		return "Removed local data: %s"
	case "CLI_UNINSTALL_LOGS_REMOVED":
		return "Removed Trim log files (when present)."
	case "CLI_UNINSTALL_DEEP_OK":
		return "Deep Mode local caches and pip packages cleaned (when present)."
	case "CLI_UNINSTALL_DEEP_SKIPPED":
		return "Deep Mode purge skipped: CLI_UNINSTALL_HF_HUB_DIRS / CLI_UNINSTALL_PIP_PACKAGES not configured in site_messages."
	case "CLI_UNINSTALL_SIDECAR_REMOVED_FMT":
		return "Removed Deep Mode sidecar: %s"
	case "CLI_UNINSTALL_BINARY_REMOVED_FMT":
		return "Removed CLI binary: %s"
	case "CLI_UNINSTALL_BINARY_MANUAL_FMT":
		return "Could not remove the running binary automatically. Delete manually: %s"
	case "CLI_UNINSTALL_PATH_REMOVED":
		return "Removed Trim install directory from the user PATH (Windows)."
	case "CLI_UNINSTALL_DONE":
		return "Trim local uninstall finished."
	case "CLI_UNINSTALL_NEXT_EXT":
		return "IDE extension: Trim: Clear API Key (clears Secret Storage + local extension state), then Command Palette → Extensions → uninstall Trim IDE."
	case "CLI_UNINSTALL_NEXT_PKG_BREW":
		return "If installed via Homebrew: brew uninstall trim"
	case "CLI_UNINSTALL_NEXT_PKG_SCOOP":
		return "If installed via Scoop: scoop uninstall trim"
	case "CLI_UNINSTALL_NEXT_PKG_WINGET":
		return "If installed via winget: winget uninstall Trim.CLI"
	case "CLI_UNINSTALL_HF_HUB_DIRS":
		return "models--microsoft--llmlingua-2-bert-base-multilingual-cased-meetingbank"
	case "CLI_UNINSTALL_HF_CACHE_REL":
		return ".cache/huggingface"
	case "CLI_UNINSTALL_HF_HUB_SUBDIR":
		return "hub"
	case "CLI_UNINSTALL_HOME_DIRS_REL":
		return ".trim,.config/trim"
	case "CLI_UNINSTALL_HOME_DIRS_MISSING":
		return "Uninstall home dirs skipped: CLI_UNINSTALL_HOME_DIRS_REL not configured in site_messages."
	case "CLI_UNINSTALL_DARWIN_LOG_RELS":
		return "Library/Logs/trim.log,Library/Logs/trim.err.log"
	case "CLI_UNINSTALL_PIP_PACKAGES":
		return "llmlingua"
	case "CLI_UNINSTALL_PIP_BINS":
		return "pip,pip3"
	case "CLI_UNINSTALL_PYTHON_BINS":
		return "python,python3"
	case "CLI_UNINSTALL_PIP_MODULE":
		return "pip"
	case "CLI_UNINSTALL_SIDECAR_NAMES":
		return "trim-deep,trim-deep.exe,optimizer,optimizer.exe,optimizer.py,requirements-deep.txt,requirements.txt"
	case "CLI_UNINSTALL_ARP_DISPLAY_NAME":
		return "Trim"
	case "CLI_UNINSTALL_ARP_PUBLISHER":
		return "Trim"
	case "CLI_UNINSTALL_ARP_REG_KEY":
		return "Trim"
	case "CLI_UNINSTALL_ARP_REMOVED":
		return "Removed Trim from Windows Apps & features."
	case "CLI_HELP_STATUS_SHORT":
		return "Show auth and quota status"
	case "CLI_HELP_STATUS_LONG":
		return "Shows cloud login/quota for your Trim account.\n\nFor local savings / Deep status after trim start, use:\n  trim stats\n  open http://127.0.0.1:<port>/dashboard → Show savings detail\n\ntrim status does not replace the local meter."
	case "CLI_HELP_STATS_SHORT":
		return "Show local proxy savings and Deep status"
	case "CLI_HELP_STATS_LONG":
		return "Reads the live local proxy (/v1/stats) and prints clear Deep status + Deep stage savings, then the raw JSON.\n\nUseful when Claude Code / Cursor shows 0% whole-body Saved: check Deep status (skipped below min, chrome frozen, fail-closed) and Deep stage tokens.\n\nEveryday:\n  trim start\n  trim stats\n  open http://127.0.0.1:<TRIM_PORT>/dashboard → Show savings detail\n\nUse --tui for the Bubble Tea live dashboard (same as trim tui)."
	case "CLI_HELP_STATS_FLAG_TUI":
		return "Open Bubble Tea live dashboard (same as trim tui)"
	case "CLI_HELP_SETUP_SHORT":
		return "Point Cursor, Continue, VS Code Chat, Claude Code, and shell env at Trim"
	case "CLI_HELP_SETUP_LONG":
		return "Writes OpenAI-compatible Base URL overrides for installed IDEs (Cursor, VS Code settings, Windsurf, Zed, Continue ~/.continue/config.yaml when present) and prints copy/paste blocks for:\n  - Claude Code (ANTHROPIC_BASE_URL = Trim origin, no /v1)\n  - Continue (config.yaml apiBase = Trim …/v1)\n  - VS Code Chat Custom Endpoint (full …/v1/chat/completions + wizard secret)\nUse --tls for local self-signed certs. Docs: /docs/ide/connect /docs/ide/continue /docs/ide/vscode"
	case "CLI_HELP_SETUP_FLAG_TLS":
		return "Generate local self-signed TLS cert and key under ~/.config/trim/tls"
	case "CLI_HELP_VERSION_SHORT":
		return "Print CLI version"
	case "CLI_HELP_DAEMON_SHORT":
		return "Background OS service for auto-start"
	case "CLI_HELP_DAEMON_INSTALL_SHORT":
		return "Install Trim proxy to start on login (launchd / systemd --user / Windows task)"
	case "CLI_HELP_DAEMON_UNINSTALL_SHORT":
		return "Remove the Trim background service"
	case "CLI_HELP_DAEMON_STATUS_SHORT":
		return "Show whether the Trim background service is installed"
	case "CLI_HELP_DAEMON_RUN_SHORT":
		return "Start proxy only when auto-start preference is on (used by OS login service)"
	case "CLI_HELP_DAEMON_SERVICE_DESC":
		return "Trim local context optimization proxy"
	case "CLI_HELP_TELEMETRY_SHORT":
		return "Anonymous usage telemetry opt-in/out"
	case "CLI_HELP_TELEMETRY_STATUS_SHORT":
		return "Show whether telemetry is enabled"
	case "CLI_HELP_TELEMETRY_DISABLE_SHORT":
		return "Opt out of anonymous usage telemetry"
	case "CLI_HELP_TELEMETRY_ENABLE_SHORT":
		return "Opt in to anonymous usage telemetry"
	case "CLI_HELP_TUI_SHORT":
		return "Open the terminal stats dashboard"
	case "CLI_HELP_TUI_LONG":
		return "Live local proxy metrics. Keep trim start running in another terminal (or via trim daemon)."
	case "CLI_HELP_COMPRESS_SHORT":
		return "Compress a file (Fast or Deep). Live IDE proxy also runs Deep when Preferences Deep is on."
	case "CLI_HELP_COMPRESS_LONG":
		return "Fast Mode is low-latency local compression (live proxy and files). Deep Mode is stronger on-machine LLMLingua: live proxy when Preferences Deep is on, and trim compress --deep for file/batch. Engines are not loaded in Trim cloud.\n\nDeep engines (pick one): v2 LLMLingua-2 (recommended, usually faster), long LongLLMLingua (question-aware; requires --question), v1 LLMLingua (classic). Licensed use still requires Trim login and network."
	case "CLI_HELP_COMPRESS_FLAG_MODE":
		return "fast | deep (required unless prefs default-tier or --deep)"
	case "CLI_HELP_COMPRESS_FLAG_DEEP":
		return "shortcut for --mode deep"
	case "CLI_HELP_COMPRESS_FLAG_ENGINE":
		return "Deep engine: v2 (recommended, usually faster) | long (needs --question) | v1 (classic). Else prefs deep-engine."
	case "CLI_HELP_COMPRESS_FLAG_QUESTION":
		return "question required when Deep engine is long"
	case "CLI_HELP_COMPRESS_FLAG_TARGET":
		return "deep target token budget (0 = prefs default)"
	case "CLI_HELP_COMPRESS_FLAG_BOOTSTRAP":
		return "install Deep Mode dependencies on this machine"
	case "CLI_HELP_COMPRESS_FLAG_OUT":
		return "write result to file instead of stdout"
	case "CLI_HELP_CONFIG_SHORT":
		return "Local Fast/Deep preferences"
	case "CLI_HELP_CONFIG_GET_SHORT":
		return "Show preferences (default-tier, deep-engine, target-token)"
	case "CLI_HELP_CONFIG_SET_SHORT":
		return "Set a preference (example: trim config set default-tier deep)"
	case "CLI_HELP_CONFIG_SYNC_SHORT":
		return "Pull Fast/Deep preferences and provider adapters from your Trim cloud profile into ~/.config/trim/"
	case "DECIDE_TOPUP_CHECKOUT":
		return "One-time credit pack checkout."
	case "DECIDE_NOT_SELF_SERVE":
		return "This plan cannot be purchased through self-serve checkout."
	case "DECIDE_NEW_CHECKOUT":
		return "No active unexpired paid subscription. New checkout required."
	case "DECIDE_SAME_PLAN":
		return "You are already on this plan and billing interval."
	case "DECIDE_INTERVAL_UPGRADE_NEW_CHECKOUT":
		return "Switching to annual requires a new checkout because the current subscription is canceled."
	case "DECIDE_INTERVAL_UPGRADE":
		return "Switching monthly to annual on the same plan is an upgrade. Remaining period is credited via Paddle proration."
	case "DECIDE_INTERVAL_CHANGE_DISABLED":
		return "Monthly to annual changes are disabled in billing settings."
	case "DECIDE_DOWNGRADE_ANNUAL_TO_MONTHLY":
		return "Downgrading from annual to monthly is not allowed while your plan is active. Wait until the period ends."
	case "DECIDE_UPGRADE_NEW_CHECKOUT":
		return "Your current plan is canceled but still active until expiry. Starting a higher plan requires a new checkout."
	case "DECIDE_UPGRADE_ALLOWED":
		return "Upgrade allowed. Active unexpired time is handled by Paddle proration on the existing subscription."
	case "DECIDE_DOWNGRADE_BLOCKED":
		return "Downgrades are not allowed while you have an active unexpired plan. Your current plan stays until it expires."
	case "DECIDE_DOWNGRADE_UNSUPPORTED":
		return "Self-serve downgrades are not supported while an unexpired paid period remains. Wait until expiry or contact support."
	case "FREE_INCLUDED_REASON":
		return "Free plan is included with every account."
	case "ENTERPRISE_CONTACT_REASON":
		return "Enterprise is sales-assisted. Submit an inquiry from the plan modal."
	case "DECIDE_CHANGE_NOT_ALLOWED":
		return "This plan change is not allowed for your current subscription."
	case "AUTH_TOKEN_MISSING":
		return "Missing or malformed authorization token."
	case "CLI_VERSION_HEADER_MISSING":
		return "Missing X-Client-Version. Upgrade the Trim CLI."
	case "CLI_VERSION_OUTDATED":
		return "CLI version is outdated. Run: trim update"
	case "AUTH_CREDENTIALS_INVALID":
		return "Invalid credentials."
	case "AUTH_CREDENTIALS_REVOKED":
		return "Invalid or revoked credentials. Sign in again."
	case "AUTH_PROVIDER_DENIED_EMPTY":
		return "Sign-in provider is not allowed. No auth providers are configured on this server."
	case "AUTH_PROVIDER_DENIED_FMT":
		return "Sign-in provider is not allowed. Enabled providers: %s."
	case "RATE_LIMIT_USER":
		return "Rate limit exceeded. Slow down and retry."
	case "FREE_TIER_PROXY_BLOCKED":
		return "Proxy or VPN requests are restricted on the free tier."
	case "HARDWARE_UUID_REQUIRED":
		return "X-Hardware-UUID is required for CLI requests on the free tier."
	case "HARDWARE_ACCOUNT_LIMIT":
		return "Hardware limit exceeded. Upgrade to Pro for more workspaces."
	case "JA4_ACCOUNT_LIMIT":
		return "Client fingerprint limit exceeded. Upgrade to Pro or contact support."
	case "DATABASE_UNAVAILABLE":
		return "Database unavailable."
	case "QUOTA_LOAD_FAILED":
		return "Failed to load quota."
	case "QUOTA_EXHAUSTED":
		return "Monthly quota exhausted."
	case "WORKSPACE_NOT_MEMBER":
		return "Not a member of this workspace."
	case "WORKSPACE_QUOTA_LOAD_FAILED":
		return "Failed to load workspace quota."
	case "WORKSPACE_QUOTA_EXHAUSTED":
		return "Workspace shared quota exhausted."
	case "POW_REQUIRED":
		return "Free tier requires proof-of-work. Solve the challenge and retry with X-Trim-PoW."
	case "CANARY_NOT_FOUND":
		return "not found"
	case "IP_BLOCKED":
		return "This network is blocked for abuse. Contact support."
	case "RATE_LIMIT_IP":
		return "Too many requests from this network. Slow down and retry."
	case "TLS_FINGERPRINT_BLOCKED":
		return "Client fingerprint blocked for abuse."
	case "SUSPICIOUS_CLIENT":
		return "Invalid client signature."
	case "TIMESTAMP_REQUIRED":
		return "X-Request-Timestamp is required for CLI requests."
	case "REQUEST_EXPIRED":
		return "Request timestamp skew too large."
	case "HARDWARE_BLOCKED":
		return "This device is blocked for abuse. Contact support."
	case "HARDWARE_UUID_MISMATCH":
		return "This API key is locked to a different device. Re-issue the key on this machine."
	case "HARDWARE_UUID_BOUND_REQUIRED":
		return "This API key requires X-Hardware-UUID."
	case "HARDWARE_DEVICE_LIMIT":
		return "This API key already has the maximum number of registered devices. Revoke and re-issue, or remove a device."
	case "API_KEY_MAX_DEVICES_MISSING":
		return "Set admin_product_settings.api_key_max_devices (1-16) before device-bound API keys can enroll."
	case "IP_DENIED":
		return "This IP address is denylisted. Contact support."
	case "ASN_DENIED":
		return "This network (ASN) is denylisted. Contact support."
	case "DENYLIST_UNAVAILABLE":
		return "Abuse denylist unavailable. Try again shortly."
	case "AGENT_ID_REQUIRED":
		return "X-Trim-Agent-Id is required for API key requests."
	case "AGENT_ID_UNKNOWN":
		return "Unknown X-Trim-Agent-Id. Use a value from agent_identity_catalog (cli, ide, ci)."
	case "AGENT_ID_MISMATCH":
		return "This API key is locked to a different agent identity."
	case "SIGNATURE_REQUIRED":
		return "X-Trim-Signature is required for CLI requests."
	case "BODY_READ_FAILED":
		return "Cannot read request body for signature check."
	case "CLI_SIGNATURE_INVALID_FMT":
		return "CLI signature invalid: %s"
	case "CLI_SIGNATURE_DETAIL_HMAC":
		return "HMAC mismatch"
	case "IDE_STATUS_TOOLTIP":
		return "Flush Trim IDE telemetry"
	case "IDE_STATUS_IDLE":
		return "Trim"
	case "IDE_STATUS_PENDING_FMT":
		return "Trim $(cloud-upload) %d"
	case "IDE_API_KEY_TITLE":
		return "Trim API key"
	case "IDE_API_KEY_PROMPT":
		return "Paste a key from Dashboard → Settings → API keys (trm_...)"
	case "IDE_API_KEY_SAVED":
		return "Trim API key saved in Secret Storage."
	case "IDE_API_KEY_CLEARED":
		return "Trim API key and local extension state cleared."
	case "IDE_TELEMETRY_FLUSHED":
		return "Trim telemetry flushed."
	case "IDE_TELEMETRY_FAILED_FMT":
		return "Trim telemetry failed (%d): %s"
	case "IDE_TELEMETRY_NETWORK_FMT":
		return "Trim telemetry network error: %s"
	case "IDE_EVENT_MODE":
		return "ide"
	case "IDE_EVENT_STATUS":
		return "success"
	case "IDE_EVENT_MODEL":
		return "ide"
	case "IDE_CMD_SET_API_KEY":
		return "Trim: Set API Key"
	case "IDE_CMD_CLEAR_API_KEY":
		return "Trim: Clear API Key"
	case "IDE_CMD_MARK_TAB_SHOWN":
		return "Trim: Mark Tab Suggestion Shown"
	case "IDE_CMD_MARK_TAB_ACCEPTED":
		return "Trim: Mark Tab Suggestion Accepted"
	case "IDE_CMD_FLUSH_TELEMETRY":
		return "Trim: Flush Telemetry Now"
	case "IDE_CMD_COPY_HARDWARE_ID":
		return "Trim: Copy Hardware ID"
	case "IDE_HARDWARE_ID_COPIED":
		return "Trim hardware ID copied. Register it under Dashboard → Settings → API keys."
	case "IDE_CONFIG_TITLE":
		return "Trim"
	case "IDE_CONFIG_API_URL":
		return "Trim Cloud API base URL (required; no default host)"
	case "IDE_CONFIG_AUTO_FLUSH":
		return "Seconds between automatic telemetry flushes"
	case "IDE_CONFIG_TRACK_EDITS":
		return "Heuristic LOC tracking from multi-line inserts/deletes (labeled mode=ide)"
	case "IDE_CONFIG_MIN_LINES":
		return "Minimum AI-edited lines before an event is queued"
	case "PADDLE_WEBHOOK_SECRET_MISSING":
		return "Paddle webhook secret is not configured."
	case "PADDLE_SIGNATURE_INVALID":
		return "Invalid Paddle webhook signature."
	case "PADDLE_PAYLOAD_INVALID":
		return "Invalid Paddle webhook JSON."
	case "PADDLE_QUEUE_BUSY":
		return "Webhook queue is busy. Retry shortly."
	case "AUTH_PROVIDERS_UNAVAILABLE":
		return "Auth providers unavailable."
	case "LEGAL_PRIVACY_UNAVAILABLE":
		return "Legal privacy content unavailable."
	case "LEGAL_TERMS_UNAVAILABLE":
		return "Legal terms content unavailable."
	case "CRON_UNAUTHORIZED":
		return "unauthorized"
	case "EXPIRE_BATCH_FAILED":
		return "expire failed"
	case "EXPIRE_BATCH_LIMIT_MISSING":
		return "TRIM_EXPIRE_BATCH_DEFAULT_LIMIT is not configured"
	case "QUOTA_NOT_FOUND":
		return "quota not found"
	case "RECEIPT_NOT_FOUND_ERROR":
		return "receipt not found"
	case "PROFILE_EMAIL_NOT_FOUND":
		return "profile email not found"
	case "INVITE_NOT_FOUND_OR_CLOSED":
		return "invite not found or already closed"
	case "INVITE_NOT_FOUND":
		return "invite not found"
	case "PROFILE_NOT_FOUND":
		return "profile not found"
	case "MEMBER_NOT_FOUND":
		return "member not found"
	case "API_KEY_NOT_FOUND_OR_REVOKED":
		return "api key not found or already revoked"
	case "INVITE_REVOKE_FAILED":
		return "failed to revoke invite"
	case "INVITE_TOKEN_REQUIRED":
		return "token is required"
	case "INVITE_NO_LONGER_PENDING":
		return "invite is no longer pending"
	case "INVITE_EXPIRED":
		return "invite has expired"
	case "WS_UNAUTHORIZED":
		return "unauthorized"
	case "WS_FORBIDDEN":
		return "forbidden"
	case "WS_INVALID_JSON":
		return "invalid json"
	case "WS_NAME_REQUIRED":
		return "name is required"
	case "WS_TX_BEGIN_FAILED":
		return "tx begin failed"
	case "WS_COMMIT_FAILED":
		return "commit failed"
	case "WS_COUNT_FAILED":
		return "failed to count workspaces"
	case "WS_LIST_FAILED":
		return "failed to list workspaces"
	case "WS_SCAN_FAILED":
		return "failed to scan workspace"
	case "WS_RESOLVE_PLAN_FAILED":
		return "failed to resolve plan"
	case "WS_RESOLVE_SEATS_FAILED":
		return "failed to resolve seat_quantity from subscription"
	case "WS_SEAT_QUANTITY_INVALID":
		return "subscription seat_quantity must be >= 1"
	case "WS_PLAN_CREDITS_MISSING":
		return "plan_catalog missing credits for active plan"
	case "WS_CREATE_FAILED":
		return "failed to create workspace"
	case "WS_ADD_OWNER_FAILED":
		return "failed to add owner"
	case "WS_INIT_QUOTA_FAILED":
		return "failed to init quota"
	case "WS_MEMBERS_COUNT_FAILED":
		return "failed to count members"
	case "WS_OWNERS_COUNT_FAILED":
		return "failed to count owners"
	case "WS_MEMBERS_LIST_FAILED":
		return "failed to list members"
	case "WS_MEMBER_SCAN_FAILED":
		return "failed to scan member"
	case "WS_APP_PUBLIC_URL_MISSING":
		return "APP_PUBLIC_URL is not configured"
	case "WS_ROLE_INVALID":
		return "role must be member or admin"
	case "WS_EMAIL_REQUIRED":
		return "valid email is required"
	case "WS_EMAIL_DOMAIN_DENIED":
		return "That invite email domain is blocked (disposable or denylisted)."
	case "LOGIN_OAUTH_EMAIL_DENIED":
		return "Sign-in with disposable or blocked email domains is not allowed. Use a lasting work or personal address."
	case "AUTH_EMAIL_DOMAIN_DENIED":
		return "This email domain is not allowed."
	case "WS_INVITER_LOAD_FAILED":
		return "failed to load inviter profile"
	case "WS_CANNOT_INVITE_SELF":
		return "cannot invite yourself"
	case "WS_ALREADY_MEMBER":
		return "user is already a member of this workspace"
	case "WS_SEAT_CAPACITY_LOAD_FAILED":
		return "failed to load seat capacity"
	case "WS_NO_SEATS_INVITE":
		return "no seats available; upgrade allocated seats before inviting"
	case "WS_INVITE_TTL_MISSING":
		return "billing_settings.workspace_invite_ttl_hours is missing"
	case "WS_INVITE_TOKEN_GEN_FAILED":
		return "failed to generate invite token"
	case "WS_REVOKE_PRIOR_INVITE_FAILED":
		return "failed to revoke prior invite"
	case "WS_CREATE_INVITE_FAILED":
		return "failed to create invite"
	case "WS_INVITES_COUNT_FAILED":
		return "failed to count invites"
	case "WS_INVITES_LIST_FAILED":
		return "failed to list invites"
	case "WS_INVITE_SCAN_FAILED":
		return "failed to scan invite"
	case "WS_INVITE_LOAD_FAILED":
		return "failed to load invite"
	case "WS_EMAIL_MISMATCH":
		return "signed-in email does not match this invite"
	case "WS_NO_SEATS_JOIN":
		return "no seats available on this workspace"
	case "WS_JOIN_FAILED":
		return "failed to join workspace"
	case "WS_MARK_ACCEPTED_FAILED":
		return "failed to mark invite accepted"
	case "WS_ADMINS_CANNOT_REMOVE_OWNERS":
		return "admins cannot remove owners"
	case "WS_CHECK_OWNERS_FAILED":
		return "failed to check owners"
	case "WS_CANNOT_REMOVE_LAST_OWNER":
		return "cannot remove the last owner"
	case "WS_REMOVE_MEMBER_FAILED":
		return "failed to remove member"
	case "PAGINATION_MAX_LIMIT_INVALID":
		return "maxLimit must be >= 1"
	case "PAGINATION_SKIP_REQUIRED":
		return "query param skip is required"
	case "PAGINATION_LIMIT_REQUIRED":
		return "query param limit is required"
	case "PAGINATION_SKIP_NOT_INT":
		return "skip must be an integer"
	case "PAGINATION_LIMIT_NOT_INT":
		return "limit must be an integer"
	case "PAGINATION_SKIP_NEGATIVE":
		return "skip must be >= 0"
	case "PAGINATION_LIMIT_TOO_SMALL":
		return "limit must be >= 1"
	case "PAGINATION_LIMIT_TOO_LARGE":
		return "limit exceeds maximum allowed"
	case "API_KEYS_COUNT_FAILED":
		return "failed to count api keys"
	case "API_KEY_SCAN_FAILED":
		return "failed to scan api key"
	case "API_KEY_ID_REQUIRED":
		return "key id is required"
	case "API_KEY_GENERATE_FAILED":
		return "failed to generate api key"
	case "COMPRESSION_TIER_FAST":
		return "fast"
	case "COMPRESSION_TIER_DEEP":
		return "deep"
	case "COMPRESSION_TIER_FAST_MISSING":
		return "COMPRESSION_TIER_FAST is not configured in site_messages."
	case "COMPRESSION_TIER_DEEP_MISSING":
		return "COMPRESSION_TIER_DEEP is not configured in site_messages."
	case "COMPRESSION_TIER_INVALID":
		return "compression_tier must be fast or deep"
	case "DEEP_ENGINE_INVALID":
		return "deep_engine must be v1, long, or v2"
	case "DEEP_TARGET_TOKEN_RANGE":
		return "deep_target_token out of range"
	case "ACCOUNT_OPERATION_FAILED":
		return "account operation failed"
	case "PLAN_NOT_FOUND":
		return "plan not found"
	case "PLAN_INTERVAL_INVALID":
		return "interval must be monthly or annual"
	case "PLAN_INVALID_JSON_BODY":
		return "invalid json body"
	case "PLAN_ID_REQUIRED":
		return "plan_id is required"
	case "PADDLE_CLIENT_MISSING":
		return "paddle api client not configured"
	case "PADDLE_API_MISSING":
		return "paddle api not configured"
	case "PADDLE_SUBSCRIPTION_ID_MISSING":
		return "active subscription missing paddle_subscription_id"
	case "PADDLE_CUSTOMER_MISSING":
		return "no paddle customer for portal"
	case "ENTERPRISE_SEATS_RANGE":
		return "estimated_seats out of range"
	case "ENTERPRISE_MESSAGE_TOO_LONG":
		return "Message exceeds the maximum length of 4000 characters."
	case "PLAN_DECIDE_FAILED":
		return "could not evaluate plan change"
	case "PLAN_CONFLICT":
		return "plan change conflict"
	case "PLAN_OPERATION_FAILED":
		return "plan operation failed"
	case "PADDLE_UPSTREAM_FAILED":
		return "paddle upstream request failed"
	case "PADDLE_UPGRADE_PREVIEW_FAILED":
		return "upgrade preview failed"
	case "PADDLE_UPGRADE_FAILED":
		return "upgrade failed"
	case "RECEIPT_ID_REQUIRED":
		return "receiptId required"
	case "RECEIPT_OPERATION_FAILED":
		return "receipt operation failed"
	case "EVENTS_COUNT_FAILED":
		return "failed to count events"
	case "EVENTS_LIST_FAILED":
		return "failed to list events"
	case "EVENTS_SCAN_FAILED":
		return "failed to scan event"
	case "EVENTS_MODE_STATUS_REQUIRED":
		return "mode and status are required"
	case "EVENTS_INVALID_METRICS":
		return "invalid metrics"
	case "EVENTS_INVALID_COUNTERS":
		return "invalid telemetry counters"
	case "EVENTS_TAB_ACCEPTED_EXCEEDS":
		return "tab_suggestions_accepted cannot exceed shown"
	case "EVENTS_STORE_FAILED":
		return "failed to store event"
	case "EVENTS_AGG_MODELS_FAILED":
		return "failed to aggregate models"
	case "EVENTS_SCAN_MODELS_FAILED":
		return "failed to scan models"
	case "EVENTS_AGG_STATUS_FAILED":
		return "failed to aggregate status"
	case "EVENTS_SCAN_STATUS_FAILED":
		return "failed to scan status"
	case "EVENTS_AGG_MODES_FAILED":
		return "failed to aggregate modes"
	case "EVENTS_SCAN_MODES_FAILED":
		return "failed to scan modes"
	case "EVENTS_AGG_SERIES_FAILED":
		return "failed to aggregate series"
	case "EVENTS_SCAN_SERIES_FAILED":
		return "failed to scan series"
	case "CHART_TOP_N_MISSING":
		return "billing_settings.chart_top_n is missing"
	case "CHART_SERIES_DAYS_MISSING":
		return "billing_settings.chart_series_days is missing"
	case "CHART_CACHE_TTL_MISSING":
		return "billing_settings.chart_cache_ttl_sec is missing"
	case "ADMIN_BILLING_CHART_CACHE_TTL":
		return "Chart cache TTL (seconds)"
	case "ADMIN_COMPLIANCE_FORCE_LOGOUT_TTL":
		return "Force-logout marker TTL (seconds)"
	case "ADMIN_FORCE_LOGOUT_TTL_MISSING":
		return "Set admin_retention_settings.force_logout_ttl_sec before force-logout."
	case "ADMIN_COMPLIANCE_EVENTS_PARTITION_AHEAD":
		return "Trim events future partitions (months ahead)"
	case "ADMIN_COMPLIANCE_EVENTS_PARTITION_ENSURE_SEC":
		return "Trim events partition ensure interval (seconds)"
	case "ADMIN_EVENTS_PARTITION_AHEAD_MISSING":
		return "Set admin_retention_settings.trim_events_partition_months_ahead before event inserts."
	case "ADMIN_EVENTS_PARTITION_ENSURE_SEC_MISSING":
		return "Set admin_retention_settings.trim_events_partition_ensure_sec before partition ensure loop."
	case "ADMIN_EVENTS_PARTITION_ENSURE_FAILED":
		return "Failed to ensure trim_events month partitions."
	case "AVATAR_PROFILE_UPDATE_FAILED":
		return "failed to update profile"
	case "AVATAR_UPLOAD_FAILED":
		return "avatar upload failed"
	case "AVATAR_CLOUDINARY_UNCONFIGURED":
		return "Cloudinary is not configured. Set CLOUDINARY_CLOUD_NAME and UPLOAD_PRESET or API key/secret."
	case "AVATAR_HTTP_TIMEOUT_MISSING":
		return "CLOUDINARY_HTTP_TIMEOUT_SEC is required when Cloudinary is configured."
	case "AVATAR_NO_URL":
		return "No avatar_url on profile."
	case "AVATAR_ALREADY_CDN":
		return "Already on Cloudinary."
	case "BILLING_SYNC_UNAVAILABLE":
		return "billing sync unavailable"
	case "BILLING_SYNC_UPSTREAM_FAILED":
		return "billing sync upstream failed"
	case "BILLING_SYNC_NO_CUSTOMER":
		return "No Paddle customer on file yet. Complete a checkout first."
	case "BILLING_SYNC_SYNCED_FMT":
		return "Synced %d completed transaction(s) from Paddle."
	case "RECEIPTS_BILL_TO_EMAIL_REQUIRED":
		return "Receipt sync requires a bill-to email on the Paddle customer or Trim profile."
	case "TELEMETRY_INVALID_PAYLOAD":
		return "invalid payload"
	case "TELEMETRY_INVALID_EVENT":
		return "invalid event"
	case "TELEMETRY_REJECTED":
		return "rejected"
	default:
		return ""
	}
}

// RoleLabel returns the display label for a workspace role code.
func RoleLabel(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "owner":
		return MessageForCode("WORKSPACE_ROLE_OWNER")
	case "admin":
		return MessageForCode("WORKSPACE_ROLE_ADMIN")
	case "member":
		return MessageForCode("WORKSPACE_ROLE_MEMBER")
	default:
		return ""
	}
}

// PlanTierLabel returns the display label for a plan tier code.
// Uses site_messages PLAN_TIER_<UPPER_ID> (no hardcoded free/pro/team/enterprise switch).
func PlanTierLabel(tier string) string {
	t := strings.ToLower(strings.TrimSpace(tier))
	if t == "" {
		return ""
	}
	return MessageForCode("PLAN_TIER_" + strings.ToUpper(t))
}

// BillingIntervalLabel returns the display label for month/year interval codes.
func BillingIntervalLabel(interval string) string {
	switch strings.ToLower(strings.TrimSpace(interval)) {
	case "month", "monthly":
		return MessageForCode("BILLING_INTERVAL_MONTH")
	case "year", "yearly", "annual":
		return MessageForCode("BILLING_INTERVAL_YEAR")
	default:
		return ""
	}
}

// EventModeLabel returns the display label for a trim event mode code.
func EventModeLabel(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "proxy":
		return MessageForCode("EVENT_MODE_PROXY")
	case "fast":
		return MessageForCode("EVENT_MODE_FAST")
	case "deep":
		return MessageForCode("EVENT_MODE_DEEP")
	case "balanced":
		return MessageForCode("EVENT_MODE_BALANCED")
	case "aggressive":
		return MessageForCode("EVENT_MODE_AGGRESSIVE")
	case "mild":
		return MessageForCode("EVENT_MODE_MILD")
	case "custom":
		return MessageForCode("EVENT_MODE_CUSTOM")
	case "local_proxy":
		return MessageForCode("EVENT_MODE_LOCAL_PROXY")
	default:
		return ""
	}
}

// SubscriptionStatusLabel returns site_messages chrome for a Paddle subscription status.
// Empty string when the status is unknown (no invent of raw Paddle strings).
func SubscriptionStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		return MessageForCode("STATUS_ACTIVE")
	case "canceled", "cancelled":
		return MessageForCode("STATUS_CANCELED")
	case "past_due":
		return MessageForCode("STATUS_PAST_DUE")
	case "trialing":
		return MessageForCode("STATUS_TRIALING")
	case "paused":
		return MessageForCode("STATUS_PAUSED")
	case "expired":
		return MessageForCode("STATUS_EXPIRED")
	case "":
		return MessageForCode("STATUS_FREE")
	default:
		// Some clients store default-tier id in status; map via DEFAULT_PLAN_TIER only.
		if IsDefaultPlanID(status) {
			return MessageForCode("STATUS_FREE")
		}
		return ""
	}
}

// EventStatusLabel returns the display label for a trim event status code.
func EventStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success":
		return MessageForCode("EVENT_STATUS_SUCCESS")
	case "error":
		return MessageForCode("EVENT_STATUS_ERROR")
	default:
		return ""
	}
}

// ReceiptStatusLabel returns the display label for a billing receipt status code.
func ReceiptStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed":
		return MessageForCode("RECEIPT_STATUS_COMPLETED")
	case "refunded":
		return MessageForCode("RECEIPT_STATUS_REFUNDED")
	case "past_due":
		return MessageForCode("RECEIPT_STATUS_PAST_DUE")
	default:
		return ""
	}
}

// InviteStatusLabel returns the display label for a workspace invite status code.
func InviteStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return MessageForCode("INVITE_STATUS_PENDING")
	case "accepted":
		return MessageForCode("INVITE_STATUS_ACCEPTED")
	case "expired":
		return MessageForCode("INVITE_STATUS_EXPIRED")
	case "revoked":
		return MessageForCode("INVITE_STATUS_REVOKED")
	default:
		return ""
	}
}

// OAuthActionCode maps a social provider id to its UI action code.
func OAuthActionCode(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "google":
		return "AUTH_OAUTH_GOOGLE"
	case "github":
		return "AUTH_OAUTH_GITHUB"
	case "gitlab":
		return "AUTH_OAUTH_GITLAB"
	default:
		return ""
	}
}

// OAuthLinkActionCode maps a social provider id to its Settings connect action code.
func OAuthLinkActionCode(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "google":
		return "AUTH_LINK_GOOGLE"
	case "github":
		return "AUTH_LINK_GITHUB"
	case "gitlab":
		return "AUTH_LINK_GITLAB"
	default:
		return ""
	}
}

// ProviderDisplayName is the short provider name for CLI phrases and UI chips.
// Uses dedicated AUTH_PROVIDER_DISPLAY_* site messages (not invent-parsed from action labels).
func ProviderDisplayName(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "google":
		return MessageForCode("AUTH_PROVIDER_DISPLAY_GOOGLE")
	case "github":
		return MessageForCode("AUTH_PROVIDER_DISPLAY_GITHUB")
	case "gitlab":
		return MessageForCode("AUTH_PROVIDER_DISPLAY_GITLAB")
	default:
		return ""
	}
}

// StaticPlanAction returns backend CTA metadata for free and enterprise catalog rows
// (those plans skip DecideChange).
func StaticPlanAction(planID, planKind string) (code, reason, label string, eligible bool) {
	defaultTier := strings.ToLower(strings.TrimSpace(MessageForCode("DEFAULT_PLAN_TIER")))
	id := strings.ToLower(strings.TrimSpace(planID))
	kind := strings.ToLower(strings.TrimSpace(planKind))
	switch {
	case defaultTier != "" && id == defaultTier:
		return "FREE_INCLUDED", MessageForCode("FREE_INCLUDED_REASON"), ActionLabelForCode("FREE_INCLUDED"), false
	case kind == "enterprise" || id == "enterprise":
		return "ENTERPRISE_CONTACT", MessageForCode("ENTERPRISE_CONTACT_REASON"), ActionLabelForCode("ENTERPRISE_CONTACT"), true
	default:
		return "", "", "", false
	}
}

func decideOK(d Decision, target PlanTarget, active *ActiveSubscription, policy Policy) (Decision, PlanTarget, *ActiveSubscription, Policy, error) {
	d.ActionLabel = ActionLabelForCode(d.Code)
	return d, target, active, policy, nil
}

// ExpireIfNeeded downgrades the user to free when their paid period has ended.
// Returns true when subscription or quota rows were updated so callers can invalidate caches.
// Enforced on auth metering and billing routes (no cron dependency).
func (s *Service) ExpireIfNeeded(ctx context.Context, userID string) (changed bool, err error) {
	tag, err := s.DB.Exec(ctx, `
		update public.subscriptions
		set status = case when status in ('active', 'trialing', 'past_due') then 'expired' else status end,
		    updated_at = now()
		where user_id = $1
		  and coalesce(expires_at, current_period_end) <= now()
		  and status in ('active', 'trialing', 'past_due', 'canceled')
	`, userID)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}

	// Only drop to free if there is no other still-active unexpired paid sub.
	active, err := s.GetActiveSubscription(ctx, userID)
	if err != nil {
		return false, err
	}
	if active != nil {
		return true, nil
	}

	// Enterprise is Paddle-only (same as Pro/Team). No active sub → drop to free.

	var freeLimit int
	defaultTier := strings.TrimSpace(MessageForCode("DEFAULT_PLAN_TIER"))
	if defaultTier == "" {
		return false, fmt.Errorf("DEFAULT_PLAN_TIER missing in site_messages")
	}
	err = s.DB.QueryRow(ctx, `
		select credits_monthly from public.plan_catalog where id = $1 and is_active = true
	`, defaultTier).Scan(&freeLimit)
	if err != nil {
		return false, fmt.Errorf("default plan credits missing in plan_catalog: %w", err)
	}

	_, err = s.DB.Exec(ctx, `
		update public.user_quotas
		set plan_tier = $3,
		    monthly_credit_limit = $2,
		    purchased_topup_credits = greatest(0, purchased_topup_credits - greatest(0, monthly_credit_used - monthly_credit_limit)),
		    monthly_credit_used = 0,
		    updated_at = now()
		where user_id = $1
	`, userID, freeLimit, defaultTier)
	if err != nil {
		return false, err
	}
	_, _ = s.DB.Exec(ctx, `
		update public.workspaces
		set plan_tier = $2, allocated_seats = 1, updated_at = now()
		where owner_id = $1::uuid
	`, userID, defaultTier)
	_, _ = s.DB.Exec(ctx, `
		update public.workspace_quotas wq
		set monthly_shared_credits = $2,
		    credits_consumed = 0,
		    updated_at = now()
		from public.workspaces w
		where w.id = wq.workspace_id and w.owner_id = $1::uuid
	`, userID, freeLimit)
	return true, nil
}

// ExpireDueBatch finds users with overdue paid periods and runs ExpireIfNeeded.
// Intended for scheduled cron (POST /internal/expire-subscriptions). Auth routes
// already call ExpireIfNeeded per request; this catches idle accounts.
func (s *Service) ExpireDueBatch(ctx context.Context, limit int) (processed int, changed int, err error) {
	if limit <= 0 {
		return 0, 0, fmt.Errorf("expire batch limit required (no invent default)")
	}
	rows, err := s.DB.Query(ctx, `
		select distinct user_id::text
		from public.subscriptions
		where coalesce(expires_at, current_period_end) <= now()
		  and status in ('active', 'trialing', 'past_due', 'canceled')
		order by user_id
		limit $1
	`, limit)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return processed, changed, err
		}
		processed++
		did, expErr := s.ExpireIfNeeded(ctx, userID)
		if expErr != nil {
			return processed, changed, expErr
		}
		if did {
			changed++
		}
	}
	return processed, changed, rows.Err()
}

func ResolvePriceID(plan PlanTarget, interval string) (priceID string, displayCents int, err error) {
	interval = normalizeInterval(interval)
	if plan.PlanKind == "topup" {
		if plan.PaddleTopup != nil && *plan.PaddleTopup != "" {
			priceID = *plan.PaddleTopup
		} else if plan.PaddleMonthly != nil && *plan.PaddleMonthly != "" {
			priceID = *plan.PaddleMonthly
		} else {
			return "", 0, fmt.Errorf("paddle price id not configured for topup plan")
		}
		if plan.PriceMonthlyCents != nil {
			displayCents = *plan.PriceMonthlyCents
		}
		return priceID, displayCents, nil
	}
	if interval == "annual" {
		if plan.PaddleYearly == nil || *plan.PaddleYearly == "" {
			return "", 0, fmt.Errorf("annual paddle price id not configured; sync plan catalog from admin")
		}
		priceID = *plan.PaddleYearly
		if plan.PriceYearlyCents != nil {
			displayCents = *plan.PriceYearlyCents
		}
		return priceID, displayCents, nil
	}
	if plan.PaddleMonthly == nil || *plan.PaddleMonthly == "" {
		return "", 0, fmt.Errorf("monthly paddle price id not configured; sync plan catalog from admin")
	}
	priceID = *plan.PaddleMonthly
	if plan.PriceMonthlyCents != nil {
		displayCents = *plan.PriceMonthlyCents
	}
	return priceID, displayCents, nil
}

// LegalSection is one privacy/terms block for public site chrome (DB-backed only).
type LegalSection struct {
	Heading      string `json:"heading,omitempty"`
	Body         string `json:"body"`
	ContactLead  string `json:"contact_lead,omitempty"`
	ContactEmail string `json:"contact_email,omitempty"`
	ContactTrail string `json:"contact_trail,omitempty"`
}

func legalContactEmail(supportEmail string) (email string, missingBody string) {
	email = strings.TrimSpace(supportEmail)
	if email == "" {
		return "", MessageForCode("LEGAL_SUPPORT_EMAIL_MISSING")
	}
	return email, ""
}

// LoadSiteMessage reads one operator-editable chrome string from public.site_messages.
// Fail-closed: empty string when the row is missing (no MessageForCode invent).
func LoadSiteMessage(ctx context.Context, db *pgxpool.Pool, code string) string {
	if db == nil || strings.TrimSpace(code) == "" {
		return ""
	}
	var body string
	err := db.QueryRow(ctx, `
		select body from public.site_messages where code = $1
	`, code).Scan(&body)
	if err != nil {
		return ""
	}
	return body
}

// LoadSiteMessages loads many site_messages rows in one query. Missing codes are omitted (fail-closed).
func LoadSiteMessages(ctx context.Context, db *pgxpool.Pool, codes []string) map[string]string {
	out := make(map[string]string, len(codes))
	if db == nil || len(codes) == 0 {
		return out
	}
	rows, err := db.Query(ctx, `
		select code, body from public.site_messages where code = any($1::text[])
	`, codes)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var code, body string
		if err := rows.Scan(&code, &body); err != nil {
			continue
		}
		out[code] = body
	}
	return out
}

// SiteMsg returns m[code] or empty (fail-closed; no invent fallback).
func SiteMsg(m map[string]string, code string) string {
	if m == nil {
		return ""
	}
	return m[code]
}

// LoadLegalSections reads privacy or terms bodies from public.site_legal_sections.
// Fail-closed: empty slice when the table has no rows (no MessageForCode invent bodies).
func LoadLegalSections(ctx context.Context, db *pgxpool.Pool, docKind, supportEmail string) ([]LegalSection, error) {
	if db == nil {
		return nil, fmt.Errorf("database required")
	}
	docKind = strings.ToLower(strings.TrimSpace(docKind))
	if docKind != "privacy" && docKind != "terms" {
		return nil, fmt.Errorf("invalid legal doc_kind")
	}
	rows, err := db.Query(ctx, `
		select coalesce(heading, ''), coalesce(body, ''),
		       coalesce(contact_lead, ''), coalesce(contact_trail, ''),
		       uses_support_email
		from public.site_legal_sections
		where doc_kind = $1
		  and published_at is not null
		order by sort_order asc
	`, docKind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	email, missing := legalContactEmail(supportEmail)
	out := make([]LegalSection, 0, 8)
	for rows.Next() {
		var sec LegalSection
		var usesEmail bool
		if err := rows.Scan(&sec.Heading, &sec.Body, &sec.ContactLead, &sec.ContactTrail, &usesEmail); err != nil {
			return nil, err
		}
		if usesEmail {
			if email != "" {
				sec.ContactEmail = email
			} else {
				sec.Body = strings.TrimSpace(strings.TrimSpace(sec.Body) + " " + missing)
				sec.ContactLead = ""
				sec.ContactTrail = ""
			}
		}
		out = append(out, sec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// LegalPrivacySections returns privacy policy body sections from the database.
func LegalPrivacySections(ctx context.Context, db *pgxpool.Pool, supportEmail string) ([]LegalSection, error) {
	return LoadLegalSections(ctx, db, "privacy", supportEmail)
}

// LegalTermsSections returns terms of service body sections from the database.
func LegalTermsSections(ctx context.Context, db *pgxpool.Pool, supportEmail string) ([]LegalSection, error) {
	return LoadLegalSections(ctx, db, "terms", supportEmail)
}
