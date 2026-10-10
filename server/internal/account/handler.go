package account

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usetrim/trim/server/internal/authsettings"
	"github.com/usetrim/trim/server/internal/billingsettings"
	"github.com/usetrim/trim/server/internal/middleware"
	"github.com/usetrim/trim/server/internal/pagination"
	"github.com/usetrim/trim/server/internal/subscriptions"
	"github.com/usetrim/trim/server/pkg/provideradapt"
)

type Handler struct {
	DB     *pgxpool.Pool
	ReadDB *pgxpool.Pool
}

func NewHandler(db, readDB *pgxpool.Pool) *Handler {
	return &Handler{DB: db, ReadDB: readDB}
}

func (h *Handler) readPool() *pgxpool.Pool {
	if h.ReadDB != nil {
		return h.ReadDB
	}
	return h.DB
}

type CreateAPIKeyRequest struct {
	HardwareUUID string `json:"hardware_uuid"`
	AgentID      string `json:"agent_id"`
	Label        string `json:"label"`
}

type CreateAPIKeyResponse struct {
	APIKey                  string `json:"api_key"`
	ID                      string `json:"id"`
	KeyPrefix               string `json:"key_prefix"`
	CreatedAt               string `json:"created_at"`
	DeviceBound             bool   `json:"device_bound"`
	ActionLabel             string `json:"action_label"`
	PendingLabel            string `json:"pending_label"`
	RetryActionLabel        string `json:"retry_action_label"`
	IssueAnotherActionLabel string `json:"issue_another_action_label"`
	CopyActionLabel         string `json:"copy_action_label"`
	CopyPendingLabel        string `json:"copy_pending_label"`
	CopiedActionLabel       string `json:"copied_action_label"`
	FreshKeyHint            string `json:"fresh_key_hint"`
	DeviceHint              string `json:"device_hint,omitempty"`
	Message                 string `json:"message"`
}

type APIKeyRow struct {
	ID          string  `json:"id"`
	KeyPrefix   string  `json:"key_prefix"`
	Revoked     bool    `json:"revoked"`
	CreatedAt   string  `json:"created_at"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
	DeviceCount int     `json:"device_count"`
	DeviceBound bool    `json:"device_bound"`
}

// ListAPIKeys returns non-secret key metadata for the dashboard.
// GET /api/v1/me/api-keys?skip=&limit=&q=
func (h *Handler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}

	db := h.readPool()
	maxLimit, err := billingsettings.MaxPageSize(r.Context(), db)
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, err.Error())
		return
	}
	skipCap, err := billingsettings.SkipToMaxPages(r.Context(), db)
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, err.Error())
		return
	}
	params, err := pagination.Parse(r, maxLimit)
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusBadRequest, err.Error())
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 200 {
		q = q[:200]
	}

	where := `k.user_id = $1::uuid`
	args := []interface{}{userID}
	argN := 2
	if q != "" {
		like := "%" + escapeILikePattern(q) + "%"
		where += fmt.Sprintf(` and (
			coalesce(k.key_prefix, '') ilike $%d escape '\'
			or k.id::text ilike $%d escape '\'
			or case when k.revoked then 'revoked' else 'active' end ilike $%d escape '\'
		)`, argN, argN, argN)
		args = append(args, like)
		argN++
	}

	var total int
	countSQL := `select count(*) from public.api_keys k where ` + where
	if err := db.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "API_KEYS_COUNT_FAILED")
		return
	}

	listSQL := fmt.Sprintf(`
		select k.id::text, k.key_prefix, k.revoked, k.created_at::text,
		       case when k.expires_at is null then null else k.expires_at::text end,
		       (select count(*)::int from public.api_key_devices d where d.key_id = k.id)
		         + case when k.hardware_uuid is not null
		                 and not exists (
		                   select 1 from public.api_key_devices d2
		                   where d2.key_id = k.id and d2.hardware_uuid = k.hardware_uuid
		                 )
		                then 1 else 0 end as device_count
		from public.api_keys k
		where %s
		order by k.created_at desc
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]interface{}{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "ACCOUNT_OPERATION_FAILED")
		return
	}
	defer rows.Close()

	items := make([]APIKeyRow, 0)
	for rows.Next() {
		var row APIKeyRow
		var expires *string
		if err := rows.Scan(&row.ID, &row.KeyPrefix, &row.Revoked, &row.CreatedAt, &expires, &row.DeviceCount); err != nil {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "API_KEY_SCAN_FAILED")
			return
		}
		row.ExpiresAt = expires
		row.DeviceBound = row.DeviceCount > 0
		items = append(items, row)
	}

	w.Header().Set("Content-Type", "application/json")
	ch := siteMsgMap(r.Context(), db, apiKeyListChromeCodes)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"items":                   items,
		"meta":                    pagination.BuildMeta(params, total, skipCap),
		"intro_message":           siteMsg(ch, "API_KEY_INTRO"),
		"empty_message":           siteMsg(ch, "API_KEY_EMPTY"),
		"status_active_label":     siteMsg(ch, "API_KEY_STATUS_ACTIVE"),
		"status_revoked_label":    siteMsg(ch, "API_KEY_STATUS_REVOKED"),
		"created_prefix_label":    siteMsg(ch, "API_KEY_CREATED_PREFIX"),
		"issue_action_label":      siteAction(ch, "API_KEY_ISSUE"),
		"issue_pending_label":     sitePending(ch, "API_KEY_ISSUE"),
		"revoke_action_label":     siteAction(ch, "API_KEY_REVOKE"),
		"revoke_pending_label":    sitePending(ch, "API_KEY_REVOKE"),
		"revoke_confirm_message":  siteMsg(ch, "API_KEY_REVOKE_CONFIRM"),
		"refresh_action_label":    siteAction(ch, "API_KEY_LIST_REFRESH"),
		"refresh_pending_label":   sitePending(ch, "API_KEY_LIST_REFRESH"),
		"key_prefix_ellipsis":     siteMsg(ch, "API_KEY_PREFIX_ELLIPSIS"),
		"table_select_all":        siteMsg(ch, "TABLE_SELECT_ALL"),
		"table_select_row":        siteMsg(ch, "TABLE_SELECT_ROW"),
		"table_selected_fmt":      siteMsg(ch, "TABLE_SELECTED_FMT"),
		"table_row_actions":       siteMsg(ch, "TABLE_ROW_ACTIONS"),
		"table_bulk_revoke":       siteMsg(ch, "TABLE_BULK_REVOKE"),
		"table_clear_selection":   siteMsg(ch, "TABLE_CLEAR_SELECTION"),
		"device_hint":             siteMsg(ch, "API_KEY_DEVICE_HINT"),
		"device_register_label":   siteAction(ch, "API_KEY_DEVICE_REGISTER"),
		"device_register_pending": sitePending(ch, "API_KEY_DEVICE_REGISTER"),
		"device_hw_label":         siteMsg(ch, "API_KEY_DEVICE_HW_LABEL"),
		"device_agent_label":      siteMsg(ch, "API_KEY_DEVICE_AGENT_LABEL"),
		"device_agent_options": []map[string]string{
			{"id": "cli", "label": siteMsg(ch, "API_KEY_DEVICE_AGENT_CLI")},
			{"id": "ide", "label": siteMsg(ch, "API_KEY_DEVICE_AGENT_IDE")},
			{"id": "ci", "label": siteMsg(ch, "API_KEY_DEVICE_AGENT_CI")},
		},
		"device_bound_label":   siteMsg(ch, "API_KEY_DEVICE_BOUND_LABEL"),
		"device_unbound_label": siteMsg(ch, "API_KEY_DEVICE_UNBOUND_LABEL"),
		"device_count_fmt":     siteMsg(ch, "API_KEY_DEVICE_COUNT_FMT"),
		"ci_agent_hint":        siteMsg(ch, "CI_AGENT_HINT"),
		"search_placeholder":   siteMsg(ch, "API_KEY_SEARCH"),
		"search_description":   siteMsg(ch, "API_KEY_SEARCH_DESC"),
		"q":                    q,
	})
}

func escapeILikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// RevokeAPIKey soft-revokes a key so CLI auth rejects it.
// DELETE /api/v1/me/api-keys/{keyId}
func (h *Handler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
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

	tag, err := h.DB.Exec(r.Context(), `
		update public.api_keys
		set revoked = true
		where id = $1::uuid and user_id = $2::uuid and revoked = false
	`, keyID, userID)
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "ACCOUNT_OPERATION_FAILED")
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusNotFound, "API_KEY_NOT_FOUND_OR_REVOKED")
		return
	}

	ch := siteMsgMap(r.Context(), h.readPool(), []string{
		"API_KEY_REVOKED",
		"ACTION:API_KEY_REVOKE", "PENDING:API_KEY_REVOKE",
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":        "revoked",
		"action_label":  siteAction(ch, "API_KEY_REVOKE"),
		"pending_label": sitePending(ch, "API_KEY_REVOKE"),
		"message":       siteMsg(ch, "API_KEY_REVOKED"),
	})
}

// CreateAPIKey issues a one-time visible CLI API key for the authenticated user.
// POST /api/v1/me/api-keys
func (h *Handler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}

	var req CreateAPIKeyRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	req.HardwareUUID = strings.TrimSpace(req.HardwareUUID)
	req.AgentID = strings.ToLower(strings.TrimSpace(req.AgentID))

	rawKey, err := generateAPIKey()
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "API_KEY_GENERATE_FAILED")
		return
	}
	keyHash := hashKey(rawKey)
	prefix := rawKey
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}

	ctx := r.Context()
	var hw interface{}
	if req.HardwareUUID != "" {
		hw = req.HardwareUUID
		_, _ = h.DB.Exec(ctx, `
			insert into public.device_fingerprints (user_id, hardware_uuid, last_seen_at)
			values ($1, $2, now())
			on conflict (user_id, hardware_uuid) do update set last_seen_at = now()
		`, userID, req.HardwareUUID)
	}

	var agent interface{}
	if req.AgentID != "" {
		var exists bool
		if err := h.DB.QueryRow(ctx, `
			select exists(select 1 from public.agent_identity_catalog where id = $1)
		`, req.AgentID).Scan(&exists); err != nil || !exists {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusBadRequest, "AGENT_ID_UNKNOWN")
			return
		}
		agent = req.AgentID
	}

	var keyID string
	err = h.DB.QueryRow(ctx, `
		insert into public.api_keys (user_id, key_hash, key_prefix, hardware_uuid, agent_id)
		values ($1, $2, $3, $4, $5)
		returning id::text
	`, userID, keyHash, prefix, hw, agent).Scan(&keyID)
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "ACCOUNT_OPERATION_FAILED")
		return
	}
	if req.HardwareUUID != "" && keyID != "" {
		_, _ = h.DB.Exec(ctx, `
			insert into public.api_key_devices (key_id, hardware_uuid, agent_id, first_seen_at, last_seen_at)
			values ($1::uuid, $2, $3, now(), now())
			on conflict (key_id, hardware_uuid) do update set last_seen_at = now()
		`, keyID, req.HardwareUUID, agent)
	}

	w.Header().Set("Content-Type", "application/json")
	deviceBound := req.HardwareUUID != ""
	ch := siteMsgMap(r.Context(), h.readPool(), apiKeyCreateChromeCodes)
	freshHint := siteMsg(ch, "API_KEY_FRESH_HINT")
	deviceHint := ""
	if !deviceBound {
		freshHint = siteMsg(ch, "API_KEY_FRESH_DEVICE_REQUIRED")
		deviceHint = siteMsg(ch, "API_KEY_DEVICE_HINT")
	}
	_ = json.NewEncoder(w).Encode(CreateAPIKeyResponse{
		APIKey:                  rawKey,
		ID:                      keyID,
		KeyPrefix:               prefix,
		CreatedAt:               time.Now().UTC().Format(time.RFC3339),
		DeviceBound:             deviceBound,
		ActionLabel:             siteAction(ch, "API_KEY_ISSUE"),
		PendingLabel:            sitePending(ch, "API_KEY_ISSUE"),
		RetryActionLabel:        siteAction(ch, "API_KEY_ISSUE_RETRY"),
		IssueAnotherActionLabel: siteAction(ch, "API_KEY_ISSUE_ANOTHER"),
		CopyActionLabel:         siteAction(ch, "API_KEY_COPY"),
		CopyPendingLabel:        sitePending(ch, "API_KEY_COPY"),
		CopiedActionLabel:       siteAction(ch, "API_KEY_COPIED"),
		FreshKeyHint:            freshHint,
		DeviceHint:              deviceHint,
		Message:                 siteMsg(ch, "API_KEY_CREATED"),
	})
}

type PreferencesEngineOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Hint  string `json:"hint,omitempty"`
}

type LinkedIdentity struct {
	IdentityID      string `json:"identity_id"`
	Provider        string `json:"provider"`
	ProviderDisplay string `json:"provider_display"`
	Email           string `json:"email,omitempty"`
	LastSignInAt    string `json:"last_sign_in_at,omitempty"`
	IsPrimary       bool   `json:"is_primary"`
	IsLastUsed      bool   `json:"is_last_used"`
}

type LinkableProvider struct {
	ID           string `json:"id"`
	DisplayName  string `json:"display_name"`
	ActionLabel  string `json:"action_label"`
	PendingLabel string `json:"pending_label"`
}

type PreferencesResponse struct {
	CompressionTier          string                    `json:"compression_tier"`
	DeepEngine               string                    `json:"deep_engine"`
	DeepTargetToken          int                       `json:"deep_target_token"`
	AutoStartWithIDE         bool                      `json:"auto_start_with_ide"`
	FastTierID               string                    `json:"fast_tier_id"`
	DeepTierID               string                    `json:"deep_tier_id"`
	Note                     string                    `json:"note"`
	SavedMessage             string                    `json:"saved_message"`
	EngineOptions            []PreferencesEngineOption `json:"engine_options"`
	SaveActionLabel          string                    `json:"save_action_label"`
	SavePendingLabel         string                    `json:"save_pending_label"`
	PageTitle                string                    `json:"page_title"`
	PageDescription          string                    `json:"page_description"`
	DeepHint                 string                    `json:"deep_hint"`
	EngineHint               string                    `json:"engine_hint"`
	TargetHint               string                    `json:"target_hint"`
	AutoStartTitle           string                    `json:"auto_start_title"`
	AutoStartLabel           string                    `json:"auto_start_label"`
	AutoStartHint            string                    `json:"auto_start_hint"`
	AutoStartNote            string                    `json:"auto_start_note"`
	LocalAgentTitle          string                    `json:"local_agent_title"`
	LocalAgentStatus         string                    `json:"local_agent_status"`
	LocalAgentStatusLabel    string                    `json:"local_agent_status_label"`
	LocalAgentHint           string                    `json:"local_agent_hint"`
	BackActionLabel          string                    `json:"back_action_label"`
	BackHref                 string                    `json:"back_href"`
	TierTitle                string                    `json:"tier_title"`
	DeepLabel                string                    `json:"deep_label"`
	EngineLabel              string                    `json:"engine_label"`
	TargetLabel              string                    `json:"target_label"`
	DeepTargetTokenMin       int                       `json:"deep_target_token_min"`
	DeepTargetTokenMax       int                       `json:"deep_target_token_max"`
	LiveDeepMinInputTokens   int                       `json:"live_deep_min_input_tokens"`
	LiveDeepOOMPolicy        string                    `json:"live_deep_oom_policy"`
	LiveDeepWarmupOnStart    bool                      `json:"live_deep_warmup_on_start"`
	LiveDeepSkipOnStream     bool                      `json:"live_deep_skip_on_stream"`
	DeepV1Model              string                    `json:"deep_v1_model"`
	DeepV2Model              string                    `json:"deep_v2_model"`
	DeepLongModel            string                    `json:"deep_long_model"`
	DeepV2ForceTokens        []string                  `json:"deep_v2_force_tokens"`
	LiveDeepMinHint          string                    `json:"live_deep_min_hint"`
	LiveDeepStreamHint       string                    `json:"live_deep_stream_hint"`
	ProviderAdapters         []provideradapt.AdapterConfig      `json:"provider_adapters"`
	OpenAIModelAliases       []provideradapt.ModelAlias         `json:"openai_model_aliases"`
	DiscoverableModels       []provideradapt.DiscoverableModel  `json:"provider_discoverable_models"`
	ProviderAdaptersSynced   bool                               `json:"provider_adapters_synced"`
	CLITitle                 string                    `json:"cli_title"`
	CLIHelpLines             []string                  `json:"cli_help_lines"`
	KeysTitle                string                    `json:"keys_title"`
	AuthProvider             string                    `json:"auth_provider"`
	AuthProviderDisplay      string                    `json:"auth_provider_display"`
	AccountTitle             string                    `json:"account_title"`
	AuthProviderPrefix       string                    `json:"auth_provider_prefix"`
	LinkedIdentities         []LinkedIdentity          `json:"linked_identities"`
	LinkableProviders        []LinkableProvider        `json:"linkable_providers"`
	LinkHint                 string                    `json:"link_hint"`
	UnlinkActionLabel        string                    `json:"unlink_action_label"`
	UnlinkPendingLabel       string                    `json:"unlink_pending_label"`
	UnlinkConfirmMessage     string                    `json:"unlink_confirm_message"`
	UnlinkLastBlockedMessage string                    `json:"unlink_last_blocked_message"`
	UnlinkFailedMessage      string                    `json:"unlink_failed_message"`
	UnlinkDoneMessage        string                    `json:"unlink_done_message"`
	IdentityPrimaryLabel     string                    `json:"identity_primary_label"`
	IdentityLastUsedLabel    string                    `json:"identity_last_used_label"`
	CanUnlink                bool                      `json:"can_unlink"`
	TableRowActions          string                    `json:"table_row_actions"`
}

