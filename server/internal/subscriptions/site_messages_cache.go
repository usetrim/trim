package subscriptions

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	_ "embed"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

//go:embed chrome_seed_codes.txt
var chromeSeedCodesRaw string

var (
	siteMsgMu     sync.RWMutex
	siteMsgCache  = map[string]string{}
	siteMsgWarmed bool
)

// SeedAndRefreshSiteMessages upserts missing chrome rows from builtins, then
// loads public.site_messages into the process cache. Runtime MessageForCode /
// ActionLabelForCode / PendingLabelForCode read this cache only (fail-closed
// empty on miss). Builtins are never returned at request time.
// Existing operator-edited rows are never overwritten (on conflict do nothing).
//
// Boot path is batched: one SELECT of existing codes, then chunked unnest INSERT
// for missing rows only (not one round-trip per chrome code).
func SeedAndRefreshSiteMessages(ctx context.Context, db *pgxpool.Pool) error {
	if db == nil {
		return fmt.Errorf("database required")
	}
	desired := make(map[string]string, 2048)
	put := func(code, body string) {
		code = strings.TrimSpace(code)
		if code == "" {
			return
		}
		if _, ok := desired[code]; ok {
			return
		}
		desired[code] = body
	}

	for _, code := range parseSeedCodes(chromeSeedCodesRaw) {
		body := builtinMessageForCode(code)
		if body == "" {
			body = builtinLocalChromeMessage(code)
		}
		if body == "" {
			continue
		}
		put(code, body)
	}

	actionCodes := []string{
		"FREE_INCLUDED", "ENTERPRISE_CONTACT", "PRICE_NOT_CONFIGURED", "TOPUP_CHECKOUT",
		"NEW_CHECKOUT", "UPGRADE_NEW_CHECKOUT", "INTERVAL_UPGRADE_NEW_CHECKOUT",
		"INTERVAL_UPGRADE", "UPGRADE_ALLOWED", "SAME_PLAN", "DOWNGRADE_BLOCKED",
		"DOWNGRADE_UNSUPPORTED", "NOT_SELF_SERVE", "INTERVAL_CHANGE_DISABLED",
		"CHANGE_NOT_ALLOWED", "DASHBOARD_CHANGE_PLAN", "DASHBOARD_UPGRADE",
		"PORTAL_OPEN", "CONFIRM_REQUIRED", "RECEIPTS_SYNC", "AVATAR_SYNC",
		"ACCOUNT_DELETE", "SIGN_OUT", "WORKSPACE_CREATE", "WORKSPACE_INVITE",
		"WORKSPACE_INVITE_ACCEPT", "WORKSPACE_INVITE_REVOKE", "WORKSPACE_INVITE_COPY",
		"WORKSPACE_INVITE_COPIED", "WORKSPACE_INVITE_SIGN_IN", "WORKSPACE_REMOVE",
		"WORKSPACE_LEAVE", "WORKSPACE_RENAME", "WORKSPACE_RENAME_SAVE", "WORKSPACE_DELETE",
		"WORKSPACE_ROLE_CHANGE", "API_KEY_REVOKE", "API_KEY_LIST_REFRESH", "PREFERENCES_SAVE",
		"API_KEY_ISSUE", "API_KEY_ISSUE_RETRY", "API_KEY_ISSUE_ANOTHER", "API_KEY_COPY",
		"API_KEY_COPIED", "API_KEY_DEVICE_REGISTER", "API_KEY_DEVICE_REMOVE",
		"RECEIPT_PRINT", "RECEIPT_DOWNLOAD_PDF", "AUTH_OAUTH_GOOGLE", "AUTH_OAUTH_GITHUB",
		"AUTH_OAUTH_GITLAB", "AUTH_LINK_GOOGLE", "AUTH_LINK_GITHUB", "AUTH_LINK_GITLAB",
		"AUTH_UNLINK", "PAGINATION_SKIP_TO", "PAGINATION_FIRST", "PAGINATION_PREV",
		"PAGINATION_NEXT", "PAGINATION_LAST", "ENTERPRISE_SEND",
	}
	for _, code := range actionCodes {
		if body := builtinActionLabelForCode(code); body != "" {
			put("ACTION:"+code, body)
		}
		if body := builtinPendingLabelForCode(code); body != "" {
			put("PENDING:"+code, body)
		}
	}

	// Extra format strings + middleware/fraud chrome (seeded even if not yet in chrome_seed_codes.txt).
	extras := map[string]string{
		"BILLING_SETTINGS_UNAVAILABLE":                      "Billing settings are unavailable",
		"BILLING_PRICING_UNBOUND":                           "Billing pricing is not bound yet. An operator must set plan amounts and currency in the admin dashboard and sync the catalog to Paddle.",
		"PRICE_NOT_CONFIGURED_REASON":                       "This plan is not synced to Paddle yet. An operator must sync the plan catalog from the admin dashboard.",
		"RECEIPT_PERIOD_JOIN_FMT":                           "%s - %s",
		"RECEIPT_TAX_RATE_FMT":                              " (%0.2f%%)",
		"AUTH_TOKEN_MISSING":                                "Missing or malformed authorization token.",
		"AUTH_ACCOUNT_DISABLED":                             "This account is disabled by the platform operator.",
		"ADMIN_FORBIDDEN":                                   "Platform admin access required.",
		"ADMIN_ROUTE_FORBIDDEN":                             "You do not have permission to open this admin page.",
		"ADMIN_FORBIDDEN_PAGE":                              "This account is not a platform admin on this instance.",
		"ADMIN_BOOTSTRAP_REQUIRED":                          "Platform admin bootstrap is required before sign-in can complete.",
		"ADMIN_ALERT_JA4_CAP_BREACH":                        "JA4 fingerprints over account cap",
		"ADMIN_ALERT_HARDWARE_CAP_BREACH":                   "Hardware fingerprints over account cap",
		"ADMIN_PERMISSION_DENIED":                           "Missing permission for this admin action.",
		"ADMIN_STEP_UP_REQUIRED":                            "",
		"ADMIN_STEP_UP_TTL_MISSING":                         "Set TRIM_ADMIN_STEP_UP_TTL_SEC (60-43200) before step-up can run.",
		"ADMIN_ORIGIN_FORBIDDEN":                            "Admin origin is not allowed for this API.",
		"ADMIN_RANGE_REQUIRED":                              "Distribution range is required (day, month, or year).",
		"ADMIN_RANGE_INVALID":                               "Distribution range must be day, month, or year.",
		"ADMIN_WEBHOOK_NOT_FOUND":                           "Webhook event not found.",
		"ADMIN_EMAIL_TEMPLATE_NOT_FOUND":                    "Email template code is not in the catalog.",
		"ADMIN_PADDLE_UNAVAILABLE":                          "Paddle API is not configured for this deployment.",
		"ADMIN_PADDLE_CATALOG_SYNC_FAILED":                  "Could not sync plan catalog to Paddle. Check API key permissions (product.write, price.write) and currency.",
		"ADMIN_PADDLE_CURRENCY_INVALID":                     "Set a real ISO-4217 default currency in Billing settings before syncing to Paddle.",
		"ADMIN_PLAN_SYNC_PADDLE":                            "Sync to Paddle",
		"ADMIN_PLAN_SYNC_PENDING":                           "Syncing…",
		"ADMIN_PLAN_PADDLE_IDS":                             "Paddle IDs (synced)",
		"ADMIN_PLAN_PRODUCT_ID":                             "Product id",
		"ADMIN_PLAN_AMOUNT_MONTHLY":                         "Monthly amount",
		"ADMIN_PLAN_AMOUNT_YEARLY":                          "Yearly amount",
		"ADMIN_PLAN_CREDITS":                                "Credits / month",
		"ADMIN_PLAN_CREDITS_DESC":                           "Monthly cloud credits granted by this plan (or pack size for top-up plans).",
		"ADMIN_PLAN_PER_SEAT":                               "Per-seat pricing",
		"ADMIN_PLAN_PER_SEAT_DESC":                          "When on, checkout quantity multiplies credits and allocates workspace seats.",
		"ADMIN_PLAN_UNLIMITED":                              "Unlimited metering",
		"ADMIN_PLAN_UNLIMITED_DESC":                         "When on, members on this plan are not debited cloud credits. credits_monthly stays for display and signup init. Default off - use as a growth dial, not forever on every plan.",
		"ADMIN_PLAN_UNLIMITED_COL":                          "Unlimited",
		"ADMIN_PLAN_POPULAR":                                "Popular plan",
		"ADMIN_PLAN_POPULAR_DESC":                           "When on, the public pricing card shows the Popular badge for this plan. You can enable it on one, several, or all plans.",
		"ADMIN_PLAN_POPULAR_COL":                            "Popular",
		"PLAN_FEATURE_UNLIMITED_METERING":                   "Unlimited cloud metering",
		"DASHBOARD_QUOTA_UNLIMITED_LABEL":                   "Unlimited",
		"DASHBOARD_METRIC_TOPUP":                            "Top-up credits",
		"DASHBOARD_QUOTA_EXHAUSTED_TITLE":                   "Cloud credits exhausted",
		"DASHBOARD_QUOTA_EXHAUSTED_BODY":                    "Upgrade your plan or buy a top-up to keep using Trim cloud. Local Fast Mode still works on your machine.",
		"CLI_QUOTA_EXHAUSTED_TITLE":                         "Cloud credits exhausted",
		"CLI_QUOTA_EXHAUSTED_BODY":                          "Upgrade your plan or buy a top-up to keep using Trim cloud. Local Fast Mode still works on your machine.",
		"CLI_QUOTA_UPGRADE_HINT_FMT":                        "Upgrade: %s",
		"CLI_QUOTA_UPGRADE_OPENING":                         "Opening upgrade in your browser…",
		"CLI_QUOTA_UPGRADE_URL_MISSING":                     "Upgrade URL unavailable. Open the Trim dashboard to upgrade or buy a top-up.",
		"CLI_HELP_UPGRADE_SHORT":                            "Open upgrade / top-up in the browser when cloud credits are exhausted",
		"IDE_QUOTA_EXHAUSTED_TITLE":                         "Cloud credits exhausted",
		"IDE_QUOTA_EXHAUSTED_BODY":                          "Upgrade your plan or buy a top-up to keep using Trim cloud. Local Fast Mode still works on your machine.",
		"IDE_QUOTA_UPGRADE_ACTION":                          "Upgrade",
		"DASHBOARD_BUY_TOPUP_LABEL":                         "Buy top-up",
		"WORKSPACE_META_SEATS_FMT":                          "%d/%d seats",
		"WORKSPACE_META_SEATS_UNIT":                         " seats",
		"ADMIN_ENTERPRISE_ACTIVATE":                         "Send Paddle offer",
		"ADMIN_ENTERPRISE_ACTIVATE_DESC":                    "Marks the inquiry as offered and notifies the customer to pay via Paddle. Does not grant Enterprise access until checkout completes.",
		"ADMIN_ENTERPRISE_ACTIVATE_PENDING":                 "Sending offer…",
		"ADMIN_ENTERPRISE_ACTIVATED":                        "Paddle offer sent. Customer must complete checkout before Enterprise entitlements apply.",
		"ADMIN_ENTERPRISE_ACTIVATE_FAILED":                  "Could not send the Paddle offer for this inquiry.",
		"ADMIN_ENTERPRISE_OFFER":                            "Send Paddle offer",
		"ADMIN_ENTERPRISE_OFFER_PENDING":                    "Sending offer…",
		"ADMIN_ENTERPRISE_OFFER_DESC":                       "Marks the inquiry as offered and notifies the customer to complete Paddle checkout for the offered seat quantity. Entitlements apply only after payment succeeds.",
		"ADMIN_ENTERPRISE_OFFERED":                          "Paddle offer sent. Customer must complete checkout before Enterprise entitlements apply.",
		"ADMIN_ENTERPRISE_OFFER_FAILED":                     "Could not send the Paddle offer for this inquiry.",
		"ADMIN_ENTERPRISE_PRICE_MISSING":                    "Enterprise plan has no active Paddle price. Sync Plans to Paddle before sending an offer.",
		"ADMIN_ENTERPRISE_ALREADY_ACTIVATED":                "This inquiry is already activated after payment.",
		"ADMIN_ENTERPRISE_ALREADY_OFFERED":                  "This inquiry already has an active Paddle offer. Move status to Contacted to revise seats, or Closed to withdraw.",
		"ADMIN_ENTERPRISE_INQUIRY_CLOSED":                   "Closed inquiries cannot receive a Paddle offer. Set status to Contacted first.",
		"ADMIN_ENTERPRISE_OFFER_LOCKED_DESC":                "Offer terms are locked while status is Offered. Move to Contacted to revise seats and re-send, or Closed to withdraw. Entitlements still apply only after the customer pays.",
		"ADMIN_ENTERPRISE_STATUS_OFFERED":                   "Offered",
		"ADMIN_ENTERPRISE_CREDITS_REQUIRED":                 "Enterprise plan has no monthly credits and is not Unlimited. Set credits_monthly or turn Unlimited on in Plans before offering checkout.",
		"ADMIN_ENTERPRISE_PLAN_MISSING":                     "No active enterprise plan in the catalog.",
		"ADMIN_ENTERPRISE_USER_MISSING":                     "Inquiry has no linked user account.",
		"ADMIN_ENTERPRISE_SEATS_REQUIRED":                   "Set offered seats before sending a Paddle offer.",
		"ADMIN_ENTERPRISE_STATUS_ACTIVATED":                 "Activated",
		"ADMIN_ENTERPRISE_STATUS_DESC":                      "Inquiry workflow status (new, contacted, offered, closed, or activated).",
		"NOTIF_USER_ENTERPRISE_ACTIVE_TITLE":                "Enterprise plan activated",
		"NOTIF_USER_ENTERPRISE_ACTIVE_BODY":                 "Your Enterprise plan is active (%s seats). Payment completed via Paddle.",
		"NOTIF_USER_ENTERPRISE_OFFER_TITLE":                 "Enterprise offer ready",
		"NOTIF_USER_ENTERPRISE_OFFER_BODY":                  "Your Enterprise offer for %s seats is ready. Open the dashboard and complete Paddle checkout to activate.",
		"NOTIF_LOADING_MORE":                                "Loading more…",
		"NOTIF_MARK_READ_PENDING":                           "Updating…",
		"NOTIF_BELL_ARIA":                                   "Notifications",
		"NOTIF_PANEL_TITLE":                                 "Notifications",
		"NOTIF_EMPTY":                                       "No notifications yet.",
		"NOTIF_MARK_ALL_READ":                               "Mark all read",
		"NOTIF_MARK_ALL_PENDING":                            "Updating…",
		"NOTIF_MARK_READ":                                   "Mark read",
		"NOTIF_LIST_FAILED":                                 "Could not load notifications.",
		"NOTIF_MARK_FAILED":                                 "Could not update notification.",
		"NOTIF_BELL_ARIA_COUNT_FMT":                         "{aria}, {count}",
		"NOTIF_POLL_INTERVAL_MS":                            "60000",
		"SEO_TITLE_TEMPLATE":                                "%s · Trim",
		"SEO_DEFAULT_TITLE":                                 "Trim",
		"SEO_DEFAULT_DESCRIPTION":                           "Local context optimization middleware for AI coding tools - compress prompts on your machine, meter cloud usage with Trim.",
		"SEO_OG_SITE_NAME":                                  "Trim",
		"SEO_OG_TYPE":                                       "website",
		"SEO_TWITTER_CARD":                                  "summary_large_image",
		"SEO_ROBOTS_INDEX":                                  "index, follow",
		"SEO_ROBOTS_NOINDEX":                                "noindex, nofollow",
		"PAGE_404_CODE":                                     "404",
		"PAGE_404_TITLE":                                    "Page not found · Trim",
		"PAGE_404_HEADING":                                  "This page could not be found",
		"PAGE_404_BODY":                                     "The link may be broken, or the page may have moved. Try the home page or documentation.",
		"PAGE_404_HOME_CTA":                                 "Back to home",
		"PAGE_404_DOCS_CTA":                                 "Read the docs",
		"PAGE_404_DOCS_HREF":                                "/docs",
		"PAGE_ERROR_TITLE":                                  "Something went wrong · Trim",
		"PAGE_ERROR_HEADING":                                "Something went wrong",
		"PAGE_ERROR_BODY":                                   "An unexpected error occurred. Try again, or return home.",
		"PAGE_ERROR_RETRY_CTA":                              "Try again",
		"PAGE_ERROR_HOME_CTA":                               "Back to home",
		"ADMIN_PAGE_404_TITLE":                              "Page not found · Trim Admin",
		"ADMIN_PAGE_404_HEADING":                            "This page could not be found",
		"ADMIN_PAGE_404_BODY":                               "That admin route does not exist, or you do not have access. Return to the console home.",
		"ADMIN_PAGE_404_HOME_CTA":                           "Admin home",
		"ADMIN_PAGE_ERROR_TITLE":                            "Something went wrong · Trim Admin",
		"ADMIN_PAGE_ERROR_HEADING":                          "Something went wrong",
		"ADMIN_PAGE_ERROR_BODY":                             "An unexpected error occurred in the admin console. Try again, or return home.",
		"ADMIN_PAGE_ERROR_RETRY_CTA":                        "Try again",
		"ADMIN_PAGE_ERROR_HOME_CTA":                         "Admin home",
		"ADMIN_SEO_DEFAULT_TITLE":                           "Trim Admin",
		"ADMIN_SEO_DEFAULT_DESCRIPTION":                     "Trim platform administration console.",
		"ADMIN_SEO_ROBOTS":                                  "noindex, nofollow",
		"LANDING_NAV_CONTACT":                               "Contact",
		"APP_PATH_CONTACT":                                  "/contact",
		"CONTACT_PAGE_TITLE":                                "Contact · Trim",
		"CONTACT_PAGE_HEADING":                              "Contact us",
		"CONTACT_PAGE_BODY":                                 "Questions about Trim, billing, enterprise, or security? Email us - clicking a topic opens your mail app with a ready-made message.",
		"CONTACT_META_DESCRIPTION":                          "Contact Trim support for product help, billing, enterprise sales, or security reports.",
		"CONTACT_EMAIL_LABEL":                               "Email",
		"CONTACT_OPEN_MAIL_CTA":                             "Open mail app",
		"CONTACT_TOPICS_HEADING":                            "Choose a topic",
		"CONTACT_TOPIC_GENERAL_LABEL":                       "General support",
		"CONTACT_TOPIC_GENERAL_DESC":                        "Product questions, account help, and feedback.",
		"CONTACT_TOPIC_GENERAL_SUBJECT":                     "Trim support request",
		"CONTACT_TOPIC_GENERAL_BODY":                        "Hi Trim team,\n\nI need help with:\n\n- Account / email:\n- What I tried:\n- What I expected:\n\nThanks.",
		"CONTACT_TOPIC_BILLING_LABEL":                       "Billing",
		"CONTACT_TOPIC_BILLING_DESC":                        "Plans, receipts, upgrades, and payment issues.",
		"CONTACT_TOPIC_BILLING_SUBJECT":                     "Trim billing inquiry",
		"CONTACT_TOPIC_BILLING_BODY":                        "Hi Trim billing team,\n\nI need help with:\n\n- Account / email:\n- Plan or receipt ID (if any):\n- Issue:\n\nThanks.",
		"CONTACT_TOPIC_SALES_LABEL":                         "Sales / Enterprise",
		"CONTACT_TOPIC_SALES_DESC":                          "Team seats, contracts, and custom needs.",
		"CONTACT_TOPIC_SALES_SUBJECT":                       "Trim Enterprise inquiry",
		"CONTACT_TOPIC_SALES_BODY":                          "Hi Trim sales team,\n\nI am interested in Enterprise for:\n\n- Company:\n- Approximate seats:\n- Use case:\n\nThanks.",
		"CONTACT_TOPIC_SECURITY_LABEL":                      "Security",
		"CONTACT_TOPIC_SECURITY_DESC":                       "Vulnerability reports and security concerns.",
		"CONTACT_TOPIC_SECURITY_SUBJECT":                    "Trim security report",
		"CONTACT_TOPIC_SECURITY_BODY":                       "Hi Trim security team,\n\nI would like to report:\n\n- Summary:\n- Impact:\n- Steps to reproduce (no secrets):\n\nThanks.",
		"CHECKOUT_SUCCESS_TITLE":                            "Payment successful",
		"CHECKOUT_SUCCESS_BODY":                             "Thanks for your purchase. We are activating your plan and emailing your order details.",
		"CHECKOUT_SUCCESS_DASHBOARD_LABEL":                  "Go to dashboard",
		"CHECKOUT_SUCCESS_RECEIPTS_LABEL":                   "View receipts",
		"CHECKOUT_SUCCESS_TOAST":                            "Payment received. Your plan is updating.",
		"STATUS_ENTERPRISE_CONTRACT":                        "Enterprise",
		"ENTERPRISE_INQUIRY_REQUIRED":                       "A sales offer is required before Enterprise checkout.",
		"ENTERPRISE_INQUIRY_NOT_OFFERED":                    "This inquiry is not ready for checkout. Wait for a sales offer.",
		"ENTERPRISE_INQUIRY_SEATS_MISSING":                  "Offered seat quantity is missing on this inquiry.",
		"ENTERPRISE_ME_TITLE":                               "Enterprise inquiries",
		"ENTERPRISE_ME_EMPTY":                               "No enterprise inquiries yet. Contact sales from the Enterprise plan card.",
		"ENTERPRISE_ME_COL_COMPANY":                         "Company",
		"ENTERPRISE_ME_COL_STATUS":                          "Status",
		"ENTERPRISE_ME_COL_REQUESTED":                       "Requested seats",
		"ENTERPRISE_ME_COL_OFFERED":                         "Offered seats",
		"ENTERPRISE_ME_COL_CREATED":                         "Created",
		"ENTERPRISE_ME_COL_UPDATED":                         "Updated",
		"ENTERPRISE_ME_COL_MESSAGE":                         "Message",
		"ENTERPRISE_ME_PAY":                                 "Pay with Paddle",
		"ENTERPRISE_ME_PAY_PENDING":                         "Opening checkout…",
		"ENTERPRISE_ME_VIEW":                                "View",
		"ENTERPRISE_ME_DETAILS":                             "Inquiry details",
		"ENTERPRISE_ME_FILTER_STATUS":                       "Status",
		"ENTERPRISE_ME_FILTER_STATUS_DESC":                  "Filter inquiries by workflow status.",
		"ENTERPRISE_ME_STATUS_NEW":                          "New",
		"ENTERPRISE_ME_STATUS_CONTACTED":                    "Contacted",
		"ENTERPRISE_ME_STATUS_OFFERED":                      "Offered - pay to activate",
		"ENTERPRISE_ME_STATUS_CLOSED":                       "Closed",
		"ENTERPRISE_ME_STATUS_ACTIVATED":                    "Activated",
		"ENTERPRISE_ME_SEARCH":                              "Search inquiries",
		"ENTERPRISE_ME_SEARCH_DESC":                         "Filters by company or message as you type.",
		"ENTERPRISE_ME_DELETE":                              "Delete",
		"ENTERPRISE_ME_DELETE_PENDING":                      "Deleting…",
		"ENTERPRISE_ME_DELETE_CONFIRM":                      "Delete this inquiry permanently? Activated inquiries cannot be deleted.",
		"ENTERPRISE_ME_BULK_DELETE_CONFIRM":                 "Delete the selected inquiries permanently? Activated inquiries are skipped. This cannot be undone.",
		"ENTERPRISE_ME_DELETED":                             "Inquiry deleted.",
		"ENTERPRISE_ME_BULK_DELETED":                        "Selected inquiries deleted.",
		"ENTERPRISE_ME_DELETE_FAILED":                       "Could not delete inquiries.",
		"ENTERPRISE_ME_DELETE_ACTIVATED":                    "Activated inquiries cannot be deleted.",
		"ENTERPRISE_ME_IDS_REQUIRED":                        "Select at least one inquiry.",
		"ENTERPRISE_ME_IDS_INVALID":                         "One or more inquiry ids are invalid.",
		"ENTERPRISE_CHECKOUT_INTERVAL_REQUIRED":             "Billing interval is not configured for Enterprise checkout.",
		"EVENTS_DELETE":                                     "Delete",
		"EVENTS_DELETE_PENDING":                             "Deleting…",
		"EVENTS_DELETE_CONFIRM":                             "Delete this trace permanently? This cannot be undone.",
		"EVENTS_BULK_DELETE_CONFIRM":                        "Delete the selected traces permanently? This cannot be undone.",
		"EVENTS_DELETED":                                    "Trace deleted.",
		"EVENTS_BULK_DELETED":                               "Selected traces deleted.",
		"EVENTS_DELETE_FAILED":                              "Could not delete traces.",
		"EVENTS_IDS_REQUIRED":                               "Select at least one trace.",
		"EVENTS_IDS_INVALID":                                "One or more trace ids are invalid.",
		"ACTION:ENTERPRISE_OFFER_CHECKOUT":                  "Pay with Paddle",
		"PENDING:ENTERPRISE_OFFER_CHECKOUT":                 "Opening checkout…",
		"DECIDE_ENTERPRISE_OFFER_CHECKOUT":                  "Complete Paddle checkout for your offered Enterprise seats.",
		"ADMIN_ENTERPRISE_COL_SEATS":                        "Requested seats",
		"ADMIN_ENTERPRISE_COL_CREATED":                      "Created",
		"ADMIN_ENTERPRISE_COL_MESSAGE":                      "Message",
		"ADMIN_ENTERPRISE_DETAILS":                          "Inquiry details",
		"ADMIN_ENTERPRISE_INQUIRY_MESSAGE":                  "Customer message",
		"ADMIN_ENTERPRISE_INQUIRY_MESSAGE_DESC":             "Message the customer submitted with this enterprise inquiry.",
		"ADMIN_ENTERPRISE_ESTIMATED_SEATS":                  "Requested seats",
		"ADMIN_ENTERPRISE_ESTIMATED_SEATS_DESC":             "Seat estimate the customer entered when submitting the inquiry (estimated_seats).",
		"ADMIN_ENTERPRISE_USER":                             "User",
		"ADMIN_ENTERPRISE_USER_DESC":                        "Signed-in account that submitted the inquiry.",
		"ADMIN_ENTERPRISE_CREATED":                          "Created",
		"ADMIN_ENTERPRISE_UPDATED":                          "Updated",
		"ADMIN_ENTERPRISE_CONTRACT_NOTES_DESC":              "Internal contract or sales notes stored on the enterprise inquiry (contract_notes).",
		"ADMIN_ENTERPRISE_OFFERED_SEATS_DESC":               "Seat quantity offered in the proposal (offered_seat_quantity).",
		"ADMIN_NAV_SECTION_OVERVIEW":                        "Overview",
		"ADMIN_NAV_SECTION_CUSTOMERS":                       "Customers",
		"ADMIN_NAV_SECTION_SALES":                           "Sales and revenue",
		"ADMIN_NAV_SECTION_CATALOG":                         "Catalog and billing config",
		"ADMIN_NAV_SECTION_ACCESS":                          "Access and security",
		"ADMIN_NAV_SECTION_PRODUCT":                         "Product and growth",
		"ADMIN_NAV_SECTION_PLATFORM":                        "Platform",
		"ADMIN_NAV_REVENUE":                                 "Revenue",
		"ADMIN_REVENUE_TITLE":                               "Revenue",
		"ADMIN_REVENUE_INTRO":                               "Settled revenue from completed receipts only. Catalog price times seats is not used here.",
		"ADMIN_REVENUE_KPI_TOTAL":                           "Settled revenue",
		"ADMIN_REVENUE_KPI_RECEIPTS":                        "Completed receipts",
		"ADMIN_REVENUE_KPI_REFUNDED":                        "Refunded receipts",
		"ADMIN_REVENUE_KPI_FREE_ACCOUNTS":                   "Free accounts",
		"ADMIN_REVENUE_KPI_PAID_ACCOUNTS":                   "Paid accounts",
		"ADMIN_REVENUE_CHART_TITLE":                         "Settled revenue by plan",
		"ADMIN_REVENUE_BY_PLAN_TITLE":                       "By plan",
		"ADMIN_REVENUE_COL_PLAN":                            "Plan",
		"ADMIN_REVENUE_COL_KIND":                            "Kind",
		"ADMIN_REVENUE_COL_REVENUE":                         "Revenue",
		"ADMIN_REVENUE_COL_RECEIPTS":                        "Receipts",
		"ADMIN_REVENUE_DRILL_TITLE":                         "Receipts in range",
		"ADMIN_REVENUE_DRILL_LINK":                          "Open receipts",
		"ADMIN_REVENUE_EMPTY":                               "No completed receipts in this range.",
		"ADMIN_REVENUE_RANGE_LABEL":                         "Range",
		"ADMIN_REVENUE_RANGE_DESC":                          "Filter settled receipt revenue by time range.",
		"ADMIN_REVENUE_RANGE_7D":                            "Last 7 days",
		"ADMIN_REVENUE_RANGE_30D":                           "Last 30 days",
		"ADMIN_REVENUE_RANGE_MTD":                           "Month to date",
		"ADMIN_REVENUE_RANGE_YTD":                           "Year to date",
		"ADMIN_REVENUE_RANGE_CUSTOM":                        "Custom calendar",
		"ADMIN_REVENUE_UNKNOWN_PLAN":                        "Unmapped",
		"ADMIN_DASHBOARD_REVENUE_HINT":                      "Deep settled revenue by plan lives under Sales and revenue.",
		"ADMIN_DASHBOARD_REVENUE_LINK":                      "Open revenue",
		"ADMIN_PLAN_CREATE":                                 "Create plan",
		"ADMIN_PLAN_CREATE_PENDING":                         "Creating…",
		"ADMIN_PENDING_CREATING":                            "Creating...",
		"ADMIN_PENDING_UPDATING":                            "Updating...",
		"ADMIN_PENDING_EXPORTING":                           "Exporting...",
		"ADMIN_BILLING_TAB_PRICING":                         "Pricing",
		"ADMIN_BILLING_TAB_DEEP":                            "Deep targets",
		"ADMIN_BILLING_TAB_LISTS":                           "Lists and charts",
		"ADMIN_BILLING_TAB_POLICY":                          "Policy",
		"ADMIN_PRODUCT_TAB_DEFAULTS":                        "Defaults",
		"ADMIN_PRODUCT_TAB_LIMITS":                          "Limits",
		"ADMIN_PRODUCT_TAB_CLI":                             "CLI",
		"ADMIN_PRODUCT_TAB_CHURN":                           "Churn",
		"ADMIN_PRODUCT_TAB_FAST":                            "Fast mode",
		"ADMIN_KPI_MRR_CENTS":                               "MRR proxy",
		"ADMIN_KPI_USERS_TOTAL":                             "Users",
		"ADMIN_KPI_SUBSCRIPTIONS_ACTIVE":                    "Active subscriptions",
		"ADMIN_KPI_EVENTS_24H":                              "Events (24h)",
		"ADMIN_KPI_INSTALL_HITS_7D":                         "Install hits (7d)",
		"ADMIN_KPI_ACTIVE_TODAY":                            "Active today",
		"ADMIN_KPI_ACTIVE_7D":                               "Active (7d)",
		"ADMIN_KPI_ACTIVE_30D":                              "Active (30d)",
		"ADMIN_KPI_PAID_SEATS":                              "Paid seats",
		"ADMIN_KPI_ARR_CENTS":                               "ARR proxy",
		"ADMIN_KPI_CHURN_30D":                               "Canceled subs (30d)",
		"ADMIN_KPI_NEW_PAID_30D":                            "New paid (30d)",
		"ADMIN_KPI_FREE_TO_PAID_30D":                        "Free to paid (30d)",
		"ADMIN_OPS_CHECKLIST_TITLE":                         "Ops checklist",
		"ADMIN_HEALTH_TITLE":                                "Health",
		"ADMIN_ALERTS_TITLE":                                "Alerts",
		"ADMIN_CHECKLIST_POSTGRES":                          "PostgreSQL reachable",
		"ADMIN_CHECKLIST_REDIS":                             "Redis reachable",
		"ADMIN_CHECKLIST_OWNERS":                            "Platform owner assigned",
		"ADMIN_CHECKLIST_GITHUB":                            "GitHub distribution configured",
		"ADMIN_CHECKLIST_PRICING":                           "Pricing bound",
		"ADMIN_CHECKLIST_COMPANY":                           "Company legal name set",
		"ADMIN_CHECKLIST_GEOLITE":                           "GeoLite MMDB configured",
		"ADMIN_CHECKLIST_IDP":                               "Auth providers configured",
		"ADMIN_CHECKLIST_MIGRATIONS":                        "TOTP migration applied",
		"ADMIN_CHECKLIST_STATUS_OK":                         "ok",
		"ADMIN_CHECKLIST_STATUS_FAIL":                       "fail",
		"ADMIN_HEALTH_POSTGRES":                             "PostgreSQL",
		"ADMIN_HEALTH_REDIS":                                "Redis",
		"ADMIN_DATE_RANGE_PLACEHOLDER":                      "Filter by date",
		"ADMIN_DATE_RANGE_CLEAR":                            "Clear dates",
		"ADMIN_DATE_RANGE_APPLY":                            "Done",
		"ADMIN_PLAN_ID":                                     "Plan id",
		"ADMIN_BILLING_PRICING_BOUND_HINT":                  "Pricing bound is set automatically after Sync to Paddle succeeds for all public sellable plans.",
		"ADMIN_BILLING_ANNUAL_DISCOUNT_HINT":                "Saving annual discount % updates yearly cents on subscription plans (monthly×12×(100−%)/100) and re-syncs Paddle yearly prices.",
		"ADMIN_PRICING_UNBOUND":                             "Pricing is not bound. Save plan amounts, then Sync to Paddle. Self-serve checkout stays fail-closed until pricing_bound is true.",
		"ADMIN_RECEIPT_RESYNC_FAILED":                       "Could not resync receipt from Paddle.",
		"ADMIN_BRAND":                                       "Trim Operator",
		"ADMIN_TAGLINE":                                     "Instance operator console",
		"ADMIN_SIGN_OUT":                                    "Sign out",
		"ADMIN_SIGN_OUT_PENDING":                            "Signing out...",
		"ADMIN_WEBAUTHN_STATUS_ON":                          "Passkey enrolled",
		"ADMIN_WEBAUTHN_STATUS_OFF":                         "Passkey not enrolled",
		"ADMIN_WEBAUTHN_ENROLL_TITLE":                       "Passkey",
		"ADMIN_WEBAUTHN_ENROLL_INTRO":                       "Uses this browser and device. After setup, you confirm sensitive actions with fingerprint, face unlock, PIN, or a security key.",
		"ADMIN_WEBAUTHN_STEP_1":                             "Optional: give this passkey a short name so you recognize it later (for example Work laptop).",
		"ADMIN_WEBAUTHN_STEP_2":                             "Register the passkey, then approve the browser prompt with your fingerprint, face unlock, PIN, or security key.",
		"ADMIN_WEBAUTHN_VERIFY_TITLE":                       "Confirm with passkey",
		"ADMIN_WEBAUTHN_BEGIN_REGISTER":                     "Register passkey",
		"ADMIN_WEBAUTHN_BEGIN_ASSERT":                       "Use passkey",
		"ADMIN_WEBAUTHN_REMOVE":                             "Remove passkey",
		"ADMIN_WEBAUTHN_NAME_LABEL":                         "Passkey name",
		"ADMIN_WEBAUTHN_NAME_PLACEHOLDER":                   "Work laptop",
		"ADMIN_WEBAUTHN_NAME_HINT":                          "Optional. Helps you tell devices apart if you add more than one.",
		"ADMIN_WEBAUTHN_RP_MISSING":                         "Passkeys are not configured on this instance. Ask an operator to set ADMIN_WEBAUTHN_RP_ID or ADMIN_ALLOWED_ORIGINS, or use an authenticator app instead.",
		"ADMIN_WEBAUTHN_UNAVAILABLE":                        "Passkey step-up is unavailable.",
		"ADMIN_WEBAUTHN_REGISTER_FAILED":                    "Passkey registration failed.",
		"ADMIN_WEBAUTHN_ASSERT_FAILED":                      "Passkey verification failed.",
		"ADMIN_WEBAUTHN_NOT_ENROLLED":                       "No passkey enrolled for this admin.",
		"ADMIN_WEBAUTHN_ALREADY":                            "A passkey with this credential already exists.",
		"ADMIN_WEBAUTHN_BROWSER_UNSUPPORTED":                "This browser does not support passkeys. Use an authenticator app instead, or try a current Chrome, Edge, Safari, or Firefox release.",
		"ADMIN_STEP_UP_FACTOR_REQUIRED":                     "Set up an authenticator app or a passkey before you can approve sensitive admin actions.",
		"ADMIN_STEP_UP_ENROLL_CHOICE":                       "You only need one method. An authenticator app works on any phone. A passkey uses this device fingerprint, face unlock, PIN, or a security key.",
		"ADMIN_STEP_UP_VERIFY_HINT":                         "Open your authenticator app for the current 6-digit code, or use your passkey on this device.",
		"ADMIN_STEP_UP_METHODS_TITLE":                       "Authenticator and passkey",
		"ADMIN_STEP_UP_METHODS_SUBTITLE":                    "Add either method, or both. You can return here anytime to finish setup.",
		"ADMIN_STEP_UP_METHOD_ENROLLED":                     "Enrolled",
		"ADMIN_STEP_UP_SHOW_METHODS":                        "Manage authenticator and passkey",
		"ADMIN_STEP_UP_SHOW_VERIFY":                         "Back to methods",
		"ADMIN_STEP_UP_OPEN_VERIFY":                         "Verify step-up now",
		"ADMIN_STEP_UP_ACTIVE":                              "Authenticator / passkey is enrolled. Protected admin writes are unlocked.",
		"ADMIN_STEP_UP_SUCCESS":                             "Authenticator or passkey enrolled. You can use protected admin actions.",
		"ADMIN_TOTP_ENROLL_SUCCESS":                         "Authenticator app enrolled successfully.",
		"ADMIN_WEBAUTHN_ENROLL_SUCCESS":                     "Passkey enrolled successfully.",
		"ADMIN_TOTP_ENROLL_TITLE":                           "Authenticator app",
		"ADMIN_TOTP_ENROLL_INTRO":                           "Uses a free authenticator app such as Google Authenticator, Microsoft Authenticator, Authy, or 1Password.",
		"ADMIN_TOTP_STEP_1":                                 "Generate a QR code for this admin account.",
		"ADMIN_TOTP_STEP_2":                                 "Open your authenticator app, add a new account, and scan this QR code. If you cannot scan, enter the manual setup key instead.",
		"ADMIN_TOTP_STEP_3":                                 "Enter the 6-digit code currently shown in the app, then confirm to finish setup.",
		"ADMIN_TOTP_VERIFY_TITLE":                           "Confirm with authenticator",
		"ADMIN_TOTP_CODE_LABEL":                             "6-digit code from your app",
		"ADMIN_TOTP_CODE_PLACEHOLDER":                       "000000",
		"ADMIN_TOTP_SECRET_LABEL":                           "Manual setup key",
		"ADMIN_TOTP_SECRET_HINT":                            "Use this only if your app cannot scan the QR code. Do not share it.",
		"ADMIN_TOTP_QR_CAPTION":                             "Scan with your authenticator app",
		"ADMIN_TOTP_BEGIN":                                  "Generate QR code",
		"ADMIN_TOTP_CONFIRM":                                "Confirm and finish",
		"ADMIN_TOTP_REQUIRED":                               "Enroll TOTP before you can approve sensitive admin actions.",
		"ADMIN_TOTP_CODE_INVALID":                           "That code is incorrect or expired. Wait for a new code in your authenticator app and try again.",
		"ADMIN_TOTP_CODE_REQUIRED":                          "Enter the 6-digit code from your authenticator app.",
		"ADMIN_CHECKLIST_WEBAUTHN_RP":                       "WebAuthn RP id configured",
		"ADMIN_NAV_CREDITS":                                 "Credits and top-ups",
		"ADMIN_LOGIN_TITLE":                                 "Operator sign in",
		"ADMIN_LOGIN_DEFAULT_NEXT":                          "/",
		"ADMIN_LOGIN_DEFAULT_NEXT_MISSING":                  "Admin sign-in destination is not configured. Set ADMIN_LOGIN_DEFAULT_NEXT in site_messages.",
		"ADMIN_CHECKLIST_MIGRATIONS_NOTIF":                  "In-app notifications migration applied",
		"ADMIN_CHECKLIST_MIGRATIONS_WEBAUTHN":               "WebAuthn credentials migration applied",
		"ADMIN_CHECKLIST_MIGRATIONS_CREDITS_NAV":            "Credits nav migration applied",
		"ADMIN_ACCESS_REVIEW_FILENAME_FMT":                  "access-review-{generated_at}.xlsx",
		"ADMIN_AUDIT_EXPORT_FILENAME_FMT":                   "admin-audit-{generated_at}.xlsx",
		"ADMIN_AUDIT_EXPORT_LIMIT_MISSING":                  "Set admin_retention_settings.audit_export_max_rows before exporting audit logs.",
		"ADMIN_ACCESS_REVIEW_ATTEST_LIMIT_MISSING":          "Set admin_retention_settings.access_review_attestations_limit before loading access review.",
		"ADMIN_CHECKLIST_MIGRATIONS_AUDIT_EXPORT":           "Audit export / chrome UI map migration applied",
		"ADMIN_COMPLIANCE_AUDIT_EXPORT_MAX":                 "Audit export max rows",
		"ADMIN_COMPLIANCE_ATTEST_LIMIT":                     "Access review attestations limit",
		"ADMIN_COMPLIANCE_BREAK_GLASS_TTL":                  "Break-glass TTL (minutes)",
		"ADMIN_RETENTION_SETTINGS_MISSING":                  "Admin retention settings row is missing.",
		"ADMIN_AUDIT_COL_ACTOR":                             "Actor",
		"ADMIN_AUDIT_COL_REASON":                            "Reason",
		"ADMIN_AUDIT_COL_STEP_UP":                           "Step-up",
		"ADMIN_AUDIT_STEP_UP_YES":                           "Yes",
		"ADMIN_AUDIT_STEP_UP_NO":                            "No",
		"ADMIN_DISPUTE_LIST_TITLE":                          "Dispute notes",
		"ADMIN_DISPUTE_COL_WHEN":                            "When",
		"ADMIN_DISPUTE_COL_CREATED_BY":                      "Created by",
		"ADMIN_DISPUTE_STATUS_WATCHING":                     "Watching",
		"ADMIN_CHROME_COL_CODE":                             "Code",
		"ADMIN_CHROME_COL_BODY":                             "Body",
		"ADMIN_RECEIPT_BILL_TO_REGION":                      "Bill-to region",
		"ADMIN_PRODUCT_MIN_CLI":                             "Minimum CLI version",
		"ADMIN_RBAC_SAVE_ROLE":                              "Save role permissions",
		"ADMIN_RBAC_SELECT_ROLE":                            "Select a role to edit permissions",
		"ADMIN_CHECKLIST_MIGRATIONS_MIN_CLI":                "Product min CLI column migration applied",
		"ADMIN_OBS_HEATMAP_COL_PAID":                        "Paid seats",
		"ADMIN_OBS_HEATMAP_PAID_TITLE":                      "Logins and paid by country",
		"ADMIN_RBAC_DELETE_ROLE":                            "Delete role",
		"ADMIN_ROLE_IN_USE":                                 "Reassign or remove admins on this role before deleting it.",
		"ADMIN_CREDITS_GRANT_TITLE":                         "Manual credit grant",
		"ADMIN_CREDITS_GRANT_USER":                          "User id",
		"ADMIN_CREDITS_GRANT_AMOUNT":                        "Credits",
		"ADMIN_CREDITS_GRANT_ACTION":                        "Grant credits",
		"ADMIN_DIST_SYNC":                                   "Sync distribution stats",
		"ADMIN_DIST_COL_PATH":                               "Path",
		"ADMIN_PRODUCT_QUEUE_WARN_DEPTH":                    "Paddle webhook queue warn depth",
		"ADMIN_CHECKLIST_MIGRATIONS_QUEUE_WARN":             "Paddle queue warn depth migration applied",
		"ADMIN_USER_LOGIN_COUNTRY":                          "Last login country",
		"ADMIN_USER_BILL_TO_COUNTRY":                        "Bill-to country",
		"ADMIN_RECEIPT_BILL_TO_COUNTRY":                     "Bill-to country",
		"ADMIN_RECEIPT_CURRENCY":                            "Currency",
		"ADMIN_HEALTH_ERROR_RATE_FMT":                       "{pct}%",
		"ADMIN_HEALTH_QUEUE_DEPTH_FMT":                      "{depth}",
		"ADMIN_HEALTH_AGE_SUFFIX_FMT":                       " ({age}s)",
		"ADMIN_RECEIPT_TAX_TOTAL_FMT":                       "{tax} / {total}",
		"ADMIN_SEGMENT_CREDITS_FMT":                         "{used}/{limit}",
		"ADMIN_RUNTIME_DEPLOYMENT_MODE":                     "Deployment mode",
		"ADMIN_RUNTIME_COMPRESSION_MODE_ENV":                "Compression mode (env)",
		"ADMIN_RUNTIME_MIN_CLI_VERSION":                     "Effective min CLI",
		"ADMIN_RUNTIME_MIN_CLI_VERSION_ENV":                 "Min CLI (env)",
		"ADMIN_RUNTIME_POW_DIFFICULTY":                      "PoW difficulty (runtime)",
		"ADMIN_RUNTIME_RATE_LIMIT_IP":                       "Rate limit IP/min (runtime)",
		"ADMIN_RUNTIME_RATE_LIMIT_USER":                     "Rate limit user/min (runtime)",
		"ADMIN_RUNTIME_MAX_ACCOUNTS_HW":                     "Max accounts per hardware (runtime)",
		"ADMIN_RUNTIME_MAX_ACCOUNTS_JA4":                    "Max accounts per JA4 (runtime)",
		"ADMIN_RUNTIME_CF_THREAT_SCORE_MIN":                 "CF threat score min (runtime)",
		"ADMIN_RUNTIME_GEOLITE_ASN_PATH":                    "GeoLite ASN MMDB",
		"ADMIN_RUNTIME_GEOLITE_ANON_PATH":                   "GeoLite anonymous MMDB",
		"ADMIN_RUNTIME_CHEAP_MODEL":                         "Cheap model",
		"ADMIN_RUNTIME_ROUTE_MAX_TOKENS":                    "Route max tokens",
		"ADMIN_RUNTIME_SMTP_CONFIGURED":                     "SMTP configured",
		"ADMIN_UI_SEP_DOT":                                  " · ",
		"ADMIN_UI_SEP_COMMA":                                ", ",
		"ADMIN_META_FIELD_FMT":                              "{label}: {value}",
		"ADMIN_PLATFORM_ADMIN_STATUS_ACTIVE":                "Active",
		"ADMIN_PLATFORM_ADMIN_STATUS_DISABLED":              "Disabled",
		"ADMIN_BREAK_GLASS_STATUS_PENDING":                  "Pending",
		"ADMIN_BREAK_GLASS_STATUS_APPROVED":                 "Approved",
		"ADMIN_BREAK_GLASS_STATUS_DENIED":                   "Denied",
		"ADMIN_BREAK_GLASS_STATUS_REVOKED":                  "Revoked",
		"ADMIN_WEBHOOK_STATUS_OK":                           "OK",
		"ADMIN_WEBHOOK_STATUS_FAILED":                       "Failed",
		"ADMIN_WEBHOOK_STATUS_PENDING":                      "Pending",
		"ADMIN_DIST_SOURCE_GITHUB":                          "GitHub",
		"ADMIN_DIST_SOURCE_INSTALL":                         "Install / CDN",
		"ADMIN_DIST_SOURCE_HOMEBREW":                        "Homebrew tap",
		"ADMIN_DIST_SOURCE_SCOOP":                           "Scoop bucket",
		"ADMIN_DIST_SOURCE_WINGET":                          "winget fork",
		"ADMIN_DIST_SOURCE_MARKETPLACE":                     "VS Marketplace",
		"ADMIN_DIST_METRIC_CLONES":                          "Clones",
		"ADMIN_DIST_METRIC_VIEWS":                           "Visitors",
		"ADMIN_DIST_METRIC_RELEASE_DOWNLOADS":               "Release downloads",
		"ADMIN_DIST_METRIC_HITS":                            "Install hits",
		"ADMIN_DIST_METRIC_INSTALLS":                        "Installs",
		"ADMIN_DIST_HOMEBREW_NOT_CONFIGURED":                "Homebrew tap sync is not configured (TRIM_DIST_HOMEBREW_TAP_REPO).",
		"ADMIN_DIST_SCOOP_NOT_CONFIGURED":                   "Scoop bucket sync is not configured (TRIM_DIST_SCOOP_BUCKET_REPO).",
		"ADMIN_DIST_WINGET_NOT_CONFIGURED":                  "winget fork sync is not configured (TRIM_DIST_WINGET_FORK_REPO).",
		"ADMIN_DIST_MARKETPLACE_NOT_CONFIGURED":             "VS Marketplace sync is not configured (TRIM_DIST_VSCODE_EXTENSION_ID).",
		"ADMIN_DIST_CHANNEL_REPO_REQUIRED":                  "Distribution channel repository is required.",
		"ADMIN_DIST_MARKETPLACE_ID_REQUIRED":                "VS Marketplace extension id is required (publisher.name).",
		"ADMIN_DIST_MARKETPLACE_STATS_MISSING":              "VS Marketplace did not return install statistics for this extension.",
		"ADMIN_GDPR_EXPORT_FILENAME_FMT":                    "gdpr-export-{user_id}-{generated_at}.xlsx",
		"ADMIN_CHURN_THRESHOLDS_MISSING":                    "Set churn_* columns on admin_product_settings before segment churn risk.",
		"ADMIN_PRODUCT_CHURN_HIGH_USAGE":                    "Churn high usage ratio",
		"ADMIN_PRODUCT_CHURN_MED_USAGE":                     "Churn medium usage ratio",
		"ADMIN_PRODUCT_CHURN_HIGH_IDLE":                     "Churn high idle days",
		"ADMIN_PRODUCT_CHURN_LOW_IDLE":                      "Churn low idle days",
		"ADMIN_CHECKLIST_MIGRATIONS_CHURN":                  "Churn thresholds / GDPR filename migration applied",
		"AUTH_GITHUB_OAUTH_SCOPES":                          "read:user user:email",
		"AUTH_OAUTH_LINK_INTENT":                            "link",
		"LOGIN_OAUTH_LINK_FAILED":                           "Could not connect that sign-in provider. Try again.",
		"AUTH_GITHUB_OAUTH_SCOPES_MISSING":                  "GitHub OAuth scopes are not configured. Set AUTH_GITHUB_OAUTH_SCOPES in site_messages.",
		"AUTH_GITLAB_OAUTH_SCOPES":                          "read_user",
		"AUTH_GITLAB_OAUTH_SCOPES_MISSING":                  "GitLab OAuth scopes are not configured. Set AUTH_GITLAB_OAUTH_SCOPES in site_messages.",
		"DEFAULT_COMPRESSION_TIER":                          "deep",
		"DEFAULT_COMPRESSION_TIER_MISSING":                  "DEFAULT_COMPRESSION_TIER is not configured in site_messages",
		"DEFAULT_DEEP_ENGINE":                               "v2",
		"DEFAULT_DEEP_ENGINE_MISSING":                       "DEFAULT_DEEP_ENGINE is not configured in site_messages",
		"DEFAULT_AUTO_START_WITH_IDE":                       "true",
		"DEFAULT_AUTO_START_WITH_IDE_MISSING":               "DEFAULT_AUTO_START_WITH_IDE is not configured in site_messages",
		"DEFAULT_AUTO_START_WITH_IDE_INVALID":               "DEFAULT_AUTO_START_WITH_IDE must be true or false",
		"LOCAL_AGENT_ONLINE_WITHIN_SEC":                     "900",
		"LOCAL_AGENT_ONLINE_WITHIN_SEC_MISSING":             "LOCAL_AGENT_ONLINE_WITHIN_SEC is not configured in site_messages",
		"LOCAL_AGENT_ONLINE_WITHIN_SEC_INVALID":             "LOCAL_AGENT_ONLINE_WITHIN_SEC must be a positive integer (seconds)",
		"COMPRESSION_TIER_FAST":                             "fast",
		"COMPRESSION_TIER_DEEP":                             "deep",
		"COMPRESSION_TIER_FAST_MISSING":                     "COMPRESSION_TIER_FAST is not configured in site_messages.",
		"COMPRESSION_TIER_DEEP_MISSING":                     "COMPRESSION_TIER_DEEP is not configured in site_messages.",
		"RECEIPT_FOOTER":                                    "Questions about this invoice? Contact support using the address on this document. Card statements may show Paddle as the merchant of record.",
		"RECEIPT_ISSUED_PREFIX":                             "Issued",
		"RECEIPT_DOCUMENT_TITLE":                            "Tax invoice",
		"RECEIPT_SECTION_BILL_TO":                           "Invoice to",
		"RECEIPT_SECTION_INVOICE_FROM":                      "Invoice from",
		"RECEIPT_SECTION_INVOICE_DETAILS":                   "Invoice details",
		"RECEIPT_SECTION_TRANSACTION":                       "Transaction",
		"RECEIPT_SECTION_TAX_BREAKDOWN":                     "Tax breakdown",
		"RECEIPT_SECTION_PAYMENT":                           "Payment method",
		"RECEIPT_SECTION_PERIOD":                            "Billing period",
		"RECEIPT_LABEL_AMOUNT_PAID":                         "Amount paid",
		"RECEIPT_LABEL_INVOICE_REFERENCE":                   "Invoice reference",
		"RECEIPT_LABEL_TRANSACTION_ID":                      "Transaction",
		"RECEIPT_LABEL_CURRENCY":                            "Currency code",
		"RECEIPT_LABEL_TAX_PERCENT":                         "Tax %",
		"RECEIPT_LABEL_TAX_TOTAL":                           "Tax total",
		"RECEIPT_LABEL_SUBTOTAL":                            "Subtotal",
		"RECEIPT_LABEL_TAX":                                 "VAT",
		"RECEIPT_LABEL_TOTAL":                               "Total",
		"RECEIPT_COL_PRODUCT":                               "Product",
		"RECEIPT_COL_DESCRIPTION":                           "Product",
		"RECEIPT_COL_QTY":                                   "Qty",
		"RECEIPT_COL_UNIT":                                  "Unit price",
		"RECEIPT_COL_TAX_RATE":                              "Tax rate",
		"RECEIPT_COL_AMOUNT":                                "Amount",
		"RECEIPT_TAX_RATE_ZERO":                             "0%",
		"RECEIPT_TAX_RATE_PERCENT_FMT":                      "%0.2f%%",
		"RECEIPT_MERCHANT_VIA":                              "via Paddle.com",
		"RECEIPT_HEADER_META_SEP":                           " - ",
		"RECEIPT_STATUS_COMPLETED":                          "PAID",
		"RECEIPT_VAT_ID_PREFIX":                             "VAT Number",
		"RECEIPT_PRICE_NAME_MONTHLY_FMT":                    "%s (monthly)",
		"RECEIPT_PRICE_NAME_YEARLY_FMT":                     "%s (yearly)",
		"PAGINATION_FIRST":                                  "First",
		"PAGINATION_LAST":                                   "Last",
		"PAGINATION_SKIP_TO_LABEL":                          "Skip to page",
		"PAGINATION_SKIP_TO_INVALID":                        "That page is outside the available range.",
		"CLI_VERSION_HEADER_MISSING":                        "Missing X-Client-Version. Upgrade the Trim CLI.",
		"CLI_VERSION_OUTDATED":                              "CLI version is outdated. Run: trim update",
		"AUTH_CREDENTIALS_INVALID":                          "Invalid credentials.",
		"AUTH_CREDENTIALS_REVOKED":                          "Invalid or revoked credentials. Sign in again.",
		"AUTH_PROVIDER_DENIED_EMPTY":                        "Sign-in provider is not allowed. No auth providers are configured on this server.",
		"AUTH_PROVIDER_DENIED_FMT":                          "Sign-in provider is not allowed. Enabled providers: %s.",
		"RATE_LIMIT_USER":                                   "Rate limit exceeded. Slow down and retry.",
		"FREE_TIER_PROXY_BLOCKED":                           "Proxy or VPN requests are restricted on the free tier.",
		"HARDWARE_UUID_REQUIRED":                            "X-Hardware-UUID is required for CLI requests on the free tier.",
		"HARDWARE_ACCOUNT_LIMIT":                            "Hardware limit exceeded. Upgrade to Pro for more workspaces.",
		"JA4_ACCOUNT_LIMIT":                                 "Client fingerprint limit exceeded. Upgrade to Pro or contact support.",
		"DATABASE_UNAVAILABLE":                              "Database unavailable.",
		"QUOTA_LOAD_FAILED":                                 "Failed to load quota.",
		"QUOTA_EXHAUSTED":                                   "Monthly quota exhausted.",
		"WORKSPACE_NOT_MEMBER":                              "Not a member of this workspace.",
		"WORKSPACE_QUOTA_LOAD_FAILED":                       "Failed to load workspace quota.",
		"WORKSPACE_QUOTA_EXHAUSTED":                         "Workspace shared quota exhausted.",
		"POW_REQUIRED":                                      "Free tier requires proof-of-work. Solve the challenge and retry with X-Trim-PoW.",
		"CANARY_NOT_FOUND":                                  "not found",
		"IP_BLOCKED":                                        "This network is blocked for abuse. Contact support.",
		"RATE_LIMIT_IP":                                     "Too many requests from this network. Slow down and retry.",
		"TLS_FINGERPRINT_BLOCKED":                           "Client fingerprint blocked for abuse.",
		"SUSPICIOUS_CLIENT":                                 "Invalid client signature.",
		"TIMESTAMP_REQUIRED":                                "X-Request-Timestamp is required for CLI requests.",
		"REQUEST_EXPIRED":                                   "Request timestamp skew too large.",
		"HARDWARE_BLOCKED":                                  "This device is blocked for abuse. Contact support.",
		"HARDWARE_UUID_MISMATCH":                            "This API key is locked to a different device. Re-issue the key on this machine.",
		"HARDWARE_UUID_BOUND_REQUIRED":                      "This API key requires X-Hardware-UUID.",
		"HARDWARE_DEVICE_LIMIT":                             "This API key already has the maximum number of registered devices. Revoke and re-issue, or remove a device.",
		"API_KEY_MAX_DEVICES_MISSING":                       "Set admin_product_settings.api_key_max_devices (1-16) before device-bound API keys can enroll.",
		"IP_DENIED":                                         "This IP address is denylisted. Contact support.",
		"ASN_DENIED":                                        "This network (ASN) is denylisted. Contact support.",
		"DENYLIST_UNAVAILABLE":                              "Abuse denylist unavailable. Try again shortly.",
		"AGENT_ID_REQUIRED":                                 "X-Trim-Agent-Id is required for API key requests.",
		"AGENT_ID_UNKNOWN":                                  "Unknown X-Trim-Agent-Id. Use a value from agent_identity_catalog (cli, ide, ci).",
		"AGENT_ID_MISMATCH":                                 "This API key is locked to a different agent identity.",
		"API_KEY_FRESH_HINT":                                "New key (copy now; it will not be shown again)",
		"API_KEY_FRESH_DEVICE_REQUIRED":                     "Copy this key now. It will not authenticate until you register a hardware UUID below (Settings → Register device).",
		"API_KEY_DEVICE_HINT":                               "Every API key must be device-bound. CLI login binds the issuing machine. For IDE/CI or dashboard-issued keys, register hardware under Settings (signed-in) before the key will authenticate.",
		"API_KEY_DEVICE_HW_LABEL":                           "Hardware UUID",
		"API_KEY_DEVICE_AGENT_LABEL":                        "Agent ID (cli, ide, or ci)",
		"API_KEY_DEVICE_AGENT_CLI":                          "cli",
		"API_KEY_DEVICE_AGENT_IDE":                          "ide",
		"API_KEY_DEVICE_AGENT_CI":                           "ci",
		"API_KEY_DEVICE_BOUND_LABEL":                        "Device-bound",
		"API_KEY_DEVICE_UNBOUND_LABEL":                      "Not device-bound",
		"API_KEY_DEVICE_COUNT_FMT":                          "%d device(s)",
		"API_KEY_DEVICE_HW_REQUIRED":                        "Hardware UUID is required.",
		"API_KEY_DEVICE_NOT_FOUND":                          "Device registration not found.",
		"API_KEY_DEVICE_UNBOUND":                            "This API key has no registered devices. Sign in to the dashboard, open Settings → API keys, and register this machine's hardware UUID before using the key.",
		"CI_AGENT_HINT":                                     "CI jobs must send X-Trim-Agent-Id: ci and X-Hardware-UUID for device-bound keys.",
		"IDE_CMD_COPY_HARDWARE_ID":                          "Trim: Copy Hardware ID",
		"IDE_HARDWARE_ID_COPIED":                            "Trim hardware ID copied. Register it under Dashboard → Settings → API keys.",
		"SIGNATURE_REQUIRED":                                "X-Trim-Signature is required for CLI requests.",
		"BODY_READ_FAILED":                                  "Cannot read request body for signature check.",
		"CLI_SIGNATURE_INVALID_FMT":                         "CLI signature invalid: %s",
		"CLI_SIGNATURE_DETAIL_HMAC":                         "HMAC mismatch",
		"IDE_STATUS_TOOLTIP":                                "Flush Trim IDE telemetry",
		"IDE_STATUS_IDLE":                                   "Trim",
		"IDE_STATUS_PENDING_FMT":                            "Trim $(cloud-upload) %d",
		"IDE_API_KEY_TITLE":                                 "Trim API key",
		"IDE_API_KEY_PROMPT":                                "Paste a key from Dashboard → Settings → API keys (trm_...)",
		"IDE_API_KEY_SAVED":                                 "Trim API key saved in Secret Storage.",
		"IDE_API_KEY_CLEARED":                               "Trim API key cleared.",
		"IDE_TELEMETRY_FLUSHED":                             "Trim telemetry flushed.",
		"IDE_TELEMETRY_FAILED_FMT":                          "Trim telemetry failed (%d): %s",
		"IDE_TELEMETRY_NETWORK_FMT":                         "Trim telemetry network error: %s",
		"IDE_EVENT_MODE":                                    "ide",
		"IDE_EVENT_STATUS":                                  "success",
		"IDE_EVENT_MODEL":                                   "ide",
		"IDE_CMD_SET_API_KEY":                               "Trim: Set API Key",
		"IDE_CMD_CLEAR_API_KEY":                             "Trim: Clear API Key",
		"IDE_CMD_MARK_TAB_SHOWN":                            "Trim: Mark Tab Suggestion Shown",
		"IDE_CMD_MARK_TAB_ACCEPTED":                         "Trim: Mark Tab Suggestion Accepted",
		"IDE_CMD_FLUSH_TELEMETRY":                           "Trim: Flush Telemetry Now",
		"IDE_CONFIG_TITLE":                                  "Trim",
		"IDE_CONFIG_API_URL":                                "Trim Cloud API base URL (required; no default host)",
		"IDE_CONFIG_AUTO_FLUSH":                             "Seconds between automatic telemetry flushes",
		"IDE_CONFIG_TRACK_EDITS":                            "Heuristic LOC tracking from multi-line inserts/deletes (labeled mode=ide)",
		"IDE_CONFIG_MIN_LINES":                              "Minimum AI-edited lines before an event is queued",
		"CLI_AUTH_COPIED_RESET_MS":                          "2000",
		"RECEIPT_PRINT_PENDING_MS":                          "400",
		"PADDLE_WEBHOOK_SECRET_MISSING":                     "Paddle webhook secret is not configured.",
		"PADDLE_SIGNATURE_INVALID":                          "Invalid Paddle webhook signature.",
		"PADDLE_PAYLOAD_INVALID":                            "Invalid Paddle webhook JSON.",
		"PADDLE_QUEUE_BUSY":                                 "Webhook queue is busy. Retry shortly.",
		"AUTH_PROVIDERS_UNAVAILABLE":                        "Auth providers unavailable.",
		"LEGAL_PRIVACY_UNAVAILABLE":                         "Legal privacy content unavailable.",
		"LEGAL_TERMS_UNAVAILABLE":                           "Legal terms content unavailable.",
		"CRON_UNAUTHORIZED":                                 "unauthorized",
		"EXPIRE_BATCH_FAILED":                               "expire failed",
		"QUOTA_NOT_FOUND":                                   "quota not found",
		"RECEIPT_NOT_FOUND_ERROR":                           "receipt not found",
		"PROFILE_EMAIL_NOT_FOUND":                           "profile email not found",
		"INVITE_NOT_FOUND_OR_CLOSED":                        "invite not found or already closed",
		"INVITE_NOT_FOUND":                                  "invite not found",
		"PROFILE_NOT_FOUND":                                 "profile not found",
		"MEMBER_NOT_FOUND":                                  "member not found",
		"API_KEY_NOT_FOUND_OR_REVOKED":                      "api key not found or already revoked",
		"INVITE_REVOKE_FAILED":                              "failed to revoke invite",
		"INVITE_TOKEN_REQUIRED":                             "token is required",
		"INVITE_NO_LONGER_PENDING":                          "invite is no longer pending",
		"INVITE_EXPIRED":                                    "invite has expired",
		"WS_UNAUTHORIZED":                                   "unauthorized",
		"WS_FORBIDDEN":                                      "forbidden",
		"WS_INVALID_JSON":                                   "invalid json",
		"WS_NAME_REQUIRED":                                  "name is required",
		"WS_TX_BEGIN_FAILED":                                "tx begin failed",
		"WS_COMMIT_FAILED":                                  "commit failed",
		"WS_COUNT_FAILED":                                   "failed to count workspaces",
		"WS_LIST_FAILED":                                    "failed to list workspaces",
		"WS_SCAN_FAILED":                                    "failed to scan workspace",
		"WS_RESOLVE_PLAN_FAILED":                            "failed to resolve plan",
		"WS_RESOLVE_SEATS_FAILED":                           "failed to resolve seat_quantity from subscription",
		"WS_SEAT_QUANTITY_INVALID":                          "subscription seat_quantity must be >= 1",
		"WS_PLAN_CREDITS_MISSING":                           "plan_catalog missing credits for active plan",
		"WS_CREATE_FAILED":                                  "failed to create workspace",
		"WS_ADD_OWNER_FAILED":                               "failed to add owner",
		"WS_INIT_QUOTA_FAILED":                              "failed to init quota",
		"WS_MEMBERS_COUNT_FAILED":                           "failed to count members",
		"WS_OWNERS_COUNT_FAILED":                            "failed to count owners",
		"WS_MEMBERS_LIST_FAILED":                            "failed to list members",
		"WS_MEMBER_SCAN_FAILED":                             "failed to scan member",
		"WS_APP_PUBLIC_URL_MISSING":                         "APP_PUBLIC_URL is not configured",
		"WS_ROLE_INVALID":                                   "role must be member or admin",
		"WS_EMAIL_REQUIRED":                                 "valid email is required",
		"WS_INVITER_LOAD_FAILED":                            "failed to load inviter profile",
		"WS_CANNOT_INVITE_SELF":                             "cannot invite yourself",
		"WS_ALREADY_MEMBER":                                 "user is already a member of this workspace",
		"WS_SEAT_CAPACITY_LOAD_FAILED":                      "failed to load seat capacity",
		"WS_NO_SEATS_INVITE":                                "no seats available; upgrade allocated seats before inviting",
		"WS_INVITE_TTL_MISSING":                             "billing_settings.workspace_invite_ttl_hours is missing",
		"WS_INVITE_TOKEN_GEN_FAILED":                        "failed to generate invite token",
		"WS_REVOKE_PRIOR_INVITE_FAILED":                     "failed to revoke prior invite",
		"WS_CREATE_INVITE_FAILED":                           "failed to create invite",
		"WS_INVITE_PATH_MISSING":                            "APP_PATH_INVITE_PREFIX is not configured in site_messages",
		"WS_INVITE_PATH_INVALID":                            "APP_PATH_INVITE_PREFIX must be a safe absolute path",
		"DEFAULT_PLAN_TIER_MISSING":                         "DEFAULT_PLAN_TIER is not configured in site_messages",
		"PADDLE_ON_PAYMENT_FAILURE":                         "prevent_change",
		"PADDLE_ON_PAYMENT_FAILURE_MISSING":                 "PADDLE_ON_PAYMENT_FAILURE is not configured in site_messages",
		"WS_INVITES_COUNT_FAILED":                           "failed to count invites",
		"WS_INVITES_LIST_FAILED":                            "failed to list invites",
		"WS_INVITE_SCAN_FAILED":                             "failed to scan invite",
		"WS_INVITE_LOAD_FAILED":                             "failed to load invite",
		"WS_EMAIL_MISMATCH":                                 "signed-in email does not match this invite",
		"WS_NO_SEATS_JOIN":                                  "no seats available on this workspace",
		"WS_JOIN_FAILED":                                    "failed to join workspace",
		"WS_MARK_ACCEPTED_FAILED":                           "failed to mark invite accepted",
		"WS_ADMINS_CANNOT_REMOVE_OWNERS":                    "admins cannot remove owners",
		"WS_CHECK_OWNERS_FAILED":                            "failed to check owners",
		"WS_CANNOT_REMOVE_LAST_OWNER":                       "cannot remove the last owner",
		"WS_REMOVE_MEMBER_FAILED":                           "failed to remove member",
		"PAGINATION_MAX_LIMIT_INVALID":                      "maxLimit must be >= 1",
		"PAGINATION_SKIP_REQUIRED":                          "query param skip is required",
		"PAGINATION_LIMIT_REQUIRED":                         "query param limit is required",
		"PAGINATION_SKIP_NOT_INT":                           "skip must be an integer",
		"PAGINATION_LIMIT_NOT_INT":                          "limit must be an integer",
		"PAGINATION_SKIP_NEGATIVE":                          "skip must be >= 0",
		"PAGINATION_LIMIT_TOO_SMALL":                        "limit must be >= 1",
		"PAGINATION_LIMIT_TOO_LARGE":                        "limit exceeds maximum allowed",
		"API_KEYS_COUNT_FAILED":                             "failed to count api keys",
		"API_KEY_SCAN_FAILED":                               "failed to scan api key",
		"API_KEY_ID_REQUIRED":                               "key id is required",
		"API_KEY_GENERATE_FAILED":                           "failed to generate api key",
		"COMPRESSION_TIER_INVALID":                          "compression_tier must be fast or deep",
		"DEEP_ENGINE_INVALID":                               "deep_engine must be v1, long, or v2",
		"DEEP_TARGET_TOKEN_RANGE":                           "deep_target_token out of range",
		"ACCOUNT_OPERATION_FAILED":                          "account operation failed",
		"ACCOUNT_DELETE_SUCCESS":                            "Account and personal data were deleted per GDPR right to be forgotten.",
		"LOGIN_ENV_APP_URL_MISSING":                         "NEXT_PUBLIC_APP_URL is not set",
		"LOGIN_OAUTH_EXCHANGE_FAILED":                       "Sign-in could not be completed. Try again.",
		"LOGIN_MISSING_SUPABASE_ENV":                        "Supabase environment is not configured.",
		"LOGIN_DEFAULT_NEXT":                                "/dashboard",
		"LOGIN_DEFAULT_NEXT_MISSING":                        "Sign-in destination is not configured. Set LOGIN_DEFAULT_NEXT in site_messages.",
		"APP_PATH_DASHBOARD":                                "/dashboard",
		"APP_PATH_TEAM":                                     "/dashboard/team",
		"APP_PATH_SETTINGS":                                 "/dashboard/settings",
		"APP_PATH_TRACES":                                   "/dashboard/traces",
		"APP_PATH_RECEIPTS":                                 "/dashboard/receipts",
		"APP_PATH_ENTERPRISE":                               "/dashboard/enterprise",
		"APP_PATH_RECEIPTS_PREFIX":                          "/dashboard/receipts/",
		"APP_PATH_LOGIN":                                    "/login",
		"APP_PATH_PRIVACY":                                  "/privacy",
		"APP_PATH_TERMS":                                    "/terms",
		"APP_PATH_HOME":                                     "/",
		"APP_PATH_DASHBOARD_ENTERPRISE_INQUIRY_PREFIX":      "/dashboard/enterprise?id=",
		"NOTIF_HREF_QUERY_ID":                               "id",
		"DASHBOARD_NAV_TRACES":                              "Traces",
		"DASHBOARD_NAV_RECEIPTS":                            "Receipts",
		"DASHBOARD_NAV_ENTERPRISE":                          "Enterprise",
		"APP_PATH_UPGRADE":                                  "/dashboard",
		"APP_PATH_AUTH_CALLBACK":                            "/auth/callback",
		"APP_PATH_CLI_AUTH":                                 "/cli/auth",
		"APP_PATH_INVITE_PREFIX":                            "/invite/",
		"DEFAULT_PLAN_TIER":                                 "free",
		"AUTH_GOOGLE_OAUTH_ACCESS_TYPE":                     "offline",
		"AUTH_GOOGLE_OAUTH_PROMPT":                          "consent",
		"AUTH_GOOGLE_OAUTH_PARAMS_MISSING":                  "Google OAuth query params are not configured. Set AUTH_GOOGLE_OAUTH_ACCESS_TYPE and AUTH_GOOGLE_OAUTH_PROMPT in site_messages.",
		"API_KEY_PREFIX_ELLIPSIS":                           "...",
		"SITE_HTML_LANG":                                    "en",
		"LOCAL_PREVIEW_TRUNC_SUFFIX":                        "\n… (truncated)",
		"LOCAL_TUI_TRUNC_SUFFIX":                            "…",
		"LOCAL_STATS_URL_FMT":                               "http://127.0.0.1:{port}/v1/stats",
		"CLI_STATS_URL_FMT":                                 "http://127.0.0.1:{port}/v1/stats",
		"CLI_STATS_URL_MISSING":                             "CLI_STATS_URL_FMT must include {port} in site_messages",
		"PLAN_NOT_FOUND":                                    "plan not found",
		"PLAN_INTERVAL_INVALID":                             "interval must be monthly or annual",
		"PLAN_INVALID_JSON_BODY":                            "invalid json body",
		"PLAN_ID_REQUIRED":                                  "plan_id is required",
		"PADDLE_CLIENT_MISSING":                             "paddle api client not configured",
		"PADDLE_API_MISSING":                                "paddle api not configured",
		"PADDLE_SUBSCRIPTION_ID_MISSING":                    "active subscription missing paddle_subscription_id",
		"PADDLE_CUSTOMER_MISSING":                           "no paddle customer for portal",
		"ENTERPRISE_SEATS_RANGE":                            "estimated_seats out of range",
		"ENTERPRISE_MESSAGE_TOO_LONG":                       "Message exceeds the maximum length of 4000 characters.",
		"ENTERPRISE_INQUIRY_RECEIVED":                       "Enterprise inquiry received. Our team will follow up using your account email.",
		"ENTERPRISE_INQUIRY_FAILED":                         "Enterprise inquiry failed.",
		"PLAN_DECIDE_FAILED":                                "could not evaluate plan change",
		"PLAN_CONFLICT":                                     "plan change conflict",
		"PLAN_OPERATION_FAILED":                             "plan operation failed",
		"PADDLE_UPSTREAM_FAILED":                            "paddle upstream request failed",
		"PADDLE_UPGRADE_PREVIEW_FAILED":                     "upgrade preview failed",
		"PADDLE_UPGRADE_FAILED":                             "upgrade failed",
		"RECEIPT_ID_REQUIRED":                               "receiptId required",
		"RECEIPT_OPERATION_FAILED":                          "receipt operation failed",
		"RECEIPT_MONEY_LOCALE_REQUIRED":                     "Receipt money locale is not configured (SITE_HTML_LANG).",
		"RECEIPT_PDF_MONEY_LOCALE_REQUIRED":                 "Receipt PDF money locale is missing; cannot format amounts.",
		"RECEIPT_PDF_CURRENCY_REQUIRED":                     "Receipt currency is missing; cannot render PDF amounts.",
		"RECEIPT_PDF_TITLE_REQUIRED":                        "Receipt document title is missing; cannot render PDF.",
		"PRICE_NOT_CONFIGURED":                              "This plan is not synced to Paddle yet. An operator must sync the plan catalog from the admin dashboard.",
		"PRICE_CATALOG_DRIFT":                               "Checkout blocked: Paddle price no longer matches admin catalog amounts. Re-sync plans to Paddle from the admin dashboard.",
		"EVENTS_COUNT_FAILED":                               "failed to count events",
		"EVENTS_LIST_FAILED":                                "failed to list events",
		"EVENTS_SCAN_FAILED":                                "failed to scan event",
		"EVENTS_MODE_STATUS_REQUIRED":                       "mode and status are required",
		"EVENTS_INVALID_METRICS":                            "invalid metrics",
		"EVENTS_INVALID_COUNTERS":                           "invalid telemetry counters",
		"EVENTS_TAB_ACCEPTED_EXCEEDS":                       "tab_suggestions_accepted cannot exceed shown",
		"EVENTS_STORE_FAILED":                               "failed to store event",
		"EVENTS_AGG_MODELS_FAILED":                          "failed to aggregate models",
		"EVENTS_SCAN_MODELS_FAILED":                         "failed to scan models",
		"EVENTS_AGG_STATUS_FAILED":                          "failed to aggregate status",
		"EVENTS_SCAN_STATUS_FAILED":                         "failed to scan status",
		"EVENTS_AGG_MODES_FAILED":                           "failed to aggregate modes",
		"EVENTS_SCAN_MODES_FAILED":                          "failed to scan modes",
		"EVENTS_AGG_SERIES_FAILED":                          "failed to aggregate series",
		"EVENTS_SCAN_SERIES_FAILED":                         "failed to scan series",
		"CHART_TOP_N_MISSING":                               "billing_settings.chart_top_n is missing",
		"CHART_SERIES_DAYS_MISSING":                         "billing_settings.chart_series_days is missing",
		"CHART_CACHE_TTL_MISSING":                           "billing_settings.chart_cache_ttl_sec is missing",
		"ADMIN_BILLING_CHART_CACHE_TTL":                     "Chart cache TTL (seconds)",
		"ADMIN_COMPLIANCE_FORCE_LOGOUT_TTL":                 "Force-logout marker TTL (seconds)",
		"ADMIN_FORCE_LOGOUT_TTL_MISSING":                    "Set admin_retention_settings.force_logout_ttl_sec before force-logout.",
		"ADMIN_COMPLIANCE_EVENTS_PARTITION_AHEAD":           "Trim events future partitions (months ahead)",
		"ADMIN_COMPLIANCE_EVENTS_PARTITION_ENSURE_SEC":      "Trim events partition ensure interval (seconds)",
		"ADMIN_EVENTS_PARTITION_AHEAD_MISSING":              "Set admin_retention_settings.trim_events_partition_months_ahead before event inserts.",
		"ADMIN_EVENTS_PARTITION_ENSURE_SEC_MISSING":         "Set admin_retention_settings.trim_events_partition_ensure_sec before partition ensure loop.",
		"ADMIN_EVENTS_PARTITION_ENSURE_FAILED":              "Failed to ensure trim_events month partitions.",
		"EVENTS_TOKENS_SEP":                                 " to ",
		"EVENTS_DATE_RANGE_PLACEHOLDER":                     "Filter by date",
		"EVENTS_DATE_RANGE_DESC":                            "Limits traces to the selected UTC calendar days. Clear to show all.",
		"EVENTS_DATE_RANGE_CLEAR":                           "Clear dates",
		"EVENTS_DATE_RANGE_APPLY":                           "Done",
		"EVENTS_DATE_FROM_INVALID":                          "from must be YYYY-MM-DD",
		"EVENTS_DATE_TO_INVALID":                            "to must be YYYY-MM-DD",
		"EVENTS_DATE_RANGE_ORDER":                           "from must be on or before to",
		"EVENTS_SEARCH":                                     "Search traces",
		"EVENTS_SEARCH_DESC":                                "Filters by model, mode, status, or request id as you type.",
		"EVENTS_COL_STATUS":                                 "Status",
		"EVENTS_COL_REQUEST_ID":                             "Request id",
		"EVENTS_COL_VIEW":                                   "View",
		"EVENTS_OPEN_LABEL":                                 "View",
		"EVENTS_PREVIEW_FIELD_DESC":                         "Trace field from this run. Use search to find other traces.",
		"RECEIPTS_PREVIEW_FIELD_DESC":                       "Summary from this receipt. Open the full receipt for line items and PDF.",
		"RECEIPTS_SEARCH":                                   "Search receipts",
		"RECEIPTS_SEARCH_DESC":                              "Filters by invoice id, status, or bill-to as you type.",
		"API_KEY_SEARCH":                                    "Search API keys",
		"API_KEY_SEARCH_DESC":                               "Filters by key prefix, id, or status as you type.",
		"WORKSPACE_SEARCH":                                  "Search workspaces",
		"WORKSPACE_SEARCH_DESC":                             "Filters by name, plan, role, or workspace id as you type.",
		"WORKSPACE_MEMBERS_SEARCH":                          "Search members",
		"WORKSPACE_MEMBERS_SEARCH_DESC":                     "Filters by email, name, or role as you type.",
		"WORKSPACE_MEMBERS_EMPTY":                           "No members match this search.",
		"WORKSPACE_INVITES_SEARCH":                          "Search invites",
		"WORKSPACE_INVITES_SEARCH_DESC":                     "Filters by email, role, or status as you type.",
		"WORKSPACE_NAME_DESC":                               "Shown to members and on invite emails.",
		"WORKSPACE_NAME_LABEL":                              "Workspace name",
		"WORKSPACE_INVITE_EMAIL_DESC":                       "Invitee must sign in with an allowed social provider using this exact email.",
		"WORKSPACE_INVITE_EMAIL_LABEL":                      "Invite email",
		"PREFERENCES_ENGINE_HINT":                           "Default selected: LLMLingua-2 (v2). Pick long (needs a question) or v1 if you prefer. Applies to live proxy Deep and trim compress when Deep Mode is on.",
		"PREFERENCES_DEEP_HINT":                             "On = Deep Mode for live IDE proxy (trim start) and trim compress. Off = Fast Mode only on the live proxy and for compress defaults. Engine and target apply whenever Deep is on. Run trim config sync after saving.",
		"PREFERENCES_TARGET_HINT":                           "Target token budget for Deep Mode (live proxy and trim compress). Only applies when Deep Mode is on.",
		"ENTERPRISE_COMPANY_DESC":                           "Legal or trade name we should use when following up.",
		"ENTERPRISE_MESSAGE_DESC":                           "Include SSO, seat count, timeline, or procurement needs so sales can respond.",
		"AVATAR_PROFILE_UPDATE_FAILED":                      "failed to update profile",
		"AVATAR_UPLOAD_FAILED":                              "avatar upload failed",
		"AVATAR_CLOUDINARY_UNCONFIGURED":                    "Cloudinary is not configured. Set CLOUDINARY_CLOUD_NAME and UPLOAD_PRESET or API key/secret.",
		"AVATAR_HTTP_TIMEOUT_MISSING":                       "CLOUDINARY_HTTP_TIMEOUT_SEC is required when Cloudinary is configured.",
		"AVATAR_NO_URL":                                     "No avatar_url on profile.",
		"AVATAR_ALREADY_CDN":                                "Already on Cloudinary.",
		"BILLING_SYNC_UNAVAILABLE":                          "billing sync unavailable",
		"BILLING_SYNC_UPSTREAM_FAILED":                      "billing sync upstream failed",
		"BILLING_SYNC_NO_CUSTOMER":                          "No Paddle customer on file yet. Complete a checkout first.",
		"TELEMETRY_INVALID_PAYLOAD":                         "invalid payload",
		"TELEMETRY_INVALID_EVENT":                           "invalid event",
		"TELEMETRY_REJECTED":                                "rejected",
		"TABLE_SELECT_ALL":                                  "Select all rows on this page",
		"TABLE_SELECT_ROW":                                  "Select row",
		"TABLE_SELECTED_FMT":                                "{count} selected",
		"TABLE_ROW_ACTIONS":                                 "Row actions",
		"TABLE_BULK_REVOKE":                                 "Revoke selected",
		"TABLE_BULK_REMOVE":                                 "Remove selected",
		"TABLE_BULK_DELETE":                                 "Delete selected",
		"TABLE_CLEAR_SELECTION":                             "Clear selection",
		"WORKSPACE_BULK_DELETE_CONFIRM":                     "Delete the selected workspaces permanently? Members and pending invites are removed. This cannot be undone.",
		"ADMIN_TABLE_SELECT_ALL":                            "Select all rows on this page",
		"ADMIN_TABLE_SELECT_ROW":                            "Select row",
		"ADMIN_TABLE_SELECTED_FMT":                          "{count} selected",
		"ADMIN_TABLE_ROW_ACTIONS":                           "Row actions",
		"ADMIN_TABLE_BULK_REMOVE":                           "Remove selected",
		"ADMIN_TABLE_CLEAR_SELECTION":                       "Clear selection",
		"ADMIN_ACTION_VIEW":                                 "View",
		"ADMIN_FILTER_ACTION":                               "Action",
		"ADMIN_FILTER_ACTOR":                                "Actor user id",
		"ADMIN_FILTER_RESOURCE":                             "Resource type",
		"DASHBOARD_USAGE_TITLE":                             "Your Usage",
		"DASHBOARD_USAGE_SUBTITLE":                          "Your usage per day across this billing period",
		"DASHBOARD_USAGE_GROUP_BY_PREFIX":                   "Group By:",
		"DASHBOARD_USAGE_GROUP_MODEL":                       "Model",
		"DASHBOARD_USAGE_GROUP_MODE":                        "Mode",
		"DASHBOARD_USAGE_Y_AXIS":                            "Cumulative Tokens",
		"DASHBOARD_USAGE_TODAY":                             "Today",
		"DASHBOARD_USAGE_EMPTY":                             "No usage recorded in this period yet. Charts fill in after Trim records proxy events from the CLI or cloud gateway.",
		"DASHBOARD_USAGE_TOOLTIP_BREAKDOWN":                 "Daily breakdown",
		"DASHBOARD_USAGE_TOOLTIP_DAILY_TOTAL":               "Daily total",
		"DASHBOARD_USAGE_TOOLTIP_CUMULATIVE_TOTAL":          "Cumulative total",
		"DASHBOARD_USAGE_TOOLTIP_SHARE_FMT":                 "{pct}%",
		"DASHBOARD_HEATMAP_TITLE":                           "AI Line Edits",
		"DASHBOARD_HEATMAP_SCOPE_ALL":                       "All",
		"DASHBOARD_HEATMAP_SCOPE_TAB":                       "Tab",
		"DASHBOARD_HEATMAP_EMPTY_FMT":                       "{date}\nNo lines edited",
		"DASHBOARD_HEATMAP_VALUE_FMT":                       "{date}\n{count} lines edited",
		"DASHBOARD_HEATMAP_EMPTY_FMT_ALL":                   "{date}\nNo lines edited",
		"DASHBOARD_HEATMAP_VALUE_FMT_ALL":                   "{date}\n{count} lines edited",
		"DASHBOARD_HEATMAP_EMPTY_FMT_TAB":                   "{date}\nNo tab accepts",
		"DASHBOARD_HEATMAP_VALUE_FMT_TAB":                   "{date}\n{count} tab accepts",
		"DASHBOARD_HEATMAP_WD_MON":                          "M",
		"DASHBOARD_HEATMAP_WD_WED":                          "W",
		"DASHBOARD_HEATMAP_WD_FRI":                          "F",
		"DASHBOARD_HEATMAP_STAT_MOST_ACTIVE_MONTH":          "Most Active Month",
		"DASHBOARD_HEATMAP_STAT_MOST_ACTIVE_DAY":            "Most Active Day",
		"DASHBOARD_HEATMAP_STAT_LONGEST_STREAK":             "Longest Streak",
		"DASHBOARD_HEATMAP_STAT_CURRENT_STREAK":             "Current Streak",
		"DASHBOARD_HEATMAP_STREAK_FMT":                      "{count}d",
		"ADMIN_USAGE_TITLE":                                 "Platform Usage",
		"ADMIN_USAGE_SUBTITLE":                              "Platform usage per day across this window",
		"ADMIN_USAGE_GROUP_BY_PREFIX":                       "Group By:",
		"ADMIN_USAGE_GROUP_MODEL":                           "Model",
		"ADMIN_USAGE_GROUP_MODE":                            "Mode",
		"ADMIN_USAGE_Y_AXIS":                                "Cumulative Tokens",
		"ADMIN_USAGE_TODAY":                                 "Today",
		"ADMIN_USAGE_EMPTY":                                 "No platform usage recorded in this window yet. Charts fill in after Trim records proxy events.",
		"ADMIN_USAGE_TOOLTIP_BREAKDOWN":                     "Daily breakdown",
		"ADMIN_USAGE_TOOLTIP_DAILY_TOTAL":                   "Daily total",
		"ADMIN_USAGE_TOOLTIP_CUMULATIVE_TOTAL":              "Cumulative total",
		"ADMIN_USAGE_TOOLTIP_SHARE_FMT":                     "{pct}%",
		"ADMIN_HEATMAP_TITLE":                               "AI Line Edits",
		"ADMIN_HEATMAP_SCOPE_ALL":                           "All",
		"ADMIN_HEATMAP_SCOPE_TAB":                           "Tab",
		"ADMIN_HEATMAP_EMPTY_FMT":                           "{date}\nNo lines edited",
		"ADMIN_HEATMAP_VALUE_FMT":                           "{date}\n{count} lines edited",
		"ADMIN_HEATMAP_EMPTY_FMT_ALL":                       "{date}\nNo lines edited",
		"ADMIN_HEATMAP_VALUE_FMT_ALL":                       "{date}\n{count} lines edited",
		"ADMIN_HEATMAP_EMPTY_FMT_TAB":                       "{date}\nNo tab accepts",
		"ADMIN_HEATMAP_VALUE_FMT_TAB":                       "{date}\n{count} tab accepts",
		"ADMIN_HEATMAP_WD_MON":                              "M",
		"ADMIN_HEATMAP_WD_WED":                              "W",
		"ADMIN_HEATMAP_WD_FRI":                              "F",
		"ADMIN_HEATMAP_STAT_MOST_ACTIVE_MONTH":              "Most Active Month",
		"ADMIN_HEATMAP_STAT_MOST_ACTIVE_DAY":                "Most Active Day",
		"ADMIN_HEATMAP_STAT_LONGEST_STREAK":                 "Longest Streak",
		"ADMIN_HEATMAP_STAT_CURRENT_STREAK":                 "Current Streak",
		"ADMIN_HEATMAP_STREAK_FMT":                          "{count}d",
		"ADMIN_PLAN_ID_DESC":                                "Stable plan identifier used in API, checkout, and Paddle sync. Cannot change after create.",
		"ADMIN_PLAN_NAME_DESC":                              "Customer-facing plan title shown in pricing and the admin plans table.",
		"ADMIN_PLAN_CURRENCY_DESC":                          "Default checkout currency comes from Billing settings; plan amounts use that ISO-4217 code.",
		"ADMIN_PLAN_PRICE_MONTHLY_DESC":                     "Recurring monthly price in major currency units (for example 9.99).",
		"ADMIN_PLAN_PRICE_YEARLY_DESC":                      "Recurring yearly price in major currency units (for example 99.99).",
		"ADMIN_FILTER_SEARCH_DESC":                          "Search by email, id, or name shown in the list.",
		"ADMIN_PLANS_SEARCH_DESC":                           "Search by plan id or display name shown in the list.",
		"ADMIN_PLAN_ACTIVE_DESC":                            "Inactive plans stay in the catalog but are not offered for new checkout.",
		"ADMIN_PLAN_KIND_DESC":                              "Subscription, top-up, or enterprise plan type. Controls which Paddle prices apply.",
		"ADMIN_PLAN_INTERVAL_DESC":                          "Default billing interval label for self-serve checkout when a plan supports both monthly and yearly.",
		"ADMIN_PLAN_DESCRIPTION_DESC":                       "Longer marketing or internal description stored on the plan record.",
		"ADMIN_PLAN_RANK_DESC":                              "Sort weight for plan ordering; higher ranks appear first where plans are listed.",
		"ADMIN_PLAN_SORT_ORDER_DESC":                        "Secondary ordering index when ranks tie.",
		"ADMIN_PLAN_PUBLIC_DESC":                            "Public plans appear in self-serve pricing; private plans are operator-only.",
		"ADMIN_PLAN_FEATURES_DESC":                          "JSON feature flags or bullet list metadata attached to the plan.",
		"ADMIN_PLAN_SYNC_PADDLE_DESC":                       "Per-save action only - not stored on the plan. Check when pushing amounts/currency to Paddle; leave off for catalog-only edits such as Unlimited or Popular. After a successful save the modal closes and this checkbox resets; that is expected.",
		"ADMIN_RBAC_ROLE_NAME_DESC":                         "Human-readable role title shown to operators.",
		"ADMIN_RBAC_ROLE_SLUG_DESC":                         "Stable slug referenced in permissions and audit logs.",
		"ADMIN_RBAC_PERMS_DESC":                             "Toggle platform permissions granted to admins assigned this role.",
		"ADMIN_RBAC_INVITE_EMAIL_DESC":                      "Email of the user to promote; they must already exist in auth.",
		"ADMIN_RBAC_INVITE_ROLE_DESC":                       "Role assigned when the invite is accepted.",
		"ADMIN_CREDITS_USER_ID_DESC":                        "Target account user id (UUID) receiving the manual grant.",
		"ADMIN_CREDITS_AMOUNT_DESC":                         "Number of credits to add to the user balance (positive integer).",
		"ADMIN_CREDITS_REASON_DESC":                         "Audit reason stored on the credit ledger row.",
		"ADMIN_BREAK_GLASS_PERM_DESC":                       "Permission code temporarily elevated for this break-glass session.",
		"ADMIN_BREAK_GLASS_REASON_DESC":                     "Operator justification recorded in audit for the elevation.",
		"ADMIN_DENYLIST_VALUE_DESC":                         "Blocked value: email domain, IP CIDR, or ASN depending on the tab.",
		"ADMIN_DENYLIST_REASON_DESC":                        "Why this entry was added; shown in audit and the denylist table.",
		"ADMIN_DISPUTE_TX_DESC":                             "Paddle transaction id tying the dispute note to a receipt.",
		"ADMIN_DISPUTE_USER_DESC":                           "Optional user id if the dispute is account-specific.",
		"ADMIN_RECEIPT_DISPUTE_REASON_DESC":                 "Free-text dispute or chargeback note for operator tracking.",
		"ADMIN_DISPUTE_STATUS_DESC":                         "Workflow status for the dispute note (open, watching, or closed).",
		"ADMIN_BILLING_CURRENCY_DESC":                       "Default ISO-4217 currency for new Paddle prices and checkout.",
		"ADMIN_BILLING_DEFAULT_PAGE_SIZE_DESC":              "Default page size for admin billing list tables.",
		"ADMIN_BILLING_MAX_PAGE_SIZE_DESC":                  "Hard cap on page size for billing API pagination.",
		"ADMIN_BILLING_SKIP_TO_MAX_DESC":                    "Maximum pages a client may skip forward in one jump.",
		"ADMIN_BILLING_CHART_TOP_N_DESC":                    "Number of top series shown on billing usage charts.",
		"ADMIN_BILLING_CHART_DAYS_DESC":                     "Days of history included in billing chart series.",
		"ADMIN_BILLING_CHART_CACHE_TTL_DESC":                "Seconds to cache computed chart payloads in Redis.",
		"ADMIN_BILLING_DATE_RANGE_MONTHS_DESC":              "Months of history available in billing date-range filters.",
		"ADMIN_BILLING_PLAN_INTERVAL_DESC":                  "Default interval preselected in self-serve plan change flows.",
		"ADMIN_BILLING_DEFAULT_SEATS_DESC":                  "Seat quantity applied when a workspace has no explicit seat count.",
		"ADMIN_BILLING_MIN_SEATS_DESC":                      "Minimum billable seats enforced at checkout and renewal.",
		"ADMIN_BILLING_DEEP_TARGET_DEFAULT_DESC":            "Default Deep Mode token target for new subscriptions.",
		"ADMIN_BILLING_DEEP_TARGET_MIN_DESC":                "Lower bound users may select for Deep token targets.",
		"ADMIN_BILLING_DEEP_TARGET_MAX_DESC":                "Upper bound users may select for Deep token targets.",
		"ADMIN_BILLING_PRORATION_MODE_DESC":                 "How Paddle prorates mid-cycle subscription upgrades.",
		"ADMIN_BILLING_CANCEL_AT_PERIOD_END_DESC":           "Allow customers to schedule cancellation at period end.",
		"ADMIN_BILLING_ALLOW_DOWNGRADES_DESC":               "Allow self-serve plan downgrades when policy permits.",
		"ADMIN_BILLING_MONTHLY_TO_ANNUAL_DESC":              "Treat monthly-to-annual interval changes as upgrades.",
		"ADMIN_COMPLIANCE_EVENTS_TTL_DESC":                  "Days to retain trim_events rows before automated purge.",
		"ADMIN_COMPLIANCE_AUDIT_TTL_DESC":                   "Days to retain platform admin audit log entries.",
		"ADMIN_COMPLIANCE_AUDIT_EXPORT_MAX_DESC":            "Maximum audit rows returned in a single export job.",
		"ADMIN_COMPLIANCE_ATTEST_LIMIT_DESC":                "Maximum access-review attestation rows kept in history.",
		"ADMIN_COMPLIANCE_BREAK_GLASS_TTL_DESC":             "Minutes a break-glass elevation remains valid.",
		"ADMIN_COMPLIANCE_FORCE_LOGOUT_TTL_DESC":            "Seconds the force-logout marker blocks new sessions.",
		"ADMIN_COMPLIANCE_EVENTS_PARTITION_AHEAD_DESC":      "How many future months of trim_events partitions to pre-create.",
		"ADMIN_COMPLIANCE_EVENTS_PARTITION_ENSURE_SEC_DESC": "Background interval that ensures trim_events partitions exist.",
		"ADMIN_ACCESS_REVIEW_PERIOD_DESC":                   "Label for the access review period being attested (for example Q1 2026).",
		"ADMIN_ACCESS_REVIEW_NOTES_DESC":                    "Optional notes stored on the attestation record.",
		"ADMIN_COMPLIANCE_ATTEST_REASON_DESC":               "Audit reason required when filing an access review attestation.",
		"ADMIN_PRODUCT_MODE_DESC":                           "Default compression tier for new accounts when unset in preferences (deep = Deep Mode on).",
		"ADMIN_PRODUCT_ENGINE_DESC":                         "Default Deep engine for new accounts (v1, long, or v2). LLMLingua-2 (v2) is recommended.",
		"ADMIN_PRODUCT_TREESITTER_DESC":                     "Require Tree-sitter parsing for supported languages in the compress pipeline.",
		"ADMIN_PRODUCT_DEEP_ATTACH_DESC":                    "Default whether Deep Mode attaches full file context on first use.",
		"ADMIN_PRODUCT_MODEL_ROUTING_DESC":                  "Enable routing compress requests across configured model providers.",
		"ADMIN_PRODUCT_RATE_IP_DESC":                        "Maximum API requests per minute allowed per client IP.",
		"ADMIN_PRODUCT_RATE_USER_DESC":                      "Maximum API requests per minute allowed per authenticated user.",
		"ADMIN_PRODUCT_POW_DESC":                            "Proof-of-work difficulty for anonymous or abusive traffic (0 disables).",
		"ADMIN_PRODUCT_MAX_HW_DESC":                         "Maximum distinct accounts allowed per hardware fingerprint.",
		"ADMIN_PRODUCT_MAX_JA4_DESC":                        "Maximum distinct accounts allowed per JA4 TLS fingerprint.",
		"ADMIN_PRODUCT_CF_THREAT_DESC":                      "Minimum Cloudflare threat score to treat traffic as suspicious.",
		"ADMIN_PRODUCT_CLI_NOTICE_DESC":                     "Message shown to CLI clients below min_cli_version.",
		"ADMIN_PRODUCT_MIN_CLI_DESC":                        "Semver floor; older CLI builds receive the force-upgrade notice.",
		"ADMIN_PRODUCT_CHURN_HIGH_USAGE_DESC":               "Usage ratio threshold marking high-risk churn segments.",
		"ADMIN_PRODUCT_CHURN_MED_USAGE_DESC":                "Usage ratio threshold for medium churn risk.",
		"ADMIN_PRODUCT_CHURN_HIGH_IDLE_DESC":                "Idle days threshold for high idle churn risk.",
		"ADMIN_PRODUCT_CHURN_LOW_IDLE_DESC":                 "Idle days threshold for low idle churn risk.",
		"ADMIN_PRODUCT_FAST_BALANCED_MIN_DESC":              "Minimum source lines before balanced Fast mode applies strong trims.",
		"ADMIN_PRODUCT_FAST_AGGRESSIVE_MIN_DESC":            "Minimum source lines before aggressive Fast mode applies strong trims.",
		"ADMIN_PRODUCT_FAST_MILD_MIN_DESC":                  "Minimum source lines before mild Fast mode applies log-focused trims.",
		"ADMIN_PRODUCT_QUEUE_WARN_DEPTH_DESC":               "Paddle webhook queue depth that triggers operator alerts.",
		"ADMIN_USER_QUOTA_LIMIT_DESC":                       "Monthly credit allowance cap for this account (user_quotas.monthly_credit_limit).",
		"ADMIN_USER_QUOTA_TOPUP_DESC":                       "Purchased top-up credits outside the monthly allowance (user_quotas.purchased_topup_credits).",
		"ADMIN_USER_CREDIT_AMOUNT_DESC":                     "Number of credits to add to the user balance (positive integer).",
		"ADMIN_CHROME_CODE_DESC":                            "Site message code in site_messages; identifies the row updated by the admin PATCH route.",
		"ADMIN_CHROME_BODY_DESC":                            "String body stored in site_messages and served to clients after cache reload.",
		"ADMIN_LEGAL_BODY_DESC":                             "Legal section content stored on the legal_docs row for this id.",
		"ADMIN_ENTERPRISE_MESSAGE_DESC":                     "Internal contract or sales notes stored on the enterprise inquiry (contract_notes).",
		"ADMIN_ENTERPRISE_SEATS_DESC":                       "Seat quantity offered in the proposal (offered_seat_quantity).",
		"ADMIN_EMAIL_TEMPLATE_BODY_DESC":                    "Email template body text for this template code in site_messages.",
	}
	for code, body := range extras {
		put(code, body)
	}

	if err := insertMissingSiteMessages(ctx, db, desired); err != nil {
		return err
	}
	return reloadSiteMessageCache(ctx, db)
}

