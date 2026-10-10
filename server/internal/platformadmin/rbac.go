package platformadmin

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/usetrim/trim/server/internal/billing"
	"github.com/usetrim/trim/server/internal/billing/catalogsync"
	"github.com/usetrim/trim/server/internal/billing/paddleapi"
	"github.com/usetrim/trim/server/internal/config"
	"github.com/usetrim/trim/server/internal/mailer"
	"github.com/usetrim/trim/server/internal/middleware"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

var errBootstrap = errors.New("platform owner bootstrap required")

type ctxAdmin struct{}

// AdminPrincipal is attached after platform RBAC gate.
type AdminPrincipal struct {
	UserID      string
	RoleID      string
	RoleSlug    string
	IsOwner     bool
	Permissions map[string]bool
}

func PrincipalFromContext(ctx context.Context) *AdminPrincipal {
	v, _ := ctx.Value(ctxAdmin{}).(*AdminPrincipal)
	return v
}

// Handler serves /api/v1/admin/*
type Handler struct {
	DB      *pgxpool.Pool
	ReadDB  *pgxpool.Pool
	Redis   *redis.Client
	Config  config.Config
	Subs    *subscriptions.Service
	Billing *billing.Handler
	Paddle  *paddleapi.Client
	Catalog *catalogsync.Syncer
	Mailer  mailer.SMTPConfig
}

func NewHandler(db, readDB *pgxpool.Pool, rdb *redis.Client, cfg config.Config, billingHandler *billing.Handler, paddleClient *paddleapi.Client, smtp mailer.SMTPConfig) *Handler {
	if readDB == nil {
		readDB = db
	}
	return &Handler{
		DB: db, ReadDB: readDB, Redis: rdb, Config: cfg,
		Subs:    subscriptions.NewService(db, readDB),
		Billing: billingHandler,
		Paddle:  paddleClient,
		Catalog: catalogsync.New(db, paddleClient),
		Mailer:  smtp,
	}
}

func (h *Handler) msg(code string) string {
	return subscriptions.MessageForCode(code)
}