func preferencesChromeFromDB(ctx context.Context, db *pgxpool.Pool) (PreferencesResponse, bool) {
	codes := []string{
		"COMPRESSION_TIER_FAST", "COMPRESSION_TIER_DEEP",
		"PREFERENCES_NOTE", "PREFERENCES_SAVED", "PREFERENCES_PAGE_TITLE", "PREFERENCES_PAGE_DESCRIPTION",
		"PREFERENCES_DEEP_HINT", "PREFERENCES_ENGINE_HINT", "PREFERENCES_TARGET_HINT", "PREFERENCES_LIVE_DEEP_MIN_HINT", "PREFERENCES_LIVE_DEEP_STREAM_HINT",
		"PREFERENCES_AUTO_START_TITLE", "PREFERENCES_AUTO_START_LABEL", "PREFERENCES_AUTO_START_HINT", "PREFERENCES_AUTO_START_NOTE",
		"PREFERENCES_LOCAL_AGENT_TITLE", "PREFERENCES_LOCAL_AGENT_HINT",
		"PREFERENCES_BACK_LABEL", "APP_PATH_DASHBOARD",
		"PREFERENCES_TIER_TITLE", "PREFERENCES_DEEP_LABEL", "PREFERENCES_ENGINE_LABEL", "PREFERENCES_TARGET_LABEL",
		"PREFERENCES_CLI_TITLE", "PREFERENCES_KEYS_TITLE",
		"PREFERENCES_CLI_HELP_1", "PREFERENCES_CLI_HELP_2", "PREFERENCES_CLI_HELP_3",
		"PREFERENCES_CLI_HELP_4", "PREFERENCES_CLI_HELP_5", "PREFERENCES_CLI_HELP_6",
		"PREFERENCES_CLI_HELP_7", "PREFERENCES_CLI_HELP_8", "PREFERENCES_CLI_HELP_9",
		"ENGINE_V2_LABEL", "ENGINE_LONG_LABEL", "ENGINE_V1_LABEL",
		"ENGINE_V2_HINT", "ENGINE_LONG_HINT", "ENGINE_V1_HINT",
		"PREFERENCES_ACCOUNT_TITLE", "PREFERENCES_AUTH_PROVIDER_PREFIX",
		"PREFERENCES_LINK_HINT", "PREFERENCES_UNLINK_CONFIRM", "PREFERENCES_UNLINK_LAST_BLOCKED",
		"PREFERENCES_UNLINK_FAILED", "PREFERENCES_UNLINK_DONE", "PREFERENCES_IDENTITY_PRIMARY_LABEL", "PREFERENCES_IDENTITY_LAST_USED_LABEL",
		"TABLE_ROW_ACTIONS",
		"ACTION:PREFERENCES_SAVE", "PENDING:PREFERENCES_SAVE",
		"ACTION:AUTH_UNLINK", "PENDING:AUTH_UNLINK",
	}
	m := subscriptions.LoadSiteMessages(ctx, db, codes)
	msg := func(code string) string { return strings.TrimSpace(subscriptions.SiteMsg(m, code)) }
	required := []string{
		"PREFERENCES_AUTO_START_TITLE", "PREFERENCES_AUTO_START_LABEL", "PREFERENCES_AUTO_START_HINT",
		"PREFERENCES_LOCAL_AGENT_TITLE", "PREFERENCES_LOCAL_AGENT_HINT",
		"COMPRESSION_TIER_FAST", "COMPRESSION_TIER_DEEP",
		"PREFERENCES_PAGE_TITLE",
		"ACTION:PREFERENCES_SAVE", "PENDING:PREFERENCES_SAVE",
	}
	for _, c := range required {
		if msg(c) == "" {
			return PreferencesResponse{}, false
		}
	}
	return PreferencesResponse{
		FastTierID:      msg("COMPRESSION_TIER_FAST"),
		DeepTierID:      msg("COMPRESSION_TIER_DEEP"),
		Note:            msg("PREFERENCES_NOTE"),
		SavedMessage:    msg("PREFERENCES_SAVED"),
		PageTitle:       msg("PREFERENCES_PAGE_TITLE"),
		PageDescription: msg("PREFERENCES_PAGE_DESCRIPTION"),
		DeepHint:        msg("PREFERENCES_DEEP_HINT"),
		EngineHint:      msg("PREFERENCES_ENGINE_HINT"),
		TargetHint:      msg("PREFERENCES_TARGET_HINT"),
		LiveDeepMinHint: msg("PREFERENCES_LIVE_DEEP_MIN_HINT"),
		LiveDeepStreamHint: msg("PREFERENCES_LIVE_DEEP_STREAM_HINT"),
		AutoStartTitle:  msg("PREFERENCES_AUTO_START_TITLE"),
		AutoStartLabel:  msg("PREFERENCES_AUTO_START_LABEL"),
		AutoStartHint:   msg("PREFERENCES_AUTO_START_HINT"),
		AutoStartNote:   msg("PREFERENCES_AUTO_START_NOTE"),
		LocalAgentTitle: msg("PREFERENCES_LOCAL_AGENT_TITLE"),
		LocalAgentHint:  msg("PREFERENCES_LOCAL_AGENT_HINT"),
		BackActionLabel: msg("PREFERENCES_BACK_LABEL"),
		BackHref:        msg("APP_PATH_DASHBOARD"),
		TierTitle:       msg("PREFERENCES_TIER_TITLE"),
		DeepLabel:       msg("PREFERENCES_DEEP_LABEL"),
		EngineLabel:     msg("PREFERENCES_ENGINE_LABEL"),
		TargetLabel:     msg("PREFERENCES_TARGET_LABEL"),
		CLITitle:        msg("PREFERENCES_CLI_TITLE"),
		KeysTitle:       msg("PREFERENCES_KEYS_TITLE"),
		CLIHelpLines: filterNonEmptyStrings([]string{
			msg("PREFERENCES_CLI_HELP_1"),
			msg("PREFERENCES_CLI_HELP_2"),
			msg("PREFERENCES_CLI_HELP_3"),
			msg("PREFERENCES_CLI_HELP_4"),
			msg("PREFERENCES_CLI_HELP_5"),
			msg("PREFERENCES_CLI_HELP_6"),
			msg("PREFERENCES_CLI_HELP_7"),
			msg("PREFERENCES_CLI_HELP_8"),
			msg("PREFERENCES_CLI_HELP_9"),
		}),
		EngineOptions: []PreferencesEngineOption{
			{ID: "v2", Label: msg("ENGINE_V2_LABEL"), Hint: msg("ENGINE_V2_HINT")},
			{ID: "long", Label: msg("ENGINE_LONG_LABEL"), Hint: msg("ENGINE_LONG_HINT")},
			{ID: "v1", Label: msg("ENGINE_V1_LABEL"), Hint: msg("ENGINE_V1_HINT")},
		},
		SaveActionLabel:          msg("ACTION:PREFERENCES_SAVE"),
		SavePendingLabel:         msg("PENDING:PREFERENCES_SAVE"),
		AccountTitle:             msg("PREFERENCES_ACCOUNT_TITLE"),
		AuthProviderPrefix:       msg("PREFERENCES_AUTH_PROVIDER_PREFIX"),
		LinkHint:                 msg("PREFERENCES_LINK_HINT"),
		UnlinkActionLabel:        msg("ACTION:AUTH_UNLINK"),
		UnlinkPendingLabel:       msg("PENDING:AUTH_UNLINK"),
		UnlinkConfirmMessage:     msg("PREFERENCES_UNLINK_CONFIRM"),
		UnlinkLastBlockedMessage: msg("PREFERENCES_UNLINK_LAST_BLOCKED"),
		UnlinkFailedMessage:      msg("PREFERENCES_UNLINK_FAILED"),
		UnlinkDoneMessage:        msg("PREFERENCES_UNLINK_DONE"),
		IdentityPrimaryLabel:     msg("PREFERENCES_IDENTITY_PRIMARY_LABEL"),
		IdentityLastUsedLabel:    msg("PREFERENCES_IDENTITY_LAST_USED_LABEL"),
		TableRowActions:          msg("TABLE_ROW_ACTIONS"),
		LinkedIdentities:         []LinkedIdentity{},
		LinkableProviders:        []LinkableProvider{},
	}, true
}

