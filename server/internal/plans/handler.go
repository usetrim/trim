package plans

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/usetrim/trim/server/internal/billing/paddleapi"
	"github.com/usetrim/trim/server/internal/billingsettings"
	"github.com/usetrim/trim/server/internal/middleware"
	"github.com/usetrim/trim/server/internal/notifications"
	"github.com/usetrim/trim/server/internal/pagination"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

type Settings struct {
	AnnualDiscountPercent       int     `json:"annual_discount_percent"`
	ApplyPaddleDiscountOnAnnual bool    `json:"apply_paddle_discount_on_annual"`
	PaddleDiscountID            *string `json:"paddle_discount_id"`
	PaddleDiscountCode          *string `json:"paddle_discount_code"`
	DefaultCurrency             string  `json:"default_currency"`
	// PricingBound is operator-gated: false until admin catalog sync binds live Paddle pri_*.
	PricingBound           bool   `json:"pricing_bound"`
	AllowDowngrades        bool   `json:"allow_downgrades"`
	AllowCancelAtPeriodEnd bool   `json:"allow_cancel_at_period_end"`
	UpgradeProrationMode   string `json:"upgrade_proration_mode"`
	// UI messages are backend-owned toast and form copy (no client invent).
	AuthRequiredPlanMessage        string `json:"auth_required_plan_message"`
	AuthRequiredEnterpriseMessage  string `json:"auth_required_enterprise_message"`
	EnterpriseMessageRequired      string `json:"enterprise_message_required"`
	PaddleJSNotReadyMessage        string `json:"paddle_js_not_ready_message"`
	CheckoutPriceMissingMessage    string `json:"checkout_price_missing_message"`
	UpgradeMissingAPIMessage       string `json:"upgrade_missing_api_message"`
	InquiryMissingConfirmMessage   string `json:"inquiry_missing_confirm_message"`
	ChangeBlockedDefaultMessage    string `json:"change_blocked_default_message"`
	CheckoutUnavailableMessage     string `json:"checkout_unavailable_message"`
	UpgradeNotAppliedMessage       string `json:"upgrade_not_applied_message"`
	CheckoutRequestFailedMessage   string `json:"checkout_request_failed_message"`
	UpgradeRequestFailedMessage    string `json:"upgrade_request_failed_message"`
	EnterpriseInquiryFailedMessage string `json:"enterprise_inquiry_failed_message"`
	PlansDialogDescription         string `json:"plans_dialog_description"`
	PlansDialogUpgradesOnlyNote    string `json:"plans_dialog_upgrades_only_note"`
	TopupDialogTitle               string `json:"topup_dialog_title"`
	TopupDialogDescription         string `json:"topup_dialog_description"`
	TopupDialogEmpty               string `json:"topup_dialog_empty"`
	AnnualToggleLabel              string `json:"annual_toggle_label"`
	MonthlyToggleLabel             string `json:"monthly_toggle_label"`
	EnterpriseDialogTitle          string `json:"enterprise_dialog_title"`
	EnterpriseDialogDescription    string `json:"enterprise_dialog_description"`
	EnterpriseEmailFallback        string `json:"enterprise_email_fallback"`
	EnterpriseCompanyLabel         string `json:"enterprise_company_label"`
	EnterpriseSeatsLabel           string `json:"enterprise_seats_label"`
	EnterpriseMessageLabel         string `json:"enterprise_message_label"`
	EnterpriseCompanyPlaceholder   string `json:"enterprise_company_placeholder"`
	EnterpriseSeatsPlaceholder     string `json:"enterprise_seats_placeholder"`
	EnterpriseMessagePlaceholder   string `json:"enterprise_message_placeholder"`
	EnterpriseCompanyDescription   string `json:"enterprise_company_description"`
	EnterpriseMessageDescription   string `json:"enterprise_message_description"`
	EnterpriseCancelLabel          string `json:"enterprise_cancel_label"`
	EnterpriseSendLabel            string `json:"enterprise_send_label"`
	EnterpriseSendPendingLabel     string `json:"enterprise_send_pending_label"`
	PlansLoadFailedMessage         string `json:"plans_load_failed_message"`
	ProrationReviewHint            string `json:"proration_review_hint"`
	SeatQuantityHint               string `json:"seat_quantity_hint"`
	SeatQuantityLabel              string `json:"seat_quantity_label"`
	PerSeatOneFmt                  string `json:"per_seat_one_fmt"`
	PerSeatManyFmt                 string `json:"per_seat_many_fmt"`
	MetaSep                        string `json:"meta_sep"`
	MaxSeatQuantity                int    `json:"max_seat_quantity"`
	MinSeatQuantity                int    `json:"min_seat_quantity"`
	DefaultSeatQuantity            int    `json:"default_seat_quantity"`
	DefaultCheckoutQuantity        int    `json:"default_checkout_quantity"`
	EnterpriseMessageRows          int    `json:"enterprise_message_rows"`
	DateRangeMonths                int    `json:"date_range_months"`
	DeepTargetTokenMin             int    `json:"deep_target_token_min"`
	DeepTargetTokenMax             int    `json:"deep_target_token_max"`
	DefaultPlanInterval            string `json:"default_plan_interval"`
	DefaultPageSize                int    `json:"default_page_size"`
	MaxPageSize                    int    `json:"max_page_size"`
	PeriodYearLabel                string `json:"period_year_label"`
	PeriodMonthLabel               string `json:"period_month_label"`
	UpgradeConfirmTitle            string `json:"upgrade_confirm_title"`
	UpgradeProrationModeFmt        string `json:"upgrade_proration_mode_fmt"`
	UpgradePlanPrefix              string `json:"upgrade_plan_prefix"`
	PlansDialogTitle               string `json:"plans_dialog_title"`
	ActivePlanPrefix               string `json:"active_plan_prefix"`
	ActivePlanExpiresFmt           string `json:"active_plan_expires_fmt"`
	ProrationPrefix                string `json:"proration_prefix"`
	UpgradeSeatsFmt                string `json:"upgrade_seats_fmt"`
	UpgradeCreditPrefix            string `json:"upgrade_credit_prefix"`
	UpgradeSubtotalPrefix          string `json:"upgrade_subtotal_prefix"`
	UpgradeTaxPrefix               string `json:"upgrade_tax_prefix"`
	UpgradeDuePrefix               string `json:"upgrade_due_prefix"`
	UpgradePreviewEmpty            string `json:"upgrade_preview_empty"`
	UpgradeCancelLabel             string `json:"upgrade_cancel_label"`
	DialogCloseLabel               string `json:"dialog_close_label"`
	UnlimitedFeatureLabel          string `json:"unlimited_feature_label"`
}

type Plan struct {
	ID                string          `json:"id"`
	DisplayName       string          `json:"display_name"`
	Description       string          `json:"description"`
	PlanKind          string          `json:"plan_kind"`
	PlanRank          int             `json:"plan_rank"`
	CurrencyCode      string          `json:"currency_code"`
	PriceMonthlyCents *int            `json:"price_monthly_cents"`
	PriceYearlyCents  *int            `json:"price_yearly_cents"`
	CreditsMonthly    int             `json:"credits_monthly"`
	PerSeat           bool            `json:"per_seat"`
	Unlimited         bool            `json:"unlimited"`
	Popular           bool            `json:"popular"`
	Features          json.RawMessage `json:"features"`
	SortOrder         int             `json:"sort_order"`
	CheckoutAvailable bool            `json:"checkout_available"`
	EffectiveCents    *int            `json:"effective_cents"`
	EffectiveInterval string          `json:"effective_interval"`
	SavingsPercent    *int            `json:"savings_percent,omitempty"`
	// SavingsLabel is backend-formatted annual savings badge text (e.g. "Save 20%").
	SavingsLabel    string  `json:"savings_label,omitempty"`
	UpgradeEligible *bool   `json:"upgrade_eligible,omitempty"`
	ChangeCode      string  `json:"change_code,omitempty"`
	ChangeReason    string  `json:"change_reason,omitempty"`
	ChangeLabel     string  `json:"change_label,omitempty"`
	PendingLabel    string  `json:"pending_label,omitempty"`
	CanProceed      *bool   `json:"can_proceed,omitempty"`
	PaddlePriceID   *string `json:"-"`
}

type ListResponse struct {
	Interval    string   `json:"interval"`
	Settings    Settings `json:"settings"`
	Plans       []Plan   `json:"plans"`
	MoneyLocale string   `json:"money_locale"`
}

type CheckoutRequest struct {
	PlanID   string `json:"plan_id"`
	Interval string `json:"interval"`
	Quantity int    `json:"quantity"`
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	// InquiryID binds Enterprise checkout to a sales-offered inquiry (Paddle-only fulfill).
	InquiryID string `json:"inquiry_id"`
	// Confirm must be true to apply an in-place Paddle upgrade after the user saw proration totals.
	Confirm bool `json:"confirm"`
}

// CheckoutResponse drives the client:
// action=new_checkout -> open Paddle.js Checkout
// action=upgrade -> server already patched the Paddle subscription; show success
// action=blocked / same_plan -> show reason, do not open checkout
type CheckoutResponse struct {
	Action               string                 `json:"action"`
	Code                 string                 `json:"code"`
	Message              string                 `json:"message"`
	PlanID               string                 `json:"plan_id"`
	PlanDisplayName      string                 `json:"plan_display_name,omitempty"`
	Interval             string                 `json:"interval"`
	IntervalLabel        string                 `json:"interval_label,omitempty"`
	Quantity             int                    `json:"quantity"`
	PriceID              string                 `json:"price_id,omitempty"`
	DiscountID           *string                `json:"discount_id,omitempty"`
	DiscountCode         *string                `json:"discount_code,omitempty"`
	CustomData           map[string]interface{} `json:"custom_data,omitempty"`
	CurrencyCode         string                 `json:"currency_code,omitempty"`
	DisplayCents         int                    `json:"display_cents,omitempty"`
	ProrationMode        string                 `json:"proration_mode,omitempty"`
	PaddleSubscriptionID string                 `json:"paddle_subscription_id,omitempty"`
	ProrationPreview     *ProrationPreview      `json:"proration_preview,omitempty"`
	Decision             subscriptions.Decision `json:"decision"`
	// ActionLabel is the confirm / primary button text (backend-driven).
	ActionLabel string `json:"action_label,omitempty"`
	// PendingLabel is the button spinner text for this checkout action (backend-driven).
	PendingLabel string `json:"pending_label,omitempty"`
}

// ProrationPreview surfaces Paddle preview totals so the client can confirm upgrade cost.
type ProrationPreview struct {
	CurrencyCode string `json:"currency_code,omitempty"`
	Subtotal     string `json:"subtotal,omitempty"`
	Tax          string `json:"tax,omitempty"`
	Credit       string `json:"credit,omitempty"`
	Balance      string `json:"balance,omitempty"`
	GrandTotal   string `json:"grand_total,omitempty"`
}

