package catalogsync

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usetrim/trim/server/internal/billing/paddleapi"
)

var (
	// ErrPaddleUnavailable means the API has no Paddle client (non-cloud / misconfig).
	ErrPaddleUnavailable = errors.New("paddle catalog sync unavailable")
	// ErrPaddleKeyInvalid means the configured API key is a placeholder or otherwise unusable for writes.
	ErrPaddleKeyInvalid = errors.New("paddle api key invalid")
	// ErrPlanNotFound means the plan_catalog row does not exist.
	ErrPlanNotFound = errors.New("plan not found")
	// ErrCurrencyInvalid means billing_settings.default_currency cannot be used for Paddle.
	ErrCurrencyInvalid = errors.New("billing currency invalid")
)

// Syncer pushes admin plan_catalog amounts into Paddle products/prices and stores returned IDs.
// Admin is the control plane; Paddle charges exactly those catalog prices at checkout.
type Syncer struct {
	DB     *pgxpool.Pool
	Paddle *paddleapi.Client
}

func New(db *pgxpool.Pool, paddle *paddleapi.Client) *Syncer {
	return &Syncer{DB: db, Paddle: paddle}
}

type planRow struct {
	ID           string
	DisplayName  string
	Description  string
	PlanKind     string
	Currency     string
	MonthlyCents *int
	YearlyCents  *int
	IsActive     bool
	IsPublic     bool
	ProductID    string
	PriceMonthly string
	PriceYearly  string
	PriceTopup   string
}

type settingsRow struct {
	AnnualDiscountPercent       int
	ApplyPaddleDiscountOnAnnual bool
	PaddleDiscountID            string
	DefaultCurrency             string
}

// PlanResult is the Paddle IDs after a successful plan sync.
type PlanResult struct {
	PlanID               string `json:"plan_id"`
	PaddleProductID      string `json:"paddle_product_id,omitempty"`
	PaddlePriceIDMonthly string `json:"paddle_price_id_monthly,omitempty"`
	PaddlePriceIDYearly  string `json:"paddle_price_id_yearly,omitempty"`
	PaddlePriceIDTopup   string `json:"paddle_price_id_topup,omitempty"`
	ClearMonthly         bool   `json:"-"`
	ClearYearly          bool   `json:"-"`
	ClearTopup           bool   `json:"-"`
	Skipped              bool   `json:"skipped,omitempty"`
	SkipReason           string `json:"skip_reason,omitempty"`
}

// CatalogResult summarizes a full catalog sync.
type CatalogResult struct {
	Plans        []PlanResult `json:"plans"`
	DiscountID   string       `json:"paddle_discount_id,omitempty"`
	PricingBound bool         `json:"pricing_bound"`
}

func (s *Syncer) requirePaddle() error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("catalog syncer not configured")
	}
	if s.Paddle == nil {
		return ErrPaddleUnavailable
	}
	if err := s.Paddle.CatalogWriteReady(); err != nil {
		return fmt.Errorf("%w: %v", ErrPaddleKeyInvalid, err)
	}
	return nil
}

// PreflightSync checks currency + Paddle write readiness without mutating catalog rows.
// Call before admin plan writes when sync_to_paddle is on, so a doomed sync does not
// leave "HTTP error but amounts already saved" ambiguity.
func (s *Syncer) PreflightSync(ctx context.Context) error {
	if err := s.requirePaddle(); err != nil {
		return err
	}
	settings, err := s.loadSettings(ctx)
	if err != nil {
		return err
	}
	if !currencyOK(settings.DefaultCurrency) {
		return ErrCurrencyInvalid
	}
	return nil
}

// catalogLockKey is a stable int64 advisory lock key for Paddle catalog mutations.
func catalogLockKey() int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte("trim:paddle:catalogsync"))
	// Keep positive for pg_advisory_lock.
	return int64(h.Sum64() & 0x7fffffffffffffff)
}

func (s *Syncer) withCatalogLock(ctx context.Context, fn func(context.Context) error) error {
	conn, err := s.DB.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, `select pg_advisory_lock($1)`, catalogLockKey()); err != nil {
		return fmt.Errorf("catalog sync lock: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), `select pg_advisory_unlock($1)`, catalogLockKey())
	}()
	return fn(ctx)
}