func attachAccountProvider(ctx context.Context, db *pgxpool.Pool, resp *PreferencesResponse, provider string) bool {
	id := strings.ToLower(strings.TrimSpace(provider))
	if id == "" {
		return false
	}
	code := "AUTH_PROVIDER_DISPLAY_" + strings.ToUpper(id)
	display := siteMsg(siteMsgMap(ctx, db, []string{code}), code)
	if display == "" {
		return false
	}
	if strings.TrimSpace(resp.AccountTitle) == "" || strings.TrimSpace(resp.AuthProviderPrefix) == "" {
		return false
	}
	resp.AuthProvider = id
	resp.AuthProviderDisplay = display
	return true
}

func (h *Handler) attachLinkedIdentities(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool, resp *PreferencesResponse, userID, primaryProvider string) bool {
	if db == nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return false
	}
	rows, err := db.Query(r.Context(), `
		select identity_id, provider, email, last_sign_in_at
		from public.list_auth_identities($1::uuid)
	`, userID)
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "ACCOUNT_IDENTITIES_UNAVAILABLE")
		return false
	}
	defer rows.Close()

	linked := make([]LinkedIdentity, 0)
	seen := map[string]struct{}{}
	for rows.Next() {
		var row LinkedIdentity
		var last *string
		if err := rows.Scan(&row.IdentityID, &row.Provider, &row.Email, &last); err != nil {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "ACCOUNT_IDENTITIES_UNAVAILABLE")
			return false
		}
		row.Provider = strings.ToLower(strings.TrimSpace(row.Provider))
		if row.IdentityID == "" || row.Provider == "" {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "ACCOUNT_IDENTITIES_UNAVAILABLE")
			return false
		}
		if last != nil {
			row.LastSignInAt = strings.TrimSpace(*last)
		}
		linked = append(linked, row)
		seen[row.Provider] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "ACCOUNT_IDENTITIES_UNAVAILABLE")
		return false
	}
	if len(linked) == 0 {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "ACCOUNT_IDENTITIES_UNAVAILABLE")
		return false
	}
	displayCodes := make([]string, 0, len(linked))
	for _, row := range linked {
		displayCodes = append(displayCodes, "AUTH_PROVIDER_DISPLAY_"+strings.ToUpper(row.Provider))
	}
	displayChrome := siteMsgMap(r.Context(), db, displayCodes)
	for i := range linked {
		linked[i].ProviderDisplay = siteMsg(displayChrome, "AUTH_PROVIDER_DISPLAY_"+strings.ToUpper(linked[i].Provider))
		if linked[i].ProviderDisplay == "" {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "ACCOUNT_IDENTITIES_UNAVAILABLE")
			return false
		}
	}

	// After unlink, profiles.auth_provider may still point at a removed identity.
	// Promote last-used (list is ordered by last_sign_in_at desc) and persist.
	primary := strings.ToLower(strings.TrimSpace(primaryProvider))
	if _, ok := seen[primary]; !ok {
		primary = linked[0].Provider
		_, err := h.DB.Exec(r.Context(), `
			update public.profiles
			set auth_provider = $1, updated_at = now()
			where id = $2::uuid
		`, primary, userID)
		if err != nil {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "ACCOUNT_OPERATION_FAILED")
			return false
		}
		if !attachAccountProvider(r.Context(), db, resp, primary) {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "ACCOUNT_AUTH_PROVIDER_MISSING")
			return false
		}
	}

	for i := range linked {
		linked[i].IsPrimary = linked[i].Provider == primary
	}
	linked[0].IsLastUsed = true

	allowed, err := authsettings.ListAllowedProviders(r.Context(), db)
	if err != nil || len(allowed) == 0 {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "AUTH_PROVIDERS_UNAVAILABLE")
		return false
	}
	linkable := make([]LinkableProvider, 0)
	linkCodes := make([]string, 0, len(allowed)*3)
	for _, p := range allowed {
		if _, ok := seen[p]; ok {
			continue
		}
		code := subscriptions.OAuthLinkActionCode(p)
		if code == "" {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "AUTH_PROVIDERS_UNAVAILABLE")
			return false
		}
		id := strings.ToLower(strings.TrimSpace(p))
		linkCodes = append(linkCodes, "ACTION:"+code, "PENDING:"+code, "AUTH_PROVIDER_DISPLAY_"+strings.ToUpper(id))
	}
	linkChrome := siteMsgMap(r.Context(), db, linkCodes)
	for _, p := range allowed {
		if _, ok := seen[p]; ok {
			continue
		}
		code := subscriptions.OAuthLinkActionCode(p)
		id := strings.ToLower(strings.TrimSpace(p))
		display := siteMsg(linkChrome, "AUTH_PROVIDER_DISPLAY_"+strings.ToUpper(id))
		action := siteAction(linkChrome, code)
		pending := sitePending(linkChrome, code)
		if code == "" || display == "" || action == "" || pending == "" {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "AUTH_PROVIDERS_UNAVAILABLE")
			return false
		}
		linkable = append(linkable, LinkableProvider{
			ID:           p,
			DisplayName:  display,
			ActionLabel:  action,
			PendingLabel: pending,
		})
	}
	if strings.TrimSpace(resp.LinkHint) == "" ||
		strings.TrimSpace(resp.UnlinkActionLabel) == "" ||
		strings.TrimSpace(resp.UnlinkPendingLabel) == "" ||
		strings.TrimSpace(resp.UnlinkConfirmMessage) == "" ||
		strings.TrimSpace(resp.UnlinkLastBlockedMessage) == "" ||
		strings.TrimSpace(resp.UnlinkFailedMessage) == "" ||
		strings.TrimSpace(resp.IdentityPrimaryLabel) == "" ||
		strings.TrimSpace(resp.IdentityLastUsedLabel) == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "ACCOUNT_IDENTITIES_UNAVAILABLE")
		return false
	}
	resp.LinkedIdentities = linked
	resp.LinkableProviders = linkable
	resp.CanUnlink = len(linked) > 1
	return true
}

