// Package localchrome holds UI chrome for the local proxy dashboard and TUI.
// Strings are loaded from site_messages (LOCAL_*) via the auth-providers API.
// Empty Default() is intentional: callers must not invent English labels.
package localchrome

import "strings"

// Chrome is layout-faithful label set for /dashboard and /v1/stats consumers.
type Chrome struct {
	DocumentTitle          string `json:"document_title"`
	Brand                  string `json:"brand"`
	LeadFmt                string `json:"lead_fmt"` // %s = active mode
	LabelRequests          string `json:"label_requests"`
	LabelTokensIn          string `json:"label_tokens_in"`
	LabelTokensOut         string `json:"label_tokens_out"`
	LabelSaved             string `json:"label_saved"`
	LabelEstUSD            string `json:"label_est_usd_saved"`
	LabelLastLatency       string `json:"label_last_latency"`
	LabelLastRequest       string `json:"label_last_request"`
	LastRequestValueFmt    string `json:"last_request_value_fmt"` // %d %d before→after
	LabelLastSaved         string `json:"label_last_saved"`
	LabelFallbacks         string `json:"label_fallbacks"`
	LabelLastDoor          string `json:"label_last_door"`
	LabelDeepStatus        string `json:"label_deep_status"`
	LabelDeepStage         string `json:"label_deep_stage"`
	LabelDeepStageSaved    string `json:"label_deep_stage_saved"`
	DeepStatusLabels       map[string]string `json:"deep_status_labels,omitempty"`
	SavingsDetailTitle     string `json:"savings_detail_title"`
	SavingsDetailLead      string `json:"savings_detail_lead"`
	SavingsDetailShow      string `json:"savings_detail_show"`
	SavingsDetailHide      string `json:"savings_detail_hide"`
	SavingsDetailWireFmt   string `json:"savings_detail_wire_fmt"`   // {before} {after} {pct}
	SavingsDetailStageFmt  string `json:"savings_detail_stage_fmt"`  // {before} {after} {pct}
	SavingsDetailStatusFmt string `json:"savings_detail_status_fmt"` // {status}
	SavingsDetailChromeTip string `json:"savings_detail_chrome_tip"`
	HeadingBefore          string `json:"heading_before"`
	HeadingAfter           string `json:"heading_after"`
	PlaygroundTitle        string `json:"playground_title"`
	PlaygroundLead         string `json:"playground_lead"`
	LabelMode              string `json:"label_mode"`
	ModeMild               string `json:"mode_mild"`
	ModeBalanced           string `json:"mode_balanced"`
	ModeAggressive         string `json:"mode_aggressive"`
	ModeCustom             string `json:"mode_custom"`
	LabelUserPrompt        string `json:"label_user_prompt"`
	UserPromptPlaceholder  string `json:"user_prompt_placeholder"`
	PreviewAction          string `json:"preview_action"`
	PreviewPending         string `json:"preview_pending"`
	InputPlaceholder       string `json:"input_placeholder"`
	HeadingPreviewIn       string `json:"heading_preview_in"`
	HeadingPreviewOut      string `json:"heading_preview_out"`
	PreviewIdle            string `json:"preview_idle"`
	EmptyPreview           string `json:"empty_preview"`
	NoRequestYet           string `json:"no_request_yet"`
	SeriesTitle            string `json:"series_title"`
	SeriesLead             string `json:"series_lead"` // may include {days}
	SeriesEmpty            string `json:"series_empty"`
	SeriesUnavailableFmt   string `json:"series_unavailable_fmt"` // {error}
	SeriesHeader           string `json:"series_header"`
	SeriesNoProvider       string `json:"series_no_provider"`
	PreviewMetaFmt         string `json:"preview_meta_fmt"`   // {mode} {before} {after} {pct}
	PreviewFailedFmt       string `json:"preview_failed_fmt"` // {error}
	TUITitle               string `json:"tui_title"`
	TUILabelRequests       string `json:"tui_label_requests"`
	TUILabelTokens         string `json:"tui_label_tokens"`
	TUILabelReduction      string `json:"tui_label_reduction"`
	TUILabelEstSaved       string `json:"tui_label_est_saved"`
	TUILabelLastLatency    string `json:"tui_label_last_latency"`
	TUILabelLastDoor       string `json:"tui_label_last_door"`
	TUIEstSavedFmt         string `json:"tui_est_saved_fmt"`  // fmt for SavedUSD float64
	TUILatencyFmt          string `json:"tui_latency_fmt"`    // fmt for last latency ms
	TUISeriesRowFmt        string `json:"tui_series_row_fmt"` // day, req, before, after, savedUSD
	TUISeriesEmpty         string `json:"tui_series_empty"`   // may include {days}
	TUISeriesTitle         string `json:"tui_series_title"`   // may include {days}
	TUIProxyURLFmt         string `json:"tui_proxy_url_fmt"`  // {port}
	TUIOfflineFmt          string `json:"tui_offline_fmt"`    // {error}
	TUIUpdatedPrefix       string `json:"tui_updated_prefix"`
	TUIFooter              string `json:"tui_footer"`
	TUIHint                string `json:"tui_hint"`
	TUICloudTitle          string `json:"tui_cloud_title"`
	TUICloudLabelTier      string `json:"tui_cloud_label_tier"`
	TUICloudLabelUsed      string `json:"tui_cloud_label_used"`
	TUICloudLabelRemaining string `json:"tui_cloud_label_remaining"`
	TUICloudLabelTopup     string `json:"tui_cloud_label_topup"`
	TUICloudSignedOut      string `json:"tui_cloud_signed_out"`
	TUICloudOfflineFmt     string `json:"tui_cloud_offline_fmt"` // {error}
	FootJSON               string `json:"foot_json"`
	FootSeries             string `json:"foot_series"`
	FootPreview            string `json:"foot_preview"`
	FootMetrics            string `json:"foot_metrics"`
	FootHealth             string `json:"foot_health"`
	ErrSeriesUnavailable   string `json:"err_series_unavailable"`
	ErrBodyRead            string `json:"err_body_read"`
	ErrUpstreamFmt         string `json:"err_upstream_fmt"` // %v = error
	ErrMethodPostOnly      string `json:"err_method_post_only"`
	ErrInvalidJSON         string `json:"err_invalid_json"`
	HtmlLang               string `json:"html_lang"`
	TUIHttpTimeoutUnset    string `json:"tui_http_timeout_unset"`
	PreviewTruncSuffix     string `json:"preview_trunc_suffix"`
	TUITruncSuffix         string `json:"tui_trunc_suffix"`
	StatsURLFmt            string `json:"stats_url_fmt"` // {port}
}