func (s *Syncer) loadSettings(ctx context.Context) (settingsRow, error) {
	var row settingsRow
	err := s.DB.QueryRow(ctx, `
		select annual_discount_percent, apply_paddle_discount_on_annual,
		       coalesce(paddle_discount_id, ''), coalesce(default_currency, '')
		from public.billing_settings where id = 'default'
	`).Scan(&row.AnnualDiscountPercent, &row.ApplyPaddleDiscountOnAnnual, &row.PaddleDiscountID, &row.DefaultCurrency)
	if err != nil {
		return settingsRow{}, err
	}
	row.PaddleDiscountID = strings.TrimSpace(row.PaddleDiscountID)
	row.DefaultCurrency = strings.ToUpper(strings.TrimSpace(row.DefaultCurrency))
	return row, nil
}

func currencyOK(code string) bool {
	code = strings.ToUpper(strings.TrimSpace(code))
	return len(code) == 3 && code != "XXX"
}

// ValidBillingCurrency reports whether code is usable for Paddle catalog sync (ISO-4217, not XXX).
func ValidBillingCurrency(code string) bool {
	return currencyOK(code)
}

func (s *Syncer) loadPlan(ctx context.Context, planID string) (planRow, error) {
	var p planRow
	err := s.DB.QueryRow(ctx, `
		select id, display_name, coalesce(description, ''), plan_kind,
		       coalesce(currency_code, ''), price_monthly_cents, price_yearly_cents,
		       is_active, is_public,
		       coalesce(paddle_product_id, ''), coalesce(paddle_price_id_monthly, ''),
		       coalesce(paddle_price_id_yearly, ''), coalesce(paddle_price_id_topup, '')
		from public.plan_catalog where id = $1
	`, planID).Scan(
		&p.ID, &p.DisplayName, &p.Description, &p.PlanKind, &p.Currency,
		&p.MonthlyCents, &p.YearlyCents, &p.IsActive, &p.IsPublic,
		&p.ProductID, &p.PriceMonthly, &p.PriceYearly, &p.PriceTopup,
	)
	if err != nil {
		return planRow{}, err
	}
	p.ProductID = strings.TrimSpace(p.ProductID)
	p.PriceMonthly = strings.TrimSpace(p.PriceMonthly)
	p.PriceYearly = strings.TrimSpace(p.PriceYearly)
	p.PriceTopup = strings.TrimSpace(p.PriceTopup)
	p.Currency = strings.ToUpper(strings.TrimSpace(p.Currency))
	p.PlanKind = strings.ToLower(strings.TrimSpace(p.PlanKind))
	return p, nil
}

// SyncPlan ensures Paddle product + prices match admin cents for one plan, then writes IDs back.
func (s *Syncer) SyncPlan(ctx context.Context, planID string) (PlanResult, error) {
	if err := s.requirePaddle(); err != nil {
		return PlanResult{}, err
	}
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return PlanResult{}, ErrPlanNotFound
	}
	var out PlanResult
	err := s.withCatalogLock(ctx, func(ctx context.Context) error {
		var serr error
		out, serr = s.syncPlanLocked(ctx, planID)
		return serr
	})
	return out, err
}

