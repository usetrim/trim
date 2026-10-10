package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/usetrim/trim/server/internal/billing/paddleapi"
	"github.com/usetrim/trim/server/internal/middleware"
	"github.com/usetrim/trim/server/internal/notifications"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

type subscriptionData struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Status     string `json:"status"`
	CustomData struct {
		UserID    string `json:"userId"`
		PlanID    string `json:"plan_id"`
		Interval  string `json:"interval"`
		InquiryID string `json:"inquiry_id"`
	} `json:"custom_data"`
	Items []struct {
		Quantity int `json:"quantity"`
		Price    struct {
			ID string `json:"id"`
		} `json:"price"`
	} `json:"items"`
	CurrentBillingPeriod struct {
		StartsAt string `json:"starts_at"`
		EndsAt   string `json:"ends_at"`
	} `json:"current_billing_period"`
}

type transactionData struct {
	ID             string  `json:"id"`
	CustomerID     string  `json:"customer_id"`
	Status         string  `json:"status"`
	CurrencyCode   string  `json:"currency_code"`
	InvoiceNumber  *string `json:"invoice_number"`
	InvoiceID      *string `json:"invoice_id"`
	SubscriptionID *string `json:"subscription_id"`
	CustomData     struct {
		UserID    string `json:"userId"`
		PlanID    string `json:"plan_id"`
		InquiryID string `json:"inquiry_id"`
	} `json:"custom_data"`
	Details struct {
		Totals struct {
			Subtotal string `json:"subtotal"`
			Tax      string `json:"tax"`
			Total    string `json:"total"`
		} `json:"totals"`
		TaxRatesUsed []struct {
			TaxRate string `json:"tax_rate"`
		} `json:"tax_rates_used"`
		// LineItems carry priced totals. transaction.items usually omit totals.
		LineItems []struct {
			PriceID  string `json:"price_id"`
			Quantity int    `json:"quantity"`
			TaxRate  string `json:"tax_rate"`
			Totals   struct {
				Subtotal string `json:"subtotal"`
				Tax      string `json:"tax"`
				Total    string `json:"total"`
			} `json:"totals"`
			UnitTotals struct {
				Subtotal string `json:"subtotal"`
				Tax      string `json:"tax"`
				Total    string `json:"total"`
			} `json:"unit_totals"`
			Product struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				SKU  string `json:"sku"`
			} `json:"product"`
		} `json:"line_items"`
	} `json:"details"`
	BillingPeriod *struct {
		StartsAt string `json:"starts_at"`
		EndsAt   string `json:"ends_at"`
	} `json:"billing_period"`
	Address *struct {
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		FirstLine   string `json:"first_line"`
		SecondLine  string `json:"second_line"`
		City        string `json:"city"`
		Region      string `json:"region"`
		PostalCode  string `json:"postal_code"`
		CountryCode string `json:"country_code"`
	} `json:"address"`
	Customer *struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"customer"`
	BillingDetails *struct {
		TaxIdentifier string `json:"tax_identifier"`
	} `json:"billing_details"`
	// Items are catalog selections on the transaction (no priced totals).
	// Priced invoice lines come from details.line_items only.
	Items []struct {
		Price struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Product     struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				SKU  string `json:"sku"`
			} `json:"product"`
		} `json:"price"`
		Quantity int `json:"quantity"`
	} `json:"items"`
	Payments []struct {
		Status        string `json:"status"`
		MethodDetails *struct {
			Type string `json:"type"`
			Card *struct {
				Type           string `json:"type"`
				Last4          string `json:"last4"`
				CardholderName string `json:"cardholder_name"`
			} `json:"card"`
		} `json:"method_details"`
	} `json:"payments"`
}

type Handler struct {
	DB                *pgxpool.Pool
	Redis             *redis.Client
	Secret            string
	Paddle            *paddleapi.Client
	ProcessTimeoutSec int
	QueueSize         int
	Workers           int
}

type rawEvent struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	Body      []byte `json:"body"`
}

const paddleWebhookQueueKey = "trim:paddle:webhook:queue"

func NewHandler(db *pgxpool.Pool, rdb *redis.Client, secret string, paddle *paddleapi.Client, processTimeoutSec, queueSize, workers int) *Handler {
	h := &Handler{
		DB:                db,
		Redis:             rdb,
		Secret:            secret,
		Paddle:            paddle,
		ProcessTimeoutSec: processTimeoutSec,
		QueueSize:         queueSize,
		Workers:           workers,
	}
	for i := 0; i < workers; i++ {
		go h.worker()
	}
	return h
}

func (h *Handler) invalidateUserQuotaCache(ctx context.Context, userID string) {
	if h.Redis == nil || userID == "" {
		return
	}
	_ = h.Redis.Del(ctx, "user_quota:"+userID).Err()
}

func (h *Handler) invalidateOwnedWorkspaceQuotaCaches(ctx context.Context, userID string) {
	if h.Redis == nil || h.DB == nil || userID == "" {
		return
	}
	rows, err := h.DB.Query(ctx, `
		select id::text from public.workspaces where owner_id = $1::uuid
	`, userID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil || id == "" {
			continue
		}
		_ = h.Redis.Del(ctx, "workspace_quota:"+id).Err()
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, subscriptions.MessageForCode("BODY_READ_FAILED"), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(h.Secret) == "" {
		http.Error(w, subscriptions.MessageForCode("PADDLE_WEBHOOK_SECRET_MISSING"), http.StatusInternalServerError)
		return
	}
	if !verifyPaddleSignature(r.Header.Get("Paddle-Signature"), body, h.Secret) {
		http.Error(w, subscriptions.MessageForCode("PADDLE_SIGNATURE_INVALID"), http.StatusUnauthorized)
		return
	}

	var envelope struct {
		EventID   string `json:"event_id"`
		EventType string `json:"event_type"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		http.Error(w, subscriptions.MessageForCode("PADDLE_PAYLOAD_INVALID"), http.StatusBadRequest)
		return
	}

	if h.Redis == nil {
		http.Error(w, subscriptions.MessageForCode("PADDLE_QUEUE_BUSY"), http.StatusServiceUnavailable)
		return
	}
	ctx := r.Context()
	depth, err := h.Redis.LLen(ctx, paddleWebhookQueueKey).Result()
	if err != nil {
		http.Error(w, subscriptions.MessageForCode("PADDLE_QUEUE_BUSY"), http.StatusServiceUnavailable)
		return
	}
	if h.QueueSize > 0 && depth >= int64(h.QueueSize) {
		http.Error(w, subscriptions.MessageForCode("PADDLE_QUEUE_BUSY"), http.StatusServiceUnavailable)
		return
	}
	payload, err := json.Marshal(rawEvent{EventID: envelope.EventID, EventType: envelope.EventType, Body: body})
	if err != nil {
		http.Error(w, subscriptions.MessageForCode("PADDLE_PAYLOAD_INVALID"), http.StatusBadRequest)
		return
	}
	if err := h.Redis.LPush(ctx, paddleWebhookQueueKey, payload).Err(); err != nil {
		http.Error(w, subscriptions.MessageForCode("PADDLE_QUEUE_BUSY"), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (h *Handler) worker() {
	for {
		if h.Redis == nil {
			time.Sleep(time.Second)
			continue
		}
		sec := h.ProcessTimeoutSec
		if sec < 1 {
			sec = 1
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sec)*time.Second)
		res, err := h.Redis.BRPop(ctx, time.Duration(sec)*time.Second, paddleWebhookQueueKey).Result()
		cancel()
		if err != nil || len(res) < 2 {
			continue
		}
		var event rawEvent
		if json.Unmarshal([]byte(res[1]), &event) != nil {
			continue
		}
		h.process(event)
	}
}

// WebhookQueueDepth returns Redis Paddle webhook backlog (0 if none).
func (h *Handler) WebhookQueueDepth() int {
	if h == nil || h.Redis == nil {
		return 0
	}
	n, err := h.Redis.LLen(context.Background(), paddleWebhookQueueKey).Result()
	if err != nil {
		return 0
	}
	return int(n)
}

func (h *Handler) process(event rawEvent) {
	if h.DB == nil {
		return
	}
	sec := h.ProcessTimeoutSec
	if sec < 1 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sec)*time.Second)
	defer cancel()

	claimed := false
	if event.EventID != "" {
		tag, err := h.DB.Exec(ctx, `
			insert into public.paddle_webhook_events (event_id, event_type, payload, process_status)
			values ($1, $2, $3::jsonb, 'claimed')
			on conflict (event_id) do nothing
		`, event.EventID, event.EventType, string(event.Body))
		if err != nil {
			log.Printf("paddle: webhook idempotency insert failed: %v", err)
			return
		}
		if tag.RowsAffected() == 0 {
			log.Printf("paddle: skipping duplicate event_id=%s type=%s", event.EventID, event.EventType)
			return
		}
		claimed = true
	}

	ok := false
	defer func() {
		if !claimed || event.EventID == "" {
			return
		}
		if ok {
			_, _ = h.DB.Exec(context.Background(), `
				update public.paddle_webhook_events
				set process_status = 'ok', last_error = null
				where event_id = $1
			`, event.EventID)
			return
		}
		// Keep the row for operator inspector. Replay deletes so Paddle can redeliver.
		_, _ = h.DB.Exec(context.Background(), `
			update public.paddle_webhook_events
			set process_status = 'failed', last_error = 'processing failed'
			where event_id = $1
		`, event.EventID)
	}()

	switch event.EventType {
	case "subscription.created", "subscription.updated", "subscription.activated":
		h.handleSubscription(ctx, event)
		ok = true
	case "subscription.canceled", "subscription.past_due":
		h.handleSubscriptionCancel(ctx, event)
		ok = true
	case "transaction.completed":
		h.handleTransactionCompleted(ctx, event)
		ok = true
	case "transaction.updated":
		h.handleTransactionUpdated(ctx, event)
		ok = true
	default:
		ok = true
	}
}

