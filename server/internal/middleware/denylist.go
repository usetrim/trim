package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// enforcePostgresDenylist blocks requests whose client IP or CF-ASN is in
// public.ip_denylist / public.asn_denylist. Results are cached in Redis.
// Cloud: DB lookup errors fail closed (DENYLIST_UNAVAILABLE).
// Self-host: log and skip (other layers still apply). Positive matches always deny.
func (m *AntiFraud) enforcePostgresDenylist(w http.ResponseWriter, r *http.Request, ip string) bool {
	if m.DB == nil || ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return false
	}
	if net.ParseIP(ip) == nil {
		return false
	}

	ctx := r.Context()
	cloud := strings.EqualFold(strings.TrimSpace(m.Config.DeploymentMode), "cloud")

	if denied, cached := m.denylistCached(ctx, "denylist:ip:"+ip); cached {
		if denied {
			writeDenylistForbidden(w, "IP_DENIED")
			return true
		}
	} else {
		denied, err := m.denylistIPQuery(ctx, ip)
		if err != nil {
			log.Printf("trim: ip_denylist lookup failed ip=%s: %v", ip, err)
			if cloud {
				writeDenylistForbidden(w, "DENYLIST_UNAVAILABLE")
				return true
			}
		} else {
			m.cacheDenylistBool(ctx, "denylist:ip:"+ip, denied)
			if denied {
				writeDenylistForbidden(w, "IP_DENIED")
				return true
			}
		}
	}

	asn := normalizeASN(firstNonEmpty(r.Header.Get("CF-ASN"), r.Header.Get("X-ASN")))
	if asn == "" {
		asn = lookupGeoLiteASN(ip)
	}
	if asn == "" {
		return false
	}
	if denied, cached := m.denylistCached(ctx, "denylist:asn:"+asn); cached {
		if denied {
			writeDenylistForbidden(w, "ASN_DENIED")
			return true
		}
		return false
	}
	denied, err := m.denylistASNQuery(ctx, asn)
	if err != nil {
		log.Printf("trim: asn_denylist lookup failed asn=%s: %v", asn, err)
		if cloud {
			writeDenylistForbidden(w, "DENYLIST_UNAVAILABLE")
			return true
		}
		return false
	}
	m.cacheDenylistBool(ctx, "denylist:asn:"+asn, denied)
	if denied {
		writeDenylistForbidden(w, "ASN_DENIED")
		return true
	}
	return false
}

func writeDenylistForbidden(w http.ResponseWriter, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":  code,
		"error": siteMsg(code),
	})
}

func normalizeASN(raw string) string {
	raw = strings.TrimSpace(strings.ToUpper(raw))
	raw = strings.TrimPrefix(raw, "AS")
	if raw == "" {
		return ""
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 1 {
		return ""
	}
	return strconv.FormatInt(n, 10)
}

func (m *AntiFraud) denylistCacheTTL() time.Duration {
	// Reuse IP-risk TTL so cloud installs need no new env (fail-closed config surface).
	return redisSec(m.Config.RedisIPRiskTTLSec)
}

func (m *AntiFraud) denylistCached(ctx context.Context, key string) (denied bool, ok bool) {
	if m.Redis == nil {
		return false, false
	}
	v, err := m.Redis.Get(ctx, key).Result()
	if err == redis.Nil || err != nil {
		return false, false
	}
	return v == "1", true
}

func (m *AntiFraud) cacheDenylistBool(ctx context.Context, key string, denied bool) {
	if m.Redis == nil {
		return
	}
	val := "0"
	if denied {
		val = "1"
	}
	_ = m.Redis.Set(ctx, key, val, m.denylistCacheTTL()).Err()
}

func (m *AntiFraud) denylistIPQuery(ctx context.Context, ip string) (bool, error) {
	var denied bool
	err := m.DB.QueryRow(ctx, `
		select exists(
			select 1 from public.ip_denylist
			where $1::inet <<= cidr
		)
	`, ip).Scan(&denied)
	return denied, err
}

func (m *AntiFraud) denylistASNQuery(ctx context.Context, asn string) (bool, error) {
	var denied bool
	err := m.DB.QueryRow(ctx, `
		select exists(
			select 1 from public.asn_denylist
			where asn = $1::bigint
		)
	`, asn).Scan(&denied)
	return denied, err
}

// lookupGeoLiteASN returns the numeric ASN string for ip when GeoLite ASN MMDB is loaded.
func lookupGeoLiteASN(ip string) string {
	geoMu.RLock()
	defer geoMu.RUnlock()
	if !geoReady || geoASNDB == nil {
		return ""
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	var rec asnRecord
	if err := geoASNDB.Lookup(parsed, &rec); err != nil || rec.AutonomousSystemNumber < 1 {
		return ""
	}
	return strconv.FormatUint(uint64(rec.AutonomousSystemNumber), 10)
}

// InvalidateDenylistCaches clears cached allow/deny entries after admin edits.
// Prefer FlushDenylistCaches on CIDR changes (one CIDR maps to many client IPs).
func InvalidateDenylistCaches(rdb *redis.Client, ipOrEmpty, asnOrEmpty string) {
	if rdb == nil {
		return
	}
	ctx := context.Background()
	if ip := strings.TrimSpace(ipOrEmpty); ip != "" {
		_ = rdb.Del(ctx, "denylist:ip:"+ip).Err()
	}
	if asn := normalizeASN(asnOrEmpty); asn != "" {
		_ = rdb.Del(ctx, "denylist:asn:"+asn).Err()
	}
}

// FlushDenylistCaches drops all denylist:* Redis keys (safe after CIDR upsert/delete).
func FlushDenylistCaches(rdb *redis.Client) {
	if rdb == nil {
		return
	}
	ctx := context.Background()
	var cursor uint64
	for {
		keys, next, err := rdb.Scan(ctx, cursor, "denylist:*", 100).Result()
		if err != nil {
			log.Printf("trim: denylist cache flush scan: %v", err)
			return
		}
		if len(keys) > 0 {
			_ = rdb.Del(ctx, keys...).Err()
		}
		cursor = next
		if cursor == 0 {
			return
		}
	}
}