func (s *Syncer) syncPlanLocked(ctx context.Context, planID string) (PlanResult, error) {
	settings, err := s.loadSettings(ctx)
	if err != nil {
		return PlanResult{}, err
	}
	if !currencyOK(settings.DefaultCurrency) {
		return PlanResult{}, ErrCurrencyInvalid
	}
	plan, err := s.loadPlan(ctx, planID)
	if err != nil {
		return PlanResult{}, ErrPlanNotFound
	}
	currency := settings.DefaultCurrency
	if plan.Currency != currency {
		if _, err := s.DB.Exec(ctx, `
			update public.plan_catalog set currency_code = $2, updated_at = now() where id = $1
		`, plan.ID, currency); err != nil {
			return PlanResult{}, err
		}
		plan.Currency = currency
	}

	out := PlanResult{PlanID: plan.ID}
	if !needsPaddleCatalog(plan) {
		s.markClearsForZeroAmounts(plan, &out)
		if out.ClearMonthly || out.ClearYearly || out.ClearTopup {
			// Persist clears first so checkout never keeps a pointer at a price we are about to archive.
			if err := s.persistPlanIDs(ctx, out); err != nil {
				return PlanResult{}, err
			}
			s.archiveCleared(ctx, plan, out)
		}
		out.Skipped = true
		out.SkipReason = "no sellable paddle prices for this plan"
		return out, nil
	}

	productID, err := s.ensureProduct(ctx, plan)
	if err != nil {
		return PlanResult{}, err
	}
	out.PaddleProductID = productID
	plan.ProductID = productID
	// Persist product id early so a later price-slot failure does not lose the product binding.
	if err := s.persistPlanIDs(ctx, out); err != nil {
		return PlanResult{}, err
	}

	persistAndArchive := func(oldID string) error {
		if err := s.persistPlanIDs(ctx, out); err != nil {
			return err
		}
		if oldID != "" {
			_ = s.Paddle.ArchivePrice(ctx, oldID)
		}
		return nil
	}

	switch plan.PlanKind {
	case "topup":
		cents := centsOrZero(plan.MonthlyCents)
		if cents <= 0 {
			out.ClearTopup = plan.PriceTopup != ""
			out.Skipped = true
			out.SkipReason = "topup amount missing"
			if err := s.persistPlanIDs(ctx, out); err != nil {
				return PlanResult{}, err
			}
			if out.ClearTopup {
				_ = s.Paddle.ArchivePrice(ctx, plan.PriceTopup)
			}
			return out, nil
		}
		priceID, oldID, err := s.ensurePrice(ctx, plan, productID, plan.PriceTopup, cents, currency, "", "topup")
		if err != nil {
			return PlanResult{}, err
		}
		out.PaddlePriceIDTopup = priceID
		if err := persistAndArchive(oldID); err != nil {
			return PlanResult{}, err
		}
	default:
		if centsOrZero(plan.MonthlyCents) > 0 {
			priceID, oldID, err := s.ensurePrice(ctx, plan, productID, plan.PriceMonthly, *plan.MonthlyCents, currency, "month", "monthly")
			if err != nil {
				return PlanResult{}, err
			}
			out.PaddlePriceIDMonthly = priceID
			if err := persistAndArchive(oldID); err != nil {
				return PlanResult{}, err
			}
		} else if plan.PriceMonthly != "" {
			out.ClearMonthly = true
			if err := s.persistPlanIDs(ctx, out); err != nil {
				return PlanResult{}, err
			}
			_ = s.Paddle.ArchivePrice(ctx, plan.PriceMonthly)
			out.ClearMonthly = false
		}
		if centsOrZero(plan.YearlyCents) > 0 {
			priceID, oldID, err := s.ensurePrice(ctx, plan, productID, plan.PriceYearly, *plan.YearlyCents, currency, "year", "yearly")
			if err != nil {
				return PlanResult{}, err
			}
			out.PaddlePriceIDYearly = priceID
			if err := persistAndArchive(oldID); err != nil {
				return PlanResult{}, err
			}
		} else if plan.PriceYearly != "" {
			out.ClearYearly = true
			if err := s.persistPlanIDs(ctx, out); err != nil {
				return PlanResult{}, err
			}
			_ = s.Paddle.ArchivePrice(ctx, plan.PriceYearly)
			out.ClearYearly = false
		}
	}

	return out, nil
}

func markClearsForZeroAmounts(plan planRow, out *PlanResult) {
	if plan.PlanKind == "topup" {
		if centsOrZero(plan.MonthlyCents) <= 0 && plan.PriceTopup != "" {
			out.ClearTopup = true
		}
		return
	}
	if plan.PlanKind == "subscription" || plan.PlanKind == "enterprise" || strings.EqualFold(plan.ID, "enterprise") {
		if centsOrZero(plan.MonthlyCents) <= 0 && plan.PriceMonthly != "" {
			out.ClearMonthly = true
		}
		if centsOrZero(plan.YearlyCents) <= 0 && plan.PriceYearly != "" {
			out.ClearYearly = true
		}
	}
}

func (s *Syncer) markClearsForZeroAmounts(plan planRow, out *PlanResult) {
	markClearsForZeroAmounts(plan, out)
}