func (h *Handler) handleSubscription(ctx context.Context, event rawEvent) {
	var envelope struct {
		Data subscriptionData `json:"data"`
	}
	if err := json.Unmarshal(event.Body, &envelope); err != nil {
		log.Printf("paddle: subscription parse error: %v", err)
		return
	}
	data := envelope.Data
	userID := data.CustomData.UserID
	if userID == "" {
		log.Printf("paddle: subscription %s missing custom_data.userId", data.ID)
		return
	}

	priceID := ""
	quantity := 0
	if len(data.Items) > 0 {
		priceID = data.Items[0].Price.ID
		quantity = data.Items[0].Quantity
	}
	if quantity < 1 {
		log.Printf("paddle: subscription %s missing items[0].quantity >= 1", data.ID)
		return
	}

	tier, baseLimit, err := h.resolvePlanFromPrice(ctx, priceID, data.CustomData.PlanID)
	if err != nil {
		log.Printf("paddle: resolve plan for price %s: %v", priceID, err)
		return
	}

	var perSeat bool
	var newRank int
	var planKind string
	err = h.DB.QueryRow(ctx, `
		select coalesce(per_seat, false), plan_rank, plan_kind
		from public.plan_catalog where id = $1 and is_active = true
	`, tier).Scan(&perSeat, &newRank, &planKind)
	if err != nil {
		log.Printf("paddle: plan_rank missing for tier %s: %v", tier, err)
		return
	}

	limit := baseLimit
	if perSeat && quantity > 1 {
		limit = baseLimit * quantity
	}

	interval := data.CustomData.Interval
	if interval == "" {
		interval = intervalFromPrice(ctx, h.DB, priceID)
	}

	// Protect entitlements: same upgrade-only policy as DecideChange / LoadPolicy
	// (LoadPolicy coerces allow_downgrades to false; never trust raw billing_settings alone).
	policy, policyErr := subscriptions.NewService(h.DB, nil).LoadPolicy(ctx)
	allowDowngrades := false
	if policyErr == nil {
		allowDowngrades = policy.AllowDowngrades
	}

	var currentRank int
	rankErr := h.DB.QueryRow(ctx, `
		select pc.plan_rank
		from public.subscriptions s
		inner join public.plan_catalog pc on pc.id = s.plan_tier and pc.is_active = true
		where s.user_id = $1::uuid
		  and s.status in ('active', 'trialing', 'past_due')
		  and coalesce(s.expires_at, s.current_period_end) > now()
		order by pc.plan_rank desc nulls last
		limit 1
	`, userID).Scan(&currentRank)
	if rankErr != nil {
		currentRank = 0
	}

	if !allowDowngrades && currentRank > 0 && newRank < currentRank {
		log.Printf(
			"paddle: blocked entitlement downgrade user=%s sub=%s from_rank=%d to_rank=%d tier=%s (allow_downgrades=false)",
			userID, data.ID, currentRank, newRank, tier,
		)
		// Keep paddle subscription row status/period in sync for ops, but do not lower quotas.
		_, _ = h.DB.Exec(ctx, `
			update public.subscriptions
			set status = $2,
			    current_period_start = coalesce(nullif($3, '')::timestamptz, current_period_start),
			    current_period_end = coalesce(nullif($4, '')::timestamptz, current_period_end),
			    expires_at = coalesce(nullif($4, '')::timestamptz, expires_at),
			    updated_at = now()
			where paddle_subscription_id = $1
		`, data.ID, data.Status,
			data.CurrentBillingPeriod.StartsAt,
			data.CurrentBillingPeriod.EndsAt,
		)
		return
	}

	periodEnd := strings.TrimSpace(data.CurrentBillingPeriod.EndsAt)
	if periodEnd == "" {
		// Fail closed: never invent a billing period. Sync status only; skip quota grant.
		log.Printf(
			"paddle: missing current_billing_period.ends_at for sub=%s user=%s; skipping entitlement grant",
			data.ID, userID,
		)
		_, _ = h.DB.Exec(ctx, `
			insert into public.subscriptions (
				user_id, paddle_customer_id, paddle_subscription_id, status, price_id, plan_tier,
				billing_interval, seat_quantity, current_period_start, current_period_end, expires_at
			) values (
				$1,$2,$3,$4,$5,$6,$7,$8,
				coalesce(nullif($9, '')::timestamptz, now()),
				null,
				null
			)
			on conflict (paddle_subscription_id) do update set
				status = excluded.status,
				price_id = excluded.price_id,
				plan_tier = excluded.plan_tier,
				billing_interval = excluded.billing_interval,
				seat_quantity = excluded.seat_quantity,
				current_period_start = excluded.current_period_start,
				updated_at = now()
		`, userID, data.CustomerID, data.ID, data.Status, priceID, tier, interval, quantity,
			data.CurrentBillingPeriod.StartsAt,
		)
		return
	}

	// Detect billing-period roll before upsert so we can burn overshoot into top-ups and reset used.
	var periodRolled bool
	_ = h.DB.QueryRow(ctx, `
		select coalesce(
			nullif($2, '')::timestamptz > current_period_start,
			false
		)
		from public.subscriptions
		where paddle_subscription_id = $1
	`, data.ID, data.CurrentBillingPeriod.StartsAt).Scan(&periodRolled)

	_, err = h.DB.Exec(ctx, `
		insert into public.subscriptions (
			user_id, paddle_customer_id, paddle_subscription_id, status, price_id, plan_tier,
			billing_interval, seat_quantity, current_period_start, current_period_end, expires_at
		) values (
			$1,$2,$3,$4,$5,$6,$7,$8,
			coalesce(nullif($9, '')::timestamptz, now()),
			$10::timestamptz,
			$10::timestamptz
		)
		on conflict (paddle_subscription_id) do update set
			status = excluded.status,
			price_id = excluded.price_id,
			plan_tier = excluded.plan_tier,
			billing_interval = excluded.billing_interval,
			seat_quantity = excluded.seat_quantity,
			current_period_start = excluded.current_period_start,
			current_period_end = excluded.current_period_end,
			expires_at = excluded.expires_at,
			updated_at = now()
	`, userID, data.CustomerID, data.ID, data.Status, priceID, tier, interval, quantity,
		data.CurrentBillingPeriod.StartsAt,
		periodEnd,
	)
	if err != nil {
		log.Printf("paddle: upsert subscription: %v", err)
		return
	}

	if periodRolled {
		_, err = h.DB.Exec(ctx, `
			update public.user_quotas
			set plan_tier = $2,
			    monthly_credit_limit = $3,
			    purchased_topup_credits = greatest(0, purchased_topup_credits - greatest(0, monthly_credit_used - monthly_credit_limit)),
			    monthly_credit_used = 0,
			    updated_at = now()
			where user_id = $1
		`, userID, tier, limit)
	} else {
		_, err = h.DB.Exec(ctx, `
			update public.user_quotas
			set plan_tier = $2, monthly_credit_limit = $3, updated_at = now()
			where user_id = $1
		`, userID, tier, limit)
	}
	if err != nil {
		log.Printf("paddle: update quotas: %v", err)
	}
	h.invalidateUserQuotaCache(ctx, userID)

	// Seat / enterprise workspace pools: catalog-driven (per_seat or enterprise kind), never invent plan ids.
	if perSeat || planKind == "enterprise" {
		if _, err := h.DB.Exec(ctx, `
			update public.workspaces
			set plan_tier = $2, allocated_seats = $3, updated_at = now()
			where owner_id = $1::uuid
		`, userID, tier, quantity); err != nil {
			log.Printf("paddle: update owned workspaces tier: %v", err)
		}
		if periodRolled {
			if _, err := h.DB.Exec(ctx, `
				update public.workspace_quotas wq
				set monthly_shared_credits = $2,
				    credits_consumed = 0,
				    updated_at = now()
				from public.workspaces w
				where w.id = wq.workspace_id and w.owner_id = $1::uuid
			`, userID, limit); err != nil {
				log.Printf("paddle: sync workspace_quotas (period roll): %v", err)
			}
		} else if _, err := h.DB.Exec(ctx, `
			update public.workspace_quotas wq
			set monthly_shared_credits = $2, updated_at = now()
			from public.workspaces w
			where w.id = wq.workspace_id and w.owner_id = $1::uuid
		`, userID, limit); err != nil {
			log.Printf("paddle: sync workspace_quotas: %v", err)
		}
	} else if periodRolled {
		if _, err := h.DB.Exec(ctx, `
			update public.workspace_quotas wq
			set credits_consumed = 0, updated_at = now()
			from public.workspaces w
			where w.id = wq.workspace_id and w.owner_id = $1::uuid
		`, userID); err != nil {
			log.Printf("paddle: reset workspace credits_consumed on period roll: %v", err)
		}
	}
	h.invalidateOwnedWorkspaceQuotaCaches(ctx, userID)

	h.fulfillEnterpriseInquiry(ctx, userID, data.CustomData.InquiryID, data.ID, "", quantity)

	_ = notifications.Insert(ctx, h.DB, notifications.InsertOpts{
		RecipientID: userID,
		Audience:    notifications.AudienceUser,
		KindCode:    notifications.KindUserSubscriptionActive,
		BodyArgs:    []string{tier, data.Status},
		DedupeKey:   "sub_active:" + data.ID + ":" + data.Status,
	})
}

