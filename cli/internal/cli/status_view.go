package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/usetrim/trim/cli/internal/cliout"
)

// quotaStatus is the cloud /api/v1/me/quota payload used for human status.
type quotaStatus struct {
	PlanTier              string `json:"plan_tier"`
	MonthlyCreditLimit    int    `json:"monthly_credit_limit"`
	MonthlyCreditUsed     int    `json:"monthly_credit_used"`
	PurchasedTopupCredits int    `json:"purchased_topup_credits"`
	Remaining             int    `json:"remaining"`
	Unlimited             bool   `json:"unlimited"`
	UnlimitedLabel        string `json:"unlimited_label"`
	TierUpgrade           string `json:"tier_upgrade"`
	Code                  string `json:"code"`
}

func parseQuotaStatus(body []byte) (quotaStatus, error) {
	var q quotaStatus
	if err := json.Unmarshal(body, &q); err != nil {
		return quotaStatus{}, err
	}
	return q, nil
}

func printHumanStatus(chrome cliChrome, q quotaStatus) {
	if header := strings.TrimSpace(chrome.StatusLoggedIn); header != "" {
		cliout.Success(header)
	}

	rows := make([]cliout.Row, 0, 6)

	if lab := strings.TrimSpace(chrome.StatusLabelPlan); lab != "" {
		plan := cliout.TitleCaseWord(q.PlanTier)
		if plan != "" {
			rows = append(rows, cliout.Row{Label: lab, Value: plan, Kind: "accent"})
		}
	}

	if lab := strings.TrimSpace(chrome.StatusLabelCredits); lab != "" && q.Unlimited {
		if credits := strings.TrimSpace(q.UnlimitedLabel); credits != "" {
			rows = append(rows, cliout.Row{Label: lab, Value: credits, Kind: "ok"})
		}
	}

	if lab := strings.TrimSpace(chrome.StatusLabelRemaining); lab != "" {
		limit := q.MonthlyCreditLimit
		if limit < 0 {
			limit = 0
		}
		usedShown := q.Remaining
		remFmt := strings.TrimSpace(chrome.StatusRemainingFmt)
		val := ""
		if remFmt != "" && strings.Contains(remFmt, "%") {
			val = fmt.Sprintf(remFmt, usedShown, limit)
		} else {
			val = fmt.Sprintf("%d / %d", usedShown, limit)
		}
		kind := "ok"
		if !q.Unlimited && limit > 0 {
			pct := float64(usedShown) / float64(limit)
			if pct <= 0.15 {
				kind = "bad"
			} else if pct <= 0.35 {
				kind = "warn"
			}
		}
		if q.Unlimited {
			kind = "ok"
		}
		rows = append(rows, cliout.Row{Label: lab, Value: val, Kind: kind})
	}

	dash := strings.TrimSpace(q.TierUpgrade)
	if lab := strings.TrimSpace(chrome.StatusLabelDashboard); lab != "" && dash != "" {
		rows = append(rows, cliout.Row{Label: lab, Value: dash})
	}

	// Actionable next step when metered credits are exhausted or running low.
	if !q.Unlimited {
		limit := q.MonthlyCreditLimit
		low := q.Remaining <= 0
		if !low && limit > 0 {
			low = float64(q.Remaining)/float64(limit) <= 0.35
		}
		if low {
			if lab := strings.TrimSpace(chrome.StatusLabelUpgrade); lab != "" {
				rows = append(rows, cliout.Row{Label: lab, Value: "trim upgrade", Kind: "warn"})
			}
		}
	}

	cliout.Table(rows)
}