// insertMissingSiteMessages inserts only codes not already in Postgres.
// Steady-state boot: 1 SELECT + 0 INSERT batches. Cold boot: 1 SELECT + chunked unnest INSERT.
func insertMissingSiteMessages(ctx context.Context, db *pgxpool.Pool, desired map[string]string) error {
	if len(desired) == 0 {
		return nil
	}
	codes := make([]string, 0, len(desired))
	for code := range desired {
		codes = append(codes, code)
	}
	rows, err := db.Query(ctx, `select code from public.site_messages where code = any($1::text[])`, codes)
	if err != nil {
		return fmt.Errorf("site_messages existing codes: %w", err)
	}
	existing := make(map[string]struct{}, len(codes))
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			rows.Close()
			return fmt.Errorf("site_messages existing codes scan: %w", err)
		}
		existing[code] = struct{}{}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("site_messages existing codes rows: %w", err)
	}

	missCodes := make([]string, 0, 64)
	missBodies := make([]string, 0, 64)
	for code, body := range desired {
		if _, ok := existing[code]; ok {
			continue
		}
		missCodes = append(missCodes, code)
		missBodies = append(missBodies, body)
	}
	if len(missCodes) == 0 {
		log.Printf("site_messages: seed ok (%d codes already present, 0 inserts)", len(desired))
		return nil
	}

	const chunk = 400
	for i := 0; i < len(missCodes); i += chunk {
		j := i + chunk
		if j > len(missCodes) {
			j = len(missCodes)
		}
		if _, err := db.Exec(ctx, `
			insert into public.site_messages (code, body)
			select c, b from unnest($1::text[], $2::text[]) as t(c, b)
			on conflict (code) do nothing
		`, missCodes[i:j], missBodies[i:j]); err != nil {
			return fmt.Errorf("seed site_messages batch: %w", err)
		}
	}
	log.Printf("site_messages: seeded %d missing of %d codes", len(missCodes), len(desired))
	return nil
}