func fillCheckoutLabels(resp *CheckoutResponse, target subscriptions.PlanTarget) {
	if resp == nil {
		return
	}
	resp.PlanDisplayName = strings.TrimSpace(target.DisplayName)
	if resp.PlanDisplayName == "" {
		resp.PlanDisplayName = subscriptions.PlanTierLabel(resp.PlanID)
	}
	resp.IntervalLabel = subscriptions.BillingIntervalLabel(resp.Interval)
}

type SubscriptionStatusResponse struct {
	HasActivePaid        bool   `json:"has_active_paid"`
	IsExpired            bool   `json:"is_expired"`
	PlanTier             string `json:"plan_tier"`
	PlanTierLabel        string `json:"plan_tier_label,omitempty"`
	PlanRank             int    `json:"plan_rank"`
	BillingInterval      string `json:"billing_interval,omitempty"`
	BillingIntervalLabel string `json:"billing_interval_label,omitempty"`
	Status               string `json:"status,omitempty"`
	// StatusLabel is the human status chip (free / active / expired). Backend-owned.
	StatusLabel string `json:"status_label"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	// ExpiresAtLabel is a UTC display string so clients never invent toLocaleString.
	ExpiresAtLabel         string `json:"expires_at_label,omitempty"`
	PeriodStart            string `json:"period_start,omitempty"`
	PeriodEnd              string `json:"period_end,omitempty"`
	AllowDowngrades        bool   `json:"allow_downgrades"`
	AllowCancelAtPeriodEnd bool   `json:"allow_cancel_at_period_end"`
	UpgradeProration       string `json:"upgrade_proration_mode"`
	// DowngradePolicyMessage explains upgrade-only policy when allow_downgrades is false.
	DowngradePolicyMessage string `json:"downgrade_policy_message,omitempty"`
	PaddleSubscriptionID   string `json:"paddle_subscription_id,omitempty"`
	// PrimaryActionLabel is the dashboard CTA that opens the plan modal (backend-driven).
	PrimaryActionLabel string `json:"primary_action_label"`
	// PortalActionLabel / PortalPendingLabel drive the Manage billing button when paid.
	PortalActionLabel       string `json:"portal_action_label,omitempty"`
	PortalPendingLabel      string `json:"portal_pending_label,omitempty"`
	PortalURLMissingMessage string `json:"portal_url_missing_message,omitempty"`
	// ReceiptsSyncActionLabel / ReceiptsSyncPendingLabel drive Sync from Paddle.
	ReceiptsSyncActionLabel  string `json:"receipts_sync_action_label,omitempty"`
	ReceiptsSyncPendingLabel string `json:"receipts_sync_pending_label,omitempty"`
	// AvatarSyncActionLabel / AvatarSyncPendingLabel drive Sync avatar.
	AvatarSyncActionLabel  string `json:"avatar_sync_action_label,omitempty"`
	AvatarSyncPendingLabel string `json:"avatar_sync_pending_label,omitempty"`
	// SignOutPendingLabel drives the Sign out spinner (action is local Supabase).
	SignOutActionLabel  string `json:"sign_out_action_label,omitempty"`
	SignOutPendingLabel string `json:"sign_out_pending_label,omitempty"`
	// AccountDeleteActionLabel / AccountDeletePendingLabel drive GDPR delete.
	AccountDeleteActionLabel  string `json:"account_delete_action_label,omitempty"`
	AccountDeletePendingLabel string `json:"account_delete_pending_label,omitempty"`
	// AccountDeleteConfirmMessage is the destructive confirm dialog body.
	AccountDeleteConfirmMessage string `json:"account_delete_confirm_message,omitempty"`
	AccountDeleteFailedMessage  string `json:"account_delete_failed_message,omitempty"`
	// Dashboard chrome (page titles, metric labels, nav). Backend-owned; no client invent.
	PageEyebrow               string `json:"page_eyebrow,omitempty"`
	WelcomePrefix             string `json:"welcome_prefix,omitempty"`
	WelcomeGuest              string `json:"welcome_guest,omitempty"`
	PageDescription           string `json:"page_description,omitempty"`
	StatusPrefix              string `json:"status_prefix,omitempty"`
	TierPrefix                string `json:"tier_prefix,omitempty"`
	IntervalPrefix            string `json:"interval_prefix,omitempty"`
	RenewsPrefix              string `json:"renews_prefix,omitempty"`
	NavTeamLabel              string `json:"nav_team_label,omitempty"`
	NavSettingsLabel          string `json:"nav_settings_label,omitempty"`
	NavTracesLabel            string `json:"nav_traces_label,omitempty"`
	NavReceiptsLabel          string `json:"nav_receipts_label,omitempty"`
	NavEnterpriseLabel        string `json:"nav_enterprise_label,omitempty"`
	MetricPlanLabel           string `json:"metric_plan_label,omitempty"`
	MetricCreditsLabel        string `json:"metric_credits_label,omitempty"`
	MetricTopupLabel          string `json:"metric_topup_label,omitempty"`
	MetricRemainingLabel      string `json:"metric_remaining_label,omitempty"`
	MetricTokensSavedLabel    string `json:"metric_tokens_saved_label,omitempty"`
	MetricTabLabel            string `json:"metric_tab_label,omitempty"`
	MetricLinesAddedLabel     string `json:"metric_lines_added_label,omitempty"`
	MetricLinesDeletedLabel   string `json:"metric_lines_deleted_label,omitempty"`
	QuotaExhaustedTitle       string `json:"quota_exhausted_title,omitempty"`
	QuotaExhaustedBody        string `json:"quota_exhausted_body,omitempty"`
	BuyTopupLabel             string `json:"buy_topup_label,omitempty"`
	AcceptanceHint            string `json:"acceptance_hint,omitempty"`
	ChartTokenSeriesTitle     string `json:"chart_token_series_title,omitempty"`
	ChartModelsTitle          string `json:"chart_models_title,omitempty"`
	ChartOutcomesTitle        string `json:"chart_outcomes_title,omitempty"`
	ChartModesTitle           string `json:"chart_modes_title,omitempty"`
	TracesTitle               string `json:"traces_title,omitempty"`
	ReceiptsTitle             string `json:"receipts_title,omitempty"`
	AccountTitle              string `json:"account_title,omitempty"`
	AvatarHint                string `json:"avatar_hint,omitempty"`
	DeleteHint                string `json:"delete_hint,omitempty"`
	SignedInPrefix            string `json:"signed_in_prefix,omitempty"`
	EventsSuffix              string `json:"events_suffix,omitempty"`
	MetaSep                   string `json:"meta_sep,omitempty"`
	DefaultPageSize           int    `json:"default_page_size,omitempty"`
	AvatarSyncFailedMessage   string `json:"avatar_sync_failed_message,omitempty"`
	ReceiptsSyncFailedMessage string `json:"receipts_sync_failed_message,omitempty"`
	TracesColWhen             string `json:"traces_col_when,omitempty"`
	TracesColModel            string `json:"traces_col_model,omitempty"`
	TracesColMode             string `json:"traces_col_mode,omitempty"`
	TracesColTokens           string `json:"traces_col_tokens,omitempty"`
	TracesColLatency          string `json:"traces_col_latency,omitempty"`
	ReceiptsColDate           string `json:"receipts_col_date,omitempty"`
	ReceiptsColInvoice        string `json:"receipts_col_invoice,omitempty"`
	ReceiptsColStatus         string `json:"receipts_col_status,omitempty"`
	ReceiptsColTotal          string `json:"receipts_col_total,omitempty"`
	ReceiptsColView           string `json:"receipts_col_view,omitempty"`
	PrivacyLinkLabel          string `json:"privacy_link_label,omitempty"`
	TermsLinkLabel            string `json:"terms_link_label,omitempty"`
	// App paths from site_messages (no client /dashboard invent).
	PathDashboard      string `json:"path_dashboard,omitempty"`
	PathTeam           string `json:"path_team,omitempty"`
	PathSettings       string `json:"path_settings,omitempty"`
	PathTraces         string `json:"path_traces,omitempty"`
	PathReceipts       string `json:"path_receipts,omitempty"`
	PathEnterprise     string `json:"path_enterprise,omitempty"`
	PathReceiptsPrefix string `json:"path_receipts_prefix,omitempty"`
	PathPrivacy        string `json:"path_privacy,omitempty"`
	PathTerms          string `json:"path_terms,omitempty"`
	PathHome           string `json:"path_home,omitempty"`
	PathUpgrade        string `json:"path_upgrade,omitempty"`
	MoneyLocale        string `json:"money_locale,omitempty"`
}

type Handler struct {
	DB                         *pgxpool.Pool
	ReadDB                     *pgxpool.Pool
	Subs                       *subscriptions.Service
	Paddle                     *paddleapi.Client
	Redis                      redis.UniversalClient
	EnterpriseNotifyURL        string
	EnterpriseNotifyTimeoutSec int
}

func NewHandler(db, readDB *pgxpool.Pool, paddle *paddleapi.Client, rdb *redis.Client, enterpriseNotifyURL string, enterpriseNotifyTimeoutSec int) *Handler {
	return &Handler{
		DB:                         db,
		ReadDB:                     readDB,
		Subs:                       subscriptions.NewService(db, readDB),
		Paddle:                     paddle,
		Redis:                      rdb,
		EnterpriseNotifyURL:        strings.TrimSpace(enterpriseNotifyURL),
		EnterpriseNotifyTimeoutSec: enterpriseNotifyTimeoutSec,
	}
}

func (h *Handler) readPool() *pgxpool.Pool {
	if h.ReadDB != nil {
		return h.ReadDB
	}
	return h.DB
}

func writeJSONErr(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": subscriptions.MessageForCode(code),
	})
}

func (h *Handler) ListPublic(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		writeJSONErr(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}

	ctx := r.Context()
	settings, err := h.loadSettings(ctx)
	if err != nil {
		if errors.Is(err, subscriptions.ErrPricingUnbound) {
			writeJSONErr(w, http.StatusServiceUnavailable, "BILLING_PRICING_UNBOUND")
			return
		}
		writeJSONErr(w, http.StatusInternalServerError, "PLAN_OPERATION_FAILED")
		return
	}

	interval := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("interval")))
	if interval == "" {
		interval = strings.ToLower(strings.TrimSpace(settings.DefaultPlanInterval))
	}
	if interval != "monthly" && interval != "annual" {
		writeJSONErr(w, http.StatusBadRequest, "PLAN_INTERVAL_INVALID")
		return
	}

	rows, err := h.readPool().Query(ctx, `
		select id, display_name, description, plan_kind, plan_rank, currency_code,
		       price_monthly_cents, price_yearly_cents, credits_monthly, per_seat, unlimited, popular,
		       features, sort_order,
		       paddle_price_id_monthly, paddle_price_id_yearly, paddle_price_id_topup
		from public.plan_catalog
		where is_public = true and is_active = true
		order by sort_order asc
	`)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "PLAN_OPERATION_FAILED")
		return
	}
	defer rows.Close()

	plans := make([]Plan, 0)
	for rows.Next() {
		var p Plan
		var monthlyID, yearlyID, topupID *string
		if err := rows.Scan(
			&p.ID, &p.DisplayName, &p.Description, &p.PlanKind, &p.PlanRank, &p.CurrencyCode,
			&p.PriceMonthlyCents, &p.PriceYearlyCents, &p.CreditsMonthly, &p.PerSeat, &p.Unlimited, &p.Popular,
			&p.Features, &p.SortOrder, &monthlyID, &yearlyID, &topupID,
		); err != nil {
			writeJSONErr(w, http.StatusInternalServerError, "PLAN_OPERATION_FAILED")
			return
		}

		p.EffectiveInterval = interval
		switch {
		case p.PlanKind == "topup":
			// One-shot credit packs: bind paddle_price_id_topup (monthly price id is not the source of truth).
			p.EffectiveInterval = "monthly"
			p.EffectiveCents = p.PriceMonthlyCents
			if topupID != nil && *topupID != "" {
				p.PaddlePriceID = topupID
				p.CheckoutAvailable = true
			} else if monthlyID != nil && *monthlyID != "" {
				// Legacy catalogs that stored top-up pri_* in the monthly column.
				p.PaddlePriceID = monthlyID
				p.CheckoutAvailable = true
			} else {
				p.CheckoutAvailable = false
			}
		case interval == "annual":
			p.PaddlePriceID = yearlyID
			// Fail closed: never invent yearly display from monthly × discount.
			// Operators must set price_yearly_cents on plan_catalog.
			if p.PriceYearlyCents != nil {
				p.EffectiveCents = p.PriceYearlyCents
			}
			// Fail closed: savings badge only from catalog yearly vs monthly math.
			// Do not invent AnnualDiscountPercent as a display substitute when cents math is <1% or missing.
			// Round to nearest percent so badge matches SQL round() used when rebaking yearly cents.
			if p.PriceMonthlyCents != nil && p.PriceYearlyCents != nil && *p.PriceMonthlyCents > 0 {
				fullYear := (*p.PriceMonthlyCents) * 12
				if fullYear > *p.PriceYearlyCents {
					computed := int(math.Round((float64(fullYear-*p.PriceYearlyCents) / float64(fullYear)) * 100))
					if computed >= 1 {
						p.SavingsPercent = &computed
					}
				}
			}
			// Fail closed: annual checkout requires a dedicated yearly Paddle price ID.
			// Do not invent monthly price + Paddle discount as a substitute.
			p.CheckoutAvailable = yearlyID != nil && *yearlyID != ""
		default:
			p.EffectiveCents = p.PriceMonthlyCents
			p.PaddlePriceID = monthlyID
			p.CheckoutAvailable = (monthlyID != nil && *monthlyID != "") || p.PlanKind == "enterprise" || subscriptions.IsDefaultPlanID(p.ID)
		}

		if p.PlanKind == "enterprise" || subscriptions.IsDefaultPlanID(p.ID) {
			p.CheckoutAvailable = subscriptions.IsDefaultPlanID(p.ID) || p.PlanKind == "enterprise"
		}

		if p.SavingsPercent != nil && *p.SavingsPercent > 0 {
			fmtStr := subscriptions.MessageForCode("PLANS_ANNUAL_SAVINGS_FMT")
			// Fail closed: no invent badge when chrome row missing.
			if strings.TrimSpace(fmtStr) != "" {
				p.SavingsLabel = fmt.Sprintf(fmtStr, *p.SavingsPercent)
			}
		}

		if p.Unlimited {
			p.Features = ensureUnlimitedFeature(p.Features)
		}

		plans = append(plans, p)
	}

	// Annotate every plan with a backend-driven CTA label (no client inventing).
	userID := middleware.UserIDFromContext(ctx)
	if userID != "" {
		_, _ = h.Subs.ExpireIfNeeded(ctx, userID)
	}
	for i := range plans {
		if code, reason, label, eligible := subscriptions.StaticPlanAction(plans[i].ID, plans[i].PlanKind); code != "" {
			plans[i].ChangeCode = code
			plans[i].ChangeReason = reason
			plans[i].ChangeLabel = label
			plans[i].PendingLabel = subscriptions.PendingLabelForCode(code)
			plans[i].UpgradeEligible = &eligible
			plans[i].CanProceed = &eligible
			continue
		}

		if !plans[i].CheckoutAvailable {
			plans[i].ChangeCode = "PRICE_NOT_CONFIGURED"
			plans[i].ChangeReason = subscriptions.MessageForCode("PRICE_NOT_CONFIGURED_REASON")
			plans[i].ChangeLabel = subscriptions.ActionLabelForCode("PRICE_NOT_CONFIGURED")
			plans[i].PendingLabel = subscriptions.PendingLabelForCode("PRICE_NOT_CONFIGURED")
			eligible := false
			plans[i].UpgradeEligible = &eligible
			plans[i].CanProceed = &eligible
			continue
		}

		if userID == "" {
			plans[i].ChangeCode = "NEW_CHECKOUT"
			plans[i].ChangeLabel = subscriptions.ActionLabelForCode("NEW_CHECKOUT")
			plans[i].PendingLabel = subscriptions.PendingLabelForCode("NEW_CHECKOUT")
			eligible := true
			plans[i].UpgradeEligible = &eligible
			plans[i].CanProceed = &eligible
			continue
		}
		d, _, _, _, err := h.Subs.DecideChange(ctx, userID, plans[i].ID, interval)
		if err != nil {
			plans[i].ChangeCode = "CHANGE_NOT_ALLOWED"
			plans[i].ChangeLabel = subscriptions.ActionLabelForCode("CHANGE_NOT_ALLOWED")
			plans[i].PendingLabel = subscriptions.PendingLabelForCode("CHANGE_NOT_ALLOWED")
			plans[i].ChangeReason = err.Error()
			eligible := false
			plans[i].UpgradeEligible = &eligible
			plans[i].CanProceed = &eligible
			continue
		}
		eligible := d.CanProceed
		plans[i].UpgradeEligible = &eligible
		plans[i].CanProceed = &eligible
		plans[i].ChangeCode = d.Code
		plans[i].ChangeReason = d.Reason
		plans[i].ChangeLabel = d.ActionLabel
		if d.ActionLabel == "" {
			plans[i].ChangeLabel = subscriptions.ActionLabelForCode(d.Code)
		}
		plans[i].PendingLabel = subscriptions.PendingLabelForCode(d.Code)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ListResponse{
		Interval:    interval,
		Settings:    settings,
		Plans:       plans,
		MoneyLocale: subscriptions.MessageForCode("SITE_HTML_LANG"),
	})
}

func (h *Handler) GetSubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		writeJSONErr(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}

	ctx := r.Context()
	_, _ = h.Subs.ExpireIfNeeded(ctx, userID)

	policy, err := h.Subs.LoadPolicy(ctx)
	if err != nil {
		if errors.Is(err, subscriptions.ErrPricingUnbound) {
			writeJSONErr(w, http.StatusServiceUnavailable, "BILLING_PRICING_UNBOUND")
			return
		}
		writeJSONErr(w, http.StatusInternalServerError, "PLAN_OPERATION_FAILED")
		return
	}

	active, err := h.Subs.GetActiveSubscription(ctx, userID)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "PLAN_OPERATION_FAILED")
		return
	}

	defaultTier := strings.TrimSpace(subscriptions.MessageForCode("DEFAULT_PLAN_TIER"))
	if defaultTier == "" {
		writeJSONErr(w, http.StatusInternalServerError, "DEFAULT_PLAN_TIER_MISSING")
		return
	}

	resp := SubscriptionStatusResponse{
		AllowDowngrades:             policy.AllowDowngrades,
		AllowCancelAtPeriodEnd:      policy.AllowCancelAtPeriodEnd,
		UpgradeProration:            policy.UpgradeProrationMode,
		IsExpired:                   false,
		PlanTier:                    defaultTier,
		PlanTierLabel:               subscriptions.PlanTierLabel(defaultTier),
		StatusLabel:                 subscriptions.MessageForCode("STATUS_FREE"),
		ReceiptsSyncActionLabel:     subscriptions.ActionLabelForCode("RECEIPTS_SYNC"),
		ReceiptsSyncPendingLabel:    subscriptions.PendingLabelForCode("RECEIPTS_SYNC"),
		AvatarSyncActionLabel:       subscriptions.ActionLabelForCode("AVATAR_SYNC"),
		AvatarSyncPendingLabel:      subscriptions.PendingLabelForCode("AVATAR_SYNC"),
		SignOutActionLabel:          subscriptions.ActionLabelForCode("SIGN_OUT"),
		SignOutPendingLabel:         subscriptions.PendingLabelForCode("SIGN_OUT"),
		AccountDeleteActionLabel:    subscriptions.ActionLabelForCode("ACCOUNT_DELETE"),
		AccountDeletePendingLabel:   subscriptions.PendingLabelForCode("ACCOUNT_DELETE"),
		AccountDeleteConfirmMessage: subscriptions.MessageForCode("ACCOUNT_DELETE_CONFIRM"),
		AccountDeleteFailedMessage:  subscriptions.MessageForCode("ACCOUNT_DELETE_FAILED"),
		PageEyebrow:                 subscriptions.MessageForCode("DASHBOARD_PAGE_EYEBROW"),
		WelcomePrefix:               subscriptions.MessageForCode("DASHBOARD_WELCOME_PREFIX"),
		WelcomeGuest:                subscriptions.MessageForCode("DASHBOARD_WELCOME_GUEST"),
		PageDescription:             subscriptions.MessageForCode("DASHBOARD_PAGE_DESCRIPTION"),
		StatusPrefix:                subscriptions.MessageForCode("DASHBOARD_STATUS_PREFIX"),
		TierPrefix:                  subscriptions.MessageForCode("DASHBOARD_TIER_PREFIX"),
		IntervalPrefix:              subscriptions.MessageForCode("DASHBOARD_INTERVAL_PREFIX"),
		RenewsPrefix:                subscriptions.MessageForCode("DASHBOARD_RENEWS_PREFIX"),
		NavTeamLabel:                subscriptions.MessageForCode("DASHBOARD_NAV_TEAM"),
		NavSettingsLabel:            subscriptions.MessageForCode("DASHBOARD_NAV_SETTINGS"),
		NavTracesLabel:              subscriptions.MessageForCode("DASHBOARD_NAV_TRACES"),
		NavReceiptsLabel:            subscriptions.MessageForCode("DASHBOARD_NAV_RECEIPTS"),
		NavEnterpriseLabel:          subscriptions.MessageForCode("DASHBOARD_NAV_ENTERPRISE"),
		MetricPlanLabel:             subscriptions.MessageForCode("DASHBOARD_METRIC_PLAN"),
		MetricCreditsLabel:          subscriptions.MessageForCode("DASHBOARD_METRIC_CREDITS"),
		MetricTopupLabel:            subscriptions.MessageForCode("DASHBOARD_METRIC_TOPUP"),
		MetricRemainingLabel:        subscriptions.MessageForCode("DASHBOARD_METRIC_REMAINING"),
		MetricTokensSavedLabel:      subscriptions.MessageForCode("DASHBOARD_METRIC_TOKENS_SAVED"),
		MetricTabLabel:              subscriptions.MessageForCode("DASHBOARD_METRIC_TAB"),
		MetricLinesAddedLabel:       subscriptions.MessageForCode("DASHBOARD_METRIC_LINES_ADDED"),
		MetricLinesDeletedLabel:     subscriptions.MessageForCode("DASHBOARD_METRIC_LINES_DELETED"),
		QuotaExhaustedTitle:         subscriptions.MessageForCode("DASHBOARD_QUOTA_EXHAUSTED_TITLE"),
		QuotaExhaustedBody:          subscriptions.MessageForCode("DASHBOARD_QUOTA_EXHAUSTED_BODY"),
		BuyTopupLabel:               subscriptions.MessageForCode("DASHBOARD_BUY_TOPUP_LABEL"),
		AcceptanceHint:              subscriptions.MessageForCode("DASHBOARD_ACCEPTANCE_HINT"),
		ChartTokenSeriesTitle:       subscriptions.MessageForCode("DASHBOARD_CHART_TOKEN_SERIES"),
		ChartModelsTitle:            subscriptions.MessageForCode("DASHBOARD_CHART_MODELS"),
		ChartOutcomesTitle:          subscriptions.MessageForCode("DASHBOARD_CHART_OUTCOMES"),
		ChartModesTitle:             subscriptions.MessageForCode("DASHBOARD_CHART_MODES"),
		TracesTitle:                 subscriptions.MessageForCode("DASHBOARD_TRACES_TITLE"),
		ReceiptsTitle:               subscriptions.MessageForCode("DASHBOARD_RECEIPTS_TITLE"),
		AccountTitle:                subscriptions.MessageForCode("DASHBOARD_ACCOUNT_TITLE"),
		AvatarHint:                  subscriptions.MessageForCode("DASHBOARD_AVATAR_HINT"),
		DeleteHint:                  subscriptions.MessageForCode("DASHBOARD_DELETE_HINT"),
		SignedInPrefix:              subscriptions.MessageForCode("DASHBOARD_SIGNED_IN_PREFIX"),
		EventsSuffix:                subscriptions.MessageForCode("DASHBOARD_EVENTS_SUFFIX"),
		MetaSep:                     subscriptions.MessageForCode("DASHBOARD_META_SEP"),
		AvatarSyncFailedMessage:     subscriptions.MessageForCode("AVATAR_SYNC_FAILED"),
		ReceiptsSyncFailedMessage:   subscriptions.MessageForCode("RECEIPTS_SYNC_FAILED"),
		TracesColWhen:               subscriptions.MessageForCode("EVENTS_COL_WHEN"),
		TracesColModel:              subscriptions.MessageForCode("EVENTS_COL_MODEL"),
		TracesColMode:               subscriptions.MessageForCode("EVENTS_COL_MODE"),
		TracesColTokens:             subscriptions.MessageForCode("EVENTS_COL_TOKENS"),
		TracesColLatency:            subscriptions.MessageForCode("EVENTS_COL_LATENCY"),
		ReceiptsColDate:             subscriptions.MessageForCode("RECEIPTS_COL_DATE"),
		ReceiptsColInvoice:          subscriptions.MessageForCode("RECEIPTS_COL_INVOICE"),
		ReceiptsColStatus:           subscriptions.MessageForCode("RECEIPTS_COL_STATUS"),
		ReceiptsColTotal:            subscriptions.MessageForCode("RECEIPTS_COL_TOTAL"),
		ReceiptsColView:             subscriptions.MessageForCode("RECEIPTS_COL_VIEW"),
		PrivacyLinkLabel:            subscriptions.MessageForCode("LEGAL_LINK_PRIVACY"),
		TermsLinkLabel:              subscriptions.MessageForCode("LEGAL_LINK_TERMS"),
		PathDashboard:               subscriptions.MessageForCode("APP_PATH_DASHBOARD"),
		PathTeam:                    subscriptions.MessageForCode("APP_PATH_TEAM"),
		PathSettings:                subscriptions.MessageForCode("APP_PATH_SETTINGS"),
		PathTraces:                  subscriptions.MessageForCode("APP_PATH_TRACES"),
		PathReceipts:                subscriptions.MessageForCode("APP_PATH_RECEIPTS"),
		PathEnterprise:              subscriptions.MessageForCode("APP_PATH_ENTERPRISE"),
		PathReceiptsPrefix:          subscriptions.MessageForCode("APP_PATH_RECEIPTS_PREFIX"),
		PathPrivacy:                 subscriptions.MessageForCode("APP_PATH_PRIVACY"),
		PathTerms:                   subscriptions.MessageForCode("APP_PATH_TERMS"),
		PathHome:                    subscriptions.MessageForCode("APP_PATH_HOME"),
		PathUpgrade:                 subscriptions.MessageForCode("APP_PATH_UPGRADE"),
		MoneyLocale:                 subscriptions.MessageForCode("SITE_HTML_LANG"),
	}
	db := h.readPool()
	_ = db.QueryRow(ctx, `
		select coalesce(default_page_size, 0) from public.billing_settings where id = 'default'
	`).Scan(&resp.DefaultPageSize)
	if !policy.AllowDowngrades {
		untilLabel := ""
		if active != nil {
			untilLabel = strings.TrimSpace(subscriptions.FormatUTCDateTime(active.ExpiresAt))
			if untilLabel == "" {
				untilLabel = strings.TrimSpace(subscriptions.FormatUTCDateTime(active.PeriodEnd))
			}
		}
		if untilLabel != "" {
			if fmtMsg := subscriptions.MessageForCode("DOWNGRADE_POLICY_UNTIL_FMT"); fmtMsg != "" {
				resp.DowngradePolicyMessage = fmt.Sprintf(fmtMsg, untilLabel)
			}
		}
		if resp.DowngradePolicyMessage == "" {
			resp.DowngradePolicyMessage = subscriptions.MessageForCode("DOWNGRADE_POLICY")
		}
	}

	if active != nil {
		resp.HasActivePaid = true
		resp.IsExpired = false
		resp.PlanTier = active.PlanTier
		resp.PlanTierLabel = subscriptions.PlanTierLabel(active.PlanTier)
		resp.PlanRank = active.PlanRank
		resp.BillingInterval = active.BillingInterval
		resp.BillingIntervalLabel = subscriptions.BillingIntervalLabel(active.BillingInterval)
		resp.Status = active.Status
		resp.StatusLabel = subscriptions.SubscriptionStatusLabel(active.Status)
		resp.ExpiresAt = active.ExpiresAt.UTC().Format(timeRFC3339)
		resp.ExpiresAtLabel = subscriptions.FormatUTCDateTime(active.ExpiresAt)
		resp.PeriodStart = active.PeriodStart.UTC().Format(timeRFC3339)
		resp.PeriodEnd = active.PeriodEnd.UTC().Format(timeRFC3339)
		resp.PaddleSubscriptionID = active.PaddleSubscriptionID
		resp.PrimaryActionLabel = subscriptions.ActionLabelForCode("DASHBOARD_CHANGE_PLAN")
		resp.PortalActionLabel = subscriptions.ActionLabelForCode("PORTAL_OPEN")
		resp.PortalPendingLabel = subscriptions.PendingLabelForCode("PORTAL_OPEN")
		resp.PortalURLMissingMessage = subscriptions.MessageForCode("PORTAL_URL_MISSING")
	} else {
		var tier string
		_ = db.QueryRow(ctx, `
			select coalesce(uq.plan_tier, '')
			from public.user_quotas uq
			where uq.user_id = $1::uuid
		`, userID).Scan(&tier)
		if strings.TrimSpace(tier) != "" {
			resp.PlanTier = strings.TrimSpace(tier)
		}
		resp.PlanTierLabel = subscriptions.PlanTierLabel(resp.PlanTier)
		// is_expired only when a prior paid period ended (never invent for never-paid free).
		// Enterprise entitlements require an active Paddle subscription (same as Pro/Team).
		var hadExpiredPaid bool
		_ = db.QueryRow(ctx, `
			select exists(
				select 1 from public.subscriptions
				where user_id = $1
				  and (
				    status = 'expired'
				    or (status = 'canceled' and coalesce(expires_at, current_period_end) <= now())
				  )
			)
		`, userID).Scan(&hadExpiredPaid)
		if hadExpiredPaid {
			resp.IsExpired = true
			resp.StatusLabel = subscriptions.MessageForCode("STATUS_EXPIRED")
			resp.PrimaryActionLabel = subscriptions.ActionLabelForCode("DASHBOARD_UPGRADE")
		} else {
			resp.IsExpired = false
			resp.StatusLabel = subscriptions.MessageForCode("STATUS_FREE")
			resp.PrimaryActionLabel = subscriptions.ActionLabelForCode("DASHBOARD_UPGRADE")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

func (h *Handler) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		writeJSONErr(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}

	var req CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "PLAN_INVALID_JSON_BODY")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	// Never trust client-supplied user_id for billing mutations.
	req.UserID = userID
	ctx := r.Context()

	req.PlanID = strings.TrimSpace(req.PlanID)
	req.Interval = strings.ToLower(strings.TrimSpace(req.Interval))
	req.InquiryID = strings.TrimSpace(req.InquiryID)
	if req.InquiryID != "" {
		var inquiryUser string
		var inquiryStatus string
		var offeredSeats *int
		iqErr := h.DB.QueryRow(ctx, `
			select coalesce(user_id::text, ''), status, offered_seat_quantity
			from public.enterprise_inquiries
			where id = $1::uuid
		`, req.InquiryID).Scan(&inquiryUser, &inquiryStatus, &offeredSeats)
		if iqErr != nil {
			if errors.Is(iqErr, pgx.ErrNoRows) {
				writeJSONErr(w, http.StatusNotFound, "ENTERPRISE_INQUIRY_REQUIRED")
				return
			}
			writeJSONErr(w, http.StatusInternalServerError, "PLAN_OPERATION_FAILED")
			return
		}
		if strings.TrimSpace(inquiryUser) != userID {
			writeJSONErr(w, http.StatusForbidden, "ENTERPRISE_INQUIRY_REQUIRED")
			return
		}
		if !strings.EqualFold(strings.TrimSpace(inquiryStatus), "offered") {
			writeJSONErr(w, http.StatusConflict, "ENTERPRISE_INQUIRY_NOT_OFFERED")
			return
		}
		if offeredSeats == nil || *offeredSeats < 1 {
			writeJSONErr(w, http.StatusConflict, "ENTERPRISE_INQUIRY_SEATS_MISSING")
			return
		}
		var enterprisePlanID string
		if err := h.DB.QueryRow(ctx, `
			select id from public.plan_catalog
			where is_active = true and plan_kind = 'enterprise'
			order by plan_rank desc nulls last, sort_order asc nulls last
			limit 1
		`).Scan(&enterprisePlanID); err != nil || strings.TrimSpace(enterprisePlanID) == "" {
			writeJSONErr(w, http.StatusConflict, "ADMIN_ENTERPRISE_PLAN_MISSING")
			return
		}
		req.PlanID = strings.TrimSpace(enterprisePlanID)
		req.Quantity = *offeredSeats
	}
	if req.PlanID == "" {
		writeJSONErr(w, http.StatusBadRequest, "PLAN_ID_REQUIRED")
		return
	}
	if req.Interval != "monthly" && req.Interval != "annual" {
		writeJSONErr(w, http.StatusBadRequest, "PLAN_INTERVAL_INVALID")
		return
	}
	if req.Quantity < 1 {
		writeJSONErr(w, http.StatusBadRequest, "CHECKOUT_QUANTITY_INVALID")
		return
	}

	_, _ = h.Subs.ExpireIfNeeded(ctx, userID)

	decision, target, active, policy, err := h.Subs.DecideChange(ctx, userID, req.PlanID, req.Interval)
	if err != nil {
		if errors.Is(err, subscriptions.ErrPricingUnbound) {
			writeJSONErr(w, http.StatusServiceUnavailable, "BILLING_PRICING_UNBOUND")
			return
		}
		writeJSONErr(w, http.StatusBadRequest, "PLAN_DECIDE_FAILED")
		return
	}

	// Sales-offered Enterprise: allow Paddle checkout with fixed offered seat quantity.
	if req.InquiryID != "" && strings.EqualFold(strings.TrimSpace(target.PlanKind), "enterprise") {
		decision = subscriptions.Decision{
			Kind:       subscriptions.ChangeNewCheckout,
			CanProceed: true,
			Code:       "ENTERPRISE_OFFER_CHECKOUT",
			Reason:     subscriptions.MessageForCode("DECIDE_ENTERPRISE_OFFER_CHECKOUT"),
			TargetTier: target.ID,
			TargetRank: target.PlanRank,
		}
		decision.ActionLabel = subscriptions.ActionLabelForCode(decision.Code)
	}

	if !decision.CanProceed {
		status := http.StatusConflict
		if decision.Kind == subscriptions.ChangeBlocked {
			status = http.StatusForbidden
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		blocked := CheckoutResponse{
			Action:       string(decision.Kind),
			Code:         decision.Code,
			Message:      decision.Reason,
			PlanID:       req.PlanID,
			Interval:     req.Interval,
			Decision:     decision,
			ActionLabel:  subscriptions.ActionLabelForCode(decision.Code),
			PendingLabel: subscriptions.PendingLabelForCode(decision.Code),
		}
		fillCheckoutLabels(&blocked, target)
		_ = json.NewEncoder(w).Encode(blocked)
		return
	}

	priceID, displayCents, err := subscriptions.ResolvePriceID(target, req.Interval)
	if err != nil {
		writeJSONErr(w, http.StatusConflict, "PRICE_NOT_CONFIGURED")
		return
	}

	// Fail closed when Paddle catalog amount drifted from admin cents (e.g. edited in Paddle UI).
	if h.Paddle != nil && displayCents > 0 && strings.TrimSpace(priceID) != "" {
		if derr := assertPaddlePriceMatchesAdmin(ctx, h.Paddle, priceID, displayCents, target.CurrencyCode); derr != nil {
			log.Printf("checkout price drift plan=%s interval=%s price=%s: %v", req.PlanID, req.Interval, priceID, derr)
			writeJSONErr(w, http.StatusConflict, "PRICE_CATALOG_DRIFT")
			return
		}
	}

	if !target.PerSeat && req.Quantity != 1 && target.PlanKind != "topup" && req.InquiryID == "" {
		req.Quantity = 1
	}
	if target.PerSeat {
		settings, settingsErr := h.loadSettings(ctx)
		if settingsErr != nil {
			if errors.Is(settingsErr, subscriptions.ErrPricingUnbound) {
				writeJSONErr(w, http.StatusServiceUnavailable, "BILLING_PRICING_UNBOUND")
				return
			}
			writeJSONErr(w, http.StatusServiceUnavailable, "BILLING_SETTINGS_UNAVAILABLE")
			return
		}
		if req.Quantity < settings.MinSeatQuantity || req.Quantity > settings.MaxSeatQuantity {
			writeJSONErr(w, http.StatusBadRequest, "CHECKOUT_QUANTITY_INVALID")
			return
		}
	}

	// Self-serve annual checkout charges the synced yearly pri_* only (admin price_yearly_cents).
	// Do not attach Paddle percentage discounts - that would double-discount vs baked yearly amounts.
	var discountID, discountCode *string

	custom := map[string]interface{}{
		"plan_id":  req.PlanID,
		"interval": req.Interval,
		"userId":   userID,
	}
	if req.InquiryID != "" {
		custom["inquiry_id"] = req.InquiryID
	}

	pooledCredits := target.CreditsMonthly
	if target.PerSeat && req.Quantity > 1 {
		pooledCredits = target.CreditsMonthly * req.Quantity
	}

	previewOnly := r.URL.Path != "" && strings.HasSuffix(r.URL.Path, "/checkout-preview")
	if previewOnly {
		req.Confirm = false
	}

	// Active paid unexpired subscription: upgrade in place via Paddle API (proration credits remaining time).
	if decision.Kind == subscriptions.ChangeUpgrade && active != nil {
		if h.Paddle == nil {
			writeJSONErr(w, http.StatusInternalServerError, "PADDLE_CLIENT_MISSING")
			return
		}
		if strings.TrimSpace(active.PaddleSubscriptionID) == "" {
			writeJSONErr(w, http.StatusConflict, "PADDLE_SUBSCRIPTION_ID_MISSING")
			return
		}

		updateBody := paddleapi.UpdateSubscriptionRequest{
			Items: []paddleapi.Item{
				{PriceID: priceID, Quantity: req.Quantity},
			},
			ProrationBillingMode: policy.UpgradeProrationMode,
			OnPaymentFailure:     strings.TrimSpace(subscriptions.MessageForCode("PADDLE_ON_PAYMENT_FAILURE")),
			CustomData:           custom,
		}
		if updateBody.OnPaymentFailure == "" {
			writeJSONErr(w, http.StatusInternalServerError, "PADDLE_ON_PAYMENT_FAILURE_MISSING")
			return
		}

		preview, err := h.Paddle.PreviewUpdate(ctx, active.PaddleSubscriptionID, updateBody)
		if err != nil {
			writeJSONErr(w, http.StatusBadGateway, "PADDLE_UPGRADE_PREVIEW_FAILED")
			return
		}

		var proration *ProrationPreview
		if preview != nil && preview.Data.ImmediateTransaction != nil {
			t := preview.Data.ImmediateTransaction.Details.Totals
			proration = &ProrationPreview{
				CurrencyCode: t.CurrencyCode,
				Subtotal:     t.Subtotal,
				Tax:          t.Tax,
				Credit:       t.Credit,
				Balance:      t.Balance,
				GrandTotal:   t.GrandTotal,
			}
		}

		// Preview-only: return totals so the client can confirm before charging.
		if !req.Confirm {
			w.Header().Set("Content-Type", "application/json")
			previewResp := CheckoutResponse{
				Action:               "upgrade_preview",
				Code:                 "CONFIRM_REQUIRED",
				Message:              subscriptions.MessageForCode("UPGRADE_CONFIRM_REQUIRED_MSG"),
				PlanID:               req.PlanID,
				Interval:             req.Interval,
				Quantity:             req.Quantity,
				PriceID:              priceID,
				CurrencyCode:         target.CurrencyCode,
				DisplayCents:         displayCents,
				ProrationMode:        policy.UpgradeProrationMode,
				PaddleSubscriptionID: active.PaddleSubscriptionID,
				ProrationPreview:     proration,
				CustomData:           custom,
				Decision:             decision,
				ActionLabel:          subscriptions.ActionLabelForCode("CONFIRM_REQUIRED"),
				PendingLabel:         subscriptions.PendingLabelForCode("CONFIRM_REQUIRED"),
			}
			fillCheckoutLabels(&previewResp, target)
			_ = json.NewEncoder(w).Encode(previewResp)
			return
		}

		updated, err := h.Paddle.UpdateSubscription(ctx, active.PaddleSubscriptionID, updateBody)
		if err != nil {
			writeJSONErr(w, http.StatusBadGateway, "PADDLE_UPGRADE_FAILED")
			return
		}

		// Optimistic local sync; webhook remains source of truth.
		_, _ = h.DB.Exec(ctx, `
			update public.subscriptions
			set plan_tier = $2, price_id = $3, billing_interval = $4, status = $5,
			    expires_at = coalesce(nullif($6, '')::timestamptz, expires_at),
			    current_period_end = coalesce(nullif($6, '')::timestamptz, current_period_end),
			    updated_at = now()
			where paddle_subscription_id = $1
		`, active.PaddleSubscriptionID, req.PlanID, priceID, req.Interval, updated.Data.Status,
			updated.Data.CurrentBillingPeriod.EndsAt)

		_, _ = h.DB.Exec(ctx, `
			update public.user_quotas
			set plan_tier = $2, monthly_credit_limit = $3, updated_at = now()
			where user_id = $1
		`, userID, req.PlanID, pooledCredits)

		// Seat / enterprise: sync owned workspace pools immediately (webhook also syncs).
		if target.PlanKind == "enterprise" || target.PerSeat {
			_, _ = h.DB.Exec(ctx, `
				update public.subscriptions
				set seat_quantity = $2, updated_at = now()
				where paddle_subscription_id = $1
			`, active.PaddleSubscriptionID, req.Quantity)
			_, _ = h.DB.Exec(ctx, `
				update public.workspaces
				set plan_tier = $2, allocated_seats = $3, updated_at = now()
				where owner_id = $1::uuid
			`, userID, req.PlanID, req.Quantity)
			_, _ = h.DB.Exec(ctx, `
				update public.workspace_quotas wq
				set monthly_shared_credits = $2, updated_at = now()
				from public.workspaces w
				where w.id = wq.workspace_id and w.owner_id = $1::uuid
			`, userID, pooledCredits)
			if h.Redis != nil {
				rows, qErr := h.DB.Query(ctx, `
					select id::text from public.workspaces where owner_id = $1::uuid
				`, userID)
				if qErr == nil {
					for rows.Next() {
						var wid string
						if rows.Scan(&wid) == nil && wid != "" {
							_ = h.Redis.Del(ctx, "workspace_quota:"+wid).Err()
						}
					}
					rows.Close()
				}
			}
		}

		if h.Redis != nil {
			_ = h.Redis.Del(ctx, "user_quota:"+userID).Err()
		}

		w.Header().Set("Content-Type", "application/json")
		upgradeResp := CheckoutResponse{
			Action:               "upgrade",
			Code:                 decision.Code,
			Message:              subscriptions.MessageForCode("UPGRADE_APPLIED_MSG"),
			PlanID:               req.PlanID,
			Interval:             req.Interval,
			Quantity:             req.Quantity,
			PriceID:              priceID,
			CurrencyCode:         target.CurrencyCode,
			DisplayCents:         displayCents,
			ProrationMode:        policy.UpgradeProrationMode,
			PaddleSubscriptionID: active.PaddleSubscriptionID,
			ProrationPreview:     proration,
			CustomData:           custom,
			Decision:             decision,
			ActionLabel:          subscriptions.ActionLabelForCode(decision.Code),
			PendingLabel:         subscriptions.PendingLabelForCode(decision.Code),
		}
		fillCheckoutLabels(&upgradeResp, target)
		_ = json.NewEncoder(w).Encode(upgradeResp)
		return
	}

	// New checkout (free / expired / top-up): client opens Paddle.js overlay.
	resp := CheckoutResponse{
		Action:       "new_checkout",
		Code:         decision.Code,
		Message:      decision.Reason,
		PlanID:       req.PlanID,
		Interval:     req.Interval,
		Quantity:     req.Quantity,
		PriceID:      priceID,
		DiscountID:   discountID,
		DiscountCode: discountCode,
		CustomData:   custom,
		CurrencyCode: target.CurrencyCode,
		DisplayCents: displayCents,
		Decision:     decision,
		ActionLabel:  subscriptions.ActionLabelForCode(decision.Code),
		PendingLabel: subscriptions.PendingLabelForCode(decision.Code),
	}
	fillCheckoutLabels(&resp, target)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// PreviewCheckoutSession returns upgrade proration totals without charging.
// Same request body as CreateCheckoutSession; upgrades never apply until confirm=true on checkout-session.
func (h *Handler) PreviewCheckoutSession(w http.ResponseWriter, r *http.Request) {
	h.CreateCheckoutSession(w, r)
}

// CreatePortalSession returns a short-lived authenticated Paddle customer portal URL.
func (h *Handler) CreatePortalSession(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		writeJSONErr(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}
	if h.Paddle == nil {
		writeJSONErr(w, http.StatusServiceUnavailable, "PADDLE_API_MISSING")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}

	ctx := r.Context()
	_, _ = h.Subs.ExpireIfNeeded(ctx, userID)

	active, err := h.Subs.GetActiveSubscription(ctx, userID)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "PLAN_OPERATION_FAILED")
		return
	}
	if active == nil || active.PaddleCustomerID == "" {
		// Fall back to any historical paddle customer so expired users can still manage invoices.
		var customerID string
		_ = h.DB.QueryRow(ctx, `
			select paddle_customer_id from public.subscriptions
			where user_id = $1 and coalesce(paddle_customer_id, '') <> ''
			order by updated_at desc
			limit 1
		`, userID).Scan(&customerID)
		if customerID == "" {
			writeJSONErr(w, http.StatusBadRequest, "PADDLE_CUSTOMER_MISSING")
			return
		}
		session, err := h.Paddle.CreatePortalSession(ctx, customerID, nil)
		if err != nil {
			writeJSONErr(w, http.StatusBadGateway, "PADDLE_UPSTREAM_FAILED")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"overview_url":  session.Data.URLs.General.Overview,
			"customer_id":   session.Data.CustomerID,
			"session_id":    session.Data.ID,
			"pending_label": subscriptions.PendingLabelForCode("PORTAL_OPEN"),
			"action_label":  subscriptions.ActionLabelForCode("PORTAL_OPEN"),
		})
		return
	}

	var subIDs []string
	if active.PaddleSubscriptionID != "" {
		subIDs = []string{active.PaddleSubscriptionID}
	}
	session, err := h.Paddle.CreatePortalSession(ctx, active.PaddleCustomerID, subIDs)
	if err != nil {
		writeJSONErr(w, http.StatusBadGateway, "PADDLE_UPSTREAM_FAILED")
		return
	}

	policy, err := h.Subs.LoadPolicy(ctx)
	if err != nil && !errors.Is(err, subscriptions.ErrPricingUnbound) {
		writeJSONErr(w, http.StatusInternalServerError, "PLAN_OPERATION_FAILED")
		return
	}
	if errors.Is(err, subscriptions.ErrPricingUnbound) {
		writeJSONErr(w, http.StatusServiceUnavailable, "BILLING_PRICING_UNBOUND")
		return
	}

	resp := map[string]interface{}{
		"overview_url":  session.Data.URLs.General.Overview,
		"customer_id":   session.Data.CustomerID,
		"session_id":    session.Data.ID,
		"pending_label": subscriptions.PendingLabelForCode("PORTAL_OPEN"),
		"action_label":  subscriptions.ActionLabelForCode("PORTAL_OPEN"),
	}
	if len(session.Data.URLs.Subscriptions) > 0 {
		s0 := session.Data.URLs.Subscriptions[0]
		resp["update_payment_url"] = s0.UpdateSubscriptionPaymentMethod
		// Cancel at period end is MoR portal flow; SKU downgrade stays blocked separately.
		if policy.AllowCancelAtPeriodEnd {
			resp["cancel_url"] = s0.CancelSubscription
		} else {
			resp["cancel_disabled_message"] = subscriptions.MessageForCode("PORTAL_CANCEL_DISABLED")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

type enterpriseInquiryRequest struct {
	CompanyName    string `json:"company_name"`
	EstimatedSeats *int   `json:"estimated_seats"`
	Message        string `json:"message"`
}

// CreateEnterpriseInquiry stores a sales lead in Postgres (source of truth).
// Self-serve checkout is not offered for enterprise; this replaces mailto-only UX.
func (h *Handler) CreateEnterpriseInquiry(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}

	var req enterpriseInquiryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	req.CompanyName = strings.TrimSpace(req.CompanyName)
	if req.Message == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": subscriptions.MessageForCode("ENTERPRISE_MESSAGE_REQUIRED"),
			"code":  "ENTERPRISE_MESSAGE_REQUIRED",
		})
		return
	}
	if len(req.Message) > 4000 {
		writeJSONErr(w, http.StatusBadRequest, "ENTERPRISE_MESSAGE_TOO_LONG")
		return
	}
	settings, err := h.loadSettings(r.Context())
	if err != nil {
		if errors.Is(err, subscriptions.ErrPricingUnbound) {
			writeJSONErr(w, http.StatusServiceUnavailable, "BILLING_PRICING_UNBOUND")
			return
		}
		writeJSONErr(w, http.StatusServiceUnavailable, "BILLING_SETTINGS_UNAVAILABLE")
		return
	}
	if req.EstimatedSeats != nil &&
		(*req.EstimatedSeats < settings.MinSeatQuantity || *req.EstimatedSeats > settings.MaxSeatQuantity) {
		writeJSONErr(w, http.StatusBadRequest, "ENTERPRISE_SEATS_RANGE")
		return
	}

	var email string
	err = h.DB.QueryRow(r.Context(), `
		select email from public.profiles where id = $1::uuid
	`, userID).Scan(&email)
	if err != nil || email == "" {
		writeJSONErr(w, http.StatusBadRequest, "PROFILE_EMAIL_NOT_FOUND")
		return
	}

	var id string
	err = h.DB.QueryRow(r.Context(), `
		insert into public.enterprise_inquiries
			(user_id, email, company_name, estimated_seats, message)
		values ($1::uuid, $2, nullif($3, ''), $4, $5)
		returning id::text
	`, userID, email, req.CompanyName, req.EstimatedSeats, req.Message).Scan(&id)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "PLAN_OPERATION_FAILED")
		return
	}

	go h.notifyEnterpriseInquiry(id, userID, email, req.CompanyName, req.EstimatedSeats, req.Message)

	company := strings.TrimSpace(req.CompanyName)
	if company == "" {
		company = email
	}
	_ = notifications.InsertForAdminsWithPermission(
		r.Context(),
		h.DB,
		"enterprise.read",
		notifications.KindAdminEnterpriseInquiry,
		[]string{email, company},
		"enterprise:"+id,
		"",
		id,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      id,
		"status":  "new",
		"message": subscriptions.MessageForCode("ENTERPRISE_INQUIRY_RECEIVED"),
		"code":    "ENTERPRISE_INQUIRY_RECEIVED",
	})
}

func meEnterpriseStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "new":
		return subscriptions.MessageForCode("ENTERPRISE_ME_STATUS_NEW")
	case "contacted":
		return subscriptions.MessageForCode("ENTERPRISE_ME_STATUS_CONTACTED")
	case "offered":
		return subscriptions.MessageForCode("ENTERPRISE_ME_STATUS_OFFERED")
	case "closed":
		return subscriptions.MessageForCode("ENTERPRISE_ME_STATUS_CLOSED")
	case "activated":
		return subscriptions.MessageForCode("ENTERPRISE_ME_STATUS_ACTIVATED")
	default:
		return ""
	}
}

// ListMeEnterpriseInquiries returns the signed-in user's sales inquiries (pay CTA when offered).
func (h *Handler) ListMeEnterpriseInquiries(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	db := h.readPool()
	maxLimit, err := billingsettings.MaxPageSize(r.Context(), db)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	skipCap, err := billingsettings.SkipToMaxPages(r.Context(), db)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	params, err := pagination.Parse(r, maxLimit)
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, err.Error())
		return
	}
	statusFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	where := `user_id = $1::uuid`
	args := []any{userID}
	argN := 2
	if statusFilter != "" {
		switch statusFilter {
		case "new", "contacted", "offered", "closed", "activated":
			where += fmt.Sprintf(` and status = $%d`, argN)
			args = append(args, statusFilter)
			argN++
		default:
			writeJSONErr(w, http.StatusBadRequest, "ADMIN_STATUS_INVALID")
			return
		}
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 200 {
		q = q[:200]
	}
	if q != "" {
		like := "%" + escapeILikePattern(q) + "%"
		where += fmt.Sprintf(` and (
			coalesce(company_name, '') ilike $%d escape '\'
			or coalesce(message, '') ilike $%d escape '\'
		)`, argN, argN)
		args = append(args, like)
		argN++
	}
	var total int
	if err := db.QueryRow(r.Context(), `select count(*) from public.enterprise_inquiries where `+where, args...).Scan(&total); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := fmt.Sprintf(`
		select id::text, coalesce(company_name, ''), coalesce(estimated_seats, 0),
		       message, status, offered_seat_quantity,
		       created_at::text, updated_at::text
		from public.enterprise_inquiries
		where %s
		order by created_at desc
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, company, message, status, created, updated string
		var estimated int
		var offered *int
		if err := rows.Scan(&id, &company, &estimated, &message, &status, &offered, &created, &updated); err != nil {
			writeJSONErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		row := map[string]any{
			"id":              id,
			"company_name":    company,
			"estimated_seats": estimated,
			"message":         message,
			"status":          status,
			"status_label":    meEnterpriseStatusLabel(status),
			"created_at":      created,
			"updated_at":      updated,
			"can_checkout":    strings.EqualFold(status, "offered") && offered != nil && *offered >= 1,
		}
		if offered != nil {
			row["offered_seat_quantity"] = *offered
		}
		items = append(items, row)
	}

	var enterprisePlanID string
	_ = db.QueryRow(r.Context(), `
		select id from public.plan_catalog
		where is_active = true and plan_kind = 'enterprise'
		order by plan_rank desc nulls last, sort_order asc nulls last
		limit 1
	`).Scan(&enterprisePlanID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":   items,
		"meta":    pagination.BuildMeta(params, total, skipCap),
		"plan_id": strings.TrimSpace(enterprisePlanID),
		"chrome": map[string]string{
			"title":                       subscriptions.MessageForCode("ENTERPRISE_ME_TITLE"),
			"empty":                       subscriptions.MessageForCode("ENTERPRISE_ME_EMPTY"),
			"col_company":                 subscriptions.MessageForCode("ENTERPRISE_ME_COL_COMPANY"),
			"col_status":                  subscriptions.MessageForCode("ENTERPRISE_ME_COL_STATUS"),
			"col_requested":               subscriptions.MessageForCode("ENTERPRISE_ME_COL_REQUESTED"),
			"col_offered":                 subscriptions.MessageForCode("ENTERPRISE_ME_COL_OFFERED"),
			"col_created":                 subscriptions.MessageForCode("ENTERPRISE_ME_COL_CREATED"),
			"col_updated":                 subscriptions.MessageForCode("ENTERPRISE_ME_COL_UPDATED"),
			"col_message":                 subscriptions.MessageForCode("ENTERPRISE_ME_COL_MESSAGE"),
			"pay_label":                   subscriptions.MessageForCode("ENTERPRISE_ME_PAY"),
			"pay_pending_label":           subscriptions.MessageForCode("ENTERPRISE_ME_PAY_PENDING"),
			"view_label":                  subscriptions.MessageForCode("ENTERPRISE_ME_VIEW"),
			"details_label":               subscriptions.MessageForCode("ENTERPRISE_ME_DETAILS"),
			"filter_status":               subscriptions.MessageForCode("ENTERPRISE_ME_FILTER_STATUS"),
			"filter_status_desc":          subscriptions.MessageForCode("ENTERPRISE_ME_FILTER_STATUS_DESC"),
			"filter_all":                  subscriptions.MessageForCode("ADMIN_FILTER_ALL"),
			"filter_search":               subscriptions.MessageForCode("ADMIN_FILTER_SEARCH"),
			"search_placeholder":          subscriptions.MessageForCode("ENTERPRISE_ME_SEARCH"),
			"search_description":          subscriptions.MessageForCode("ENTERPRISE_ME_SEARCH_DESC"),
			"status_new":                  subscriptions.MessageForCode("ENTERPRISE_ME_STATUS_NEW"),
			"status_contacted":            subscriptions.MessageForCode("ENTERPRISE_ME_STATUS_CONTACTED"),
			"status_offered":              subscriptions.MessageForCode("ENTERPRISE_ME_STATUS_OFFERED"),
			"status_closed":               subscriptions.MessageForCode("ENTERPRISE_ME_STATUS_CLOSED"),
			"status_activated":            subscriptions.MessageForCode("ENTERPRISE_ME_STATUS_ACTIVATED"),
			"checkout_interval_required":  subscriptions.MessageForCode("ENTERPRISE_CHECKOUT_INTERVAL_REQUIRED"),
			"table_select_all":            subscriptions.MessageForCode("TABLE_SELECT_ALL"),
			"table_select_row":            subscriptions.MessageForCode("TABLE_SELECT_ROW"),
			"table_selected_fmt":          subscriptions.MessageForCode("TABLE_SELECTED_FMT"),
			"table_row_actions":           subscriptions.MessageForCode("TABLE_ROW_ACTIONS"),
			"table_bulk_delete":           subscriptions.MessageForCode("TABLE_BULK_DELETE"),
			"table_clear_selection":       subscriptions.MessageForCode("TABLE_CLEAR_SELECTION"),
			"delete_action_label":         subscriptions.MessageForCode("ENTERPRISE_ME_DELETE"),
			"delete_pending_label":        subscriptions.MessageForCode("ENTERPRISE_ME_DELETE_PENDING"),
			"delete_confirm_message":      subscriptions.MessageForCode("ENTERPRISE_ME_DELETE_CONFIRM"),
			"bulk_delete_confirm_message": subscriptions.MessageForCode("ENTERPRISE_ME_BULK_DELETE_CONFIRM"),
		},
		"q": q,
	})
}