func (h *Handler) handleSubscriptionCancel(ctx context.Context, event rawEvent) {
	var envelope struct {
		Data subscriptionData `json:"data"`
	}
	if err := json.Unmarshal(event.Body, &envelope); err != nil {
		return
	}
	data := envelope.Data
	userID := data.CustomData.UserID

	endsAt := data.CurrentBillingPeriod.EndsAt
	_, err := h.DB.Exec(ctx, `
		update public.subscriptions
		set status = $2,
		    canceled_at = now(),
		    expires_at = coalesce(nullif($3, '')::timestamptz, expires_at, current_period_end),
		    current_period_end = coalesce(nullif($3, '')::timestamptz, current_period_end),
		    updated_at = now()
		where paddle_subscription_id = $1
	`, data.ID, data.Status, endsAt)
	if err != nil {
		log.Printf("paddle: cancel subscription update: %v", err)
		return
	}

	if userID == "" {
		return
	}

	var stillActive bool
	_ = h.DB.QueryRow(ctx, `
		select exists(
			select 1 from public.subscriptions
			where user_id = $1
			  and status in ('active', 'trialing', 'past_due')
			  and coalesce(expires_at, current_period_end) > now()
		)
	`, userID).Scan(&stillActive)
	if stillActive {
		return
	}

	var periodOpen bool
	_ = h.DB.QueryRow(ctx, `
		select exists(
			select 1 from public.subscriptions
			where paddle_subscription_id = $1
			  and coalesce(expires_at, current_period_end) > now()
		)
	`, data.ID).Scan(&periodOpen)
	if periodOpen {
		return
	}

	var freeLimit int
	defaultTier := strings.TrimSpace(subscriptions.MessageForCode("DEFAULT_PLAN_TIER"))
	if defaultTier == "" {
		log.Printf("paddle: DEFAULT_PLAN_TIER missing in site_messages")
		return
	}
	err = h.DB.QueryRow(ctx, `
		select credits_monthly from public.plan_catalog where id = $1 and is_active = true
	`, defaultTier).Scan(&freeLimit)
	if err != nil {
		log.Printf("paddle: default plan credits missing in plan_catalog: %v", err)
		return
	}

	_, _ = h.DB.Exec(ctx, `
		update public.user_quotas
		set plan_tier = $3,
		    monthly_credit_limit = $2,
		    purchased_topup_credits = greatest(0, purchased_topup_credits - greatest(0, monthly_credit_used - monthly_credit_limit)),
		    monthly_credit_used = 0,
		    updated_at = now()
		where user_id = $1
	`, userID, freeLimit, defaultTier)
	h.invalidateUserQuotaCache(ctx, userID)
	_, _ = h.DB.Exec(ctx, `
		update public.workspaces
		set plan_tier = $2, allocated_seats = 1, updated_at = now()
		where owner_id = $1::uuid
	`, userID, defaultTier)
	_, _ = h.DB.Exec(ctx, `
		update public.workspace_quotas wq
		set monthly_shared_credits = $2,
		    credits_consumed = 0,
		    updated_at = now()
		from public.workspaces w
		where w.id = wq.workspace_id and w.owner_id = $1::uuid
	`, userID, freeLimit)
	h.invalidateOwnedWorkspaceQuotaCaches(ctx, userID)
	_, _ = h.DB.Exec(ctx, `
		update public.subscriptions
		set status = 'expired', updated_at = now()
		where paddle_subscription_id = $1
	`, data.ID)

	_ = notifications.Insert(ctx, h.DB, notifications.InsertOpts{
		RecipientID: userID,
		Audience:    notifications.AudienceUser,
		KindCode:    notifications.KindUserSubscriptionCanceled,
		BodyArgs:    nil,
		DedupeKey:   "sub_canceled:" + data.ID,
	})
}