const siteMessagesReloadChannel = "trim:site_messages:reload"

// ReloadSiteMessages refreshes the in-process site_messages cache from Postgres
// after admin chrome edits so live /me and billing chrome strings update without
// an API process restart.
func ReloadSiteMessages(ctx context.Context, db *pgxpool.Pool) error {
	return reloadSiteMessageCache(ctx, db)
}

// ReloadAndBroadcastSiteMessages reloads local cache then publishes so other API replicas reload.
func ReloadAndBroadcastSiteMessages(ctx context.Context, db *pgxpool.Pool, rdb *redis.Client) error {
	if err := ReloadSiteMessages(ctx, db); err != nil {
		return err
	}
	if rdb == nil {
		return nil
	}
	return rdb.Publish(ctx, siteMessagesReloadChannel, "1").Err()
}

// SubscribeSiteMessagesReload listens for chrome reload broadcasts (multi-replica safe)
// and periodically reloads from Postgres so a missed pub/sub message cannot starve replicas.
func SubscribeSiteMessagesReload(ctx context.Context, db *pgxpool.Pool, rdb *redis.Client, reloadSec int) {
	if db == nil {
		return
	}
	if reloadSec > 0 {
		go func() {
			t := time.NewTicker(time.Duration(reloadSec) * time.Second)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					if err := ReloadSiteMessages(ctx, db); err != nil {
						log.Printf("site_messages: periodic reload failed: %v", err)
					}
				}
			}
		}()
	}
	if rdb == nil {
		return
	}
	sub := rdb.Subscribe(ctx, siteMessagesReloadChannel)
	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			_ = sub.Close()
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if msg == nil {
				continue
			}
			if err := ReloadSiteMessages(ctx, db); err != nil {
				log.Printf("site_messages: reload after pub/sub failed: %v", err)
			}
		}
	}
}

