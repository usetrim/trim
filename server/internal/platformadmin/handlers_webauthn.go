package platformadmin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

type webauthnUser struct {
	id          []byte
	name        string
	displayName string
	creds       []webauthn.Credential
}

func (u *webauthnUser) WebAuthnID() []byte                         { return u.id }
func (u *webauthnUser) WebAuthnName() string                       { return u.name }
func (u *webauthnUser) WebAuthnDisplayName() string                { return u.displayName }
func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential { return u.creds }

func (h *Handler) webauthnRPID() (string, bool) {
	id := strings.TrimSpace(h.Config.AdminWebAuthnRPID)
	if id != "" {
		return id, true
	}
	for _, origin := range h.Config.AdminAllowedOrigins {
		u, err := url.Parse(strings.TrimSpace(origin))
		if err != nil || u.Hostname() == "" {
			continue
		}
		return u.Hostname(), true
	}
	return "", false
}

func (h *Handler) webauthnDisplayName() string {
	name := strings.TrimSpace(h.Config.CompanyLegalName)
	if name != "" {
		return name
	}
	return strings.TrimSpace(h.msg("ADMIN_BRAND"))
}

func (h *Handler) newWebAuthn() (*webauthn.WebAuthn, error) {
	rpid, ok := h.webauthnRPID()
	if !ok {
		return nil, errSentinel("ADMIN_WEBAUTHN_RP_MISSING")
	}
	display := h.webauthnDisplayName()
	if display == "" {
		return nil, errSentinel("ADMIN_WEBAUTHN_RP_MISSING")
	}
	origins := make([]string, 0, len(h.Config.AdminAllowedOrigins))
	for _, o := range h.Config.AdminAllowedOrigins {
		o = strings.TrimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}
	if len(origins) == 0 {
		return nil, errSentinel("ADMIN_WEBAUTHN_RP_MISSING")
	}
	return webauthn.New(&webauthn.Config{
		RPID:          rpid,
		RPDisplayName: display,
		RPOrigins:     origins,
	})
}

