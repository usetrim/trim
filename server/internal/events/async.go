package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// HandlerOpts configures the events handler (fail-closed sizes from config.Load).
type HandlerOpts struct {
	DB               *pgxpool.Pool
	ReadDB           *pgxpool.Pool
	Redis            *redis.Client
	InsertTimeoutSec int
	QueueSize        int
	Workers          int
	OutboxPollSec    int
	OutboxBatch      int
}

func NewHandler(opts HandlerOpts) *Handler {
	read := opts.ReadDB
	if read == nil {
		read = opts.DB
	}
	h := &Handler{
		DB:               opts.DB,
		ReadDB:           read,
		Redis:            opts.Redis,
		InsertTimeoutSec: opts.InsertTimeoutSec,
		wake:             make(chan struct{}, opts.QueueSize),
		workers:          opts.Workers,
		outboxPollSec:    opts.OutboxPollSec,
		outboxBatch:      opts.OutboxBatch,
	}
	for i := 0; i < opts.Workers; i++ {
		go h.outboxWorker()
	}
	go h.drainOutbox()
	if opts.OutboxPollSec > 0 {
		go h.outboxPoller()
	}
	return h
}

// RecordAsync writes a durable outbox row then wakes a bounded worker channel.
func (h *Handler) RecordAsync(userID, requestID, model string, before, after, latencyMs int, status, mode, errorCode string) {
	if h == nil || h.DB == nil {
		return
	}
	if userID == "" {
		log.Printf("events: outbox skip (empty user_id)")
		return
	}
	status = strings.TrimSpace(status)
	mode = strings.TrimSpace(mode)
	errorCode = strings.TrimSpace(strings.ToLower(errorCode))
	if status == "" || mode == "" {
		log.Printf("events: outbox skip (empty status/mode) user=%s", userID)
		return
	}
	sec := h.InsertTimeoutSec
	if sec < 1 {
		log.Printf("events: outbox skip (InsertTimeoutSec unset) user=%s", userID)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sec)*time.Second)
	defer cancel()
	_, err := h.DB.Exec(ctx, `
		insert into public.trim_event_outbox (
			user_id, request_id, model, tokens_before, tokens_after, latency_ms, mode, status, error_code
		) values ($1, nullif($2, ''), nullif($3, ''), $4, $5, $6, $7, $8, nullif($9, ''))
	`, userID, requestID, model, before, after, latencyMs, mode, status, errorCode)
	if err != nil {
		log.Printf("events: outbox insert failed (telemetry dropped): %v", err)
		return
	}
	select {
	case h.wake <- struct{}{}:
	default:
		// Channel full: poller / next wake will drain; row is already durable in outbox.
	}
}

func (h *Handler) outboxPoller() {
	sec := h.outboxPollSec
	if sec < 1 {
		return
	}
	t := time.NewTicker(time.Duration(sec) * time.Second)
	defer t.Stop()
	for range t.C {
		h.drainOutbox()
	}
}

func (h *Handler) outboxWorker() {
	for range h.wake {
		h.drainOutbox()
	}
}

func (h *Handler) drainOutbox() {
	if h == nil || h.DB == nil {
		return
	}
	sec := h.InsertTimeoutSec
	if sec < 1 {
		return
	}
	batch := h.outboxBatch
	if batch < 1 {
		return
	}
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sec)*time.Second)
		n, err := h.claimAndInsertBatch(ctx, batch)
		if err != nil && isNoPartitionErr(err) {
			if ensureErr := ensureEventPartitions(ctx, h.DB); ensureErr != nil {
				cancel()
				log.Printf("events: outbox drain partition ensure failed: %v (cause: %v)", ensureErr, err)
				return
			}
			n, err = h.claimAndInsertBatch(ctx, batch)
		}
		cancel()
		if err != nil {
			log.Printf("events: outbox drain failed: %v", err)
			return
		}
		if n == 0 {
			return
		}
	}
}

func (h *Handler) claimAndInsertBatch(ctx context.Context, limit int) (int, error) {
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		with cte as (
			select id from public.trim_event_outbox
			order by id
			limit $1
			for update skip locked
		)
		delete from public.trim_event_outbox o
		using cte
		where o.id = cte.id
		returning o.user_id, o.request_id, o.model, o.tokens_before, o.tokens_after,
		          o.latency_ms, o.mode, o.status, o.error_code
	`, limit)
	if err != nil {
		return 0, err
	}
	type row struct {
		userID, requestID, model, mode, status, errorCode string
		before, after, latency                            int
	}
	batch := make([]row, 0, limit)
	for rows.Next() {
		var r row
		var reqID, model, errCode *string
		if err := rows.Scan(&r.userID, &reqID, &model, &r.before, &r.after, &r.latency, &r.mode, &r.status, &errCode); err != nil {
			rows.Close()
			return 0, err
		}
		if reqID != nil {
			r.requestID = *reqID
		}
		if model != nil {
			r.model = *model
		}
		if errCode != nil {
			r.errorCode = *errCode
		}
		batch = append(batch, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, r := range batch {
		_, err := tx.Exec(ctx, `
			insert into public.trim_events (
				user_id, request_id, model, tokens_before, tokens_after, latency_ms, mode, status, error_code
			) values ($1, nullif($2, ''), nullif($3, ''), $4, $5, $6, $7, $8, nullif($9, ''))
		`, r.userID, r.requestID, r.model, r.before, r.after, r.latency, r.mode, r.status, r.errorCode)
		if err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(batch), nil
}

func (h *Handler) chartCacheKey(userID, groupBy, heatmapScope string, seriesDays, topN int) string {
	return fmt.Sprintf("chart:v3:%s:%s:%s:%d:%d", userID, groupBy, heatmapScope, seriesDays, topN)
}

func (h *Handler) getCachedCharts(ctx context.Context, key string, ttlSec int) (*DashboardCharts, bool) {
	if h.Redis == nil || ttlSec < 1 {
		return nil, false
	}
	b, err := h.Redis.Get(ctx, key).Bytes()
	if err != nil || len(b) == 0 {
		return nil, false
	}
	var c DashboardCharts
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, false
	}
	return &c, true
}

func (h *Handler) setCachedCharts(ctx context.Context, key string, c DashboardCharts, ttlSec int) {
	if h.Redis == nil || ttlSec < 1 {
		return
	}
	b, err := json.Marshal(c)
	if err != nil {
		log.Printf("events: chart cache marshal failed: %v", err)
		return
	}
	if err := h.Redis.Set(ctx, key, b, time.Duration(ttlSec)*time.Second).Err(); err != nil {
		log.Printf("events: chart cache set failed: %v", err)
	}
}