func (h *Handler) handleTransactionUpdated(ctx context.Context, event rawEvent) {
	var envelope struct {
		Data transactionData `json:"data"`
	}
	if err := json.Unmarshal(event.Body, &envelope); err != nil {
		return
	}
	data := envelope.Data
	status := strings.ToLower(strings.TrimSpace(data.Status))
	if status != "refunded" && status != "partially_refunded" && status != "canceled" {
		return
	}
	_, err := h.DB.Exec(ctx, `
		update public.billing_receipts
		set status = $2, updated_at = now()
		where paddle_transaction_id = $1
	`, data.ID, status)
	if err != nil {
		log.Printf("paddle: receipt refund status update: %v", err)
	}

	if status != "refunded" && status != "partially_refunded" {
		return
	}
	rows, err := h.DB.Query(ctx, `
		select id::text, user_id::text, credits_granted
		from public.billing_topup_ledger
		where paddle_transaction_id = $1
		  and clawed_at is null
	`, data.ID)
	if err != nil {
		log.Printf("paddle: topup clawback query txn=%s: %v", data.ID, err)
		return
	}
	defer rows.Close()
	clawedUsers := map[string]struct{}{}
	for rows.Next() {
		var ledgerID, uid string
		var credits int
		if rows.Scan(&ledgerID, &uid, &credits) != nil || ledgerID == "" || uid == "" || credits < 1 {
			continue
		}
		_, err = h.DB.Exec(ctx, `
			update public.user_quotas
			set purchased_topup_credits = greatest(0, purchased_topup_credits - $2),
			    updated_at = now()
			where user_id = $1::uuid
		`, uid, credits)
		if err != nil {
			log.Printf("paddle: topup clawback quota txn=%s ledger=%s: %v", data.ID, ledgerID, err)
			continue
		}
		_, err = h.DB.Exec(ctx, `
			update public.billing_topup_ledger
			set clawed_at = now()
			where id = $1::uuid and clawed_at is null
		`, ledgerID)
		if err != nil {
			log.Printf("paddle: topup clawback mark txn=%s ledger=%s: %v", data.ID, ledgerID, err)
			continue
		}
		clawedUsers[uid] = struct{}{}
	}
	for uid := range clawedUsers {
		h.invalidateUserQuotaCache(ctx, uid)
	}
}

func (h *Handler) handleTransactionCompleted(ctx context.Context, event rawEvent) {
	var envelope struct {
		Data transactionData `json:"data"`
	}
	if err := json.Unmarshal(event.Body, &envelope); err != nil {
		log.Printf("paddle: transaction parse error: %v", err)
		return
	}
	data := envelope.Data
	if data.CustomData.UserID == "" {
		log.Printf("paddle: transaction %s missing custom_data.userId", data.ID)
		return
	}
	receiptID, err := h.upsertCompletedTransaction(ctx, data)
	if err != nil {
		log.Printf("paddle: upsert receipt txn=%s: %v", data.ID, err)
		return
	}
	qty := 0
	if len(data.Items) > 0 {
		qty = data.Items[0].Quantity
	}
	subID := ""
	if data.SubscriptionID != nil {
		subID = strings.TrimSpace(*data.SubscriptionID)
	}
	h.fulfillEnterpriseInquiry(ctx, data.CustomData.UserID, data.CustomData.InquiryID, subID, data.ID, qty)
	h.notifyReceiptReady(ctx, data.CustomData.UserID, data.ID, receiptID)
}

