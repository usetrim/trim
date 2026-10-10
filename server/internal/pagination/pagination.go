package pagination

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/usetrim/trim/server/internal/subscriptions"
)

// Params is OFFSET/LIMIT pagination driven by query params skip and limit.
type Params struct {
	Skip  int
	Limit int
}

// Meta is returned with every list response so the client never invents totals.
type Meta struct {
	Skip               int    `json:"skip"`
	Limit              int    `json:"limit"`
	Total              int    `json:"total"`
	HasMore            bool   `json:"has_more"`
	NextSkip           *int   `json:"next_skip"`
	PrevSkip           *int   `json:"prev_skip"`
	Page               int    `json:"page"`
	TotalPages         int    `json:"total_pages"`
	PrevActionLabel    string `json:"prev_action_label"`
	NextActionLabel    string `json:"next_action_label"`
	FirstSkip          *int   `json:"first_skip"`
	LastSkip           *int   `json:"last_skip"`
	FirstActionLabel   string `json:"first_action_label"`
	LastActionLabel    string `json:"last_action_label"`
	FirstPendingLabel  string `json:"first_pending_label"`
	PrevPendingLabel   string `json:"prev_pending_label"`
	NextPendingLabel   string `json:"next_pending_label"`
	LastPendingLabel   string `json:"last_pending_label"`
	SkipToLabel        string `json:"skip_to_label"`
	SkipToActionLabel  string `json:"skip_to_action_label"`
	SkipToPendingLabel string `json:"skip_to_pending_label"`
	SkipToInvalid      string `json:"skip_to_invalid"`
	PageSummary        string `json:"page_summary"`
	// PageSkips[i] is the database OFFSET for 1-based page i+1.
	// Built only when total_pages is within the skip-to window (no client (page-1)*limit invent).
	PageSkips []int `json:"page_skips,omitempty"`
}

// CodeError is an error whose Error() returns a site_messages code.
type CodeError struct {
	Code string
}

func (e CodeError) Error() string { return e.Code }

// Parse reads skip and limit from the request. Both must be present as integers.
// limit must be between 1 and maxLimit inclusive.
// Returned errors are site_messages codes (via CodeError).
func Parse(r *http.Request, maxLimit int) (Params, error) {
	if maxLimit < 1 {
		return Params{}, CodeError{Code: "PAGINATION_MAX_LIMIT_INVALID"}
	}

	skipRaw := r.URL.Query().Get("skip")
	limitRaw := r.URL.Query().Get("limit")
	if skipRaw == "" {
		return Params{}, CodeError{Code: "PAGINATION_SKIP_REQUIRED"}
	}
	if limitRaw == "" {
		return Params{}, CodeError{Code: "PAGINATION_LIMIT_REQUIRED"}
	}

	skip, err := strconv.Atoi(skipRaw)
	if err != nil {
		return Params{}, CodeError{Code: "PAGINATION_SKIP_NOT_INT"}
	}
	limit, err := strconv.Atoi(limitRaw)
	if err != nil {
		return Params{}, CodeError{Code: "PAGINATION_LIMIT_NOT_INT"}
	}
	if skip < 0 {
		return Params{}, CodeError{Code: "PAGINATION_SKIP_NEGATIVE"}
	}
	if limit < 1 {
		return Params{}, CodeError{Code: "PAGINATION_LIMIT_TOO_SMALL"}
	}
	if limit > maxLimit {
		return Params{}, CodeError{Code: "PAGINATION_LIMIT_TOO_LARGE"}
	}

	return Params{Skip: skip, Limit: limit}, nil
}

// BuildMeta computes pagination metadata from a real total count returned by the database.
// maxSkipToPages comes from billing_settings.pagination_skip_to_max_pages (no invent const).
// When total_pages exceeds maxSkipToPages, page_skips is omitted (first/last/prev/next only).
func BuildMeta(p Params, total int, maxSkipToPages int) Meta {
	if total < 0 {
		total = 0
	}
	page := 1
	if p.Limit > 0 {
		page = (p.Skip / p.Limit) + 1
	}
	totalPages := 0
	if p.Limit > 0 && total > 0 {
		totalPages = (total + p.Limit - 1) / p.Limit
	}

	summaryFmt := subscriptions.MessageForCode("PAGINATION_PAGE_SUMMARY_FMT")
	pageSummary := ""
	if summaryFmt != "" {
		pageSummary = fmt.Sprintf(summaryFmt, page, totalPages, total)
	}
	meta := Meta{
		Skip:               p.Skip,
		Limit:              p.Limit,
		Total:              total,
		HasMore:            p.Skip+p.Limit < total,
		Page:               page,
		TotalPages:         totalPages,
		PrevActionLabel:    subscriptions.MessageForCode("PAGINATION_PREV"),
		NextActionLabel:    subscriptions.MessageForCode("PAGINATION_NEXT"),
		FirstActionLabel:   subscriptions.MessageForCode("PAGINATION_FIRST"),
		LastActionLabel:    subscriptions.MessageForCode("PAGINATION_LAST"),
		FirstPendingLabel:  subscriptions.PendingLabelForCode("PAGINATION_FIRST"),
		PrevPendingLabel:   subscriptions.PendingLabelForCode("PAGINATION_PREV"),
		NextPendingLabel:   subscriptions.PendingLabelForCode("PAGINATION_NEXT"),
		LastPendingLabel:   subscriptions.PendingLabelForCode("PAGINATION_LAST"),
		SkipToLabel:        subscriptions.MessageForCode("PAGINATION_SKIP_TO_LABEL"),
		SkipToActionLabel:  subscriptions.ActionLabelForCode("PAGINATION_SKIP_TO"),
		SkipToPendingLabel: subscriptions.PendingLabelForCode("PAGINATION_SKIP_TO"),
		SkipToInvalid:      subscriptions.MessageForCode("PAGINATION_SKIP_TO_INVALID"),
		PageSummary:        pageSummary,
	}

	if meta.HasMore {
		next := p.Skip + p.Limit
		meta.NextSkip = &next
	}
	if p.Skip > 0 {
		prev := p.Skip - p.Limit
		if prev < 0 {
			prev = 0
		}
		meta.PrevSkip = &prev
		zero := 0
		meta.FirstSkip = &zero
	}
	if totalPages > 1 && page < totalPages && p.Limit > 0 {
		last := (totalPages - 1) * p.Limit
		meta.LastSkip = &last
	}

	// Backend-owned skip-to offsets (cap from billing_settings; beyond this, first/last/prev/next only).
	if maxSkipToPages > 0 && totalPages > 0 && totalPages <= maxSkipToPages && p.Limit > 0 {
		skips := make([]int, totalPages)
		for i := 0; i < totalPages; i++ {
			skips[i] = i * p.Limit
		}
		meta.PageSkips = skips
	}

	return meta
}
