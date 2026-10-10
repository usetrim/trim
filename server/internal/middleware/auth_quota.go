package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/usetrim/trim/server/internal/authsettings"
	"github.com/usetrim/trim/server/internal/config"
	"github.com/usetrim/trim/server/internal/subscriptions"
	"github.com/usetrim/trim/server/pkg/clisign"
	"github.com/usetrim/trim/server/pkg/metrics"
	"github.com/usetrim/trim/server/pkg/pow"
)

type AuthQuota struct {
	DB      *pgxpool.Pool
	Redis   *redis.Client
	Config  config.Config
	Subs    *subscriptions.Service
	Metrics *metrics.Registry
}

func NewAuthQuota(db *pgxpool.Pool, rdb *redis.Client, cfg config.Config) *AuthQuota {
	return &AuthQuota{
		DB:     db,
		Redis:  rdb,
		Config: cfg,
		Subs:   subscriptions.NewService(db, nil),
	}
}

func (m *AuthQuota) redisTTL(sec int) time.Duration {
	if sec < 1 {
		return 0
	}
	return time.Duration(sec) * time.Second
}

func (m *AuthQuota) expireRedis(ctx context.Context, key string, sec int) {
	if m.Redis == nil || sec < 1 {
		return
	}
	_ = m.Redis.Expire(ctx, key, m.redisTTL(sec)).Err()
}