func (h *Handler) notifyReceiptReady(ctx context.Context, userID, paddleTxnID, receiptID string) {
	userID = strings.TrimSpace(userID)
	paddleTxnID = strings.TrimSpace(paddleTxnID)
	receiptID = strings.TrimSpace(receiptID)
	if userID == "" || paddleTxnID == "" || receiptID == "" || h.DB == nil {
		return
	}
	_ = notifications.Insert(ctx, h.DB, notifications.InsertOpts{
		RecipientID:  userID,
		Audience:     notifications.AudienceUser,
		KindCode:     notifications.KindUserReceiptReady,
		BodyArgs:     nil,
		DedupeKey:    "receipt:" + paddleTxnID,
		HrefEntityID: receiptID,
	})
	_ = notifications.InsertForAdminsWithPermission(
		ctx,
		h.DB,
		"billing.receipts",
		notifications.KindAdminReceiptReady,
		nil,
		"admin_receipt:"+paddleTxnID,
		"",
		receiptID,
	)
}

// fulfillEnterpriseInquiry marks a sales offer activated after Paddle payment.
// Idempotent: only moves offered → activated for the owning user.
func (h *Handler) fulfillEnterpriseInquiry(ctx context.Context, userID, inquiryID, subscriptionID, transactionID string, seats int) {
	inquiryID = strings.TrimSpace(inquiryID)
	userID = strings.TrimSpace(userID)
	if inquiryID == "" || userID == "" {
		return
	}
	note := "paddle checkout completed"
	if seats > 0 {
		note = fmt.Sprintf("paddle checkout completed seats=%d", seats)
	}
	tag, err := h.DB.Exec(ctx, `
		update public.enterprise_inquiries
		set status = 'activated',
		    paddle_subscription_id = coalesce(nullif($3, ''), paddle_subscription_id),
		    paddle_transaction_id = coalesce(nullif($4, ''), paddle_transaction_id),
		    contract_notes = case
		      when coalesce(contract_notes, '') = '' then $5
		      else contract_notes || E'\n' || $5
		    end,
		    updated_at = now()
		where id = $1::uuid
		  and user_id = $2::uuid
		  and status = 'offered'
	`, inquiryID, userID, strings.TrimSpace(subscriptionID), strings.TrimSpace(transactionID), note)
	if err != nil {
		log.Printf("paddle: fulfill enterprise inquiry %s: %v", inquiryID, err)
		return
	}
	if tag.RowsAffected() == 0 {
		return
	}
	seatArg := ""
	if seats > 0 {
		seatArg = fmt.Sprintf("%d", seats)
	} else {
		var offered *int
		_ = h.DB.QueryRow(ctx, `
			select offered_seat_quantity from public.enterprise_inquiries where id = $1::uuid
		`, inquiryID).Scan(&offered)
		if offered != nil && *offered > 0 {
			seatArg = fmt.Sprintf("%d", *offered)
		}
	}
	_ = notifications.Insert(ctx, h.DB, notifications.InsertOpts{
		RecipientID:  userID,
		Audience:     notifications.AudienceUser,
		KindCode:     notifications.KindUserEnterpriseActivated,
		BodyArgs:     []string{seatArg},
		DedupeKey:    "enterprise_activated:" + inquiryID,
		HrefEntityID: inquiryID,
	})
}

func writeJSONErr(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": subscriptions.MessageForCode(code),
	})
}

// SyncReceipts pulls completed Paddle transactions for the signed-in user and
// upserts billing_receipts (idempotent; safe to re-run after missed webhooks).
func (h *Handler) SyncReceipts(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil || h.Paddle == nil {
		writeJSONErr(w, http.StatusServiceUnavailable, "BILLING_SYNC_UNAVAILABLE")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}

	var customerID string
	err := h.DB.QueryRow(r.Context(), `
		select paddle_customer_id from public.subscriptions
		where user_id = $1 and coalesce(paddle_customer_id, '') <> ''
		order by updated_at desc nulls last
		limit 1
	`, userID).Scan(&customerID)
	if err != nil || customerID == "" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"synced":        0,
			"message":       subscriptions.MessageForCode("BILLING_SYNC_NO_CUSTOMER"),
			"action_label":  subscriptions.ActionLabelForCode("RECEIPTS_SYNC"),
			"pending_label": subscriptions.PendingLabelForCode("RECEIPTS_SYNC"),
		})
		return
	}

	txns, err := h.Paddle.ListTransactions(r.Context(), customerID, "completed")
	if err != nil {
		writeJSONErr(w, http.StatusBadGateway, "BILLING_SYNC_UPSTREAM_FAILED")
		return
	}

	synced := 0
	for _, rawTxn := range txns {
		data, mapErr := transactionDataFromAPI(rawTxn)
		if mapErr != nil {
			log.Printf("paddle: sync map txn=%s: %v", rawTxn.ID, mapErr)
			continue
		}
		if data.CustomData.UserID == "" {
			data.CustomData.UserID = userID
		}
		if data.CustomData.UserID != userID {
			continue
		}
		receiptID, upsertErr := h.upsertCompletedTransaction(r.Context(), data)
		if upsertErr != nil {
			log.Printf("paddle: sync upsert txn=%s: %v", data.ID, upsertErr)
			continue
		}
		h.notifyReceiptReady(r.Context(), data.CustomData.UserID, data.ID, receiptID)
		synced++
	}

	syncMsg := ""
	if fmtMsg := subscriptions.MessageForCode("BILLING_SYNC_SYNCED_FMT"); fmtMsg != "" {
		syncMsg = fmt.Sprintf(fmtMsg, synced)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"synced":        synced,
		"customer_id":   customerID,
		"message":       syncMsg,
		"action_label":  subscriptions.ActionLabelForCode("RECEIPTS_SYNC"),
		"pending_label": subscriptions.PendingLabelForCode("RECEIPTS_SYNC"),
	})
}

func transactionDataFromAPI(src paddleapi.TransactionJSON) (transactionData, error) {
	raw, err := json.Marshal(src)
	if err != nil {
		return transactionData{}, err
	}
	var data transactionData
	if err := json.Unmarshal(raw, &data); err != nil {
		return transactionData{}, err
	}
	return data, nil
}