func (s *Syncer) archiveCleared(ctx context.Context, plan planRow, out PlanResult) {
	if out.ClearMonthly && plan.PriceMonthly != "" {
		_ = s.Paddle.ArchivePrice(ctx, plan.PriceMonthly)
	}
	if out.ClearYearly && plan.PriceYearly != "" {
		_ = s.Paddle.ArchivePrice(ctx, plan.PriceYearly)
	}
	if out.ClearTopup && plan.PriceTopup != "" {
		_ = s.Paddle.ArchivePrice(ctx, plan.PriceTopup)
	}
}

func needsPaddleCatalog(plan planRow) bool {
	if !plan.IsActive {
		return false
	}
	switch plan.PlanKind {
	case "topup":
		return centsOrZero(plan.MonthlyCents) > 0
	case "subscription", "enterprise":
		return centsOrZero(plan.MonthlyCents) > 0 || centsOrZero(plan.YearlyCents) > 0
	default:
		// Legacy rows keyed by id=enterprise without plan_kind set.
		if strings.EqualFold(plan.ID, "enterprise") {
			return centsOrZero(plan.MonthlyCents) > 0 || centsOrZero(plan.YearlyCents) > 0
		}
		return false
	}
}

func centsOrZero(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func (s *Syncer) ensureProduct(ctx context.Context, plan planRow) (string, error) {
	name := strings.TrimSpace(plan.DisplayName)
	if name == "" {
		name = plan.ID
	}
	desc := strings.TrimSpace(plan.Description)
	custom := map[string]interface{}{"trim_plan_id": plan.ID}

	if plan.ProductID != "" {
		_, err := s.Paddle.UpdateProduct(ctx, plan.ProductID, paddleapi.UpdateProductRequest{
			Name:        &name,
			Description: &desc,
			CustomData:  custom,
		})
		if err == nil {
			return plan.ProductID, nil
		}
		if !paddleNotFound(err) {
			return "", err
		}
	}

	created, err := s.createProductWithTaxFallback(ctx, name, desc, custom)
	if err != nil {
		return "", err
	}
	return created.Data.ID, nil
}

func (s *Syncer) createProductWithTaxFallback(ctx context.Context, name, desc string, custom map[string]interface{}) (*paddleapi.ProductResponse, error) {
	// saas is preferred for cloud software; fall back to standard if account has not enabled saas.
	for _, tax := range []string{"saas", "standard"} {
		created, err := s.Paddle.CreateProduct(ctx, paddleapi.CreateProductRequest{
			Name:        name,
			TaxCategory: tax,
			Description: desc,
			Type:        "standard",
			CustomData:  custom,
		})
		if err == nil {
			return created, nil
		}
		if tax == "standard" {
			return nil, err
		}
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "tax") || strings.Contains(msg, "invalid") || strings.Contains(msg, "category") {
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("paddle create product failed")
}

// ensurePrice returns the active price id matching admin cents.
// If a replacement price is created, oldPriceID is the previous id to archive AFTER the caller
// persists the new id (so plan_catalog never points at an archived price).
func (s *Syncer) ensurePrice(
	ctx context.Context,
	plan planRow,
	productID, existingPriceID string,
	cents int,
	currency, interval, slot string,
) (priceID string, oldPriceID string, err error) {
	amount := strconv.Itoa(cents)
	currency = strings.ToUpper(strings.TrimSpace(currency))
	name := priceName(plan.DisplayName, slot)
	desc := fmt.Sprintf("Trim %s %s", plan.ID, slot)
	custom := map[string]interface{}{
		"trim_plan_id": plan.ID,
		"trim_slot":    slot,
	}
	var cycle *paddleapi.Duration
	if interval != "" {
		cycle = &paddleapi.Duration{Interval: interval, Frequency: 1}
	}

	req := paddleapi.CreatePriceRequest{
		Description:  desc,
		Name:         name,
		ProductID:    productID,
		UnitPrice:    paddleapi.Money{Amount: amount, CurrencyCode: currency},
		BillingCycle: cycle,
		TaxMode:      "account_setting",
		Type:         "standard",
		CustomData:   custom,
	}

	if existingPriceID != "" {
		got, gerr := s.Paddle.GetPrice(ctx, existingPriceID)
		if gerr == nil && priceMatches(got, productID, amount, currency, interval) {
			return existingPriceID, "", nil
		}
		if gerr != nil && !paddleNotFound(gerr) {
			return "", "", gerr
		}
		created, cerr := s.Paddle.CreatePrice(ctx, req)
		if cerr != nil {
			return "", "", cerr
		}
		// Archive only after caller persists created.Data.ID.
		return created.Data.ID, existingPriceID, nil
	}

	created, err := s.Paddle.CreatePrice(ctx, req)
	if err != nil {
		return "", "", err
	}
	return created.Data.ID, "", nil
}

func paddleNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// Paddle returns error.code=not_found but doJSON prefers error.detail, e.g.
	// "pro_… Product not found." / "Price pri_… not found." - match both forms.
	return strings.Contains(msg, "not_found") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "entity_not_found") ||
		strings.Contains(msg, "404")
}

