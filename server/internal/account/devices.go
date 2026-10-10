package account

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/usetrim/trim/server/internal/middleware"
)

type RegisterAPIKeyDeviceRequest struct {
	HardwareUUID string `json:"hardware_uuid"`
	AgentID      string `json:"agent_id"`
}

type APIKeyDeviceRow struct {
	HardwareUUID string `json:"hardware_uuid"`
	AgentID      string `json:"agent_id,omitempty"`
	FirstSeenAt  string `json:"first_seen_at"`
	LastSeenAt   string `json:"last_seen_at"`
}

// ListAPIKeyDevices returns enrolled hardware for a key owned by the caller.
// GET /api/v1/me/api-keys/{keyId}/devices
func (h *Handler) ListAPIKeyDevices(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	keyID := strings.TrimSpace(chi.URLParam(r, "keyId"))
	if keyID == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusBadRequest, "API_KEY_ID_REQUIRED")
		return
	}
	ctx := r.Context()
	var owned bool
	if err := h.DB.QueryRow(ctx, `
		select exists(
			select 1 from public.api_keys
			where id = $1::uuid and user_id = $2::uuid and revoked = false
		)
	`, keyID, userID).Scan(&owned); err != nil || !owned {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusNotFound, "API_KEY_NOT_FOUND_OR_REVOKED")
		return
	}

	db := h.readPool()
	rows, err := db.Query(ctx, `
		select hardware_uuid,
		       coalesce(agent_id, ''),
		       first_seen_at::text,
		       last_seen_at::text
		from public.api_key_devices
		where key_id = $1::uuid
		order by first_seen_at asc
	`, keyID)
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "ACCOUNT_OPERATION_FAILED")
		return
	}
	defer rows.Close()

	items := make([]APIKeyDeviceRow, 0)
	for rows.Next() {
		var row APIKeyDeviceRow
		if err := rows.Scan(&row.HardwareUUID, &row.AgentID, &row.FirstSeenAt, &row.LastSeenAt); err != nil {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "ACCOUNT_OPERATION_FAILED")
			return
		}
		items = append(items, row)
	}

	// Surface legacy api_keys.hardware_uuid when not yet in api_key_devices.
	var legacyHW string
	_ = db.QueryRow(ctx, `
		select coalesce(hardware_uuid, '') from public.api_keys where id = $1::uuid
	`, keyID).Scan(&legacyHW)
	legacyHW = strings.TrimSpace(legacyHW)
	if legacyHW != "" {
		found := false
		for _, it := range items {
			if strings.EqualFold(it.HardwareUUID, legacyHW) {
				found = true
				break
			}
		}
		if !found {
			items = append([]APIKeyDeviceRow{{
				HardwareUUID: legacyHW,
				FirstSeenAt:  "",
				LastSeenAt:   "",
			}}, items...)
		}
	}

	ch := siteMsgMap(r.Context(), db, apiKeyDeviceChromeCodes)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"items":              items,
		"device_hint":        siteMsg(ch, "API_KEY_DEVICE_HINT"),
		"device_hw_label":    siteMsg(ch, "API_KEY_DEVICE_HW_LABEL"),
		"device_agent_label": siteMsg(ch, "API_KEY_DEVICE_AGENT_LABEL"),
		"device_agent_options": []map[string]string{
			{"id": "cli", "label": siteMsg(ch, "API_KEY_DEVICE_AGENT_CLI")},
			{"id": "ide", "label": siteMsg(ch, "API_KEY_DEVICE_AGENT_IDE")},
			{"id": "ci", "label": siteMsg(ch, "API_KEY_DEVICE_AGENT_CI")},
		},
		"device_register_label":   siteAction(ch, "API_KEY_DEVICE_REGISTER"),
		"device_register_pending": sitePending(ch, "API_KEY_DEVICE_REGISTER"),
		"device_remove_label":     siteAction(ch, "API_KEY_DEVICE_REMOVE"),
		"device_remove_pending":   sitePending(ch, "API_KEY_DEVICE_REMOVE"),
		"ci_agent_hint":           siteMsg(ch, "CI_AGENT_HINT"),
	})
}

