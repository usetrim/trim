package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

// Registry is a minimal OpenMetrics text exporter (no third-party deps).
type Registry struct {
	proxyRequests     atomic.Int64
	proxyTokensBefore atomic.Int64
	proxyTokensAfter  atomic.Int64
	proxyErrors       atomic.Int64
	quotaExhausted    atomic.Int64
	hmacFailures      atomic.Int64
}

func New() *Registry {
	return &Registry{}
}

func (r *Registry) IncProxyRequest(before, after int, isError bool) {
	r.proxyRequests.Add(1)
	r.proxyTokensBefore.Add(int64(before))
	r.proxyTokensAfter.Add(int64(after))
	if isError {
		r.proxyErrors.Add(1)
	}
}

func (r *Registry) IncQuotaExhausted() { r.quotaExhausted.Add(1) }
func (r *Registry) IncHMACFailure()    { r.hmacFailures.Add(1) }

func (r *Registry) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = fmt.Fprintf(w, "# HELP trim_proxy_requests_total Total proxy completions forwarded\n")
		_, _ = fmt.Fprintf(w, "# TYPE trim_proxy_requests_total counter\n")
		_, _ = fmt.Fprintf(w, "trim_proxy_requests_total %d\n", r.proxyRequests.Load())
		_, _ = fmt.Fprintf(w, "# HELP trim_proxy_tokens_before_total Estimated tokens before trim\n")
		_, _ = fmt.Fprintf(w, "# TYPE trim_proxy_tokens_before_total counter\n")
		_, _ = fmt.Fprintf(w, "trim_proxy_tokens_before_total %d\n", r.proxyTokensBefore.Load())
		_, _ = fmt.Fprintf(w, "# HELP trim_proxy_tokens_after_total Estimated tokens after trim\n")
		_, _ = fmt.Fprintf(w, "# TYPE trim_proxy_tokens_after_total counter\n")
		_, _ = fmt.Fprintf(w, "trim_proxy_tokens_after_total %d\n", r.proxyTokensAfter.Load())
		_, _ = fmt.Fprintf(w, "# HELP trim_proxy_errors_total Upstream proxy responses with status >= 400\n")
		_, _ = fmt.Fprintf(w, "# TYPE trim_proxy_errors_total counter\n")
		_, _ = fmt.Fprintf(w, "trim_proxy_errors_total %d\n", r.proxyErrors.Load())
		_, _ = fmt.Fprintf(w, "# HELP trim_quota_exhausted_total Quota denials (HTTP 402)\n")
		_, _ = fmt.Fprintf(w, "# TYPE trim_quota_exhausted_total counter\n")
		_, _ = fmt.Fprintf(w, "trim_quota_exhausted_total %d\n", r.quotaExhausted.Load())
		_, _ = fmt.Fprintf(w, "# HELP trim_cli_hmac_failures_total Failed CLI HMAC signature checks\n")
		_, _ = fmt.Fprintf(w, "# TYPE trim_cli_hmac_failures_total counter\n")
		_, _ = fmt.Fprintf(w, "trim_cli_hmac_failures_total %d\n", r.hmacFailures.Load())
	}
}