// ResyncTransactionByID fetches one Paddle transaction and upserts billing_receipts (admin / ops).
func (h *Handler) ResyncTransactionByID(ctx context.Context, transactionID string) error {
	if h.Paddle == nil {
		return fmt.Errorf("ADMIN_PADDLE_UNAVAILABLE")
	}
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return fmt.Errorf("ADMIN_RECEIPT_RESYNC_FAILED")
	}
	rawTxn, err := h.Paddle.GetTransaction(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("ADMIN_RECEIPT_RESYNC_FAILED")
	}
	data, err := transactionDataFromAPI(*rawTxn)
	if err != nil {
		return fmt.Errorf("ADMIN_RECEIPT_RESYNC_FAILED")
	}
	if data.CustomData.UserID == "" {
		var uid string
		_ = h.DB.QueryRow(ctx, `
			select coalesce(user_id::text, '') from public.billing_receipts
			where paddle_transaction_id = $1
		`, transactionID).Scan(&uid)
		data.CustomData.UserID = strings.TrimSpace(uid)
	}
	if data.CustomData.UserID == "" {
		return fmt.Errorf("ADMIN_RECEIPT_RESYNC_FAILED")
	}
	receiptID, err := h.upsertCompletedTransaction(ctx, data)
	if err != nil {
		return err
	}
	h.notifyReceiptReady(ctx, data.CustomData.UserID, data.ID, receiptID)
	return nil
}

// RefreshInvoicePDF fetches a fresh short-lived Paddle invoice PDF URL and stores it on the receipt.
func (h *Handler) RefreshInvoicePDF(ctx context.Context, receiptID, transactionID string) (string, error) {
	if h.Paddle == nil {
		return "", fmt.Errorf("ADMIN_PADDLE_UNAVAILABLE")
	}
	transactionID = strings.TrimSpace(transactionID)
	receiptID = strings.TrimSpace(receiptID)
	if transactionID == "" || receiptID == "" {
		return "", fmt.Errorf("ADMIN_RECEIPT_PDF_FAILED")
	}
	url, err := h.Paddle.GetTransactionInvoicePDF(ctx, transactionID)
	if err != nil || strings.TrimSpace(url) == "" {
		return "", fmt.Errorf("ADMIN_RECEIPT_PDF_FAILED")
	}
	tag, err := h.DB.Exec(ctx, `
		update public.billing_receipts
		set paddle_invoice_pdf_url = $1, updated_at = now()
		where id = $2::uuid
	`, url, receiptID)
	if err != nil {
		return "", fmt.Errorf("ADMIN_RECEIPT_PDF_FAILED")
	}
	if tag.RowsAffected() == 0 {
		return "", fmt.Errorf("ADMIN_RECEIPT_PDF_FAILED")
	}
	return url, nil
}

