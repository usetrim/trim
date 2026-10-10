package metrics

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Store persists local proxy trim stats in SQLite under ~/.trim/metrics.db.
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

type Summary struct {
	Requests     int64   `json:"requests"`
	TokensBefore int64   `json:"tokens_before"`
	TokensAfter  int64   `json:"tokens_after"`
	SavedUSD     float64 `json:"saved_usd_est"`
	LastLatency  float64 `json:"last_latency_ms"`
}

// DayPoint is one UTC day of aggregated local proxy usage.
type DayPoint struct {
	Day          string  `json:"day"`
	Requests     int64   `json:"requests"`
	TokensBefore int64   `json:"tokens_before"`
	TokensAfter  int64   `json:"tokens_after"`
	SavedUSD     float64 `json:"saved_usd_est"`
	AvgLatency   float64 `json:"avg_latency_ms"`
}

var (
	global     *Store
	globalOnce sync.Once
	globalErr  error
)

// OpenDefault opens (or returns) the process-wide metrics store.
func OpenDefault() (*Store, error) {
	globalOnce.Do(func() {
		home, err := os.UserHomeDir()
		if err != nil {
			globalErr = err
			return
		}
		dir := filepath.Join(home, ".trim")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			globalErr = err
			return
		}
		global, globalErr = Open(filepath.Join(dir, "metrics.db"))
	})
	return global, globalErr
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		create table if not exists trim_local_events (
			id integer primary key autoincrement,
			created_at text not null,
			model text not null default '',
			tokens_before integer not null,
			tokens_after integer not null,
			latency_ms real not null,
			status text not null,
			mode text not null
		);
		create index if not exists idx_trim_local_events_created on trim_local_events(created_at);
	`)
	if err != nil {
		return err
	}
	// Additive column for older DBs created before mode existed (no invent default on new rows).
	_, _ = s.db.Exec(`alter table trim_local_events add column mode text`)
	return nil
}

// Record appends one proxy trim event.
func (s *Store) Record(model string, before, after int, latencyMs float64, status, mode string) error {
	if s == nil {
		return fmt.Errorf("METRICS_STORE_NIL")
	}
	if mode == "" || status == "" {
		return fmt.Errorf("METRICS_MODE_STATUS_REQUIRED")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		`insert into trim_local_events (created_at, model, tokens_before, tokens_after, latency_ms, status, mode)
		 values (?, ?, ?, ?, ?, ?, ?)`,
		time.Now().UTC().Format(time.RFC3339Nano),
		model,
		before,
		after,
		latencyMs,
		status,
		mode,
	)
	return err
}

// TodaySummary aggregates today's local proxy usage.
// usdPerMTok must come from TRIM_SAVINGS_USD_PER_MTOK (no hardcoded rate).
func (s *Store) TodaySummary(usdPerMTok float64) (Summary, error) {
	var out Summary
	if s == nil {
		return out, fmt.Errorf("METRICS_STORE_NIL")
	}
	if usdPerMTok <= 0 {
		return out, fmt.Errorf("METRICS_SAVINGS_USD_REQUIRED")
	}
	day := time.Now().UTC().Format("2006-01-02")
	row := s.db.QueryRow(`
		select
			count(*),
			coalesce(sum(tokens_before), 0),
			coalesce(sum(tokens_after), 0),
			coalesce(avg(latency_ms), 0)
		from trim_local_events
		where substr(created_at, 1, 10) = ?
	`, day)
	if err := row.Scan(&out.Requests, &out.TokensBefore, &out.TokensAfter, &out.LastLatency); err != nil {
		return out, err
	}
	savedTok := out.TokensBefore - out.TokensAfter
	if savedTok > 0 {
		out.SavedUSD = float64(savedTok) / 1_000_000.0 * usdPerMTok
	}
	return out, nil
}

// Series returns daily aggregates for the last n days (UTC).
func (s *Store) Series(days int, usdPerMTok float64) ([]DayPoint, error) {
	if s == nil {
		return nil, fmt.Errorf("METRICS_STORE_NIL")
	}
	if days <= 0 {
		return nil, fmt.Errorf("METRICS_SERIES_DAYS_REQUIRED")
	}
	if usdPerMTok <= 0 {
		return nil, fmt.Errorf("METRICS_SAVINGS_USD_REQUIRED")
	}
	since := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")
	rows, err := s.db.Query(`
		select substr(created_at, 1, 10) as day,
		       count(*),
		       coalesce(sum(tokens_before), 0),
		       coalesce(sum(tokens_after), 0),
		       coalesce(avg(latency_ms), 0)
		from trim_local_events
		where substr(created_at, 1, 10) >= ?
		group by 1
		order by 1 asc
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]DayPoint, 0)
	for rows.Next() {
		var p DayPoint
		if err := rows.Scan(&p.Day, &p.Requests, &p.TokensBefore, &p.TokensAfter, &p.AvgLatency); err != nil {
			return nil, err
		}
		saved := p.TokensBefore - p.TokensAfter
		if saved > 0 {
			p.SavedUSD = float64(saved) / 1_000_000.0 * usdPerMTok
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
