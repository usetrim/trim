package middleware

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

const runtimeOverridesRedisKey = "admin:runtime:overrides"

// RuntimeOverrides are optional operator DB overrides of env fraud/rate settings.
// Zero / missing numeric fields mean "use Config env values" (fail closed to env, never invent).
// MinCLIVersion empty means use Config.MinCLIVersion.
type RuntimeOverrides struct {
	RateLimitIP            int    `json:"rate_limit_ip_per_min,omitempty"`
	RateLimitUser          int    `json:"rate_limit_user_per_min,omitempty"`
	PoWDifficulty          int    `json:"pow_difficulty,omitempty"`
	MaxAccountsHW          int    `json:"max_accounts_per_hardware,omitempty"`
	MaxAccountsJA4         int    `json:"max_accounts_per_ja4,omitempty"`
	CFThreatScoreMin       int    `json:"cf_threat_score_min,omitempty"`
	FastBalancedMinLines   int    `json:"fast_balanced_min_lines,omitempty"`
	FastAggressiveMinLines int    `json:"fast_aggressive_min_lines,omitempty"`
	FastMildMinLines       int    `json:"fast_mild_min_lines,omitempty"`
	MinCLIVersion          string `json:"min_cli_version,omitempty"`
}

var (
	overridesMu    sync.RWMutex
	overridesCache *RuntimeOverrides
	overridesAt    time.Time
)

// PublishRuntimeOverrides writes overrides to Redis and local cache.
func PublishRuntimeOverrides(ctx context.Context, rdb *redis.Client, o RuntimeOverrides) error {
	if rdb == nil {
		overridesMu.Lock()
		overridesCache = &o
		overridesAt = time.Now()
		overridesMu.Unlock()
		return nil
	}
	b, err := json.Marshal(o)
	if err != nil {
		return err
	}
	if err := rdb.Set(ctx, runtimeOverridesRedisKey, b, 0).Err(); err != nil {
		return err
	}
	overridesMu.Lock()
	overridesCache = &o
	overridesAt = time.Now()
	overridesMu.Unlock()
	return nil
}

// LoadRuntimeOverridesFromDB reads nullable admin_product_settings override columns.
func LoadRuntimeOverridesFromDB(ctx context.Context, db *pgxpool.Pool) (RuntimeOverrides, error) {
	var o RuntimeOverrides
	var ip, user, pow, hw, ja4, cf, bal, agg, mild *int
	var minCLI *string
	err := db.QueryRow(ctx, `
		select rate_limit_ip_per_min, rate_limit_user_per_min, pow_difficulty,
		       max_accounts_per_hardware, max_accounts_per_ja4, cf_threat_score_min,
		       fast_balanced_min_lines, fast_aggressive_min_lines, fast_mild_min_lines,
		       nullif(btrim(coalesce(min_cli_version, '')), '')
		from public.admin_product_settings where id = 'default'
	`).Scan(&ip, &user, &pow, &hw, &ja4, &cf, &bal, &agg, &mild, &minCLI)
	if err != nil {
		return o, err
	}
	if ip != nil {
		o.RateLimitIP = *ip
	}
	if user != nil {
		o.RateLimitUser = *user
	}
	if pow != nil {
		o.PoWDifficulty = *pow
	}
	if hw != nil {
		o.MaxAccountsHW = *hw
	}
	if ja4 != nil {
		o.MaxAccountsJA4 = *ja4
	}
	if cf != nil {
		o.CFThreatScoreMin = *cf
	}
	if bal != nil {
		o.FastBalancedMinLines = *bal
	}
	if agg != nil {
		o.FastAggressiveMinLines = *agg
	}
	if mild != nil {
		o.FastMildMinLines = *mild
	}
	if minCLI != nil {
		o.MinCLIVersion = strings.TrimSpace(*minCLI)
	}
	return o, nil
}

