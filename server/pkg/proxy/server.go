package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/usetrim/trim/server/pkg/cache"
	"github.com/usetrim/trim/server/pkg/localchrome"
	"github.com/usetrim/trim/server/pkg/provideradapt"
	"github.com/usetrim/trim/server/pkg/trimmer"
)

// Options are upstream targets for the local or cloud proxy. No silent defaults.
type Options struct {
	UpstreamOpenAI    string
	UpstreamAnthropic string
	// UpstreamOpenAIFailover is a same-shape OpenAI-compatible backup (Azure OpenAI, vLLM, etc.).
	// Used when the primary OpenAI upstream returns a transport error or HTTP 5xx.
	UpstreamOpenAIFailover string
	// UpstreamAnthropicFailover is a same-shape Anthropic-compatible backup URL.
	UpstreamAnthropicFailover string
	NeverTrim                 func(path string) bool
	MaxLogBytes               int
	LogCompactMinBytes        int
	LogCompactMaxLines        int
	// LogNoiseSubstrings from .trimrc; empty = no invent noise filter.
	LogNoiseSubstrings   []string
	ActiveFileProtection bool
	// CompressionMode: mild | balanced | aggressive | custom (from .trimrc).
	CompressionMode string
	// MinLines small-file exemption; 0 uses mode default.
	MinLines int
	// BalancedMinLines / AggressiveMinLines optional DB overrides when MinLines is 0.
	BalancedMinLines   int
	AggressiveMinLines int
	MildMinLines       int
	// DisableSkeletonize when mild or custom_logs_only.
	DisableSkeletonize bool
	// AlwaysCompactLogs for mild / aggressive / custom logs-only.
	AlwaysCompactLogs bool
	// CustomQueries from .trimrc for playground display (optional).
	CustomQueries []string
	// FallbackUncompressed retries once with the raw body when a trimmed upstream call fails.
	FallbackUncompressed bool
	// CheapModel + RouteMaxTokens: when prompt token estimate is at or below RouteMaxTokens,
	// rewrite the request model to CheapModel (backend-driven routing, empty disables).
	CheapModel     string
	RouteMaxTokens int
	// SavingsUsdPerMTok estimates USD saved from trimmed tokens. Must be set from env/config.
	SavingsUsdPerMTok float64
	// OnEvent is called after each trim forward. Receives the original client request
	// so cloud middleware can read user id / request id from context.
	// errorCode is optional machine code (e.g. oom); empty on success or unclassified errors.
	OnEvent func(r *http.Request, model string, before, after int, latencyMs float64, status, mode, errorCode string)
	// OnMetrics is optional Prometheus-style counters.
	OnMetrics func(before, after int, isError bool)
	// SeriesProvider returns durable daily aggregates (e.g. local SQLite) for /v1/stats/series.
	// days is requested window; provider chooses clamp. Nil disables the series endpoint body.
	SeriesProvider func(days int) (any, error)
	// DefaultSeriesDays is used when /v1/stats/series has no days query (required; no invent 14).
	DefaultSeriesDays int
	// PreviewMaxChars truncates last before/after dashboard previews (required; no invent 1200).
	PreviewMaxChars int
	// HistoryKeepTurns keeps the last N non-system chat turns verbatim (after text trim).
	// Older turns are replaced with a short stub. 0 disables history sliding-window (no invent).
	HistoryKeepTurns int
	// UpstreamHTTPTimeout is the outbound LLM HTTP client timeout. Required (no invent 120s).
	UpstreamHTTPTimeout time.Duration
	// Chrome is dashboard/TUI label set from site_messages (LOCAL_*). Empty is fail-closed (no invent).
	Chrome localchrome.Chrome
	// OnLocalShutdown is invoked by POST /v1/control/shutdown (loopback only).
	// start.go wires this to http.Server.Shutdown. Nil = shutdown endpoint returns 503.
	OnLocalShutdown func()
	// DeepOptimize runs after Fast Mode when the local CLI wires Deep prefs (compression_tier=deep).
	// Nil on cloud proxy (Deep never loads in Trim cloud). On hard error the request fails closed (502).
	// Soft skips / expansion fail-closed return Fast body with Status set for the local meter (no invent).
	DeepOptimize func(optimized []byte, path string) (DeepResult, error)
	// LiveModeLabel overrides dashboard/telemetry mode when set (e.g. deep/v2 from site_messages). Empty = CompressionMode only.
	LiveModeLabel string
	// ErrDeepFmt formats Deep failures (%v = error). From site_messages; empty = generic body.
	ErrDeepFmt string
	// ProviderAdapters is DB-synced OpenAI-compat → dialect registry (nil = passthrough only; no invent).
	ProviderAdapters *provideradapt.Registry
	// AnthropicWorkspaceID is optional local fallback (TRIM_ANTHROPIC_WORKSPACE_ID) for org-scoped
	// Anthropic API keys when the client omits anthropic-workspace-id. Workspace-scoped keys leave it empty.
	AnthropicWorkspaceID string
	// OpenAIOrganization / OpenAIProject are optional local fallbacks (TRIM_OPENAI_ORGANIZATION /
	// TRIM_OPENAI_PROJECT) when the client omits OpenAI-Organization / OpenAI-Project headers.
	OpenAIOrganization string
	OpenAIProject      string
	// DoorOpenAI / DoorAnthropic are site_messages labels for native doors (fail-closed empty → omit header).
	DoorOpenAI     string
	DoorAnthropic  string
	// Adapter chrome formats from site_messages (empty = generic errors).
	AdapterTranslateFmt string
	AdapterAuthFmt      string
	AdapterResponseFmt  string
	// Models / gateway chrome from site_messages (empty = generic errors).
	ModelsNotSynced  string
	ModelsMethod     string
	BaseURLDoubleV1  string
	ModelNotFoundFmt string
	// UnknownModelFmt is used when adapters are synced but no prefix matches (%q = model).
	// Empty = generic error. Prevents silent cross-host invent (e.g. unknown id → Gemini).
	UnknownModelFmt string
	// UpstreamAuthFmt remaps upstream auth failures (%q = model). Empty = pass upstream body.
	UpstreamAuthFmt string
	// UpstreamQuotaFmt remaps upstream rate-limit / quota failures (%q = model). Empty = pass upstream body.
	UpstreamQuotaFmt string
	// UpstreamUnavailableFmt remaps upstream HTML/5xx outages (%q = model). Empty = pass upstream body.
	UpstreamUnavailableFmt string
	// ModelsUpstream* chrome for GET /v1/models?adapter=… proxy-through.
	ModelsUpstreamAdapterRequired string
	ModelsUpstreamNotFoundFmt     string
	ModelsUpstreamAuthRequired    string
	ModelsUpstreamUnsupportedFmt  string
}

type Stats struct {
	Requests          int64   `json:"requests"`
	TokensBefore      int64   `json:"tokens_before"`
	TokensAfter       int64   `json:"tokens_after"`
	SavedUSD          float64 `json:"saved_usd_est"`
	LastLatency       float64 `json:"last_latency_ms"`
	Fallbacks         int64   `json:"uncompressed_fallbacks"`
	UpstreamFailovers int64   `json:"upstream_failovers"`
	LastBeforeTokens  int     `json:"last_before_tokens"`
	LastAfterTokens   int     `json:"last_after_tokens"`
	LastBeforePreview string  `json:"last_before_preview,omitempty"`
	LastAfterPreview  string  `json:"last_after_preview,omitempty"`
	LastDoor          string  `json:"last_door,omitempty"`
	AdapterRequests   int64   `json:"adapter_requests"`
	// LastDeep* explains live Deep for worldwide users when whole-body Saved % looks like 0.
	LastDeepStatus           string  `json:"last_deep_status,omitempty"`
	LastDeepStageBefore      int     `json:"last_deep_stage_before_tokens"`
	LastDeepStageAfter       int     `json:"last_deep_stage_after_tokens"`
	LastDeepStageSavedPct    float64 `json:"last_deep_stage_saved_percent"`
}

type Server struct {
	opts  Options
	mu    sync.RWMutex
	stats Stats
	lru   *cache.LRU
	files *trimmer.FileHistory
	http  *http.Client
}

func NewServer(opts Options) *Server {
	return &Server{
		opts:  opts,
		lru:   cache.NewLRU(1024),
		files: trimmer.NewFileHistory(512),
		// Non-stream upstream calls use a hard deadline.
		// Streaming SSE/chat uses a separate client (Timeout 0) with ResponseHeaderTimeout only.
		http: &http.Client{Timeout: opts.UpstreamHTTPTimeout},
	}
}

// streamHTTPClient returns an outbound client safe for text/event-stream.
// Overall Timeout must stay 0 or long agent streams are killed mid-flight.
// ResponseHeaderTimeout uses UpstreamHTTPTimeout when set (no invent).
func (s *Server) streamHTTPClient() *http.Client {
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if s.opts.UpstreamHTTPTimeout > 0 {
		tr.ResponseHeaderTimeout = s.opts.UpstreamHTTPTimeout
	}
	return &http.Client{
		Timeout:   0,
		Transport: tr,
	}
}

func (s *Server) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/v1/control/shutdown", s.handleLocalShutdown)
	mux.HandleFunc("/v1/stats", s.handleStats)
	mux.HandleFunc("/v1/stats/series", s.handleStatsSeries)
	mux.HandleFunc("/v1/trim/preview", s.handleTrimPreview)
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.Handle("/brand/", brandMarkHandler())
	mux.HandleFunc("/dashboard", s.handleDashboard)
	mux.HandleFunc("/v1/chat/completions", s.handleOpenAIChat)
	mux.HandleFunc("/v1/messages", s.handleAnthropicMessages)
	mux.HandleFunc("/v1/messages/count_tokens", s.handleAnthropicCountTokens)
	mux.HandleFunc("/v1/models", s.handleModels)
	mux.HandleFunc("/", s.handlePassthrough)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","proxy":"trim"}`))
}

