package dbpool

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Open creates a pgx pool with explicit max/min conns (no silent invent).
// maxConns and minConns must already be validated by config.Load.
// Uses QueryExecModeExec so Supabase/PgBouncer transaction poolers do not
// hit "prepared statement already exists" (SQLSTATE 42P05).
func Open(ctx context.Context, databaseURL string, maxConns, minConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	if maxConns < 1 {
		return nil, fmt.Errorf("maxConns must be >= 1")
	}
	if minConns < 0 || minConns > maxConns {
		return nil, fmt.Errorf("minConns must be 0..maxConns")
	}
	cfg.MaxConns = maxConns
	cfg.MinConns = minConns
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