type PreferencesPatchRequest struct {
	CompressionTier  *string `json:"compression_tier"`
	DeepEngine       *string `json:"deep_engine"`
	DeepTargetToken  *int    `json:"deep_target_token"`
	AutoStartWithIDE *bool   `json:"auto_start_with_ide"`
}

func filterNonEmptyStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

// attachLocalAgentStatus sets online/offline/unknown from api_key_devices last_seen.
// Window seconds and labels come from public.site_messages (fail-closed if missing/invalid).
func (h *Handler) attachLocalAgentStatus(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool, resp *PreferencesResponse, userID string) bool {
	m := subscriptions.LoadSiteMessages(r.Context(), db, []string{
		"LOCAL_AGENT_ONLINE_WITHIN_SEC",
		"PREFERENCES_LOCAL_AGENT_ONLINE",
		"PREFERENCES_LOCAL_AGENT_OFFLINE",
		"PREFERENCES_LOCAL_AGENT_UNKNOWN",
	})
	raw := strings.TrimSpace(subscriptions.SiteMsg(m, "LOCAL_AGENT_ONLINE_WITHIN_SEC"))
	if raw == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "LOCAL_AGENT_ONLINE_WITHIN_SEC_MISSING")
		return false
	}
	sec, err := strconv.Atoi(raw)
	if err != nil || sec < 1 {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "LOCAL_AGENT_ONLINE_WITHIN_SEC_INVALID")
		return false
	}
	onlineLabel := strings.TrimSpace(subscriptions.SiteMsg(m, "PREFERENCES_LOCAL_AGENT_ONLINE"))
	offlineLabel := strings.TrimSpace(subscriptions.SiteMsg(m, "PREFERENCES_LOCAL_AGENT_OFFLINE"))
	unknownLabel := strings.TrimSpace(subscriptions.SiteMsg(m, "PREFERENCES_LOCAL_AGENT_UNKNOWN"))
	if onlineLabel == "" || offlineLabel == "" || unknownLabel == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "PREFERENCES_LOCAL_AGENT_UNAVAILABLE")
		return false
	}

	var lastSeen *time.Time
	err = db.QueryRow(r.Context(), `
		select max(d.last_seen_at)
		from public.api_key_devices d
		inner join public.api_keys k on k.id = d.key_id
		where k.user_id = $1::uuid
		  and coalesce(k.revoked, false) = false
	`, userID).Scan(&lastSeen)
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "ACCOUNT_OPERATION_FAILED")
		return false
	}
	if lastSeen == nil {
		resp.LocalAgentStatus = "unknown"
		resp.LocalAgentStatusLabel = unknownLabel
		return true
	}
	if time.Since(*lastSeen) <= time.Duration(sec)*time.Second {
		resp.LocalAgentStatus = "online"
		resp.LocalAgentStatusLabel = onlineLabel
		return true
	}
	resp.LocalAgentStatus = "offline"
	resp.LocalAgentStatusLabel = offlineLabel
	return true
}

