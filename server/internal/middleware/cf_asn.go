package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func redisSec(sec int) time.Duration {
	if sec < 1 {
		return 0
	}
	return time.Duration(sec) * time.Second
}

// enrichIPRiskFromCloudflare seeds Redis ip_risk from Cloudflare edge headers.
// On cache miss it still kicks an async warm lookup; free-tier gates call EnsureIPRiskSync.
func (m *AntiFraud) enrichIPRiskFromCloudflare(r *http.Request, ip string) {
	if m.Redis == nil || ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return
	}
	ctx := r.Context()
	cacheKey := "ip_risk:proxy:" + ip
	if _, err := m.Redis.Get(ctx, cacheKey).Result(); err == nil {
		return
	}

	threatMin := m.effCFThreat()
	if threatMin > 0 {
		if seedEdgeIPRisk(m.Redis, ctx, r, ip, threatMin, m.Config.RedisIPRiskTTLSec, m.Config.RedisASNCheckedTTLSec) {
			return
		}
	}

	// Warm cache for later requests (AuthQuota uses EnsureIPRiskSync on free tier).
	// Fail closed: no invented default timeout. Skip remote lookup when unset (self-host).
	timeoutMs := m.Config.IPRiskLookupTimeoutMs
	if timeoutMs <= 0 {
		return
	}
	go lookupIPRisk(m.Redis, ip, time.Duration(timeoutMs)*time.Millisecond, m.Config.RedisIPRiskTTLSec, m.Config.RedisASNCheckedTTLSec, m.Config.RedisASNMissTTLSec)
}

// EnsureIPRiskSync resolves VPN/proxy/hosting risk before free-tier allow/deny.
// Returns true when the IP is known risky. Uses Redis, then CF edge, then sync ipwho.is.
func EnsureIPRiskSync(rdb *redis.Client, r *http.Request, ip string, threatMin, timeoutMs, riskTTLSec, asnTTLSec, asnMissTTLSec int) bool {
	if rdb == nil || ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return false
	}
	ctx := r.Context()
	cacheKey := "ip_risk:proxy:" + ip
	if isProxy, err := rdb.Get(ctx, cacheKey).Bool(); err == nil {
		return isProxy
	}
	if threatMin > 0 {
		if seedEdgeIPRisk(rdb, ctx, r, ip, threatMin, riskTTLSec, asnTTLSec) {
			if isProxy, err := rdb.Get(ctx, cacheKey).Bool(); err == nil {
				return isProxy
			}
			return true
		}
	}
	// Prefer offline GeoLite when mounted (no third-party round trip).
	if risky, ok := lookupGeoLiteRisk(ip); ok {
		_ = rdb.Set(ctx, cacheKey, risky, redisSec(riskTTLSec)).Err()
		_ = rdb.Set(ctx, "ip_risk:asn_checked:"+ip, "1", redisSec(asnTTLSec)).Err()
		return risky
	}
	// Fail closed: without an explicit timeout, do not invent a remote lookup window.
	if timeoutMs <= 0 {
		return false
	}
	lookupIPRisk(rdb, ip, time.Duration(timeoutMs)*time.Millisecond, riskTTLSec, asnTTLSec, asnMissTTLSec)
	if isProxy, err := rdb.Get(ctx, cacheKey).Bool(); err == nil {
		return isProxy
	}
	return false
}

func seedEdgeIPRisk(rdb *redis.Client, ctx context.Context, r *http.Request, ip string, threatMin, riskTTLSec, asnTTLSec int) bool {
	cacheKey := "ip_risk:proxy:" + ip
	asnCache := "ip_risk:asn_checked:" + ip
	riskTTL := redisSec(riskTTLSec)
	asnTTL := redisSec(asnTTLSec)

	if score := strings.TrimSpace(r.Header.Get("CF-Threat-Score")); score != "" {
		if n, err := strconv.Atoi(score); err == nil && n >= threatMin {
			_ = rdb.Set(ctx, cacheKey, true, riskTTL).Err()
			_ = rdb.Set(ctx, asnCache, "1", asnTTL).Err()
			return true
		}
	}
	if strings.EqualFold(r.Header.Get("CF-IPCountry"), "T1") {
		_ = rdb.Set(ctx, cacheKey, true, riskTTL).Err()
		_ = rdb.Set(ctx, asnCache, "1", asnTTL).Err()
		return true
	}
	if asn := strings.TrimSpace(r.Header.Get("CF-ASN")); asn != "" {
		if asnBlockedOffline(asn) {
			_ = rdb.Set(ctx, cacheKey, true, riskTTL).Err()
			_ = rdb.Set(ctx, asnCache, "1", asnTTL).Err()
			return true
		}
	}
	return false
}

func lookupIPRisk(rdb *redis.Client, ip string, timeout time.Duration, riskTTLSec, asnTTLSec, asnMissTTLSec int) {
	if rdb == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	asnCache := "ip_risk:asn_checked:" + ip
	cacheKey := "ip_risk:proxy:" + ip
	riskTTL := redisSec(riskTTLSec)
	asnTTL := redisSec(asnTTLSec)
	missTTL := redisSec(asnMissTTLSec)

	if ok, _ := rdb.Exists(ctx, asnCache).Result(); ok > 0 {
		return
	}

	if risky, resolved := lookupGeoLiteRisk(ip); resolved {
		_ = rdb.Set(ctx, cacheKey, risky, riskTTL).Err()
		_ = rdb.Set(ctx, asnCache, "1", asnTTL).Err()
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ipwho.is/"+ip, nil)
	if err != nil {
		_ = rdb.Set(ctx, asnCache, "1", missTTL).Err()
		return
	}
	req.Header.Set("User-Agent", "TrimAntiFraud/1.0")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		_ = rdb.Set(ctx, asnCache, "1", missTTL).Err()
		return
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 64*1024))
	if err != nil {
		_ = rdb.Set(ctx, asnCache, "1", missTTL).Err()
		return
	}

	var payload struct {
		Success    bool `json:"success"`
		Connection struct {
			ASN int    `json:"asn"`
			Org string `json:"org"`
			ISP string `json:"isp"`
		} `json:"connection"`
		Security struct {
			VPN     bool `json:"vpn"`
			Proxy   bool `json:"proxy"`
			Tor     bool `json:"tor"`
			Hosting bool `json:"hosting"`
		} `json:"security"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || !payload.Success {
		_ = rdb.Set(ctx, asnCache, "1", missTTL).Err()
		return
	}

	// Provider security flags only. ASN hosting blocklist comes from offline
	// TRIM_GEOLITE_* files (asnBlockedOffline), never a hardcoded invent map.
	risky := payload.Security.VPN || payload.Security.Proxy || payload.Security.Tor || payload.Security.Hosting
	if !risky && payload.Connection.ASN > 0 {
		asn := fmt.Sprintf("%d", payload.Connection.ASN)
		risky = asnBlockedOffline(asn)
	}
	if risky {
		_ = rdb.Set(ctx, cacheKey, true, riskTTL).Err()
	} else {
		_ = rdb.Set(ctx, cacheKey, false, riskTTL).Err()
	}
	_ = rdb.Set(ctx, asnCache, "1", asnTTL).Err()
}