func (h *Handler) upsertCompletedTransaction(ctx context.Context, data transactionData) (string, error) {
	userID := data.CustomData.UserID
	if userID == "" {
		return "", fmt.Errorf("missing user id")
	}

	// Receipt money comes only from Paddle transaction details (Billing API minor units).
	subtotal, err := parsePaddleMinorUnits(data.Details.Totals.Subtotal)
	if err != nil {
		return "", fmt.Errorf("paddle details.totals.subtotal: %w", err)
	}
	tax, err := parsePaddleMinorUnits(data.Details.Totals.Tax)
	if err != nil {
		return "", fmt.Errorf("paddle details.totals.tax: %w", err)
	}
	total, err := parsePaddleMinorUnits(data.Details.Totals.Total)
	if err != nil {
		return "", fmt.Errorf("paddle details.totals.total: %w", err)
	}
	if len(data.Details.LineItems) == 0 {
		return "", fmt.Errorf("paddle details.line_items required")
	}
	taxRateBps := taxRateToBPS(data.Details.TaxRatesUsed)

	var periodStart, periodEnd *string
	if data.BillingPeriod != nil {
		periodStart = &data.BillingPeriod.StartsAt
		periodEnd = &data.BillingPeriod.EndsAt
	}

	email := ""
	if data.Customer != nil {
		email = strings.TrimSpace(data.Customer.Email)
	}
	if email == "" {
		_ = h.DB.QueryRow(ctx, `select email from public.profiles where id = $1`, userID).Scan(&email)
		email = strings.TrimSpace(email)
	}
	if email == "" {
		// Fail closed: never invent a placeholder bill-to email.
		msg := subscriptions.MessageForCode("RECEIPTS_BILL_TO_EMAIL_REQUIRED")
		if msg == "" {
			msg = "RECEIPTS_BILL_TO_EMAIL_REQUIRED"
		}
		return "", fmt.Errorf("%s", msg)
	}

	billToName := ""
	billToCompany := ""
	if data.Customer != nil {
		billToName = strings.TrimSpace(data.Customer.Name)
	}
	if data.Address != nil {
		composed := strings.TrimSpace(data.Address.FirstName + " " + data.Address.LastName)
		if billToName == "" {
			billToName = composed
		}
	}

	taxID := ""
	if data.BillingDetails != nil {
		taxID = strings.TrimSpace(data.BillingDetails.TaxIdentifier)
	}

	var line1, line2, city, region, postal, country *string
	if data.Address != nil {
		line1 = strPtr(data.Address.FirstLine)
		line2 = strPtr(data.Address.SecondLine)
		city = strPtr(data.Address.City)
		region = strPtr(data.Address.Region)
		postal = strPtr(data.Address.PostalCode)
		country = strPtr(data.Address.CountryCode)
	}

	invoiceNumber := data.InvoiceNumber
	if invoiceNumber == nil || *invoiceNumber == "" {
		invoiceNumber = data.InvoiceID
	}

	pdfURL := ""
	if h.Paddle != nil {
		if url, err := h.Paddle.GetTransactionInvoicePDF(ctx, data.ID); err == nil {
			pdfURL = url
		} else {
			log.Printf("paddle: invoice pdf for %s: %v", data.ID, err)
		}
	}

	paymentSummary := paymentMethodSummary(data)

	var receiptID string
	err = h.DB.QueryRow(ctx, `
		insert into public.billing_receipts (
			user_id, paddle_transaction_id, paddle_invoice_number, paddle_customer_id,
			paddle_subscription_id, status, currency_code,
			subtotal_cents, tax_cents, total_cents, tax_rate_bps, tax_id,
			bill_to_email, bill_to_name, bill_to_company,
			bill_to_address_line1, bill_to_address_line2, bill_to_city, bill_to_region,
			bill_to_postal_code, bill_to_country,
			paddle_invoice_pdf_url, payment_method_summary,
			period_start, period_end, paid_at
		) values (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, nullif($12, ''),
			$13, nullif($14, ''), nullif($15, ''),
			$16, $17, $18, $19, $20, $21,
			nullif($22, ''), nullif($23, ''),
			nullif($24, '')::timestamptz,
			nullif($25, '')::timestamptz,
			now()
		)
		on conflict (paddle_transaction_id) do update set
			status = excluded.status,
			subtotal_cents = excluded.subtotal_cents,
			tax_cents = excluded.tax_cents,
			total_cents = excluded.total_cents,
			tax_rate_bps = excluded.tax_rate_bps,
			tax_id = coalesce(excluded.tax_id, billing_receipts.tax_id),
			bill_to_email = excluded.bill_to_email,
			bill_to_name = coalesce(excluded.bill_to_name, billing_receipts.bill_to_name),
			bill_to_company = coalesce(excluded.bill_to_company, billing_receipts.bill_to_company),
			bill_to_address_line1 = coalesce(excluded.bill_to_address_line1, billing_receipts.bill_to_address_line1),
			bill_to_address_line2 = coalesce(excluded.bill_to_address_line2, billing_receipts.bill_to_address_line2),
			bill_to_city = coalesce(excluded.bill_to_city, billing_receipts.bill_to_city),
			bill_to_region = coalesce(excluded.bill_to_region, billing_receipts.bill_to_region),
			bill_to_postal_code = coalesce(excluded.bill_to_postal_code, billing_receipts.bill_to_postal_code),
			bill_to_country = coalesce(excluded.bill_to_country, billing_receipts.bill_to_country),
			paddle_invoice_pdf_url = coalesce(excluded.paddle_invoice_pdf_url, billing_receipts.paddle_invoice_pdf_url),
			paddle_invoice_number = coalesce(excluded.paddle_invoice_number, billing_receipts.paddle_invoice_number),
			payment_method_summary = coalesce(excluded.payment_method_summary, billing_receipts.payment_method_summary),
			updated_at = now()
		returning id::text
	`,
		userID, data.ID, nullString(invoiceNumber), data.CustomerID,
		nullString(data.SubscriptionID), data.Status, data.CurrencyCode,
		subtotal, tax, total, taxRateBps, taxID,
		email, billToName, billToCompany,
		nullString(line1), nullString(line2), nullString(city), nullString(region),
		nullString(postal), nullString(country),
		pdfURL, paymentSummary,
		deref(periodStart), deref(periodEnd),
	).Scan(&receiptID)
	if err != nil {
		return "", err
	}

	if _, err := h.DB.Exec(ctx, `delete from public.billing_receipt_line_items where receipt_id = $1::uuid`, receiptID); err != nil {
		return "", fmt.Errorf("delete receipt line items: %w", err)
	}

	priceNameByID := map[string]string{}
	for _, item := range data.Items {
		id := strings.TrimSpace(item.Price.ID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(item.Price.Name)
		if name == "" {
			name = strings.TrimSpace(item.Price.Description)
		}
		if name != "" {
			priceNameByID[id] = name
		}
	}

	for i, li := range data.Details.LineItems {
		priceID := strings.TrimSpace(li.PriceID)
		if priceID == "" {
			return "", fmt.Errorf("paddle details.line_items[%d].price_id required", i)
		}
		desc := strings.TrimSpace(li.Product.Name)
		if desc == "" {
			return "", fmt.Errorf("paddle details.line_items[%d].product.name required", i)
		}
		sku := strings.TrimSpace(li.Product.SKU)
		if sku == "" {
			sku = strings.TrimSpace(li.Product.ID)
		}
		if sku == "" {
			return "", fmt.Errorf("paddle details.line_items[%d].product.id required", i)
		}
		if li.Quantity < 1 {
			return "", fmt.Errorf("paddle details.line_items[%d].quantity required", i)
		}
		unitCents, err := parsePaddleMinorUnits(li.UnitTotals.Subtotal)
		if err != nil {
			return "", fmt.Errorf("paddle details.line_items[%d].unit_totals.subtotal: %w", i, err)
		}
		amountCents, err := parsePaddleMinorUnits(li.Totals.Total)
		if err != nil {
			return "", fmt.Errorf("paddle details.line_items[%d].totals.total: %w", i, err)
		}
		priceName := strings.TrimSpace(priceNameByID[priceID])
		if priceName == "" {
			priceName = h.catalogPriceName(ctx, priceID)
		}

		if _, err := h.DB.Exec(ctx, `
			insert into public.billing_receipt_line_items (
				receipt_id, position, description, quantity, unit_amount_cents, amount_cents, product_sku, price_id, price_name
			) values ($1::uuid, $2, $3, $4, $5, $6, $7, $8, nullif($9, ''))
		`, receiptID, i, desc, li.Quantity, unitCents, amountCents, sku, priceID, priceName); err != nil {
			return "", fmt.Errorf("insert receipt line item: %w", err)
		}

		h.applyTopupCredits(ctx, userID, data.ID, priceID, data.CustomData.PlanID, li.Quantity)
	}
	return receiptID, nil
}

// catalogPriceName builds the Paddle-style price subtitle from plan_catalog
// when the transaction payload omitted price.name (e.g. "Pro (monthly)").
func (h *Handler) catalogPriceName(ctx context.Context, priceID string) string {
	priceID = strings.TrimSpace(priceID)
	if priceID == "" || h.DB == nil {
		return ""
	}
	var display string
	var slot string
	err := h.DB.QueryRow(ctx, `
		select display_name,
		       case
		         when paddle_price_id_monthly = $1 then 'monthly'
		         when paddle_price_id_yearly = $1 then 'yearly'
		         else ''
		       end
		from public.plan_catalog
		where paddle_price_id_monthly = $1 or paddle_price_id_yearly = $1
		limit 1
	`, priceID).Scan(&display, &slot)
	if err != nil {
		return ""
	}
	display = strings.TrimSpace(display)
	slot = strings.TrimSpace(slot)
	if display == "" || slot == "" {
		return ""
	}
	var fmtCode string
	switch slot {
	case "monthly":
		fmtCode = "RECEIPT_PRICE_NAME_MONTHLY_FMT"
	case "yearly":
		fmtCode = "RECEIPT_PRICE_NAME_YEARLY_FMT"
	default:
		return ""
	}
	tmpl := subscriptions.MessageForCode(fmtCode)
	if tmpl == "" || !strings.Contains(tmpl, "%s") {
		return ""
	}
	return fmt.Sprintf(tmpl, display)
}

func paymentMethodSummary(data transactionData) string {
	for _, p := range data.Payments {
		if p.MethodDetails == nil {
			continue
		}
		md := p.MethodDetails
		if md.Card != nil {
			brand := strings.TrimSpace(md.Card.Type)
			last4 := strings.TrimSpace(md.Card.Last4)
			// Paddle invoice pattern: "visa - 7299"
			if brand != "" && last4 != "" {
				return brand + " - " + last4
			}
			if last4 != "" {
				return last4
			}
		}
		if t := strings.TrimSpace(md.Type); t != "" {
			return t
		}
	}
	return ""
}

// applyTopupCredits grants one-time pack credits exactly once per
// (paddle_transaction_id, price_id). Retried webhooks cannot double-credit.
func (h *Handler) applyTopupCredits(ctx context.Context, userID, transactionID, priceID, planIDHint string, quantity int) {
	if quantity < 1 {
		quantity = 1
	}
	if transactionID == "" {
		log.Printf("paddle: topup skipped (missing transaction id) user=%s price=%s", userID, priceID)
		return
	}
	credits, ok := h.resolveTopupCredits(ctx, priceID, planIDHint)
	if !ok || credits <= 0 {
		return
	}
	total := credits * quantity

	var ledgerID string
	err := h.DB.QueryRow(ctx, `
		insert into public.billing_topup_ledger (
			user_id, paddle_transaction_id, price_id, plan_id_hint, credits_granted, quantity
		) values ($1, $2, $3, nullif($4, ''), $5, $6)
		on conflict (paddle_transaction_id, price_id) do nothing
		returning id::text
	`, userID, transactionID, priceID, planIDHint, total, quantity).Scan(&ledgerID)
	if err != nil {
		// Conflict / no row: already applied for this transaction line.
		if err == pgx.ErrNoRows {
			log.Printf("paddle: topup already applied txn=%s price=%s user=%s", transactionID, priceID, userID)
			return
		}
		log.Printf("paddle: topup ledger insert failed user=%s txn=%s: %v", userID, transactionID, err)
		return
	}
	if ledgerID == "" {
		return
	}

	_, err = h.DB.Exec(ctx, `
		update public.user_quotas
		set purchased_topup_credits = purchased_topup_credits + $2,
		    updated_at = now()
		where user_id = $1
	`, userID, total)
	if err != nil {
		log.Printf("paddle: topup credit apply failed user=%s ledger=%s: %v", userID, ledgerID, err)
		// Release ledger row so a later webhook retry can re-apply.
		_, _ = h.DB.Exec(ctx, `delete from public.billing_topup_ledger where id = $1::uuid`, ledgerID)
		return
	}
	h.invalidateUserQuotaCache(ctx, userID)
	h.invalidateOwnedWorkspaceQuotaCaches(ctx, userID)
	log.Printf("paddle: applied %d topup credits to user %s txn=%s", total, userID, transactionID)
}

func (h *Handler) resolveTopupCredits(ctx context.Context, priceID, planIDHint string) (int, bool) {
	if planIDHint != "" {
		var credits int
		err := h.DB.QueryRow(ctx, `
			select credits_monthly from public.plan_catalog
			where id = $1 and is_active = true and plan_kind = 'topup'
		`, planIDHint).Scan(&credits)
		if err == nil {
			return credits, true
		}
	}
	if priceID == "" {
		return 0, false
	}
	var credits int
	err := h.DB.QueryRow(ctx, `
		select credits_monthly from public.plan_catalog
		where is_active = true
		  and plan_kind = 'topup'
		  and (
		    paddle_price_id_topup = $1
		    or paddle_price_id_monthly = $1
		  )
		limit 1
	`, priceID).Scan(&credits)
	if err != nil {
		return 0, false
	}
	return credits, true
}

func (h *Handler) resolvePlanFromPrice(ctx context.Context, priceID, planIDHint string) (tier string, limit int, err error) {
	if planIDHint != "" {
		err = h.DB.QueryRow(ctx, `
			select id, credits_monthly from public.plan_catalog
			where id = $1 and is_active = true
			  and plan_kind in ('subscription', 'enterprise')
		`, planIDHint).Scan(&tier, &limit)
		if err == nil {
			return tier, limit, nil
		}
		if err != pgx.ErrNoRows {
			return "", 0, err
		}
	}

	if priceID == "" {
		return "", 0, pgx.ErrNoRows
	}

	err = h.DB.QueryRow(ctx, `
		select id, credits_monthly from public.plan_catalog
		where is_active = true
		  and plan_kind in ('subscription', 'enterprise')
		  and (
		    paddle_price_id_monthly = $1
		    or paddle_price_id_yearly = $1
		  )
		limit 1
	`, priceID).Scan(&tier, &limit)
	return tier, limit, err
}

func verifyPaddleSignature(header string, body []byte, secret string) bool {
	parts := strings.Split(header, ";")
	var ts, h1 string
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "ts":
			ts = kv[1]
		case "h1":
			h1 = kv[1]
		}
	}
	if ts == "" || h1 == "" {
		return false
	}
	signed := ts + ":" + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signed))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(h1), []byte(expected))
}

