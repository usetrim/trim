package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usetrim/trim/server/internal/config"
	"github.com/usetrim/trim/server/internal/middleware"
	"github.com/usetrim/trim/server/internal/subscriptions"
	"github.com/usetrim/trim/server/pkg/metrics"
	pkgproxy "github.com/usetrim/trim/server/pkg/proxy"
	"github.com/usetrim/trim/server/pkg/provideradapt"
)

// EventSink records a completed trim for cloud telemetry (trim_events).
type EventSink func(userID, requestID, model string, before, after int, latencyMs float64, status, mode, errorCode string)

// Stats is the live proxy meter snapshot.
type Stats = pkgproxy.Stats

// Server adapts cloud config onto the shared pkg/proxy engine.
type Server struct {
	cfg     config.Config
	db      *pgxpool.Pool
	inner   *pkgproxy.Server
	onEvent EventSink
	Metrics *metrics.Registry

	mu          sync.Mutex
	appliedBal  int
	appliedAgg  int
	appliedMild int
	lastRefresh time.Time
}

func NewServer(cfg config.Config, db *pgxpool.Pool) *Server {
	s := &Server{cfg: cfg, db: db, Metrics: metrics.New()}
	s.rebuild(0, 0, 0)
	return s
}

func (s *Server) SetEventSink(fn EventSink) {
	s.onEvent = fn
	s.mu.Lock()
	bal, agg, mild := s.appliedBal, s.appliedAgg, s.appliedMild
	s.mu.Unlock()
	s.rebuild(bal, agg, mild)
}