func (m *AuthQuota) effectiveOverrides() RuntimeOverrides {
	overridesMu.RLock()
	ttl := runtimeOverridesCacheTTL()
	if overridesCache != nil && ttl > 0 && time.Since(overridesAt) < ttl {
		o := *overridesCache
		overridesMu.RUnlock()
		return o
	}
	overridesMu.RUnlock()
	if m.Redis == nil {
		return RuntimeOverrides{}
	}
	b, err := m.Redis.Get(context.Background(), runtimeOverridesRedisKey).Bytes()
	if err != nil || len(b) == 0 {
		return RuntimeOverrides{}
	}
	var o RuntimeOverrides
	if json.Unmarshal(b, &o) != nil {
		return RuntimeOverrides{}
	}
	overridesMu.Lock()
	overridesCache = &o
	overridesAt = time.Now()
	overridesMu.Unlock()
	return o
}

func (m *AuthQuota) effRateLimitIP() int {
	if o := m.effectiveOverrides(); o.RateLimitIP > 0 {
		return o.RateLimitIP
	}
	return m.Config.RateLimitPerIPPerMin
}

func (m *AuthQuota) effRateLimitUser() int {
	if o := m.effectiveOverrides(); o.RateLimitUser > 0 {
		return o.RateLimitUser
	}
	return m.Config.RateLimitPerUserPerMin
}

func (m *AuthQuota) effPoW() int {
	if o := m.effectiveOverrides(); o.PoWDifficulty > 0 {
		return o.PoWDifficulty
	}
	return m.Config.FreeTierPoWDifficulty
}

func (m *AuthQuota) effMaxHW() int {
	if o := m.effectiveOverrides(); o.MaxAccountsHW > 0 {
		return o.MaxAccountsHW
	}
	return m.Config.MaxAccountsPerHardware
}

func (m *AuthQuota) effMaxJA4() int {
	if o := m.effectiveOverrides(); o.MaxAccountsJA4 > 0 {
		return o.MaxAccountsJA4
	}
	return m.Config.MaxAccountsPerJA4
}

func (m *AuthQuota) effCFThreat() int {
	if o := m.effectiveOverrides(); o.CFThreatScoreMin > 0 {
		return o.CFThreatScoreMin
	}
	return m.Config.CFThreatScoreMin
}

func (m *AuthQuota) effMinCLIVersion() string {
	if o := m.effectiveOverrides(); strings.TrimSpace(o.MinCLIVersion) != "" {
		return strings.TrimSpace(o.MinCLIVersion)
	}
	return strings.TrimSpace(m.Config.MinCLIVersion)
}

// CurrentRuntimeOverrides returns cached overrides, refreshing from Redis when stale.
func CurrentRuntimeOverrides() RuntimeOverrides {
	overridesMu.RLock()
	ttl := runtimeOverridesCacheTTL()
	if overridesCache != nil && ttl > 0 && time.Since(overridesAt) < ttl {
		o := *overridesCache
		overridesMu.RUnlock()
		return o
	}
	cached := overridesCache
	overridesMu.RUnlock()
	if cached != nil {
		return *cached
	}
	return RuntimeOverrides{}
}

// runtimeOverridesCacheTTL reads RUNTIME_OVERRIDES_CACHE_MS. Fail-closed: 0 = do not invent TTL.
func runtimeOverridesCacheTTL() time.Duration {
	raw := strings.TrimSpace(subscriptions.MessageForCode("RUNTIME_OVERRIDES_CACHE_MS"))
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0
	}
	return time.Duration(n) * time.Millisecond
}

func (m *AntiFraud) effRateLimitIP() int {
	// AntiFraud shares the same Redis override cache as AuthQuota.
	aq := &AuthQuota{Redis: m.Redis, Config: m.Config}
	return aq.effRateLimitIP()
}

func (m *AntiFraud) effCFThreat() int {
	aq := &AuthQuota{Redis: m.Redis, Config: m.Config}
	return aq.effCFThreat()
}