func parseSeedCodes(raw string) []string {
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	seen := map[string]struct{}{}
	for _, line := range lines {
		code := strings.TrimSpace(line)
		if code == "" || strings.HasPrefix(code, "#") {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out
}

func reloadSiteMessageCache(ctx context.Context, db *pgxpool.Pool) error {
	rows, err := db.Query(ctx, `select code, body from public.site_messages`)
	if err != nil {
		return err
	}
	defer rows.Close()
	next := make(map[string]string)
	for rows.Next() {
		var code, body string
		if err := rows.Scan(&code, &body); err != nil {
			return err
		}
		next[code] = body
	}
	if err := rows.Err(); err != nil {
		return err
	}
	siteMsgMu.Lock()
	siteMsgCache = next
	siteMsgWarmed = true
	siteMsgMu.Unlock()
	return nil
}

func cachedSiteMessage(code string) (string, bool) {
	siteMsgMu.RLock()
	defer siteMsgMu.RUnlock()
	if !siteMsgWarmed {
		return "", false
	}
	v, ok := siteMsgCache[code]
	return v, ok
}

// FormatUTCDateTime formats an instant as UTC RFC3339-style display (YYYY-MM-DD HH:MM UTC).
// Empty when t is zero. Used so clients never invent toLocaleString.
func FormatUTCDateTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02 15:04 UTC")
}

// FormatUTCDate formats a calendar day in UTC (YYYY-MM-DD).
func FormatUTCDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02")
}