func priceName(display, slot string) string {
	display = strings.TrimSpace(display)
	if display == "" {
		return slot
	}
	switch slot {
	case "monthly":
		return display + " (monthly)"
	case "yearly":
		return display + " (yearly)"
	case "topup":
		return display
	default:
		return display
	}
}

func priceMatches(got *paddleapi.PriceResponse, productID, amount, currency, interval string) bool {
	if got == nil {
		return false
	}
	if strings.EqualFold(got.Data.Status, "archived") {
		return false
	}
	if productID != "" && got.Data.ProductID != "" && got.Data.ProductID != productID {
		return false
	}
	if strings.TrimSpace(got.Data.UnitPrice.Amount) != amount {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(got.Data.UnitPrice.CurrencyCode), currency) {
		return false
	}
	if interval == "" {
		return got.Data.BillingCycle == nil
	}
	if got.Data.BillingCycle == nil {
		return false
	}
	return strings.EqualFold(got.Data.BillingCycle.Interval, interval) && got.Data.BillingCycle.Frequency == 1
}

func (s *Syncer) persistPlanIDs(ctx context.Context, r PlanResult) error {
	_, err := s.DB.Exec(ctx, `
		update public.plan_catalog set
			paddle_product_id = case when $2 <> '' then $2 else paddle_product_id end,
			paddle_price_id_monthly = case
				when $6 then null
				when $3 <> '' then $3
				else paddle_price_id_monthly
			end,
			paddle_price_id_yearly = case
				when $7 then null
				when $4 <> '' then $4
				else paddle_price_id_yearly
			end,
			paddle_price_id_topup = case
				when $8 then null
				when $5 <> '' then $5
				else paddle_price_id_topup
			end,
			updated_at = now()
		where id = $1
	`, r.PlanID, r.PaddleProductID, r.PaddlePriceIDMonthly, r.PaddlePriceIDYearly, r.PaddlePriceIDTopup,
		r.ClearMonthly, r.ClearYearly, r.ClearTopup)
	return err
}

// SyncDiscount is intentionally a no-op for self-serve billing.
// Annual pricing is encoded in plan_catalog.price_yearly_cents → yearly pri_*.
func (s *Syncer) SyncDiscount(ctx context.Context) (string, error) {
	if err := s.requirePaddle(); err != nil {
		return "", err
	}
	settings, err := s.loadSettings(ctx)
	if err != nil {
		return "", err
	}
	return settings.PaddleDiscountID, nil
}

// ApplyAnnualDiscountToYearlyCents sets price_yearly_cents from monthly × 12 × (100-pct)/100
// for active subscription plans. Operator-owned write (admin settings), not read-time invent.
func (s *Syncer) ApplyAnnualDiscountToYearlyCents(ctx context.Context, percent int) (int64, error) {
	if percent < 0 || percent > 90 {
		return 0, fmt.Errorf("annual_discount_percent must be 0-90")
	}
	var affected int64
	err := s.withCatalogLock(ctx, func(ctx context.Context) error {
		tag, err := s.DB.Exec(ctx, `
			update public.plan_catalog pc
			set price_yearly_cents = round(
			      pc.price_monthly_cents::numeric * 12 * (100 - $1::numeric) / 100
			    )::int,
			    updated_at = now()
			where pc.is_active = true
			  and pc.plan_kind = 'subscription'
			  and pc.price_monthly_cents is not null
			  and pc.price_monthly_cents > 0
		`, percent)
		if err != nil {
			return err
		}
		affected = tag.RowsAffected()
		return nil
	})
	return affected, err
}