func (h *Handler) writeErr(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": h.msg(code), "code": code})
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// AdminStepUpCRUDPolicy is returned on gated admin responses so operators/tools
// can confirm enroll-only CRUD (no per-action passkey/TOTP re-prompt).
var AdminStepUpCRUDPolicy = "enroll_only"

// RequirePermission middleware factory.
// When catalog.step_up_required is true, the operator must have enrolled TOTP and/or
// a passkey. Enrollment is the second factor for admin writes - we do NOT re-prompt
// passkey/TOTP on every CRUD once enrolled (that UX is hostile and unprofessional).
// Optional Redis elevation (issueStepUpToken) remains for audit "step_up_used" only.
func (h *Handler) RequirePermission(code string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Trim-Admin-StepUp-Policy", AdminStepUpCRUDPolicy)
			p := PrincipalFromContext(r.Context())
			if p == nil || !p.Permissions[PermAdminAccess] {
				h.writeErr(w, http.StatusForbidden, "ADMIN_FORBIDDEN")
				return
			}
			if !p.Permissions[code] && !p.IsOwner {
				// Break-glass may grant a single elevated permission temporarily.
				if !h.breakGlassGrants(r.Context(), p.UserID, code) {
					h.writeErr(w, http.StatusForbidden, "ADMIN_PERMISSION_DENIED")
					return
				}
			}
			needStepUp, err := h.permissionNeedsStepUp(r.Context(), code)
			if err != nil {
				h.writeErr(w, http.StatusInternalServerError, "ADMIN_PERMISSION_DENIED")
				return
			}
			if needStepUp {
				enrolled, terr := h.stepUpFactorEnrolled(r.Context(), p.UserID)
				if terr != nil {
					h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
					return
				}
				if !enrolled {
					h.writeErr(w, http.StatusForbidden, "ADMIN_STEP_UP_FACTOR_REQUIRED")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (h *Handler) permissionNeedsStepUp(ctx context.Context, code string) (bool, error) {
	var need bool
	err := h.DB.QueryRow(ctx, `
		select step_up_required from public.platform_permission_catalog where code = $1
	`, code).Scan(&need)
	if err == pgx.ErrNoRows {
		return true, nil
	}
	return need, err
}

func (h *Handler) stepUpOK(r *http.Request, userID string) bool {
	if h.Redis == nil {
		return false
	}
	key := "admin:stepup:" + userID
	val, err := h.Redis.Get(r.Context(), key).Result()
	if err != nil || val == "" {
		return false
	}
	tok := strings.TrimSpace(r.Header.Get("X-Trim-Step-Up"))
	if tok == "" {
		// Elevation is server-side for TRIM_ADMIN_STEP_UP_TTL_SEC.
		// Client header is best-effort; lost sessionStorage must not re-prompt every write.
		return true
	}
	return hmacEqual(val, tok) || val == tok
}

func hmacEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

// Gate loads platform admin row + permissions after JWT auth.
func (h *Handler) Gate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.adminOriginOK(r) {
			h.writeErr(w, http.StatusForbidden, "ADMIN_ORIGIN_FORBIDDEN")
			return
		}
		if !h.adminIPOK(r) {
			h.writeErr(w, http.StatusForbidden, "ADMIN_IP_FORBIDDEN")
			return
		}
		if ok, code := h.adminCFAccessOK(r); !ok {
			h.writeErr(w, http.StatusForbidden, code)
			return
		}
		userID := middleware.UserIDFromContext(r.Context())
		if userID == "" {
			h.writeErr(w, http.StatusUnauthorized, "AUTH_TOKEN_MISSING")
			return
		}
		if err := h.ensureBootstrapOwners(r.Context()); err != nil {
			h.writeErr(w, http.StatusServiceUnavailable, "ADMIN_BOOTSTRAP_REQUIRED")
			return
		}
		p, err := h.loadPrincipal(r.Context(), userID)
		if err != nil || p == nil {
			h.writeErr(w, http.StatusForbidden, "ADMIN_FORBIDDEN")
			return
		}
		if !p.Permissions[PermAdminAccess] {
			h.writeErr(w, http.StatusForbidden, "ADMIN_FORBIDDEN")
			return
		}
		ctx := context.WithValue(r.Context(), ctxAdmin{}, p)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// adminOriginOK enforces ADMIN_ALLOWED_ORIGINS when configured (required in cloud).
// Empty allow-list (self-host without lock) permits; nonempty fails closed on missing/mismatched Origin.
func (h *Handler) adminOriginOK(r *http.Request) bool {
	allow := h.Config.AdminAllowedOrigins
	if len(allow) == 0 {
		return true
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		if ref := strings.TrimSpace(r.Header.Get("Referer")); ref != "" {
			if u, err := url.Parse(ref); err == nil && u.Scheme != "" && u.Host != "" {
				origin = u.Scheme + "://" + u.Host
			}
		}
	}
	if origin == "" {
		return false
	}
	for _, a := range allow {
		if strings.EqualFold(strings.TrimSpace(a), origin) {
			return true
		}
	}
	return false
}

// adminIPOK enforces ADMIN_ALLOWED_CIDRS when configured.
// Empty list permits (self-host without IP lock). Nonempty fails closed.
func (h *Handler) adminIPOK(r *http.Request) bool {
	cidrs := h.Config.AdminAllowedCIDRs
	if len(cidrs) == 0 {
		return true
	}
	ipStr := strings.TrimSpace(r.Header.Get("CF-Connecting-IP"))
	if ipStr == "" {
		ipStr = strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
		if i := strings.Index(ipStr, ","); i >= 0 {
			ipStr = strings.TrimSpace(ipStr[:i])
		}
	}
	if ipStr == "" {
		host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
		if err == nil {
			ipStr = host
		} else {
			ipStr = strings.TrimSpace(r.RemoteAddr)
		}
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, raw := range cidrs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if !strings.Contains(raw, "/") {
			if parsed := net.ParseIP(raw); parsed != nil && parsed.Equal(ip) {
				return true
			}
			continue
		}
		_, network, err := net.ParseCIDR(raw)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func (h *Handler) loadPrincipal(ctx context.Context, userID string) (*AdminPrincipal, error) {
	var roleID, roleSlug string
	var isOwner bool
	var status string
	err := h.DB.QueryRow(ctx, `
		select a.role_id::text, r.slug, r.is_owner, a.status
		from public.platform_admins a
		join public.platform_roles r on r.id = a.role_id
		where a.user_id = $1::uuid
	`, userID).Scan(&roleID, &roleSlug, &isOwner, &status)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if status != "active" {
		return nil, nil
	}
	perms := map[string]bool{}
	rows, err := h.DB.Query(ctx, `
		select permission_code from public.platform_role_permissions where role_id = $1::uuid
	`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		perms[code] = true
	}
	if isOwner {
		// Owner always has every catalog permission (defense in depth).
		cat, err := h.DB.Query(ctx, `select code from public.platform_permission_catalog`)
		if err == nil {
			defer cat.Close()
			for cat.Next() {
				var code string
				if cat.Scan(&code) == nil {
					perms[code] = true
				}
			}
		}
	}
	return &AdminPrincipal{
		UserID:      userID,
		RoleID:      roleID,
		RoleSlug:    roleSlug,
		IsOwner:     isOwner,
		Permissions: perms,
	}, nil
}

// ensureBootstrapOwners promotes emails from TRIM_PLATFORM_OWNER_EMAILS to owner role once.
func (h *Handler) ensureBootstrapOwners(ctx context.Context) error {
	emails := h.Config.PlatformOwnerEmails
	if len(emails) == 0 {
		var n int
		_ = h.DB.QueryRow(ctx, `select count(*) from public.platform_admins a
			join public.platform_roles r on r.id = a.role_id
			where a.status = 'active' and r.is_owner`).Scan(&n)
		if n == 0 {
			return errBootstrap
		}
		return nil
	}
	ownerRole := "a0000000-0000-4000-8000-000000000001"
	for _, email := range emails {
		email = strings.ToLower(strings.TrimSpace(email))
		if email == "" {
			continue
		}
		var uid string
		err := h.DB.QueryRow(ctx, `select id::text from public.profiles where lower(email) = $1`, email).Scan(&uid)
		if err != nil {
			continue
		}
		_, _ = h.DB.Exec(ctx, `
			insert into public.platform_admins (user_id, role_id, status)
			values ($1::uuid, $2::uuid, 'active')
			on conflict (user_id) do update
			  set role_id = excluded.role_id, status = 'active', updated_at = now()
		`, uid, ownerRole)
	}
	var n int
	_ = h.DB.QueryRow(ctx, `select count(*) from public.platform_admins a
		join public.platform_roles r on r.id = a.role_id
		where a.status = 'active' and r.is_owner`).Scan(&n)
	if n == 0 {
		return errBootstrap
	}
	return nil
}

func (h *Handler) audit(ctx context.Context, r *http.Request, action, resourceType, resourceID string, before, after any, reason string, stepUp bool) {
	p := PrincipalFromContext(ctx)
	actor := ""
	if p != nil {
		actor = p.UserID
	}
	var beforeB, afterB []byte
	if before != nil {
		beforeB, _ = json.Marshal(before)
	}
	if after != nil {
		afterB, _ = json.Marshal(after)
	}
	ip := r.Header.Get("CF-Connecting-IP")
	if ip == "" {
		ip = r.RemoteAddr
	}
	country := r.Header.Get("CF-IPCountry")
	ja4 := r.Header.Get("CF-JA4")
	_, _ = h.DB.Exec(ctx, `
		insert into public.admin_audit_log
		  (actor_user_id, action, resource_type, resource_id, before_json, after_json, reason, ip, ja4, country, step_up_used)
		values
		  (nullif($1, '')::uuid, $2, $3, $4, $5::jsonb, $6::jsonb, $7, $8, $9, $10, $11)
	`, actor, action, resourceType, resourceID, nullJSON(beforeB), nullJSON(afterB), reason, ip, ja4, country, stepUp)
}

func nullJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}

func (h *Handler) stepUpTTL() (time.Duration, bool) {
	sec := h.Config.AdminStepUpTTLSec
	if sec < 60 || sec > 43200 {
		return 0, false
	}
	return time.Duration(sec) * time.Second, true
}
