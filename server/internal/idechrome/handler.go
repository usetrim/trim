package idechrome

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

// Codes returned by GET /api/v1/public/ide-chrome for the VS Code / Cursor extension.
var chromeCodes = []string{
	"IDE_STATUS_TOOLTIP",
	"IDE_STATUS_IDLE",
	"IDE_STATUS_PENDING_FMT",
	"IDE_API_KEY_TITLE",
	"IDE_API_KEY_PROMPT",
	"IDE_API_KEY_SAVED",
	"IDE_API_KEY_CLEARED",
	"IDE_TELEMETRY_FLUSHED",
	"IDE_TELEMETRY_FAILED_FMT",
	"IDE_TELEMETRY_NETWORK_FMT",
	"IDE_EVENT_MODE",
	"IDE_EVENT_STATUS",
	"IDE_EVENT_MODEL",
	"IDE_CMD_COPY_HARDWARE_ID",
	"IDE_HARDWARE_ID_COPIED",
	"IDE_AUTOSTART_ENSURING",
	"IDE_AUTOSTART_PROXY_OK",
	"IDE_AUTOSTART_PROXY_STARTED",
	"IDE_AUTOSTART_PROXY_FAILED_FMT",
	"IDE_AUTOSTART_SKIPPED_OFF",
	"IDE_AUTOSTART_SKIPPED_UNSET",
	"IDE_AUTOSTART_CLI_MISSING",
	"IDE_PROXY_HEALTH_URL",
	"IDE_AUTOSTART_SETTLE_MS",
	"IDE_AUTOSTART_STOP_ON_QUIT",
	"IDE_PROXY_SHUTDOWN_URL",
	"IDE_AUTOSTART_STATUS_OK",
	"IDE_AUTOSTART_STATUS_FAILED",
	"IDE_AUTOSTART_WARN_FAILED",
	"IDE_AUTOSTART_SKIPPED_LOCAL_OFF",
	"IDE_AUTOSTART_SKIPPED_MANAGED",
	"IDE_AUTOSTART_SHUTDOWN_TIMEOUT_MS",
	"IDE_AUTOSTART_PREF_POLL_MS",
	"AUTOSTART_PREF_POLL_MIN_MS",
	"AUTOSTART_PREF_POLL_MAX_MS",
	"AUTOSTART_TIMEOUT_MIN_MS",
	"AUTOSTART_TIMEOUT_MAX_MS",
	"IDE_AUTOSTART_HEALTH_AFTER_START_FAILED",
	"IDE_HTTP_TIMEOUT_MIN_SEC",
	"IDE_HTTP_TIMEOUT_MAX_SEC",
	"IDE_AUTO_FLUSH_MIN_SEC",
	"IDE_AUTO_FLUSH_MAX_SEC",
	"IDE_AUTO_FLUSH_MIN_ENABLED_SEC",
	"IDE_HTTP_TIMEOUT_INVALID",
	"IDE_AUTO_FLUSH_INVALID",
	"IDE_CONFIG_TRACK_EDITS_MISSING",
	"IDE_CONFIG_MIN_LINES_INVALID",
	"IDE_CONFIG_API_URL_EMPTY",
	"IDE_CONFIG_API_URL_INVALID",
	"DEFAULT_AUTO_START_WITH_IDE",
	"IDE_QUOTA_EXHAUSTED_TITLE",
	"IDE_QUOTA_EXHAUSTED_BODY",
	"IDE_QUOTA_UPGRADE_ACTION",
}

// PublicChromeHandler serves backend-driven chrome for the Trim IDE extension.
// Fail-closed: only public.site_messages bodies (no MessageForCode invent).
func PublicChromeHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		siteDB := subscriptions.LoadSiteMessages(r.Context(), db, chromeCodes)
		out := make(map[string]string, len(chromeCodes))
		for _, code := range chromeCodes {
			if body := subscriptions.SiteMsg(siteDB, code); body != "" {
				out[code] = body
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": out,
		})
	}
}