// OptionalAuth attaches user identity when a Bearer token is present, without requiring auth.
// Used for public catalog endpoints that annotate upgrade eligibility for signed-in users.
func (m *AuthQuota) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.Config.DeploymentMode == "self" {
			next.ServeHTTP(w, r)
			return
		}
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			next.ServeHTTP(w, r)
			return
		}
		rawToken := strings.TrimPrefix(authHeader, "Bearer ")
		userID, err := m.resolveIdentity(r.Context(), rawToken, "", "")
		if err != nil || userID == "" {
			next.ServeHTTP(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthQuota) Secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.Config.DeploymentMode == "self" {
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			m.fail(w, http.StatusUnauthorized, siteMsg("AUTH_TOKEN_MISSING"))
			return
		}
		rawToken := strings.TrimPrefix(authHeader, "Bearer ")
		hardwareUUID := strings.TrimSpace(r.Header.Get("X-Hardware-UUID"))
		agentID := strings.TrimSpace(r.Header.Get("X-Trim-Agent-Id"))
		clientVersion := r.Header.Get("X-Client-Version")
		workspaceID := strings.TrimSpace(r.Header.Get("X-Workspace-Id"))
		isAPIKey := strings.Count(rawToken, ".") != 2

		// CLI API keys must send X-Client-Version so MIN_CLI_VERSION kill-switch cannot be skipped.
		// IDE (X-Trim-Agent-Id: ide) uses trim-ide/<semver> and is not gated by MIN_CLI_VERSION.
		minCLI := m.effMinCLIVersion()
		if isAPIKey && minCLI != "" && !strings.EqualFold(agentID, "ide") {
			if clientVersion == "" {
				m.fail(w, http.StatusUpgradeRequired, siteMsg("CLI_VERSION_HEADER_MISSING"))
				return
			}
			if isOutdated(clientVersion, minCLI) {
				m.fail(w, http.StatusUpgradeRequired, siteMsg("CLI_VERSION_OUTDATED"))
				return
			}
		}

		if prefix := strings.TrimSpace(m.Config.CanaryAPIKeyPrefix); prefix != "" && strings.HasPrefix(rawToken, prefix) {
			m.triggerCanary(r, hardwareUUID)
			m.fail(w, http.StatusForbidden, siteMsg("AUTH_CREDENTIALS_INVALID"))
			return
		}

		userID, err := m.resolveIdentity(ctx, rawToken, hardwareUUID, agentID)
		isAccountPurge := r.Method == http.MethodDelete &&
			(r.URL.Path == "/api/v1/me/account" || strings.HasSuffix(r.URL.Path, "/me/account"))
		if err != nil || userID == "" {
			if err != nil && isAccountPurge && strings.Contains(err.Error(), "auth provider not allowed") {
				// Allow GDPR delete for just-created sessions from a disallowed OAuth provider.
				if sub, resolveErr := m.resolveJWTSubjectOnly(rawToken); resolveErr == nil && sub != "" {
					userID = sub
					err = nil
				}
			}
			if err != nil || userID == "" {
				if err != nil && strings.Contains(err.Error(), "auth provider not allowed") {
					m.fail(w, http.StatusForbidden, m.authProviderDeniedMessage(ctx))
					return
				}
				if err != nil {
					switch {
					case errors.Is(err, errHardwareMismatch):
						m.fail(w, http.StatusForbidden, siteMsg("HARDWARE_UUID_MISMATCH"))
						return
					case errors.Is(err, errHardwareBoundRequired):
						m.fail(w, http.StatusBadRequest, siteMsg("HARDWARE_UUID_BOUND_REQUIRED"))
						return
					case errors.Is(err, errHardwareUnbound):
						m.fail(w, http.StatusForbidden, siteMsg("API_KEY_DEVICE_UNBOUND"))
						return
					case errors.Is(err, errHardwareDeviceLimit):
						m.fail(w, http.StatusForbidden, siteMsg("HARDWARE_DEVICE_LIMIT"))
						return
					case errors.Is(err, errAPIKeyMaxDevicesMissing):
						m.fail(w, http.StatusServiceUnavailable, siteMsg("API_KEY_MAX_DEVICES_MISSING"))
						return
					case errors.Is(err, errAgentRequired):
						m.fail(w, http.StatusBadRequest, siteMsg("AGENT_ID_REQUIRED"))
						return
					case errors.Is(err, errAgentUnknown):
						m.fail(w, http.StatusBadRequest, siteMsg("AGENT_ID_UNKNOWN"))
						return
					case errors.Is(err, errAgentMismatch):
						m.fail(w, http.StatusForbidden, siteMsg("AGENT_ID_MISMATCH"))
						return
					}
				}
				m.fail(w, http.StatusUnauthorized, siteMsg("AUTH_CREDENTIALS_REVOKED"))
				return
			}
		}

		// Operator force-logout: Redis stores unix cutover; JWTs issued at/before it fail.
		// Fresh sign-in (newer iat) passes. API keys were revoked in the admin action.
		if m.Redis != nil && !isAPIKey {
			cutRaw, rerr := m.Redis.Get(ctx, "admin:force_logout:"+userID).Result()
			if rerr == nil {
				cut, perr := strconv.ParseInt(strings.TrimSpace(cutRaw), 10, 64)
				if perr == nil && cut > 0 {
					iat := jwtIssuedAt(rawToken)
					if iat > 0 && iat <= cut {
						m.fail(w, http.StatusUnauthorized, siteMsg("AUTH_FORCE_LOGOUT"))
						return
					}
				}
			}
		}

		// Account self-delete must not be blocked by free-tier VPN / hardware / PoW gates.
		if isAccountPurge {
			ctx = context.WithValue(ctx, ctxUserID{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Platform account_status gate (suspended/banned/shadowbanned). Fail closed on empty.
		var accountStatus string
		err = m.DB.QueryRow(ctx, `select coalesce(account_status, '') from public.profiles where id = $1::uuid`, userID).Scan(&accountStatus)
		if err != nil || strings.TrimSpace(accountStatus) == "" {
			m.fail(w, http.StatusForbidden, siteMsg("AUTH_ACCOUNT_DISABLED"))
			return
		}
		switch accountStatus {
		case "active":
			// ok
		case "suspended", "banned", "shadowbanned", "pending_delete":
			m.fail(w, http.StatusForbidden, siteMsg("AUTH_ACCOUNT_DISABLED"))
			return
		default:
			m.fail(w, http.StatusForbidden, siteMsg("AUTH_ACCOUNT_DISABLED"))
			return
		}
		// Best-effort last login country (CF only; never invent).
		if cc := strings.TrimSpace(r.Header.Get("CF-IPCountry")); cc != "" && len(cc) <= 8 {
			_, _ = m.DB.Exec(ctx, `
				update public.profiles
				set last_login_country = $2, last_login_at = now(), updated_at = now()
				where id = $1::uuid
			`, userID, strings.ToUpper(cc))
		}

		// Enforce subscription period end on every protected request (server-side source of truth).
		// Fail closed on expire errors: never leave stale paid Redis/DB quotas in force.
		if m.Subs != nil {
			changed, expErr := m.Subs.ExpireIfNeeded(ctx, userID)
			if expErr != nil {
				log.Printf("trim: ExpireIfNeeded user=%s: %v", userID, expErr)
				m.fail(w, http.StatusServiceUnavailable, siteMsg("DATABASE_UNAVAILABLE"))
				return
			}
			if changed {
				m.invalidateQuotaCaches(ctx, userID)
			}
		}

		planTier := m.planTier(ctx, userID)
		ua := r.Header.Get("User-Agent")
		isCLI := strings.Contains(ua, "TrimCLI/")

		// Per-user velocity (Redis minute buckets). 0 disables.
		if m.effRateLimitUser() > 0 && m.Redis != nil {
			if !m.allowVelocity(ctx, "rl:user:"+userID, m.effRateLimitUser()) {
				m.fail(w, http.StatusTooManyRequests, siteMsg("RATE_LIMIT_USER"))
				return
			}
		}

		// Free tier only: block known VPN / proxy / hosting IPs.
		// Sync edge + ipwho lookup so the first request is not a free pass while cache warms.
		if isFreePlan(planTier) {
			ip := clientIP(r)
			if isSyncEdgeProxy(r, m.effCFThreat()) {
				m.fail(w, http.StatusForbidden, siteMsg("FREE_TIER_PROXY_BLOCKED"))
				return
			}
			if EnsureIPRiskSync(m.Redis, r, ip, m.effCFThreat(), m.Config.IPRiskLookupTimeoutMs, m.Config.RedisIPRiskTTLSec, m.Config.RedisASNCheckedTTLSec, m.Config.RedisASNMissTTLSec) {
				m.fail(w, http.StatusForbidden, siteMsg("FREE_TIER_PROXY_BLOCKED"))
				return
			}
		}

		// Free-tier TrimCLI must send hardware UUID (fail closed; cannot skip account graph).
		if isFreePlan(planTier) && isCLI && hardwareUUID == "" {
			m.fail(w, http.StatusBadRequest, siteMsg("HARDWARE_UUID_REQUIRED"))
			return
		}

		// Free tier only: cap distinct accounts per physical machine (Postgres source of truth).
		if hardwareUUID != "" {
			if err := m.recordHardware(ctx, userID, hardwareUUID); err != nil {
				log.Printf("trim: device fingerprint persist failed: %v", err)
			}
			if isFreePlan(planTier) {
				count, err := m.countHardwareAccounts(ctx, hardwareUUID)
				maxHW := m.effMaxHW()
				if maxHW < 1 {
					// Fail closed: unset limit means do not invent a floor; skip graph block.
				} else if err == nil && int(count) > maxHW {
					m.fail(w, http.StatusForbidden, siteMsg("HARDWARE_ACCOUNT_LIMIT"))
					return
				}
				// Keep Redis warm for fast path on subsequent requests.
				if m.Redis != nil {
					hwKey := "hw_accounts:" + hardwareUUID
					_ = m.Redis.SAdd(ctx, hwKey, userID).Err()
					m.expireRedis(ctx, hwKey, m.Config.RedisHardwareGraphTTLSec)
				}
			} else if m.Redis != nil {
				hwKey := "hw_accounts:" + hardwareUUID
				_ = m.Redis.SAdd(ctx, hwKey, userID).Err()
				m.expireRedis(ctx, hwKey, m.Config.RedisHardwareGraphTTLSec)
			}
		}

		// Free tier: JA4/JA3 TLS fingerprint multi-account graph (edge-forwarded fingerprints).
		if isFreePlan(planTier) {
			if blocked, reason := m.enforceJA4AccountGraph(ctx, r, userID); blocked {
				m.fail(w, http.StatusForbidden, reason)
				return
			}
		}

		if shouldMeterPath(r.URL.Path, r.Method) {
			// Free-tier Hashcash for TrimCLI clients (anti-automation). IDE UAs skip PoW.
			if isFreePlan(planTier) && m.effPoW() > 0 && isCLI {
				if !m.requireFreeTierPoW(w, r, userID) {
					return
				}
			}
			if workspaceID != "" {
				if err := m.consumeWorkspaceCredit(ctx, w, userID, workspaceID); err != nil {
					return
				}
			} else if err := m.consumeCredit(ctx, w, userID); err != nil {
				return
			}
		}

		ctx = context.WithValue(ctx, ctxUserID{}, userID)
		if workspaceID != "" {
			ctx = context.WithValue(ctx, ctxWorkspaceID{}, workspaceID)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthQuota) recordHardware(ctx context.Context, userID, hardwareUUID string) error {
	if m.DB == nil {
		return nil
	}
	_, err := m.DB.Exec(ctx, `
		insert into public.device_fingerprints (user_id, hardware_uuid, last_seen_at)
		values ($1::uuid, $2, now())
		on conflict (user_id, hardware_uuid) do update
		set last_seen_at = now()
	`, userID, hardwareUUID)
	return err
}

func (m *AuthQuota) countHardwareAccounts(ctx context.Context, hardwareUUID string) (int64, error) {
	if m.DB == nil {
		return 0, fmt.Errorf("database unavailable")
	}
	var count int64
	err := m.DB.QueryRow(ctx, `
		select count(distinct user_id)
		from public.device_fingerprints
		where hardware_uuid = $1
	`, hardwareUUID).Scan(&count)
	return count, err
}

// isSyncEdgeProxy uses immediate Cloudflare edge signals (no network round trip).
// When threatMin <= 0, CF threat-score gating is disabled (no invented floor).
func isSyncEdgeProxy(r *http.Request, threatMin int) bool {
	if strings.EqualFold(r.Header.Get("CF-IPCountry"), "T1") {
		return true
	}
	if threatMin <= 0 {
		return false
	}
	if score := strings.TrimSpace(r.Header.Get("CF-Threat-Score")); score != "" {
		if n, err := strconv.Atoi(score); err == nil && n >= threatMin {
			return true
		}
	}
	return false
}

// allowVelocity increments a per-minute Redis counter and returns false when over limit.
func (m *AuthQuota) allowVelocity(ctx context.Context, keyPrefix string, limitPerMin int) bool {
	if m.Redis == nil || limitPerMin <= 0 {
		return true
	}
	bucket := time.Now().UTC().Unix() / 60
	key := fmt.Sprintf("%s:%d", keyPrefix, bucket)
	n, err := m.Redis.Incr(ctx, key).Result()
	if err != nil {
		return true
	}
	if n == 1 {
		m.expireRedis(ctx, key, m.Config.RedisVelocityBucketTTLSec)
	}
	return n <= int64(limitPerMin)
}

func shouldMeterPath(path, method string) bool {
	// Only LLM proxy posts burn credits. Dashboard and billing never meter.
	if method != http.MethodPost {
		return false
	}
	switch path {
	case "/v1/chat/completions", "/v1/messages":
		return true
	default:
		return false
	}
}

var (
	errHardwareMismatch      = errors.New("hardware uuid mismatch")
	errHardwareBoundRequired = errors.New("hardware uuid bound required")
	errAgentRequired         = errors.New("agent id required")
	errAgentUnknown          = errors.New("agent id unknown")
	errAgentMismatch         = errors.New("agent id mismatch")
)

type cachedAPIKeyIdentity struct {
	UserID string `json:"u"`
	KeyID  string `json:"k"`
	Agent  string `json:"a"`
}

func (m *AuthQuota) resolveIdentity(ctx context.Context, rawToken, hardwareUUID, agentID string) (string, error) {
	hardwareUUID = strings.TrimSpace(hardwareUUID)
	agentID = strings.ToLower(strings.TrimSpace(agentID))

	// Prefer Supabase JWT (three base64 segments) for the web/admin dashboard.
	// Supports ES256 (JWKS) and legacy HS256 (JWT_SECRET) - Supabase signing keys.
	if strings.Count(rawToken, ".") == 2 {
		sub, provider, err := verifySupabaseAccessToken(rawToken, m.Config.JWTSecret, m.Config.SupabaseURL)
		if err == nil && sub != "" {
			if err := m.ensureAllowedProfileProvider(ctx, sub, provider); err != nil {
				return "", err
			}
			return sub, nil
		}
	}

	keyHash := hashKey(rawToken)
	cacheKey := "apikey:" + keyHash
	var userID, keyID, boundAgent, legacyHW string

	if m.Redis != nil {
		raw, err := m.Redis.Get(ctx, cacheKey).Result()
		if err == nil && raw != "" {
			if strings.Contains(raw, "{") {
				var cached cachedAPIKeyIdentity
				if json.Unmarshal([]byte(raw), &cached) == nil && cached.UserID != "" && cached.KeyID != "" {
					userID, keyID, boundAgent = cached.UserID, cached.KeyID, cached.Agent
				}
			}
		} else if err != nil && err != redis.Nil {
			return "", err
		}
	}

	if userID == "" || keyID == "" {
		if m.DB == nil {
			return "", fmt.Errorf("database unavailable")
		}
		var revoked bool
		var hwPtr, agentPtr *string
		q := `
			select id::text, user_id, revoked, hardware_uuid, agent_id
			from public.api_keys
			where key_hash = $1 and (expires_at is null or expires_at > now())
		`
		if err := m.DB.QueryRow(ctx, q, keyHash).Scan(&keyID, &userID, &revoked, &hwPtr, &agentPtr); err != nil {
			return "", err
		}
		if revoked {
			return "", fmt.Errorf("revoked")
		}
		if hwPtr != nil {
			legacyHW = strings.TrimSpace(*hwPtr)
		}
		if agentPtr != nil {
			boundAgent = strings.ToLower(strings.TrimSpace(*agentPtr))
		}
		if m.Redis != nil {
			payload, _ := json.Marshal(cachedAPIKeyIdentity{UserID: userID, KeyID: keyID, Agent: boundAgent})
			if m.Config.RedisAPIKeyCacheTTLSec > 0 {
				_ = m.Redis.Set(ctx, cacheKey, payload, m.redisTTL(m.Config.RedisAPIKeyCacheTTLSec)).Err()
			} else {
				_ = m.Redis.Set(ctx, cacheKey, payload, 0).Err()
			}
		}
	} else if m.DB != nil {
		// Refresh legacy HW for allowlist merge (cheap; keep cache lean).
		_ = m.DB.QueryRow(ctx, `
			select coalesce(hardware_uuid, '') from public.api_keys where id = $1::uuid
		`, keyID).Scan(&legacyHW)
		legacyHW = strings.TrimSpace(legacyHW)
	}

	// Agent identity: required on every API-key request; must be in catalog;
	// when the key has agent_id set, header must match.
	if agentID == "" {
		return "", errAgentRequired
	}
	ok, err := m.agentIDAllowed(ctx, agentID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errAgentUnknown
	}
	if boundAgent != "" && boundAgent != agentID {
		return "", errAgentMismatch
	}

	// Device allowlist (multi-device: CLI + IDE). Unbound keys skip.
	if err := m.enforceAPIKeyDeviceAllowlist(ctx, keyID, legacyHW, hardwareUUID, agentID); err != nil {
		return "", err
	}

	if err := m.ensureAllowedProfileProvider(ctx, userID, ""); err != nil {
		return "", err
	}
	return userID, nil
}

const authProvidersCacheKey = "auth:allowed_providers"

// allowedAuthProviders reads public.auth_settings (source of truth), with a short Redis cache.
// Fail closed: empty or DB error means no providers (matches signup trigger and public API).
func (m *AuthQuota) allowedAuthProviders(ctx context.Context) []string {
	if ctx == nil {
		ctx = context.Background()
	}
	if m.Redis != nil {
		cached, err := m.Redis.Get(ctx, authProvidersCacheKey).Result()
		if err == nil && strings.TrimSpace(cached) != "" {
			parts := strings.Split(cached, ",")
			out := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.ToLower(strings.TrimSpace(p))
				if p != "" {
					out = append(out, p)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	if m.DB == nil {
		return nil
	}
	providers, err := authsettings.ListAllowedProviders(ctx, m.DB)
	if err != nil || len(providers) == 0 {
		return nil
	}
	if m.Redis != nil {
		ttl := m.redisTTL(m.Config.RedisAuthProvidersTTLSec)
		_ = m.Redis.Set(ctx, authProvidersCacheKey, strings.Join(providers, ","), ttl).Err()
	}
	return providers
}

func (m *AuthQuota) authProviderDeniedMessage(ctx context.Context) string {
	allowed := m.allowedAuthProviders(ctx)
	if len(allowed) == 0 {
		return siteMsg("AUTH_PROVIDER_DENIED_EMPTY")
	}
	fmtStr := siteMsg("AUTH_PROVIDER_DENIED_FMT")
	if fmtStr == "" {
		return ""
	}
	return fmt.Sprintf(fmtStr, strings.Join(allowed, ", "))
}

// resolveJWTSubjectOnly extracts the JWT sub without provider allow-list checks.
// Used only for DELETE /me/account so disallowed OAuth signups can self-purge.
func (m *AuthQuota) resolveJWTSubjectOnly(rawToken string) (string, error) {
	if strings.Count(rawToken, ".") != 2 {
		return "", fmt.Errorf("not a jwt")
	}
	sub, _, err := verifySupabaseAccessToken(rawToken, m.Config.JWTSecret, m.Config.SupabaseURL)
	if err != nil || sub == "" {
		return "", fmt.Errorf("invalid jwt")
	}
	return sub, nil
}

func (m *AuthQuota) providerAllowed(ctx context.Context, provider string) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return false
	}
	for _, a := range m.allowedAuthProviders(ctx) {
		if strings.EqualFold(a, provider) {
			return true
		}
	}
	return false
}

func (m *AuthQuota) ensureAllowedProfileProvider(ctx context.Context, userID, jwtProvider string) error {
	if m.DB == nil {
		return nil
	}
	var authProvider string
	err := m.DB.QueryRow(ctx, `
		select lower(auth_provider) from public.profiles where id = $1::uuid
	`, userID).Scan(&authProvider)
	if err != nil {
		// Profile may not exist yet on first JWT hit before trigger.
		// Fail closed unless the JWT itself names an allowed provider.
		if !m.providerAllowed(ctx, jwtProvider) {
			return fmt.Errorf("auth provider not allowed")
		}
		return nil
	}
	if !m.providerAllowed(ctx, authProvider) {
		return fmt.Errorf("auth provider not allowed")
	}
	return nil
}

// enforceJA4AccountGraph links free-tier user IDs that share a TLS fingerprint.
// Postgres is the durable source of truth; Redis stays a warm set for fast paths.
func (m *AuthQuota) enforceJA4AccountGraph(ctx context.Context, r *http.Request, userID string) (bool, string) {
	ja4 := firstNonEmpty(r.Header.Get("CF-JA4"), r.Header.Get("X-JA4"), r.Header.Get("X-JA3"))
	if ja4 == "" {
		return false, ""
	}
	if err := m.recordJA4(ctx, userID, ja4); err != nil {
		log.Printf("trim: ja4 fingerprint persist failed: %v", err)
	}
	maxJA4 := m.effMaxJA4()
	if maxJA4 < 1 {
		return false, ""
	}
	count, err := m.countJA4Accounts(ctx, ja4)
	if err != nil {
		return false, ""
	}
	if m.Redis != nil {
		key := "ja4_accounts:" + ja4
		_ = m.Redis.SAdd(ctx, key, userID).Err()
		m.expireRedis(ctx, key, m.Config.RedisHardwareGraphTTLSec)
	}
	if count > int64(maxJA4) {
		return true, siteMsg("JA4_ACCOUNT_LIMIT")
	}
	return false, ""
}

func (m *AuthQuota) recordJA4(ctx context.Context, userID, ja4Hash string) error {
	if m.DB == nil {
		return nil
	}
	_, err := m.DB.Exec(ctx, `
		insert into public.ja4_fingerprints (user_id, ja4_hash, last_seen_at)
		values ($1::uuid, $2, now())
		on conflict (user_id, ja4_hash) do update
		set last_seen_at = now()
	`, userID, ja4Hash)
	return err
}

func (m *AuthQuota) countJA4Accounts(ctx context.Context, ja4Hash string) (int64, error) {
	if m.DB == nil {
		return 0, fmt.Errorf("database unavailable")
	}
	var count int64
	err := m.DB.QueryRow(ctx, `
		select count(distinct user_id)
		from public.ja4_fingerprints
		where ja4_hash = $1
	`, ja4Hash).Scan(&count)
	return count, err
}

func (m *AuthQuota) consumeCredit(ctx context.Context, w http.ResponseWriter, userID string) error {
	quotaKey := "user_quota:" + userID
	val, _ := m.Redis.HGetAll(ctx, quotaKey).Result()
	var limit, used, topup int
	if len(val) == 0 {
		if m.DB == nil {
			m.fail(w, http.StatusServiceUnavailable, siteMsg("DATABASE_UNAVAILABLE"))
			return fmt.Errorf("db")
		}
		q := `select monthly_credit_limit, monthly_credit_used, purchased_topup_credits, plan_tier from public.user_quotas where user_id = $1`
		var tier string
		if err := m.DB.QueryRow(ctx, q, userID).Scan(&limit, &used, &topup, &tier); err != nil {
			m.fail(w, http.StatusInternalServerError, siteMsg("QUOTA_LOAD_FAILED"))
			return err
		}
		unlimited, err := m.planUnlimited(ctx, tier)
		if err != nil {
			m.fail(w, http.StatusInternalServerError, siteMsg("QUOTA_LOAD_FAILED"))
			return err
		}
		_ = m.Redis.HSet(ctx, quotaKey, map[string]interface{}{
			"limit":     limit,
			"used":      used,
			"topup":     topup,
			"tier":      strings.ToLower(strings.TrimSpace(tier)),
			"unlimited": boolToRedis(unlimited),
		}).Err()
		m.expireRedis(ctx, quotaKey, m.Config.RedisQuotaCacheTTLSec)
		if unlimited {
			return nil
		}
	} else {
		fmt.Sscanf(val["limit"], "%d", &limit)
		fmt.Sscanf(val["used"], "%d", &used)
		fmt.Sscanf(val["topup"], "%d", &topup)
		unlimited, err := m.cachedOrLoadUnlimited(ctx, w, quotaKey, val, userID, "")
		if err != nil {
			return err
		}
		if unlimited {
			return nil
		}
	}

	available := (limit - used) + topup
	if available <= 0 {
		if m.Metrics != nil {
			m.Metrics.IncQuotaExhausted()
		}
		m.writeQuotaExhausted(w, "QUOTA_EXHAUSTED", "APP_PATH_UPGRADE")
		return fmt.Errorf("exhausted")
	}

	if used < limit {
		_ = m.Redis.HIncrBy(ctx, quotaKey, "used", 1).Err()
	} else {
		newTopup, err := m.Redis.HIncrBy(ctx, quotaKey, "topup", -1).Result()
		if err != nil || newTopup < 0 {
			if newTopup < 0 {
				_ = m.Redis.HIncrBy(ctx, quotaKey, "topup", 1).Err()
			}
			if m.Metrics != nil {
				m.Metrics.IncQuotaExhausted()
			}
			m.writeQuotaExhausted(w, "QUOTA_EXHAUSTED", "APP_PATH_UPGRADE")
			return fmt.Errorf("exhausted")
		}
	}
	go m.syncQuota(userID)
	return nil
}

// consumeWorkspaceCredit debits the shared workspace pool when X-Workspace-Id is set.
// Membership is verified against workspace_members. Personal top-ups are not used here.
func (m *AuthQuota) consumeWorkspaceCredit(ctx context.Context, w http.ResponseWriter, userID, workspaceID string) error {
	if m.DB == nil {
		m.fail(w, http.StatusServiceUnavailable, siteMsg("DATABASE_UNAVAILABLE"))
		return fmt.Errorf("db")
	}

	var role string
	err := m.DB.QueryRow(ctx, `
		select role from public.workspace_members
		where workspace_id = $1 and user_id = $2
	`, workspaceID, userID).Scan(&role)
	if err != nil || role == "" {
		m.fail(w, http.StatusForbidden, siteMsg("WORKSPACE_NOT_MEMBER"))
		return fmt.Errorf("not member")
	}

	quotaKey := "workspace_quota:" + workspaceID
	val, _ := m.Redis.HGetAll(ctx, quotaKey).Result()
	var limit, used int
	if len(val) == 0 {
		q := `
			select wq.monthly_shared_credits, wq.credits_consumed, coalesce(w.plan_tier, '')
			from public.workspace_quotas wq
			inner join public.workspaces w on w.id = wq.workspace_id
			where wq.workspace_id = $1
		`
		var planTier string
		if err := m.DB.QueryRow(ctx, q, workspaceID).Scan(&limit, &used, &planTier); err != nil {
			m.fail(w, http.StatusInternalServerError, siteMsg("WORKSPACE_QUOTA_LOAD_FAILED"))
			return err
		}
		unlimited, uerr := m.planUnlimited(ctx, planTier)
		if uerr != nil {
			m.fail(w, http.StatusInternalServerError, siteMsg("WORKSPACE_QUOTA_LOAD_FAILED"))
			return uerr
		}
		_ = m.Redis.HSet(ctx, quotaKey, map[string]interface{}{
			"limit":     limit,
			"used":      used,
			"tier":      strings.ToLower(strings.TrimSpace(planTier)),
			"unlimited": boolToRedis(unlimited),
		}).Err()
		m.expireRedis(ctx, quotaKey, m.Config.RedisQuotaCacheTTLSec)
		if unlimited {
			return nil
		}
	} else {
		fmt.Sscanf(val["limit"], "%d", &limit)
		fmt.Sscanf(val["used"], "%d", &used)
		unlimited, uerr := m.cachedOrLoadUnlimited(ctx, w, quotaKey, val, "", workspaceID)
		if uerr != nil {
			return uerr
		}
		if unlimited {
			return nil
		}
	}

	if used >= limit {
		m.writeQuotaExhausted(w, "WORKSPACE_QUOTA_EXHAUSTED", "APP_PATH_TEAM")
		return fmt.Errorf("workspace exhausted")
	}

	_ = m.Redis.HIncrBy(ctx, quotaKey, "used", 1).Err()
	go m.syncWorkspaceQuota(workspaceID)
	return nil
}

func boolToRedis(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

// planUnlimited reads plan_catalog.unlimited for an active plan_tier. Fail closed if missing.
func (m *AuthQuota) planUnlimited(ctx context.Context, planTier string) (bool, error) {
	planTier = strings.ToLower(strings.TrimSpace(planTier))
	if planTier == "" || m.DB == nil {
		return false, fmt.Errorf("plan tier unavailable")
	}
	var unlimited bool
	err := m.DB.QueryRow(ctx, `
		select unlimited from public.plan_catalog
		where id = $1 and is_active = true
	`, planTier).Scan(&unlimited)
	if err != nil {
		return false, err
	}
	return unlimited, nil
}

// cachedOrLoadUnlimited always refreshes from plan_catalog (source of truth) and
// writes Redis. Trusting a cached unlimited bit caused paid enterprise
// users to stay "exhausted" after catalog.unlimited was repaired to true.
func (m *AuthQuota) cachedOrLoadUnlimited(ctx context.Context, w http.ResponseWriter, quotaKey string, val map[string]string, userID, workspaceID string) (bool, error) {
	tier := strings.ToLower(strings.TrimSpace(val["tier"]))
	if tier == "" && m.DB != nil {
		var err error
		if userID != "" {
			err = m.DB.QueryRow(ctx, `select plan_tier from public.user_quotas where user_id = $1`, userID).Scan(&tier)
		} else if workspaceID != "" {
			err = m.DB.QueryRow(ctx, `select coalesce(plan_tier, '') from public.workspaces where id = $1`, workspaceID).Scan(&tier)
		}
		if err != nil {
			m.fail(w, http.StatusInternalServerError, siteMsg("QUOTA_LOAD_FAILED"))
			return false, err
		}
		tier = strings.ToLower(strings.TrimSpace(tier))
		if tier != "" {
			_ = m.Redis.HSet(ctx, quotaKey, "tier", tier).Err()
		}
	}
	unlimited, err := m.planUnlimited(ctx, tier)
	if err != nil {
		m.fail(w, http.StatusInternalServerError, siteMsg("QUOTA_LOAD_FAILED"))
		return false, err
	}
	_ = m.Redis.HSet(ctx, quotaKey, "unlimited", boolToRedis(unlimited)).Err()
	return unlimited, nil
}

type ctxUserID struct{}
type ctxWorkspaceID struct{}

func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxUserID{}).(string)
	return v
}

func WorkspaceIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxWorkspaceID{}).(string)
	return v
}

func (m *AuthQuota) syncQuota(userID string) {
	if m.DB == nil {
		return
	}
	sec := m.Config.QuotaSyncTimeoutSec
	if sec < 1 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sec)*time.Second)
	defer cancel()
	quotaKey := "user_quota:" + userID
	used, err := m.Redis.HGet(ctx, quotaKey, "used").Int()
	if err != nil {
		return
	}
	topup, err := m.Redis.HGet(ctx, quotaKey, "topup").Int()
	if err != nil {
		return
	}
	_, _ = m.DB.Exec(ctx, `
		update public.user_quotas
		set monthly_credit_used = $1,
		    purchased_topup_credits = $2,
		    updated_at = now()
		where user_id = $3
	`, used, topup, userID)
}