// parsePaddleMinorUnits parses a Paddle Billing money string.
// Paddle Billing amounts are integer strings in the lowest currency unit
// (e.g. "100" = 100 USD cents = $1.00). No major-unit decimals, no guessing.
func parsePaddleMinorUnits(v string) (int64, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, fmt.Errorf("empty")
	}
	neg := false
	if strings.HasPrefix(v, "-") {
		neg = true
		v = strings.TrimSpace(v[1:])
		if v == "" {
			return 0, fmt.Errorf("empty")
		}
	}
	if strings.ContainsAny(v, ".eE") {
		return 0, fmt.Errorf("invalid %q: expected integer minor units", v)
	}
	n := int64(0)
	for _, c := range v {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid %q: non-digit", v)
		}
		n = n*10 + int64(c-'0')
	}
	if neg {
		return -n, nil
	}
	return n, nil
}

func taxRateToBPS(rates []struct {
	TaxRate string `json:"tax_rate"`
}) int {
	if len(rates) == 0 {
		return 0
	}
	raw := strings.TrimSpace(rates[0].TaxRate)
	if raw == "" {
		return 0
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	if f <= 1 {
		return int(f * 10000)
	}
	return int(f * 100)
}

func nullString(p *string) interface{} {
	if p == nil || *p == "" {
		return nil
	}
	return *p
}

func strPtr(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func intervalFromPrice(ctx context.Context, db *pgxpool.Pool, priceID string) string {
	if db == nil || priceID == "" {
		return ""
	}
	var monthly, yearly *string
	err := db.QueryRow(ctx, `
		select paddle_price_id_monthly, paddle_price_id_yearly
		from public.plan_catalog
		where paddle_price_id_monthly = $1 or paddle_price_id_yearly = $1
		limit 1
	`, priceID).Scan(&monthly, &yearly)
	if err != nil {
		return ""
	}
	if yearly != nil && *yearly == priceID {
		return "annual"
	}
	if monthly != nil && *monthly == priceID {
		return "monthly"
	}
	return ""
}