// GetPreferences returns Fast/Deep compression defaults for the user.
// GET /api/v1/me/preferences
func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}

	db := h.readPool()
	var tier, engine, authProvider string
	var target int
	var autoStart bool
	err := db.QueryRow(r.Context(), `
		select compression_tier, deep_engine, deep_target_token, auto_start_with_ide, lower(auth_provider)
		from public.profiles where id = $1::uuid
	`, userID).Scan(&tier, &engine, &target, &autoStart, &authProvider)
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "ACCOUNT_OPERATION_FAILED")
		return
	}

	var deepMin, deepMax, liveMin int
	var oomPolicy, deepV1, deepV2, deepLong string
	var warmup, skipStream bool
	var forceRaw []byte
	err = db.QueryRow(r.Context(), `
		select coalesce(deep_target_token_min, 0), coalesce(deep_target_token_max, 0),
		       coalesce(live_deep_min_input_tokens, -1), coalesce(live_deep_oom_policy, ''),
		       coalesce(live_deep_warmup_on_start, false), coalesce(live_deep_skip_on_stream, false),
		       coalesce(deep_v1_model, ''), coalesce(deep_v2_model, ''), coalesce(deep_long_model, ''),
		       coalesce(deep_v2_force_tokens, 'null'::jsonb)
		from public.billing_settings where id = 'default'
	`).Scan(&deepMin, &deepMax, &liveMin, &oomPolicy, &warmup, &skipStream, &deepV1, &deepV2, &deepLong, &forceRaw)
	if err != nil || deepMin < 1 || deepMax < deepMin || liveMin < 0 {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "BILLING_SETTINGS_UNAVAILABLE")
		return
	}
	oomPolicy = strings.ToLower(strings.TrimSpace(oomPolicy))
	if (oomPolicy != "fail" && oomPolicy != "skip") ||
		strings.TrimSpace(deepV1) == "" || strings.TrimSpace(deepV2) == "" || strings.TrimSpace(deepLong) == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "BILLING_SETTINGS_UNAVAILABLE")
		return
	}
	var forceTokens []string
	if err := json.Unmarshal(forceRaw, &forceTokens); err != nil || forceTokens == nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "BILLING_SETTINGS_UNAVAILABLE")
		return
	}

	resp, ok := preferencesChromeFromDB(r.Context(), db)
	if !ok {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "PREFERENCES_CHROME_UNAVAILABLE")
		return
	}
	resp.CompressionTier = tier
	resp.DeepEngine = engine
	resp.DeepTargetToken = target
	resp.AutoStartWithIDE = autoStart
	resp.DeepTargetTokenMin = deepMin
	resp.DeepTargetTokenMax = deepMax
	resp.LiveDeepMinInputTokens = liveMin
	resp.LiveDeepOOMPolicy = oomPolicy
	resp.LiveDeepWarmupOnStart = warmup
	resp.LiveDeepSkipOnStream = skipStream
	resp.DeepV1Model = deepV1
	resp.DeepV2Model = deepV2
	resp.DeepLongModel = deepLong
	resp.DeepV2ForceTokens = forceTokens
	adapters, aerr := loadProviderAdaptersFromDB(r.Context(), db)
	if aerr != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "PROVIDER_ADAPTERS_UNAVAILABLE")
		return
	}
	aliases, alerr := loadOpenAIModelAliasesFromDB(r.Context(), db)
	if alerr != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "PROVIDER_ADAPTERS_UNAVAILABLE")
		return
	}
	disco, derr := loadDiscoverableModelsFromDB(r.Context(), db)
	if derr != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "PROVIDER_ADAPTERS_UNAVAILABLE")
		return
	}
	resp.ProviderAdapters = adapters
	resp.OpenAIModelAliases = aliases
	resp.DiscoverableModels = disco
	resp.ProviderAdaptersSynced = true
	if !attachAccountProvider(r.Context(), db, &resp, authProvider) {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "ACCOUNT_AUTH_PROVIDER_MISSING")
		return
	}
	if !h.attachLinkedIdentities(w, r, db, &resp, userID, authProvider) {
		return
	}
	if !h.attachLocalAgentStatus(w, r, db, &resp, userID) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// PatchPreferences updates Fast/Deep defaults (source of truth in Postgres).