func (h *Handler) webauthnEnrolled(ctx context.Context, userID string) (bool, error) {
	db := h.readPool()
	var n int
	err := db.QueryRow(ctx, `
		select count(*) from public.platform_admin_webauthn_credentials where user_id = $1::uuid
	`, userID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (h *Handler) stepUpFactorEnrolled(ctx context.Context, userID string) (bool, error) {
	totpOK, err := h.totpEnrolled(ctx, userID)
	if err != nil {
		return false, err
	}
	if totpOK {
		return true, nil
	}
	return h.webauthnEnrolled(ctx, userID)
}

func (h *Handler) loadWebAuthnUser(ctx context.Context, userID string) (*webauthnUser, error) {
	var email string
	err := h.DB.QueryRow(ctx, `
		select coalesce(email, '') from public.profiles where id = $1::uuid
	`, userID).Scan(&email)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	rows, err := h.DB.Query(ctx, `
		select credential_id, public_key, attestation_type, coalesce(transport, '{}'),
		       sign_count, clone_warning, aaguid
		from public.platform_admin_webauthn_credentials
		where user_id = $1::uuid
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	creds := make([]webauthn.Credential, 0)
	for rows.Next() {
		var (
			credID, pubKey, aaguid []byte
			attType                string
			transports             []string
			signCount              int64
			cloneWarn              bool
		)
		if err := rows.Scan(&credID, &pubKey, &attType, &transports, &signCount, &cloneWarn, &aaguid); err != nil {
			return nil, err
		}
		tr := make([]protocol.AuthenticatorTransport, 0, len(transports))
		for _, t := range transports {
			t = strings.TrimSpace(t)
			if t != "" {
				tr = append(tr, protocol.AuthenticatorTransport(t))
			}
		}
		creds = append(creds, webauthn.Credential{
			ID:              credID,
			PublicKey:       pubKey,
			AttestationType: attType,
			Transport:       tr,
			Authenticator: webauthn.Authenticator{
				AAGUID:       aaguid,
				SignCount:    uint32(signCount),
				CloneWarning: cloneWarn,
			},
		})
	}
	name := strings.TrimSpace(email)
	if name == "" {
		name = userID
	}
	return &webauthnUser{
		id:          uid[:],
		name:        name,
		displayName: name,
		creds:       creds,
	}, nil
}

func (h *Handler) storeWebAuthnSession(ctx context.Context, kind, userID string, session *webauthn.SessionData) error {
	if h.Redis == nil || session == nil {
		return fmt.Errorf("ADMIN_WEBAUTHN_UNAVAILABLE")
	}
	ttl, ok := h.stepUpTTL()
	if !ok || ttl < time.Second {
		return fmt.Errorf("ADMIN_STEP_UP_TTL_MISSING")
	}
	raw, err := json.Marshal(session)
	if err != nil {
		return err
	}
	key := "admin:webauthn:" + kind + ":" + userID
	return h.Redis.Set(ctx, key, raw, ttl).Err()
}

func (h *Handler) takeWebAuthnSession(ctx context.Context, kind, userID string) (*webauthn.SessionData, error) {
	if h.Redis == nil {
		return nil, fmt.Errorf("ADMIN_WEBAUTHN_UNAVAILABLE")
	}
	key := "admin:webauthn:" + kind + ":" + userID
	raw, err := h.Redis.Get(ctx, key).Bytes()
	if err != nil || len(raw) == 0 {
		return nil, fmt.Errorf("ADMIN_WEBAUTHN_UNAVAILABLE")
	}
	_ = h.Redis.Del(ctx, key).Err()
	var session webauthn.SessionData
	if err := json.Unmarshal(raw, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (h *Handler) issueStepUpToken(ctx context.Context, r *http.Request, userID, method string) (string, int, error) {
	if h.Redis == nil {
		return "", 0, fmt.Errorf("ADMIN_STEP_UP_INVALID")
	}
	tok, err := randomStepUpToken()
	if err != nil {
		return "", 0, err
	}
	ttl, ok := h.stepUpTTL()
	if !ok {
		return "", 0, fmt.Errorf("ADMIN_STEP_UP_TTL_MISSING")
	}
	key := "admin:stepup:" + userID
	if err := h.Redis.Set(ctx, key, tok, ttl).Err(); err != nil {
		return "", 0, err
	}
	sec := int(ttl.Seconds())
	h.audit(ctx, r, "admin.step_up", "session", userID, nil, map[string]any{
		"expires_in_sec": sec, "method": method,
	}, "", false)
	return tok, sec, nil
}

func (h *Handler) getWebAuthnStatus(w http.ResponseWriter, r *http.Request) {
	db := h.readPool()
	p := PrincipalFromContext(r.Context())
	enrolled, err := h.webauthnEnrolled(r.Context(), p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	label := h.msg("ADMIN_WEBAUTHN_STATUS_OFF")
	if enrolled {
		label = h.msg("ADMIN_WEBAUTHN_STATUS_ON")
	}
	_, rpOK := h.webauthnRPID()
	items := make([]map[string]any, 0)
	rows, err := db.Query(r.Context(), `
		select id::text, coalesce(friendly_name, ''), created_at::text, coalesce(last_used_at::text, '')
		from public.platform_admin_webauthn_credentials
		where user_id = $1::uuid
		order by created_at asc
	`, p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, name, created, last string
		if err := rows.Scan(&id, &name, &created, &last); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		items = append(items, map[string]any{
			"id": id, "friendly_name": name, "created_at": created, "last_used_at": last,
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"enrolled":      enrolled,
		"status_label":  label,
		"rp_configured": rpOK && len(h.Config.AdminAllowedOrigins) > 0,
		"items":         items,
	})
}

func (h *Handler) postWebAuthnRegisterBegin(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	wa, err := h.newWebAuthn()
	if err != nil {
		h.writeErr(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	user, err := h.loadWebAuthnUser(r.Context(), p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	options, session, err := wa.BeginRegistration(user)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_WEBAUTHN_REGISTER_FAILED")
		return
	}
	if err := h.storeWebAuthnSession(r.Context(), "reg", p.UserID, session); err != nil {
		h.writeErr(w, http.StatusServiceUnavailable, "ADMIN_WEBAUTHN_UNAVAILABLE")
		return
	}
	h.audit(r.Context(), r, "admin.webauthn_register_begin", "platform_admin_webauthn_credentials", p.UserID, nil, map[string]any{"pending": true}, "", false)
	h.writeJSON(w, http.StatusOK, options)
}

func (h *Handler) postWebAuthnRegisterFinish(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	wa, err := h.newWebAuthn()
	if err != nil {
		h.writeErr(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	session, err := h.takeWebAuthnSession(r.Context(), "reg", p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_WEBAUTHN_UNAVAILABLE")
		return
	}
	user, err := h.loadWebAuthnUser(r.Context(), p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	parsed, err := protocol.ParseCredentialCreationResponseBody(r.Body)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_WEBAUTHN_REGISTER_FAILED")
		return
	}
	cred, err := wa.CreateCredential(user, *session, parsed)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_WEBAUTHN_REGISTER_FAILED")
		return
	}
	friendly := strings.TrimSpace(r.URL.Query().Get("friendly_name"))
	if len(friendly) > 120 {
		friendly = friendly[:120]
	}
	transports := make([]string, 0, len(cred.Transport))
	for _, t := range cred.Transport {
		transports = append(transports, string(t))
	}
	var id string
	err = h.DB.QueryRow(r.Context(), `
		insert into public.platform_admin_webauthn_credentials
		  (user_id, credential_id, public_key, attestation_type, transport, sign_count, clone_warning, aaguid, friendly_name)
		values ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9)
		returning id::text
	`, p.UserID, cred.ID, cred.PublicKey, cred.AttestationType, transports,
		int64(cred.Authenticator.SignCount), cred.Authenticator.CloneWarning,
		cred.Authenticator.AAGUID, friendly).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			h.writeErr(w, http.StatusConflict, "ADMIN_WEBAUTHN_ALREADY")
			return
		}
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	h.audit(r.Context(), r, "admin.webauthn_register", "platform_admin_webauthn_credentials", id, nil, map[string]any{"enrolled": true}, "", false)
	// Registration ceremony proves possession - mint elevation for the step-up TTL.
	tok, sec, terr := h.issueStepUpToken(r.Context(), r, p.UserID, "webauthn_enroll")
	if terr != nil {
		code := terr.Error()
		if strings.HasPrefix(code, "ADMIN_") {
			h.writeErr(w, http.StatusServiceUnavailable, code)
			return
		}
		h.writeErr(w, http.StatusInternalServerError, "ADMIN_STEP_UP_INVALID")
		return
	}
	h.writeJSON(w, http.StatusCreated, map[string]any{
		"id":             id,
		"step_up_token":  tok,
		"expires_in_sec": sec,
	})
}

func (h *Handler) postWebAuthnAssertBegin(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	wa, err := h.newWebAuthn()
	if err != nil {
		h.writeErr(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	user, err := h.loadWebAuthnUser(r.Context(), p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if len(user.creds) == 0 {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_WEBAUTHN_NOT_ENROLLED")
		return
	}
	options, session, err := wa.BeginLogin(user)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_WEBAUTHN_ASSERT_FAILED")
		return
	}
	if err := h.storeWebAuthnSession(r.Context(), "auth", p.UserID, session); err != nil {
		h.writeErr(w, http.StatusServiceUnavailable, "ADMIN_WEBAUTHN_UNAVAILABLE")
		return
	}
	h.writeJSON(w, http.StatusOK, options)
}

func (h *Handler) postWebAuthnAssertFinish(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	wa, err := h.newWebAuthn()
	if err != nil {
		h.writeErr(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	session, err := h.takeWebAuthnSession(r.Context(), "auth", p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_WEBAUTHN_UNAVAILABLE")
		return
	}
	user, err := h.loadWebAuthnUser(r.Context(), p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	parsed, err := protocol.ParseCredentialRequestResponseBody(r.Body)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_WEBAUTHN_ASSERT_FAILED")
		return
	}
	cred, err := wa.ValidateLogin(user, *session, parsed)
	if err != nil {
		h.writeErr(w, http.StatusForbidden, "ADMIN_WEBAUTHN_ASSERT_FAILED")
		return
	}
	_, err = h.DB.Exec(r.Context(), `
		update public.platform_admin_webauthn_credentials set
		  sign_count = $2,
		  clone_warning = $3,
		  last_used_at = now()
		where user_id = $1::uuid and credential_id = $4
	`, p.UserID, int64(cred.Authenticator.SignCount), cred.Authenticator.CloneWarning, cred.ID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	tok, sec, err := h.issueStepUpToken(r.Context(), r, p.UserID, "webauthn")
	if err != nil {
		code := err.Error()
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

func (h *Handler) deleteWebAuthnCredential(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_WEBAUTHN_NOT_ENROLLED")
		return
	}
	ct, err := h.DB.Exec(r.Context(), `
		delete from public.platform_admin_webauthn_credentials
		where id = $1::uuid and user_id = $2::uuid
	`, id, p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if ct.RowsAffected() == 0 {
		h.writeErr(w, http.StatusNotFound, "ADMIN_WEBAUTHN_NOT_ENROLLED")
		return
	}
	h.audit(r.Context(), r, "admin.webauthn_remove", "platform_admin_webauthn_credentials", id, nil, map[string]any{"removed": true}, "", true)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_WEBAUTHN_REMOVED")})
}
