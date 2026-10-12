package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/usetrim/trim/cli/internal/cliout"
)

func TestParseQuotaStatus(t *testing.T) {
	q, err := parseQuotaStatus([]byte(`{
		"plan_tier":"team",
		"monthly_credit_limit":7000,
		"remaining":7000,
		"unlimited":true,
		"unlimited_label":"Unlimited",
		"tier_upgrade":"https://use-trim.com/dashboard"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if q.PlanTier != "team" || !q.Unlimited || q.Remaining != 7000 {
		t.Fatalf("%+v", q)
	}
}

func TestPrintHumanStatus_Unlimited(t *testing.T) {
	var buf bytes.Buffer
	prev := cliout.Writer
	cliout.Writer = &buf
	t.Cleanup(func() { cliout.Writer = prev })
	t.Setenv("NO_COLOR", "1")

	ch := cliChrome{}
	applyHumanChromeFallbacks(&ch)
	printHumanStatus(ch, quotaStatus{
		PlanTier:           "team",
		MonthlyCreditLimit: 7000,
		Remaining:          7000,
		Unlimited:          true,
		UnlimitedLabel:     "Unlimited",
		TierUpgrade:        "https://use-trim.com/dashboard",
	})
	out := buf.String()
	for _, want := range []string{"Logged in", "Plan", "Team", "Credits", "Unlimited", "Remaining", "7000 / 7000", "Dashboard", "use-trim.com"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "monthly_credit") || strings.Contains(out, "API status") {
		t.Fatalf("raw API leak:\n%s", out)
	}
}

func TestPrintHumanStatus_MeteredLow(t *testing.T) {
	var buf bytes.Buffer
	prev := cliout.Writer
	cliout.Writer = &buf
	t.Cleanup(func() { cliout.Writer = prev })
	t.Setenv("NO_COLOR", "1")

	ch := cliChrome{}
	applyHumanChromeFallbacks(&ch)
	printHumanStatus(ch, quotaStatus{
		PlanTier:           "pro",
		MonthlyCreditLimit: 2000,
		Remaining:          100,
		Unlimited:          false,
		TierUpgrade:        "https://use-trim.com/dashboard",
	})
	out := buf.String()
	if !strings.Contains(out, "Pro") || !strings.Contains(out, "100 / 2000") {
		t.Fatalf("out:\n%s", out)
	}
	if strings.Contains(out, "Credits") {
		t.Fatalf("credits row should be unlimited-only:\n%s", out)
	}
	if !strings.Contains(out, "Upgrade") || !strings.Contains(out, "trim upgrade") {
		t.Fatalf("low metered should show upgrade action:\n%s", out)
	}
}