// SyncCatalog syncs every active plan that needs Paddle prices, then pricing_bound.
func (s *Syncer) SyncCatalog(ctx context.Context) (CatalogResult, error) {
	if err := s.requirePaddle(); err != nil {
		return CatalogResult{}, err
	}
	var out CatalogResult
	err := s.withCatalogLock(ctx, func(ctx context.Context) error {
		settings, err := s.loadSettings(ctx)
		if err != nil {
			return err
		}
		if !currencyOK(settings.DefaultCurrency) {
			return ErrCurrencyInvalid
		}

		rows, err := s.DB.Query(ctx, `
			select id from public.plan_catalog
			where is_active = true
			  and plan_kind in ('subscription', 'topup', 'enterprise')
			order by sort_order asc, id asc
		`)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			// Already holding catalog lock; call locked path to avoid re-entrant deadlock.
			res, err := s.syncPlanLocked(ctx, id)
			if err != nil {
				return fmt.Errorf("sync plan %s: %w", id, err)
			}
			out.Plans = append(out.Plans, res)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		discID, err := s.SyncDiscount(ctx)
		if err != nil {
			return fmt.Errorf("sync discount: %w", err)
		}
		out.DiscountID = discID

		bound, err := s.refreshPricingBoundLocked(ctx)
		if err != nil {
			return err
		}
		out.PricingBound = bound
		return nil
	})
	return out, err
}

// RefreshPricingBound sets pricing_bound true only when currency is valid and every
// public active sellable plan has the required Paddle price IDs.
func (s *Syncer) RefreshPricingBound(ctx context.Context) (bool, error) {
	var ready bool
	err := s.withCatalogLock(ctx, func(ctx context.Context) error {
		var rerr error
		ready, rerr = s.refreshPricingBoundLocked(ctx)
		return rerr
	})
	return ready, err
}

func (s *Syncer) refreshPricingBoundLocked(ctx context.Context) (bool, error) {
	settings, err := s.loadSettings(ctx)
	if err != nil {
		return false, err
	}
	ready := currencyOK(settings.DefaultCurrency)
	if ready {
		var missing int
		err := s.DB.QueryRow(ctx, `
			select count(*) from public.plan_catalog
			where is_active = true and is_public = true
			  and (
			    (plan_kind = 'subscription'
			      and coalesce(price_monthly_cents, 0) > 0
			      and nullif(btrim(coalesce(paddle_price_id_monthly, '')), '') is null)
			    or
			    (plan_kind = 'subscription'
			      and coalesce(price_yearly_cents, 0) > 0
			      and nullif(btrim(coalesce(paddle_price_id_yearly, '')), '') is null)
			    or
			    (plan_kind = 'topup'
			      and coalesce(price_monthly_cents, 0) > 0
			      and nullif(btrim(coalesce(paddle_price_id_topup, '')), '') is null)
			  )
		`).Scan(&missing)
		if err != nil {
			return false, err
		}
		// Require at least one public sellable plan that is fully bound.
		// Currency-only / free-only catalogs must stay unbound (no silent "ready").
		var boundSellable int
		err = s.DB.QueryRow(ctx, `
			select count(*) from public.plan_catalog
			where is_active = true and is_public = true
			  and (
			    (plan_kind = 'subscription'
			      and coalesce(price_monthly_cents, 0) > 0
			      and nullif(btrim(coalesce(paddle_price_id_monthly, '')), '') is not null)
			    or
			    (plan_kind = 'subscription'
			      and coalesce(price_yearly_cents, 0) > 0
			      and nullif(btrim(coalesce(paddle_price_id_yearly, '')), '') is not null)
			    or
			    (plan_kind = 'topup'
			      and coalesce(price_monthly_cents, 0) > 0
			      and nullif(btrim(coalesce(paddle_price_id_topup, '')), '') is not null)
			  )
		`).Scan(&boundSellable)
		if err != nil {
			return false, err
		}
		ready = missing == 0 && boundSellable > 0
	}

	_, err = s.DB.Exec(ctx, `
		update public.billing_settings
		set pricing_bound = $1, updated_at = now()
		where id = 'default'
	`, ready)
	if err != nil {
		return false, err
	}
	return ready, nil
}
