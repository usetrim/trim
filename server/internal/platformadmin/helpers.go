package platformadmin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/usetrim/trim/server/internal/billingsettings"
	"github.com/usetrim/trim/server/internal/pagination"
)

const ownerRoleID = "a0000000-0000-4000-8000-000000000001"

func (h *Handler) listPage(r *http.Request) (pagination.Params, int, error) {
	maxLimit, err := billingsettings.MaxPageSize(r.Context(), h.readPool())
	if err != nil {
		return pagination.Params{}, 0, err
	}
	skipCap, err := billingsettings.SkipToMaxPages(r.Context(), h.readPool())
	if err != nil {
		return pagination.Params{}, 0, err
	}
	params, err := pagination.Parse(r, maxLimit)
	if err != nil {
		return pagination.Params{}, 0, err
	}
	return params, skipCap, nil
}

// querySearchQ returns a trimmed, length-capped list search term (empty if absent).
func querySearchQ(r *http.Request) string {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 200 {
		q = q[:200]
	}
	return q
}

func (h *Handler) writePaginateErr(w http.ResponseWriter, err error) {
	if pe, ok := err.(pagination.CodeError); ok {
		h.writeErr(w, http.StatusBadRequest, pe.Code)
		return
	}
	code := err.Error()
	if code == "DATABASE_UNAVAILABLE" || code == "PAGINATION_MAX_LIMIT_INVALID" || code == "PAGINATION_SKIP_TO_MAX_INVALID" {
		h.writeErr(w, http.StatusInternalServerError, code)
		return
	}
	h.writeErr(w, http.StatusBadRequest, code)
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func (h *Handler) stepUpUsed(r *http.Request, userID string) bool {
	return h.stepUpOK(r, userID)
}

func randomStepUpToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (h *Handler) countActiveOwners(ctx context.Context) (int, error) {
	var n int
	err := h.readPool().QueryRow(ctx, `
		select count(*) from public.platform_admins a
		join public.platform_roles r on r.id = a.role_id
		where a.status = 'active' and r.is_owner = true
	`).Scan(&n)
	return n, err
}

// accountStatusLabel is DB chrome only. Unknown / empty status fails closed to "".
func (h *Handler) accountStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		return h.msg("ADMIN_STATUS_ACTIVE")
	case "suspended":
		return h.msg("ADMIN_STATUS_SUSPENDED")
	case "banned":
		return h.msg("ADMIN_STATUS_BANNED")
	case "pending_delete":
		return h.msg("ADMIN_STATUS_PENDING_DELETE")
	case "shadowbanned":
		return h.msg("ADMIN_STATUS_SHADOWBANNED")
	default:
		return ""
	}
}

// enterpriseInquiryStatusLabel is DB chrome only. Unknown fails closed to "".
func (h *Handler) enterpriseInquiryStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "new":
		return h.msg("ADMIN_ENTERPRISE_STATUS_NEW")
	case "contacted":
		return h.msg("ADMIN_ENTERPRISE_STATUS_CONTACTED")
	case "closed":
		return h.msg("ADMIN_ENTERPRISE_STATUS_CLOSED")
	case "offered":
		return h.msg("ADMIN_ENTERPRISE_STATUS_OFFERED")
	case "activated":
		return h.msg("ADMIN_ENTERPRISE_STATUS_ACTIVATED")
	default:
		return ""
	}
}

// disputeStatusLabel is DB chrome only. Unknown fails closed to "".
func (h *Handler) disputeStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "open":
		return h.msg("ADMIN_DISPUTE_STATUS_OPEN")
	case "watching":
		return h.msg("ADMIN_DISPUTE_STATUS_WATCHING")
	case "closed":
		return h.msg("ADMIN_DISPUTE_STATUS_CLOSED")
	default:
		return ""
	}
}

// platformAdminStatusLabel is DB chrome only. Unknown fails closed to "".
func (h *Handler) platformAdminStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		return h.msg("ADMIN_PLATFORM_ADMIN_STATUS_ACTIVE")
	case "disabled":
		return h.msg("ADMIN_PLATFORM_ADMIN_STATUS_DISABLED")
	default:
		return ""
	}
}

// breakGlassStatusLabel is DB chrome only. Unknown fails closed to "".
func (h *Handler) breakGlassStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return h.msg("ADMIN_BREAK_GLASS_STATUS_PENDING")
	case "approved":
		return h.msg("ADMIN_BREAK_GLASS_STATUS_APPROVED")
	case "denied":
		return h.msg("ADMIN_BREAK_GLASS_STATUS_DENIED")
	case "revoked":
		return h.msg("ADMIN_BREAK_GLASS_STATUS_REVOKED")
	default:
		return ""
	}
}

// webhookProcessStatusLabel is DB chrome only. Unknown fails closed to "".
func (h *Handler) webhookProcessStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ok":
		return h.msg("ADMIN_WEBHOOK_STATUS_OK")
	case "failed":
		return h.msg("ADMIN_WEBHOOK_STATUS_FAILED")
	case "pending":
		return h.msg("ADMIN_WEBHOOK_STATUS_PENDING")
	default:
		return ""
	}
}

// distSourceLabel is DB chrome only. Unknown fails closed to "".
func (h *Handler) distSourceLabel(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "github":
		return h.msg("ADMIN_DIST_SOURCE_GITHUB")
	case "install":
		return h.msg("ADMIN_DIST_SOURCE_INSTALL")
	case "homebrew":
		return h.msg("ADMIN_DIST_SOURCE_HOMEBREW")
	case "scoop":
		return h.msg("ADMIN_DIST_SOURCE_SCOOP")
	case "winget":
		return h.msg("ADMIN_DIST_SOURCE_WINGET")
	case "marketplace":
		return h.msg("ADMIN_DIST_SOURCE_MARKETPLACE")
	default:
		return ""
	}
}

// distMetricLabel is DB chrome only. Unknown fails closed to "".
func (h *Handler) distMetricLabel(metric string) string {
	switch strings.ToLower(strings.TrimSpace(metric)) {
	case "clones":
		return h.msg("ADMIN_DIST_METRIC_CLONES")
	case "views":
		return h.msg("ADMIN_DIST_METRIC_VIEWS")
	case "release_downloads":
		return h.msg("ADMIN_DIST_METRIC_RELEASE_DOWNLOADS")
	case "hits":
		return h.msg("ADMIN_DIST_METRIC_HITS")
	case "installs":
		return h.msg("ADMIN_DIST_METRIC_INSTALLS")
	default:
		return ""
	}
}

// asPositiveInt coerces JSON numbers (and whole-number floats from encoding/json) to int >= 1.
func asPositiveInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		if n < 1 || n != float64(int(n)) {
			return 0, false
		}
		return int(n), true
	case int:
		if n < 1 {
			return 0, false
		}
		return n, true
	case int64:
		if n < 1 {
			return 0, false
		}
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		if err != nil || i < 1 {
			return 0, false
		}
		return int(i), true
	default:
		return 0, false
	}
}

// asNonNegInt coerces JSON numbers to int >= 0 (whole numbers only).
func asNonNegInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		if n < 0 || n != float64(int(n)) {
			return 0, false
		}
		return int(n), true
	case int:
		if n < 0 {
			return 0, false
		}
		return n, true
	case int64:
		if n < 0 {
			return 0, false
		}
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		if err != nil || i < 0 {
			return 0, false
		}
		return int(i), true
	default:
		return 0, false
	}
}
