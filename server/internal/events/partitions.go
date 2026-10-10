package events

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func isNoPartitionErr(err error) bool {
	if err == nil {
		return false
	}
	// Postgres: no partition of relation "trim_events" found for row
	return strings.Contains(strings.ToLower(err.Error()), "no partition of relation")
}

// ensureEventPartitions creates missing UTC month partitions from DB settings.
// Fail-closed: no invent months_ahead when admin_retention_settings row is missing.
func ensureEventPartitions(ctx context.Context, db *pgxpool.Pool) error {
	if db == nil {
		return fmt.Errorf("ADMIN_EVENTS_PARTITION_ENSURE_FAILED")
	}
	var ahead int
	err := db.QueryRow(ctx, `
		select trim_events_partition_months_ahead
		from public.admin_retention_settings where id = 'default'
	`).Scan(&ahead)
	if err == pgx.ErrNoRows || ahead < 1 || ahead > 36 {
		return fmt.Errorf("ADMIN_EVENTS_PARTITION_AHEAD_MISSING")
	}
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `select public.ensure_trim_events_month_partitions($1, 1)`, ahead)
	return err
}
