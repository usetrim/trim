package proxy

// DeepResult is the outcome of optional live Deep after Fast.
// Status uses stable machine codes for /dashboard and /v1/stats (chrome maps labels).
type DeepResult struct {
	Body       []byte
	Origin     int
	Compressed int
	Status     string
}

// Live Deep status codes (API + dashboard). Do not invent ad-hoc strings at call sites.
const (
	DeepStatusApplied          = "applied"
	DeepStatusSkippedMin       = "skipped_min"
	DeepStatusSkippedStream    = "skipped_stream"
	DeepStatusSkippedEmpty     = "skipped_empty"
	DeepStatusSkippedOff       = "skipped_off"
	DeepStatusPathSkip         = "path_skip"
	DeepStatusFailClosedExpand = "fail_closed_expand"
	DeepStatusFailClosedError  = "fail_closed_error"
	DeepStatusOOMSkip          = "oom_skip"
	DeepStatusRejected         = "rejected"
	DeepStatusFastOnly         = "fast_only"
)

// DeepStageSavedPercent is Deep-stage savings (origin→compressed), not whole-wire %.
func DeepStageSavedPercent(origin, compressed int) float64 {
	if origin <= 0 {
		return 0
	}
	if compressed < 0 {
		compressed = 0
	}
	if compressed >= origin {
		return 0
	}
	return float64(origin-compressed) / float64(origin) * 100
}
