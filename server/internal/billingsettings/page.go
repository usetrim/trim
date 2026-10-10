package billingsettings

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MaxPageSize returns billing_settings.max_page_size for skip/limit APIs.
// Fail-closed: missing row or non-positive value is an error (no invent 50/100).
func MaxPageSize(ctx context.Context, db *pgxpool.Pool) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	var n int
	err := db.QueryRow(ctx, `
		select coalesce(max_page_size, 0)
		from public.billing_settings
		where id = 'default'
	`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	if n < 1 {
		return 0, fmt.Errorf("PAGINATION_MAX_LIMIT_INVALID")
	}
	return n, nil
}

// SkipToMaxPages returns billing_settings.pagination_skip_to_max_pages.
// Fail-closed: missing or non-positive is an error (no invent 500).
func SkipToMaxPages(ctx context.Context, db *pgxpool.Pool) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	var n int
	err := db.QueryRow(ctx, `
		select coalesce(pagination_skip_to_max_pages, 0)
		from public.billing_settings
		where id = 'default'
	`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	if n < 1 {
		return 0, fmt.Errorf("PAGINATION_SKIP_TO_MAX_INVALID")
	}
	return n, nil
}

// DefaultSeatQuantity returns billing_settings.default_seat_quantity for free/pro workspace create.
// Fail-closed: missing or non-positive is an error (no invent 1).
func DefaultSeatQuantity(ctx context.Context, db *pgxpool.Pool) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	var n int
	err := db.QueryRow(ctx, `
		select coalesce(default_seat_quantity, 0)
		from public.billing_settings
		where id = 'default'
	`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	if n < 1 {
		return 0, fmt.Errorf("DEFAULT_SEAT_QUANTITY_INVALID")
	}
	return n, nil
}

// ChartTopN returns billing_settings.chart_top_n for breakdown / heatmap caps.
// Fail-closed: missing or non-positive is an error (no invent 25/100).
func ChartTopN(ctx context.Context, db *pgxpool.Pool) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	var n int
	err := db.QueryRow(ctx, `
		select coalesce(chart_top_n, 0)
		from public.billing_settings
		where id = 'default'
	`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	if n < 1 {
		return 0, fmt.Errorf("CHART_TOP_N_MISSING")
	}
	return n, nil
}

// ChartSeriesDays returns billing_settings.chart_series_days for usage/heatmap windows.
// Fail-closed: missing or non-positive is an error (no invent 30/365).
func ChartSeriesDays(ctx context.Context, db *pgxpool.Pool) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	var n int
	err := db.QueryRow(ctx, `
		select coalesce(chart_series_days, 0)
		from public.billing_settings
		where id = 'default'
	`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	if n < 1 {
		return 0, fmt.Errorf("CHART_SERIES_DAYS_MISSING")
	}
	return n, nil
}

// ChartCacheTTLSec returns billing_settings.chart_cache_ttl_sec for Redis chart JSON cache.
// Fail-closed: missing or non-positive is an error (no invent TTL).
func ChartCacheTTLSec(ctx context.Context, db *pgxpool.Pool) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	var n int
	err := db.QueryRow(ctx, `
		select coalesce(chart_cache_ttl_sec, 0)
		from public.billing_settings
		where id = 'default'
	`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	if n < 1 {
		return 0, fmt.Errorf("CHART_CACHE_TTL_MISSING")
	}
	return n, nil
}