// DeleteMeEnterpriseInquiries deletes the caller's non-activated inquiries (DataTable selection).
func (h *Handler) DeleteMeEnterpriseInquiries(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	if h.DB == nil {
		writeJSONErr(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	ids := make([]string, 0, len(req.IDs))
	seen := map[string]struct{}{}
	for _, raw := range req.IDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		writeJSONErr(w, http.StatusBadRequest, "ENTERPRISE_ME_IDS_REQUIRED")
		return
	}
	maxLimit, err := billingsettings.MaxPageSize(r.Context(), h.DB)
	if err != nil || maxLimit < 1 {
		writeJSONErr(w, http.StatusInternalServerError, "BILLING_SETTINGS_UNAVAILABLE")
		return
	}
	if len(ids) > maxLimit {
		ids = ids[:maxLimit]
	}
	tag, err := h.DB.Exec(r.Context(), `
		delete from public.enterprise_inquiries
		where user_id = $1::uuid
		  and id = any($2::uuid[])
		  and status <> 'activated'
	`, userID, ids)
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, "ENTERPRISE_ME_IDS_INVALID")
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSONErr(w, http.StatusConflict, "ENTERPRISE_ME_DELETE_ACTIVATED")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":        "ok",
		"deleted":       tag.RowsAffected(),
		"action_label":  subscriptions.MessageForCode("ENTERPRISE_ME_DELETE"),
		"pending_label": subscriptions.MessageForCode("ENTERPRISE_ME_DELETE_PENDING"),
		"message":       subscriptions.MessageForCode("ENTERPRISE_ME_BULK_DELETED"),
	})
}

func escapeILikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func (h *Handler) notifyEnterpriseInquiry(
	id, userID, email, company string,
	seats *int,
	message string,
) {
	url := h.EnterpriseNotifyURL
	if url == "" {
		return
	}
	payload := map[string]interface{}{
		"event":      "enterprise_inquiry.created",
		"id":         id,
		"user_id":    userID,
		"email":      email,
		"company":    company,
		"message":    message,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}
	if seats != nil {
		payload["estimated_seats"] = *seats
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if h.EnterpriseNotifyTimeoutSec < 1 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.EnterpriseNotifyTimeoutSec)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("enterprise notify: build request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TrimAPI/enterprise-inquiry")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("enterprise notify: post failed: %v", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		log.Printf("enterprise notify: status %d for inquiry %s", res.StatusCode, id)
	}
}

func (h *Handler) loadSettings(ctx context.Context) (Settings, error) {
	var s Settings
	var discID, discCode *string
	err := h.readPool().QueryRow(ctx, `
		select annual_discount_percent, apply_paddle_discount_on_annual,
		       paddle_discount_id, paddle_discount_code, default_currency,
		       coalesce(pricing_bound, false),
		       allow_downgrades, coalesce(allow_cancel_at_period_end, true),
		       upgrade_proration_mode,
		       coalesce(default_page_size, 0), coalesce(max_seat_quantity, 0),
		       coalesce(min_seat_quantity, 0),
		       coalesce(default_seat_quantity, 0), coalesce(default_checkout_quantity, 0),
		       coalesce(enterprise_message_rows, 0), coalesce(default_plan_interval, ''),
		       coalesce(max_page_size, 0),
		       coalesce(date_range_months, 0),
		       coalesce(deep_target_token_min, 0), coalesce(deep_target_token_max, 0)
		from public.billing_settings
		where id = 'default'
	`).Scan(
		&s.AnnualDiscountPercent,
		&s.ApplyPaddleDiscountOnAnnual,
		&discID,
		&discCode,
		&s.DefaultCurrency,
		&s.PricingBound,
		&s.AllowDowngrades,
		&s.AllowCancelAtPeriodEnd,
		&s.UpgradeProrationMode,
		&s.DefaultPageSize,
		&s.MaxSeatQuantity,
		&s.MinSeatQuantity,
		&s.DefaultSeatQuantity,
		&s.DefaultCheckoutQuantity,
		&s.EnterpriseMessageRows,
		&s.DefaultPlanInterval,
		&s.MaxPageSize,
		&s.DateRangeMonths,
		&s.DeepTargetTokenMin,
		&s.DeepTargetTokenMax,
	)
	if err != nil {
		return Settings{}, fmt.Errorf("billing_settings not found: %w", err)
	}
	// Match LoadPolicy: checkout never enables downgrades. Coerce so plan-modal
	// upgrade-only chrome stays visible even if billing_settings.allow_downgrades is true.
	if s.AllowDowngrades {
		s.AllowDowngrades = false
	}
	if !s.PricingBound {
		return Settings{}, subscriptions.ErrPricingUnbound
	}
	if strings.EqualFold(strings.TrimSpace(s.DefaultCurrency), "XXX") {
		return Settings{}, fmt.Errorf("billing_settings.default_currency invalid")
	}
	if s.DefaultPlanInterval != "monthly" && s.DefaultPlanInterval != "annual" {
		return Settings{}, fmt.Errorf("billing_settings.default_plan_interval invalid")
	}
	if s.DefaultSeatQuantity < 1 {
		return Settings{}, fmt.Errorf("billing_settings.default_seat_quantity invalid")
	}
	if s.MinSeatQuantity < 1 {
		return Settings{}, fmt.Errorf("billing_settings.min_seat_quantity invalid")
	}
	if s.MaxSeatQuantity < s.MinSeatQuantity {
		return Settings{}, fmt.Errorf("billing_settings.max_seat_quantity invalid")
	}
	if s.DefaultCheckoutQuantity < 1 {
		return Settings{}, fmt.Errorf("billing_settings.default_checkout_quantity invalid")
	}
	if s.EnterpriseMessageRows < 2 {
		return Settings{}, fmt.Errorf("billing_settings.enterprise_message_rows invalid")
	}
	if s.MaxPageSize < 1 {
		return Settings{}, fmt.Errorf("billing_settings.max_page_size invalid")
	}
	if s.DateRangeMonths < 1 {
		return Settings{}, fmt.Errorf("billing_settings.date_range_months invalid")
	}
	if s.DeepTargetTokenMin < 1 || s.DeepTargetTokenMax < s.DeepTargetTokenMin {
		return Settings{}, fmt.Errorf("billing_settings.deep_target_token bounds invalid")
	}
	if s.DefaultPageSize < 1 {
		return Settings{}, fmt.Errorf("billing_settings.default_page_size invalid")
	}
	if len(strings.TrimSpace(s.DefaultCurrency)) != 3 {
		return Settings{}, fmt.Errorf("billing_settings.default_currency invalid")
	}
	if s.AnnualDiscountPercent < 0 || s.AnnualDiscountPercent > 90 {
		return Settings{}, fmt.Errorf("billing_settings.annual_discount_percent invalid")
	}
	s.PaddleDiscountID = discID
	s.PaddleDiscountCode = discCode
	s.AuthRequiredPlanMessage = subscriptions.MessageForCode("AUTH_REQUIRED_PLAN")
	s.AuthRequiredEnterpriseMessage = subscriptions.MessageForCode("AUTH_REQUIRED_ENTERPRISE")
	s.EnterpriseMessageRequired = subscriptions.MessageForCode("ENTERPRISE_MESSAGE_REQUIRED")
	s.PaddleJSNotReadyMessage = subscriptions.MessageForCode("PADDLE_JS_NOT_READY")
	s.CheckoutPriceMissingMessage = subscriptions.MessageForCode("CHECKOUT_PRICE_MISSING")
	s.UpgradeMissingAPIMessage = subscriptions.MessageForCode("UPGRADE_MISSING_API_MESSAGE")
	s.InquiryMissingConfirmMessage = subscriptions.MessageForCode("INQUIRY_MISSING_CONFIRM")
	s.ChangeBlockedDefaultMessage = subscriptions.MessageForCode("CHANGE_BLOCKED_DEFAULT")
	s.CheckoutUnavailableMessage = subscriptions.MessageForCode("CHECKOUT_UNAVAILABLE_DEFAULT")
	s.UpgradeNotAppliedMessage = subscriptions.MessageForCode("UPGRADE_NOT_APPLIED")
	s.CheckoutRequestFailedMessage = subscriptions.MessageForCode("CHECKOUT_REQUEST_FAILED")
	s.UpgradeRequestFailedMessage = subscriptions.MessageForCode("UPGRADE_REQUEST_FAILED")
	s.EnterpriseInquiryFailedMessage = subscriptions.MessageForCode("ENTERPRISE_INQUIRY_FAILED")
	s.PlansDialogDescription = subscriptions.MessageForCode("PLANS_DIALOG_DESCRIPTION")
	s.PlansDialogUpgradesOnlyNote = subscriptions.MessageForCode("PLANS_DIALOG_UPGRADES_ONLY_NOTE")
	s.TopupDialogTitle = subscriptions.MessageForCode("TOPUP_DIALOG_TITLE")
	s.TopupDialogDescription = subscriptions.MessageForCode("TOPUP_DIALOG_DESCRIPTION")
	s.TopupDialogEmpty = subscriptions.MessageForCode("TOPUP_DIALOG_EMPTY")
	s.AnnualToggleLabel = subscriptions.MessageForCode("PLANS_ANNUAL_TOGGLE")
	s.MonthlyToggleLabel = subscriptions.MessageForCode("PLANS_MONTHLY_TOGGLE")
	s.EnterpriseDialogTitle = subscriptions.MessageForCode("ENTERPRISE_DIALOG_TITLE")
	s.EnterpriseDialogDescription = subscriptions.MessageForCode("ENTERPRISE_DIALOG_DESCRIPTION")
	s.EnterpriseEmailFallback = subscriptions.MessageForCode("ENTERPRISE_EMAIL_FALLBACK")
	s.EnterpriseCompanyLabel = subscriptions.MessageForCode("ENTERPRISE_COMPANY_LABEL")
	s.EnterpriseSeatsLabel = subscriptions.MessageForCode("ENTERPRISE_SEATS_LABEL")
	s.EnterpriseMessageLabel = subscriptions.MessageForCode("ENTERPRISE_MESSAGE_LABEL")
	s.EnterpriseCompanyPlaceholder = subscriptions.MessageForCode("ENTERPRISE_COMPANY_PLACEHOLDER")
	s.EnterpriseSeatsPlaceholder = subscriptions.MessageForCode("ENTERPRISE_SEATS_PLACEHOLDER")
	s.EnterpriseMessagePlaceholder = subscriptions.MessageForCode("ENTERPRISE_MESSAGE_PLACEHOLDER")
	s.EnterpriseCompanyDescription = subscriptions.MessageForCode("ENTERPRISE_COMPANY_DESC")
	s.EnterpriseMessageDescription = subscriptions.MessageForCode("ENTERPRISE_MESSAGE_DESC")
	s.EnterpriseCancelLabel = subscriptions.MessageForCode("ENTERPRISE_CANCEL")
	s.EnterpriseSendLabel = subscriptions.MessageForCode("ENTERPRISE_SEND")
	s.EnterpriseSendPendingLabel = subscriptions.PendingLabelForCode("ENTERPRISE_SEND")
	s.PlansLoadFailedMessage = subscriptions.MessageForCode("PLANS_LOAD_FAILED")
	s.ProrationReviewHint = subscriptions.MessageForCode("PRORATION_REVIEW_HINT")
	s.SeatQuantityHint = subscriptions.MessageForCode("PLANS_SEAT_HINT")
	s.SeatQuantityLabel = subscriptions.MessageForCode("PLANS_SEAT_LABEL")
	s.PerSeatOneFmt = subscriptions.MessageForCode("PLANS_PER_SEAT_ONE_FMT")
	s.PerSeatManyFmt = subscriptions.MessageForCode("PLANS_PER_SEAT_MANY_FMT")
	s.MetaSep = subscriptions.MessageForCode("DASHBOARD_META_SEP")
	s.PeriodYearLabel = subscriptions.MessageForCode("PLANS_PERIOD_YEAR")
	s.PeriodMonthLabel = subscriptions.MessageForCode("PLANS_PERIOD_MONTH")
	s.UpgradeConfirmTitle = subscriptions.MessageForCode("UPGRADE_CONFIRM_TITLE")
	s.UpgradeProrationModeFmt = subscriptions.MessageForCode("UPGRADE_PRORATION_MODE_FMT")
	s.UpgradePlanPrefix = subscriptions.MessageForCode("UPGRADE_PLAN_PREFIX")
	s.PlansDialogTitle = subscriptions.MessageForCode("PLANS_DIALOG_TITLE")
	s.ActivePlanPrefix = subscriptions.MessageForCode("ACTIVE_PLAN_PREFIX")
	s.ActivePlanExpiresFmt = subscriptions.MessageForCode("ACTIVE_PLAN_EXPIRES_FMT")
	s.ProrationPrefix = subscriptions.MessageForCode("PRORATION_PREFIX")
	s.UpgradeSeatsFmt = subscriptions.MessageForCode("UPGRADE_SEATS_FMT")
	s.UpgradeCreditPrefix = subscriptions.MessageForCode("UPGRADE_CREDIT_PREFIX")
	s.UpgradeSubtotalPrefix = subscriptions.MessageForCode("UPGRADE_SUBTOTAL_PREFIX")
	s.UpgradeTaxPrefix = subscriptions.MessageForCode("UPGRADE_TAX_PREFIX")
	s.UpgradeDuePrefix = subscriptions.MessageForCode("UPGRADE_DUE_PREFIX")
	s.UpgradePreviewEmpty = subscriptions.MessageForCode("UPGRADE_PREVIEW_EMPTY")
	s.UpgradeCancelLabel = subscriptions.MessageForCode("UPGRADE_CANCEL")
	s.DialogCloseLabel = subscriptions.MessageForCode("DIALOG_CLOSE")
	s.UnlimitedFeatureLabel = subscriptions.MessageForCode("PLAN_FEATURE_UNLIMITED_METERING")
	return s, nil
}