// PATCH /api/v1/me/preferences
func (h *Handler) PatchPreferences(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}

	var req PreferencesPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}

	tier := ""
	engine := ""
	target := 0
	autoStart := false
	err := h.DB.QueryRow(r.Context(), `
		select compression_tier, deep_engine, deep_target_token, auto_start_with_ide
		from public.profiles where id = $1::uuid
	`, userID).Scan(&tier, &engine, &target, &autoStart)
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "ACCOUNT_OPERATION_FAILED")
		return
	}

	if req.CompressionTier != nil {
		v := strings.ToLower(strings.TrimSpace(*req.CompressionTier))
		tierMsgs := subscriptions.LoadSiteMessages(r.Context(), h.DB, []string{
			"COMPRESSION_TIER_FAST", "COMPRESSION_TIER_DEEP",
		})
		fastID := strings.ToLower(strings.TrimSpace(subscriptions.SiteMsg(tierMsgs, "COMPRESSION_TIER_FAST")))
		deepID := strings.ToLower(strings.TrimSpace(subscriptions.SiteMsg(tierMsgs, "COMPRESSION_TIER_DEEP")))
		if fastID == "" || deepID == "" {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "COMPRESSION_TIER_FAST_MISSING")
			return
		}
		if v != fastID && v != deepID {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusBadRequest, "COMPRESSION_TIER_INVALID")
			return
		}
		tier = v
	}
	if req.DeepEngine != nil {
		v := strings.ToLower(strings.TrimSpace(*req.DeepEngine))
		if v != "v1" && v != "long" && v != "v2" {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusBadRequest, "DEEP_ENGINE_INVALID")
			return
		}
		engine = v
	}
	if req.DeepTargetToken != nil {
		var deepMin, deepMax int
		err := h.DB.QueryRow(r.Context(), `
			select coalesce(deep_target_token_min, 0), coalesce(deep_target_token_max, 0)
			from public.billing_settings where id = 'default'
		`).Scan(&deepMin, &deepMax)
		if err != nil || deepMin < 1 || deepMax < deepMin {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "BILLING_SETTINGS_UNAVAILABLE")
			return
		}
		if *req.DeepTargetToken < deepMin || *req.DeepTargetToken > deepMax {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusBadRequest, "DEEP_TARGET_TOKEN_RANGE")
			return
		}
		target = *req.DeepTargetToken
	}
	if req.AutoStartWithIDE != nil {
		autoStart = *req.AutoStartWithIDE
	}

	_, err = h.DB.Exec(r.Context(), `
		update public.profiles
		set compression_tier = $2,
		    deep_engine = $3,
		    deep_target_token = $4,
		    auto_start_with_ide = $5,
		    updated_at = now()
		where id = $1::uuid
	`, userID, tier, engine, target, autoStart)
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "ACCOUNT_OPERATION_FAILED")
		return
	}

	var authProvider string
	_ = h.DB.QueryRow(r.Context(), `
		select lower(auth_provider) from public.profiles where id = $1::uuid
	`, userID).Scan(&authProvider)

	w.Header().Set("Content-Type", "application/json")
	resp, ok := preferencesChromeFromDB(r.Context(), h.DB)
	if !ok {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "PREFERENCES_CHROME_UNAVAILABLE")
		return
	}
	resp.CompressionTier = tier
	resp.DeepEngine = engine
	resp.DeepTargetToken = target
	resp.AutoStartWithIDE = autoStart
	var deepMin, deepMax, liveMin int
	var oomPolicy, deepV1, deepV2, deepLong string
	var warmup, skipStream bool
	var forceRaw []byte
	if err := h.DB.QueryRow(r.Context(), `
		select coalesce(deep_target_token_min, 0), coalesce(deep_target_token_max, 0),
		       coalesce(live_deep_min_input_tokens, -1), coalesce(live_deep_oom_policy, ''),
		       coalesce(live_deep_warmup_on_start, false), coalesce(live_deep_skip_on_stream, false),
		       coalesce(deep_v1_model, ''), coalesce(deep_v2_model, ''), coalesce(deep_long_model, ''),
		       coalesce(deep_v2_force_tokens, 'null'::jsonb)
		from public.billing_settings where id = 'default'
	`).Scan(&deepMin, &deepMax, &liveMin, &oomPolicy, &warmup, &skipStream, &deepV1, &deepV2, &deepLong, &forceRaw); err == nil && deepMin >= 1 && deepMax >= deepMin && liveMin >= 0 {
		oomPolicy = strings.ToLower(strings.TrimSpace(oomPolicy))
		var forceTokens []string
		if json.Unmarshal(forceRaw, &forceTokens) == nil && forceTokens != nil &&
			(oomPolicy == "fail" || oomPolicy == "skip") &&
			strings.TrimSpace(deepV1) != "" && strings.TrimSpace(deepV2) != "" && strings.TrimSpace(deepLong) != "" {
			resp.DeepTargetTokenMin = deepMin
			resp.DeepTargetTokenMax = deepMax
			resp.LiveDeepMinInputTokens = liveMin
			resp.LiveDeepOOMPolicy = oomPolicy
			resp.LiveDeepWarmupOnStart = warmup
			resp.LiveDeepSkipOnStream = skipStream
			resp.DeepV1Model = deepV1
			resp.DeepV2Model = deepV2
			resp.DeepLongModel = deepLong
			resp.DeepV2ForceTokens = forceTokens
		}
	}
	if adapters, aerr := loadProviderAdaptersFromDB(r.Context(), h.DB); aerr == nil {
		if aliases, alerr := loadOpenAIModelAliasesFromDB(r.Context(), h.DB); alerr == nil {
			if disco, derr := loadDiscoverableModelsFromDB(r.Context(), h.DB); derr == nil {
				resp.ProviderAdapters = adapters
				resp.OpenAIModelAliases = aliases
				resp.DiscoverableModels = disco
				resp.ProviderAdaptersSynced = true
			}
		}
	}
	if !attachAccountProvider(r.Context(), h.DB, &resp, authProvider) {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "ACCOUNT_AUTH_PROVIDER_MISSING")
		return
	}
	if !h.attachLinkedIdentities(w, r, h.DB, &resp, userID, authProvider) {
		return
	}
	if !h.attachLocalAgentStatus(w, r, h.DB, &resp, userID) {
		return
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// DeleteAccount runs GDPR hard-delete for the authenticated user.
// DELETE /api/v1/me/account
func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}

	ctx := r.Context()
	_, err := h.DB.Exec(ctx, `select public.execute_gdpr_deletion($1::uuid)`, userID)
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, "ACCOUNT_OPERATION_FAILED")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	ch := siteMsgMap(r.Context(), h.readPool(), []string{"ACCOUNT_DELETE_SUCCESS"})
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "deleted",
		"code":    "ACCOUNT_DELETE_SUCCESS",
		"message": siteMsg(ch, "ACCOUNT_DELETE_SUCCESS"),
	})
}

func generateAPIKey() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "trm_" + hex.EncodeToString(buf), nil
}

func hashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