func (m *AuthQuota) syncWorkspaceQuota(workspaceID string) {
	if m.DB == nil {
		return
	}
	sec := m.Config.QuotaSyncTimeoutSec
	if sec < 1 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sec)*time.Second)
	defer cancel()
	quotaKey := "workspace_quota:" + workspaceID
	used, err := m.Redis.HGet(ctx, quotaKey, "used").Int()
	if err != nil {
		return
	}
	_, _ = m.DB.Exec(ctx, `
		update public.workspace_quotas
		set credits_consumed = $1, updated_at = now()
		where workspace_id = $2
	`, used, workspaceID)
}

func (m *AuthQuota) fail(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// writeQuotaExhausted returns HTTP 402 with DB-backed copy + absolute upgrade URL.
func (m *AuthQuota) writeQuotaExhausted(w http.ResponseWriter, code, pathCode string) {
	upgradePath := strings.TrimRight(m.Config.AppPublicURL, "/") + subscriptions.MessageForCode(pathCode)
	payload := map[string]interface{}{
		"error":        siteMsg(code),
		"code":         code,
		"tier_upgrade": upgradePath,
	}
	if title := siteMsg("IDE_QUOTA_EXHAUSTED_TITLE"); title != "" {
		payload["exhausted_title"] = title
	}
	if body := siteMsg("IDE_QUOTA_EXHAUSTED_BODY"); body != "" {
		payload["exhausted_body"] = body
	}
	if label := siteMsg("IDE_QUOTA_UPGRADE_ACTION"); label != "" {
		payload["upgrade_action_label"] = label
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired)
	_ = json.NewEncoder(w).Encode(payload)
}

// invalidateQuotaCaches drops Redis user + owned-workspace quota hashes after
// ExpireIfNeeded (or any entitlement change) so the next debit reads Postgres.
func (m *AuthQuota) invalidateQuotaCaches(ctx context.Context, userID string) {
	if m.Redis == nil || userID == "" {
		return
	}
	_ = m.Redis.Del(ctx, "user_quota:"+userID).Err()
	if m.DB == nil {
		return
	}
	rows, err := m.DB.Query(ctx, `
		select id::text from public.workspaces where owner_id = $1::uuid
	`, userID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil || id == "" {
			continue
		}
		_ = m.Redis.Del(ctx, "workspace_quota:"+id).Err()
	}
}

// planTier returns the user's current plan_tier from Redis quota cache or Postgres.
// Empty string means unknown or unpaid; callers must use isFreePlan for free-tier gates.
func (m *AuthQuota) planTier(ctx context.Context, userID string) string {
	quotaKey := "user_quota:" + userID
	if m.Redis != nil {
		if t, err := m.Redis.HGet(ctx, quotaKey, "tier").Result(); err == nil && t != "" {
			return strings.ToLower(t)
		}
	}
	if m.DB == nil {
		return ""
	}
	var tier string
	err := m.DB.QueryRow(ctx, `
		select plan_tier from public.user_quotas where user_id = $1
	`, userID).Scan(&tier)
	if err != nil || strings.TrimSpace(tier) == "" {
		return ""
	}
	tier = strings.ToLower(strings.TrimSpace(tier))
	if m.Redis != nil {
		_ = m.Redis.HSet(ctx, quotaKey, "tier", tier).Err()
	}
	return tier
}

// isFreePlan treats empty/unknown tiers and the configured default tier as free
// so free-tier fraud gates still apply (DEFAULT_PLAN_TIER from site_messages).
func isFreePlan(tier string) bool {
	t := strings.ToLower(strings.TrimSpace(tier))
	if t == "" {
		return true
	}
	defaultTier := strings.ToLower(strings.TrimSpace(subscriptions.MessageForCode("DEFAULT_PLAN_TIER")))
	if defaultTier == "" {
		return false
	}
	return t == defaultTier
}

func hashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func jwtIssuedAt(token string) int64 {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0
	}
	var claims struct {
		Iat int64 `json:"iat"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return 0
	}
	return claims.Iat
}

func (m *AuthQuota) requireFreeTierPoW(w http.ResponseWriter, r *http.Request, userID string) bool {
	ch := pow.CurrentChallenge(userID, m.effPoW())
	nonce := strings.TrimSpace(r.Header.Get(pow.HeaderName))
	if nonce != "" && pow.Verify(ch, nonce) {
		return true
	}
	prev := ch
	prev.Bucket--
	if nonce != "" && pow.Verify(prev, nonce) {
		return true
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(pow.ChallengeHeader, ch.Encode())
	w.WriteHeader(http.StatusPreconditionRequired)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":      "POW_REQUIRED",
		"error":     siteMsg("POW_REQUIRED"),
		"challenge": ch.Encode(),
		"header":    pow.HeaderName,
	})
	return false
}

func (m *AuthQuota) triggerCanary(r *http.Request, hardwareUUID string) {
	ip := clientIP(r)
	log.Printf("trim: canary API key presented ip=%s hw=%s", ip, hardwareUUID)
	if m.Redis == nil {
		return
	}
	ctx := r.Context()
	if ip != "" && m.Config.RedisIPBlockTTLSec > 0 {
		_ = m.Redis.Set(ctx, "ip_block:"+ip, true, m.redisTTL(m.Config.RedisIPBlockTTLSec)).Err()
	} else if ip != "" {
		_ = m.Redis.Set(ctx, "ip_block:"+ip, true, 0).Err()
	}
	if hardwareUUID != "" {
		_ = m.Redis.Set(ctx, "hw_block:"+hardwareUUID, true, 0).Err()
	}
}

func isOutdated(current, minimum string) bool {
	return compareSemver(current, minimum) < 0
}

func compareSemver(a, b string) int {
	pa := parseSemver(a)
	pb := parseSemver(b)
	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	return 0
}

func parseSemver(v string) [3]int {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	// Strip product prefixes such as TrimCLI/1.2.3 or trim-ide/0.1.0.
	if i := strings.LastIndex(v, "/"); i >= 0 && i+1 < len(v) {
		v = v[i+1:]
	}
	parts := strings.Split(v, ".")
	var out [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		fmt.Sscanf(parts[i], "%d", &out[i])
	}
	return out
}

type AntiFraud struct {
	DB            *pgxpool.Pool
	Redis         *redis.Client
	CLIHMACSecret string
	Config        config.Config
	OnHMACFail    func()
}

func NewAntiFraud(db *pgxpool.Pool, rdb *redis.Client, cfg config.Config) *AntiFraud {
	return &AntiFraud{
		DB:            db,
		Redis:         rdb,
		CLIHMACSecret: strings.TrimSpace(cfg.CLIHMACSecret),
		Config:        cfg,
	}
}

// Denylist enforces Postgres IP/ASN denylist only (no rate-limit / UA / HMAC).
// Use on public /api/v1 routes that do not go through Protect.
func (m *AntiFraud) Denylist(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if ip != "" && m.enforcePostgresDenylist(w, r, ip) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *AntiFraud) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || r.URL.Path == "/healthz" || r.URL.Path == "/readyz" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		ip := clientIP(r)
		ua := r.Header.Get("User-Agent")
		hw := strings.TrimSpace(r.Header.Get("X-Hardware-UUID"))

		// Layer 8 canary / decoy paths: any hit permanently flags the device + IP.
		if isCanaryPath(r.URL.Path) {
			m.banClient(r, ip, hw, "canary_path")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": siteMsg("CANARY_NOT_FOUND")})
			return
		}

		if ip != "" {
			if m.Redis != nil {
				if blocked, err := m.Redis.Get(r.Context(), "ip_block:"+ip).Bool(); err == nil && blocked {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"code":  "IP_BLOCKED",
						"error": siteMsg("IP_BLOCKED"),
					})
					return
				}
			}
			// Admin Postgres IP/ASN denylist (hot path; Redis-cached when available).
			if m.enforcePostgresDenylist(w, r, ip) {
				return
			}
			if m.Redis != nil {
				m.enrichIPRiskFromCloudflare(r, ip)
				// VPN / proxy IP risk is recorded here. Free-tier enforcement happens in
				// AuthQuota after identity is known (paid plans may use VPN).

				// Per-IP velocity (Redis minute buckets). 0 disables.
				if m.effRateLimitIP() > 0 {
					bucket := time.Now().UTC().Unix() / 60
					key := fmt.Sprintf("rl:ip:%s:%d", ip, bucket)
					n, err := m.Redis.Incr(r.Context(), key).Result()
					if err == nil {
						if n == 1 && m.Config.RedisVelocityBucketTTLSec > 0 {
							_ = m.Redis.Expire(r.Context(), key, time.Duration(m.Config.RedisVelocityBucketTTLSec)*time.Second).Err()
						}
						if n > int64(m.effRateLimitIP()) {
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(http.StatusTooManyRequests)
							_ = json.NewEncoder(w).Encode(map[string]string{
								"code":  "RATE_LIMITED",
								"error": siteMsg("RATE_LIMIT_IP"),
							})
							return
						}
					}
				}
			}
		}

		// JA4 / JA3 ready hooks: Cloudflare or edge proxies may forward fingerprints.
		if m.Redis != nil {
			ja4 := firstNonEmpty(r.Header.Get("CF-JA4"), r.Header.Get("X-JA4"), r.Header.Get("X-JA3"))
			if ja4 != "" {
				if blocked, err := m.Redis.Get(r.Context(), "ja4_block:"+ja4).Bool(); err == nil && blocked {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"code":  "TLS_FINGERPRINT_BLOCKED",
						"error": siteMsg("TLS_FINGERPRINT_BLOCKED"),
					})
					return
				}
				// Soft score retained for ops; AuthQuota enforces ja4_accounts graph on free tier.
				ja4TTL := time.Duration(m.Config.RedisJA4SeenTTLSec) * time.Second
				if m.Config.RedisJA4SeenTTLSec < 1 {
					ja4TTL = 0
				}
				_ = m.Redis.Set(r.Context(), "ja4_seen:"+ja4, time.Now().UTC().Format(time.RFC3339), ja4TTL).Err()
			}
		}

		if ua == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"code":  "SUSPICIOUS_CLIENT",
				"error": siteMsg("SUSPICIOUS_CLIENT"),
			})
			return
		}

		// CLI clients must send a fresh timestamp (replay protection) and HMAC signature.
		isCLI := strings.Contains(ua, "TrimCLI/")
		ts := r.Header.Get("X-Request-Timestamp")
		// Fail closed: never invent a 30s skew. Cloud requires TRIM_CLI_TIMESTAMP_SKEW_SEC.
		// Self-host with skew unset skips expiry window (timestamp still required for CLI).
		skewSec := m.Config.CLITimestampSkewSec
		skew := time.Duration(skewSec) * time.Second
		if isCLI {
			if ts == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"code":  "TIMESTAMP_REQUIRED",
					"error": siteMsg("TIMESTAMP_REQUIRED"),
				})
				return
			}
			if skewSec > 0 {
				var t int64
				fmt.Sscanf(ts, "%d", &t)
				reqTime := time.Unix(t, 0)
				if time.Since(reqTime) > skew || time.Until(reqTime) > skew {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"code":  "REQUEST_EXPIRED",
						"error": siteMsg("REQUEST_EXPIRED"),
					})
					return
				}
			}
			if m.CLIHMACSecret != "" {
				if !verifyCLIHMAC(w, r, m.CLIHMACSecret, ts) {
					if m.OnHMACFail != nil {
						m.OnHMACFail()
					}
					return
				}
			}
		} else if ts != "" && skewSec > 0 {
			var t int64
			fmt.Sscanf(ts, "%d", &t)
			reqTime := time.Unix(t, 0)
			if time.Since(reqTime) > skew || time.Until(reqTime) > skew {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"code":  "REQUEST_EXPIRED",
					"error": siteMsg("REQUEST_EXPIRED"),
				})
				return
			}
		}

		if m.Redis != nil && hw != "" {
			blocked, err := m.Redis.Get(r.Context(), "hw_block:"+hw).Bool()
			if err == nil && blocked {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"code":  "HARDWARE_BLOCKED",
					"error": siteMsg("HARDWARE_BLOCKED"),
				})
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func isCanaryPath(path string) bool {
	switch path {
	case "/api/v1/internal/free-credits",
		"/api/v1/internal/grant-credits",
		"/v1/internal/free-credits",
		"/api/v1/admin/bypass-quota":
		return true
	default:
		return false
	}
}

func (m *AntiFraud) banClient(r *http.Request, ip, hw, reason string) {
	log.Printf("trim: canary trip reason=%s ip=%s hw=%s path=%s", reason, ip, hw, r.URL.Path)
	if m.Redis == nil {
		return
	}
	ctx := r.Context()
	if ip != "" && m.Config.RedisIPBlockTTLSec > 0 {
		_ = m.Redis.Set(ctx, "ip_block:"+ip, true, time.Duration(m.Config.RedisIPBlockTTLSec)*time.Second).Err()
	} else if ip != "" {
		_ = m.Redis.Set(ctx, "ip_block:"+ip, true, 0).Err()
	}
	if hw != "" {
		_ = m.Redis.Set(ctx, "hw_block:"+hw, true, 0).Err()
	}
	if ja4 := firstNonEmpty(r.Header.Get("CF-JA4"), r.Header.Get("X-JA4")); ja4 != "" {
		_ = m.Redis.Set(ctx, "ja4_block:"+ja4, true, 0).Err()
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func verifyCLIHMAC(w http.ResponseWriter, r *http.Request, secret, ts string) bool {
	sig := strings.TrimSpace(r.Header.Get(clisign.HeaderName))
	if sig == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"code":  "SIGNATURE_REQUIRED",
			"error": siteMsg("SIGNATURE_REQUIRED"),
		})
		return false
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"code":  "BODY_READ_FAILED",
			"error": siteMsg("BODY_READ_FAILED"),
		})
		return false
	}
	_ = r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))
	r.ContentLength = int64(len(body))

	if !clisign.Verify(secret, ts, r.Method, r.URL.Path, body, sig) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		detail := siteMsg("CLI_SIGNATURE_DETAIL_HMAC")
		errMsg := siteMsg("CLI_SIGNATURE_INVALID_FMT")
		if errMsg != "" && detail != "" {
			errMsg = fmt.Sprintf(errMsg, detail)
		} else if errMsg == "" {
			errMsg = detail
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"code":  "SIGNATURE_INVALID",
			"error": errMsg,
		})
		return false
	}
	return true
}

func siteMsg(code string) string {
	return subscriptions.MessageForCode(code)
}

func clientIP(r *http.Request) string {
	if cf := r.Header.Get("CF-Connecting-IP"); cf != "" {
		return cf
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