func (s *Server) rebuild(bal, agg, mild int) {
	routeMax := 0
	if s.cfg.RouteMaxTokens != "" {
		if n, err := strconv.Atoi(s.cfg.RouteMaxTokens); err == nil {
			routeMax = n
		}
	}
	mode := strings.TrimSpace(s.cfg.CompressionMode)
	// Fail-closed: missing/invalid site_messages => false (do not invent true).
	fallback := strings.EqualFold(strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_FALLBACK_UNCOMPRESSED")), "true")
	activeFile := strings.EqualFold(strings.TrimSpace(subscriptions.MessageForCode("PROXY_ACTIVE_FILE_PROTECTION")), "true")
	reg := s.loadAdapterRegistry()
	s.inner = pkgproxy.NewServer(pkgproxy.Options{
		UpstreamOpenAI:            s.cfg.UpstreamOpenAI,
		UpstreamAnthropic:         s.cfg.UpstreamAnthropic,
		UpstreamOpenAIFailover:    s.cfg.UpstreamOpenAIFailover,
		UpstreamAnthropicFailover: s.cfg.UpstreamAnthropicFailover,
		ActiveFileProtection:      activeFile,
		CompressionMode:           mode,
		BalancedMinLines:          bal,
		AggressiveMinLines:        agg,
		MildMinLines:              mild,
		FallbackUncompressed:      fallback,
		CheapModel:                s.cfg.CheapModel,
		RouteMaxTokens:            routeMax,
		SavingsUsdPerMTok:         s.cfg.SavingsUsdPerMTok,
		HistoryKeepTurns:          s.cfg.HistoryKeepTurns,
		UpstreamHTTPTimeout:       time.Duration(s.cfg.ProxyHTTPTimeoutSec) * time.Second,
		ProviderAdapters:          reg,
		DoorOpenAI:                strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_DOOR_OPENAI")),
		DoorAnthropic:             strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_DOOR_ANTHROPIC")),
		AdapterTranslateFmt:       strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ADAPTER_TRANSLATE_FMT")),
		AdapterAuthFmt:            strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ADAPTER_AUTH_FMT")),
		AdapterResponseFmt:        strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ADAPTER_RESPONSE_FMT")),
		ModelsNotSynced:           strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_MODELS_NOT_SYNCED")),
		ModelsMethod:              strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_MODELS_METHOD")),
		BaseURLDoubleV1:           strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ERR_BASE_URL_DOUBLE_V1")),
		ModelNotFoundFmt:          strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ERR_MODEL_NOT_FOUND_FMT")),
		UnknownModelFmt:           strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ERR_UNKNOWN_MODEL_FMT")),
		UpstreamAuthFmt:           strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ERR_UPSTREAM_AUTH_FMT")),
		UpstreamQuotaFmt:          strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ERR_UPSTREAM_QUOTA_FMT")),
		UpstreamUnavailableFmt:    strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ERR_UPSTREAM_UNAVAILABLE_FMT")),
		ModelsUpstreamAdapterRequired: strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_MODELS_UPSTREAM_ADAPTER_REQUIRED")),
		ModelsUpstreamNotFoundFmt:     strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_MODELS_UPSTREAM_NOT_FOUND_FMT")),
		ModelsUpstreamAuthRequired:    strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_MODELS_UPSTREAM_AUTH")),
		ModelsUpstreamUnsupportedFmt:  strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_MODELS_UPSTREAM_UNSUPPORTED_FMT")),
		OnMetrics: func(before, after int, isError bool) {
			if s.Metrics != nil {
				s.Metrics.IncProxyRequest(before, after, isError)
			}
		},
		OnEvent: func(r *http.Request, model string, before, after int, latencyMs float64, status, mode, errorCode string) {
			if s.onEvent == nil || r == nil {
				return
			}
			s.onEvent(
				middleware.UserIDFromContext(r.Context()),
				chimw.GetReqID(r.Context()),
				model,
				before,
				after,
				latencyMs,
				status,
				mode,
				errorCode,
			)
		},
	})
	s.mu.Lock()
	s.appliedBal = bal
	s.appliedAgg = agg
	s.appliedMild = mild
	s.mu.Unlock()
}

func (s *Server) loadAdapterRegistry() *provideradapt.Registry {
	if s.db == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := s.db.Query(ctx, `
		select id, enabled, sort_order, dialect, door_label,
		       match_model_prefixes, model_aliases,
		       upstream_kind, upstream_path, coalesce(upstream_base_url, ''), auth_mode,
		       coalesce(anthropic_version, ''), coalesce(anthropic_workspace_id, ''), coalesce(default_max_tokens, 0), require_alias
		from public.provider_adapters
		order by sort_order asc, id asc
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	adapters := make([]provideradapt.AdapterConfig, 0)
	for rows.Next() {
		var c provideradapt.AdapterConfig
		var prefixesRaw, aliasesRaw []byte
		if err := rows.Scan(
			&c.ID, &c.Enabled, &c.SortOrder, &c.Dialect, &c.DoorLabel,
			&prefixesRaw, &aliasesRaw,
			&c.UpstreamKind, &c.UpstreamPath, &c.UpstreamBaseURL, &c.AuthMode,
			&c.AnthropicVersion, &c.AnthropicWorkspaceID, &c.DefaultMaxTokens, &c.RequireAlias,
		); err != nil {
			return nil
		}
		if err := json.Unmarshal(prefixesRaw, &c.MatchModelPrefixes); err != nil {
			return nil
		}
		if err := json.Unmarshal(aliasesRaw, &c.ModelAliases); err != nil {
			return nil
		}
		adapters = append(adapters, c)
	}
	if err := rows.Err(); err != nil {
		return nil
	}
	arows, err := s.db.Query(ctx, `
		select client_model, upstream_model, coalesce(upstream_host_contains, ''), enabled
		from public.openai_model_aliases
		order by client_model asc
	`)
	if err != nil {
		return nil
	}
	defer arows.Close()
	aliases := make([]provideradapt.ModelAlias, 0)
	for arows.Next() {
		var a provideradapt.ModelAlias
		if err := arows.Scan(&a.ClientModel, &a.UpstreamModel, &a.UpstreamHostContains, &a.Enabled); err != nil {
			return nil
		}
		aliases = append(aliases, a)
	}
	if err := arows.Err(); err != nil {
		return nil
	}
	drows, err := s.db.Query(ctx, `
		select id, enabled, sort_order,
		       coalesce(display_name, ''), coalesce(description, ''),
		       owned_by, coalesce(adapter_id, ''),
		       extract(epoch from updated_at)::bigint
		from public.provider_discoverable_models
		order by sort_order asc, id asc
	`)
	if err != nil {
		return nil
	}
	defer drows.Close()
	disco := make([]provideradapt.DiscoverableModel, 0)
	for drows.Next() {
		var m provideradapt.DiscoverableModel
		if err := drows.Scan(
			&m.ID, &m.Enabled, &m.SortOrder,
			&m.DisplayName, &m.Description,
			&m.OwnedBy, &m.AdapterID, &m.Created,
		); err != nil {
			return nil
		}
		disco = append(disco, m)
	}
	if err := drows.Err(); err != nil {
		return nil
	}
	reg, err := provideradapt.NewRegistry(adapters, aliases, disco, provideradapt.Chrome{
		TranslateFmt:      strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ADAPTER_TRANSLATE_FMT")),
		AuthFmt:           strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ADAPTER_AUTH_FMT")),
		ResponseFmt:       strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ADAPTER_RESPONSE_FMT")),
		ModelRequired:     strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ADAPTER_MODEL_REQUIRED")),
		AliasRequiredFmt:  strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ADAPTER_ALIAS_REQUIRED_FMT")),
		UnknownDialectFmt: strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ADAPTER_UNKNOWN_DIALECT_FMT")),
		ModelsNotSynced:   strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_MODELS_NOT_SYNCED")),
		ModelsMethod:      strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_MODELS_METHOD")),
		BaseURLDoubleV1:   strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ERR_BASE_URL_DOUBLE_V1")),
		ModelNotFoundFmt:  strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ERR_MODEL_NOT_FOUND_FMT")),
		UpstreamAuthFmt:   strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ERR_UPSTREAM_AUTH_FMT")),
		UpstreamQuotaFmt:  strings.TrimSpace(subscriptions.MessageForCode("CLI_PROXY_ERR_UPSTREAM_QUOTA_FMT")),
	})
	if err != nil {
		return nil
	}
	return reg
}

func (s *Server) Stats() Stats {
	return s.inner.Stats()
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshEvery := proxyOptionsRefreshInterval()
		s.mu.Lock()
		stale := refreshEvery > 0 && time.Since(s.lastRefresh) > refreshEvery
		s.mu.Unlock()
		if stale {
			ovr := middleware.CurrentRuntimeOverrides()
			s.mu.Lock()
			need := ovr.FastBalancedMinLines != s.appliedBal ||
				ovr.FastAggressiveMinLines != s.appliedAgg ||
				ovr.FastMildMinLines != s.appliedMild
			s.lastRefresh = time.Now()
			s.mu.Unlock()
			if need {
				s.rebuild(ovr.FastBalancedMinLines, ovr.FastAggressiveMinLines, ovr.FastMildMinLines)
			}
		}
		s.inner.Handler().ServeHTTP(w, r)
	})
}

// proxyOptionsRefreshInterval reads PROXY_OPTIONS_REFRESH_MS. Fail-closed: 0 = never invent.
func proxyOptionsRefreshInterval() time.Duration {
	raw := strings.TrimSpace(subscriptions.MessageForCode("PROXY_OPTIONS_REFRESH_MS"))
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0
	}
	return time.Duration(n) * time.Millisecond
}
