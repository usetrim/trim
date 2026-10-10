package account

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

// writeJSONErr loads the error body from public.site_messages (fail-closed empty).
func writeJSONErr(w http.ResponseWriter, ctx context.Context, db *pgxpool.Pool, status int, code string) {
	msg := ""
	if db != nil && strings.TrimSpace(code) != "" {
		m := subscriptions.LoadSiteMessages(ctx, db, []string{code})
		msg = strings.TrimSpace(subscriptions.SiteMsg(m, code))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func siteMsgMap(ctx context.Context, db *pgxpool.Pool, codes []string) map[string]string {
	return subscriptions.LoadSiteMessages(ctx, db, codes)
}

func siteMsg(m map[string]string, code string) string {
	return strings.TrimSpace(subscriptions.SiteMsg(m, code))
}

func siteAction(m map[string]string, code string) string {
	return siteMsg(m, "ACTION:"+code)
}

func sitePending(m map[string]string, code string) string {
	return siteMsg(m, "PENDING:"+code)
}

var apiKeyListChromeCodes = []string{
	"API_KEY_INTRO", "API_KEY_EMPTY", "API_KEY_STATUS_ACTIVE", "API_KEY_STATUS_REVOKED",
	"API_KEY_CREATED_PREFIX", "API_KEY_REVOKE_CONFIRM", "API_KEY_PREFIX_ELLIPSIS",
	"TABLE_SELECT_ALL", "TABLE_SELECT_ROW", "TABLE_SELECTED_FMT", "TABLE_ROW_ACTIONS",
	"TABLE_BULK_REVOKE", "TABLE_CLEAR_SELECTION",
	"API_KEY_DEVICE_HINT", "API_KEY_DEVICE_HW_LABEL", "API_KEY_DEVICE_AGENT_LABEL",
	"API_KEY_DEVICE_AGENT_CLI", "API_KEY_DEVICE_AGENT_IDE", "API_KEY_DEVICE_AGENT_CI",
	"API_KEY_DEVICE_BOUND_LABEL", "API_KEY_DEVICE_UNBOUND_LABEL", "API_KEY_DEVICE_COUNT_FMT",
	"CI_AGENT_HINT", "API_KEY_SEARCH", "API_KEY_SEARCH_DESC",
	"ACTION:API_KEY_ISSUE", "PENDING:API_KEY_ISSUE",
	"ACTION:API_KEY_REVOKE", "PENDING:API_KEY_REVOKE",
	"ACTION:API_KEY_LIST_REFRESH", "PENDING:API_KEY_LIST_REFRESH",
	"ACTION:API_KEY_DEVICE_REGISTER", "PENDING:API_KEY_DEVICE_REGISTER",
}

var apiKeyCreateChromeCodes = []string{
	"API_KEY_CREATED",
	"API_KEY_FRESH_HINT", "API_KEY_FRESH_DEVICE_REQUIRED", "API_KEY_DEVICE_HINT",
	"ACTION:API_KEY_ISSUE", "PENDING:API_KEY_ISSUE",
	"ACTION:API_KEY_ISSUE_RETRY", "ACTION:API_KEY_ISSUE_ANOTHER",
	"ACTION:API_KEY_COPY", "PENDING:API_KEY_COPY", "ACTION:API_KEY_COPIED",
}

var apiKeyDeviceChromeCodes = []string{
	"API_KEY_DEVICE_HINT", "API_KEY_DEVICE_HW_LABEL", "API_KEY_DEVICE_AGENT_LABEL",
	"API_KEY_DEVICE_AGENT_CLI", "API_KEY_DEVICE_AGENT_IDE", "API_KEY_DEVICE_AGENT_CI",
	"CI_AGENT_HINT",
	"ACTION:API_KEY_DEVICE_REGISTER", "PENDING:API_KEY_DEVICE_REGISTER",
	"ACTION:API_KEY_DEVICE_REMOVE", "PENDING:API_KEY_DEVICE_REMOVE",
}