// RegisterAPIKeyDevice enrolls hardware for a key (JWT session only - never via stolen API key).
// POST /api/v1/me/api-keys/{keyId}/devices
func (h *Handler) RegisterAPIKeyDevice(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	keyID := strings.TrimSpace(chi.URLParam(r, "keyId"))
	if keyID == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusBadRequest, "API_KEY_ID_REQUIRED")
		return
	}

	var req RegisterAPIKeyDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	hw := strings.TrimSpace(req.HardwareUUID)
	agentID := strings.ToLower(strings.TrimSpace(req.AgentID))
	if hw == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusBadRequest, "API_KEY_DEVICE_HW_REQUIRED")
		return
	}
	if agentID == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusBadRequest, "AGENT_ID_REQUIRED")
		return
	}

	ctx := r.Context()
	var legacyHW *string
	if err := h.DB.QueryRow(ctx, `
		select hardware_uuid
		from public.api_keys
		where id = $1::uuid and user_id = $2::uuid and revoked = false
	`, keyID, userID).Scan(&legacyHW); err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusNotFound, "API_KEY_NOT_FOUND_OR_REVOKED")
		return
	}

	var agentOK bool
	if err := h.DB.QueryRow(ctx, `
		select exists(select 1 from public.agent_identity_catalog where id = $1)
	`, agentID).Scan(&agentOK); err != nil || !agentOK {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusBadRequest, "AGENT_ID_UNKNOWN")
		return
	}

	var maxDevices *int
	if err := h.DB.QueryRow(ctx, `
		select api_key_max_devices from public.admin_product_settings where id = 'default'
	`).Scan(&maxDevices); err != nil || maxDevices == nil || *maxDevices < 1 {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "API_KEY_MAX_DEVICES_MISSING")
		return
	}

	var deviceCount int
	if err := h.DB.QueryRow(ctx, `
		select count(*)::int from public.api_key_devices where key_id = $1::uuid
	`, keyID).Scan(&deviceCount); err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "ACCOUNT_OPERATION_FAILED")
		return
	}
	legacy := ""
	if legacyHW != nil {
		legacy = strings.TrimSpace(*legacyHW)
	}
	already := false
	if err := h.DB.QueryRow(ctx, `
		select exists(
			select 1 from public.api_key_devices
			where key_id = $1::uuid and hardware_uuid = $2
		)
	`, keyID, hw).Scan(&already); err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "ACCOUNT_OPERATION_FAILED")
		return
	}
	effective := deviceCount
	if legacy != "" {
		var legacyInTable bool
		_ = h.DB.QueryRow(ctx, `
			select exists(
				select 1 from public.api_key_devices
				where key_id = $1::uuid and hardware_uuid = $2
			)
		`, keyID, legacy).Scan(&legacyInTable)
		if !legacyInTable {
			effective++
		}
	}
	if !already && !strings.EqualFold(hw, legacy) && effective >= *maxDevices {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusForbidden, "HARDWARE_DEVICE_LIMIT")
		return
	}

	_, err := h.DB.Exec(ctx, `
		insert into public.api_key_devices (key_id, hardware_uuid, agent_id, first_seen_at, last_seen_at)
		values ($1::uuid, $2, $3, now(), now())
		on conflict (key_id, hardware_uuid) do update
		set last_seen_at = now(),
		    agent_id = coalesce(excluded.agent_id, public.api_key_devices.agent_id)
	`, keyID, hw, agentID)
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "ACCOUNT_OPERATION_FAILED")
		return
	}
	_, _ = h.DB.Exec(ctx, `
		insert into public.device_fingerprints (user_id, hardware_uuid, last_seen_at)
		values ($1::uuid, $2, now())
		on conflict (user_id, hardware_uuid) do update set last_seen_at = now()
	`, userID, hw)

	ch := siteMsgMap(r.Context(), h.readPool(), []string{
		"API_KEY_DEVICE_REGISTERED",
		"ACTION:API_KEY_DEVICE_REGISTER", "PENDING:API_KEY_DEVICE_REGISTER",
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":        "registered",
		"action_label":  siteAction(ch, "API_KEY_DEVICE_REGISTER"),
		"pending_label": sitePending(ch, "API_KEY_DEVICE_REGISTER"),
		"message":       siteMsg(ch, "API_KEY_DEVICE_REGISTERED"),
	})
}

// DeleteAPIKeyDevice removes an enrolled hardware UUID from a key.
// DELETE /api/v1/me/api-keys/{keyId}/devices/{hardwareUUID}
func (h *Handler) DeleteAPIKeyDevice(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	keyID := strings.TrimSpace(chi.URLParam(r, "keyId"))
	hw := strings.TrimSpace(chi.URLParam(r, "hardwareUUID"))
	if keyID == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusBadRequest, "API_KEY_ID_REQUIRED")
		return
	}
	if hw == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusBadRequest, "API_KEY_DEVICE_HW_REQUIRED")
		return
	}

	ctx := r.Context()
	var owned bool
	if err := h.DB.QueryRow(ctx, `
		select exists(
			select 1 from public.api_keys
			where id = $1::uuid and user_id = $2::uuid and revoked = false
		)
	`, keyID, userID).Scan(&owned); err != nil || !owned {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusNotFound, "API_KEY_NOT_FOUND_OR_REVOKED")
		return
	}

	tag, err := h.DB.Exec(ctx, `
		delete from public.api_key_devices
		where key_id = $1::uuid and hardware_uuid = $2
	`, keyID, hw)
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "ACCOUNT_OPERATION_FAILED")
		return
	}
	if tag.RowsAffected() == 0 {
		// Clearing legacy column counts as remove when table had no row.
		tag2, err2 := h.DB.Exec(ctx, `
			update public.api_keys
			set hardware_uuid = null
			where id = $1::uuid and user_id = $2::uuid
			  and hardware_uuid is not null
			  and lower(hardware_uuid) = lower($3)
		`, keyID, userID, hw)
		if err2 != nil {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "ACCOUNT_OPERATION_FAILED")
			return
		}
		if tag2.RowsAffected() == 0 {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusNotFound, "API_KEY_DEVICE_NOT_FOUND")
			return
		}
	} else {
		// Keep legacy column consistent if it pointed at the removed device.
		_, _ = h.DB.Exec(ctx, `
			update public.api_keys
			set hardware_uuid = null
			where id = $1::uuid and user_id = $2::uuid
			  and hardware_uuid is not null
			  and lower(hardware_uuid) = lower($3)
		`, keyID, userID, hw)
	}

	ch := siteMsgMap(r.Context(), h.readPool(), []string{
		"API_KEY_DEVICE_REMOVED",
		"ACTION:API_KEY_DEVICE_REMOVE", "PENDING:API_KEY_DEVICE_REMOVE",
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":        "removed",
		"action_label":  siteAction(ch, "API_KEY_DEVICE_REMOVE"),
		"pending_label": sitePending(ch, "API_KEY_DEVICE_REMOVE"),
		"message":       siteMsg(ch, "API_KEY_DEVICE_REMOVED"),
	})
}