// Default returns empty chrome. Callers must load FromMessages / API; no invented English.
func Default() Chrome {
	return Chrome{}
}

// ReplaceDays substitutes {days} in series chrome strings.
func ReplaceDays(s, days string) string {
	if s == "" || days == "" {
		return s
	}
	return strings.ReplaceAll(s, "{days}", days)
}

// FromMessages builds Chrome from site_message codes (LOCAL_*).
func FromMessages(get func(code string) string) Chrome {
	if get == nil {
		return Chrome{}
	}
	return Chrome{
		DocumentTitle:          get("LOCAL_DOCUMENT_TITLE"),
		Brand:                  get("LOCAL_BRAND"),
		LeadFmt:                get("LOCAL_LEAD_FMT"),
		LabelRequests:          get("LOCAL_LABEL_REQUESTS"),
		LabelTokensIn:          get("LOCAL_LABEL_TOKENS_IN"),
		LabelTokensOut:         get("LOCAL_LABEL_TOKENS_OUT"),
		LabelSaved:             get("LOCAL_LABEL_SAVED"),
		LabelEstUSD:            get("LOCAL_LABEL_EST_USD"),
		LabelLastLatency:       get("LOCAL_LABEL_LAST_LATENCY"),
		LabelLastRequest:       get("LOCAL_LABEL_LAST_REQUEST"),
		LastRequestValueFmt:    get("LOCAL_LAST_REQUEST_VALUE_FMT"),
		LabelLastSaved:         get("LOCAL_LABEL_LAST_SAVED"),
		LabelFallbacks:         get("LOCAL_LABEL_FALLBACKS"),
		LabelLastDoor:          get("LOCAL_STATS_LAST_DOOR"),
		LabelDeepStatus:        get("LOCAL_LABEL_DEEP_STATUS"),
		LabelDeepStage:         get("LOCAL_LABEL_DEEP_STAGE"),
		LabelDeepStageSaved:    get("LOCAL_LABEL_DEEP_STAGE_SAVED"),
		DeepStatusLabels: map[string]string{
			"applied":             get("LOCAL_DEEP_STATUS_APPLIED"),
			"skipped_min":         get("LOCAL_DEEP_STATUS_SKIPPED_MIN"),
			"skipped_stream":      get("LOCAL_DEEP_STATUS_SKIPPED_STREAM"),
			"skipped_empty":       get("LOCAL_DEEP_STATUS_SKIPPED_EMPTY"),
			"skipped_off":         get("LOCAL_DEEP_STATUS_SKIPPED_OFF"),
			"path_skip":           get("LOCAL_DEEP_STATUS_PATH_SKIP"),
			"fail_closed_expand":  get("LOCAL_DEEP_STATUS_FAIL_CLOSED_EXPAND"),
			"fail_closed_error":   get("LOCAL_DEEP_STATUS_FAIL_CLOSED_ERROR"),
			"oom_skip":            get("LOCAL_DEEP_STATUS_OOM_SKIP"),
			"rejected":            get("LOCAL_DEEP_STATUS_REJECTED"),
			"fast_only":           get("LOCAL_DEEP_STATUS_FAST_ONLY"),
		},
		SavingsDetailTitle:     get("LOCAL_SAVINGS_DETAIL_TITLE"),
		SavingsDetailLead:      get("LOCAL_SAVINGS_DETAIL_LEAD"),
		SavingsDetailShow:      get("LOCAL_SAVINGS_DETAIL_SHOW"),
		SavingsDetailHide:      get("LOCAL_SAVINGS_DETAIL_HIDE"),
		SavingsDetailWireFmt:   get("LOCAL_SAVINGS_DETAIL_WIRE_FMT"),
		SavingsDetailStageFmt:  get("LOCAL_SAVINGS_DETAIL_STAGE_FMT"),
		SavingsDetailStatusFmt: get("LOCAL_SAVINGS_DETAIL_STATUS_FMT"),
		SavingsDetailChromeTip: get("LOCAL_SAVINGS_DETAIL_CHROME_TIP"),
		HeadingBefore:          get("LOCAL_HEADING_BEFORE"),
		HeadingAfter:           get("LOCAL_HEADING_AFTER"),
		PlaygroundTitle:        get("LOCAL_PLAYGROUND_TITLE"),
		PlaygroundLead:         get("LOCAL_PLAYGROUND_LEAD"),
		LabelMode:              get("LOCAL_LABEL_MODE"),
		ModeMild:               get("LOCAL_MODE_MILD"),
		ModeBalanced:           get("LOCAL_MODE_BALANCED"),
		ModeAggressive:         get("LOCAL_MODE_AGGRESSIVE"),
		ModeCustom:             get("LOCAL_MODE_CUSTOM"),
		LabelUserPrompt:        get("LOCAL_LABEL_USER_PROMPT"),
		UserPromptPlaceholder:  get("LOCAL_USER_PROMPT_PLACEHOLDER"),
		PreviewAction:          get("LOCAL_PREVIEW_ACTION"),
		PreviewPending:         get("LOCAL_PREVIEW_PENDING"),
		InputPlaceholder:       get("LOCAL_INPUT_PLACEHOLDER"),
		HeadingPreviewIn:       get("LOCAL_HEADING_PREVIEW_IN"),
		HeadingPreviewOut:      get("LOCAL_HEADING_PREVIEW_OUT"),
		PreviewIdle:            get("LOCAL_PREVIEW_IDLE"),
		EmptyPreview:           get("LOCAL_EMPTY_PREVIEW"),
		NoRequestYet:           get("LOCAL_NO_REQUEST_YET"),
		SeriesTitle:            get("LOCAL_SERIES_TITLE"),
		SeriesLead:             get("LOCAL_SERIES_LEAD"),
		SeriesEmpty:            get("LOCAL_SERIES_EMPTY"),
		SeriesUnavailableFmt:   get("LOCAL_SERIES_UNAVAILABLE_FMT"),
		SeriesHeader:           get("LOCAL_SERIES_HEADER"),
		SeriesNoProvider:       get("LOCAL_SERIES_NO_PROVIDER"),
		PreviewMetaFmt:         get("LOCAL_PREVIEW_META_FMT"),
		PreviewFailedFmt:       get("LOCAL_PREVIEW_FAILED_FMT"),
		TUITitle:               get("LOCAL_TUI_TITLE"),
		TUILabelRequests:       get("LOCAL_TUI_LABEL_REQUESTS"),
		TUILabelTokens:         get("LOCAL_TUI_LABEL_TOKENS"),
		TUILabelReduction:      get("LOCAL_TUI_LABEL_REDUCTION"),
		TUILabelEstSaved:       get("LOCAL_TUI_LABEL_EST_SAVED"),
		TUILabelLastLatency:    get("LOCAL_TUI_LABEL_LAST_LATENCY"),
		TUILabelLastDoor:       get("LOCAL_TUI_LABEL_LAST_DOOR"),
		TUIEstSavedFmt:         get("LOCAL_TUI_EST_SAVED_FMT"),
		TUILatencyFmt:          get("LOCAL_TUI_LATENCY_FMT"),
		TUISeriesRowFmt:        get("LOCAL_TUI_SERIES_ROW_FMT"),
		TUISeriesEmpty:         get("LOCAL_TUI_SERIES_EMPTY"),
		TUISeriesTitle:         get("LOCAL_TUI_SERIES_TITLE"),
		TUIProxyURLFmt:         get("LOCAL_TUI_PROXY_URL_FMT"),
		TUIOfflineFmt:          get("LOCAL_TUI_OFFLINE_FMT"),
		TUIUpdatedPrefix:       get("LOCAL_TUI_UPDATED_PREFIX"),
		TUIFooter:              get("LOCAL_TUI_FOOTER"),
		TUIHint:                get("LOCAL_TUI_HINT"),
		TUICloudTitle:          get("LOCAL_TUI_CLOUD_TITLE"),
		TUICloudLabelTier:      get("LOCAL_TUI_CLOUD_LABEL_TIER"),
		TUICloudLabelUsed:      get("LOCAL_TUI_CLOUD_LABEL_USED"),
		TUICloudLabelRemaining: get("LOCAL_TUI_CLOUD_LABEL_REMAINING"),
		TUICloudLabelTopup:     get("LOCAL_TUI_CLOUD_LABEL_TOPUP"),
		TUICloudSignedOut:      get("LOCAL_TUI_CLOUD_SIGNED_OUT"),
		TUICloudOfflineFmt:     get("LOCAL_TUI_CLOUD_OFFLINE_FMT"),
		FootJSON:               get("LOCAL_FOOT_JSON"),
		FootSeries:             get("LOCAL_FOOT_SERIES"),
		FootPreview:            get("LOCAL_FOOT_PREVIEW"),
		FootMetrics:            get("LOCAL_FOOT_METRICS"),
		FootHealth:             get("LOCAL_FOOT_HEALTH"),
		ErrSeriesUnavailable:   get("LOCAL_ERR_SERIES_UNAVAILABLE"),
		ErrBodyRead:            get("LOCAL_ERR_BODY_READ"),
		ErrUpstreamFmt:         get("LOCAL_ERR_UPSTREAM_FMT"),
		ErrMethodPostOnly:      get("LOCAL_ERR_METHOD_POST_ONLY"),
		ErrInvalidJSON:         get("LOCAL_ERR_INVALID_JSON"),
		HtmlLang:               get("LOCAL_HTML_LANG"),
		TUIHttpTimeoutUnset:    get("LOCAL_TUI_HTTP_TIMEOUT_UNSET"),
		PreviewTruncSuffix:     get("LOCAL_PREVIEW_TRUNC_SUFFIX"),
		TUITruncSuffix:         get("LOCAL_TUI_TRUNC_SUFFIX"),
		StatsURLFmt:            get("LOCAL_STATS_URL_FMT"),
	}
}
