package platformadmin

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (h *Handler) totpKey() ([]byte, bool) {
	key, err := parseTOTPKey(h.Config.AdminTOTPKey)
	if err != nil {
		return nil, false
	}
	return key, true
}

func (h *Handler) totpEnrolled(ctx context.Context, userID string) (bool, error) {
	db := h.readPool()
	var enrolled bool
	err := db.QueryRow(ctx, `
		select enrolled_at is not null from public.platform_admin_totp where user_id = $1::uuid
	`, userID).Scan(&enrolled)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	return enrolled, err
}

func (h *Handler) loadEnrolledSecret(ctx context.Context, userID string) (string, error) {
	key, ok := h.totpKey()
	if !ok {
		return "", errTOTPKey
	}
	var ct, nonce []byte
	var enrolledAt *time.Time
	err := h.DB.QueryRow(ctx, `
		select secret_ciphertext, secret_nonce, enrolled_at
		from public.platform_admin_totp where user_id = $1::uuid
	`, userID).Scan(&ct, &nonce, &enrolledAt)
	if err == pgx.ErrNoRows || enrolledAt == nil {
		return "", errTOTPNotEnrolled
	}
	if err != nil {
		return "", err
	}
	plain, err := decryptSecret(key, ct, nonce)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

var (
	errTOTPKey         = errSentinel("ADMIN_TOTP_KEY_MISSING")
	errTOTPNotEnrolled = errSentinel("ADMIN_TOTP_NOT_ENROLLED")
)

type errSentinel string

func (e errSentinel) Error() string { return string(e) }

func (h *Handler) getTOTPStatus(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	enrolled, err := h.totpEnrolled(r.Context(), p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	label := h.msg("ADMIN_TOTP_STATUS_OFF")
	if enrolled {
		label = h.msg("ADMIN_TOTP_STATUS_ON")
	}
	_, keyOK := h.totpKey()
	h.writeJSON(w, http.StatusOK, map[string]any{
		"enrolled":       enrolled,
		"status_label":   label,
		"key_configured": keyOK,
	})
}

func (h *Handler) postTOTPBegin(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	key, ok := h.totpKey()
	if !ok {
		h.writeErr(w, http.StatusServiceUnavailable, "ADMIN_TOTP_KEY_MISSING")
		return
	}
	enrolled, err := h.totpEnrolled(r.Context(), p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if enrolled {
		h.writeErr(w, http.StatusConflict, "ADMIN_TOTP_ALREADY_ENROLLED")
		return
	}
	secret, err := generateTOTPSecret()
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "ADMIN_STEP_UP_INVALID")
		return
	}
	ct, nonce, err := encryptSecret(key, []byte(secret))
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "ADMIN_STEP_UP_INVALID")
		return
	}
	_, err = h.DB.Exec(r.Context(), `
		insert into public.platform_admin_totp (user_id, secret_ciphertext, secret_nonce, pending_ciphertext, pending_nonce, enrolled_at, updated_at)
		values ($1::uuid, $2, $3, $2, $3, null, now())
		on conflict (user_id) do update set
		  pending_ciphertext = excluded.pending_ciphertext,
		  pending_nonce = excluded.pending_nonce,
		  updated_at = now()
		where platform_admin_totp.enrolled_at is null
	`, p.UserID, ct, nonce)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	issuer := strings.TrimSpace(h.Config.CompanyLegalName)
	if issuer == "" {
		issuer = strings.TrimSpace(h.msg("ADMIN_BRAND"))
	}
	var email string
	_ = h.DB.QueryRow(r.Context(), `select coalesce(email, '') from public.profiles where id = $1::uuid`, p.UserID).Scan(&email)
	h.audit(r.Context(), r, "admin.totp_begin", "platform_admin_totp", p.UserID, nil, map[string]any{"pending": true}, "", false)
	h.writeJSON(w, http.StatusOK, map[string]any{
		"secret":      secret,
		"otpauth_url": otpAuthURL(issuer, email, secret),
	})
}

func (h *Handler) postTOTPConfirm(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var body struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	key, ok := h.totpKey()
	if !ok {
		h.writeErr(w, http.StatusServiceUnavailable, "ADMIN_TOTP_KEY_MISSING")
		return
	}
	var pendCT, pendNonce []byte
	var enrolledAt *time.Time
	err := h.DB.QueryRow(r.Context(), `
		select pending_ciphertext, pending_nonce, enrolled_at
		from public.platform_admin_totp where user_id = $1::uuid
	`, p.UserID).Scan(&pendCT, &pendNonce, &enrolledAt)
	if err == pgx.ErrNoRows || len(pendCT) == 0 {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_TOTP_NOT_ENROLLED")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if enrolledAt != nil {
		h.writeErr(w, http.StatusConflict, "ADMIN_TOTP_ALREADY_ENROLLED")
		return
	}
	plain, err := decryptSecret(key, pendCT, pendNonce)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "ADMIN_STEP_UP_INVALID")
		return
	}
	if !verifyTOTP(string(plain), body.Code, time.Now()) {
		h.writeErr(w, http.StatusForbidden, "ADMIN_TOTP_CODE_INVALID")
		return
	}
	_, err = h.DB.Exec(r.Context(), `
		update public.platform_admin_totp set
		  secret_ciphertext = pending_ciphertext,
		  secret_nonce = pending_nonce,
		  pending_ciphertext = null,
		  pending_nonce = null,
		  enrolled_at = now(),
		  updated_at = now()
		where user_id = $1::uuid and enrolled_at is null
	`, p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	h.audit(r.Context(), r, "admin.totp_enroll", "platform_admin_totp", p.UserID, nil, map[string]any{"enrolled": true}, "", false)
	// Same proof that finishes enroll elevates the operator for TRIM_ADMIN_STEP_UP_TTL_SEC
	// so the next write does not immediately re-prompt for TOTP.
	tok, sec, terr := h.issueStepUpToken(r.Context(), r, p.UserID, "totp_enroll")
	if terr != nil {
		code := terr.Error()
		if strings.HasPrefix(code, "ADMIN_") {
			h.writeErr(w, http.StatusServiceUnavailable, code)
			return
		}
		h.writeErr(w, http.StatusInternalServerError, "ADMIN_STEP_UP_INVALID")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"step_up_token":  tok,
		"expires_in_sec": sec,
	})
}

func (h *Handler) postTOTPDisable(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var body struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	secret, err := h.loadEnrolledSecret(r.Context(), p.UserID)
	if err == errTOTPNotEnrolled {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_TOTP_NOT_ENROLLED")
		return
	}
	if err == errTOTPKey {
		h.writeErr(w, http.StatusServiceUnavailable, "ADMIN_TOTP_KEY_MISSING")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if !verifyTOTP(secret, body.Code, time.Now()) {
		h.writeErr(w, http.StatusForbidden, "ADMIN_TOTP_CODE_INVALID")
		return
	}
	_, err = h.DB.Exec(r.Context(), `
		delete from public.platform_admin_totp where user_id = $1::uuid
	`, p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if h.Redis != nil {
		_ = h.Redis.Del(r.Context(), "admin:stepup:"+p.UserID).Err()
	}
	h.audit(r.Context(), r, "admin.totp_disable", "platform_admin_totp", p.UserID, nil, map[string]any{"enrolled": false}, "", true)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_TOTP_DISABLED")})
}
