package emaildenylist

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DomainOf returns the lowercased domain part of an email, or false if malformed.
func DomainOf(email string) (string, bool) {
	addr := strings.TrimSpace(strings.ToLower(email))
	at := strings.LastIndex(addr, "@")
	if at < 1 || at == len(addr)-1 {
		return "", false
	}
	local := addr[:at]
	domain := addr[at+1:]
	if local == "" || domain == "" || strings.Contains(domain, "@") || strings.ContainsAny(domain, " \t\r\n") {
		return "", false
	}
	return domain, true
}

// IsDenied reports whether the email domain is on email_domain_denylist
// or the address is malformed. Prefer DB RPC for a single source of truth.
func IsDenied(ctx context.Context, db *pgxpool.Pool, email string) (bool, error) {
	if db == nil {
		return true, nil
	}
	var denied bool
	err := db.QueryRow(ctx, `select public.email_domain_is_denied($1)`, email).Scan(&denied)
	if err != nil {
		return true, err
	}
	return denied, nil
}
