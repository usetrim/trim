package subscriptions

// builtinLocalChromeMessage returns English seed copy for LOCAL_* site_messages.
// After SeedAndRefreshSiteMessages, DB rows win; this is seed/bootstrap only.
func builtinLocalChromeMessage(code string) string {
	switch code {
	case "LOCAL_DOCUMENT_TITLE":
		return "Trim local dashboard"
	case "LOCAL_BRAND":
		return "Trim"
	case "LOCAL_LEAD_FMT":
		return "Local proxy live meter. Side-by-side shows the last request before and after Trim. Active mode: %s."
	case "LOCAL_LABEL_REQUESTS":
		return "Requests"
	case "LOCAL_LABEL_TOKENS_IN":
		return "Tokens in"
	case "LOCAL_LABEL_TOKENS_OUT":
		return "Tokens out"
	case "LOCAL_LABEL_SAVED":
		return "Saved"
	case "LOCAL_LABEL_EST_USD":
		return "Est. USD saved"
	case "LOCAL_LABEL_LAST_LATENCY":
		return "Last latency"
	case "LOCAL_LABEL_LAST_REQUEST":
		return "Last request"
	case "LOCAL_LAST_REQUEST_VALUE_FMT":
		return "%d -> %d"
	case "LOCAL_LABEL_LAST_SAVED":
		return "Last saved"
	case "LOCAL_LABEL_FALLBACKS":
		return "Uncompressed fallbacks"
	case "LOCAL_STATS_LAST_DOOR":
		return "Last door"
	case "LOCAL_LABEL_DEEP_STATUS":
		return "Deep status"
	case "LOCAL_LABEL_DEEP_STAGE":
		return "Deep stage"
	case "LOCAL_LABEL_DEEP_STAGE_SAVED":
		return "Deep stage saved"
	case "LOCAL_DEEP_STATUS_APPLIED":
		return "Deep applied"
	case "LOCAL_DEEP_STATUS_SKIPPED_MIN":
		return "Deep skipped (below min input tokens)"
	case "LOCAL_DEEP_STATUS_SKIPPED_STREAM":
		return "Deep skipped (stream request)"
	case "LOCAL_DEEP_STATUS_SKIPPED_EMPTY":
		return "Deep skipped (nothing compressible / chrome frozen)"
	case "LOCAL_DEEP_STATUS_SKIPPED_OFF":
		return "Deep off for this request"
	case "LOCAL_DEEP_STATUS_PATH_SKIP":
		return "Deep skipped (this path is Fast-only)"
	case "LOCAL_DEEP_STATUS_FAIL_CLOSED_EXPAND":
		return "Deep kept Fast (expansion fail-closed)"
	case "LOCAL_DEEP_STATUS_FAIL_CLOSED_ERROR":
		return "Deep kept Fast (engine error)"
	case "LOCAL_DEEP_STATUS_OOM_SKIP":
		return "Deep skipped after OOM (kept Fast)"
	case "LOCAL_DEEP_STATUS_REJECTED":
		return "Deep result rejected (kept Fast)"
	case "LOCAL_DEEP_STATUS_FAST_ONLY":
		return "Fast only (Deep not enabled)"
	case "LOCAL_SAVINGS_DETAIL_TITLE":
		return "Savings detail"
	case "LOCAL_SAVINGS_DETAIL_LEAD":
		return "Whole-request Saved % can look low when IDE agent chrome is frozen. Deep stage shows savings on the compressible text Trim actually rewrote."
	case "LOCAL_SAVINGS_DETAIL_SHOW":
		return "Show savings detail"
	case "LOCAL_SAVINGS_DETAIL_HIDE":
		return "Hide savings detail"
	case "LOCAL_SAVINGS_DETAIL_WIRE_FMT":
		return "Whole request: {before} → {after} ({pct}% saved)"
	case "LOCAL_SAVINGS_DETAIL_STAGE_FMT":
		return "Deep stage: {before} → {after} ({pct}% saved)"
	case "LOCAL_SAVINGS_DETAIL_STATUS_FMT":
		return "Status: {status}"
	case "LOCAL_SAVINGS_DETAIL_CHROME_TIP":
		return "0% whole-request Saved on Claude Code / Cursor agent turns is often normal: system reminders, tools, and git status stay frozen on purpose so the agent loop stays safe."
	case "LOCAL_HEADING_BEFORE":
		return "Before (raw IDE payload)"
	case "LOCAL_HEADING_AFTER":
		return "After (Trim optimized)"
	case "LOCAL_PLAYGROUND_TITLE":
		return "Rule playground"
	case "LOCAL_PLAYGROUND_LEAD":
		return "Paste a prompt or markdown code fence, pick a compression mode, and preview what Trim would forward. Uses the same Fast Mode path as the live proxy."
	case "LOCAL_LABEL_MODE":
		return "Mode"
	case "LOCAL_MODE_MILD":
		return "mild (logs only)"
	case "LOCAL_MODE_BALANCED":
		return "balanced (default Fast)"
	case "LOCAL_MODE_AGGRESSIVE":
		return "aggressive (stronger Fast)"
	case "LOCAL_MODE_CUSTOM":
		return "custom (.trimrc rules)"
	case "LOCAL_LABEL_USER_PROMPT":
		return "User prompt hint"
	case "LOCAL_USER_PROMPT_PLACEHOLDER":
		return "e.g. refactor auth.go handleAuth"
	case "LOCAL_PREVIEW_ACTION":
		return "Preview trim"
	case "LOCAL_PREVIEW_PENDING":
		return "Previewing..."
	case "LOCAL_INPUT_PLACEHOLDER":
		return "Paste markdown with code fences, terminal logs, or a sample IDE prompt..."
	case "LOCAL_HEADING_PREVIEW_IN":
		return "Preview input"
	case "LOCAL_HEADING_PREVIEW_OUT":
		return "Preview output"
	case "LOCAL_PREVIEW_IDLE":
		return "(run preview)"
	case "LOCAL_EMPTY_PREVIEW":
		return "(empty)"
	case "LOCAL_NO_REQUEST_YET":
		return "(no request yet)"
	case "LOCAL_SERIES_TITLE":
		return "Daily series (SQLite)"
	case "LOCAL_SERIES_LEAD":
		return "Durable local history from ~/.trim/metrics.db (last {days} days)."
	case "LOCAL_SERIES_EMPTY":
		return "No local series yet. Run traffic through trim start."
	case "LOCAL_SERIES_UNAVAILABLE_FMT":
		return "Series unavailable: {error}"
	case "LOCAL_SERIES_HEADER":
		return "day | requests | before | after | saved$ | avg_ms"
	case "LOCAL_SERIES_NO_PROVIDER":
		return "No durable series provider (start via trim start to enable SQLite series)."
	case "LOCAL_PREVIEW_META_FMT":
		return "Mode {mode} · {before} → {after} tokens ({pct}% saved)"
	case "LOCAL_PREVIEW_FAILED_FMT":
		return "Preview failed: {error}"
	case "LOCAL_TUI_TITLE":
		return "Trim local stats"
	case "LOCAL_TUI_LABEL_REQUESTS":
		return "Requests"
	case "LOCAL_TUI_LABEL_TOKENS":
		return "Tokens"
	case "LOCAL_TUI_LABEL_REDUCTION":
		return "Reduction"
	case "LOCAL_TUI_LABEL_EST_SAVED":
		return "Est. saved"
	case "LOCAL_TUI_LABEL_LAST_LATENCY":
		return "Last latency"
	case "LOCAL_TUI_EST_SAVED_FMT":
		return "$%.4f"
	case "LOCAL_TUI_LATENCY_FMT":
		return "%.1f ms"
	case "LOCAL_TUI_SERIES_ROW_FMT":
		return "  %s  %d req  %d->%d  ~$%.4f\n"
	case "LOCAL_TUI_SERIES_EMPTY":
		return "Daily series ({days}d): none yet"
	case "LOCAL_TUI_SERIES_TITLE":
		return "Daily series ({days}d SQLite)"
	case "LOCAL_TUI_PROXY_URL_FMT":
		return "Proxy: http://localhost:{port}/v1/stats"
	case "LOCAL_TUI_OFFLINE_FMT":
		return "Proxy offline: {error}"
	case "LOCAL_TUI_UPDATED_PREFIX":
		return "Updated "
	case "LOCAL_TUI_FOOTER":
		return "q quit  |  r refresh  |  live proxy or ~/.trim/metrics.db"
	case "LOCAL_TUI_HINT":
		return "Cloud quota panel when signed in (trim login)  |  Daemon: trim daemon status"
	case "LOCAL_TUI_CLOUD_TITLE":
		return "Cloud quota"
	case "LOCAL_TUI_CLOUD_LABEL_TIER":
		return "Plan"
	case "LOCAL_TUI_CLOUD_LABEL_USED":
		return "Used"
	case "LOCAL_TUI_CLOUD_LABEL_REMAINING":
		return "Remaining"
	case "LOCAL_TUI_CLOUD_LABEL_TOPUP":
		return "Top-up"
	case "LOCAL_TUI_CLOUD_SIGNED_OUT":
		return "Not signed in. Run trim login to load cloud quota."
	case "LOCAL_TUI_CLOUD_OFFLINE_FMT":
		return "Cloud offline: {error}"
	case "LOCAL_FOOT_JSON":
		return "JSON"
	case "LOCAL_FOOT_SERIES":
		return "Series"
	case "LOCAL_FOOT_PREVIEW":
		return "Preview API"
	case "LOCAL_FOOT_METRICS":
		return "OpenMetrics"
	case "LOCAL_FOOT_HEALTH":
		return "Health"
	case "LOCAL_ERR_SERIES_UNAVAILABLE":
		return "series unavailable"
	case "LOCAL_ERR_BODY_READ":
		return "cannot read body"
	case "LOCAL_ERR_UPSTREAM_FMT":
		return "upstream error: %v"
	case "LOCAL_ERR_METHOD_POST_ONLY":
		return "POST only"
	case "LOCAL_ERR_INVALID_JSON":
		return "invalid JSON"
	case "LOCAL_HTML_LANG":
		return "en"
	case "RECEIPT_LOCALITY_JOIN_SEP":
		return ", "
	case "RECEIPT_SECTION_STATUS":
		return "Status"
	default:
		return ""
	}
}