// ensureUnlimitedFeature prepends PLAN_FEATURE_UNLIMITED_METERING from site_messages when missing.
func ensureUnlimitedFeature(raw json.RawMessage) json.RawMessage {
	label := strings.TrimSpace(subscriptions.MessageForCode("PLAN_FEATURE_UNLIMITED_METERING"))
	if label == "" {
		return raw
	}
	var features []string
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &features)
	}
	for _, f := range features {
		if strings.EqualFold(strings.TrimSpace(f), label) {
			return raw
		}
	}
	out := append([]string{label}, features...)
	b, err := json.Marshal(out)
	if err != nil {
		return raw
	}
	return b
}

// assertPaddlePriceMatchesAdmin ensures the live Paddle price amount/currency matches admin cents.
// Fail-closed: GetPrice errors, archived prices, unparseable amounts, and mismatches all block checkout
// so Paddle cannot charge a price the admin catalog no longer owns / cannot verify.
func assertPaddlePriceMatchesAdmin(ctx context.Context, client *paddleapi.Client, priceID string, adminCents int, currency string) error {
	if client == nil || adminCents <= 0 || strings.TrimSpace(priceID) == "" {
		return nil
	}
	got, err := client.GetPrice(ctx, priceID)
	if err != nil {
		return fmt.Errorf("paddle get price %s: %w", priceID, err)
	}
	if strings.EqualFold(strings.TrimSpace(got.Data.Status), "archived") {
		return fmt.Errorf("paddle price %s is archived", priceID)
	}
	amt, err := strconv.Atoi(strings.TrimSpace(got.Data.UnitPrice.Amount))
	if err != nil {
		return fmt.Errorf("paddle price amount unparseable: %q", got.Data.UnitPrice.Amount)
	}
	if amt != adminCents {
		return fmt.Errorf("paddle amount %d != admin cents %d", amt, adminCents)
	}
	cur := strings.ToUpper(strings.TrimSpace(currency))
	if cur != "" && !strings.EqualFold(strings.TrimSpace(got.Data.UnitPrice.CurrencyCode), cur) {
		return fmt.Errorf("paddle currency %s != admin %s", got.Data.UnitPrice.CurrencyCode, cur)
	}
	return nil
}
