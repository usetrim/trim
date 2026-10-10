package platformadmin

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usetrim/trim/server/internal/billingsettings"
	"github.com/usetrim/trim/server/internal/events"
)

func (h *Handler) readPool() *pgxpool.Pool {
	if h.ReadDB != nil {
		return h.ReadDB
	}
	return h.DB
}

func (h *Handler) getCachedAdminCharts(ctx context.Context, key string) (events.DashboardCharts, bool) {
	var empty events.DashboardCharts
	ttl, err := billingsettings.ChartCacheTTLSec(ctx, h.readPool())
	if err != nil || h.Redis == nil || ttl < 1 {
		return empty, false
	}
	b, err := h.Redis.Get(ctx, key).Bytes()
	if err != nil || len(b) == 0 {
		return empty, false
	}
	var c events.DashboardCharts
	if err := json.Unmarshal(b, &c); err != nil {
		return empty, false
	}
	return c, true
}

func (h *Handler) setCachedAdminCharts(ctx context.Context, key string, c events.DashboardCharts) {
	ttl, err := billingsettings.ChartCacheTTLSec(ctx, h.readPool())
	if err != nil || h.Redis == nil || ttl < 1 {
		return
	}
	b, err := json.Marshal(c)
	if err != nil {
		log.Printf("admin: chart cache marshal failed: %v", err)
		return
	}
	if err := h.Redis.Set(ctx, key, b, time.Duration(ttl)*time.Second).Err(); err != nil {
		log.Printf("admin: chart cache set failed: %v", err)
	}
}