// handleLocalShutdown is loopback-only graceful stop for IDE quit / trim stop.
func (s *Server) handleLocalShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	host := r.RemoteAddr
	if fwd := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); fwd != "" {
		// Never trust forwarded client for shutdown - reject if any proxy hop claimed.
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"shutdown_forbidden"}`))
		return
	}
	ip := host
	if i := strings.LastIndex(host, ":"); i > 0 {
		ip = host[:i]
	}
	ip = strings.Trim(ip, "[]")
	if ip != "127.0.0.1" && ip != "::1" && ip != "localhost" {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"shutdown_loopback_only"}`))
		return
	}
	if s.opts.OnLocalShutdown == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"shutdown_unavailable"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"stopping","proxy":"trim"}`))
	go s.opts.OnLocalShutdown()
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	st := s.Stats()
	_ = json.NewEncoder(w).Encode(map[string]any{
		"requests":                       st.Requests,
		"tokens_before":                  st.TokensBefore,
		"tokens_after":                   st.TokensAfter,
		"saved_usd_est":                  st.SavedUSD,
		"last_latency_ms":                st.LastLatency,
		"uncompressed_fallbacks":         st.Fallbacks,
		"upstream_failovers":             st.UpstreamFailovers,
		"last_before_tokens":             st.LastBeforeTokens,
		"last_after_tokens":              st.LastAfterTokens,
		"last_before_preview":            st.LastBeforePreview,
		"last_after_preview":             st.LastAfterPreview,
		"last_door":                      st.LastDoor,
		"adapter_requests":               st.AdapterRequests,
		"last_deep_status":               st.LastDeepStatus,
		"last_deep_stage_before_tokens":  st.LastDeepStageBefore,
		"last_deep_stage_after_tokens":   st.LastDeepStageAfter,
		"last_deep_stage_saved_percent":  st.LastDeepStageSavedPct,
		"chrome":                         s.opts.Chrome,
	})
}

func (s *Server) handleStatsSeries(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	chrome := s.opts.Chrome
	if s.opts.SeriesProvider == nil {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items":   []any{},
			"message": chrome.SeriesNoProvider,
			"chrome":  chrome,
		})
		return
	}
	days := s.opts.DefaultSeriesDays
	if days < 1 {
		msg := chrome.ErrSeriesUnavailable
		if msg == "" {
			msg = "series days not configured"
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
		return
	}
	if q := strings.TrimSpace(r.URL.Query().Get("days")); q != "" {
		n := 0
		for _, ch := range q {
			if ch < '0' || ch > '9' {
				n = 0
				break
			}
			n = n*10 + int(ch-'0')
		}
		if n > 0 {
			days = n
		}
		if days > 90 {
			days = 90
		}
	}
	items, err := s.opts.SeriesProvider(days)
	if err != nil {
		msg := chrome.ErrSeriesUnavailable
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
		return
	}
	msg := ""
	switch v := items.(type) {
	case nil:
		msg = chrome.SeriesEmpty
	case []any:
		if len(v) == 0 {
			msg = chrome.SeriesEmpty
		}
	default:
		// Slice of structs from metrics.Series: empty check via JSON round-trip size is overkill;
		// callers pass []metrics.DayPoint; detect via reflection-free fmt empty array encoding.
		b, _ := json.Marshal(items)
		if string(b) == "null" || string(b) == "[]" {
			msg = chrome.SeriesEmpty
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"days":    days,
		"items":   items,
		"message": msg,
		"chrome":  chrome,
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	st := s.Stats()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = fmt.Fprintf(w, "# HELP trim_local_proxy_requests_total Local proxy request count\n")
	_, _ = fmt.Fprintf(w, "# TYPE trim_local_proxy_requests_total counter\n")
	_, _ = fmt.Fprintf(w, "trim_local_proxy_requests_total %d\n", st.Requests)
	_, _ = fmt.Fprintf(w, "# HELP trim_local_proxy_tokens_before_total Tokens before trim\n")
	_, _ = fmt.Fprintf(w, "# TYPE trim_local_proxy_tokens_before_total counter\n")
	_, _ = fmt.Fprintf(w, "trim_local_proxy_tokens_before_total %d\n", st.TokensBefore)
	_, _ = fmt.Fprintf(w, "# HELP trim_local_proxy_tokens_after_total Tokens after trim\n")
	_, _ = fmt.Fprintf(w, "# TYPE trim_local_proxy_tokens_after_total counter\n")
	_, _ = fmt.Fprintf(w, "trim_local_proxy_tokens_after_total %d\n", st.TokensAfter)
	_, _ = fmt.Fprintf(w, "# HELP trim_local_proxy_fallbacks_total Uncompressed fallback retries\n")
	_, _ = fmt.Fprintf(w, "# TYPE trim_local_proxy_fallbacks_total counter\n")
	_, _ = fmt.Fprintf(w, "trim_local_proxy_fallbacks_total %d\n", st.Fallbacks)
}

func (s *Server) handleOpenAIChat(w http.ResponseWriter, r *http.Request) {
	s.interceptAndForward(w, r, s.opts.UpstreamOpenAI, "/v1/chat/completions", strings.TrimSpace(s.opts.DoorOpenAI))
}

func (s *Server) handleAnthropicMessages(w http.ResponseWriter, r *http.Request) {
	s.interceptAndForward(w, r, s.opts.UpstreamAnthropic, "/v1/messages", strings.TrimSpace(s.opts.DoorAnthropic))
}

// handleAnthropicCountTokens forwards Anthropic's official count_tokens path (Door A companion).
func (s *Server) handleAnthropicCountTokens(w http.ResponseWriter, r *http.Request) {
	s.interceptAndForward(w, r, s.opts.UpstreamAnthropic, "/v1/messages/count_tokens", strings.TrimSpace(s.opts.DoorAnthropic))
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	chrome := s.opts.Chrome
	if r.Method != http.MethodGet {
		msg := strings.TrimSpace(s.opts.ModelsMethod)
		if msg == "" {
			msg = chrome.ErrMethodPostOnly
		}
		if msg == "" {
			msg = "GET only"
		}
		http.Error(w, msg, http.StatusMethodNotAllowed)
		return
	}
	if s.opts.ProviderAdapters == nil {
		msg := strings.TrimSpace(s.opts.ModelsNotSynced)
		if msg == "" {
			http.Error(w, "models not synced", http.StatusServiceUnavailable)
		} else {
			http.Error(w, msg, http.StatusServiceUnavailable)
		}
		return
	}
	// Optional live discovery: GET /v1/models?adapter=<provider_adapters.id>
	// Proxies the user's key to that host's /models (OpenAI-compat only). Default remains DB catalog.
	if adapterID := strings.TrimSpace(r.URL.Query().Get("adapter")); adapterID != "" {
		s.proxyUpstreamModels(w, r, adapterID)
		return
	}
	limit, err := provideradapt.ParseModelsLimit(r.URL.Query().Get("limit"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.opts.ProviderAdapters.ListModels(limit))
}

// proxyUpstreamModels forwards GET /models to an openai_compat adapter host with the client's credential.
func (s *Server) proxyUpstreamModels(w http.ResponseWriter, r *http.Request, adapterID string) {
	cfg, ok := s.opts.ProviderAdapters.ByID(adapterID)
	if !ok {
		msg := strings.TrimSpace(s.opts.ModelsUpstreamNotFoundFmt)
		if msg == "" {
			http.Error(w, fmt.Sprintf("adapter %q not found or disabled", adapterID), http.StatusBadRequest)
		} else {
			http.Error(w, fmt.Sprintf(msg, adapterID), http.StatusBadRequest)
		}
		return
	}
	if strings.TrimSpace(cfg.Dialect) != provideradapt.DialectOpenAICompat {
		msg := strings.TrimSpace(s.opts.ModelsUpstreamUnsupportedFmt)
		if msg == "" {
			http.Error(w, fmt.Sprintf("adapter %q does not support upstream /models proxy-through", adapterID), http.StatusBadRequest)
		} else {
			http.Error(w, fmt.Sprintf(msg, adapterID), http.StatusBadRequest)
		}
		return
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		msg := strings.TrimSpace(s.opts.ModelsUpstreamAuthRequired)
		if msg == "" {
			msg = "Authorization Bearer required for upstream model discovery"
		}
		http.Error(w, msg, http.StatusUnauthorized)
		return
	}
	base, err := provideradapt.ResolveUpstreamBase(cfg, s.opts.UpstreamOpenAI, s.opts.UpstreamAnthropic)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	modelsURL, err := provideradapt.OpenAICompatModelsURL(base, cfg.UpstreamPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, modelsURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	req.Header.Set("Authorization", auth)
	if v := strings.TrimSpace(r.Header.Get(provideradapt.HeaderOpenAIOrganization)); v != "" {
		req.Header.Set(provideradapt.HeaderOpenAIOrganization, v)
	}
	if v := strings.TrimSpace(r.Header.Get(provideradapt.HeaderOpenAIProject)); v != "" {
		req.Header.Set(provideradapt.HeaderOpenAIProject, v)
	}
	provideradapt.ApplyOpenAIOrgProject(req.Header, s.opts.OpenAIOrganization, s.opts.OpenAIProject)

	client := &http.Client{Timeout: s.opts.UpstreamHTTPTimeout}
	if client.Timeout <= 0 {
		client.Timeout = 30 * time.Second
	}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	if door := strings.TrimSpace(cfg.DoorLabel); door != "" {
		w.Header().Set("X-Trim-Door", door)
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) handlePassthrough(w http.ResponseWriter, r *http.Request) {
	if isDoubleV1Path(r.URL.Path) {
		msg := strings.TrimSpace(s.opts.BaseURLDoubleV1)
		if msg == "" {
			http.Error(w, "invalid /v1/v1 path", http.StatusBadRequest)
		} else {
			http.Error(w, msg, http.StatusBadRequest)
		}
		return
	}
	target := s.opts.UpstreamOpenAI
	door := strings.TrimSpace(s.opts.DoorOpenAI)
	if strings.Contains(r.URL.Path, "anthropic") || r.Header.Get("anthropic-version") != "" {
		target = s.opts.UpstreamAnthropic
		door = strings.TrimSpace(s.opts.DoorAnthropic)
	}
	s.interceptAndForward(w, r, target, r.URL.Path, door)
}

func isDoubleV1Path(path string) bool {
	p := strings.ToLower(path)
	return strings.HasPrefix(p, "/v1/v1/") || p == "/v1/v1"
}

// remapUpstreamErrorChrome replaces known upstream failure bodies with Trim site_messages chrome.
// Fail-closed: empty fmt leaves the upstream body untouched. Priority: model_not_found → auth → quota.
func remapUpstreamErrorChrome(body []byte, status int, model string, opts Options) []byte {
	var fmtMsg string
	var errType string
	switch {
	case provideradapt.LooksLikeModelNotFound(body):
		fmtMsg = strings.TrimSpace(opts.ModelNotFoundFmt)
		errType = "invalid_request_error"
	case provideradapt.LooksLikeAuthError(body, status):
		fmtMsg = strings.TrimSpace(opts.UpstreamAuthFmt)
		errType = "authentication_error"
	case provideradapt.LooksLikeQuotaError(body, status):
		fmtMsg = strings.TrimSpace(opts.UpstreamQuotaFmt)
		errType = "rate_limit_error"
	case provideradapt.LooksLikeUpstreamUnavailable(body, status):
		fmtMsg = strings.TrimSpace(opts.UpstreamUnavailableFmt)
		errType = "api_error"
	default:
		return nil
	}
	if fmtMsg == "" {
		return nil
	}
	msg := fmt.Sprintf(fmtMsg, model)
	b, err := json.Marshal(map[string]any{
		"error": map[string]string{
			"type":    errType,
			"message": msg,
		},
	})
	if err != nil || len(b) == 0 {
		return nil
	}
	return b
}

func (s *Server) interceptAndForward(w http.ResponseWriter, r *http.Request, upstreamBase, path, doorLabel string) {
	start := time.Now()
	chrome := s.opts.Chrome
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, chrome.ErrBodyRead, http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()

	// Strip UTF-8 BOM (Windows editors / some clients). encoding/json rejects BOM and
	// breaks openai_compat adapters (GPT/Gemini/DeepSeek/Mistral) with "invalid JSON body".
	body = bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF})

	// Cursor Agent BYOK / CLI: Responses→Chat + content-part + tool-name enrich for
	// openai_compat doors (GPT/Gemini/DeepSeek/Mistral) ONLY.
	// Native Anthropic /v1/messages (and count_tokens) must NOT run Chat tool wrapping -
	// that smashes official {name,input_schema} / dated server tools into function shape
	// (same smash/garble class as whole-body Deep: agent chrome destroyed before upstream).
	if !strings.Contains(path, "/messages") {
		body = NormalizeOpenAICompatRequestBody(body)
	}
	// Unique Cursor custom ids map to real upstream model names (DB openai_model_aliases).
	body = s.rewriteUpstreamModelAlias(body, upstreamBase)

	optimized, before, after := s.optimizePayload(body)
	optimized = s.maybeRouteModel(optimized, before)

	eventMode := strings.TrimSpace(s.opts.LiveModeLabel)
	if eventMode == "" {
		eventMode = strings.TrimSpace(s.opts.CompressionMode)
	}

	deepStatus := DeepStatusFastOnly
	deepStageBefore, deepStageAfter := 0, 0
	if s.opts.DeepOptimize != nil && !skipLiveDeepForRequestPath(path) {
		deepRes, deepErr := s.opts.DeepOptimize(optimized, path)
		if deepErr != nil {
			msg := s.opts.ErrDeepFmt
			if msg == "" {
				http.Error(w, deepErr.Error(), http.StatusBadGateway)
			} else {
				http.Error(w, fmt.Sprintf(msg, deepErr), http.StatusBadGateway)
			}
			return
		}
		deepStatus = strings.TrimSpace(deepRes.Status)
		deepStageBefore, deepStageAfter = deepRes.Origin, deepRes.Compressed
		deepOut := deepRes.Body
		if len(deepOut) == 0 {
			deepOut = optimized
		}
		if shouldAcceptDeepResult(optimized, deepOut, deepRes.Origin, deepRes.Compressed) {
			fastWire := estimate(optimized)
			deepWire := estimate(deepOut)
			optimized = deepOut
			if deepRes.Origin > 0 && deepRes.Compressed >= 0 && deepRes.Compressed < deepRes.Origin {
				saved := deepRes.Origin - deepRes.Compressed
				if after > saved {
					after = after - saved
				} else if deepRes.Compressed < after {
					after = deepRes.Compressed
				}
			}
			// Huge frozen Anthropic system chrome can make Fast text-sum dwarf Deep-stage
			// savings. If the wire body still shrank, reflect that on the dashboard.
			if deepWire < fastWire && after >= before {
				if wireSaved := fastWire - deepWire; after > wireSaved {
					after = after - wireSaved
				}
			}
			if deepStatus == "" {
				deepStatus = DeepStatusApplied
			}
		} else {
			// Callback soft-fail already sets Status; only fill when missing.
			if deepStatus == "" || deepStatus == DeepStatusApplied {
				if deepRes.Origin > 0 && deepRes.Compressed >= deepRes.Origin {
					deepStatus = DeepStatusFailClosedExpand
				} else {
					deepStatus = DeepStatusRejected
				}
			}
		}
	} else if skipLiveDeepForRequestPath(path) {
		deepStatus = DeepStatusPathSkip
	}

	model := gjson.GetBytes(body, "model").String()
	if routed := gjson.GetBytes(optimized, "model").String(); routed != "" {
		model = routed
	}

	forwardHeaders := r.Header
	adapterUsed := false
	var adapterCfg provideradapt.AdapterConfig
	if path == "/v1/chat/completions" && s.opts.ProviderAdapters != nil {
		if strings.TrimSpace(model) == "" {
			msg := strings.TrimSpace(s.opts.ProviderAdapters.Chrome.ModelRequired)
			if msg == "" {
				msg = "model required"
			}
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		cfg, ok := s.opts.ProviderAdapters.Match(model)
		if !ok {
			msg := strings.TrimSpace(s.opts.UnknownModelFmt)
			if msg == "" {
				http.Error(w, fmt.Sprintf("no provider adapter matched model %q", model), http.StatusBadRequest)
			} else {
				http.Error(w, fmt.Sprintf(msg, model), http.StatusBadRequest)
			}
			return
		}
		resolved, rerr := s.opts.ProviderAdapters.ResolveUpstreamModel(cfg, model)
		if rerr != nil {
			http.Error(w, rerr.Error(), http.StatusBadRequest)
			return
		}
		translatedPath, translatedBody, terr := provideradapt.TranslateRequest(cfg, optimized, resolved)
		if terr != nil {
			msg := s.opts.AdapterTranslateFmt
			if msg == "" {
				http.Error(w, terr.Error(), http.StatusBadRequest)
			} else {
				http.Error(w, fmt.Sprintf(msg, cfg.ID, terr), http.StatusBadRequest)
			}
			return
		}
		hdr, aerr := provideradapt.MapRequestAuth(r.Header, cfg)
		if aerr != nil {
			msg := s.opts.AdapterAuthFmt
			if msg == "" {
				http.Error(w, aerr.Error(), http.StatusUnauthorized)
			} else {
				http.Error(w, fmt.Sprintf(msg, cfg.ID, aerr), http.StatusUnauthorized)
			}
			return
		}
		// Org/multi-workspace keys: client anthropic-workspace-id wins; else TRIM_ANTHROPIC_WORKSPACE_ID.
		// Workspace-scoped keys omit both (official Anthropic behavior).
		provideradapt.ApplyAnthropicWorkspaceID(hdr, s.opts.AnthropicWorkspaceID)
		// OpenAI org/project: client headers win; else TRIM_OPENAI_ORGANIZATION / TRIM_OPENAI_PROJECT.
		if strings.TrimSpace(cfg.Dialect) == provideradapt.DialectOpenAICompat {
			provideradapt.ApplyOpenAIOrgProject(hdr, s.opts.OpenAIOrganization, s.opts.OpenAIProject)
		}
		resolvedBase, berr := provideradapt.ResolveUpstreamBase(cfg, s.opts.UpstreamOpenAI, s.opts.UpstreamAnthropic)
		if berr != nil {
			http.Error(w, berr.Error(), http.StatusBadGateway)
			return
		}
		upstreamBase = resolvedBase
		path = translatedPath
		optimized = translatedBody
		forwardHeaders = hdr
		doorLabel = strings.TrimSpace(cfg.DoorLabel)
		adapterUsed = true
		adapterCfg = cfg
		model = resolved
	}

	// Door A (native Anthropic /v1/messages) and any Anthropic upstream: support org-scoped keys.
	if !adapterUsed && isAnthropicForward(path, upstreamBase, r.Header) {
		forwardHeaders = r.Header.Clone()
		provideradapt.ApplyAnthropicWorkspaceID(forwardHeaders, s.opts.AnthropicWorkspaceID)
	}
	// Native OpenAI chat door (no adapter used): optional OpenAI org/project env fallbacks.
	if !adapterUsed && path == "/v1/chat/completions" {
		forwardHeaders = r.Header.Clone()
		provideradapt.ApplyOpenAIOrgProject(forwardHeaders, s.opts.OpenAIOrganization, s.opts.OpenAIProject)
	}

	// Official Anthropic: tool_result blocks must lead user content after tool_use.
	// Native /v1/messages door (Claude Code) + openai_to_anthropic adapter both land here.
	if strings.Contains(path, "/messages") {
		optimized = provideradapt.SanitizeAnthropicMessagesToolOrder(optimized)
		// Official Anthropic: Claude 4.7+ rejects thinking.type=enabled - migrate to adaptive
		// BEFORE tool_choice normalize (adaptive allows forced tools; enabled does not).
		optimized = provideradapt.NormalizeAnthropicThinkingForModel(optimized)
		// Official Anthropic: manual thinking + forced tool_choice (any/tool) → HTTP 400.
		optimized = provideradapt.NormalizeAnthropicToolChoiceForThinking(optimized)
		// Official Anthropic: thinking / Claude 4.7+ always reject temperature/top_p/top_k.
		optimized = provideradapt.SanitizeAnthropicThinkingIncompatibleFields(optimized)
		// Official Anthropic Claude 4.x: temperature + top_p together → HTTP 400.
		optimized = provideradapt.SanitizeAnthropicTemperatureTopPMutex(optimized)
		// Official Anthropic: Claude 4.6+ reject trailing assistant prefill (HTTP 400).
		optimized = provideradapt.SanitizeAnthropicTrailingAssistantPrefill(optimized)
	}

	resp, err := s.forwardUpstreamWithHeader(r, upstreamBase, path, optimized, forwardHeaders)
	if err != nil {
		// Same-shape provider failover (OpenAI -> Azure/vLLM, Anthropic -> backup).
		if alt := s.failoverBase(upstreamBase); alt != "" {
			resp, err = s.forwardUpstreamWithHeader(r, alt, path, optimized, forwardHeaders)
			if err == nil {
				s.mu.Lock()
				s.stats.UpstreamFailovers++
				s.mu.Unlock()
			}
		}
		if err != nil {
			msg := chrome.ErrUpstreamFmt
			if msg == "" {
				http.Error(w, err.Error(), http.StatusBadGateway)
			} else {
				http.Error(w, fmt.Sprintf(msg, err), http.StatusBadGateway)
			}
			return
		}
	}

	// Uncompressed fallback: if trimming changed the payload and upstream rejected the
	// body shape, retry once with the raw client body.
	// Skip when an adapter rewrote the wire shape (raw OpenAI body is not valid on Anthropic).
	// Do NOT fall back on auth/quota/rate-limit (401/403/429/…) - those fail the same on
	// raw and would erase Fast/Deep savings (Anthropic door showed fake 0% Saved while
	// Gemini openai_compat kept savings because adapterUsed skips this path).
	if !adapterUsed && s.opts.FallbackUncompressed && before > after && resp.StatusCode == http.StatusBadRequest {
		_ = resp.Body.Close()
		s.mu.Lock()
		s.stats.Fallbacks++
		s.mu.Unlock()
		resp, err = s.forwardUpstreamWithHeader(r, upstreamBase, path, body, forwardHeaders)
		if err != nil {
			if alt := s.failoverBase(upstreamBase); alt != "" {
				resp, err = s.forwardUpstreamWithHeader(r, alt, path, body, forwardHeaders)
				if err == nil {
					s.mu.Lock()
					s.stats.UpstreamFailovers++
					s.mu.Unlock()
				}
			}
			if err != nil {
				msg := chrome.ErrUpstreamFmt
				if msg == "" {
					http.Error(w, err.Error(), http.StatusBadGateway)
				} else {
					http.Error(w, fmt.Sprintf(msg, err), http.StatusBadGateway)
				}
				return
			}
		}
		after = before // report raw size on fallback path
	}

	// Provider failover on primary 5xx after a successful HTTP round-trip.
	if resp.StatusCode >= 500 {
		if alt := s.failoverBase(upstreamBase); alt != "" {
			_ = resp.Body.Close()
			retryBody := optimized
			if !adapterUsed && after == before {
				retryBody = body
			}
			altResp, altErr := s.forwardUpstreamWithHeader(r, alt, path, retryBody, forwardHeaders)
			if altErr == nil {
				resp = altResp
				s.mu.Lock()
				s.stats.UpstreamFailovers++
				s.mu.Unlock()
			}
		}
	}

	elapsed := float64(time.Since(start).Microseconds()) / 1000.0
	isError := resp.StatusCode >= 400
	s.recordWithPreview(before, after, elapsed, string(body), string(optimized), doorLabel, adapterUsed, deepStatus, deepStageBefore, deepStageAfter)
	if s.opts.OnMetrics != nil {
		s.opts.OnMetrics(before, after, isError)
	}

	// Adapter stream: translate Anthropic SSE → OpenAI chunks.
	if adapterUsed && provideradapt.NeedsResponseTranslate(adapterCfg) && wantsStream(optimized) && resp.StatusCode < 400 {
	for k, vals := range resp.Header {
			lk := strings.ToLower(k)
			if lk == "content-length" || lk == "content-encoding" || lk == "transfer-encoding" {
				continue
			}
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Del("Content-Length")
		w.Header().Del("Content-Encoding")
	w.Header().Set("X-Trim-Tokens-Before", fmt.Sprintf("%d", before))
	w.Header().Set("X-Trim-Tokens-After", fmt.Sprintf("%d", after))
	w.Header().Set("X-Trim-Latency-Ms", fmt.Sprintf("%.2f", elapsed))
		if doorLabel != "" {
			w.Header().Set("X-Trim-Door", doorLabel)
		}
	w.WriteHeader(resp.StatusCode)
		_ = provideradapt.PipeAnthropicSSEToOpenAI(w, resp.Body, model)
	_ = resp.Body.Close()
		if s.opts.OnEvent != nil && eventMode != "" {
			s.opts.OnEvent(r, model, before, after, elapsed, "success", eventMode, "")
		}
		return
	}

	var upstreamSnippet []byte
	if isError && resp.Body != nil {
		const peekLimit = 4096
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, peekLimit))
		rest, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		full := append(append([]byte{}, snippet...), rest...)
		if adapterUsed && provideradapt.NeedsResponseTranslate(adapterCfg) {
			if fixed, ferr := provideradapt.TranslateResponseBody(adapterCfg, full, resp.StatusCode); ferr == nil {
				full = fixed
			}
		} else if fixed := normalizeOpenAICompatErrorBody(snippet); fixed != nil && len(rest) == 0 {
			full = fixed
		} else if fixed := normalizeOpenAICompatErrorBody(full); fixed != nil {
			full = fixed
		}
		if remapped := remapUpstreamErrorChrome(full, resp.StatusCode, model, s.opts); remapped != nil {
			full = remapped
		}
		upstreamSnippet = full
		resp.Body = io.NopCloser(bytes.NewReader(full))
		clearRebufferedBodyEncoding(resp.Header)
		resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(full)))
		resp.ContentLength = int64(len(full))
	} else if adapterUsed && provideradapt.NeedsResponseTranslate(adapterCfg) && resp.Body != nil {
		raw, rerr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if rerr != nil {
			msg := s.opts.AdapterResponseFmt
			if msg == "" {
				http.Error(w, rerr.Error(), http.StatusBadGateway)
			} else {
				http.Error(w, fmt.Sprintf(msg, adapterCfg.ID, rerr), http.StatusBadGateway)
			}
			return
		}
		fixed, ferr := provideradapt.TranslateResponseBody(adapterCfg, raw, resp.StatusCode)
		if ferr != nil {
			msg := s.opts.AdapterResponseFmt
			if msg == "" {
				http.Error(w, ferr.Error(), http.StatusBadGateway)
			} else {
				http.Error(w, fmt.Sprintf(msg, adapterCfg.ID, ferr), http.StatusBadGateway)
			}
			return
		}
		resp.Body = io.NopCloser(bytes.NewReader(fixed))
		clearRebufferedBodyEncoding(resp.Header)
		resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(fixed)))
		resp.ContentLength = int64(len(fixed))
		resp.Header.Set("Content-Type", "application/json")
	}

	for k, vals := range resp.Header {
		lk := strings.ToLower(k)
		// Never advertise compression for bodies we may have rebuffered as plain JSON.
		if lk == "content-encoding" {
			continue
		}
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.Header().Set("X-Trim-Tokens-Before", fmt.Sprintf("%d", before))
	w.Header().Set("X-Trim-Tokens-After", fmt.Sprintf("%d", after))
	w.Header().Set("X-Trim-Latency-Ms", fmt.Sprintf("%.2f", elapsed))
	if doorLabel != "" {
		w.Header().Set("X-Trim-Door", doorLabel)
	}
	w.WriteHeader(resp.StatusCode)
	s.copyUpstreamBody(w, resp)

	if s.opts.OnEvent != nil {
		status := "success"
		errCode := ""
		if isError {
			status = "error"
			errCode = classifyUpstreamError(resp.StatusCode, upstreamSnippet)
		}
		// Fail-closed: do not invent a mode when unset; caller must configure LiveModeLabel or CompressionMode.
		if eventMode != "" {
			s.opts.OnEvent(r, model, before, after, elapsed, status, eventMode, errCode)
		}
	}
}

func classifyUpstreamError(statusCode int, body []byte) string {
	lower := strings.ToLower(string(body))
	switch {
	case strings.Contains(lower, "out of memory"),
		strings.Contains(lower, "cuda out of memory"),
		strings.Contains(lower, "oom"),
		strings.Contains(lower, "insufficient memory"),
		strings.Contains(lower, "memoryerror"):
		return "oom"
	case statusCode == 529 || strings.Contains(lower, "overloaded"):
		return "upstream_overloaded"
	default:
		return ""
	}
}

func (s *Server) failoverBase(primary string) string {
	primary = strings.TrimRight(strings.TrimSpace(primary), "/")
	openai := strings.TrimRight(strings.TrimSpace(s.opts.UpstreamOpenAI), "/")
	anthropic := strings.TrimRight(strings.TrimSpace(s.opts.UpstreamAnthropic), "/")
	switch primary {
	case openai:
		return strings.TrimSpace(s.opts.UpstreamOpenAIFailover)
	case anthropic:
		return strings.TrimSpace(s.opts.UpstreamAnthropicFailover)
	default:
		return ""
	}
}

func isAnthropicForward(path, upstreamBase string, hdr http.Header) bool {
	if strings.HasPrefix(path, "/v1/messages") {
		return true
	}
	if hdr != nil && strings.TrimSpace(hdr.Get("anthropic-version")) != "" {
		return true
	}
	u := strings.ToLower(strings.TrimSpace(upstreamBase))
	return strings.Contains(u, "anthropic.com") || strings.Contains(u, "anthropic")
}

func (s *Server) forwardUpstream(r *http.Request, upstreamBase, path string, payload []byte) (*http.Response, error) {
	return s.forwardUpstreamWithHeader(r, upstreamBase, path, payload, r.Header)
}

func (s *Server) forwardUpstreamWithHeader(r *http.Request, upstreamBase, path string, payload []byte, hdr http.Header) (*http.Response, error) {
	targetURL, err := url.Parse(upstreamBase)
	if err != nil {
		return nil, err
	}
	u := *targetURL
	u.Path = joinUpstreamPath(targetURL.Path, path, targetURL.Host)
	u.RawQuery = r.URL.RawQuery

	req, err := http.NewRequestWithContext(r.Context(), r.Method, u.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	if hdr == nil {
		hdr = r.Header
	}
	for k, vals := range hdr {
		lk := strings.ToLower(k)
		// Drop Accept-Encoding so net/http Transport can negotiate gzip itself and
		// transparently decompress. Forwarding the client's Accept-Encoding leaves
		// Content-Encoding on the upstream response while adapters re-buffer plain
		// JSON/SSE → VS Code Chromium then fails with net::ERR_CONTENT_DECODING_FAILED.
		if lk == "host" || lk == "content-length" || lk == "accept-encoding" {
			continue
		}
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(payload)))
	req.Host = targetURL.Host

	client := s.http
	if wantsStream(payload) {
		client = s.streamHTTPClient()
	}
	return client.Do(req)
}

// rewriteUpstreamModelAlias maps client model ids onto real provider ids from DB aliases.
func (s *Server) rewriteUpstreamModelAlias(body []byte, upstreamBase string) []byte {
	if !gjson.ValidBytes(body) || s.opts.ProviderAdapters == nil {
		return body
	}
	model := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if model == "" {
		return body
	}
	mapped := s.opts.ProviderAdapters.RewriteModel(model, upstreamBase)
	if mapped == "" || mapped == model {
		return body
	}
	out, err := sjson.SetBytes(body, "model", mapped)
	if err != nil {
		return body
	}
	return out
}

// NormalizeOpenAICompatRequestBody rewrites Cursor BYOK Responses-shaped bodies into
// Chat Completions shape for GPT / Gemini / DeepSeek / Mistral openai_compat doors
// (and for CLI Deep compress). Steps: input→messages, content part migrate,
// legacy function_call→tool_calls, role map, tool message name/id enrich.
// Idempotent when messages already exist.
func NormalizeOpenAICompatRequestBody(body []byte) []byte {
	body = normalizeCursorBYOKChatBody(body)
	// Flat Anthropic-shaped tools can arrive with messages already set (Chat Completions
	// door). Always wrap - do not gate behind Responses-only input→messages rewrite.
	body = normalizeOpenAICompatLegacyFunctionsArray(body)
	body = normalizeOpenAICompatFlatTools(body)
	body = normalizeOpenAICompatToolChoice(body)
	body = normalizeOpenAICompatMessageContentParts(body)
	// Official OpenAI: deprecated assistant.function_call must become tool_calls before
	// role=function→tool and tool_call_id enrich (GPT/Gemini/DeepSeek/Mistral + Claude via Chat).
	body = normalizeOpenAICompatLegacyFunctionCalls(body)
	body = normalizeOpenAICompatMessageRoles(body)
	// Official DeepSeek/Gemini/Mistral/Kimi: role=developer → 400. Map to system
	// (GPT keeps developer). Must run after role normalize, before Fast/Deep freeze.
	body = mapDeveloperRoleForOpenAICompat(body)
	body = enrichOpenAIToolMessageNames(body)
	// Official OpenAI/Gemini/Kimi: tool results must sit contiguously after tool_calls
	// (interleaved user chrome → HTTP 400; Gemini parallel thought_signature loops).
	body = EnsureContiguousToolResultsAfterToolCalls(body)
	// Official OpenAI: GPT-5 / o-series reject max_tokens; both fields together → 400.
	body = sanitizeOpenAIMaxTokensForReasoningModels(body)
	// Official OpenAI/Azure: GPT-5 / o-series reject temperature/top_p/penalties/logprobs/stop.
	body = sanitizeOpenAIReasoningUnsupportedSampling(body)
	// Official Gemini OpenAI-compat: unknown store/metadata/logprobs → HTTP 400.
	body = sanitizeGeminiOpenAICompatUnsupportedFields(body)
	// Official OpenAI (GPT-5.4+): Chat Completions rejects tools + reasoning_effort≠none
	// with HTTP 400 ("Please use /v1/responses instead"). Responses→Chat migrate can
	// inject reasoning_effort from reasoning.effort - sanitize before upstream.
	body = sanitizeGPTChatReasoningEffortWithTools(body)
	// Official DeepSeek / Kimi thinking + tools: every prior assistant turn must pass back
	// reasoning_content (missing/"" → 400). Heal before Fast/Deep so chrome freezes correctly.
	body = provideradapt.EnsureDeepSeekToolCallReasoningContent(body)
	// Official Mistral: tool_call ids must be 9-char alphanumeric before Fast/Deep.
	body = provideradapt.EnsureMistralToolCallIDs(body)
	// Official Mistral openai-compat: strip store / max_completion_tokens / logit_bias /
	// logprobs (422 class); map required→any; clamp temperature [0,1].
	body = provideradapt.SanitizeMistralOpenAICompatRequest(body)
	// Official Gemini: assistant tool_calls content:null → 400; empty tool id → heal.
	body = provideradapt.EnsureGeminiAssistantToolCallContent(body)
	return body
}

// normalizeCursorBYOKChatBody rewrites Cursor Agent Responses-shaped payloads
// (input/max_output_tokens/flat tools) into Chat Completions shape that Gemini
// OpenAI-compat, DeepSeek, Mistral, and GPT gateways accept on /v1/chat/completions.
// Official OpenAI Responses tool loop: function_call + function_call_output items
// must become assistant.tool_calls + role=tool (never smash into role=user).
func normalizeCursorBYOKChatBody(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	if gjson.GetBytes(body, "messages").Exists() {
		return body
	}
	input := gjson.GetBytes(body, "input")
	if !input.Exists() {
		return body
	}

	out := body
	var err error

	// input -> messages
	var msgs []map[string]any
	switch {
	case input.Type == gjson.String:
		msgs = []map[string]any{{"role": "user", "content": input.String()}}
	case input.IsArray():
		var pendingReasoning string
		for _, item := range input.Array() {
			if item.Type == gjson.String {
				msgs = append(msgs, map[string]any{"role": "user", "content": item.String()})
				continue
			}
			if !item.IsObject() {
				continue
			}
			typ := strings.ToLower(strings.TrimSpace(item.Get("type").String()))
			switch typ {
			case "function_call":
				callID := item.Get("call_id").String()
				if callID == "" {
					callID = item.Get("id").String()
				}
				name := item.Get("name").String()
				args := item.Get("arguments").String()
				if args == "" && item.Get("arguments").Exists() {
					args = item.Get("arguments").Raw
				}
				if name == "" {
					continue
				}
				tc := map[string]any{
					"id":   callID,
					"type": "function",
					"function": map[string]any{
						"name":      name,
						"arguments": args,
					},
				}
				// Gemini OpenAI-compat: omit thought_signature → HTTP 400 on tool loops.
				if ec := item.Get("extra_content"); ec.Exists() {
					tc["extra_content"] = ec.Value()
				}
				if sig := strings.TrimSpace(item.Get("thought_signature").String()); sig != "" {
					tc["thought_signature"] = sig
				}
				msgs = appendAssistantToolCall(msgs, tc, &pendingReasoning)
				continue
			case "custom_tool_call":
				// Official Chat Completions: ChatCompletionMessageCustomToolCall
				// {id, type:"custom", custom:{name, input}} - NOT type:function.
				// Responses custom_tool_call uses plain-text `input` (not JSON arguments).
				callID := item.Get("call_id").String()
				if callID == "" {
					callID = item.Get("id").String()
				}
				name := strings.TrimSpace(item.Get("name").String())
				if name == "" {
					continue
				}
				input := item.Get("input").String()
				if input == "" && item.Get("input").Exists() && item.Get("input").Type != gjson.String {
					input = item.Get("input").Raw
				}
				// Some gateways still emit arguments for custom; prefer input, else arguments text.
				if input == "" {
					input = item.Get("arguments").String()
					if input == "" && item.Get("arguments").Exists() {
						input = item.Get("arguments").Raw
					}
				}
				tc := map[string]any{
					"id":   callID,
					"type": "custom",
					"custom": map[string]any{
						"name":  name,
						"input": input,
					},
				}
				if ec := item.Get("extra_content"); ec.Exists() {
					tc["extra_content"] = ec.Value()
				}
				if sig := strings.TrimSpace(item.Get("thought_signature").String()); sig != "" {
					tc["thought_signature"] = sig
				}
				msgs = appendAssistantToolCall(msgs, tc, &pendingReasoning)
				continue
			case "function_call_output", "custom_tool_call_output", "tool_result":
				callID := item.Get("call_id").String()
				if callID == "" {
					callID = item.Get("id").String()
				}
				content := ""
				if o := item.Get("output"); o.Exists() {
					if o.Type == gjson.String {
						content = o.String()
					} else {
						// Official Chat Completions: tool content must be a string (or text
						// parts). Serialize structured output to JSON text - not a user-role
						// dump (garble class). This is intentional json.dumps, not smash.
						content = o.Raw
					}
				} else if c := item.Get("content"); c.Exists() {
					if c.Type == gjson.String {
						content = c.String()
					} else {
						content = c.Raw
					}
				}
				msg := map[string]any{"role": "tool", "content": content}
				if callID != "" {
					msg["tool_call_id"] = callID
				}
				// Mistral (and many openai_compat hosts) expect name on tool messages.
				name := strings.TrimSpace(item.Get("name").String())
				if name == "" {
					name = lookupToolNameFromMessages(msgs, callID)
				}
				if name != "" {
					msg["name"] = name
				}
				msgs = append(msgs, msg)
				continue
			case "reasoning":
				// DeepSeek openai_compat / Chat Completions: plain-text reasoning merges
				// into adjacent assistant as reasoning_content (official DeepSeek Responses
				// bridge). Never inject as user text (would falsely trigger Deep / garble).
				// encrypted_content stays Responses-only - do not put it in reasoning_content.
				text := extractResponsesReasoningPlainText(item)
				if text == "" {
					continue
				}
				if !mergeReasoningIntoPriorAssistant(msgs, text) {
					if pendingReasoning != "" {
						pendingReasoning = pendingReasoning + "\n" + text
					} else {
						pendingReasoning = text
					}
				}
				continue
			case "web_search_call", "file_search_call", "code_interpreter_call",
				"computer_call", "computer_call_output", "image_generation_call",
				"local_shell_call", "local_shell_call_output", "shell_call", "shell_call_output",
				"apply_patch_call", "apply_patch_call_output", "mcp_list_tools",
				"mcp_approval_request", "mcp_approval_response", "mcp_call",
				"tool_search_call", "tool_search_output", "item_reference", "compaction":
				// Responses-only server/tool items have no Chat Completions equivalent.
				// Skip - never dump as role=user (that reintroduces the garble failure).
				continue
			}

			role := item.Get("role").String()
			content := item.Get("content")
			// Unknown typed Responses items that were not handled above must not become
			// invented role=user chrome (same smash/garble class as dumping tool JSON).
			if typ != "" && typ != "message" && role == "" {
				continue
			}
			if role == "" {
				role = "user"
			}
			// Normalize before append so tool_call / reasoning merge sees Chat roles
			// (Gemini role=model, OpenAI legacy role=function).
			role = normalizeCompatMessageRole(role)
			msg := map[string]any{"role": role}
			if content.Exists() {
				msg["content"] = normalizeResponsesContentToChat(content)
			} else if t := item.Get("text"); t.Exists() {
				msg["content"] = t.String()
			} else {
				// No content/text: skip. Never dump item.Raw as user text (garble/expansion).
				continue
			}
			if id := item.Get("call_id").String(); id != "" {
				msg["tool_call_id"] = id
			}
			if id := item.Get("id").String(); id != "" && role == "tool" {
				msg["tool_call_id"] = id
			}
			if isAssistantLikeRole(role) && pendingReasoning != "" {
				msg["reasoning_content"] = pendingReasoning
				pendingReasoning = ""
			}
			// Official OpenAI GPT-5.4+/5.5: assistant phase (commentary|final_answer) must
			// survive Responses→Chat replay or preambles are treated as final answers.
			if isAssistantLikeRole(role) {
				if phase := strings.TrimSpace(item.Get("phase").String()); phase != "" {
					msg["phase"] = phase
				}
			}
			msgs = append(msgs, msg)
		}
	default:
		return body
	}
	if len(msgs) == 0 {
		return body
	}
	out, err = sjson.SetBytes(out, "messages", msgs)
	if err != nil {
		return body
	}
	out, _ = sjson.DeleteBytes(out, "input")

	// Responses top-level instructions == Chat Completions system/developer guidance
	// (official OpenAI migrate guide). Must not be dropped - Gemini/DeepSeek/Mistral
	// openai_compat expect it inside messages, not as a Responses-only field.
	if instr := gjson.GetBytes(out, "instructions"); instr.Exists() {
		var sysContent any
		switch {
		case instr.Type == gjson.String:
			if s := strings.TrimSpace(instr.String()); s != "" {
				sysContent = s
			}
		case instr.IsArray():
			var chunks []string
			for _, item := range instr.Array() {
				if item.Type == gjson.String {
					if s := strings.TrimSpace(item.String()); s != "" {
						chunks = append(chunks, s)
					}
					continue
				}
				if c := item.Get("content"); c.Exists() {
					if c.Type == gjson.String {
						chunks = append(chunks, c.String())
					} else if t := item.Get("text"); t.Exists() {
						chunks = append(chunks, t.String())
					} else if c.IsArray() {
						for _, part := range c.Array() {
							if t := part.Get("text"); t.Exists() {
								chunks = append(chunks, t.String())
							}
						}
					}
				} else if t := item.Get("text"); t.Exists() {
					chunks = append(chunks, t.String())
				}
			}
			if joined := strings.TrimSpace(strings.Join(chunks, "\n")); joined != "" {
				sysContent = joined
			}
		}
		if sysContent != nil {
			existing := gjson.GetBytes(out, "messages").Array()
			prefixed := make([]any, 0, len(existing)+1)
			prefixed = append(prefixed, map[string]any{"role": "system", "content": sysContent})
			for _, m := range existing {
				prefixed = append(prefixed, m.Value())
			}
			out, _ = sjson.SetBytes(out, "messages", prefixed)
		}
		out, _ = sjson.DeleteBytes(out, "instructions")
	}

	// Official migrate: Responses reasoning.effort → Chat Completions reasoning_effort.
	// Must map BEFORE deleting the Responses-only `reasoning` object, or GPT/o-series
	// Cursor BYOK payloads silently lose effort (same class as dropping text.format).
	if !gjson.GetBytes(out, "reasoning_effort").Exists() {
		if effort := gjson.GetBytes(out, "reasoning.effort"); effort.Exists() {
			if s := strings.TrimSpace(effort.String()); s != "" {
				out, _ = sjson.SetBytes(out, "reasoning_effort", s)
			}
		}
	}

	// Official migrate (OpenAI + Microsoft Learn):
	// Responses max_output_tokens → Chat Completions. GPT-5 / o-series reject
	// max_tokens (use max_completion_tokens only). Never set both (OpenAI 400).
	if mot := gjson.GetBytes(out, "max_output_tokens"); mot.Exists() {
		n := mot.Int()
		model := gjson.GetBytes(out, "model").String()
		if chatModelRejectsMaxTokens(model) {
			if !gjson.GetBytes(out, "max_completion_tokens").Exists() {
				out, _ = sjson.SetBytes(out, "max_completion_tokens", n)
			}
			if gjson.GetBytes(out, "max_tokens").Exists() {
				out, _ = sjson.DeleteBytes(out, "max_tokens")
			}
		} else {
			if !gjson.GetBytes(out, "max_tokens").Exists() {
				out, _ = sjson.SetBytes(out, "max_tokens", n)
			}
			// Legacy: some clients also want max_completion_tokens when effort present,
			// but OpenAI rejects both fields together - only set completion when
			// max_tokens is absent (already handled above for reasoning models).
			if gjson.GetBytes(out, "reasoning_effort").Exists() &&
				!gjson.GetBytes(out, "max_tokens").Exists() &&
				!gjson.GetBytes(out, "max_completion_tokens").Exists() {
				out, _ = sjson.SetBytes(out, "max_completion_tokens", n)
			}
		}
		out, _ = sjson.DeleteBytes(out, "max_output_tokens")
	}

	// Official migrate guide: Responses text.format → Chat Completions response_format.
	// Deleting text without this mapping silently drops Structured Outputs on GPT
	// (and any openai_compat host that honors response_format).
	if !gjson.GetBytes(out, "response_format").Exists() {
		if fmtObj := gjson.GetBytes(out, "text.format"); fmtObj.Exists() {
			typ := strings.ToLower(strings.TrimSpace(fmtObj.Get("type").String()))
			switch typ {
			case "json_object", "text":
				out, _ = sjson.SetBytes(out, "response_format", map[string]any{"type": typ})
			case "json_schema":
				js := map[string]any{}
				if name := strings.TrimSpace(fmtObj.Get("name").String()); name != "" {
					js["name"] = name
				}
				if fmtObj.Get("strict").Exists() {
					js["strict"] = fmtObj.Get("strict").Bool()
				}
				if schema := fmtObj.Get("schema"); schema.Exists() {
					js["schema"] = schema.Value()
				} else if schema := fmtObj.Get("json_schema.schema"); schema.Exists() {
					// Tolerate already-nested mistaken shapes.
					js["schema"] = schema.Value()
				}
				if nested := fmtObj.Get("json_schema"); nested.Exists() && !fmtObj.Get("schema").Exists() {
					// Some clients nest json_schema under format already.
					out, _ = sjson.SetBytes(out, "response_format", map[string]any{
						"type":        "json_schema",
						"json_schema": nested.Value(),
					})
				} else {
					out, _ = sjson.SetBytes(out, "response_format", map[string]any{
						"type":        "json_schema",
						"json_schema": js,
					})
				}
			}
		}
	}

	// Official migrate: Responses text.verbosity → Chat Completions top-level verbosity.
	// Must map BEFORE deleting Responses-only `text`, or GPT-5 Cursor BYOK loses it.
	if !gjson.GetBytes(out, "verbosity").Exists() {
		if v := gjson.GetBytes(out, "text.verbosity"); v.Exists() {
			if s := strings.TrimSpace(v.String()); s != "" {
				out, _ = sjson.SetBytes(out, "verbosity", s)
			}
		}
	}

	// Drop Responses-only knobs that confuse chat/completions gateways.
	// Do NOT drop Chat Completions fields (official OpenAI API) that openai_compat
	// doors need: prompt_cache_key, safety_identifier, service_tier, prompt_cache_options,
	// store, stream_options.include_usage, stream_options.include_obfuscation,
	// reasoning_effort, max_completion_tokens, verbosity, parallel_tool_calls, metadata.
	for _, key := range []string{
		"include", "reasoning", "text",
		"previous_response_id", "conversation", "truncation", "prompt",
		// No Chat Completions equivalent (official OpenAI): would 400 on strict hosts
		// (GPT/Gemini/DeepSeek/Mistral openai_compat).
		"max_tool_calls", "background", "context_management",
	} {
		if gjson.GetBytes(out, key).Exists() {
			out, _ = sjson.DeleteBytes(out, key)
		}
	}
	// Never wipe stream_options: Chat Completions officially supports both
	// include_usage and include_obfuscation (GPT/Gemini/DeepSeek/Mistral openai_compat).

	// Responses path: wrap flat tools before return (also run again from
	// NormalizeOpenAICompatRequestBody for messages-already-present Chat bodies).
	out = normalizeOpenAICompatFlatTools(out)
	out = normalizeOpenAICompatToolChoice(out)
	// Same GPT-5.4+ tools+reasoning_effort gate (Responses migrate injects effort).
	out = sanitizeGPTChatReasoningEffortWithTools(out)
	// GPT-5 / o-series max_tokens heal after Responses migrate.
	out = sanitizeOpenAIMaxTokensForReasoningModels(out)
	out = sanitizeOpenAIReasoningUnsupportedSampling(out)
	out = sanitizeGeminiOpenAICompatUnsupportedFields(out)
	// DeepSeek thinking passback heal (Responses→Chat often drops reasoning item text).
	out = provideradapt.EnsureDeepSeekToolCallReasoningContent(out)
	// Mistral tool_call_id length/charset heal after Responses→Chat id migration.
	out = provideradapt.EnsureMistralToolCallIDs(out)
	// Mistral openai-compat field sanitize after Responses→Chat OpenAI field inject.
	out = provideradapt.SanitizeMistralOpenAICompatRequest(out)
	// Gemini assistant tool_calls content/id heal after Responses→Chat.
	return provideradapt.EnsureGeminiAssistantToolCallContent(out)
}

// sanitizeOpenAIMaxTokensForReasoningModels moves max_tokens → max_completion_tokens
// for GPT-5 / GPT-6 / o-series (official: max_tokens unsupported → HTTP 400). Also
// drops max_tokens when both fields are present (OpenAI: simultaneous set → 400),
// and strips stop on o3 / o4-mini (official: not supported).
func sanitizeOpenAIMaxTokensForReasoningModels(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	model := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	out := body
	hasMax := gjson.GetBytes(out, "max_tokens").Exists()
	hasComp := gjson.GetBytes(out, "max_completion_tokens").Exists()

	if chatModelRejectsMaxTokens(model) {
		if hasMax {
			n := gjson.GetBytes(out, "max_tokens").Int()
			if !hasComp && n > 0 {
				var err error
				out, err = sjson.SetBytes(out, "max_completion_tokens", n)
				if err != nil {
					return body
				}
				hasComp = true
			}
			var err error
			out, err = sjson.DeleteBytes(out, "max_tokens")
			if err != nil {
				return body
			}
			hasMax = false
		}
		// Official Azure/OpenAI: stop rejected on GPT-5.x / o-series reasoning deployments.
		if gjson.GetBytes(out, "stop").Exists() {
			var err error
			out, err = sjson.DeleteBytes(out, "stop")
			if err != nil {
				return body
			}
		}
	} else if hasMax && hasComp && isOpenAIGPTOrOSeriesModel(model) {
		// gpt-4o accepts max_completion_tokens alone, but rejects both together.
		var err error
		out, err = sjson.DeleteBytes(out, "max_tokens")
		if err != nil {
			return body
		}
	}
	return out
}

// sanitizeOpenAIReasoningUnsupportedSampling strips Chat Completions sampling fields
// that GPT-5 / GPT-6 / o-series reject (Microsoft Azure Foundry + OpenAI live 400s):
// temperature, top_p, presence_penalty, frequency_penalty, logprobs, top_logprobs, logit_bias.
// GPT-5 only accepts temperature=1 (default) - omitting is the fail-closed path.
func sanitizeOpenAIReasoningUnsupportedSampling(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	model := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	if !chatModelRejectsMaxTokens(model) {
		return body
	}
	out := body
	for _, key := range []string{
		"temperature", "top_p", "presence_penalty", "frequency_penalty",
		"logprobs", "top_logprobs", "logit_bias",
		// Official OpenAI/Azure GPT-5 reasoning: seed / n / best_of are unsupported
		// or greylisted (live unsupported_parameter 400s on strict routes).
		"seed", "n", "best_of",
	} {
		if !gjson.GetBytes(out, key).Exists() {
			continue
		}
		var err error
		out, err = sjson.DeleteBytes(out, key)
		if err != nil {
			return body
		}
	}
	return out
}

// sanitizeGeminiOpenAICompatUnsupportedFields removes Chat Completions fields the
// Google AI OpenAI-compat endpoint rejects with "Unknown name … Cannot find field"
// (store, metadata, logprobs, verbosity, service_tier, safety_identifier, …).
// Cursor/OpenAI SDKs inject these by default.
func sanitizeGeminiOpenAICompatUnsupportedFields(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	model := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	if !strings.Contains(model, "gemini") {
		return body
	}
	out := body
	for _, key := range []string{
		"store", "metadata", "logprobs", "top_logprobs",
		// OpenAI-only Chat fields not in Gemini openai-compat schema (strict 400).
		"verbosity", "service_tier", "safety_identifier",
		"prompt_cache_key", "prompt_cache_retention", "prompt_cache_options",
		"prediction", "web_search_options", "user",
		// Live Google AI openai-compat 400s (forum-confirmed Unknown name):
		"parallel_tool_calls", "presence_penalty", "frequency_penalty",
		"seed", "logit_bias",
	} {
		if !gjson.GetBytes(out, key).Exists() {
			continue
		}
		var err error
		out, err = sjson.DeleteBytes(out, key)
		if err != nil {
			return body
		}
	}
	return out
}

// chatModelRejectsMaxTokens reports models that 400 on Chat Completions max_tokens
// (must use max_completion_tokens instead). Excludes gpt-*-chat* / search-api chat
// variants that still accept max_tokens + sampling (official OpenAI gpt-5-chat-latest).
func chatModelRejectsMaxTokens(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if openAIChatVariantKeepsSampling(m) {
		return false
	}
	if strings.HasPrefix(m, "o1") || strings.HasPrefix(m, "o3") || strings.HasPrefix(m, "o4") {
		return true
	}
	if strings.HasPrefix(m, "gpt-5") || strings.HasPrefix(m, "gpt-6") {
		return true
	}
	return false
}

// openAIChatVariantKeepsSampling reports non-reasoning GPT-5 Chat / search variants
// that officially support temperature/top_p/max_tokens (unlike gpt-5 / gpt-5-mini).
func openAIChatVariantKeepsSampling(modelLower string) bool {
	if strings.Contains(modelLower, "-chat") || strings.Contains(modelLower, "chat-latest") {
		return true
	}
	if strings.Contains(modelLower, "search-api") {
		return true
	}
	return false
}

func isOpenAIGPTOrOSeriesModel(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if strings.HasPrefix(m, "gpt-") {
		return true
	}
	if strings.HasPrefix(m, "o1") || strings.HasPrefix(m, "o3") || strings.HasPrefix(m, "o4") {
		return true
	}
	return false
}

// sanitizeGPTChatReasoningEffortWithTools enforces the official GPT-5.4+ Chat Completions
// gate for function tools (HTTP 400: "Please use /v1/responses instead" / Azure:
// "set reasoning_effort to 'none'"). Covers gpt-5.4 … gpt-5.9 families
// (including -mini/-pro/dated/Azure -sol suffixes).
//
// Critical (Microsoft Azure Foundry docs for gpt-5.6+): these models default
// reasoning_effort to "medium". A tools request that OMITS the field still 400s -
// so we must SET "none" whenever tools are present, not only rewrite non-none values.
func sanitizeGPTChatReasoningEffortWithTools(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	if !chatModelRejectsReasoningEffortWithTools(gjson.GetBytes(body, "model").String()) {
		return body
	}
	tools := gjson.GetBytes(body, "tools")
	functions := gjson.GetBytes(body, "functions")
	hasTools := (tools.IsArray() && len(tools.Array()) > 0) ||
		(functions.IsArray() && len(functions.Array()) > 0)
	if !hasTools {
		return body
	}
	effort := gjson.GetBytes(body, "reasoning_effort")
	if effort.Exists() && effort.Type != gjson.Null &&
		strings.EqualFold(strings.TrimSpace(effort.String()), "none") {
		return body
	}
	out, err := sjson.SetBytes(body, "reasoning_effort", "none")
	if err != nil {
		return body
	}
	return out
}

// chatModelRejectsReasoningEffortWithTools reports GPT-5.4+ Chat Completions models
// that reject tools + non-none reasoning_effort (official OpenAI + live 400s).
// Excludes gpt-*-chat* non-reasoning variants (gpt-5-chat-latest keeps sampling).
func chatModelRejectsReasoningEffortWithTools(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if openAIChatVariantKeepsSampling(m) {
		return false
	}
	if !strings.HasPrefix(m, "gpt-5.") {
		return false
	}
	rest := m[len("gpt-5."):]
	if rest == "" {
		return false
	}
	// Minor version digit: 4–9 (gpt-5.4, gpt-5.5, …). gpt-5.3 and below stay unchanged.
	c := rest[0]
	return c >= '4' && c <= '9'
}

// normalizeOpenAICompatToolChoice maps Responses tool_choice shapes onto Chat Completions.
// Official: Responses function force is flat {"type":"function","name":"x"}; Chat needs
// nested {"type":"function","function":{"name":"x"}}. Hosted/MCP/shell choices have no
// Chat equivalent - fail-closed to "auto" (never leave a 400-causing Responses shape).
func normalizeOpenAICompatToolChoice(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	tc := gjson.GetBytes(body, "tool_choice")
	if !tc.Exists() {
		return body
	}
	if tc.Type == gjson.String {
		return body
	}
	if !tc.IsObject() {
		return body
	}
	typ := strings.ToLower(strings.TrimSpace(tc.Get("type").String()))
	switch typ {
	case "function":
		if tc.Get("function.name").Exists() {
			return body // already Chat-shaped
		}
		name := strings.TrimSpace(tc.Get("name").String())
		if name == "" {
			out, err := sjson.SetBytes(body, "tool_choice", "auto")
			if err != nil {
				return body
			}
			return out
		}
		out, err := sjson.SetBytes(body, "tool_choice", map[string]any{
			"type":     "function",
			"function": map[string]any{"name": name},
		})
		if err != nil {
			return body
		}
		return out
	case "custom":
		// Official Chat Completions: {"type":"custom","custom":{"name":"x"}}.
		// Responses/flat may send {"type":"custom","name":"x"}. Never force to "auto".
		if tc.Get("custom.name").Exists() {
			return body
		}
		name := strings.TrimSpace(tc.Get("name").String())
		if name == "" {
			out, err := sjson.SetBytes(body, "tool_choice", "auto")
			if err != nil {
				return body
			}
			return out
		}
		out, err := sjson.SetBytes(body, "tool_choice", map[string]any{
			"type":   "custom",
			"custom": map[string]any{"name": name},
		})
		if err != nil {
			return body
		}
		return out
	case "allowed_tools":
		// Keep allowed_tools but strip Responses-only hosted/MCP refs that Chat rejects.
		refs := tc.Get("tools")
		if !refs.IsArray() {
			return body
		}
		var kept []any
		for _, ref := range refs.Array() {
			rt := strings.ToLower(strings.TrimSpace(ref.Get("type").String()))
			switch {
			case ref.Get("function").Exists():
				kept = append(kept, ref.Value())
			case ref.Get("custom").Exists() || rt == "custom":
				// Official Chat Completions custom tool refs in allowed_tools.
				if ref.Get("custom.name").Exists() || (rt == "custom" && ref.Get("name").Exists()) {
					name := strings.TrimSpace(ref.Get("custom.name").String())
					if name == "" {
						name = strings.TrimSpace(ref.Get("name").String())
					}
					if name == "" {
						continue
					}
					if ref.Get("custom.name").Exists() {
						kept = append(kept, ref.Value())
					} else {
						kept = append(kept, map[string]any{"type": "custom", "name": name})
					}
					continue
				}
			case rt == "function" || (rt == "" && strings.TrimSpace(ref.Get("name").String()) != ""):
				name := strings.TrimSpace(ref.Get("name").String())
				if name == "" {
					continue
				}
				kept = append(kept, map[string]any{"type": "function", "name": name})
			default:
				// Drop web_search / mcp / image_generation / … refs (no Chat equivalent).
			}
		}
		if len(kept) == 0 {
			out, err := sjson.SetBytes(body, "tool_choice", "auto")
			if err != nil {
				return body
			}
			return out
		}
		mode := strings.TrimSpace(tc.Get("mode").String())
		if mode == "" {
			mode = "auto"
		}
		out, err := sjson.SetBytes(body, "tool_choice", map[string]any{
			"type":  "allowed_tools",
			"mode":  mode,
			"tools": kept,
		})
		if err != nil {
			return body
		}
		return out
	case "mcp", "apply_patch", "shell", "file_search", "web_search_preview",
		"web_search", "computer_use_preview", "computer_use", "code_interpreter",
		"image_generation", "local_shell":
		out, err := sjson.SetBytes(body, "tool_choice", "auto")
		if err != nil {
			return body
		}
		return out
	default:
		return body
	}
}

// normalizeOpenAICompatLegacyFunctionsArray maps deprecated top-level functions[] +
// function_call onto official tools[] + tool_choice (OpenAI Chat Completions).
// Older GPT/Cursor payloads still use functions; Gemini/DeepSeek/Mistral openai_compat
// expect tools - leaving only functions silently disables tool calling (agent smash).
func normalizeOpenAICompatLegacyFunctionsArray(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	out := body
	if !gjson.GetBytes(out, "tools").Exists() {
		fns := gjson.GetBytes(out, "functions")
		if fns.IsArray() && len(fns.Array()) > 0 {
			var tools []any
			for _, f := range fns.Array() {
				name := strings.TrimSpace(f.Get("name").String())
				if name == "" {
					continue
				}
				fn := map[string]any{"name": name}
				if d := f.Get("description"); d.Exists() {
					fn["description"] = d.String()
				}
				if p := f.Get("parameters"); p.Exists() {
					fn["parameters"] = p.Value()
				}
				tools = append(tools, map[string]any{"type": "function", "function": fn})
			}
			if len(tools) > 0 {
				var err error
				out, err = sjson.SetBytes(out, "tools", tools)
				if err != nil {
					return body
				}
				out, err = sjson.DeleteBytes(out, "functions")
				if err != nil {
					return body
				}
			}
		}
	}
	// Deprecated function_call → tool_choice when tool_choice absent.
	if gjson.GetBytes(out, "tool_choice").Exists() {
		return out
	}
	fc := gjson.GetBytes(out, "function_call")
	if !fc.Exists() {
		return out
	}
	var choice any
	switch {
	case fc.Type == gjson.String:
		switch strings.ToLower(strings.TrimSpace(fc.String())) {
		case "none", "auto":
			choice = strings.ToLower(strings.TrimSpace(fc.String()))
		default:
			return out
		}
	case fc.IsObject():
		name := strings.TrimSpace(fc.Get("name").String())
		if name == "" {
			return out
		}
		choice = map[string]any{
			"type":     "function",
			"function": map[string]any{"name": name},
		}
	default:
		return out
	}
	var err error
	out, err = sjson.SetBytes(out, "tool_choice", choice)
	if err != nil {
		return body
	}
	out, err = sjson.DeleteBytes(out, "function_call")
	if err != nil {
		return body
	}
	return out
}

// normalizeOpenAICompatFlatTools wraps Anthropic/Responses flat tools
// ({type,name,description,parameters|input_schema,strict}) into Chat Completions
// {type:"function",function:{...}}, preserving cache_control + function.strict
// (official OpenAI + DeepSeek strict tool mode).
func normalizeOpenAICompatFlatTools(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	tools := gjson.GetBytes(body, "tools")
	if !tools.IsArray() {
		return body
	}
	var fixed []any
	changed := false
	for _, t := range tools.Array() {
		typ := strings.ToLower(strings.TrimSpace(t.Get("type").String()))
		// Already Chat-shaped nested function: keep.
		if t.Get("function").Exists() {
			fixed = append(fixed, t.Value())
			continue
		}
		// Official Chat Completions custom tools: {type:"custom", custom:{name,...}}.
		// Responses may send flat {type:"custom", name, description, format}.
		if typ == "custom" {
			if t.Get("custom").Exists() {
				fixed = append(fixed, t.Value())
				continue
			}
			name := strings.TrimSpace(t.Get("name").String())
			if name == "" {
				changed = true
				continue
			}
			changed = true
			custom := map[string]any{"name": name}
			if d := t.Get("description"); d.Exists() {
				custom["description"] = d.String()
			}
			if f := t.Get("format"); f.Exists() {
				custom["format"] = f.Value()
			}
			tool := map[string]any{"type": "custom", "custom": custom}
			if cc := t.Get("cache_control"); cc.Exists() {
				tool["cache_control"] = cc.Value()
			}
			fixed = append(fixed, tool)
			continue
		}
		name := t.Get("name").String()
		if name == "" {
			// Responses built-ins (web_search_preview, file_search, code_interpreter, …)
			// have no Chat Completions function equivalent. Forwarding them causes 400s
			// on GPT/Gemini/DeepSeek/Mistral openai_compat - drop (never smash into text).
			if typ != "" && typ != "function" {
				changed = true
				continue
			}
			fixed = append(fixed, t.Value())
			continue
		}
		// Anthropic dated server tools (bash_20250124, web_search_20250305, …) must stay
		// native. Chat door should not wrap them into function (would 400 GPT/Gemini and
		// smash Claude if this normalizer were ever mis-applied). Client tools that only
		// carry input_schema are wrapped below for openai_compat / openai_to_anthropic;
		// native /v1/messages never reaches this function (path gate).
		if isAnthropicDatedServerToolType(typ) {
			fixed = append(fixed, t.Value())
			continue
		}
		changed = true
		fn := map[string]any{"name": name}
		if d := t.Get("description"); d.Exists() {
			fn["description"] = d.String()
		}
		if p := t.Get("parameters"); p.Exists() {
			fn["parameters"] = p.Value()
		} else if p := t.Get("input_schema"); p.Exists() {
			fn["parameters"] = p.Value()
		}
		// Responses puts strict at tool top-level; Chat Completions nests it under function.
		// Dropping it silently disables Structured Outputs / DeepSeek strict tool mode.
		if t.Get("strict").Exists() {
			fn["strict"] = t.Get("strict").Bool()
		}
		tool := map[string]any{"type": "function", "function": fn}
		if cc := t.Get("cache_control"); cc.Exists() {
			tool["cache_control"] = cc.Value()
		}
		fixed = append(fixed, tool)
	}
	if !changed {
		return body
	}
	out, err := sjson.SetBytes(body, "tools", fixed)
	if err != nil {
		return body
	}
	return out
}

// isAnthropicDatedServerToolType detects official Anthropic server-tool type strings
// (e.g. bash_20250124, web_search_20250305, text_editor_20250124). These must never be
// wrapped into Chat Completions function tools or dropped as OpenAI hosted builtins.
func isAnthropicDatedServerToolType(typ string) bool {
	typ = strings.ToLower(strings.TrimSpace(typ))
	if typ == "" || typ == "function" || typ == "custom" {
		return false
	}
	// Dated suffix _YYYYMMDD (Anthropic convention).
	for i := 0; i+9 <= len(typ); i++ {
		if typ[i] != '_' {
			continue
		}
		digits := typ[i+1:]
		if len(digits) < 8 {
			continue
		}
		ok := true
		for _, c := range digits[:8] {
			if c < '0' || c > '9' {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// normalizeResponsesContentToChat maps Responses message content parts to Chat Completions
// shapes (official migrate guide: input_text→text, input_image→image_url, input_file→file).
// Forwarding Responses types to GPT/Gemini/DeepSeek/Mistral openai_compat causes 400s and
// leaves Deep unable to see the real question (typ != "text" was skipped).
func normalizeResponsesContentToChat(content gjson.Result) any {
	if !content.Exists() {
		return ""
	}
	if content.Type == gjson.String {
		return content.String()
	}
	if !content.IsArray() {
		return content.Value()
	}
	parts := make([]any, 0, len(content.Array()))
	for _, part := range content.Array() {
		if part.Type == gjson.String {
			parts = append(parts, map[string]any{"type": "text", "text": part.String()})
			continue
		}
		typ := strings.ToLower(strings.TrimSpace(part.Get("type").String()))
		switch typ {
		case "input_text", "output_text", "text", "":
			text := part.Get("text").String()
			if typ == "" && !part.Get("text").Exists() {
				parts = append(parts, part.Value())
				continue
			}
			p := map[string]any{"type": "text", "text": text}
			copyContentPartChrome(part, p)
			parts = append(parts, p)
		case "input_image":
			img := map[string]any{}
			switch {
			case part.Get("image_url").Type == gjson.String:
				if u := strings.TrimSpace(part.Get("image_url").String()); u != "" {
					img["url"] = u
				}
			case part.Get("image_url").IsObject():
				if u := strings.TrimSpace(part.Get("image_url.url").String()); u != "" {
					img["url"] = u
				}
			}
			detail := strings.ToLower(strings.TrimSpace(part.Get("detail").String()))
			switch detail {
			case "low", "high", "auto":
				img["detail"] = detail
			case "original":
				// Chat Completions detail enum has no "original"; use high fidelity.
				img["detail"] = "high"
			}
			if _, ok := img["url"]; !ok {
				// file_id-only images have no Chat Completions equivalent; skip (don't garble).
				continue
			}
			// Official OpenAI: image_url parts may carry prompt_cache_breakpoint.
			p := map[string]any{"type": "image_url", "image_url": img}
			copyContentPartChrome(part, p)
			parts = append(parts, p)
		case "image_url":
			parts = append(parts, part.Value())
		case "input_file":
			file := map[string]any{}
			if id := strings.TrimSpace(part.Get("file_id").String()); id != "" {
				file["file_id"] = id
			}
			if data := strings.TrimSpace(part.Get("file_data").String()); data != "" {
				file["file_data"] = data
			}
			if name := strings.TrimSpace(part.Get("filename").String()); name != "" {
				file["filename"] = name
			}
			// file_url is Responses-only; Chat Completions PDF path needs file_id/file_data.
			if len(file) == 0 {
				continue
			}
			// Official OpenAI: file parts may carry prompt_cache_breakpoint.
			p := map[string]any{"type": "file", "file": file}
			copyContentPartChrome(part, p)
			parts = append(parts, p)
		case "file", "image", "document", "input_audio", "audio":
			parts = append(parts, part.Value())
		case "refusal":
			// Official Chat Completions: keep type=refusal (supports prompt_cache_breakpoint).
			// Never smash into type=text (loses cache markers + refusal semantics).
			if r := strings.TrimSpace(part.Get("refusal").String()); r != "" {
				p := map[string]any{"type": "refusal", "refusal": r}
				copyContentPartChrome(part, p)
				parts = append(parts, p)
			}
		default:
			// Preserve unknown structured parts; never stringify into a text blob.
			parts = append(parts, part.Value())
		}
	}
	return parts
}

func copyContentPartChrome(src gjson.Result, dst map[string]any) {
	if cc := src.Get("cache_control"); cc.Exists() {
		dst["cache_control"] = cc.Value()
	}
	// OpenAI official Chat Completions / Responses: explicit prompt-cache breakpoint.
	if pcb := src.Get("prompt_cache_breakpoint"); pcb.Exists() {
		dst["prompt_cache_breakpoint"] = pcb.Value()
	}
	if cites := src.Get("citations"); cites.Exists() {
		dst["citations"] = cites.Value()
	}
	if ann := src.Get("annotations"); ann.Exists() {
		dst["annotations"] = ann.Value()
	}
	if sig := strings.TrimSpace(src.Get("thought_signature").String()); sig != "" {
		dst["thought_signature"] = sig
	}
	if sig := strings.TrimSpace(src.Get("thoughtSignature").String()); sig != "" {
		dst["thoughtSignature"] = sig
	}
	if ec := src.Get("extra_content"); ec.Exists() {
		dst["extra_content"] = ec.Value()
	}
}

// normalizeOpenAICompatLegacyFunctionCalls converts deprecated assistant.function_call
// into official tool_calls[] (OpenAI Chat Completions). Without this, role=function→tool
// enrichment has no id→name map, Gemini/DeepSeek/Mistral refuse missing tool_call_id,
// and openai_to_anthropic drops the tool_use (silent tool-loop smash).
func normalizeOpenAICompatLegacyFunctionCalls(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return body
	}
	out := body
	changed := false
	for i, m := range messages.Array() {
		if m.Get("tool_calls").Exists() {
			continue
		}
		fc := m.Get("function_call")
		if !fc.IsObject() {
			continue
		}
		name := strings.TrimSpace(fc.Get("name").String())
		if name == "" {
			continue
		}
		args := fc.Get("arguments").String()
		if args == "" && fc.Get("arguments").Exists() && fc.Get("arguments").Type != gjson.String {
			args = fc.Get("arguments").Raw
		}
		id := strings.TrimSpace(fc.Get("id").String())
		if id == "" {
			id = fmt.Sprintf("call_legacy_%d", i)
		}
		tc := []any{
			map[string]any{
				"id":   id,
				"type": "function",
				"function": map[string]any{
					"name":      name,
					"arguments": args,
				},
			},
		}
		var err error
		out, err = sjson.SetBytes(out, fmt.Sprintf("messages.%d.tool_calls", i), tc)
		if err != nil {
			return body
		}
		out, err = sjson.DeleteBytes(out, fmt.Sprintf("messages.%d.function_call", i))
		if err != nil {
			return body
		}
		changed = true
	}
	if !changed {
		return body
	}
	return out
}

// normalizeOpenAICompatMessageRoles maps non-Chat roles that leak into openai_compat
// bodies onto official Chat Completions roles (GPT/Gemini/DeepSeek/Mistral).
// Gemini native uses role=model; OpenAI deprecated role=function → role=tool.
func normalizeOpenAICompatMessageRoles(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return body
	}
	out := body
	changed := false
	for i, m := range messages.Array() {
		role := strings.ToLower(strings.TrimSpace(m.Get("role").String()))
		mapped := normalizeCompatMessageRole(role)
		if mapped == role || mapped == "" {
			continue
		}
		var err error
		out, err = sjson.SetBytes(out, fmt.Sprintf("messages.%d.role", i), mapped)
		if err != nil {
			return body
		}
		changed = true
	}
	if !changed {
		return body
	}
	return out
}

// normalizeOpenAICompatMessageContentParts rewrites messages[].content parts that still
// carry Responses types (Cursor may send messages with input_text even without top-level input).
func normalizeOpenAICompatMessageContentParts(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return body
	}
	out := body
	changed := false
	for i, m := range messages.Array() {
		content := m.Get("content")
		if !content.IsArray() {
			continue
		}
		needs := false
		for _, part := range content.Array() {
			switch strings.ToLower(strings.TrimSpace(part.Get("type").String())) {
			case "input_text", "output_text", "input_image", "input_file", "refusal":
				needs = true
			}
			if needs {
				break
			}
		}
		if !needs {
			continue
		}
		mapped := normalizeResponsesContentToChat(content)
		var err error
		out, err = sjson.SetBytes(out, fmt.Sprintf("messages.%d.content", i), mapped)
		if err != nil {
			return body
		}
		changed = true
	}
	if !changed {
		return body
	}
	return out
}

// extractResponsesReasoningPlainText pulls visible reasoning text from a Responses
// reasoning item (content and/or summary[].text). encrypted_content is intentionally
// ignored - it is not Chat Completions reasoning_content.
func extractResponsesReasoningPlainText(item gjson.Result) string {
	var parts []string
	if c := item.Get("content"); c.Exists() {
		switch {
		case c.Type == gjson.String:
			if s := strings.TrimSpace(c.String()); s != "" {
				parts = append(parts, s)
			}
		case c.IsArray():
			for _, p := range c.Array() {
				if t := p.Get("text"); t.Exists() {
					if s := strings.TrimSpace(t.String()); s != "" {
						parts = append(parts, s)
					}
				} else if p.Type == gjson.String {
					if s := strings.TrimSpace(p.String()); s != "" {
						parts = append(parts, s)
					}
				}
			}
		}
	}
	if sum := item.Get("summary"); sum.IsArray() {
		for _, s := range sum.Array() {
			if t := s.Get("text"); t.Exists() {
				if txt := strings.TrimSpace(t.String()); txt != "" {
					parts = append(parts, txt)
				}
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// normalizeCompatMessageRole maps leaked non-Chat roles onto Chat Completions roles.
// Official OpenAI accepts role=developer (GPT-4o/GPT-5); DeepSeek / Gemini / Mistral /
// Kimi openai-compat reject it with HTTP 400 ("invalid value developer"). Mapping
// happens in normalizeOpenAICompatMessageRoles only for non-GPT models - see
// mapDeveloperRoleForOpenAICompat.
func normalizeCompatMessageRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "model":
		return "assistant"
	case "function":
		return "tool"
	default:
		return strings.ToLower(strings.TrimSpace(role))
	}
}

// mapDeveloperRoleForOpenAICompat rewrites role=developer → role=system for providers
// whose OpenAI-compat endpoints reject developer (DeepSeek live 400; Gemini/Mistral/
// Kimi same class). GPT / o-series keep developer (official OpenAI Chat Completions).
func mapDeveloperRoleForOpenAICompat(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	model := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	if isOpenAIGPTOrOSeriesModel(model) {
		return body
	}
	// Only rewrite for known non-OpenAI openai_compat families (fail-closed for
	// developer rejection). Unknown custom OpenAI-compat hosts that DO accept
	// developer are rare; GPT-shaped agents hitting DeepSeek/Gemini/Mistral is the
	// live failure class.
	if !(strings.Contains(model, "deepseek") ||
		strings.Contains(model, "gemini") ||
		strings.Contains(model, "mistral") ||
		strings.Contains(model, "codestral") ||
		strings.Contains(model, "magistral") ||
		strings.Contains(model, "pixtral") ||
		strings.Contains(model, "ministral") ||
		strings.Contains(model, "kimi") ||
		strings.Contains(model, "moonshot") ||
		// Official MiniMax openai-compat: role=developer → 400 "invalid role: developer (2013)".
		strings.Contains(model, "minimax")) {
		return body
	}
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return body
	}
	out := body
	changed := false
	for i, m := range messages.Array() {
		role := strings.ToLower(strings.TrimSpace(m.Get("role").String()))
		if role != "developer" {
			continue
		}
		var err error
		out, err = sjson.SetBytes(out, fmt.Sprintf("messages.%d.role", i), "system")
		if err != nil {
			return body
		}
		changed = true
	}
	if !changed {
		return body
	}
	return out
}

// mergeReasoningIntoPriorAssistant attaches reasoning text to the nearest prior
// assistant message (skipping tool turns). Returns false if none found.
func mergeReasoningIntoPriorAssistant(msgs []map[string]any, text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		role, _ := msgs[i]["role"].(string)
		role = strings.ToLower(strings.TrimSpace(role))
		if isAssistantLikeRole(role) {
			if existing, ok := msgs[i]["reasoning_content"].(string); ok && strings.TrimSpace(existing) != "" {
				msgs[i]["reasoning_content"] = strings.TrimSpace(existing) + "\n" + text
			} else {
				msgs[i]["reasoning_content"] = text
			}
			// Defense: ensure Chat role spelling if a model alias leaked in.
			msgs[i]["role"] = "assistant"
			return true
		}
		switch role {
		case "user", "system", "developer":
			return false
		}
	}
	return false
}

// lookupToolNameFromMessages finds function.name for a tool_call id already emitted
// into msgs (Responses normalize builds assistant.tool_calls before tool outputs).
// appendAssistantToolCall attaches a Chat tool_call onto the prior assistant message
// (parallel calls / message+call siblings) or starts a new assistant tool_calls turn.
func appendAssistantToolCall(msgs []map[string]any, tc map[string]any, pendingReasoning *string) []map[string]any {
	if len(msgs) > 0 {
		last := msgs[len(msgs)-1]
		if role, _ := last["role"].(string); isAssistantLikeRole(role) {
			last["role"] = "assistant"
			if tcs, ok := last["tool_calls"].([]any); ok {
				last["tool_calls"] = append(tcs, tc)
			} else {
				last["tool_calls"] = []any{tc}
			}
			if pendingReasoning != nil && *pendingReasoning != "" {
				if _, has := last["reasoning_content"]; !has {
					last["reasoning_content"] = *pendingReasoning
				}
				*pendingReasoning = ""
			}
			return msgs
		}
	}
	asst := map[string]any{
		"role":       "assistant",
		"content":    nil,
		"tool_calls": []any{tc},
	}
	if pendingReasoning != nil && *pendingReasoning != "" {
		asst["reasoning_content"] = *pendingReasoning
		*pendingReasoning = ""
	}
	return append(msgs, asst)
}

func lookupToolNameFromMessages(msgs []map[string]any, callID string) string {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return ""
	}
	for _, m := range msgs {
		tcs, ok := m["tool_calls"].([]any)
		if !ok {
			continue
		}
		for _, raw := range tcs {
			tc, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			id, _ := tc["id"].(string)
			if strings.TrimSpace(id) != callID {
				continue
			}
			if fn, ok := tc["function"].(map[string]any); ok {
				if name, _ := fn["name"].(string); strings.TrimSpace(name) != "" {
					return strings.TrimSpace(name)
				}
			}
			// Official Chat custom tool_calls: custom.name (not function.name).
			if custom, ok := tc["custom"].(map[string]any); ok {
				if name, _ := custom["name"].(string); strings.TrimSpace(name) != "" {
					return strings.TrimSpace(name)
				}
			}
		}
	}
	return ""
}

// enrichOpenAIToolMessageNames fills messages[i].name for role=tool when missing,
// using the matching assistant tool_calls[].function.name or custom.name.
// Also fills missing tool_call_id from name (legacy role=function). Mistral docs
// include name on tool messages; OpenAI accepts it; required for Gemini/DeepSeek ids.
func enrichOpenAIToolMessageNames(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return body
	}
	idToName := map[string]string{}
	nameToIDs := map[string][]string{}
	usedIDs := map[string]bool{}
	for _, m := range messages.Array() {
		tcs := m.Get("tool_calls")
		if tcs.IsArray() {
			for _, tc := range tcs.Array() {
				id := strings.TrimSpace(tc.Get("id").String())
				name := strings.TrimSpace(tc.Get("function.name").String())
				if name == "" {
					name = strings.TrimSpace(tc.Get("custom.name").String())
				}
				if id != "" && name != "" {
					idToName[id] = name
					nameToIDs[name] = append(nameToIDs[name], id)
				}
			}
		}
		// Defense: still-present legacy function_call (if convert was skipped).
		if fc := m.Get("function_call"); fc.IsObject() {
			name := strings.TrimSpace(fc.Get("name").String())
			id := strings.TrimSpace(fc.Get("id").String())
			if name != "" && id != "" {
				idToName[id] = name
				nameToIDs[name] = append(nameToIDs[name], id)
			}
		}
		role := strings.ToLower(strings.TrimSpace(m.Get("role").String()))
		if role == "tool" || role == "function" {
			if id := strings.TrimSpace(m.Get("tool_call_id").String()); id != "" {
				usedIDs[id] = true
			}
		}
	}
	if len(idToName) == 0 && len(nameToIDs) == 0 {
		return body
	}
	out := body
	for i, m := range messages.Array() {
		role := strings.ToLower(strings.TrimSpace(m.Get("role").String()))
		if role != "tool" && role != "function" {
			continue
		}
		name := strings.TrimSpace(m.Get("name").String())
		id := strings.TrimSpace(m.Get("tool_call_id").String())
		// Official Chat: tool messages need tool_call_id. Legacy role=function often
		// only has name - assign next unused id for that name (parallel tool calls).
		if id == "" && name != "" {
			for _, candidate := range nameToIDs[name] {
				if usedIDs[candidate] {
					continue
				}
				var err error
				out, err = sjson.SetBytes(out, fmt.Sprintf("messages.%d.tool_call_id", i), candidate)
				if err != nil {
					return body
				}
				id = candidate
				usedIDs[candidate] = true
				break
			}
		}
		// Mistral (and some Gemini OpenAI-compat builds) want name on tool messages.
		if name == "" && id != "" {
			if mapped := idToName[id]; mapped != "" {
				var err error
				out, err = sjson.SetBytes(out, fmt.Sprintf("messages.%d.name", i), mapped)
				if err != nil {
					return body
				}
			}
		}
	}
	return out
}

// EnsureContiguousToolResultsAfterToolCalls pulls role=tool messages that belong to an
// assistant.tool_calls turn so they sit immediately after that assistant, before any
// interleaved user/system chrome. Official OpenAI + Gemini + Kimi/Mistral-strict:
// assistant(tool_calls) must be followed by contiguous tool results (interleave → 400;
// Gemini thought_signature parallel loops reject FC1,FR1,FC2,FR2 style splits).
func EnsureContiguousToolResultsAfterToolCalls(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return body
	}
	arr := messages.Array()
	n := len(arr)
	if n < 3 {
		return body
	}

	type msgView struct {
		raw  string
		role string
	}
	views := make([]msgView, n)
	for i, m := range arr {
		views[i] = msgView{
			raw:  m.Raw,
			role: strings.ToLower(strings.TrimSpace(m.Get("role").String())),
		}
	}

	used := make([]bool, n)
	var rebuilt []any
	changed := false
	for i := 0; i < n; i++ {
		if used[i] {
			continue
		}
		m := arr[i]
		role := views[i].role
		tcs := m.Get("tool_calls")
		hasTC := (role == "assistant" || role == "model") && tcs.IsArray() && len(tcs.Array()) > 0
		if !hasTC {
			var v any
			if err := json.Unmarshal([]byte(views[i].raw), &v); err != nil {
				return body
			}
			rebuilt = append(rebuilt, v)
			used[i] = true
			continue
		}
		need := map[string]bool{}
		for _, tc := range tcs.Array() {
			if id := strings.TrimSpace(tc.Get("id").String()); id != "" {
				need[id] = true
			}
		}
		var asst any
		if err := json.Unmarshal([]byte(views[i].raw), &asst); err != nil {
			return body
		}
		rebuilt = append(rebuilt, asst)
		used[i] = true
		if len(need) == 0 {
			continue
		}

		var matchingIdx, heldIdx []int
		j := i + 1
		for j < n {
			if used[j] {
				j++
				continue
			}
			mj := arr[j]
			rj := views[j].role
			// Next assistant tool-call turn starts a new batch.
			if (rj == "assistant" || rj == "model") &&
				mj.Get("tool_calls").IsArray() && len(mj.Get("tool_calls").Array()) > 0 {
				break
			}
			if rj == "tool" || rj == "function" {
				id := strings.TrimSpace(mj.Get("tool_call_id").String())
				if id != "" && need[id] {
					matchingIdx = append(matchingIdx, j)
					delete(need, id)
				} else {
					heldIdx = append(heldIdx, j)
				}
			} else {
				heldIdx = append(heldIdx, j)
			}
			j++
			if len(need) == 0 {
				break
			}
		}
		// Detect interleave: a held message appears before the last matching tool in
		// original order (user between tools), or matching tools are not contiguous
		// right after the assistant.
		if len(matchingIdx) > 0 {
			expect := i + 1
			for _, idx := range matchingIdx {
				if idx != expect {
					changed = true
					break
				}
				expect++
			}
			if !changed {
				for _, idx := range heldIdx {
					if idx < matchingIdx[len(matchingIdx)-1] {
						changed = true
						break
					}
				}
			}
		}
		for _, idx := range matchingIdx {
			var v any
			if err := json.Unmarshal([]byte(views[idx].raw), &v); err != nil {
				return body
			}
			rebuilt = append(rebuilt, v)
			used[idx] = true
		}
		for _, idx := range heldIdx {
			var v any
			if err := json.Unmarshal([]byte(views[idx].raw), &v); err != nil {
				return body
			}
			rebuilt = append(rebuilt, v)
			used[idx] = true
		}
	}
	if !changed {
		return body
	}
	out, err := sjson.SetBytes(body, "messages", rebuilt)
	if err != nil {
		return body
	}
	return out
}

// normalizeOpenAICompatErrorBody unwraps Gemini-style [{ "error": ... }] to { "error": ... }.
func normalizeOpenAICompatErrorBody(body []byte) []byte {
	trim := bytes.TrimSpace(body)
	if len(trim) < 2 || trim[0] != '[' {
		return nil
	}
	if !gjson.ValidBytes(trim) {
		return nil
	}
	first := gjson.GetBytes(trim, "0")
	if !first.Exists() || !first.IsObject() {
		return nil
	}
	if !first.Get("error").Exists() {
		return nil
	}
	return []byte(first.Raw)
}

// joinUpstreamPath keeps OpenAI host roots on /v1/..., and preserves path prefixes
// for OpenAI-compatible gateways (e.g. Gemini: /v1beta/openai + /chat/completions).
func joinUpstreamPath(basePath, reqPath, host string) string {
	reqPath = "/" + strings.TrimPrefix(strings.TrimSpace(reqPath), "/")
	basePath = strings.TrimSuffix(strings.TrimSpace(basePath), "/")
	if basePath == "" || basePath == "/" {
		return reqPath
	}
	rel := reqPath
	if strings.Contains(strings.ToLower(host), "generativelanguage.googleapis.com") &&
		strings.HasPrefix(rel, "/v1/") {
		rel = strings.TrimPrefix(rel, "/v1")
		if rel == "" {
			rel = "/"
		}
	}
	if !strings.HasPrefix(rel, "/") {
		rel = "/" + rel
	}
	return basePath + rel
}

// wantsStream detects OpenAI/Anthropic stream:true in the outbound JSON body.
func wantsStream(payload []byte) bool {
	if !gjson.ValidBytes(payload) {
		return false
	}
	return gjson.GetBytes(payload, "stream").Bool()
}

// clearRebufferedBodyEncoding drops compression headers after Trim replaces the
// upstream body with a plain buffer (error remap / anthropic_messages translate).
// Leaving Content-Encoding: gzip on uncompressed JSON breaks Chromium clients
// (VS Code Custom Endpoint → net::ERR_CONTENT_DECODING_FAILED).
func clearRebufferedBodyEncoding(h http.Header) {
	if h == nil {
		return
	}
	h.Del("Content-Encoding")
	h.Del("Transfer-Encoding")
}

// copyUpstreamBody streams SSE chunks with Flush so IDEs see tokens in real time.
// Non-stream responses use a single io.Copy.
func (s *Server) copyUpstreamBody(w http.ResponseWriter, resp *http.Response) {
	defer resp.Body.Close()
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	streaming := strings.Contains(ct, "text/event-stream") ||
		strings.Contains(ct, "application/stream") ||
		strings.Contains(ct, "ndjson")
	if !streaming {
		_, _ = io.Copy(w, resp.Body)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		_, _ = io.Copy(w, resp.Body)
		return
	}
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return
			}
			flusher.Flush()
		}
		if err != nil {
			return
		}
	}
}

func (s *Server) maybeRouteModel(body []byte, tokenEstimate int) []byte {
	cheap := strings.TrimSpace(s.opts.CheapModel)
	if cheap == "" || s.opts.RouteMaxTokens <= 0 {
		return body
	}
	if tokenEstimate > s.opts.RouteMaxTokens {
		return body
	}
	out, err := sjson.SetBytes(body, "model", cheap)
	if err != nil {
		return body
	}
	return out
}

func (s *Server) optimizePayload(body []byte) ([]byte, int, int) {
	if !gjson.ValidBytes(body) {
		return body, estimate(body), estimate(body)
	}

	userPrompt := ""
	messages := gjson.GetBytes(body, "messages")
	if messages.IsArray() {
		arr := messages.Array()
		// Prefer the trailing user turn's plain text (not system/tool chrome, not JSON array dump).
		// Array content .String() is empty in gjson - that used to leave userPrompt blank on
		// multimodal Claude/GPT/Gemini turns and weakened Fast active-file protection.
		for i := len(arr) - 1; i >= 0; i-- {
			role := strings.ToLower(strings.TrimSpace(arr[i].Get("role").String()))
			if role != "user" {
				continue
			}
			userPrompt = extractUserPromptText(arr[i].Get("content"))
			break
		}
		if userPrompt == "" && len(arr) > 0 {
			userPrompt = extractUserPromptText(arr[len(arr)-1].Get("content"))
		}
	}

	totalBefore := 0
	totalAfter := 0
	out := body

	// Prompt-cache awareness: never mutate Anthropic top-level system or any
	// message/part that carries cache_control / role=system. Altering those
	// bytes breaks provider prompt-cache hashes and can raise API cost.
	systemField := gjson.GetBytes(body, "system")
	if systemField.Exists() {
		n := estimate([]byte(systemField.Raw))
		totalBefore += n
		totalAfter += n
	}

	// Official Anthropic automatic caching (top-level cache_control) and OpenAI
	// GPT-5.6+ prompt_cache_options / legacy prompt_cache_key (+ deprecated
	// prompt_cache_retention): the reusable prefix is byte-stable across turns.
	// History-stubbing or Fast-mutating earlier messages invalidates that prefix
	// (same smash/cost class as mutating cache_control parts). Last-user-only Fast
	// matches Deep's last-user-only rule for these cache-aware requests - and only
	// when the trailing message is that user turn (Deep ShouldSkipDeep parity).
	requestPromptCache := gjson.GetBytes(body, "cache_control").Exists() ||
		gjson.GetBytes(body, "prompt_cache_options").Exists() ||
		gjson.GetBytes(body, "prompt_cache_key").Exists() ||
		gjson.GetBytes(body, "prompt_cache_retention").Exists()
	lastUserIdx := -1
	msgCount := 0
	if messages.IsArray() {
		arr := messages.Array()
		msgCount = len(arr)
		for i := len(arr) - 1; i >= 0; i-- {
			if strings.ToLower(strings.TrimSpace(arr[i].Get("role").String())) == "user" {
				lastUserIdx = i
				break
			}
		}
	}
	// Only the fresh trailing user turn may Fast-mutate under request-level cache.
	// If the body ends on assistant/tool (mid-loop), freeze everything.
	cacheAllowFastIdx := -1
	if requestPromptCache && lastUserIdx >= 0 && lastUserIdx == msgCount-1 {
		cacheAllowFastIdx = lastUserIdx
	}

	collapseIdx := map[int]bool{}
	if messages.IsArray() && s.opts.HistoryKeepTurns > 0 && !requestPromptCache {
		var convo []int
		for i, msg := range messages.Array() {
			role := strings.ToLower(strings.TrimSpace(msg.Get("role").String()))
			// Preserve prompt-cache, system/developer, Claude reminders, and tool protocol
			// for Anthropic + OpenAI-compat agents (GPT / Gemini / DeepSeek / Cursor).
			if role == "system" || role == "developer" || role == "tool" || role == "function" ||
				msg.Get("cache_control").Exists() || msg.Get("prompt_cache_breakpoint").Exists() ||
				messageHasPromptCacheBreakpoint(msg) ||
				messageHasCitedContent(msg) ||
				messageHasRefusalChrome(msg) ||
				messageHasSystemReminder(msg) ||
				messageHasToolScaffolding(msg) || messageHasReasoningChrome(msg) ||
				messageHasMultimodalParts(msg) ||
				(isAssistantLikeRole(role) && strings.TrimSpace(msg.Get("phase").String()) != "") ||
				// Official Mistral: trailing/history assistant with prefix:true is prepended
				// verbatim to the model output - Fast must not mutate or stub that text.
				(isAssistantLikeRole(role) && msg.Get("prefix").Bool()) {
				continue
			}
			convo = append(convo, i)
		}
		if len(convo) > s.opts.HistoryKeepTurns {
			cut := len(convo) - s.opts.HistoryKeepTurns
			for _, idx := range convo[:cut] {
				collapseIdx[idx] = true
			}
		}
	}

	if messages.IsArray() {
		for i, msg := range messages.Array() {
			role := strings.ToLower(strings.TrimSpace(msg.Get("role").String()))
			content := msg.Get("content")
			path := fmt.Sprintf("messages.%d.content", i)
			freezeMsg := role == "system" || role == "developer" || role == "tool" || role == "function" ||
				msg.Get("cache_control").Exists() || msg.Get("prompt_cache_breakpoint").Exists() ||
				messageHasPromptCacheBreakpoint(msg) ||
				messageHasCitedContent(msg) ||
				messageHasRefusalChrome(msg) ||
				messageHasReasoningChrome(msg) ||
				messageHasSystemReminder(msg) ||
				// Official Chat Completions: message.audio lives outside content[].
				// Fast-mutating paired content while leaving audio would desync the
				// audio round-trip (same smash class as stubbing the turn).
				(msg.Get("audio").Exists() && msg.Get("audio").Type != gjson.Null) ||
				// Official OpenAI GPT-5.4+/5.5: assistant phase must stay byte-stable with
				// its content (mutating commentary preambles → final-answer misbehavior).
				(isAssistantLikeRole(role) && strings.TrimSpace(msg.Get("phase").String()) != "") ||
				// Official Mistral: prefix:true means this assistant text is prepended to
				// the model completion - mutating it breaks the forced-prefix contract.
				(isAssistantLikeRole(role) && msg.Get("prefix").Bool()) ||
				// Assistant tool-call turns (GPT/Gemini/DeepSeek): Fast must not mutate the
				// paired content while leaving tool_calls alone - keep the whole turn stable.
				// role=model is Gemini native assistant alias (may leak into openai_compat bodies).
				(isAssistantLikeRole(role) && messageHasToolScaffolding(msg)) ||
				// Request-level prompt cache: freeze every turn except a trailing user
				// (new content is not part of the prior reusable prefix). Mid-loop
				// bodies ending on assistant/tool freeze all turns (Deep parity).
				(requestPromptCache && i != cacheAllowFastIdx)
			// Note: user messages with tool_result+text are NOT fully frozen - non-text parts
			// are skipped (type != "text"); freezing the whole user turn blocked Fast on the
			// follow-up question (same class of bug as Deep).

			if collapseIdx[i] {
				raw := content.String()
				if content.IsArray() {
					raw = content.Raw
				}
				b := estimate([]byte(raw))
				totalBefore += b
				stub := trimmer.MarkerHistoryCompacted
				totalAfter += estimate([]byte(stub))
				var err error
				out, err = sjson.SetBytes(out, path, stub)
				if err != nil {
					return body, estimate(body), estimate(body)
				}
				continue
			}

			if content.Type == gjson.String {
				raw := content.String()
				b := estimate([]byte(raw))
				// Claude Code may place reminders in a single string content field.
				if freezeMsg || strings.Contains(raw, "<system-reminder>") {
					totalBefore += b
					totalAfter += b
					continue
				}
				optimized, before, after := s.optimizeText(raw, userPrompt)
				totalBefore += before
				totalAfter += after
				var err error
				out, err = sjson.SetBytes(out, path, optimized)
				if err != nil {
					return body, estimate(body), estimate(body)
				}
			} else if content.IsArray() {
				for j, part := range content.Array() {
					typ := strings.ToLower(strings.TrimSpace(part.Get("type").String()))
					// Chat Completions uses "text"; Responses leftovers may still say input_text/output_text
					// until normalizeOpenAICompatMessageContentParts runs (defense in depth).
					if typ != "" && typ != "text" && typ != "input_text" && typ != "output_text" {
						continue
					}
					if !part.Get("text").Exists() {
						continue
					}
					text := part.Get("text").String()
					b := estimate([]byte(text))
					// Claude Code gateway: never mutate attribution / system-reminder blocks
					// (official protocol: preserve client scaffolding; cache_control optional).
					// OpenAI: never mutate parts carrying prompt_cache_breakpoint (explicit caching).
					// Anthropic citations / OpenAI annotations: mutating text orphans structured refs.
					// Gemini: never mutate parts carrying thought_signature (omit → HTTP 400).
					freezePart := freezeMsg || part.Get("cache_control").Exists() ||
						part.Get("prompt_cache_breakpoint").Exists() ||
						part.Get("citations").Exists() || part.Get("annotations").Exists() ||
						strings.Contains(text, "<system-reminder>") ||
						partHasThoughtSignature(part)
					if freezePart {
						totalBefore += b
						totalAfter += b
						continue
					}
					optimized, before, after := s.optimizeText(text, userPrompt)
					totalBefore += before
					totalAfter += after
					p := fmt.Sprintf("messages.%d.content.%d.text", i, j)
					var err error
					out, err = sjson.SetBytes(out, p, optimized)
					if err != nil {
						return body, estimate(body), estimate(body)
					}
					// If Responses type leaked through, rewrite type to Chat Completions "text".
					if typ == "input_text" || typ == "output_text" {
						out, err = sjson.SetBytes(out, fmt.Sprintf("messages.%d.content.%d.type", i, j), "text")
						if err != nil {
							return body, estimate(body), estimate(body)
						}
					}
				}
			}
		}
	}

	if totalBefore == 0 {
		totalBefore = estimate(body)
		totalAfter = estimate(out)
	}
	return out, totalBefore, totalAfter
}

// isAssistantLikeRole reports Chat Completions assistant or Gemini native role=model
// (official Gemini GenerateContent uses "model"; some clients leak it into openai_compat).
func isAssistantLikeRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "assistant", "model":
		return true
	default:
		return false
	}
}

// messageHasSystemReminder detects Claude Code attribution / reminder blocks that
// must not be history-stubbed or Fast-mutated (gateway protocol preserve).
func messageHasSystemReminder(msg gjson.Result) bool {
	content := msg.Get("content")
	switch {
	case content.Type == gjson.String:
		return strings.Contains(content.String(), "<system-reminder>")
	case content.IsArray():
		for _, part := range content.Array() {
			if t := part.Get("text"); t.Exists() && strings.Contains(t.String(), "<system-reminder>") {
				return true
			}
		}
	}
	return false
}

// messageHasToolScaffolding detects OpenAI Chat Completions + Anthropic Messages tool
// protocol chrome (tool_calls, role=tool, tool_use/tool_result parts). Must stay verbatim
// for GPT / Gemini / DeepSeek / Mistral openai_compat and Claude Code agent loops.
func messageHasToolScaffolding(msg gjson.Result) bool {
	role := strings.ToLower(strings.TrimSpace(msg.Get("role").String()))
	switch role {
	case "tool", "function":
		return true
	}
	if msg.Get("tool_calls").Exists() || msg.Get("function_call").Exists() {
		return true
	}
	if strings.TrimSpace(msg.Get("tool_call_id").String()) != "" {
		return true
	}
	content := msg.Get("content")
	if !content.IsArray() {
		return false
	}
	for _, part := range content.Array() {
		if isToolProtocolPartType(strings.ToLower(strings.TrimSpace(part.Get("type").String()))) {
			return true
		}
	}
	return false
}

// isToolProtocolPartType matches Anthropic/OpenAI tool protocol content-block types.
// Suffix heuristic covers server tools (bash/code execution, web_fetch, text editor, …).
func isToolProtocolPartType(typ string) bool {
	switch typ {
	case "tool_use", "tool_result", "server_tool_use", "web_search_tool_result",
		"function_call", "function_call_output", "tool_call",
		"mcp_tool_use", "mcp_tool_result", "mcp_tool_listing", "tool_reference":
		return true
	}
	if strings.HasSuffix(typ, "_tool_result") || strings.HasSuffix(typ, "_tool_use") {
		return true
	}
	return false
}

// messageHasReasoningChrome detects DeepSeek/OpenAI-compat reasoning_content,
// Anthropic thinking blocks, and Gemini thought signatures that must not be
// Fast-mutated or history-stubbed (official: omit thought_signature → HTTP 400;
// DeepSeek: omit reasoning_content on tool turns → HTTP 400).
func messageHasReasoningChrome(msg gjson.Result) bool {
	if msg.Get("reasoning_content").Exists() {
		return true
	}
	// Official MiniMax openai-compat: reasoning_details must round-trip with tool loops
	// (reasoning_split=true). Fast-mutating / history-stubbing drops the chain.
	if rd := msg.Get("reasoning_details"); rd.Exists() && rd.Type != gjson.Null {
		return true
	}
	// OpenAI-compat / openai_to_anthropic stream round-trip: thinking signatures live on
	// top-level thinking_blocks (not only inside content[]). Omitting → Anthropic HTTP 400.
	if tbs := msg.Get("thinking_blocks"); tbs.IsArray() && len(tbs.Array()) > 0 {
		return true
	}
	if strings.TrimSpace(msg.Get("extra_content.google.thought_signature").String()) != "" {
		return true
	}
	if tcs := msg.Get("tool_calls"); tcs.IsArray() {
		for _, tc := range tcs.Array() {
			if strings.TrimSpace(tc.Get("extra_content.google.thought_signature").String()) != "" {
				return true
			}
			if strings.TrimSpace(tc.Get("thought_signature").String()) != "" {
				return true
			}
		}
	}
	content := msg.Get("content")
	// Official MiniMax native openai-compat: thinking is embedded in content as
	// <think>...</think> when reasoning_split=false. Mutating that string breaks
	// interleaved thinking replay (same smash class as stripping reasoning_content).
	if content.Type == gjson.String {
		s := content.String()
		if strings.Contains(s, "<think>") || strings.Contains(s, "</think>") {
			return true
		}
		return false
	}
	if !content.IsArray() {
		return false
	}
	for _, part := range content.Array() {
		switch strings.ToLower(strings.TrimSpace(part.Get("type").String())) {
		case "thinking", "redacted_thinking", "reasoning":
			return true
		}
		if t := part.Get("text"); t.Exists() {
			s := t.String()
			if strings.Contains(s, "<think>") || strings.Contains(s, "</think>") {
				return true
			}
		}
		if strings.TrimSpace(part.Get("thought_signature").String()) != "" ||
			strings.TrimSpace(part.Get("thoughtSignature").String()) != "" ||
			strings.TrimSpace(part.Get("extra_content.google.thought_signature").String()) != "" {
			return true
		}
		// Official Gemini: thought summaries use thought:true on content parts.
		if part.Get("thought").Bool() {
			return true
		}
	}
	return false
}

// messageHasMultimodalParts detects OpenAI/Anthropic non-text content that must stay
// verbatim (official: preserve image_url / input_audio / file / document across history).
// History-stubbing these into a text marker destroys vision/audio context for GPT/Gemini/Claude.
func messageHasMultimodalParts(msg gjson.Result) bool {
	// Official Chat Completions: assistant message.audio (id/data/transcript) is not inside
	// content[]. Stubbing the turn to a text marker destroys the audio round-trip.
	if msg.Get("audio").Exists() && msg.Get("audio").Type != gjson.Null {
		return true
	}
	content := msg.Get("content")
	if !content.IsArray() {
		return false
	}
	for _, part := range content.Array() {
		switch strings.ToLower(strings.TrimSpace(part.Get("type").String())) {
		case "image_url", "input_audio", "file", "image", "document", "audio",
			"inline_data", "inlineData", "container_upload", "search_result",
			// Official Anthropic browser toolset: never history-stub browser_state turns.
			"browser_state",
			// Responses leftovers before in-place migrate (defense if normalize skipped).
			"input_image", "input_file",
			// Official Chat Completions refusal parts (may carry prompt_cache_breakpoint).
			// History-stubbing to a text marker destroys refusal semantics (garble class).
			"refusal":
			return true
		}
	}
	return false
}

// extractUserPromptText pulls plain text for Fast AST active-file protection.
// Never returns structured JSON dumps (array/object) - that reintroduces garble-class noise.
func extractUserPromptText(content gjson.Result) string {
	if !content.Exists() || content.Type == gjson.Null {
		return ""
	}
	if content.Type == gjson.String {
		return strings.TrimSpace(stripSystemReminderForPrompt(content.String()))
	}
	if !content.IsArray() {
		return ""
	}
	var chunks []string
	for _, part := range content.Array() {
		typ := strings.ToLower(strings.TrimSpace(part.Get("type").String()))
		if typ != "" && typ != "text" && typ != "input_text" && typ != "output_text" {
			continue
		}
		t := part.Get("text")
		if !t.Exists() {
			continue
		}
		text := t.String()
		if strings.Contains(text, "<system-reminder>") {
			continue
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		chunks = append(chunks, text)
	}
	return strings.TrimSpace(strings.Join(chunks, "\n"))
}

func stripSystemReminderForPrompt(s string) string {
	const open, closeTag = "<system-reminder>", "</system-reminder>"
	for {
		start := strings.Index(s, open)
		if start < 0 {
			break
		}
		end := strings.Index(s[start:], closeTag)
		if end < 0 {
			s = s[:start]
			break
		}
		s = s[:start] + s[start+end+len(closeTag):]
	}
	return s
}

// messageHasCitedContent detects Anthropic citations / OpenAI annotations on the message
// or content parts. Official Chat Completions puts url_citation annotations on the
// assistant message itself (not only inside content[]). Fast must not mutate or
// history-stub those turns (orphaned refs = garbled grounding / smash class).
func messageHasCitedContent(msg gjson.Result) bool {
	if msg.Get("annotations").Exists() {
		return true
	}
	content := msg.Get("content")
	if !content.IsArray() {
		return false
	}
	for _, part := range content.Array() {
		if part.Get("citations").Exists() || part.Get("annotations").Exists() {
			return true
		}
	}
	return false
}

// messageHasRefusalChrome detects official OpenAI Chat Completions refusal chrome:
// top-level message.refusal and/or typed content parts. History-stubbing or Fast-
// mutating these turns destroys refusal semantics (same smash class as tool chrome).
func messageHasRefusalChrome(msg gjson.Result) bool {
	if r := msg.Get("refusal"); r.Exists() && r.Type != gjson.Null {
		if strings.TrimSpace(r.String()) != "" || r.Type != gjson.String {
			return true
		}
	}
	content := msg.Get("content")
	if !content.IsArray() {
		return false
	}
	for _, part := range content.Array() {
		if strings.EqualFold(strings.TrimSpace(part.Get("type").String()), "refusal") {
			return true
		}
	}
	return false
}

// messageHasPromptCacheBreakpoint detects Anthropic cache_control OR OpenAI
// prompt_cache_breakpoint on the message or any content part. Official prompt-caching:
// the reusable prefix is everything up to and including the breakpoint - Fast/history
// stubbing any sibling in that message would invalidate the prefix (Claude + GPT).
func messageHasPromptCacheBreakpoint(msg gjson.Result) bool {
	if msg.Get("prompt_cache_breakpoint").Exists() || msg.Get("cache_control").Exists() {
		return true
	}
	content := msg.Get("content")
	if !content.IsArray() {
		return false
	}
	for _, part := range content.Array() {
		if part.Get("prompt_cache_breakpoint").Exists() || part.Get("cache_control").Exists() {
			return true
		}
	}
	return false
}

// partHasThoughtSignature detects Gemini thinking signatures on a content part.
// Also freezes thought summaries (thought:true) that must be replayed verbatim
// (official Gemini thinking docs: do not merge/mutate thought parts).
func partHasThoughtSignature(part gjson.Result) bool {
	if strings.TrimSpace(part.Get("thought_signature").String()) != "" {
		return true
	}
	if strings.TrimSpace(part.Get("thoughtSignature").String()) != "" {
		return true
	}
	if strings.TrimSpace(part.Get("extra_content.google.thought_signature").String()) != "" {
		return true
	}
	if part.Get("thought").Bool() {
		return true
	}
	return false
}

func (s *Server) optimizeText(text, userPrompt string) (string, int, int) {
	mode := trimmer.NormalizeMode(s.opts.CompressionMode)
	key := cache.Hash(text + "\x00" + userPrompt + "\x00" + mode +
		fmt.Sprintf("\x00%d\x00%v\x00%v", s.opts.MinLines, s.opts.DisableSkeletonize, s.opts.AlwaysCompactLogs))
	var optimized string
	var before, after int
	if cached, ok := s.lru.Get(key); ok {
		optimized = cached
		before = estimate([]byte(text))
		after = estimate([]byte(cached))
	} else {
		// Skeletonize without FileHistory so LRU stores stable AST output.
		optimized, before, after = trimmer.ProcessPromptTextOpts(text, userPrompt, trimmer.ProcessOptions{
			NeverTrim:            s.opts.NeverTrim,
			MaxLogBytes:          s.opts.MaxLogBytes,
			LogCompactMinBytes:   s.opts.LogCompactMinBytes,
			LogCompactMaxLines:   s.opts.LogCompactMaxLines,
			LogNoiseSubstrings:   s.opts.LogNoiseSubstrings,
			ActiveFileProtection: s.opts.ActiveFileProtection,
			Mode:                 mode,
			MinLines:             s.opts.MinLines,
			BalancedMinLines:     s.opts.BalancedMinLines,
			AggressiveMinLines:   s.opts.AggressiveMinLines,
			MildMinLines:         s.opts.MildMinLines,
			DisableSkeletonize:   s.opts.DisableSkeletonize,
			AlwaysCompactLogs:    s.opts.AlwaysCompactLogs,
			CustomQueries:        s.opts.CustomQueries,
		})
		s.lru.Set(key, optimized)
	}
	// Path-keyed unchanged stubs / unified diffs across turns (after LRU).
	if s.files != nil {
		optimized = trimmer.ApplyFileHistoryToFences(optimized, s.files)
		after = estimate([]byte(optimized))
	}
	return optimized, before, after
}

type trimPreviewRequest struct {
	Text       string `json:"text"`
	UserPrompt string `json:"user_prompt"`
	Mode       string `json:"mode"`
	MinLines   int    `json:"min_lines"`
	LogsOnly   bool   `json:"logs_only"`
}

type trimPreviewResponse struct {
	Mode          string   `json:"mode"`
	BeforeTokens  int      `json:"before_tokens"`
	AfterTokens   int      `json:"after_tokens"`
	SavedPercent  float64  `json:"saved_percent"`
	Output        string   `json:"output"`
	CustomQueries []string `json:"custom_queries,omitempty"`
}

func (s *Server) handleTrimPreview(w http.ResponseWriter, r *http.Request) {
	chrome := s.opts.Chrome
	if r.Method != http.MethodPost {
		http.Error(w, chrome.ErrMethodPostOnly, http.StatusMethodNotAllowed)
		return
	}
	var req trimPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, chrome.ErrInvalidJSON, http.StatusBadRequest)
		return
	}
	mode := trimmer.NormalizeMode(req.Mode)
	if mode == "" || req.Mode == "" {
		mode = trimmer.NormalizeMode(s.opts.CompressionMode)
	}
	minLines := req.MinLines
	if minLines <= 0 {
		minLines = s.opts.MinLines
	}
	disableSkel := s.opts.DisableSkeletonize
	alwaysLogs := s.opts.AlwaysCompactLogs
	if req.Mode != "" {
		if mode == trimmer.ModeCustom {
			disableSkel = s.opts.DisableSkeletonize || req.LogsOnly
			alwaysLogs = s.opts.AlwaysCompactLogs || req.LogsOnly
			if minLines <= 0 {
				minLines = s.opts.MinLines
			}
		} else {
			disableSkel = mode == trimmer.ModeMild || req.LogsOnly
			alwaysLogs = mode == trimmer.ModeMild || mode == trimmer.ModeAggressive || req.LogsOnly
		}
	}
	out, before, after := trimmer.ProcessPromptTextOpts(req.Text, req.UserPrompt, trimmer.ProcessOptions{
		NeverTrim:            s.opts.NeverTrim,
		MaxLogBytes:          s.opts.MaxLogBytes,
		LogCompactMinBytes:   s.opts.LogCompactMinBytes,
		LogCompactMaxLines:   s.opts.LogCompactMaxLines,
		LogNoiseSubstrings:   s.opts.LogNoiseSubstrings,
		ActiveFileProtection: s.opts.ActiveFileProtection,
		Mode:                 mode,
		MinLines:             minLines,
		BalancedMinLines:     s.opts.BalancedMinLines,
		AggressiveMinLines:   s.opts.AggressiveMinLines,
		MildMinLines:         s.opts.MildMinLines,
		DisableSkeletonize:   disableSkel,
		AlwaysCompactLogs:    alwaysLogs,
		CustomQueries:        s.opts.CustomQueries,
	})
	saved := 0.0
	if before > 0 {
		saved = float64(before-after) / float64(before) * 100
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(trimPreviewResponse{
		Mode:          mode,
		BeforeTokens:  before,
		AfterTokens:   after,
		SavedPercent:  saved,
		Output:        out,
		CustomQueries: s.opts.CustomQueries,
	})
}

func (s *Server) record(before, after int, latencyMs float64) {
	s.recordWithPreview(before, after, latencyMs, "", "", "", false, DeepStatusFastOnly, 0, 0)
}

func (s *Server) recordWithPreview(before, after int, latencyMs float64, beforePrev, afterPrev, door string, adapterUsed bool, deepStatus string, deepStageBefore, deepStageAfter int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.Requests++
	s.stats.TokensBefore += int64(before)
	s.stats.TokensAfter += int64(after)
	rate := s.opts.SavingsUsdPerMTok
	if rate > 0 {
		saved := float64(before-after) / 1_000_000.0 * rate
		if saved > 0 {
			s.stats.SavedUSD += saved
		}
	}
	s.stats.LastLatency = latencyMs
	s.stats.LastBeforeTokens = before
	s.stats.LastAfterTokens = after
	if beforePrev != "" {
		s.stats.LastBeforePreview = s.truncatePreview(beforePrev)
	}
	if afterPrev != "" {
		s.stats.LastAfterPreview = s.truncatePreview(afterPrev)
	}
	if door != "" {
		s.stats.LastDoor = door
	}
	if adapterUsed {
		s.stats.AdapterRequests++
	}
	s.stats.LastDeepStatus = strings.TrimSpace(deepStatus)
	s.stats.LastDeepStageBefore = deepStageBefore
	s.stats.LastDeepStageAfter = deepStageAfter
	s.stats.LastDeepStageSavedPct = DeepStageSavedPercent(deepStageBefore, deepStageAfter)
}

func (s *Server) truncatePreview(text string) string {
	max := s.opts.PreviewMaxChars
	if max <= 0 {
		// Fail closed: unset max means no invent preview length.
		return ""
	}
	if len(text) <= max {
		return text
	}
	suffix := strings.TrimSpace(s.opts.Chrome.PreviewTruncSuffix)
	if suffix == "" {
		return text[:max]
	}
	if len(suffix) >= max {
		return suffix[:max]
	}
	return text[:max-len(suffix)] + suffix
}

// skipLiveDeepForRequestPath mirrors cli/internal/deepopt.SkipLiveDeepForRequestPath.
// Anthropic count_tokens is Fast-only: no LLMLingua (latency + body mismatch vs /messages).
func skipLiveDeepForRequestPath(path string) bool {
	pl := strings.ToLower(strings.TrimSpace(path))
	return strings.Contains(pl, "count_tokens")
}

// shouldAcceptDeepResult is the fail-closed gate for applying Deep on every door
// (Anthropic Messages + OpenAI-compat GPT/Gemini/DeepSeek/Mistral).
// Must stay the logical inverse of cli/internal/deepopt.DeepExpanded (same checks).
// Compares wire body to wire body and Deep-stage tokens to Deep-stage tokens.
// Never compare estimate(fullJSON) against Fast's summed message-text `after`
// (that unit mismatch silently discarded Deep for all providers).
func shouldAcceptDeepResult(fastBody, deepBody []byte, deepOrigin, deepCompressed int) bool {
	if len(deepBody) == 0 {
		return false
	}
	if len(deepBody) > len(fastBody) {
		return false
	}
	if deepOrigin > 0 && deepCompressed >= deepOrigin {
		return false
	}
	if estimate(deepBody) > estimate(fastBody) {
		return false
	}
	return true
}

func estimate(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	return (len(b) + 3) / 4
}