// FormatUTCDateTimeFromRFC3339 parses common DB/API timestamp strings and formats them.
func FormatUTCDateTimeFromRFC3339(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999-07",
		"2006-01-02 15:04:05-07",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return FormatUTCDateTime(t)
		}
	}
	return ""
}

// FormatUTCDateFromRFC3339 parses timestamps to a UTC calendar day label.
func FormatUTCDateFromRFC3339(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999-07",
		"2006-01-02 15:04:05-07",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return FormatUTCDate(t)
		}
	}
	if len(raw) >= 10 {
		return raw[:10]
	}
	return raw
}

// FormatUTCChartDay labels a YYYY-MM-DD series key for charts (no client locale invent).
func FormatUTCChartDay(day string) string {
	day = strings.TrimSpace(day)
	if day == "" {
		return ""
	}
	if t, err := time.Parse("2006-01-02", day); err == nil {
		return t.UTC().Format("Jan 02")
	}
	return ""
}

// FormatUTCChartDayLong labels a YYYY-MM-DD key for heatmap popovers (no client locale invent).
func FormatUTCChartDayLong(day string) string {
	day = strings.TrimSpace(day)
	if day == "" {
		return ""
	}
	if t, err := time.Parse("2006-01-02", day); err == nil {
		return t.UTC().Format("Monday, January 2, 2006")
	}
	return ""
}

// FormatUTCChartDayMedium labels a YYYY-MM-DD key as "Jan 2, 2006" for heatmap stats.
func FormatUTCChartDayMedium(day string) string {
	day = strings.TrimSpace(day)
	if day == "" {
		return ""
	}
	if t, err := time.Parse("2006-01-02", day); err == nil {
		return t.UTC().Format("Jan 2, 2006")
	}
	return ""
}

// FormatUTCChartMonthLetter is the first letter of the English month abbr for calendar headers.
func FormatUTCChartMonthLetter(day string) string {
	day = strings.TrimSpace(day)
	if day == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return ""
	}
	abbr := t.UTC().Format("Jan")
	if abbr == "" {
		return ""
	}
	return abbr[:1]
}

// WeekdayMon0 returns Monday=0 .. Sunday=6 for a YYYY-MM-DD key (-1 if invalid).
func WeekdayMon0(day string) int {
	day = strings.TrimSpace(day)
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return -1
	}
	// time.Weekday: Sunday=0 .. Saturday=6 → Monday=0 .. Sunday=6
	return (int(t.UTC().Weekday()) + 6) % 7
}
